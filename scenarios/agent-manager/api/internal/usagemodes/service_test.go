package usagemodes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/usageforecast"

	"github.com/vrooli/cli-core/usagemode"
)

const codexPolicy = `{"runner":"codex","model_aliases":{"gpt-6.1-terra":{"canonical_model":"openai/gpt-6.1-terra"},"gpt-6-luna":{"canonical_model":"openai/gpt-6-luna"}},
"usage_tiers":{"model_map":{"gpt-6.1-terra":"gpt-6-luna"},"session_model":"gpt-6-luna","exempt_roles":["code.fallback"]}}`

const claudePolicy = `{"runner":"claude-code","model_aliases":{"opus":{"canonical_model":"anthropic/claude-opus-5"},"sonnet":{"canonical_model":"anthropic/claude-sonnet-5"}},
"usage_tiers":{"model_map":{"opus":"sonnet"},"session_model":"sonnet"}}`

type fakeTiers map[string]string

func (f fakeTiers) Tiers(_ context.Context, runner string) (usagemode.Tiers, error) {
	raw, ok := f[runner]
	if !ok {
		return usagemode.Tiers{}, errors.New("resource unavailable")
	}
	return usagemode.LoadTiers([]byte(raw))
}

type fakeReadings []pricing.QuotaReadingSpan

func (f fakeReadings) QuotaReadingSpans(context.Context, string, time.Time) ([]pricing.QuotaReadingSpan, error) {
	return f, nil
}

var t0 = time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)

func span(provider string, minutes int64, reset, first, last time.Time, percent float64) pricing.QuotaReadingSpan {
	r := reset.Truncate(time.Minute)
	return pricing.QuotaReadingSpan{Provider: provider, Pool: "primary", Window: "w", WindowMinutes: &minutes, ResetAt: &r, UsedPercent: &percent, Standing: pricing.QuotaStandingAvailable, FirstAt: first, LastAt: last, Samples: 1}
}

func newService(t *testing.T, readings ...pricing.QuotaReadingSpan) *Service {
	t.Helper()
	return &Service{
		Tiers:    fakeTiers{"codex": codexPolicy, "claude-code": claudePolicy},
		Readings: fakeReadings(readings),
		Now:      func() time.Time { return t0 },
		Path:     filepath.Join(t.TempDir(), "mode.json"),
	}
}

func TestStartRecordsOperatorAndLowerTier(t *testing.T) {
	s := newService(t)
	result, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "72h", StartedBy: "operator@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	m := result.Mode
	if m.StartedBy != "operator@example.com" || m.Source != "agent-manager" || m.Until.Kind != usagemode.UntilTime {
		t.Fatalf("mode = %+v", m)
	}
	if m.ExpiresAt == nil || !m.ExpiresAt.Equal(t0.Add(72*time.Hour)) {
		t.Fatalf("expiry = %v", m.ExpiresAt)
	}
	settings, ok := m.SettingsFor("codex")
	if !ok || settings.ModelMap["gpt-6.1-terra"] != "gpt-6-luna" {
		t.Fatalf("settings = %+v", settings)
	}
	if listed := s.List(); len(listed) != 1 || listed[0].ID != m.ID {
		t.Fatalf("list = %+v", listed)
	}
}

func TestStartGlobalCoversBothRunnersAndNeedsRunnerForModel(t *testing.T) {
	s := newService(t)
	result, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Until: "off"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mode.Runners) != 2 || result.Mode.Until.Kind != usagemode.UntilOff || result.Mode.ExpiresAt != nil {
		t.Fatalf("global off mode = %+v", result.Mode)
	}
	_, err = s.Start(context.Background(), StartRequest{Model: "gpt-6-luna", Until: "1h"})
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "--runner") {
		t.Fatalf("model without runner err = %v", err)
	}
}

func TestStartReplacesIdenticalScopeOnly(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	first, _ := s.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h"})
	other, _ := s.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "claude", Until: "1h"})
	second, err := s.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "2h"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.ReplacedIDs) != 1 || second.ReplacedIDs[0] != first.Mode.ID {
		t.Fatalf("replaced = %v, want %s", second.ReplacedIDs, first.Mode.ID)
	}
	ids := map[string]bool{}
	for _, m := range s.List() {
		ids[m.ID] = true
	}
	if len(ids) != 2 || !ids[second.Mode.ID] || !ids[other.Mode.ID] {
		t.Fatalf("active = %v", ids)
	}
}

func TestStartRejectsBadRequests(t *testing.T) {
	s := newService(t)
	for name, req := range map[string]StartRequest{
		"no until":       {Preset: PresetOneTierDown, Runner: "codex"},
		"no preset":      {Runner: "codex", Until: "1h"},
		"bad preset":     {Preset: "two-down", Runner: "codex", Until: "1h"},
		"bad runner":     {Preset: PresetOneTierDown, Runner: "grok", Until: "1h"},
		"bad until":      {Preset: PresetOneTierDown, Runner: "codex", Until: "soon"},
		"past time":      {Preset: PresetOneTierDown, Runner: "codex", Until: "2026-10-30T17:00:00Z"},
		"unknown target": {Model: "no-such-model", Runner: "codex", Until: "1h"},
	} {
		if _, err := s.Start(context.Background(), req); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want invalid request", name, err)
		}
	}
	if len(s.List()) != 0 {
		t.Fatalf("rejected starts wrote modes: %+v", s.List())
	}
}

func TestStartUntilRFC3339(t *testing.T) {
	s := newService(t)
	when := t0.Add(5 * time.Hour)
	result, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: when.Format(time.RFC3339)})
	if err != nil || result.Mode.ExpiresAt == nil || !result.Mode.ExpiresAt.Equal(when) {
		t.Fatalf("result = %+v, err = %v", result.Mode, err)
	}
}

func TestWindowResetUsesLatestReading(t *testing.T) {
	reset := t0.Add(51 * time.Hour)
	s := newService(t, span("openai", 10080, reset, t0.Add(-time.Hour), t0.Add(-time.Minute), 53))
	result, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "weekly-reset"})
	if err != nil {
		t.Fatal(err)
	}
	m := result.Mode
	if m.Until.Kind != usagemode.UntilWindowReset || m.Until.Provider != "openai" || m.Until.WindowMinutes != 10080 {
		t.Fatalf("until = %+v", m.Until)
	}
	if m.ExpiresAt == nil || !m.ExpiresAt.Equal(reset.Truncate(time.Minute)) {
		t.Fatalf("expiry = %v, want the observed reset %v", m.ExpiresAt, reset)
	}
}

func TestWindowResetRefusals(t *testing.T) {
	ctx := context.Background()
	none := newService(t)
	_, err := none.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "claude", Until: "5h-reset"})
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "no anthropic 5-hour quota reading") {
		t.Fatalf("no reading err = %v", err)
	}
	_, err = none.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "5h-reset"})
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "codex reports only a weekly window") {
		t.Fatalf("codex 5h err = %v", err)
	}
	_, err = none.Start(ctx, StartRequest{Preset: PresetOneTierDown, Until: "weekly-reset"})
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "--runner") {
		t.Fatalf("global reset err = %v", err)
	}
	if len(none.List()) != 0 {
		t.Fatal("a refused reset start wrote a mode")
	}
}

func TestClaudeFiveHourReset(t *testing.T) {
	reset := t0.Add(3 * time.Hour)
	s := newService(t, span("anthropic", 300, reset, t0.Add(-time.Hour), t0.Add(-time.Minute), 12))
	result, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "claude", Until: "5h-reset"})
	if err != nil || result.Mode.Until.WindowMinutes != 300 {
		t.Fatalf("result = %+v, err = %v", result.Mode, err)
	}
}

func TestEndByIDAndAll(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	a, _ := s.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h"})
	b, _ := s.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "claude", Until: "1h"})
	ended, missing, err := s.End([]string{a.Mode.ID, "um-none"}, false)
	if err != nil || len(ended) != 1 || ended[0] != a.Mode.ID || len(missing) != 1 || missing[0] != "um-none" {
		t.Fatalf("ended=%v missing=%v err=%v", ended, missing, err)
	}
	if listed := s.List(); len(listed) != 1 || listed[0].ID != b.Mode.ID {
		t.Fatalf("after end by id: %+v", listed)
	}
	ended, _, _ = s.End(nil, true)
	if len(ended) != 1 || len(s.List()) != 0 {
		t.Fatalf("end all: ended=%v list=%+v", ended, s.List())
	}
	if _, _, err := s.End(nil, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty end err = %v", err)
	}
}

func TestExpiredModesAreInactive(t *testing.T) {
	s := newService(t)
	if _, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h"}); err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return t0.Add(2 * time.Hour) }
	if len(s.List()) != 0 {
		t.Fatal("an expired mode is still listed")
	}
}

func forecastFor(provider string, minutes int64, reset, last time.Time) []usageforecast.Forecast {
	spans := []pricing.QuotaReadingSpan{span(provider, minutes, reset, last.Add(-time.Minute), last, 5)}
	return usageforecast.Current(spans, last)
}

func TestEndRolledWindowsEndsOnlyAfterTheWindowRolls(t *testing.T) {
	reset := t0.Add(4 * time.Hour)
	s := newService(t, span("anthropic", 300, reset, t0.Add(-time.Hour), t0.Add(-time.Minute), 12))
	started, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "claude", Until: "5h-reset"})
	if err != nil {
		t.Fatal(err)
	}
	timed, _ := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "6h"})

	// The same reset (provider jitter of a few seconds) keeps the mode.
	later := t0.Add(30 * time.Minute)
	same := forecastFor("anthropic", 300, reset.Add(20*time.Second), later)
	if ended, err := s.EndRolledWindows(same, later); err != nil || len(ended) != 0 {
		t.Fatalf("unrolled window ended %v (err %v)", ended, err)
	}

	// A reading whose window resets later than the stored time shows it rolled early.
	rolled := forecastFor("anthropic", 300, later.Add(5*time.Hour), later)
	ended, err := s.EndRolledWindows(rolled, later)
	if err != nil || len(ended) != 1 || ended[0] != started.Mode.ID {
		t.Fatalf("ended = %v (err %v), want %s", ended, err, started.Mode.ID)
	}
	if again, _ := s.EndRolledWindows(rolled, later); len(again) != 0 {
		t.Fatalf("second pass ended %v; early end must be idempotent", again)
	}
	if listed := s.List(); len(listed) != 1 || listed[0].ID != timed.Mode.ID {
		t.Fatalf("a time mode or other window must stay: %+v", listed)
	}
}

func TestEndRolledWindowsIgnoresOtherProvidersAndStaleReadings(t *testing.T) {
	reset := t0.Add(4 * time.Hour)
	s := newService(t, span("anthropic", 300, reset, t0.Add(-time.Hour), t0.Add(-time.Minute), 12))
	if _, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "claude", Until: "5h-reset"}); err != nil {
		t.Fatal(err)
	}
	later := t0.Add(time.Hour)
	if ended, _ := s.EndRolledWindows(forecastFor("openai", 300, later.Add(5*time.Hour), later), later); len(ended) != 0 {
		t.Fatalf("another provider's window ended %v", ended)
	}
	// A reading older than the mode cannot show the window rolled during it.
	if ended, _ := s.EndRolledWindows(forecastFor("anthropic", 300, t0.Add(9*time.Hour), t0.Add(-time.Minute)), later); len(ended) != 0 {
		t.Fatalf("a reading from before the start ended %v", ended)
	}
}

// TestShippedResourcePoliciesStartRealModes starts a mode from each resource's
// committed model-policy.json, through the same loader production uses.
func TestShippedResourcePoliciesStartRealModes(t *testing.T) {
	read := func(resource string) string {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "resources", resource, "model-policy.json"))
		if err != nil {
			t.Skip("repo resource policies not available: ", err)
		}
		return string(raw)
	}
	s := newService(t)
	s.Tiers = fakeTiers{"codex": read("codex"), "claude-code": read("claude-code")}
	result, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Until: "off", StartedBy: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	codex, _ := result.Mode.SettingsFor("codex")
	claude, _ := result.Mode.SettingsFor("claude")
	if codex.ModelMap["gpt-6.1-terra"] != "gpt-6-luna" || len(claude.ModelMap) == 0 {
		t.Fatalf("shipped tiers produced codex=%+v claude=%+v", codex, claude)
	}
}

func TestModelOverrideMayOnlyLowerAndALearnChallengerNeverExceedsTheIncumbent(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	if _, err := s.Start(ctx, StartRequest{Model: "sonnet", Runner: "claude", Until: "1h"}); err != nil {
		t.Fatalf("lowering to the floor: %v", err)
	}
	for _, model := range []string{"opus", "haiku"} {
		if _, err := s.Start(ctx, StartRequest{Model: model, Runner: "claude", Until: "1h"}); !errors.Is(err, ErrInvalid) {
			t.Errorf("start --model %s must be refused: %v", model, err)
		}
		if _, err := s.StartLearn(ctx, LearnRequest{Runner: "claude", Role: "code.default", Model: model, Until: "off"}); !errors.Is(err, ErrInvalid) {
			t.Errorf("learn challenger %s must be refused: %v", model, err)
		}
	}
	if modes := s.List(); len(modes) != 1 {
		t.Fatalf("refused starts changed the file: %+v", modes)
	}
}

func TestSeparateWritersNeverLoseEachOthersModesAndKeepUnknownFields(t *testing.T) {
	a := newService(t)
	b := &Service{Tiers: a.Tiers, Readings: a.Readings, Now: a.Now, Path: a.Path} // another process: its own mutex
	if err := os.WriteFile(a.Path, []byte(`{"schema_version":1,"future":true,"modes":[{"id":"um-old","kind":"save","started_at":"2026-10-30T10:00:00Z","until":{"kind":"off"},"scope":{"runner":"claude"},"runners":{"claude":{"session_model":"sonnet"}},"future_mode_field":7}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 2)
	go func() {
		_, err := a.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h"})
		done <- err
	}()
	go func() {
		_, err := b.StartLearn(context.Background(), LearnRequest{Runner: "codex", Role: "code.default", Until: "off"})
		done <- err
	}()
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := b.End([]string{"nothing"}, false); err != nil {
		t.Fatal(err)
	}
	if got := len(a.List()); got != 3 {
		t.Fatalf("lost update: %d modes, want 3", got)
	}
	raw, _ := os.ReadFile(a.Path)
	if !strings.Contains(string(raw), "future_mode_field") || !strings.Contains(string(raw), `"future"`) {
		t.Fatalf("a rewrite dropped fields it does not know:\n%s", raw)
	}
}
