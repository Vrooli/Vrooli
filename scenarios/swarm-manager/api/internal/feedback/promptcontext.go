package feedback

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// PromptVariables is the variable map handed to the prompt-manager skill
// renderer. Keys mirror the placeholders in
// `swarm-manager-initiative-feedback/SKILL.md`.
type PromptVariables = map[string]string

// PromptInputs carries the data the spawner adapter has already loaded
// from disk so BuildPromptVariables can stay pure (no I/O). The adapter
// owns the read paths; this helper just composes the strings.
type PromptInputs struct {
	InitiativeName        string
	InitiativeTitle       string
	InitiativeDescription string

	// CurrentGraphJSON is the raw graph.json bytes for the initiative.
	// Empty string is allowed — the rendered prompt simply gets `{}`.
	CurrentGraphJSON string

	// ItemSummaries lists each backlog item in the initiative.
	ItemSummaries []ItemSummary

	// PriorRounds lists prior feedback rounds (excluding the current one)
	// with their decisions so the agent can avoid re-proposing what has
	// already been rejected/accepted.
	PriorRounds []Round

	// PriorHandoffs collects per-item handoff/conclusion summaries the
	// adapter has gathered. Each entry is a self-contained markdown
	// fragment; the helper concatenates them with a separator.
	PriorHandoffs []HandoffSummary

	// ItemFolderIndex maps each item ref to its on-disk folder so the
	// agent can drill in via read-only CLI calls.
	ItemFolderIndex []ItemFolderEntry

	// Attachments describes the images attached to this round. The actual
	// image bytes ride along as ContextAttachments — this string is the
	// human-readable list the prompt references in the body.
	Attachments []AttachmentSummary

	// ThisFeedback is the user's submission text for this round.
	ThisFeedback string
}

// ItemSummary is one row in the rendered ITEM_SUMMARIES table.
type ItemSummary struct {
	Ref         string
	Title       string
	Status      string
	Priority    int
	Effort      string
	Description string
}

// HandoffSummary is one collected agent handoff/conclusion.
type HandoffSummary struct {
	Ref     string // "kind/name"
	Source  string // file path the summary came from
	Content string
}

// ItemFolderEntry maps an item ref to its folder path on disk.
type ItemFolderEntry struct {
	Ref  string
	Path string
}

// AttachmentSummary describes a single attachment the user uploaded.
type AttachmentSummary struct {
	Filename    string
	ContentType string
	SizeBytes   int64
}

// BuildPromptVariables composes the rendered prompt-variable map from
// loaded inputs. Pure: no I/O, no time calls.
func BuildPromptVariables(in PromptInputs) PromptVariables {
	graph := strings.TrimSpace(in.CurrentGraphJSON)
	if graph == "" {
		graph = "{}"
	}
	return PromptVariables{
		"INITIATIVE_NAME":        in.InitiativeName,
		"INITIATIVE_TITLE":       in.InitiativeTitle,
		"INITIATIVE_DESCRIPTION": strings.TrimSpace(in.InitiativeDescription),
		"CURRENT_GRAPH":          graph,
		"ITEM_SUMMARIES":         renderItemSummaries(in.ItemSummaries),
		"PRIOR_FEEDBACK":         renderPriorRounds(in.PriorRounds),
		"PRIOR_HANDOFFS":         renderHandoffs(in.PriorHandoffs),
		"ITEM_FOLDER_INDEX":      renderFolderIndex(in.ItemFolderIndex),
		"THIS_FEEDBACK":          strings.TrimSpace(in.ThisFeedback),
		"ATTACHMENT_IMAGES":      renderAttachments(in.Attachments),
	}
}

func renderItemSummaries(items []ItemSummary) string {
	if len(items) == 0 {
		return "_(no items in initiative)_"
	}
	sorted := append([]ItemSummary(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Ref < sorted[j].Ref })

	var b strings.Builder
	for _, it := range sorted {
		fmt.Fprintf(&b, "- **%s** — %s [status=%s, priority=%d", it.Ref, it.Title, it.Status, it.Priority)
		if it.Effort != "" {
			fmt.Fprintf(&b, ", effort=%s", it.Effort)
		}
		b.WriteString("]\n")
		if desc := strings.TrimSpace(it.Description); desc != "" {
			fmt.Fprintf(&b, "  %s\n", truncatePromptString(desc, 240))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderPriorRounds(rounds []Round) string {
	if len(rounds) == 0 {
		return "_(no prior feedback rounds)_"
	}
	sorted := append([]Round(nil), rounds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Number < sorted[j].Number })

	var b strings.Builder
	for _, r := range sorted {
		fmt.Fprintf(&b, "### Round %03d (%s) — %s\n", r.Number, r.Type, r.Status)
		if txt := strings.TrimSpace(r.Submission.Text); txt != "" {
			fmt.Fprintf(&b, "Submission: %s\n", truncatePromptString(txt, 240))
		}
		if r.Decision != nil {
			fmt.Fprintf(&b, "Decision: %s", r.Decision.Kind)
			if rationale := strings.TrimSpace(r.Decision.Rationale); rationale != "" {
				fmt.Fprintf(&b, " — %s", truncatePromptString(rationale, 200))
			}
			b.WriteString("\n")
			if len(r.Decision.AcceptedMutationIDs) > 0 {
				fmt.Fprintf(&b, "Accepted: %s\n", strings.Join(r.Decision.AcceptedMutationIDs, ", "))
			}
			if len(r.Decision.RejectedMutationIDs) > 0 {
				fmt.Fprintf(&b, "Rejected: %s\n", strings.Join(r.Decision.RejectedMutationIDs, ", "))
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderHandoffs(handoffs []HandoffSummary) string {
	if len(handoffs) == 0 {
		return "_(no prior agent handoffs)_"
	}
	var b strings.Builder
	for _, h := range handoffs {
		fmt.Fprintf(&b, "### %s\n", h.Ref)
		if src := strings.TrimSpace(h.Source); src != "" {
			fmt.Fprintf(&b, "_Source: %s_\n", src)
		}
		if content := strings.TrimSpace(h.Content); content != "" {
			b.WriteString(truncatePromptString(content, 1000))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderFolderIndex(entries []ItemFolderEntry) string {
	if len(entries) == 0 {
		return "_(no items)_"
	}
	sorted := append([]ItemFolderEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Ref < sorted[j].Ref })

	var b strings.Builder
	for _, e := range sorted {
		fmt.Fprintf(&b, "- %s → `%s`\n", e.Ref, e.Path)
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderAttachments(atts []AttachmentSummary) string {
	if len(atts) == 0 {
		return "_(no attachments)_"
	}
	var b strings.Builder
	b.WriteString("Attached images (raw bytes provided as ContextAttachment):\n")
	for _, a := range atts {
		ct := a.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		fmt.Fprintf(&b, "- %s (%s, %d bytes)\n", a.Filename, ct, a.SizeBytes)
	}
	return strings.TrimRight(b.String(), "\n")
}

// truncatePromptString trims a string to at most max runes, appending an
// ellipsis when truncated. Conservative — we keep the prompt budget for the
// agent's reasoning, not for context dumps.
func truncatePromptString(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// MarshalGraphForPrompt encodes a value (typically the materialized graph
// struct) as compact JSON suitable for the CURRENT_GRAPH variable. Returns
// "{}" on error so the prompt is never empty.
func MarshalGraphForPrompt(v any) string {
	if v == nil {
		return "{}"
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}
