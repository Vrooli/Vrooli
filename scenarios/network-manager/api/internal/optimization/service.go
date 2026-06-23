package optimization

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"network-manager/internal/adapters"
	"network-manager/internal/snapshot"
)

type Service struct {
	repo         Repository
	capabilities CapabilitySource
	snapshots    SnapshotReader
	runner       SnapshotRunner
	applier      Applier
	now          func() time.Time
}

type Config struct {
	Repo         Repository
	Capabilities CapabilitySource
	Snapshots    SnapshotReader
	Runner       SnapshotRunner
	Applier      Applier
	Now          func() time.Time
}

func NewService(cfg Config) *Service {
	s := &Service{
		repo:         cfg.Repo,
		capabilities: cfg.Capabilities,
		snapshots:    cfg.Snapshots,
		runner:       cfg.Runner,
		applier:      cfg.Applier,
		now:          cfg.Now,
	}
	if s.applier == nil {
		s.applier = ManualApplier{}
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}
	return s
}

func (s *Service) CreateRun(ctx context.Context, scoringProfile string, dryRun bool) (Run, error) {
	profile := normalizeProfile(scoringProfile)
	baseline, err := s.latestBaseline(ctx)
	if err != nil {
		return Run{}, err
	}
	caps, err := s.capabilities.ListCapabilities(ctx)
	if err != nil {
		return Run{}, err
	}
	run := Run{
		ID:                 uuid.NewString(),
		Status:             "draft",
		ScoringProfile:     profile,
		BaselineSnapshotID: baseline.ID,
		Recommendation:     fmt.Sprintf("Baseline %s is ready; run candidates before scoring.", baseline.ID),
		CreatedAt:          s.now().UTC(),
		UpdatedAt:          s.now().UTC(),
	}
	run.Candidates = candidatesFromCapabilities(run, caps)
	if len(run.Candidates) == 0 {
		run.Status = "manual_required"
		run.Recommendation = "No supported optimization-capable adapters are available; keep read-only snapshots and configure a supported resolver adapter."
		run.Candidates = []Candidate{manualCandidate(run, "manual-optimization-review", "No supported adapter can safely test optimization changes on this platform.")}
	}
	if dryRun {
		run.ID = "optimization-dry-run"
		run.Status = "dry_run"
		run.Recommendation = "Dry run only; no optimization ledger was persisted."
		return run, nil
	}
	if s.repo == nil {
		return Run{}, fmt.Errorf("optimization repository is required")
	}
	saved, err := s.repo.SaveRun(ctx, run)
	if err != nil {
		return Run{}, err
	}
	for _, c := range run.Candidates {
		c.RunID = saved.ID
		if _, err := s.repo.SaveCandidate(ctx, c); err != nil {
			return Run{}, err
		}
	}
	return s.repo.GetRun(ctx, saved.ID)
}

func (s *Service) RunCandidate(ctx context.Context, runID, candidateID string) (Run, error) {
	run, candidate, err := s.loadRunCandidate(ctx, runID, candidateID)
	if err != nil {
		return Run{}, err
	}
	if candidate.Status == "manual_required" {
		run.Status = "manual_required"
		run.Recommendation = "Candidate requires manual execution; no network settings were changed."
		return s.updateRunWithCandidate(ctx, run, candidate)
	}
	if s.runner == nil {
		candidate.Status = "manual_required"
		candidate.Evidence = append(candidate.Evidence, "Snapshot runner unavailable; candidate cannot gather comparable evidence.")
		run.Status = "manual_required"
		return s.updateRunWithCandidate(ctx, run, candidate)
	}
	run.Status = "baseline_running"
	run.UpdatedAt = s.now().UTC()
	if _, err := s.repo.UpdateRun(ctx, run); err != nil {
		return Run{}, err
	}
	snap, err := s.runner.Run(ctx, "optimization-candidate", false)
	if err != nil {
		candidate.Status = "failed"
		candidate.Evidence = append(candidate.Evidence, fmt.Sprintf("Candidate snapshot failed: %v", err))
		run.Status = "failed"
		return s.updateRunWithCandidate(ctx, run, candidate)
	}
	candidate.CandidateSnapshotID = snap.ID
	candidate.Status = "candidate_captured"
	candidate.Evidence = append(candidate.Evidence, fmt.Sprintf("Candidate snapshot %s captured: %s", snap.ID, snap.Summary))
	run.Status = "candidates_running"
	run.Recommendation = "Candidate evidence captured; score candidates before approval."
	return s.updateRunWithCandidate(ctx, run, candidate)
}

func (s *Service) Score(ctx context.Context, runID string) (Run, error) {
	run, err := s.repo.GetRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return Run{}, err
	}
	approvalNeeded := false
	for i := range run.Candidates {
		c := run.Candidates[i]
		c.Score = s.scoreCandidate(ctx, c)
		if c.Status != "manual_required" && c.Status != "failed" {
			c.Status = "scored"
		}
		c.Evidence = append(c.Evidence, fmt.Sprintf("Reliability-first score %.2f calculated from baseline and candidate evidence.", c.Score))
		c.UpdatedAt = s.now().UTC()
		if c.ApprovalRequired {
			approvalNeeded = true
		}
		if _, err := s.repo.UpdateCandidate(ctx, c); err != nil {
			return Run{}, err
		}
	}
	run.Status = "scored"
	run.Recommendation = "Scores are ready; review evidence before approval."
	if approvalNeeded {
		run.Status = "awaiting_approval"
		run.Recommendation = "At least one candidate requires explicit approval before persistent apply."
	}
	run.UpdatedAt = s.now().UTC()
	if _, err := s.repo.UpdateRun(ctx, run); err != nil {
		return Run{}, err
	}
	return s.repo.GetRun(ctx, run.ID)
}

func (s *Service) Approve(ctx context.Context, runID, candidateID string, approved bool) (Run, error) {
	run, candidate, err := s.loadRunCandidate(ctx, runID, candidateID)
	if err != nil {
		return Run{}, err
	}
	if !approved {
		candidate.Status = "approval_required"
		candidate.Evidence = append(candidate.Evidence, "Persistent optimization apply requires --approved acknowledgement.")
		run.Status = "awaiting_approval"
		return s.updateRunWithCandidate(ctx, run, candidate)
	}
	if candidate.Status == "manual_required" || !candidate.RollbackSupported {
		candidate.Status = "manual_required"
		candidate.Evidence = append(candidate.Evidence, "Candidate lacks automatic rollback support; use manual steps instead of persistent apply.")
		run.Status = "manual_required"
		return s.updateRunWithCandidate(ctx, run, candidate)
	}
	if _, err := s.repo.SaveApproval(ctx, ApprovalRecord{
		ID:          uuid.NewString(),
		RunID:       run.ID,
		CandidateID: candidate.ID,
		Approved:    true,
		Note:        "operator approved persistent optimization apply",
		CreatedAt:   s.now().UTC(),
	}); err != nil {
		return Run{}, err
	}
	result, err := s.applier.Apply(ctx, run, candidate)
	if err != nil {
		if errors.Is(err, ErrManualRequired) {
			candidate.Status = "manual_required"
			candidate.Evidence = append(candidate.Evidence, result.Evidence...)
			run.Status = "manual_required"
			return s.updateRunWithCandidate(ctx, run, candidate)
		}
		candidate.Status = "apply_failed"
		candidate.Evidence = append(candidate.Evidence, err.Error())
		run.Status = "failed"
		return s.updateRunWithCandidate(ctx, run, candidate)
	}
	candidate.Status = "applied"
	candidate.RollbackHandle = result.RollbackHandle
	candidate.Evidence = append(candidate.Evidence, result.Evidence...)
	if s.runner != nil {
		if snap, err := s.runner.Run(ctx, "optimization-after", false); err == nil {
			candidate.AfterSnapshotID = snap.ID
			candidate.Evidence = append(candidate.Evidence, fmt.Sprintf("After snapshot %s captured: %s", snap.ID, snap.Summary))
			run.Status = "verified"
		} else {
			run.Status = "applied"
			candidate.Evidence = append(candidate.Evidence, fmt.Sprintf("After snapshot failed: %v", err))
		}
	} else {
		run.Status = "applied"
	}
	run.Recommendation = "Approved candidate applied; review after snapshot and rollback if degraded."
	return s.updateRunWithCandidate(ctx, run, candidate)
}

func (s *Service) Rollback(ctx context.Context, runID string) (Run, error) {
	run, err := s.repo.GetRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return Run{}, err
	}
	candidate, ok := appliedCandidate(run.Candidates)
	if !ok {
		run.Status = "manual_required"
		run.Recommendation = "No applied candidate with rollback state is available."
		run.UpdatedAt = s.now().UTC()
		return s.repo.UpdateRun(ctx, run)
	}
	result, err := s.applier.Rollback(ctx, run, candidate)
	if err != nil {
		candidate.Status = "rollback_failed"
		candidate.Evidence = append(candidate.Evidence, err.Error())
		run.Status = "failed"
		_, _ = s.repo.SaveRollback(ctx, RollbackRecord{ID: uuid.NewString(), RunID: run.ID, CandidateID: candidate.ID, Status: "failed", Details: candidate.Evidence, CreatedAt: s.now().UTC()})
		return s.updateRunWithCandidate(ctx, run, candidate)
	}
	candidate.Status = "rolled_back"
	candidate.Evidence = append(candidate.Evidence, result.Evidence...)
	run.Status = "rolled_back"
	run.Recommendation = "Optimization candidate was rolled back."
	if _, err := s.repo.SaveRollback(ctx, RollbackRecord{ID: uuid.NewString(), RunID: run.ID, CandidateID: candidate.ID, Status: "rolled_back", Details: result.Evidence, CreatedAt: s.now().UTC()}); err != nil {
		return Run{}, err
	}
	return s.updateRunWithCandidate(ctx, run, candidate)
}

func (s *Service) latestBaseline(ctx context.Context) (snapshot.Snapshot, error) {
	if s.snapshots == nil {
		return snapshot.Snapshot{}, ErrBaselineRequired
	}
	snaps, err := s.snapshots.List(ctx)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	for _, snap := range snaps {
		if snap.Status == "baseline" {
			return snap, nil
		}
	}
	if len(snaps) > 0 {
		return snaps[0], nil
	}
	return snapshot.Snapshot{}, ErrBaselineRequired
}

func candidatesFromCapabilities(run Run, caps []adapters.Capability) []Candidate {
	var out []Candidate
	for _, cap := range caps {
		if !cap.Supported {
			continue
		}
		switch cap.Action {
		case "manage_dns_filtering":
			status := "not_run"
			if cap.RequiresAdmin || !cap.RollbackSupported {
				status = "manual_required"
			}
			out = append(out, Candidate{
				ID:                 candidateID(cap, "dns-filtering-stability"),
				RunID:              run.ID,
				Description:        "Evaluate DNS filtering stability against the stored baseline.",
				Status:             status,
				Evidence:           []string{fmt.Sprintf("%s supports %s: %s", cap.Adapter, cap.Action, cap.Reason)},
				ApprovalRequired:   true,
				RollbackSupported:  cap.RollbackSupported,
				BaselineSnapshotID: run.BaselineSnapshotID,
				CreatedAt:          run.CreatedAt,
				UpdatedAt:          run.UpdatedAt,
			})
		case "resolver_status":
			out = append(out, Candidate{
				ID:                 candidateID(cap, "resolver-health-check"),
				RunID:              run.ID,
				Description:        "Compare resolver health and DNS latency without persistent changes.",
				Status:             "not_run",
				Evidence:           []string{fmt.Sprintf("%s supports %s: %s", cap.Adapter, cap.Action, cap.Reason)},
				ApprovalRequired:   false,
				RollbackSupported:  true,
				BaselineSnapshotID: run.BaselineSnapshotID,
				CreatedAt:          run.CreatedAt,
				UpdatedAt:          run.UpdatedAt,
			})
		case "read_network_status":
			out = append(out, Candidate{
				ID:                 candidateID(cap, "read-only-baseline-compare"),
				RunID:              run.ID,
				Description:        "Run a read-only comparison snapshot and report reliability deltas.",
				Status:             "not_run",
				Evidence:           []string{fmt.Sprintf("%s supports %s: %s", cap.Adapter, cap.Action, cap.Reason)},
				ApprovalRequired:   false,
				RollbackSupported:  true,
				BaselineSnapshotID: run.BaselineSnapshotID,
				CreatedAt:          run.CreatedAt,
				UpdatedAt:          run.UpdatedAt,
			})
		}
	}
	return out
}

func manualCandidate(run Run, id, reason string) Candidate {
	return Candidate{ID: id, RunID: run.ID, Description: "Manual optimization review", Status: "manual_required", Evidence: []string{reason}, BaselineSnapshotID: run.BaselineSnapshotID, CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt}
}

func candidateID(cap adapters.Capability, suffix string) string {
	adapter := strings.NewReplacer(" ", "-", "_", "-", "/", "-").Replace(strings.ToLower(strings.TrimSpace(cap.Adapter)))
	if adapter == "" {
		adapter = "adapter"
	}
	return adapter + "-" + suffix
}

func (s *Service) loadRunCandidate(ctx context.Context, runID, candidateID string) (Run, Candidate, error) {
	run, err := s.repo.GetRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return Run{}, Candidate{}, err
	}
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" && len(run.Candidates) == 1 {
		return run, run.Candidates[0], nil
	}
	for _, c := range run.Candidates {
		if c.ID == candidateID {
			return run, c, nil
		}
	}
	return Run{}, Candidate{}, ErrCandidateNotFound
}

func (s *Service) updateRunWithCandidate(ctx context.Context, run Run, candidate Candidate) (Run, error) {
	candidate.UpdatedAt = s.now().UTC()
	if _, err := s.repo.UpdateCandidate(ctx, candidate); err != nil {
		return Run{}, err
	}
	run.UpdatedAt = s.now().UTC()
	if _, err := s.repo.UpdateRun(ctx, run); err != nil {
		return Run{}, err
	}
	return s.repo.GetRun(ctx, run.ID)
}

func (s *Service) scoreCandidate(ctx context.Context, candidate Candidate) float64 {
	score := 50.0
	if candidate.CandidateSnapshotID != "" && s.snapshots != nil {
		if snaps, err := s.snapshots.List(ctx); err == nil {
			for _, snap := range snaps {
				if snap.ID == candidate.CandidateSnapshotID {
					score += scoreSnapshot(snap)
					break
				}
			}
		}
	}
	if candidate.ApprovalRequired {
		score -= 5
	}
	if candidate.Status == "manual_required" {
		score = math.Min(score, 25)
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func scoreSnapshot(s snapshot.Snapshot) float64 {
	var delta float64
	for _, m := range s.Metrics {
		switch m.Status {
		case "healthy":
			delta += 8
		case "degraded":
			delta -= 5
		case "unavailable", "unsupported":
			delta -= 1
		default:
			delta -= 15
		}
	}
	return delta
}

func appliedCandidate(candidates []Candidate) (Candidate, bool) {
	for _, c := range candidates {
		if c.Status == "applied" || c.Status == "verified" {
			return c, true
		}
	}
	return Candidate{}, false
}

func normalizeProfile(profile string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return "balanced-reliability"
	}
	return profile
}
