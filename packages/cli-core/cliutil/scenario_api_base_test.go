package cliutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveScenarioAPIBaseURLUsesConfiguredBaseBeforeDiscovery(t *testing.T) {
	t.Setenv("VROOLI_HOME", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("OFFER_DESK_API_BASE_URL", " https://offer.example/api/v1/ ")
	called := false
	restore := SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome {
		called = true
		return ScenarioPortOutcome{Port: "19123"}
	})
	defer restore()

	got, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{EnvVars: []string{"OFFER_DESK_API_BASE_URL"}, Target: "offer-desk"})
	if err != nil || got != "https://offer.example/api/v1" {
		t.Fatalf("base = %q, err=%v", got, err)
	}
	if called {
		t.Fatal("port lookup ran despite configured base URL")
	}
}

func TestResolveScenarioAPIBaseURLPreservesQuerySlash(t *testing.T) {
	t.Setenv("PEER_API_BASE", "https://peer.example/api?tenant=/")
	got, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{Target: "peer", EnvVars: []string{"PEER_API_BASE"}})
	if err != nil || got != "https://peer.example/api?tenant=/" {
		t.Fatalf("base=%q err=%v", got, err)
	}
}

func TestResolveScenarioAPIBaseURLUsesSharedPortLookupAndIsFreshByDefault(t *testing.T) {
	t.Setenv("VROOLI_HOME", filepath.Join(t.TempDir(), "runtime"))
	var targets, portVars []string
	count := 0
	restore := SetPortLookupRunner(func(_ context.Context, target, portVar string) ScenarioPortOutcome {
		targets = append(targets, target)
		portVars = append(portVars, portVar)
		count++
		return ScenarioPortOutcome{Port: []string{"19123", "19124"}[count-1]}
	})
	defer restore()

	for _, want := range []string{"http://localhost:19123", "http://localhost:19124"} {
		got, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{Target: "peer"})
		if err != nil || got != want {
			t.Fatalf("base = %q, err=%v; want %q", got, err, want)
		}
	}
	if !reflect.DeepEqual(targets, []string{"peer", "peer"}) || !reflect.DeepEqual(portVars, []string{"API_PORT", "API_PORT"}) {
		t.Fatalf("lookups targets=%v ports=%v", targets, portVars)
	}
}

func TestResolveScenarioAPIBaseURLUsesIPv6SafeAuthority(t *testing.T) {
	got, err := scenarioLoopbackBase("19125")
	if err != nil || got != "http://localhost:19125" {
		t.Fatalf("base=%q err=%v", got, err)
	}
	if _, err := NormalizeScenarioAPIBaseURL("file:///tmp/api"); err == nil {
		t.Fatal("non-http URL accepted")
	}
	if _, err := NormalizeScenarioAPIBaseURL("https://host.example/prefix"); err != nil {
		t.Fatalf("https base rejected: %v", err)
	}
	for _, invalid := range []string{"", "oops", "+80", "0", "65536"} {
		if _, err := scenarioLoopbackBase(invalid); err == nil {
			t.Errorf("invalid port %q accepted", invalid)
		}
	}
}

func TestNormalizeScenarioAPIBaseURLValidatesAuthorityAndPort(t *testing.T) {
	t.Parallel()

	for _, base := range []string{
		"http://localhost",
		"http://localhost:1",
		"https://peer.example:65535/api",
		"http://[::1]:8080/api",
		"https://[2001:db8::1]/api",
	} {
		t.Run(base, func(t *testing.T) {
			if got, err := NormalizeScenarioAPIBaseURL(base); err != nil || got != base {
				t.Fatalf("NormalizeScenarioAPIBaseURL(%q) = %q, %v; want unchanged valid base", base, got, err)
			}
		})
	}

	for _, base := range []string{
		"http://peer.example:0",
		"http://peer.example:65536",
		"http://peer.example:",
		"http://:8080",
		"http://[::1]:",
	} {
		t.Run(base, func(t *testing.T) {
			if got, err := NormalizeScenarioAPIBaseURL(base); err == nil {
				t.Fatalf("NormalizeScenarioAPIBaseURL(%q) = %q, want invalid authority/port error", base, got)
			}
		})
	}
}

func TestResolveScenarioAPIBaseURLRequiresPeerTarget(t *testing.T) {
	if _, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{}); err == nil {
		t.Fatal("missing target accepted")
	}
}

func TestResolveScenarioAPIBaseURLLiveFallbackIsTypedAndOptIn(t *testing.T) {
	t.Setenv("VROOLI_HOME", filepath.Join(t.TempDir(), "runtime"))
	var targets []string
	restore := SetPortLookupRunner(func(_ context.Context, target, _ string) ScenarioPortOutcome {
		targets = append(targets, target)
		if target == "peer@shadow" {
			return ScenarioPortOutcome{Err: &ScenarioPortError{Code: ScenarioPortCodeNotRunning}}
		}
		return ScenarioPortOutcome{Port: "19126"}
	})
	defer restore()

	if _, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{Target: "peer@shadow"}); err == nil {
		t.Fatal("fallback occurred without opt-in")
	}
	if len(targets) != 1 {
		t.Fatalf("default lookup targets = %v", targets)
	}
	targets = nil
	got, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{Target: "peer@shadow", AllowLiveFallback: true})
	if err != nil || got != "http://localhost:19126" {
		t.Fatalf("fallback base=%q err=%v", got, err)
	}
	if !reflect.DeepEqual(targets, []string{"peer@shadow", "peer"}) {
		t.Fatalf("fallback targets=%v", targets)
	}
}

func TestResolveScenarioAPIBaseURLNeverFallsBackForVariantDependency(t *testing.T) {
	t.Setenv("VROOLI_HOME", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv(EnvInstanceVariant, "presentation")
	t.Setenv(EnvVariantDependencies, "peer")
	var targets []string
	restore := SetPortLookupRunner(func(_ context.Context, target, _ string) ScenarioPortOutcome {
		targets = append(targets, target)
		return ScenarioPortOutcome{Err: &ScenarioPortError{Code: ScenarioPortCodeNotRunning}}
	})
	defer restore()

	_, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{Target: "peer", AllowLiveFallback: true})
	if err == nil {
		t.Fatal("missing variant dependency fell back to live")
	}
	if !reflect.DeepEqual(targets, []string{"peer@presentation"}) {
		t.Fatalf("variant dependency targets = %v, want only presentation target", targets)
	}
}

func TestResolveScenarioAPIBaseURLPreservesExplicitNamedVariant(t *testing.T) {
	t.Setenv("VROOLI_HOME", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv(EnvInstanceVariant, "live")
	t.Setenv(EnvVariantDependencies, "")
	var targets []string
	restore := SetPortLookupRunner(func(_ context.Context, target, _ string) ScenarioPortOutcome {
		targets = append(targets, target)
		return ScenarioPortOutcome{Err: &ScenarioPortError{Code: ScenarioPortCodeNotRunning}}
	})
	defer restore()

	_, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{Target: "peer@presentation", AllowLiveFallback: true})
	if err == nil {
		t.Fatal("explicit named variant fell back to live")
	}
	if !reflect.DeepEqual(targets, []string{"peer@presentation"}) {
		t.Fatalf("explicit variant targets = %v, want only presentation target", targets)
	}
}

func TestResolveScenarioAPIBaseURLDoesNotFallbackOnTransientFailure(t *testing.T) {
	t.Setenv("VROOLI_HOME", filepath.Join(t.TempDir(), "runtime"))
	var targets []string
	restore := SetPortLookupRunner(func(_ context.Context, target, _ string) ScenarioPortOutcome {
		targets = append(targets, target)
		return ScenarioPortOutcome{Err: errors.New("runtime timeout")}
	})
	defer restore()
	_, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{Target: "peer@shadow", AllowLiveFallback: true})
	if err == nil || !reflect.DeepEqual(targets, []string{"peer@shadow"}) {
		t.Fatalf("err=%v targets=%v", err, targets)
	}
}

func TestResolveScenarioAPIBaseURLDoesNotRequireHostSpecificRuntimeFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, "runtime"))
	if err := os.MkdirAll(filepath.Join(home, "runtime"), 0o700); err != nil {
		t.Fatal(err)
	}
	restore := SetPortLookupRunner(func(context.Context, string, string) ScenarioPortOutcome { return ScenarioPortOutcome{Port: "19127"} })
	defer restore()
	got, err := ResolveScenarioAPIBaseURL(context.Background(), ScenarioAPIBaseOptions{Target: "peer"})
	if err != nil || got != "http://localhost:19127" {
		t.Fatalf("base=%q err=%v", got, err)
	}
}
