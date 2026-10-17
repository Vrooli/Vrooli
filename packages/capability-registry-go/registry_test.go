package capabilityregistry

import (
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProjectManifestUsesManifestAuthorityAndOverlay(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/service.json"
	data := `{"dependencies":{"scenarios":{"zeta":{"enabled":true,"required":true,"startup_policy":"try_start","description":"z"},"disabled":{"enabled":false}},"resources":{"postgres":{"enabled":true,"required":false,"startup_policy":"on_demand","description":"db"}}}}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	defs, err := ProjectManifest(path, map[string]Overlay{"zeta": {ID: "zeta", Name: "Zeta", Features: []string{"stats"}, ActionKind: ActionKindScenarioRestart}})
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 2 || defs[0].ID != "postgres" || defs[1].ID != "zeta" {
		t.Fatalf("defs = %+v", defs)
	}
	if defs[1].Name != "Zeta" || !defs[1].Required || defs[1].StartupPolicy != "try_start" || len(defs[1].Features) != 1 {
		t.Fatalf("overlay = %+v", defs[1])
	}
	if defs[1].Origin != "manifest" {
		t.Fatalf("origin = %q, want manifest", defs[1].Origin)
	}
}

func TestProjectManifestPreservesOverlayIdentityAndManifestSlug(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/service.json"
	data := `{"dependencies":{"resources":{"claude-code":{"required":false,"startup_policy":"manual","description":"agent"}}}}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	defs, err := ProjectManifest(path, map[string]Overlay{
		"claude-code": {ID: "claude-code", IntegrationID: "claude", ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource install claude-code --json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 || defs[0].ID != "claude" || defs[0].DependencySlug != "claude-code" || defs[0].ActionKind != ActionKindOperatorCommand {
		t.Fatalf("projected alias = %+v", defs)
	}
}

func TestValidateAcceptsControlPlane(t *testing.T) {
	reg := New([]Def{{ID: "core", Description: "control plane", DependencyKind: DependencyControlPlane, DependencySlug: "vrooli"}}, map[string]Checker{"core": checkerFunc(func(context.Context) (Status, string) { return StatusAvailable, "ok" })}, 0)
	if err := reg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateOverlaysRejectUnknownAndIllegalActions(t *testing.T) {
	defs := []Def{{DependencySlug: "postgres", DependencyKind: DependencyResource}}
	if err := ValidateOverlays(defs, map[string]Overlay{"missing": {}}); err == nil {
		t.Fatal("unknown overlay accepted")
	}
	if err := ValidateOverlays(defs, map[string]Overlay{"postgres": {ActionKind: ActionKindScenarioRestart}}); err == nil {
		t.Fatal("resource restart accepted")
	}
}

func TestRegistryCoalescesConcurrentFullResolves(t *testing.T) {
	checker := &blockingChecker{started: make(chan struct{}), release: make(chan struct{})}
	reg := New([]Def{{ID: "svc", Description: "service", DependencyKind: DependencyScenario, DependencySlug: "svc"}}, map[string]Checker{"svc": checker}, time.Minute)
	var wg sync.WaitGroup
	results := make(chan []State, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- reg.Resolve(context.Background())
		}()
	}
	select {
	case <-checker.started:
	case <-time.After(time.Second):
		t.Fatal("first resolve did not start")
	}
	close(checker.release)
	wg.Wait()
	close(results)
	if calls := checker.calls.Load(); calls != 1 {
		t.Fatalf("checker calls = %d, want one coalesced probe", calls)
	}
	for states := range results {
		if len(states) != 1 || states[0].Status != StatusAvailable {
			t.Fatalf("states = %+v", states)
		}
	}
}

type blockingChecker struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (c *blockingChecker) Check(ctx context.Context) (Status, string) {
	c.calls.Add(1)
	select {
	case c.started <- struct{}{}:
	default:
	}
	select {
	case <-c.release:
		return StatusAvailable, "released"
	case <-ctx.Done():
		return StatusUnknown, ctx.Err().Error()
	}
}

type checkerFunc func(context.Context) (Status, string)

func (f checkerFunc) Check(ctx context.Context) (Status, string) { return f(ctx) }

func validRegistry() *Registry {
	return New([]Def{{ID: "audio-tools", Name: "Audio Tools", Description: "speech", DependencyKind: DependencyScenario, DependencySlug: "audio-tools"}}, map[string]Checker{
		"audio-tools": checkerFunc(func(context.Context) (Status, string) { return StatusAvailable, "healthy" }),
	}, 0)
}

func TestValidateRejectsMalformedDefinitions(t *testing.T) {
	cases := []struct {
		name string
		def  Def
		want string
	}{
		{"id", Def{Description: "x", DependencyKind: DependencyScenario, DependencySlug: "x"}, "no id"},
		{"description", Def{ID: "x", DependencyKind: DependencyScenario, DependencySlug: "x"}, "no description"},
		{"kind", Def{ID: "x", Description: "x", DependencySlug: "x"}, "invalid dependency kind"},
		{"slug", Def{ID: "x", Description: "x", DependencyKind: DependencyScenario}, "no dependency slug"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := New([]Def{tc.def}, nil, 0).Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDescribeContainsDefinitionsAndStates(t *testing.T) {
	data, err := validRegistry().Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"audio-tools"`) || !strings.Contains(string(data), `"available"`) {
		t.Fatalf("Describe() = %s", data)
	}
}

func TestValidateRequiresReachabilityChecker(t *testing.T) {
	reg := New([]Def{{ID: "audio-tools", Description: "speech", DependencyKind: DependencyScenario, DependencySlug: "audio-tools"}}, nil, 0)
	if err := reg.Validate(); err == nil || !strings.Contains(err.Error(), "reachability checker") {
		t.Fatalf("Validate() error = %v, want missing reachability checker", err)
	}
}

func TestValidateStatesRequiresRecoveryAction(t *testing.T) {
	reg := validRegistry()
	if err := reg.ValidateStates([]State{{Def: Def{ID: "audio-tools"}, Status: StatusUnavailable}}); err == nil || !strings.Contains(err.Error(), "operator action") {
		t.Fatalf("ValidateStates() error = %v, want missing operator action", err)
	}
}

func TestOptionalUnavailableProviderDoesNotMarkOtherCapability(t *testing.T) {
	states := []State{
		{Def: Def{ID: "optional", Criticality: CriticalityOptional}, Status: StatusUnavailable},
		{Def: Def{ID: "working", Criticality: CriticalityRequired}, Status: StatusAvailable},
	}
	groups := RollupByCapability(states, func(state State) []string {
		if state.ID == "optional" {
			return []string{"enhancement"}
		}
		return []string{"core"}
	})
	if len(groups) != 2 || groups[0].Serviceable || !groups[1].Serviceable {
		t.Fatalf("rollups = %+v, want optional unavailable and core serviceable", groups)
	}
}

func TestCriticalityDefaultsToOptional(t *testing.T) {
	if got := (Def{}).ResolvedCriticality(); got != CriticalityOptional {
		t.Fatalf("empty criticality = %q, want %q", got, CriticalityOptional)
	}
	if got := (Def{Criticality: CriticalityRequired}).ResolvedCriticality(); got != CriticalityRequired {
		t.Fatalf("required criticality = %q, want %q", got, CriticalityRequired)
	}
}

func TestPlatformUnsupportedProviderIsReportedByRollup(t *testing.T) {
	state := State{Def: Def{
		ID:       "native",
		Platform: PlatformVerdict{Support: PlatformUnsupported, Reason: "no native adapter"},
	}, Status: StatusUnavailable, Message: "unavailable by design: no native adapter"}
	groups := RollupByCapability([]State{state}, func(State) []string { return []string{"speech"} })
	if len(groups) != 1 || groups[0].Serviceable || len(groups[0].UnavailableProviders) != 1 || groups[0].UnavailableProviders[0] != "native" {
		t.Fatalf("rollups = %+v, want reported unsupported provider", groups)
	}
	if !strings.Contains(state.Message, "unavailable by design") {
		t.Fatalf("message = %q, want by-design explanation", state.Message)
	}
}

func TestRollupByCapabilityAvailableProviderMakesCapabilityServiceable(t *testing.T) {
	states := []State{
		{Def: Def{ID: "local"}, Status: StatusUnavailable},
		{Def: Def{ID: "byok"}, Status: StatusAvailable},
	}
	groups := RollupByCapability(states, func(State) []string { return []string{"speech"} })
	if len(groups) != 1 || !groups[0].Serviceable {
		t.Fatalf("rollups = %+v, want serviceable with one available provider", groups)
	}
	if len(groups[0].UnavailableProviders) != 1 || groups[0].UnavailableProviders[0] != "local" {
		t.Fatalf("unavailable providers = %v, want [local]", groups[0].UnavailableProviders)
	}
}

func TestResolveLivenessDoesNotReadFullTierCache(t *testing.T) {
	defs := []Def{{ID: "provider", Name: "Provider"}}
	full := checkerFunc(func(context.Context) (Status, string) { return StatusUnavailable, "deep failure" })
	live := checkerFunc(func(context.Context) (Status, string) { return StatusAvailable, "reachable" })
	reg := New(defs, map[string]Checker{"provider": full}, time.Minute)
	reg.SetLivenessCheckers(map[string]Checker{"provider": live})

	reg.Resolve(context.Background())
	states := reg.ResolveLiveness(context.Background())
	if states[0].Status != StatusAvailable || states[0].Message != "reachable" {
		t.Fatalf("liveness result = %+v, want independent live result", states[0])
	}
}

func TestResolveLivenessRunsLivenessCheckers(t *testing.T) {
	calls := 0
	defs := []Def{{ID: "one", Name: "One"}, {ID: "two", Name: "Two"}}
	live := checkerFunc(func(context.Context) (Status, string) { calls++; return StatusAvailable, "ok" })
	reg := New(defs, map[string]Checker{
		"one": checkerFunc(func(context.Context) (Status, string) { return StatusUnavailable, "full" }),
		"two": checkerFunc(func(context.Context) (Status, string) { return StatusUnavailable, "full" }),
	}, time.Minute)
	reg.SetLivenessCheckers(map[string]Checker{"one": live, "two": live})

	reg.ResolveLiveness(context.Background())
	if calls != len(defs) {
		t.Fatalf("liveness checker calls = %d, want %d", calls, len(defs))
	}
}

func TestResolveLivenessWithoutCheckersDoesNotRunFullTier(t *testing.T) {
	fullCalls := 0
	reg := New([]Def{{ID: "provider", Name: "Provider"}}, map[string]Checker{
		"provider": checkerFunc(func(context.Context) (Status, string) {
			fullCalls++
			return StatusAvailable, "full result"
		}),
	}, time.Minute)

	states := reg.ResolveLiveness(context.Background())
	if fullCalls != 0 {
		t.Fatalf("full checker calls = %d, want 0", fullCalls)
	}
	if states[0].Status != StatusUnknown {
		t.Fatalf("state = %+v, want unknown without a liveness checker", states[0])
	}
}

func TestOneSlowCheckerDegradesOnlyItself(t *testing.T) {
	defs := []Def{{ID: "slow", Name: "Slow"}, {ID: "fast-a", Name: "Fast A"}, {ID: "fast-b", Name: "Fast B"}}
	slow := checkerFunc(func(ctx context.Context) (Status, string) {
		select {
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
		}
		return StatusUnavailable, "slow checker timed out"
	})
	fast := checkerFunc(func(context.Context) (Status, string) { return StatusAvailable, "ok" })
	reg := New(defs, map[string]Checker{"slow": slow, "fast-a": fast, "fast-b": fast}, time.Minute)

	states := reg.Resolve(context.Background())
	if states[0].Status != StatusUnavailable {
		t.Fatalf("slow state = %+v, want unavailable", states[0])
	}
	for _, state := range states[1:] {
		if state.Status != StatusAvailable {
			t.Fatalf("fast state = %+v, want available", state)
		}
	}
}

func TestUnreachedProviderReportsUnknown(t *testing.T) {
	defs := []Def{{ID: "first", Name: "First"}, {ID: "second", Name: "Second"}}
	reg := New(defs, map[string]Checker{
		"first": checkerFunc(func(ctx context.Context) (Status, string) {
			select {
			case <-time.After(30 * time.Millisecond):
			case <-ctx.Done():
			}
			return StatusUnavailable, "timed out"
		}),
		"second": checkerFunc(func(context.Context) (Status, string) { return StatusAvailable, "ok" }),
	}, time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	states := reg.Resolve(ctx)
	if states[1].Status != StatusUnknown || states[1].Message != "not evaluated: deadline reached" {
		t.Fatalf("unreached state = %+v, want unknown deadline state", states[1])
	}
}

func TestFullRefreshDoesNotBlockLivenessChecks(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	full := checkerFunc(func(context.Context) (Status, string) {
		close(started)
		<-release
		return StatusAvailable, "full"
	})
	live := checkerFunc(func(context.Context) (Status, string) { return StatusAvailable, "live" })
	reg := New(
		[]Def{{ID: "provider", Name: "Provider"}},
		map[string]Checker{"provider": full},
		time.Minute,
	)
	reg.SetLivenessCheckers(map[string]Checker{"provider": live})

	refreshDone := make(chan struct{})
	go func() {
		reg.ResolveForce(context.Background())
		close(refreshDone)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("full refresh did not start")
	}

	livenessDone := make(chan bool, 1)
	go func() { livenessDone <- reg.IsProviderLive(context.Background(), "provider") }()
	select {
	case got := <-livenessDone:
		if !got {
			t.Fatal("liveness check = false, want true")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("liveness check blocked behind full refresh")
	}

	close(release)
	select {
	case <-refreshDone:
	case <-time.After(time.Second):
		t.Fatal("full refresh did not finish")
	}
}
