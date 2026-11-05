// Package usagemodes is Agent Manager's operator surface over the shared usage
// mode file (docs/configuration/usage-modes.md): it starts, lists and ends
// modes and ends a window-reset mode early when its window rolled. The file
// format, tier validation and per-launch resolution live in
// github.com/vrooli/cli-core/usagemode; this package adds only what needs the
// API: resource tier sources and the provider quota readings.
package usagemodes

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/usageforecast"

	"github.com/vrooli/cli-core/usagemode"
)

// Presets a start request may name.
const PresetOneTierDown = "one-tier-down"

// TierSource returns a resource's usage tiers for a runner ("claude-code" or
// "codex"). Agent Manager never reads model catalogs itself; production wires
// the resource-owned policy path resolution.
type TierSource interface {
	Tiers(ctx context.Context, runner string) (usagemode.Tiers, error)
}

// windowTarget maps an --until reset keyword to the provider window it follows.
type windowTarget struct {
	provider string
	minutes  int
	label    string
}

// providers by normalized runner.
var runnerProvider = map[string]string{"claude": "anthropic", "codex": "openai"}

// StartRequest is one operator start.
type StartRequest struct {
	Preset             string
	Model              string
	Effort             string
	ContextLimitTokens int
	Runner             string
	Role               string
	Profile            string
	Until              string
	Label              string
	StartedBy          string
}

// StartResult is the started mode and what it replaced.
type StartResult struct {
	Mode        usagemode.Mode
	ReplacedIDs []string
}

// ErrInvalid marks a request the operator can correct.
var ErrInvalid = errors.New("invalid usage mode request")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Service reads and writes the mode file. Path overrides the default location
// (tests); empty uses usagemode.Path.
type Service struct {
	Tiers    TierSource
	Readings pricing.QuotaReadingSource
	Now      func() time.Time
	Path     string
	mu       sync.Mutex
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) path() string {
	if s.Path != "" {
		return s.Path
	}
	return usagemode.Path()
}

// File is the mode file's path, for operator output.
func (s *Service) File() string { return s.path() }

// List returns the unexpired modes, oldest first.
func (s *Service) List() []usagemode.Mode { return s.listAt(s.now()) }

func (s *Service) listAt(now time.Time) []usagemode.Mode {
	modes := usagemode.ReadFile(s.path(), now)
	sort.SliceStable(modes, func(i, j int) bool { return modes[i].StartedAt.Before(modes[j].StartedAt) })
	return modes
}

func sameScope(a, b usagemode.Scope) bool {
	return usagemode.NormalizeRunner(a.Runner) == usagemode.NormalizeRunner(b.Runner) && a.Role == b.Role && a.Profile == b.Profile
}

func resourceRunner(runner string) string {
	if usagemode.NormalizeRunner(runner) == "claude" {
		return "claude-code"
	}
	return usagemode.NormalizeRunner(runner)
}

// Start builds a mode from the resources' tiers and replaces any active mode
// with an identical scope.
func (s *Service) Start(ctx context.Context, req StartRequest) (StartResult, error) {
	if s.Tiers == nil {
		return StartResult{}, errors.New("usage mode tiers are not configured")
	}
	runner := usagemode.NormalizeRunner(req.Runner)
	switch runner {
	case "", "claude", "codex":
	default:
		return StartResult{}, invalid("runner %q has no usage tiers; use claude or codex", req.Runner)
	}
	if req.Preset != "" && req.Preset != PresetOneTierDown {
		return StartResult{}, invalid("unknown preset %q; the only preset is %s", req.Preset, PresetOneTierDown)
	}
	if req.Preset == "" && strings.TrimSpace(req.Model) == "" {
		return StartResult{}, invalid("choose --preset %s or --model", PresetOneTierDown)
	}
	if strings.TrimSpace(req.Model) != "" && runner == "" {
		return StartResult{}, invalid("--model names one runner's model; pass --runner")
	}
	if req.ContextLimitTokens < 0 {
		return StartResult{}, invalid("context limit must not be negative")
	}
	now := s.now()
	untilOpts, err := s.parseUntil(ctx, req.Until, runner, now)
	if err != nil {
		return StartResult{}, err
	}
	runners := []string{runner}
	if runner == "" {
		runners = []string{"claude", "codex"}
	}
	var merged usagemode.Mode
	for i, name := range runners {
		tiers, err := s.Tiers.Tiers(ctx, resourceRunner(name))
		if err != nil {
			if runner == "" {
				return StartResult{}, fmt.Errorf("resolve %s usage tiers: %w", name, err)
			}
			return StartResult{}, fmt.Errorf("resolve usage tiers: %w", err)
		}
		if model := strings.TrimSpace(req.Model); model != "" {
			if tiers, err = tiers.OverrideModel(model); err != nil {
				return StartResult{}, invalid("%s: %v", name, err)
			}
		}
		opts := untilOpts
		opts.Label, opts.Kind, opts.Source, opts.StartedBy, opts.Now = req.Label, "save", "agent-manager", req.StartedBy, now
		opts.Scope = usagemode.Scope{Runner: runner, Role: strings.TrimSpace(req.Role), Profile: strings.TrimSpace(req.Profile)}
		opts.Effort, opts.ContextLimitTokens = req.Effort, req.ContextLimitTokens
		mode, err := usagemode.NewMode(tiers, opts)
		if err != nil {
			return StartResult{}, invalid("%s: %v", name, err)
		}
		if i == 0 {
			merged = mode
			continue
		}
		for key, settings := range mode.Runners {
			merged.Runners[key] = settings
		}
		merged.ExemptRoles = unionStrings(merged.ExemptRoles, mode.ExemptRoles)
	}
	if merged.Label == "" {
		merged.Label = describe(merged)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var replaced []string
	err = usagemode.UpdateFile(s.path(), func(file *usagemode.File) error {
		file.Prune(now)
		kept := []usagemode.Mode{}
		for _, existing := range file.Modes {
			if !existing.IsLearn() && sameScope(existing.Scope, merged.Scope) { // a save start never discards a comparison
				replaced = append(replaced, existing.ID)
				continue
			}
			kept = append(kept, existing)
		}
		file.Modes = append(kept, merged)
		return nil
	})
	if err != nil {
		return StartResult{}, fmt.Errorf("write usage mode file: %w", err)
	}
	return StartResult{Mode: merged, ReplacedIDs: replaced}, nil
}

func describe(m usagemode.Mode) string {
	scope := "all runners"
	if m.Scope.Runner != "" {
		scope = m.Scope.Runner
	}
	if m.Scope.Role != "" {
		scope += " role " + m.Scope.Role
	}
	if m.Scope.Profile != "" {
		scope += " profile " + m.Scope.Profile
	}
	return "Lower tier for " + scope
}

func unionStrings(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range append(append([]string(nil), a...), b...) {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

// parseUntil turns an --until value into the mode's expiry options.
func (s *Service) parseUntil(ctx context.Context, raw, runner string, now time.Time) (usagemode.NewModeOptions, error) {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "":
		return usagemode.NewModeOptions{}, invalid("--until is required: 5h-reset, weekly-reset, off, an RFC 3339 time or a duration such as 72h")
	case "off":
		return usagemode.NewModeOptions{}, nil
	case "5h-reset", "weekly-reset":
		return s.windowReset(ctx, raw, runner, now)
	}
	if at, err := time.Parse(time.RFC3339, raw); err == nil {
		if !at.After(now) {
			return usagemode.NewModeOptions{}, invalid("--until %s is not in the future", raw)
		}
		return usagemode.NewModeOptions{ExpiresAt: &at, UntilKind: usagemode.UntilTime}, nil
	}
	if days, ok := strings.CutSuffix(raw, "d"); ok { // 90d: a comparison runs for weeks
		if n, err := strconv.Atoi(days); err == nil && n > 0 {
			return usagemode.NewModeOptions{Duration: time.Duration(n) * 24 * time.Hour}, nil
		}
	}
	if d, err := time.ParseDuration(raw); err == nil {
		if d <= 0 {
			return usagemode.NewModeOptions{}, invalid("--until duration must be positive")
		}
		return usagemode.NewModeOptions{Duration: d}, nil
	}
	return usagemode.NewModeOptions{}, invalid("--until %q is not 5h-reset, weekly-reset, off, an RFC 3339 time or a duration such as 72h", raw)
}

func windowFor(keyword, runner string) (windowTarget, error) {
	minutes, label := 300, "5-hour"
	if keyword == "weekly-reset" {
		minutes, label = 10080, "weekly"
	}
	if runner == "" {
		return windowTarget{}, invalid("--until %s follows one provider's window; pass --runner claude or --runner codex", keyword)
	}
	if runner == "codex" && minutes == 300 {
		return windowTarget{}, invalid("codex reports only a weekly window; use --until weekly-reset (or a time or duration)")
	}
	return windowTarget{provider: runnerProvider[runner], minutes: minutes, label: label}, nil
}

// currentWindow finds the provider's current window of the given length from
// the newest quota readings.
func (s *Service) currentWindow(ctx context.Context, target windowTarget, now time.Time) (*usageforecast.Window, error) {
	if s.Readings == nil {
		return nil, errors.New("provider quota readings are unavailable")
	}
	spans, err := s.Readings.QuotaReadingSpans(ctx, target.provider, now.Add(-usageforecast.CurrentLookback))
	if err != nil {
		return nil, fmt.Errorf("read quota readings: %w", err)
	}
	var best *usageforecast.Window
	for _, forecast := range usageforecast.Current(spans, now) {
		w := forecast.Window
		if w == nil || w.Provider != target.provider || w.WindowMinutes != int64(target.minutes) || w.ResetAt.IsZero() || !w.ResetAt.After(now) {
			continue
		}
		if best == nil || w.LastAt.After(best.LastAt) {
			best = w
		}
	}
	return best, nil
}

func (s *Service) windowReset(ctx context.Context, keyword, runner string, now time.Time) (usagemode.NewModeOptions, error) {
	target, err := windowFor(keyword, runner)
	if err != nil {
		return usagemode.NewModeOptions{}, err
	}
	w, err := s.currentWindow(ctx, target, now)
	if err != nil {
		return usagemode.NewModeOptions{}, err
	}
	if w == nil {
		return usagemode.NewModeOptions{}, invalid("no %s %s quota reading exists, so its reset time is unknown; use a time or duration instead", target.provider, target.label)
	}
	reset := w.ResetAt.UTC()
	return usagemode.NewModeOptions{ExpiresAt: &reset, UntilKind: usagemode.UntilWindowReset, Provider: target.provider, WindowMinutes: target.minutes}, nil
}

// End removes the named modes, or every mode. It reports the ids it ended and
// the requested ids that were not active.
func (s *Service) End(ids []string, all bool) (ended, missing []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !all && len(ids) == 0 {
		return nil, nil, invalid("name a mode id or pass all")
	}
	now := s.now()
	err = usagemode.UpdateFile(s.path(), func(file *usagemode.File) error {
		ended, missing = nil, nil
		file.Prune(now)
		want := map[string]bool{}
		for _, id := range ids {
			want[strings.TrimSpace(id)] = true
		}
		kept := []usagemode.Mode{}
		for _, mode := range file.Modes {
			if all || want[mode.ID] {
				ended = append(ended, mode.ID)
				delete(want, mode.ID)
				continue
			}
			kept = append(kept, mode)
		}
		for id := range want {
			missing = append(missing, id)
		}
		sort.Strings(missing)
		if len(ended) == 0 {
			return errUnchanged
		}
		file.Modes = kept
		return nil
	})
	if errors.Is(err, errUnchanged) {
		return nil, missing, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("write usage mode file: %w", err)
	}
	return ended, missing, nil
}

// errUnchanged aborts an update that found nothing to change, so the file is
// not rewritten.
var errUnchanged = errors.New("usage mode file unchanged")

// rollTolerance is how far a provider's jittered reset may differ from the
// stored one before the window counts as rolled.
const rollTolerance = 10 * time.Minute

// EndRolledWindows ends every window_reset mode whose window rolled before its
// stored expiry: a newer reading shows a window of the same provider and length
// with a later reset while no reading still matches the stored reset. It is
// idempotent (an ended mode is gone) and returns the ids it ended.
func (s *Service) EndRolledWindows(forecasts []usageforecast.Forecast, now time.Time) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ended []string
	err := usagemode.UpdateFile(s.path(), func(file *usagemode.File) error {
		ended = nil
		file.Prune(now)
		kept := []usagemode.Mode{}
		for _, mode := range file.Modes {
			if windowRolled(mode, forecasts, now) {
				ended = append(ended, mode.ID)
				continue
			}
			kept = append(kept, mode)
		}
		if len(ended) == 0 {
			return errUnchanged
		}
		file.Modes = kept
		return nil
	})
	if errors.Is(err, errUnchanged) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("write usage mode file: %w", err)
	}
	return ended, nil
}

func windowRolled(mode usagemode.Mode, forecasts []usageforecast.Forecast, now time.Time) bool {
	if mode.Until.Kind != usagemode.UntilWindowReset || mode.ExpiresAt == nil || !now.Before(*mode.ExpiresAt) {
		return false
	}
	rolled := false
	for _, forecast := range forecasts {
		w := forecast.Window
		if w == nil || w.Provider != mode.Until.Provider || w.WindowMinutes != int64(mode.Until.WindowMinutes) || w.ResetAt.IsZero() {
			continue
		}
		if !w.LastAt.After(mode.StartedAt) {
			continue
		}
		delta := w.ResetAt.Sub(*mode.ExpiresAt)
		if delta < 0 {
			delta = -delta
		}
		if delta <= rollTolerance {
			return false // a reading still follows the stored reset
		}
		if w.ResetAt.After(*mode.ExpiresAt) {
			rolled = true
		}
	}
	return rolled
}

// Extension targets for Extend.
const (
	ExtendWeeklyReset = "weekly-reset"
	ExtendTwoHours    = "2h"
	ExtendOff         = "off"
)

// Extend moves one active mode's end and keeps everything else (id, start,
// starter, settings), so its burn since start stays continuous. to is
// weekly-reset (the runner's weekly window), a duration added to the later of
// now and the current end, or off.
func (s *Service) Extend(ctx context.Context, id, to string) (usagemode.Mode, error) {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out usagemode.Mode
	var refused error // the operator's mistake, reported as is rather than as a write failure
	err := usagemode.UpdateFile(s.path(), func(file *usagemode.File) error {
		file.Prune(now)
		idx := -1
		for i, mode := range file.Modes {
			if mode.ID == strings.TrimSpace(id) {
				idx = i
			}
		}
		if idx < 0 {
			return refuse(&refused, invalid("usage mode %q is not active", id))
		}
		mode := file.Modes[idx]
		switch to = strings.TrimSpace(to); to {
		case ExtendOff:
			mode.Until, mode.ExpiresAt = usagemode.Until{Kind: usagemode.UntilOff}, nil
		case ExtendWeeklyReset:
			opts, err := s.windowReset(ctx, "weekly-reset", usagemode.NormalizeRunner(mode.Scope.Runner), now)
			if err != nil {
				return refuse(&refused, err)
			}
			if mode.ExpiresAt != nil && !opts.ExpiresAt.After(*mode.ExpiresAt) {
				return refuse(&refused, invalid("the weekly reset is not later than the mode's current end"))
			}
			mode.Until = usagemode.Until{Kind: usagemode.UntilWindowReset, Provider: opts.Provider, WindowMinutes: opts.WindowMinutes}
			mode.ExpiresAt = opts.ExpiresAt
		default:
			d, err := time.ParseDuration(to)
			if err != nil || d <= 0 {
				return refuse(&refused, invalid("extend to %q is not weekly-reset, off or a positive duration", to))
			}
			base := now
			if mode.ExpiresAt != nil && mode.ExpiresAt.After(base) {
				base = *mode.ExpiresAt
			}
			end := base.Add(d)
			mode.Until, mode.ExpiresAt = usagemode.Until{Kind: usagemode.UntilTime}, &end
		}
		file.Modes[idx] = mode
		out = mode
		return nil
	})
	if refused != nil {
		return usagemode.Mode{}, refused
	}
	if err != nil {
		return usagemode.Mode{}, fmt.Errorf("write usage mode file: %w", err)
	}
	return out, nil
}

// refuse records a callback failure that is not a file problem and returns it
// so the update aborts without writing.
func refuse(slot *error, err error) error {
	*slot = err
	return err
}

// WeeklyResetAfter is the runner's weekly reset when it is later than after,
// for offering "extend to weekly reset" only when it can be honoured.
func (s *Service) WeeklyResetAfter(ctx context.Context, runner string, after time.Time) (time.Time, bool) {
	opts, err := s.windowReset(ctx, "weekly-reset", usagemode.NormalizeRunner(runner), s.now())
	if err != nil || opts.ExpiresAt == nil || !opts.ExpiresAt.After(after) {
		return time.Time{}, false
	}
	return *opts.ExpiresAt, true
}
