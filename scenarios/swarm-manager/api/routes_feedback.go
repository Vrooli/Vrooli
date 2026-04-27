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
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/feedback"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/proposals"
)

// initiativeFeedbackSkillFallbackID is used when the catalog lookup fails
// (e.g. catalog entry got removed mid-deploy). Hard-coded so feedback
// keeps working even if the catalog drifts.
const initiativeFeedbackSkillFallbackID = "swarm-manager-initiative-feedback"

var errFeedbackExecutionUnavailable = errors.New("execution service is not wired; interrupt_in_progress unavailable")

// registerFeedbackRoutes wires the feedback package: proposals.Applier,
// feedback.Service (with agent-manager spawner adapter, graph-backed state
// builder, on-disk lock, and the agent-state poller used by Handler.Get
// to advance rounds), and mounts feedback.Handler on the router.
func (s *Server) registerFeedbackRoutes(materializer *graph.Materializer) {
	if s.backlogHandler == nil || s.initiativeService == nil || s.initStore == nil {
		return
	}

	// Guard against the typed-nil-into-interface trap: ServiceConfig.Events
	// is the CreationEventEmitter interface. Assigning a nil *eventlog.Emitter
	// would yield a non-nil interface that passes the `s.events != nil` check
	// in Service.Create and then panic on first method call. Only set Events
	// when we actually have a constructed emitter.
	cfg := backlog.ServiceConfig{
		Store:       s.backlogHandler.Store(),
		Assigner:    s.initiativeService,
		Invalidator: materializer,
		// Workshop and CycleChecker intentionally omitted: proposal-
		// applied items skip auto-workshop (agent already chose the
		// item) and cycle validation is performed by the Applier's
		// Validate phase using CurrentState.
	}
	if s.emitter != nil {
		cfg.Events = s.emitter
	}
	creator, err := backlog.NewService(cfg)
	if err != nil {
		slog.Warn("feedback: failed to build backlog.Service for proposals", "err", err)
		return
	}

	applier, err := proposals.NewApplier(proposals.Config{
		Store:       s.backlogHandler.Store(),
		Assigner:    s.initiativeService,
		Creator:     creator,
		Canceller:   newExecutionCancellerAdapter(s.executionSvc),
		Invalidator: materializer,
		Events:      &feedbackEventEmitter{eventlog: s.emitter},
	})
	if err != nil {
		slog.Warn("feedback: failed to build proposals.Applier", "err", err)
		return
	}

	store := feedback.NewStore(s.initiativeService.InitDir)
	lock := &initiativelock.Lock{Dir: s.initiativeService.InitDir}
	if s.initStore != nil {
		sweepStaleFeedbackLocks(s.initStore, lock)
	}

	promptClient := promptmanager.NewHTTPClient()
	spawner := newFeedbackSpawnerAdapter(feedbackSpawnerConfig{
		Agent:         s.agentSvc,
		ActivitySvc:   s.agentActivitySvc,
		PromptClient:  promptClient,
		Materializer:  materializer,
		BacklogStore:  s.backlogHandler.Store(),
		InitStore:     s.initStore,
		FeedbackStore: store,
	})
	stateBuilder := newFeedbackStateBuilder(materializer, s.initStore, s.backlogHandler.Store())
	activity := newFeedbackActivityChecker(s.agentActivitySvc, s.initStore)
	poller := newFeedbackPoller(s.agentSvc)
	canceller := newFeedbackCanceller(s.agentSvc)

	svc, err := feedback.NewService(feedback.Config{
		Store:        store,
		Lock:         lock,
		Spawner:      spawner,
		Activity:     activity,
		Poller:       poller,
		Canceller:    canceller,
		Apply:        applier,
		StateBuilder: stateBuilder,
	})
	if err != nil {
		slog.Warn("feedback: failed to build Service", "err", err)
		return
	}

	handler := feedback.NewHandler(svc)
	handler.RegisterRoutes(s.router)

	// Stuck-round safety net: synchronous boot-time sweep clears any
	// rounds left in agent_thinking from a prior process (e.g. server
	// crashed while a feedback agent was running). The ticker goroutine
	// then keeps the invariant going for the lifetime of the process.
	sweeper := feedback.NewSweeper(svc, &initiativeNameLister{store: s.initStore})
	if dismissed, err := sweeper.RunOnce(context.Background()); err != nil {
		slog.Warn("feedback: boot-time stuck-round sweep failed", "err", err)
	} else if dismissed > 0 {
		slog.Info("feedback: boot-time stuck-round sweep dismissed rounds", "count", dismissed)
	}
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			<-s.feedbackSweeperStop
			cancel()
		}()
		sweeper.Start(ctx)
	}()
}

// initiativeNameLister adapts initiatives.Store to feedback.InitiativeLister
// so the sweeper can enumerate initiatives without coupling the feedback
// package to the initiatives package.
type initiativeNameLister struct {
	store *initiatives.Store
}

func (l *initiativeNameLister) ListNames() ([]string, error) {
	if l == nil || l.store == nil {
		return nil, nil
	}
	all, err := l.store.LoadAll()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	for _, i := range all {
		if strings.TrimSpace(i.Name) == "" {
			continue
		}
		names = append(names, i.Name)
	}
	return names, nil
}

// --- Adapters ----------------------------------------------------------

// executionCancellerAdapter wires execution.Service into the proposals
// Applier's ExecutionCanceller interface — finds the most recent
// non-terminal execution for the backlog ref and cancels it.
type executionCancellerAdapter struct {
	svc *execution.Service
}

func newExecutionCancellerAdapter(svc *execution.Service) *executionCancellerAdapter {
	if svc == nil {
		return nil
	}
	return &executionCancellerAdapter{svc: svc}
}

func (a *executionCancellerAdapter) CancelForBacklog(ctx context.Context, kind, name string) error {
	if a == nil || a.svc == nil {
		return errFeedbackExecutionUnavailable
	}
	records, err := a.svc.List(ctx, execution.ListFilters{
		BacklogKind: kind,
		BacklogName: name,
	})
	if err != nil {
		return err
	}
	for _, r := range records {
		if isCancelableStatus(r.Status) {
			if _, err := a.svc.Cancel(ctx, r.ExecutionID); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func isCancelableStatus(s execution.Status) bool {
	switch s {
	case execution.StatusPending, execution.StatusStarting, execution.StatusRunning,
		execution.StatusNeedsReview, execution.StatusValidating, execution.StatusNeedsFixup:
		return true
	}
	return false
}

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

// newFeedbackStateBuilder closes over the materializer + initiative store
// so the feedback service can rebuild proposal CurrentState from disk on
// each Decide call.
func newFeedbackStateBuilder(
	m *graph.Materializer,
	initStore *initiatives.Store,
	backlogStore backlog.Store,
) func(string) (proposals.CurrentState, error) {
	return func(initiativeName string) (proposals.CurrentState, error) {
		mg, err := m.ReadGraph(initiativeName)
		if err != nil {
			return proposals.CurrentState{}, fmt.Errorf("read initiative graph: %w", err)
		}
		// First-touch materialization: initiatives without a graph.json
		// yet get one synchronously so proposal validation reasons
		// against real nodes. Silently falling back to a nil graph
		// would misreport every existing-target mutation as
		// "unknown target".
		if mg == nil {
			if mErr := m.MaterializeInitiative(context.Background(), initiativeName); mErr != nil {
				return proposals.CurrentState{}, fmt.Errorf("materialize initiative graph: %w", mErr)
			}
			mg, err = m.ReadGraph(initiativeName)
			if err != nil {
				return proposals.CurrentState{}, fmt.Errorf("read initiative graph after materialize: %w", err)
			}
			if mg == nil {
				return proposals.CurrentState{}, fmt.Errorf("initiative %q graph is still unavailable after materialize", initiativeName)
			}
		}
		known, err := knownInitiativeNames(initStore)
		if err != nil {
			return proposals.CurrentState{}, err
		}
		inProgress := inProgressRefs(backlogStore, mg)
		return proposals.FromMaterializedGraph(mg, known, inProgress)
	}
}

func knownInitiativeNames(store *initiatives.Store) ([]string, error) {
	all, err := store.LoadAll()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	for _, i := range all {
		if strings.TrimSpace(i.Name) == "" {
			continue
		}
		names = append(names, i.Name)
	}
	return names, nil
}

func inProgressRefs(store backlog.Store, mg *graph.MaterializedGraph) []string {
	if mg == nil {
		return nil
	}
	refs := make([]string, 0)
	for _, n := range mg.Nodes {
		item, err := store.LoadItem(backlog.BacklogKind(n.Kind), n.Name)
		if err != nil {
			continue
		}
		if item.Status == backlog.StatusInProgress {
			refs = append(refs, n.ID)
		}
	}
	return refs
}

// --- Activity / Poller / Events ---------------------------------------

// feedbackActivityChecker enumerates each initiative item and asks
// agentactivity.Service whether a backlog-owned agent is running for it.
type feedbackActivityChecker struct {
	svc       *agentactivity.Service
	initStore *initiatives.Store
}

func newFeedbackActivityChecker(svc *agentactivity.Service, initStore *initiatives.Store) *feedbackActivityChecker {
	if svc == nil || initStore == nil {
		return nil
	}
	return &feedbackActivityChecker{svc: svc, initStore: initStore}
}

func (c *feedbackActivityChecker) ActiveRunsForInitiative(initiativeName string) ([]feedback.ItemActivity, error) {
	if c == nil {
		return nil, nil
	}
	init, err := c.initStore.Load(initiativeName)
	if err != nil || init == nil {
		return nil, err
	}
	var out []feedback.ItemActivity
	for _, ref := range init.Items {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			continue
		}
		if !c.svc.HasActiveAgent(context.Background(), parts[0], parts[1]) {
			continue
		}
		records, listErr := c.svc.List(context.Background(), agentactivity.ListFilters{
			OwnerType:  string(agentactivity.OwnerBacklog),
			OwnerKind:  parts[0],
			OwnerName:  parts[1],
			ActiveOnly: true,
		})
		if listErr != nil || len(records) == 0 {
			out = append(out, feedback.ItemActivity{Ref: ref})
			continue
		}
		out = append(out, feedback.ItemActivity{
			Ref:     ref,
			RunID:   records[0].RunID,
			Purpose: string(records[0].Purpose),
		})
	}
	return out, nil
}

// feedbackPoller adapts agentmanager.AgentService to feedback.AgentRunPoller
// so Handler.Get can advance rounds via the same pull pattern that
// clarification uses (see internal/backlog/clarification_state.go).
type feedbackPoller struct {
	agent *agentmanager.AgentService
}

func newFeedbackPoller(agent *agentmanager.AgentService) *feedbackPoller {
	if agent == nil {
		return nil
	}
	return &feedbackPoller{agent: agent}
}

func (p *feedbackPoller) IsEnabled() bool {
	return p != nil && p.agent != nil && p.agent.IsEnabled()
}

func (p *feedbackPoller) GetRunState(ctx context.Context, runID string) (feedback.RunState, error) {
	state, err := p.agent.GetRunState(ctx, runID)
	if err != nil {
		return feedback.RunState{}, err
	}
	return feedback.RunState{
		Status:   state.Status,
		Summary:  state.Summary,
		ErrorMsg: state.ErrorMsg,
	}, nil
}

// feedbackCanceller adapts agentmanager.AgentService to feedback.RunCanceller.
// Used by the override path to cancel the preempted holder (and any busy
// item runs) before the new round takes the lock — turning "override" from
// a lock-file rename into actual single-agent enforcement.
//
// Nil-safe: when the agent service is not wired in (tests, degraded mode),
// we return nil so feedback.Service treats the canceller as absent and
// falls back to the plain lock-overwrite behavior rather than panicking.
type feedbackCanceller struct {
	agent *agentmanager.AgentService
}

func newFeedbackCanceller(agent *agentmanager.AgentService) *feedbackCanceller {
	if agent == nil {
		return nil
	}
	return &feedbackCanceller{agent: agent}
}

func (c *feedbackCanceller) StopRun(ctx context.Context, runID string) error {
	if c == nil || c.agent == nil {
		return nil
	}
	if strings.TrimSpace(runID) == "" {
		return nil
	}
	return c.agent.StopRun(ctx, runID)
}

// feedbackEventEmitter satisfies proposals.EventEmitter by appending a
// EventBacklogProposalApplied to the durable event log so attribution
// (feedback round / review round, decided-by, mutation id, op, target)
// survives restarts and is queryable per affected backlog item. A nil
// eventlog falls through to slog so test wiring still records the call.
type feedbackEventEmitter struct {
	eventlog *eventlog.Emitter
}

func (e *feedbackEventEmitter) EmitProposalMutationApplied(source proposals.Source, m proposals.Mutation) {
	target := proposalEventTarget(m)
	payload := eventlog.ProposalAppliedPayload{
		InitiativeName:  source.InitiativeName,
		FeedbackRoundID: source.FeedbackRoundID,
		ReviewRoundID:   source.ReviewRoundID,
		RoundNumber:     source.RoundNumber,
		RoundSlug:       source.RoundSlug,
		Entrypoint:      source.Entrypoint,
		DecidedBy:       source.DecidedBy,
		MutationID:      m.ID,
		Op:              string(m.Op),
		Target:          target,
	}
	if e.eventlog != nil {
		e.eventlog.EmitBacklogProposalApplied(target, payload)
		return
	}
	slog.Info("proposals: mutation applied (no eventlog wired)",
		"initiative", source.InitiativeName,
		"feedback_round", source.FeedbackRoundID,
		"mutation_id", m.ID,
		"op", m.Op,
		"target", target,
	)
}

// proposalEventTarget picks the backlog ref to attach the event to:
// add_item / split_item use the new item's ref; edge ops use From; all
// others use Target. Empty when the op has no natural per-item entity.
func proposalEventTarget(m proposals.Mutation) string {
	switch m.Op {
	case proposals.OpAddItem:
		if m.Item != nil {
			return m.Item.Ref()
		}
	case proposals.OpAddEdge, proposals.OpRemoveEdge:
		return m.From
	}
	return m.Target
}

// sweepStaleFeedbackLocks releases lock files older than MaxAge for every
// initiative on disk. Run on boot so a server crash doesn't strand the
// initiative behind a dead lock.
func sweepStaleFeedbackLocks(initStore *initiatives.Store, lock *initiativelock.Lock) {
	all, err := initStore.LoadAll()
	if err != nil {
		slog.Warn("feedback: stale-lock sweep skipped", "err", err)
		return
	}
	swept := 0
	for _, init := range all {
		if init.Name == "" {
			continue
		}
		ok, err := lock.SweepStale(init.Name)
		if err != nil {
			slog.Warn("feedback: stale-lock sweep error", "initiative", init.Name, "err", err)
			continue
		}
		if ok {
			slog.Info("feedback: swept stale lock", "initiative", init.Name)
			swept++
		}
	}
	if swept > 0 {
		slog.Info("feedback: swept stale locks on boot", "count", swept)
	}
}
