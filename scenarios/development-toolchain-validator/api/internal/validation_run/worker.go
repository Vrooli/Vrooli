package validation_run

import (
	"context"
	"errors"
	"log"
	"time"

	"development-toolchain-validator/internal/clock"
	manifest "development-toolchain-validator/internal/manifest"
	vr "development-toolchain-validator/internal/validation_record"
)

// WorkerConfig groups the worker's tunables.
type WorkerConfig struct {
	// Concurrency caps simultaneous in-flight runs. Default 2.
	Concurrency int

	// AgentManagerTimeout caps each WaitForTerminal call. Default 30m.
	AgentManagerTimeout time.Duration

	// ToolTimeout caps a single tool invocation (TupleKindTool). A
	// test-genie comprehensive run against a golden is heavy (build +
	// runtime phases), so this is generous. Default 30m. A timeout
	// classifies the run as run_failure (the tool could not complete).
	ToolTimeout time.Duration

	// PollInterval controls how often the worker re-checks for queued
	// runs when no notification has fired. Default 5s.
	PollInterval time.Duration
}

// WorkerDeps wires all seams + recordkeeping the worker needs.
type WorkerDeps struct {
	Repo      Repository
	Records   vr.Service
	AgentMgr  AgentManagerClient
	Tools     ToolRunner
	Sandbox   WorkspaceSandboxClient // optional; nil tolerated
	Goldens   GoldenMaterializer
	Manifests ManifestSource
	Skills    SkillContentSource // optional; nil tolerated
	Clock     clock.Clock
	Logger    *log.Logger
}

// Worker drives queued validation runs to terminal status. One worker
// instance is spawned in main.go.
type Worker struct {
	deps   WorkerDeps
	config WorkerConfig
	notify chan struct{}
}

// NewWorker constructs a Worker with the given dependencies. Use
// (*Worker).Run(ctx) to start the loop; the loop returns when ctx is
// canceled.
func NewWorker(deps WorkerDeps, cfg WorkerConfig) *Worker {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 2
	}
	if cfg.AgentManagerTimeout <= 0 {
		cfg.AgentManagerTimeout = 30 * time.Minute
	}
	if cfg.ToolTimeout <= 0 {
		cfg.ToolTimeout = 30 * time.Minute
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &Worker{
		deps:   deps,
		config: cfg,
		notify: make(chan struct{}, 1),
	}
}

// Notify nudges the worker to check for queued runs immediately.
// Service.Start passes this as its Notify callback.
func (w *Worker) Notify() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

// Run advances queued runs until ctx is canceled. Concurrency cap is
// enforced via a counting semaphore.
func (w *Worker) Run(ctx context.Context) {
	sem := make(chan struct{}, w.config.Concurrency)
	tick := time.NewTicker(w.config.PollInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.notify:
		case <-tick.C:
		}
		// Drain queued runs up to concurrency cap.
		for {
			select {
			case sem <- struct{}{}:
			default:
				goto waitNext
			}
			run, err := w.deps.Repo.ClaimNextQueued(ctx)
			if err != nil {
				<-sem
				if !IsNotFound(err) {
					w.deps.Logger.Printf("validation_run.worker: claim: %v", err)
				}
				break
			}
			go func(r Run) {
				defer func() { <-sem }()
				w.processRun(ctx, r)
			}(run)
		}
	waitNext:
	}
}

func (w *Worker) processRun(ctx context.Context, r Run) {
	now := w.deps.Clock.Now().UTC()
	r.StartedAt = now
	r.Status = StatusRunning
	if err := w.deps.Repo.UpdateStatus(ctx, r); err != nil {
		w.deps.Logger.Printf("validation_run.worker: update running: %v", err)
		return
	}

	materialized, err := w.deps.Goldens.Materialize(ctx, r.GoldenSlug)
	if err != nil {
		w.terminate(ctx, r, vr.VerdictRunFailure, "materialize golden: "+err.Error(), RunSummary{StartedAt: now, EndedAt: w.deps.Clock.Now().UTC()})
		return
	}
	defer func() {
		if materialized.Cleanup != nil {
			if err := materialized.Cleanup(); err != nil {
				w.deps.Logger.Printf("validation_run.worker: cleanup materialized golden %q: %v", materialized.PhysicalPath, err)
			}
		}
	}()

	switch r.TupleKind {
	case vr.TupleKindSkill:
		w.processSkillRun(ctx, r, materialized, now)
	case vr.TupleKindTool:
		w.processToolRun(ctx, r, materialized, now)
	default:
		w.terminate(ctx, r, vr.VerdictRunFailure, "unknown tuple_kind", RunSummary{StartedAt: now, EndedAt: w.deps.Clock.Now().UTC()})
	}
}

func (w *Worker) processSkillRun(ctx context.Context, r Run, materialized MaterializedGolden, startedAt time.Time) {
	// Resolve the skill's instruction text so the sandboxed agent has the
	// actual skill to execute. A missing source or fetch error is not
	// fatal — the adapter falls back to a generic description — but we log
	// it because a skill run without its prompt cannot meaningfully
	// validate the skill.
	var skillPrompt string
	if w.deps.Skills != nil {
		content, err := w.deps.Skills.SkillContent(ctx, r.SubjectID)
		if err != nil {
			w.deps.Logger.Printf("validation_run.worker: fetch skill content for %q: %v", r.SubjectID, err)
		} else {
			skillPrompt = content
		}
	}
	runID, err := w.deps.AgentMgr.StartSandboxedRun(ctx, SandboxedRunSpec{
		SkillID: r.SubjectID, GoldenSlug: r.GoldenSlug, GoldenPath: materialized.PhysicalPath,
		SkillPrompt: skillPrompt,
	})
	if err != nil {
		var unavailable ErrDependencyUnavailable
		if errors.As(err, &unavailable) {
			w.terminate(ctx, r, vr.VerdictRunFailure, err.Error(), RunSummary{StartedAt: startedAt, EndedAt: w.deps.Clock.Now().UTC()})
			return
		}
		w.terminate(ctx, r, vr.VerdictRunFailure, "agent-manager start: "+err.Error(), RunSummary{StartedAt: startedAt, EndedAt: w.deps.Clock.Now().UTC()})
		return
	}
	r.AgentManagerRunID = runID
	r.Status = StatusEvaluating
	if err := w.deps.Repo.UpdateStatus(ctx, r); err != nil {
		w.deps.Logger.Printf("validation_run.worker: update evaluating: %v", err)
	}
	summary, err := w.deps.AgentMgr.WaitForTerminal(ctx, runID, w.config.AgentManagerTimeout)
	if err != nil {
		w.terminate(ctx, r, vr.VerdictRunFailure, "agent-manager wait: "+err.Error(),
			RunSummary{AgentManagerRunID: runID, StartedAt: startedAt, EndedAt: w.deps.Clock.Now().UTC()})
		return
	}
	summary.AgentManagerRunID = runID
	summary.DiffPaths = NormalizeDiffPaths(summary.DiffPaths, materialized)
	man, manErr := w.deps.Manifests.GetManifest(ctx, r.SubjectID, r.GoldenSlug)
	if manErr != nil {
		// No manifest pinned for this tuple: classify as RUN_FAILURE
		// with a clear cause; an operator must pin a manifest first.
		w.terminate(ctx, r, vr.VerdictRunFailure, "no manifest pinned for (skill, golden): "+manErr.Error(), summary)
		return
	}
	verdict := Evaluate(EvaluatorInput{Manifest: man, Summary: summary})
	w.persistTerminal(ctx, r, verdict.Verdict, verdict.ErrorMessage, summary, man, verdict.Violations, nil)
}

func (w *Worker) processToolRun(ctx context.Context, r Run, materialized MaterializedGolden, startedAt time.Time) {
	toolCtx, cancel := context.WithTimeout(ctx, w.config.ToolTimeout)
	defer cancel()
	res, err := w.deps.Tools.Invoke(toolCtx, r.SubjectID, r.GoldenSlug, materialized.PhysicalPath)
	if err != nil {
		// A seam-level error means the tool could not be run at all
		// (unknown tool, unreadable expectation config): a run failure,
		// not a tool/template regression.
		w.terminateTool(ctx, r, vr.VerdictRunFailure, "tool invoke: "+err.Error(),
			RunSummary{StartedAt: startedAt, EndedAt: w.deps.Clock.Now().UTC()}, &res)
		return
	}
	summary := RunSummary{StartedAt: res.StartedAt, EndedAt: res.EndedAt}
	verdict := Evaluate(EvaluatorInput{ToolResult: &res, Summary: summary})
	w.terminateTool(ctx, r, verdict.Verdict, verdict.ErrorMessage, summary, &res)
}

func (w *Worker) terminate(ctx context.Context, r Run, verdict vr.Verdict, msg string, summary RunSummary) {
	w.persistTerminal(ctx, r, verdict, msg, summary, manifest.Manifest{}, nil, nil)
}

// terminateTool persists a terminal tool run, carrying the tool's
// expectation detail and captured output into the record.
func (w *Worker) terminateTool(ctx context.Context, r Run, verdict vr.Verdict, msg string, summary RunSummary, tool *ToolResult) {
	w.persistTerminal(ctx, r, verdict, msg, summary, manifest.Manifest{}, nil, tool)
}

func (w *Worker) persistTerminal(ctx context.Context, r Run, verdict vr.Verdict, msg string, summary RunSummary, man manifest.Manifest, _ []manifest.Violation, tool *ToolResult) {
	endedAt := summary.EndedAt
	if endedAt.IsZero() {
		endedAt = w.deps.Clock.Now().UTC()
	}
	r.Status = StatusTerminal
	r.TerminalVerdict = verdict
	r.EndedAt = endedAt
	r.ErrorMessage = msg
	if summary.AgentManagerRunID != "" {
		r.AgentManagerRunID = summary.AgentManagerRunID
	}
	if err := w.deps.Repo.UpdateStatus(ctx, r); err != nil {
		w.deps.Logger.Printf("validation_run.worker: update terminal %q: %v", r.ID, err)
	}
	var toolDetail, toolRawOutput string
	if tool != nil {
		toolDetail = tool.Detail
		toolRawOutput = string(tool.RawOutput)
	}
	_, err := w.deps.Records.Append(ctx, vr.AppendInput{
		TupleKind:                    r.TupleKind,
		SubjectID:                    r.SubjectID,
		GoldenSlug:                   r.GoldenSlug,
		StartedAt:                    summary.StartedAt,
		EndedAt:                      endedAt,
		TokensUsed:                   summary.TokensUsed,
		CostUSDMicro:                 summary.CostUSDMicro,
		Verdict:                      verdict,
		DiffHash:                     summary.DiffHash,
		DiffPathCount:                int32(len(summary.DiffPaths)),
		AgentManagerRunID:            summary.AgentManagerRunID,
		ManifestTemplateVersionAtRun: man.TemplateVersionPinned,
		ManifestSkillVersionAtRun:    man.SkillVersionPinned,
		ErrorMessage:                 msg,
		ToolDetail:                   toolDetail,
		ToolRawOutput:                toolRawOutput,
	})
	if err != nil {
		w.deps.Logger.Printf("validation_run.worker: append record %q: %v", r.ID, err)
	}
}
