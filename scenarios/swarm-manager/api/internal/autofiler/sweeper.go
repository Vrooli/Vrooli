package autofiler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/settings"
)

type SettingsProvider interface {
	LoadAutoFiler() (settings.AutoFilerSettings, error)
}

type FindingSource interface {
	Findings(ctx context.Context, target Target) ([]Finding, error)
}

type BacklogReconciler interface {
	ArchiveItem(ctx context.Context, kind backlog.BacklogKind, name, reason string) (backlog.BacklogItem, error)
	AnnotateItem(ctx context.Context, kind backlog.BacklogKind, name, note string) (backlog.BacklogItem, error)
}

type SweepResult struct {
	Enabled          bool
	Strategy         Strategy
	Mode             Mode
	Candidates       int
	Findings         int
	Created          int
	ReconciledClosed int
	ReconciledNoted  int
	SkippedDismissed int
	OpenAutoFiled    int
	RemainingBudget  int
	Brake            BrakeState
	LastError        string
	RanAt            time.Time
}

type Sweeper struct {
	Settings   SettingsProvider
	Backlog    BacklogReader
	Counter    TransitionCounter
	Filer      *Filer
	Reconciler BacklogReconciler
	Dismissals *DismissalStore
	Feature    TargetingStrategy
	Importance TargetingStrategy
	Source     FindingSource
	Interval   time.Duration
	Limit      int

	wake chan struct{}
	mu   sync.Mutex
	last SweepResult
}

func NewSweeper(settings SettingsProvider, backlog BacklogReader, counter TransitionCounter, filer *Filer, source FindingSource) *Sweeper {
	return &Sweeper{
		Settings: settings,
		Backlog:  backlog,
		Counter:  counter,
		Filer:    filer,
		Source:   source,
		wake:     make(chan struct{}, 1),
	}
}

func (s *Sweeper) WakeAutoFiler() {
	if s == nil {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Sweeper) LastResult() SweepResult {
	if s == nil {
		return SweepResult{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

func (s *Sweeper) RunOnce(ctx context.Context) (SweepResult, error) {
	result := SweepResult{RanAt: time.Now().UTC()}
	if s == nil || s.Settings == nil {
		result.LastError = "auto-filer sweeper is not configured"
		return result, nil
	}
	cfg, err := s.Settings.LoadAutoFiler()
	if err != nil {
		result.LastError = err.Error()
		s.remember(result)
		return result, fmt.Errorf("load auto-filer settings: %w", err)
	}
	result.Enabled = cfg.Enabled
	result.Strategy = Strategy(cfg.Strategy)
	result.Mode = Mode(cfg.Mode)
	if !cfg.Enabled {
		s.remember(result)
		return result, nil
	}
	if s.Backlog == nil || s.Filer == nil || s.Source == nil {
		result.LastError = "auto-filer dependencies are incomplete"
		s.remember(result)
		return result, nil
	}

	targets, err := s.strategy(result.Strategy).Candidates(ctx, s.targetLimit())
	if err != nil {
		result.LastError = err.Error()
		s.remember(result)
		return result, fmt.Errorf("select auto-filer targets: %w", err)
	}
	result.Candidates = len(targets)
	activeFindings := map[string]struct{}{}
	sourceComplete := true
	var fileable []Finding
	for _, target := range targets {
		findings, err := s.Source.Findings(ctx, target)
		if err != nil {
			sourceComplete = false
			slog.Warn("autofiler: finding source failed", "scenario", target.Scenario, "err", err)
			continue
		}
		result.Findings += len(findings)
		for _, finding := range findings {
			activeFindings[finding.StableID()] = struct{}{}
			if s.dismissed(finding.StableID()) {
				result.SkippedDismissed++
				continue
			}
			fileable = append(fileable, finding)
		}
	}
	if sourceComplete {
		closed, noted := s.reconcileResolved(ctx, activeFindings)
		result.ReconciledClosed = closed
		result.ReconciledNoted = noted
	}
	brake, err := VelocityBrake(ctx, s.Counter, cfg, result.RanAt)
	if err != nil {
		result.LastError = err.Error()
		s.remember(result)
		return result, fmt.Errorf("compute velocity brake: %w", err)
	}
	result.Brake = brake
	budget, err := RemainingBudget(s.Backlog, cfg.MaxOpenAutoFiled)
	if err != nil {
		result.LastError = err.Error()
		s.remember(result)
		return result, fmt.Errorf("compute auto-filer budget: %w", err)
	}
	result.RemainingBudget = budget
	result.OpenAutoFiled = cfg.MaxOpenAutoFiled - budget
	if brake.Braked || budget <= 0 {
		s.remember(result)
		return result, nil
	}

	for _, finding := range fileable {
		if budget <= 0 {
			result.RemainingBudget = 0
			s.remember(result)
			return result, nil
		}
		filed, err := s.Filer.File(ctx, finding, FileOptions{
			Mode:     result.Mode,
			Strategy: result.Strategy,
			GoalName: cfg.GoalName,
		})
		if err != nil {
			slog.Warn("autofiler: file finding failed", "finding", finding.StableID(), "err", err)
			continue
		}
		if filed.Created {
			result.Created++
			budget--
			result.RemainingBudget = budget
		}
	}
	s.remember(result)
	return result, nil
}

func (s *Sweeper) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.runWithRecover(ctx)
	interval := s.interval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runWithRecover(ctx)
		case <-s.wake:
			s.runWithRecover(ctx)
		}
	}
}

func (s *Sweeper) runWithRecover(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("autofiler: sweeper panic recovered", "panic", rec)
		}
	}()
	result, err := s.RunOnce(ctx)
	if err != nil {
		slog.Warn("autofiler: run failed", "err", err)
		return
	}
	slog.Info("autofiler: cycle complete",
		"enabled", result.Enabled,
		"strategy", result.Strategy,
		"mode", result.Mode,
		"candidates", result.Candidates,
		"findings", result.Findings,
		"created", result.Created,
		"braked", result.Brake.Braked,
		"remaining_budget", result.RemainingBudget)
}

func (s *Sweeper) strategy(strategy Strategy) TargetingStrategy {
	switch strategy {
	case StrategyImportance:
		if s.Importance != nil {
			return s.Importance
		}
		return ImportanceStrategy{}
	default:
		if s.Feature != nil {
			return s.Feature
		}
		return FeaturePendingStrategy{BacklogReader: s.Backlog, SelfScenarioName: "swarm-manager"}
	}
}

func (s *Sweeper) dismissed(findingID string) bool {
	if s.Dismissals == nil {
		return false
	}
	dismissed, err := s.Dismissals.IsDismissed(findingID)
	if err != nil {
		slog.Warn("autofiler: dismissal lookup failed", "finding", findingID, "err", err)
		return false
	}
	return dismissed
}

func (s *Sweeper) reconcileResolved(ctx context.Context, activeFindings map[string]struct{}) (int, int) {
	if s.Reconciler == nil || s.Backlog == nil {
		return 0, 0
	}
	items, err := s.Backlog.LoadAll(nil)
	if err != nil {
		slog.Warn("autofiler: reconcile load failed", "err", err)
		return 0, 0
	}
	closed := 0
	noted := 0
	for _, item := range OpenAutoFiled(items) {
		findingID := strings.TrimSpace(item.FindingRef)
		if findingID == "" {
			continue
		}
		if _, active := activeFindings[findingID]; active {
			continue
		}
		reason := "Auto-filer finding no longer appears in the configured source; marking this suggestion resolved."
		if item.Status == backlog.StatusSuggested {
			if _, err := s.Reconciler.ArchiveItem(ctx, item.Kind, item.Name, reason); err != nil {
				slog.Warn("autofiler: archive resolved suggestion failed", "item", ItemRef(item), "err", err)
				continue
			}
			closed++
			continue
		}
		note := "Auto-filer finding no longer appears in the configured source; item was already accepted, so it remains in the normal backlog flow."
		if _, err := s.Reconciler.AnnotateItem(ctx, item.Kind, item.Name, note); err != nil {
			slog.Warn("autofiler: annotate resolved accepted item failed", "item", ItemRef(item), "err", err)
			continue
		}
		noted++
	}
	return closed, noted
}

func (s *Sweeper) interval() time.Duration {
	if s.Interval > 0 {
		return s.Interval
	}
	if s.Settings != nil {
		cfg, err := s.Settings.LoadAutoFiler()
		if err == nil && cfg.IntervalMinutes > 0 {
			return time.Duration(cfg.IntervalMinutes) * time.Minute
		}
	}
	return time.Duration(settings.DefaultSettings().AutoFiler.IntervalMinutes) * time.Minute
}

func (s *Sweeper) targetLimit() int {
	if s.Limit > 0 {
		return s.Limit
	}
	return 500
}

func (s *Sweeper) remember(result SweepResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = result
}
