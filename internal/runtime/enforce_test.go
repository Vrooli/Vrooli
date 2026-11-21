package runtime

import (
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestEnforceShortCircuitsOnEmptyResolution(t *testing.T) {
	ensureCalls := 0
	deps := enforceDeps{
		resolve: func(_, _ string, _ hostreq.ResolveOptions) (hostreq.Resolution, error) {
			return hostreq.Resolution{}, nil
		},
		ensure: func(_ EnsureOptions, _ hostreq.Resolution) (Report, error) {
			ensureCalls++
			return Report{}, nil
		},
	}
	report, err := enforceWithDeps(deps, Options{Label: "test"})
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if ensureCalls != 0 {
		t.Fatalf("ensure should not run when resolution is empty; calls=%d", ensureCalls)
	}
	if len(report.Tools)+len(report.Safeguards) != 0 {
		t.Fatalf("report should be empty, got %+v", report)
	}
}

func TestEnforcePropagatesResolveErrorWithLabel(t *testing.T) {
	boom := errors.New("boom")
	deps := enforceDeps{
		resolve: func(_, _ string, _ hostreq.ResolveOptions) (hostreq.Resolution, error) {
			return hostreq.Resolution{}, boom
		},
		ensure: func(_ EnsureOptions, _ hostreq.Resolution) (Report, error) {
			t.Fatal("ensure must not run when resolve fails")
			return Report{}, nil
		},
	}
	_, err := enforceWithDeps(deps, Options{Label: "scenario:test"})
	if err == nil {
		t.Fatal("expected resolve error")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped boom, got %v", err)
	}
	if want := "scenario:test"; !containsEnforcementString(err.Error(), want) {
		t.Fatalf("error %q missing label %q", err, want)
	}
}

func TestEnforcePassesThroughToEnsureWhenResolutionHasItems(t *testing.T) {
	resolution := hostreq.Resolution{
		Tools: []hostreqspec.ResolvedRequirement{{Name: "git", Kind: hostreqspec.KindTool, Required: true}},
	}
	var capturedResolve hostreq.ResolveOptions
	var capturedEnsure EnsureOptions
	deps := enforceDeps{
		resolve: func(_, _ string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
			capturedResolve = opts
			return resolution, nil
		},
		ensure: func(opts EnsureOptions, _ hostreq.Resolution) (Report, error) {
			capturedEnsure = opts
			return Report{Environment: opts.Environment}, nil
		},
	}
	_, err := enforceWithDeps(deps, Options{
		Root:        "/root",
		Home:        "/home",
		Environment: "development",
		When:        "develop",
		Resources:   "postgres,redis",
		Scenarios:   "foo",
		Platform:    "linux",
		SudoMode:    "prompt",
		AutoInstall: true,
		Label:       "scenario:foo",
	})
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if capturedResolve.When != "develop" {
		t.Fatalf("When = %q, want develop", capturedResolve.When)
	}
	if capturedResolve.Scenarios != "foo" {
		t.Fatalf("Scenarios = %q, want foo", capturedResolve.Scenarios)
	}
	if capturedResolve.Resources != "postgres,redis" {
		t.Fatalf("Resources = %q", capturedResolve.Resources)
	}
	if capturedResolve.Platform != "linux" {
		t.Fatalf("Platform = %q", capturedResolve.Platform)
	}
	if !capturedEnsure.AutoInstall {
		t.Fatal("AutoInstall should be forwarded")
	}
	if capturedEnsure.SudoMode != "prompt" {
		t.Fatalf("SudoMode = %q", capturedEnsure.SudoMode)
	}
}

func TestEnforceWrapsEnsureErrorUnlessDryRun(t *testing.T) {
	boom := errors.New("install failed")
	resolution := hostreq.Resolution{Tools: []hostreqspec.ResolvedRequirement{{Name: "git", Kind: hostreqspec.KindTool, Required: true}}}
	deps := enforceDeps{
		resolve: func(_, _ string, _ hostreq.ResolveOptions) (hostreq.Resolution, error) {
			return resolution, nil
		},
		ensure: func(_ EnsureOptions, _ hostreq.Resolution) (Report, error) {
			return Report{}, boom
		},
	}
	if _, err := enforceWithDeps(deps, Options{Label: "resource:postgres"}); err == nil {
		t.Fatal("expected ensure error to surface")
	} else if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want wraps boom", err)
	}
	if _, err := enforceWithDeps(deps, Options{Label: "resource:postgres", DryRun: true}); err != nil {
		t.Fatalf("DryRun should not surface ensure error: %v", err)
	}
}

func TestEnforceDefaultsPlatformToCurrentWhenEmpty(t *testing.T) {
	var captured string
	deps := enforceDeps{
		resolve: func(_, _ string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
			captured = opts.Platform
			return hostreq.Resolution{}, nil
		},
	}
	if _, err := enforceWithDeps(deps, Options{}); err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if captured == "" {
		t.Fatal("platform should default to CurrentPlatform")
	}
	if captured != hostreqspec.CurrentPlatform() {
		t.Fatalf("platform = %q, want %q", captured, hostreqspec.CurrentPlatform())
	}
}

func containsEnforcementString(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
