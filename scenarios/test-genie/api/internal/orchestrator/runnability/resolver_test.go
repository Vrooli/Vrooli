package runnability_test

import (
	"strings"
	"testing"

	"test-genie/internal/orchestrator/runnability"
	"test-genie/internal/orchestrator/runnability/mocks"
)

func TestStandardResolver_Matrix(t *testing.T) {
	r := runnability.NewResolver()

	allLive := runnability.Surfaces{UI: true, API: true}
	noneLive := runnability.Surfaces{}

	cases := []struct {
		name      string
		caps      runnability.PhaseCapabilities
		rc        runnability.RunContext
		wantKind  runnability.VerdictKind
		reasonHas string
	}{
		{
			name:     "static phase always runs",
			caps:     runnability.PhaseCapabilities{Phase: "structure"},
			rc:       runnability.RunContext{TargetIsSelf: true, LiveSurfaces: noneLive},
			wantKind: runnability.VerdictRun,
		},
		{
			name:     "surface phase runs when surface live (self)",
			caps:     runnability.PhaseCapabilities{Phase: "smoke", NeedsUI: true},
			rc:       runnability.RunContext{TargetIsSelf: true, LiveSurfaces: allLive},
			wantKind: runnability.VerdictRun,
		},
		{
			name:      "surface phase skips when surface absent on self (self-host guard)",
			caps:      runnability.PhaseCapabilities{Phase: "smoke", NeedsUI: true},
			rc:        runnability.RunContext{TargetIsSelf: true, LiveSurfaces: noneLive},
			wantKind:  runnability.VerdictSkip,
			reasonHas: "self-test",
		},
		{
			name:     "surface phase runs when surface absent on other target (start allowed)",
			caps:     runnability.PhaseCapabilities{Phase: "smoke", NeedsUI: true},
			rc:       runnability.RunContext{TargetIsSelf: false, LiveSurfaces: noneLive},
			wantKind: runnability.VerdictRun,
		},
		{
			name: "deferred-lifecycle phase runs on self when surface live (provider decides isolation)",
			caps: runnability.PhaseCapabilities{
				Phase: "playbooks", NeedsUI: true, MutatesLifecycle: true,
				DBIsolation: runnability.DBIsolationRouted, LifecycleDecisionDeferred: true,
			},
			rc:       runnability.RunContext{TargetIsSelf: true, LiveSurfaces: allLive},
			wantKind: runnability.VerdictRun,
		},
		{
			name: "deferred-lifecycle phase still skips on self when its surface is absent",
			caps: runnability.PhaseCapabilities{
				Phase: "playbooks", NeedsUI: true, MutatesLifecycle: true,
				DBIsolation: runnability.DBIsolationRouted, LifecycleDecisionDeferred: true,
			},
			rc:        runnability.RunContext{TargetIsSelf: true, LiveSurfaces: noneLive},
			wantKind:  runnability.VerdictSkip,
			reasonHas: "self-test",
		},
		{
			name:      "missing resource skips",
			caps:      runnability.PhaseCapabilities{Phase: "integration", NeedsAPI: true, RequiredResources: []string{"postgres"}},
			rc:        runnability.RunContext{TargetIsSelf: false, LiveSurfaces: allLive, Resources: map[string]bool{}},
			wantKind:  runnability.VerdictSkip,
			reasonHas: "postgres",
		},
		{
			name:     "required resource present runs",
			caps:     runnability.PhaseCapabilities{Phase: "integration", NeedsAPI: true, RequiredResources: []string{"postgres"}},
			rc:       runnability.RunContext{TargetIsSelf: false, LiveSurfaces: allLive, Resources: map[string]bool{"postgres": true}},
			wantKind: runnability.VerdictRun,
		},
		{
			// Smoke runs on the BAS workflow engine; when BAS is unreachable
			// the gate skips smoke (resource unavailable) rather than failing it.
			name:      "smoke skips when BAS unavailable",
			caps:      runnability.PhaseCapabilities{Phase: "smoke", NeedsUI: true, RequiredResources: []string{runnability.ResourceBAS}},
			rc:        runnability.RunContext{TargetIsSelf: false, LiveSurfaces: allLive, Resources: map[string]bool{runnability.ResourceBAS: false}},
			wantKind:  runnability.VerdictSkip,
			reasonHas: runnability.ResourceBAS,
		},
		{
			name:     "smoke runs when BAS available",
			caps:     runnability.PhaseCapabilities{Phase: "smoke", NeedsUI: true, RequiredResources: []string{runnability.ResourceBAS}},
			rc:       runnability.RunContext{TargetIsSelf: false, LiveSurfaces: allLive, Resources: map[string]bool{runnability.ResourceBAS: true}},
			wantKind: runnability.VerdictRun,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := r.Resolve(tc.caps, tc.rc)
			if got.Kind != tc.wantKind {
				t.Fatalf("Resolve kind = %v, want %v (reason=%q)", got.Kind, tc.wantKind, got.Reason)
			}
			if tc.reasonHas != "" && !strings.Contains(got.Reason, tc.reasonHas) {
				t.Fatalf("Resolve reason = %q, want substring %q", got.Reason, tc.reasonHas)
			}
		})
	}
}

func TestVerdictHelpers(t *testing.T) {
	if !(runnability.Verdict{Kind: runnability.VerdictRun}).IsRun() {
		t.Error("VerdictRun should be IsRun")
	}
	if !(runnability.Verdict{Kind: runnability.VerdictRunDegraded}).IsRun() {
		t.Error("VerdictRunDegraded should be IsRun")
	}
	if (runnability.Verdict{Kind: runnability.VerdictSkip}).IsRun() {
		t.Error("VerdictSkip should not be IsRun")
	}
	if !(runnability.Verdict{Kind: runnability.VerdictSkip}).IsSkip() {
		t.Error("VerdictSkip should be IsSkip")
	}
}

// TestFakeResolver_SatisfiesSeam exercises the test double so its compile-time
// assertion and recording behavior stay honest.
func TestFakeResolver_SatisfiesSeam(t *testing.T) {
	var resolver runnability.Resolver = &mocks.FakeResolver{
		Verdict: runnability.Verdict{Kind: runnability.VerdictSkip, Reason: "canned"},
	}
	got := resolver.Resolve(runnability.PhaseCapabilities{Phase: "x"}, runnability.RunContext{})
	if got.Kind != runnability.VerdictSkip || got.Reason != "canned" {
		t.Fatalf("fake returned %+v", got)
	}

	programmable := &mocks.FakeResolver{
		Func: func(caps runnability.PhaseCapabilities, _ runnability.RunContext) runnability.Verdict {
			return runnability.Verdict{Kind: runnability.VerdictRun, Reason: caps.Phase}
		},
	}
	if v := programmable.Resolve(runnability.PhaseCapabilities{Phase: "smoke"}, runnability.RunContext{}); v.Reason != "smoke" {
		t.Fatalf("programmable fake returned %+v", v)
	}
	if len(programmable.Calls) != 1 {
		t.Fatalf("expected 1 recorded call, got %d", len(programmable.Calls))
	}
}
