package initiativereview

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/review"
	"swarm-manager/internal/workshop"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// buildContextAttachments assembles the initiative-scoped context the review
// agent needs to render a verdict:
//
//   - initiative metadata (title, goal, priority, items list)
//   - materialized graph.json (for graph-shape reasoning)
//   - per-item summaries (kind, title, status, archived, deliverable path)
//   - per-item latest review round snapshots (so the initiative review isn't
//     redoing work the per-item reviews already did — it synthesizes)
//   - aggregate deliverable content for completed items (union of plan.md /
//     conclusion.md) so the agent can spot cross-item regressions
//   - the union of affected scenarios across all member items and a
//     *fresh* GCT (git-control-tower) verdict per scenario run at review
//     start — this is the "is the whole thing still working together?"
//     integration signal the initiative review is specifically designed
//     to assess. Uses the same attachment keys (affected-scenarios,
//     gct-review-results) as backlog review so the skill sees a single
//     vocabulary across owner types.
//
// affectedScenarios and freshGCT are gathered upstream in startReview so
// the attachment writer stays purely about attachment shape.
//
// All attachments are "note" type, separated by key so BuildSplitPrompt on
// the agent-manager side can route them appropriately.
func (s *Service) buildContextAttachments(init *initiatives.Initiative, affectedScenarios []string, freshGCT map[string]*GCTResult) ([]*domainpb.ContextAttachment, error) {
	var atts []*domainpb.ContextAttachment

	atts = appendNote(atts, "initiative-summary", "Initiative Summary",
		"Name, title, goal, and rollup",
		renderInitiativeSummary(init), "text", "high")

	if mg, err := s.graphReader.ReadGraph(init.Name); err == nil && mg != nil {
		if payload, err := json.MarshalIndent(mg, "", "  "); err == nil {
			atts = appendNote(atts, "initiative-graph", "Initiative Item Graph",
				fmt.Sprintf("%d nodes, %d edges", len(mg.Nodes), len(mg.Edges)),
				string(payload), "json", "high")
		}
	}

	summaries, reviewSnapshots, deliverables := s.collectItemEvidence(init)
	if len(summaries) > 0 {
		atts = appendNote(atts, "item-summaries", "Member Item Summaries",
			fmt.Sprintf("%d items", len(summaries)),
			strings.Join(summaries, "\n\n"), "markdown", "high")
	}
	if len(reviewSnapshots) > 0 {
		atts = appendNote(atts, "item-review-snapshots", "Per-Item Review Snapshots",
			"Latest review round assessment/classification per item",
			strings.Join(reviewSnapshots, "\n\n"), "markdown", "medium")
	}
	if trimmed := strings.TrimSpace(deliverables); trimmed != "" {
		atts = appendNote(atts, "item-deliverables", "Aggregated Item Deliverables",
			"Plan / conclusion content from member items",
			trimmed, "markdown", "medium")
	}

	atts = appendFreshGCTAttachments(atts, affectedScenarios, freshGCT, len(init.Items))

	return atts, nil
}

// appendFreshGCTAttachments adds affected-scenarios + gct-review-results
// to the attachment slice. Key names intentionally match the backlog
// review flow so the skill sees one vocabulary across owner types.
// When no scenarios are in scope (executionLookup not wired, or no item
// has a finalization), both keys are omitted — the review still runs,
// just without integration evidence.
func appendFreshGCTAttachments(atts []*domainpb.ContextAttachment, scenarios []string, freshGCT map[string]*GCTResult, itemCount int) []*domainpb.ContextAttachment {
	if len(scenarios) == 0 {
		return atts
	}

	atts = appendNote(atts, "affected-scenarios", "Affected Scenarios",
		fmt.Sprintf("%d scenarios touched across %d items", len(scenarios), itemCount),
		strings.Join(scenarios, "\n"), "text", "medium")

	if len(freshGCT) == 0 {
		return atts
	}

	// Stable ordering — agent output is sensitive to context churn, and a
	// map marshal would otherwise re-order fields across review runs.
	ordered := make([]*GCTResult, 0, len(freshGCT))
	keys := make([]string, 0, len(freshGCT))
	for k := range freshGCT {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if res := freshGCT[k]; res != nil {
			ordered = append(ordered, res)
		}
	}

	if payload, err := json.MarshalIndent(ordered, "", "  "); err == nil {
		atts = appendNote(atts, "gct-review-results", "GCT Review Results",
			"Fresh GCT verdict per affected scenario, collected at review start",
			string(payload), "json", "high")
	}

	return atts
}

// renderInitiativeSummary produces a compact Markdown block with the fields
// the review agent needs to judge the initiative as a whole. Description
// is the stated goal — the agent grades against it.
func renderInitiativeSummary(init *initiatives.Initiative) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", fallbackInitiativeTitle(init))
	fmt.Fprintf(&b, "**Name:** `%s`\n", init.Name)
	fmt.Fprintf(&b, "**Status:** `%s`\n", init.Status)
	if init.Priority > 0 {
		fmt.Fprintf(&b, "**Priority:** %d\n", init.Priority)
	}
	if len(init.DependsOn) > 0 {
		fmt.Fprintf(&b, "**Depends on initiatives:** %s\n", strings.Join(init.DependsOn, ", "))
	}
	if desc := strings.TrimSpace(init.Description); desc != "" {
		b.WriteString("\n## Stated goal\n\n")
		b.WriteString(desc)
		b.WriteString("\n")
	}
	if note := strings.TrimSpace(init.Note); note != "" {
		b.WriteString("\n## Note\n\n")
		b.WriteString(note)
		b.WriteString("\n")
	}
	return b.String()
}

// collectItemEvidence walks the initiative's member items once and returns
// parallel slices of summaries, review snapshots, and deliverable content.
// Missing items / unreadable files are silently skipped — the attachment
// is best-effort, not a correctness gate.
func (s *Service) collectItemEvidence(init *initiatives.Initiative) ([]string, []string, string) {
	summaries := make([]string, 0, len(init.Items))
	reviewSnaps := make([]string, 0, len(init.Items))
	var deliverablesBuf strings.Builder

	// Sort for stable output — agents seeing rearranged context across
	// runs triggers false-positive "something changed" reactions.
	refs := append([]string(nil), init.Items...)
	sort.Strings(refs)

	for _, ref := range refs {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			continue
		}
		item, err := s.backlogLoader.LoadItem(kind, parts[1])
		if err != nil {
			continue
		}
		summaries = append(summaries, renderItemSummary(item))

		itemDir := s.backlogLoader.ItemDir(kind, parts[1])
		if snap := renderItemReviewSnapshot(itemDir); snap != "" {
			reviewSnaps = append(reviewSnaps, fmt.Sprintf("## %s/%s\n\n%s", kind, item.Name, snap))
		}
		if item.Status == backlog.StatusCompleted {
			if content := loadItemDeliverable(kind, itemDir); content != "" {
				fmt.Fprintf(&deliverablesBuf, "## %s/%s — %s\n\n%s\n\n", kind, item.Name, item.Title, content)
			}
		}
	}
	return summaries, reviewSnaps, deliverablesBuf.String()
}

// renderItemSummary is intentionally terse — one paragraph. The agent
// drills into deliverables/review snapshots separately.
func renderItemSummary(item backlog.BacklogItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s/%s — %s\n", item.Kind, item.Name, item.Title)
	fmt.Fprintf(&b, "- **Status:** `%s`\n", item.Status)
	if item.Priority > 0 {
		fmt.Fprintf(&b, "- **Priority:** %d\n", item.Priority)
	}
	if item.Effort != "" {
		fmt.Fprintf(&b, "- **Effort:** %s\n", item.Effort)
	}
	if len(item.DependsOn) > 0 {
		fmt.Fprintf(&b, "- **Depends on:** %s\n", strings.Join(item.DependsOn, ", "))
	}
	if item.ArchivedAt != nil {
		fmt.Fprintf(&b, "- **Archived:** %s\n", *item.ArchivedAt)
	}
	if desc := strings.TrimSpace(item.Description); desc != "" {
		b.WriteString("\n")
		b.WriteString(desc)
	}
	return b.String()
}

// renderItemReviewSnapshot returns a short block describing the item's
// latest review round (if any). Empty string if no review has run yet.
func renderItemReviewSnapshot(itemDir string) string {
	latest, _, err := review.LoadLatestRound(itemDir)
	if err != nil || latest == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "- Round %d · status `%s`", latest.RoundNum, latest.Status)
	if latest.Classification != "" {
		fmt.Fprintf(&b, " · classification `%s`", latest.Classification)
	}
	if assessment := strings.TrimSpace(latest.AgentAssessment); assessment != "" {
		b.WriteString("\n\n")
		b.WriteString(assessment)
	}
	if len(latest.Evidence) > 0 {
		fmt.Fprintf(&b, "\n\n*%d evidence item(s) attached in the item's review folder.*", len(latest.Evidence))
	}
	return b.String()
}

// loadItemDeliverable returns the plan.md or conclusion.md content for the
// item (whichever matches the kind's convention), or empty string if absent.
func loadItemDeliverable(kind backlog.BacklogKind, itemDir string) string {
	deliverable := backlog.DeliverableForKind(kind)
	if strings.TrimSpace(deliverable) == "" {
		return ""
	}
	if content := workshop.LoadPlanContentByName(itemDir, deliverable); content != "" {
		return content
	}
	// Fallback: try a few common deliverable filenames. `backlog.DeliverableForKind`
	// is the authoritative mapping; this catches older-shape items.
	for _, name := range []string{"plan.md", "conclusion.md"} {
		data, err := os.ReadFile(filepath.Join(itemDir, name))
		if err == nil {
			return string(data)
		}
	}
	return ""
}

func appendNote(atts []*domainpb.ContextAttachment, key, label, summary, content, format, priority string) []*domainpb.ContextAttachment {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return atts
	}
	return append(atts, &domainpb.ContextAttachment{
		Type:     "note",
		Key:      key,
		Label:    label,
		Summary:  summary,
		Content:  trimmed,
		Format:   format,
		Priority: priority,
	})
}
