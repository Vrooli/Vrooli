package runs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/freshness-go/treedigest"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	sharedruns "test-genie/internal/shared/runs"

	freshness "github.com/vrooli/freshness-go"
)

func TestSourceIdentityIgnoresRoutedArtifacts(t *testing.T) {
	for _, operation := range []string{"freshness", "find-current"} {
		t.Run(operation, func(t *testing.T) {
			svc, sourceRoot := newTestService(t)
			artifactRoot := newFleetRoot(t)
			svc.SetArtifactRootResolver(testArtifactRoot(artifactRoot))
			source := filepath.Join(sourceRoot, "demo", "relevant.txt")
			if err := os.WriteFile(source, []byte("before"), 0o644); err != nil {
				t.Fatal(err)
			}
			digest, err := treedigest.Compute(filepath.Dir(source))
			if err != nil {
				t.Fatal(err)
			}
			phaseDigest := phases.PhaseSetDigest([]string{"tidiness"})
			seedRecord(t, artifactRoot, sharedruns.RunRecord{
				RunID: "source-qualified", Scenario: "demo", Status: sharedruns.StatusPassed,
				StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(), TreeDigest: digest,
				PhaseSetDigest: phaseDigest, PlannedPhases: []string{"tidiness"},
				Phases: []sharedruns.PhaseRecord{{Name: "tidiness", Status: "passed"}},
			})
			index := sharedruns.NewIndex(filepath.Join(artifactRoot, "demo"))
			if err := index.Finalize("source-qualified", &orchestrator.SuiteExecutionResult{
				SourceFingerprint: digest, SourceScope: "scenario:demo", SourceStable: true,
				ConfigurationFingerprint: "cfg:source",
			}, func(*sharedruns.RunRecord) error { return nil }); err != nil {
				t.Fatal(err)
			}
			svc.planner = fixedExecutionPlanner{preview: &execution.ExecutionPlanPreview{
				ScenarioName: "demo", PhaseSetDigest: phaseDigest, ConfigurationFingerprint: "cfg:source",
			}}
			check := func(wantFresh bool) {
				t.Helper()
				if operation == "freshness" {
					resp, err := svc.CheckFreshness(context.Background(), connect.NewRequest(&runspb.CheckFreshnessRequest{Target: "demo", Phases: []string{"tidiness"}}))
					if err != nil {
						t.Fatal(err)
					}
					wantDigest, err := treedigest.Compute(filepath.Dir(source))
					if err != nil {
						t.Fatal(err)
					}
					if resp.Msg.GetTreeDigest() != wantDigest {
						t.Errorf("source digest = %q, want %q", resp.Msg.GetTreeDigest(), wantDigest)
					}
					if got := resp.Msg.GetPhases()[0].GetStatus() == freshness.StatusFresh; got != wantFresh {
						t.Errorf("fresh = %v, want %v", got, wantFresh)
					}
				} else {
					resp, err := svc.FindRun(context.Background(), connect.NewRequest(&runspb.FindRunRequest{Target: "demo", MatchCurrentSource: true}))
					if err != nil {
						t.Fatal(err)
					}
					if resp.Msg.GetFound() != wantFresh {
						t.Errorf("found = %v, want %v", resp.Msg.GetFound(), wantFresh)
					}
				}
			}
			check(true)
			if err := os.WriteFile(filepath.Join(artifactRoot, "demo", "evidence-only.txt"), []byte("new evidence"), 0o644); err != nil {
				t.Fatal(err)
			}
			check(true)
			if err := os.WriteFile(source, []byte("changed behavior"), 0o644); err != nil {
				t.Fatal(err)
			}
			check(false)
		})
	}
}

// The fresh/stale/unknown verdict semantics are owned (and tested) by the
// shared freshness-go package. The tests here cover what stays test-genie's:
// the wire conversion and the required-set SSOT.

func TestToFreshnessResponseConvertsAllFields(t *testing.T) {
	rec := sharedruns.RunRecord{
		RunID:       "r1",
		Status:      sharedruns.StatusPassed,
		TreeDigest:  "td:x",
		CompletedAt: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Phases:      []sharedruns.PhaseRecord{{Name: "unit", Status: "passed"}},
	}
	report := freshness.Check([]sharedruns.RunRecord{rec}, "td:x", []string{"unit", "business"})
	resp := toFreshnessResponse(report)

	if resp.GetTreeDigest() != "td:x" {
		t.Fatalf("TreeDigest = %q", resp.GetTreeDigest())
	}
	if len(resp.GetPhases()) != 2 {
		t.Fatalf("got %d phases, want 2", len(resp.GetPhases()))
	}
	unit := resp.GetPhases()[0]
	if unit.GetPhase() != "unit" || unit.GetStatus() != freshness.StatusFresh ||
		unit.GetLastRunId() != "r1" || unit.GetLastRunCompletedAt() != "2026-06-10T00:00:00Z" {
		t.Fatalf("unit verdict not converted faithfully: %+v", unit)
	}
	if business := resp.GetPhases()[1]; business.GetStatus() != freshness.StatusStale {
		t.Fatalf("business verdict = %q, want stale", business.GetStatus())
	}
}

func TestFreshnessDefaultSetIsStableRequiredEvidenceProfile(t *testing.T) {
	want := []string{"structure", "docs", "unit", "business", "proto"}
	got := phases.FreshnessRequired()
	if len(got) != len(want) {
		t.Fatalf("FreshnessRequired = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("FreshnessRequired = %v, want %v", got, want)
		}
	}
	// The promoted business phase must be part of the required set — that is
	// the whole point of WS-B feeding WS-D.
	found := false
	for _, p := range got {
		if p == "business" {
			found = true
		}
	}
	if !found {
		t.Fatal("required set must include the business phase")
	}
}
