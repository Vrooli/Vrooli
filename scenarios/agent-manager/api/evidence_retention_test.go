package main

import (
	"path/filepath"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEffortLifecycleTreatsOnlyWithdrawnOrFinalizedDeliveryAsTerminal(t *testing.T) {
	finalized := time.Date(2026, 10, 26, 13, 50, 0, 0, time.UTC)
	updated := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		e        *pb.EffortEnrollment
		terminal bool
		at       time.Time
	}{
		{"live", &pb.EffortEnrollment{EffortRef: "effort:a", DisplayName: "x"}, false, time.Time{}},
		{"withdrawn", &pb.EffortEnrollment{EffortRef: "effort:b", Withdrawn: true, UpdatedAt: timestamppb.New(updated)}, true, updated},
		{"stopped not finalized", &pb.EffortEnrollment{EffortRef: "effort:c", Delivery: &pb.EffortDelivery{State: "stopped"}}, false, time.Time{}},
		{"paused", &pb.EffortEnrollment{EffortRef: "effort:d", Delivery: &pb.EffortDelivery{State: "paused", FinalizedAt: timestamppb.New(finalized)}}, false, time.Time{}},
		{"finalized", &pb.EffortEnrollment{EffortRef: "effort:e", Delivery: &pb.EffortDelivery{
			State: "stopped", CommissionId: "nh-1", FinalizedAt: timestamppb.New(finalized),
			Outcome: &pb.EffortGoalHome{Path: ".vrooli/work/delivery/notification-hub-delivery"},
		}}, true, finalized},
	}
	for _, tc := range cases {
		got := effortLifecycle(tc.e)
		if got.Terminal != tc.terminal || !got.TerminalAt.Equal(tc.at) || got.Ref != tc.e.EffortRef {
			t.Errorf("%s: %+v", tc.name, got)
		}
	}
	keys := effortLifecycle(cases[4].e).Keys
	if keys[len(keys)-1] != ".vrooli/work/delivery/notification-hub-delivery" || keys[len(keys)-2] != "nh-1" {
		t.Errorf("delivery keys = %v", keys)
	}
}

func TestDestinationHomeNamesTheDeliveryHome(t *testing.T) {
	for ref, want := range map[string]string{
		"path:.vrooli/work/delivery/desktop-packaging-delivery/CHARTER.md":                  ".vrooli/work/delivery/desktop-packaging-delivery",
		"workspace:rcl-prompt-manager-ux-design-campaign/README.md#outcome":                 "rcl-prompt-manager-ux-design-campaign",
		"sources/operator-request-2026-09-14.md#s4--this-efforts-founding-request-verbatim": "sources",
		"": "",
	} {
		if got := destinationHome(ref); got != want {
			t.Errorf("destinationHome(%q) = %q, want %q", ref, got, want)
		}
	}
}

func TestEvidenceRootsResolveThroughTheRuntimeHomeContract(t *testing.T) {
	home := t.TempDir()
	want := func(key string) string {
		t.Helper()
		path, err := repocontract.RuntimeHomeEntryPath(home, key)
		if err != nil {
			t.Fatal(err)
		}
		return path
	}
	roots, err := resolveEvidenceRoots(home, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if roots.evidence != want(repocontract.HomeKeyEvidence) {
		t.Errorf("evidence root = %q, want the contract entry %q", roots.evidence, want(repocontract.HomeKeyEvidence))
	}
	references := []string{want(repocontract.HomeKeyPlanArtifacts), want(repocontract.HomeKeyResearch)}
	if len(roots.references) != len(references) || roots.references[0] != references[0] || roots.references[1] != references[1] {
		t.Errorf("reference roots = %v, want %v", roots.references, references)
	}

	override := filepath.Join(t.TempDir(), "evidence")
	roots, err = resolveEvidenceRoots(home, func(key string) string {
		if key == "AGENT_MANAGER_EVIDENCE_ROOT" {
			return override
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if roots.evidence != override || roots.references[0] != references[0] {
		t.Errorf("override moved the wrong roots: %+v", roots)
	}
}
