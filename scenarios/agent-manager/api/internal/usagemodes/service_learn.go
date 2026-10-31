package usagemodes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/usageforecast"

	"github.com/vrooli/cli-core/usagemode"
)

// LearnRequest starts a Learn comparison: a role's new managed runs are split
// at random between the model it runs on today and one lower model.
type LearnRequest struct {
	Runner  string
	Role    string
	Profile string
	// Model overrides the challenger the resource's usage_tiers name.
	Model string
	Until string
	Label string
	// SplitPercent is the challenger share, 1 to 99; zero is 50.
	SplitPercent int
	// Sessions opts operator sessions in as randomized day blocks.
	Sessions  bool
	BlockDays int
	StartedBy string
}

// StartLearn creates a learn-kind mode. It refuses an exempt role, a runner
// with no usage tiers and a second comparison for the same runner and role:
// replacing one would silently discard its assignments. The challenger lives
// in the mode's Learn block, never in Runners, so a reader that does not know
// Learn leaves every launch alone.
func (s *Service) StartLearn(ctx context.Context, req LearnRequest) (usagemode.Mode, error) {
	if s.Tiers == nil {
		return usagemode.Mode{}, fmt.Errorf("usage mode tiers are not configured")
	}
	runner := usagemode.NormalizeRunner(req.Runner)
	if runner != "claude" && runner != "codex" {
		return usagemode.Mode{}, invalid("a comparison needs --runner claude or codex")
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		return usagemode.Mode{}, invalid("a comparison covers one role: pass --role")
	}
	if req.SplitPercent < 0 || req.SplitPercent > 99 {
		return usagemode.Mode{}, invalid("the challenger share must be between 1 and 99 percent")
	}
	if req.BlockDays < 0 || req.BlockDays > 14 {
		return usagemode.Mode{}, invalid("a session block is 1 to 14 days")
	}
	tiers, err := s.Tiers.Tiers(ctx, resourceRunner(runner))
	if err != nil {
		return usagemode.Mode{}, fmt.Errorf("resolve usage tiers: %w", err)
	}
	probe := usagemode.Mode{ExemptRoles: tiers.ExemptRoles}
	if probe.RoleExempt(role) {
		return usagemode.Mode{}, invalid("role %s is exempt from usage modes and is never randomized", role)
	}
	if model := strings.TrimSpace(req.Model); model != "" {
		// A challenger may never exceed the incumbent.
		if tiers, err = tiers.OverrideModel(model); err != nil {
			return usagemode.Mode{}, invalid("%s: %v", runner, err)
		}
	}
	now := s.now()
	untilOpts, err := s.parseUntil(ctx, req.Until, runner, now)
	if err != nil {
		return usagemode.Mode{}, err
	}
	untilOpts.Label, untilOpts.Kind, untilOpts.Source, untilOpts.StartedBy, untilOpts.Now = req.Label, usagemode.KindLearn, "agent-manager", req.StartedBy, now
	untilOpts.Scope = usagemode.Scope{Runner: runner, Role: role, Profile: strings.TrimSpace(req.Profile)}
	mode, err := usagemode.NewMode(tiers, untilOpts)
	if err != nil {
		return usagemode.Mode{}, invalid("%s: %v", runner, err)
	}
	split := req.SplitPercent
	if split == 0 {
		split = usagemode.DefaultSplitPercent
	}
	mode.Learn = &usagemode.Learn{SplitPercent: split, Runners: mode.Runners}
	mode.Runners = map[string]usagemode.RunnerSettings{}
	if req.Sessions {
		blocks := req.BlockDays
		if blocks == 0 {
			blocks = usagemode.DefaultBlockDays
		}
		mode.Learn.Sessions = &usagemode.SessionBlocks{OptIn: true, BlockDays: blocks, OptedInBy: req.StartedBy, OptedInAt: now}
	}
	if mode.Label == "" {
		mode.Label = fmt.Sprintf("Learn: %s on %s, %d%% on the lower model", role, runner, split)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var refused error
	err = usagemode.UpdateFile(s.path(), func(file *usagemode.File) error {
		file.Prune(now)
		for _, existing := range file.Modes {
			if existing.IsLearn() && existing.Scope.Runner == runner && existing.Scope.Role == role {
				return refuse(&refused, invalid("a comparison for %s on %s is already running (%s); end it first", role, runner, existing.ID))
			}
		}
		file.Modes = append(file.Modes, mode)
		return nil
	})
	if refused != nil {
		return usagemode.Mode{}, refused
	}
	if err != nil {
		return usagemode.Mode{}, fmt.Errorf("write usage mode file: %w", err)
	}
	return mode, nil
}

// PauseReason prefixes a pause the forecast made, so only the forecast resumes it.
const PauseReason = "forecast: "

// ApplyForecasts pauses every learn mode whose provider window is forecast to
// run out and resumes the ones the forecast paused once it no longer is. A
// pause stops new assignments only; work already assigned keeps its arm. It
// returns the ids it paused and resumed.
func (s *Service) ApplyForecasts(forecasts []usageforecast.Forecast, now time.Time) (paused, resumed []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = usagemode.UpdateFile(s.path(), func(file *usagemode.File) error {
		paused, resumed = nil, nil
		file.Prune(now)
		changed := false
		for i := range file.Modes {
			mode := &file.Modes[i]
			if !mode.IsLearn() || mode.Learn == nil {
				continue
			}
			provider := runnerProvider[usagemode.NormalizeRunner(mode.Scope.Runner)]
			running := false
			for _, forecast := range forecasts {
				if w := forecast.Window; w != nil && w.Provider == provider && forecast.Status == usageforecast.StatusWillExhaust {
					running = true
					break
				}
			}
			switch {
			case running && !mode.Learn.Paused:
				at := now
				mode.Learn.Paused, mode.Learn.PausedAt = true, &at
				mode.Learn.PausedReason = PauseReason + provider + " quota is forecast to run out before it resets; Save mode takes priority"
				paused, changed = append(paused, mode.ID), true
			case !running && mode.Learn.Paused && strings.HasPrefix(mode.Learn.PausedReason, PauseReason):
				mode.Learn.Paused, mode.Learn.PausedAt, mode.Learn.PausedReason = false, nil, ""
				resumed, changed = append(resumed, mode.ID), true
			}
		}
		if !changed {
			return errUnchanged
		}
		return nil
	})
	if errors.Is(err, errUnchanged) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("write usage mode file: %w", err)
	}
	return paused, resumed, nil
}

// RoleExempt reports a role that usage modes never touch for a runner: the
// default exempt roles and the ones the resource's usage_tiers add.
func (s *Service) RoleExempt(ctx context.Context, runner, role string) bool {
	probe := usagemode.Mode{}
	if s.Tiers != nil {
		if tiers, err := s.Tiers.Tiers(ctx, resourceRunner(usagemode.NormalizeRunner(runner))); err == nil {
			probe.ExemptRoles = tiers.ExemptRoles
		}
	}
	return probe.RoleExempt(strings.TrimSpace(role))
}
