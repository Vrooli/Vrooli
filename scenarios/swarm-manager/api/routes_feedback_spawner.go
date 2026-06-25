package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/feedback"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/promptmanager"
)

// feedbackSpawnerConfig bundles the dependencies the spawner adapter
// needs to render the initiative-feedback skill prompt and propagate
// attachments to agent-manager. ActivitySvc, when set, routes spawns
// through the activity tracker so feedback runs share the audit trail
// used by every other agent-spawning surface.
type feedbackSpawnerConfig struct {
	Agent         *agentmanager.AgentService
	ActivitySvc   *agentactivity.Service
	PromptClient  promptmanager.Client
	Materializer  *graph.Materializer
	BacklogStore  backlog.Store
	InitStore     *initiatives.Store
	FeedbackStore *feedback.Store
}

// feedbackSpawnerAdapter implements feedback.AgentSpawner. The spawn flow:
// load initiative + graph + items + prior rounds → render the skill via
// promptmanager → spawn the agent with rendered prompt + image attachments.
type feedbackSpawnerAdapter struct {
	cfg feedbackSpawnerConfig
}

func newFeedbackSpawnerAdapter(cfg feedbackSpawnerConfig) *feedbackSpawnerAdapter {
	if cfg.Agent == nil {
		return nil
	}
	return &feedbackSpawnerAdapter{cfg: cfg}
}

func (a *feedbackSpawnerAdapter) SpawnInitiativeFeedback(ctx context.Context, req feedback.SpawnRequest) (string, error) {
	rendered, attachments, err := a.buildPromptAndAttachments(ctx, req)
	if err != nil {
		// Fall back to the raw submission text so we still spawn an agent
		// (degraded mode); log so operators can spot the rendering gap.
		slog.Warn("feedback: prompt rendering failed; spawning with raw text",
			"err", err, "initiative", req.InitiativeName, "round", req.RoundNumber)
		rendered = req.SubmissionText
		attachments = nil
	}

	purpose := normalizeFeedbackPurpose(req.Purpose)
	spawnReq := agentmanager.InitiativeSpawnRequest{
		Name:               req.InitiativeName,
		Description:        req.SubmissionText,
		Prompt:             rendered,
		Purpose:            req.Purpose,
		RoundNumber:        req.RoundNumber,
		RoundSlug:          req.RoundSlug,
		ContextAttachments: attachments,
	}

	// Prefer the tracked path so the run lands in agent-activity with
	// owner_type=initiative + round metadata. Fall back to the raw
	// agent-manager call only when the activity service isn't wired
	// (e.g. some test harnesses).
	if a.cfg.ActivitySvc != nil {
		spec := agentactivity.Spec{
			OwnerType:   agentactivity.OwnerInitiative,
			OwnerName:   req.InitiativeName,
			OwnerTitle:  a.lookupInitiativeTitle(req.InitiativeName),
			Purpose:     purpose,
			RequestedBy: "swarm-manager",
			Metadata: map[string]string{
				"round_number": fmt.Sprintf("%d", req.RoundNumber),
				"round_slug":   req.RoundSlug,
				"entrypoint":   "initiative.feedback",
			},
		}
		ctx = agentactivity.WithSpec(ctx, spec)
		res, err := a.cfg.ActivitySvc.SpawnInitiative(ctx, spawnReq)
		if err != nil {
			return "", err
		}
		return res.RunID, nil
	}

	res, err := a.cfg.Agent.SpawnInitiative(ctx, spawnReq)
	if err != nil {
		return "", err
	}
	return res.RunID, nil
}

func (a *feedbackSpawnerAdapter) ContinueRun(ctx context.Context, req feedback.ContinueRequest) error {
	if a.cfg.ActivitySvc != nil {
		spec := agentactivity.Spec{
			OwnerType:   agentactivity.OwnerInitiative,
			OwnerName:   req.InitiativeName,
			OwnerTitle:  a.lookupInitiativeTitle(req.InitiativeName),
			Purpose:     agentactivity.PurposeFeedbackContinue,
			RequestedBy: "swarm-manager",
			Metadata: map[string]string{
				"round_number": fmt.Sprintf("%d", req.RoundNumber),
				"round_slug":   req.RoundSlug,
				"entrypoint":   "initiative.feedback.continue",
			},
		}
		ctx = agentactivity.WithSpec(ctx, spec)
		return a.cfg.ActivitySvc.ContinueRun(ctx, req.RunID, req.Message)
	}
	return a.cfg.Agent.ContinueRun(ctx, req.RunID, req.Message)
}

// lookupInitiativeTitle returns the initiative's title for the activity
// record, or the empty string if it can't be resolved (which is OK —
// owner_title is optional in the activity schema).
func (a *feedbackSpawnerAdapter) lookupInitiativeTitle(name string) string {
	if a.cfg.InitStore == nil {
		return ""
	}
	init, err := a.cfg.InitStore.Load(name)
	if err != nil || init == nil {
		return ""
	}
	return init.Title
}

// normalizeFeedbackPurpose maps the feedback service's purpose strings to
// agentactivity.Purpose constants. Unknown values default to
// PurposeFeedback so a misrouted continue (purpose left empty) still
// records something tracked.
func normalizeFeedbackPurpose(p string) agentactivity.Purpose {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case string(agentactivity.PurposeFeedbackContinue), "continue":
		return agentactivity.PurposeFeedbackContinue
	case string(agentactivity.PurposeInitiativeReview), "review":
		return agentactivity.PurposeInitiativeReview
	default:
		return agentactivity.PurposeFeedback
	}
}

// buildPromptAndAttachments hydrates the prompt context from disk and
// renders the initiative-feedback skill via prompt-manager. Errors here
// are non-fatal at the caller — the adapter falls back to the raw text.
func (a *feedbackSpawnerAdapter) buildPromptAndAttachments(
	ctx context.Context,
	req feedback.SpawnRequest,
) (string, []*domainpb.ContextAttachment, error) {
	inputs, err := a.collectPromptInputs(req)
	if err != nil {
		return "", nil, fmt.Errorf("collect prompt inputs: %w", err)
	}
	vars := feedback.BuildPromptVariables(inputs)

	if a.cfg.PromptClient == nil {
		return "", nil, errors.New("prompt client not wired")
	}
	skillID := initiativeFeedbackSkillFallbackID
	if entry, ok := promptcatalog.ResolveInitiativeSkill(req.Purpose); ok && entry.SkillID != "" {
		skillID = entry.SkillID
	}
	prompt, err := a.cfg.PromptClient.ReadSkill(ctx, skillID, vars, false)
	if err != nil {
		return "", nil, fmt.Errorf("read skill %s: %w", skillID, err)
	}
	atts := a.buildContextAttachments(req)
	return prompt, atts, nil
}

func (a *feedbackSpawnerAdapter) collectPromptInputs(req feedback.SpawnRequest) (feedback.PromptInputs, error) {
	in := feedback.PromptInputs{
		InitiativeName: req.InitiativeName,
		ThisFeedback:   req.SubmissionText,
	}

	if a.cfg.InitStore != nil {
		init, err := a.cfg.InitStore.Load(req.InitiativeName)
		if err == nil && init != nil {
			in.InitiativeTitle = init.Title
			in.InitiativeDescription = init.Description
			in.ItemSummaries, in.ItemFolderIndex, in.PriorHandoffs = a.collectItemContext(init.Items)
		}
	}

	if a.cfg.Materializer != nil {
		mg, err := a.cfg.Materializer.ReadGraph(req.InitiativeName)
		if err == nil && mg != nil {
			in.CurrentGraphJSON = feedback.MarshalGraphForPrompt(mg)
		}
	}

	if a.cfg.FeedbackStore != nil {
		rounds, err := a.cfg.FeedbackStore.ListRounds(req.InitiativeName)
		if err == nil {
			for _, r := range rounds {
				if r.Number == req.RoundNumber {
					continue
				}
				in.PriorRounds = append(in.PriorRounds, r)
			}
		}
	}

	in.Attachments = a.collectAttachmentSummaries(req)
	return in, nil
}

// collectItemContext builds the per-item summary, folder index, and the
// list of prior agent handoff/conclusion documents. Items that fail to
// load are skipped — a missing item shouldn't sink the whole prompt.
func (a *feedbackSpawnerAdapter) collectItemContext(refs []string) (
	[]feedback.ItemSummary,
	[]feedback.ItemFolderEntry,
	[]feedback.HandoffSummary,
) {
	var summaries []feedback.ItemSummary
	var folders []feedback.ItemFolderEntry
	var handoffs []feedback.HandoffSummary

	for _, ref := range refs {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			continue
		}
		kind := backlog.BacklogKind(parts[0])
		item, err := a.cfg.BacklogStore.LoadItem(kind, parts[1])
		if err != nil {
			continue
		}
		summaries = append(summaries, feedback.ItemSummary{
			Ref:         ref,
			Title:       item.Title,
			Status:      string(item.Status),
			Priority:    item.Priority,
			Effort:      item.Effort,
			Description: item.Description,
		})
		dir := a.cfg.BacklogStore.ItemDir(kind, parts[1])
		folders = append(folders, feedback.ItemFolderEntry{Ref: ref, Path: dir})
		if h, ok := readHandoffSummary(dir, item.Kind); ok {
			h.Ref = ref
			handoffs = append(handoffs, h)
		}
	}
	return summaries, folders, handoffs
}

// readHandoffSummary returns the deliverable file (plan.md / conclusion.md)
// content for a backlog item if present. Used to give the feedback agent
// a window into what each item has converged on.
func readHandoffSummary(itemDir string, kind backlog.BacklogKind) (feedback.HandoffSummary, bool) {
	deliverable := backlog.DeliverableForKind(kind)
	if deliverable == "" {
		return feedback.HandoffSummary{}, false
	}
	path := filepath.Join(itemDir, deliverable)
	data, err := os.ReadFile(path)
	if err != nil {
		return feedback.HandoffSummary{}, false
	}
	return feedback.HandoffSummary{
		Source:  filepath.Join(filepath.Base(itemDir), deliverable),
		Content: string(data),
	}, true
}

// collectAttachmentSummaries inspects each persisted attachment file so
// the prompt can name it. Failure to stat a single file is non-fatal —
// the agent still gets the bytes via ContextAttachments.
func (a *feedbackSpawnerAdapter) collectAttachmentSummaries(req feedback.SpawnRequest) []feedback.AttachmentSummary {
	if len(req.AttachmentIDs) == 0 || a.cfg.FeedbackStore == nil {
		return nil
	}
	var out []feedback.AttachmentSummary
	for _, id := range req.AttachmentIDs {
		path, ok := a.cfg.FeedbackStore.ResolveAttachment(req.InitiativeName, req.RoundNumber, req.RoundSlug, id)
		if !ok {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		out = append(out, feedback.AttachmentSummary{
			Filename:    filepath.Base(path),
			ContentType: feedback.ContentTypeForAttachment(id),
			SizeBytes:   info.Size(),
		})
	}
	return out
}

// buildContextAttachments converts each persisted attachment into the
// proto shape agent-manager expects. Image bytes are inlined as base64 in
// the `content` field with type=image; the agent's vision pass picks them
// up. Non-resolvable IDs are silently skipped — the prompt's
// ATTACHMENT_IMAGES list is the source of truth for "what was uploaded".
func (a *feedbackSpawnerAdapter) buildContextAttachments(req feedback.SpawnRequest) []*domainpb.ContextAttachment {
	if len(req.AttachmentIDs) == 0 || a.cfg.FeedbackStore == nil {
		return nil
	}
	out := make([]*domainpb.ContextAttachment, 0, len(req.AttachmentIDs))
	for _, id := range req.AttachmentIDs {
		path, ok := a.cfg.FeedbackStore.ResolveAttachment(req.InitiativeName, req.RoundNumber, req.RoundSlug, id)
		if !ok {
			continue
		}
		mediaType := feedback.ContentTypeForAttachment(id)
		// Use absolute path so the agent can read the file via its own
		// filesystem access. Type=image triggers the vision pipeline.
		out = append(out, &domainpb.ContextAttachment{
			Type:    "image",
			Path:    path,
			Label:   filepath.Base(path),
			Format:  formatForMediaType(mediaType),
			Summary: fmt.Sprintf("Image uploaded with feedback round %d", req.RoundNumber),
		})
	}
	return out
}

func formatForMediaType(ct string) string {
	mt, _, _ := mime.ParseMediaType(ct)
	switch strings.ToLower(mt) {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return "image"
	}
	return ""
}
