package usagemodes

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/usageforecast"

	"github.com/vrooli/cli-core/usagemode"
)

// newLearnService keeps the mode file where the shared readers look for it.
func newLearnService(t *testing.T) *Service {
	t.Helper()
	t.Setenv("VROOLI_HOME", t.TempDir())
	s := newService(t)
	s.Path = usagemode.Path()
	return s
}

func startLearn(t *testing.T, s *Service, mutate func(*LearnRequest)) (usagemode.Mode, error) {
	t.Helper()
	req := LearnRequest{Runner: "codex", Role: "code.economy.delivery", Until: "90d", StartedBy: "operator@example.com"}
	if mutate != nil {
		mutate(&req)
	}
	return s.StartLearn(context.Background(), req)
}

func TestStartLearnBuildsAnInertLookingModeWithTheChallengerInsideLearn(t *testing.T) {
	s := newLearnService(t)
	mode, err := startLearn(t, s, func(r *LearnRequest) { r.Sessions = true })
	if err != nil {
		t.Fatal(err)
	}
	if mode.Kind != usagemode.KindLearn || mode.Scope.Role != "code.economy.delivery" || mode.StartedBy != "operator@example.com" || mode.ExpiresAt == nil || mode.ExpiresAt.Sub(t0) != 90*24*time.Hour {
		t.Fatalf("mode = %+v", mode)
	}
	if len(mode.Runners) != 0 {
		t.Fatalf("a reader that does not know Learn must find no settings, got %+v", mode.Runners)
	}
	settings, ok := mode.ChallengerSettings("codex")
	if !ok || settings.ModelMap["gpt-6.1-terra"] != "gpt-6-luna" || mode.Learn.SplitPercent != 50 {
		t.Fatalf("challenger = %+v split %d", settings, mode.Learn.SplitPercent)
	}
	if mode.Learn.Sessions == nil || !mode.Learn.Sessions.OptIn || mode.Learn.Sessions.OptedInBy != "operator@example.com" || mode.Learn.Sessions.BlockDays != 1 {
		t.Fatalf("session opt-in = %+v", mode.Learn.Sessions)
	}
	// Sessions are opt-in only.
	plain, err := startLearn(t, newService(t), nil)
	if err != nil || plain.Learn.Sessions != nil {
		t.Fatalf("sessions must not join unless asked: %+v %v", plain.Learn.Sessions, err)
	}
}

func TestStartLearnRefusesWhatMustNeverBeRandomized(t *testing.T) {
	cases := map[string]func(*LearnRequest){
		"exempt role from the resource": func(r *LearnRequest) { r.Role = "code.fallback" },
		"default exempt role":           func(r *LearnRequest) { r.Role = "code.review" },
		"no role":                       func(r *LearnRequest) { r.Role = "" },
		"unknown runner":                func(r *LearnRequest) { r.Runner = "grok" },
		"split of 100":                  func(r *LearnRequest) { r.SplitPercent = 100 },
		"no until":                      func(r *LearnRequest) { r.Until = "" },
	}
	for name, mutate := range cases {
		if _, err := startLearn(t, newService(t), mutate); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

func TestStartLearnSeparatesFromSaveModes(t *testing.T) {
	s := newLearnService(t)
	learn, err := startLearn(t, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := startLearn(t, s, nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("a second comparison for the same role and runner would discard assignments: %v", err)
	}
	// A save mode with the identical scope does not replace the comparison.
	if _, err := s.Start(context.Background(), StartRequest{Preset: PresetOneTierDown, Runner: "codex", Role: "code.economy.delivery", Until: "2h"}); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range s.List() {
		ids = append(ids, m.ID)
	}
	if len(ids) != 2 || (ids[0] != learn.ID && ids[1] != learn.ID) {
		t.Fatalf("modes = %v, want the comparison kept beside the save mode", ids)
	}
	// The save mode wins resolution; the comparison is not consulted.
	query := usagemode.Query{Runner: "codex", Role: "code.economy.delivery", Now: t0.Add(time.Minute)}
	if _, ok := usagemode.ResolveLearn(query); ok {
		t.Fatal("save must take priority over learn")
	}
}

func willExhaust(provider, status string) usageforecast.Forecast {
	return usageforecast.Forecast{Window: &usageforecast.Window{Provider: provider}, Status: status}
}

func TestForecastPausesLearnUntilTheWindowIsSafeAgain(t *testing.T) {
	s := newLearnService(t)
	mode, err := startLearn(t, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	paused, resumed, err := s.ApplyForecasts([]usageforecast.Forecast{willExhaust("anthropic", usageforecast.StatusWillExhaust), willExhaust("openai", usageforecast.StatusOnTrack)}, t0)
	if err != nil || len(paused) != 0 || len(resumed) != 0 {
		t.Fatalf("another provider running out must not pause this comparison: %v %v %v", paused, resumed, err)
	}
	paused, _, err = s.ApplyForecasts([]usageforecast.Forecast{willExhaust("openai", usageforecast.StatusWillExhaust)}, t0)
	if err != nil || len(paused) != 1 || paused[0] != mode.ID {
		t.Fatalf("paused = %v %v", paused, err)
	}
	if _, ok := usagemode.ResolveLearn(usagemode.Query{Runner: "codex", Role: "code.economy.delivery", Now: t0.Add(time.Minute)}); ok {
		t.Fatal("a paused comparison must not assign new work")
	}
	if again, _, _ := s.ApplyForecasts([]usageforecast.Forecast{willExhaust("openai", usageforecast.StatusWillExhaust)}, t0); len(again) != 0 {
		t.Fatalf("pausing is idempotent, got %v", again)
	}
	_, resumed, err = s.ApplyForecasts([]usageforecast.Forecast{willExhaust("openai", usageforecast.StatusOnTrack)}, t0.Add(time.Hour))
	if err != nil || len(resumed) != 1 {
		t.Fatalf("resumed = %v %v", resumed, err)
	}
	if _, ok := usagemode.ResolveLearn(usagemode.Query{Runner: "codex", Role: "code.economy.delivery", Now: t0.Add(2 * time.Hour)}); !ok {
		t.Fatal("the comparison should assign work again once the window is safe")
	}
}

func TestAnOperatorPauseIsNotResumedByTheForecast(t *testing.T) {
	s := newLearnService(t)
	mode, err := startLearn(t, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	mode.Learn.Paused, mode.Learn.PausedReason = true, "operator: waiting for the release"
	if err := usagemode.WriteFile(s.path(), usagemode.File{Modes: []usagemode.Mode{mode}}); err != nil {
		t.Fatal(err)
	}
	if _, resumed, _ := s.ApplyForecasts([]usageforecast.Forecast{willExhaust("openai", usageforecast.StatusOnTrack)}, t0); len(resumed) != 0 {
		t.Fatalf("resumed %v a pause the forecast did not make", resumed)
	}
}
