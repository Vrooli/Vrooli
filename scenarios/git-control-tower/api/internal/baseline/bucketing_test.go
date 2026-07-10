package baseline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func TestProductionDoesNotRouteOnTestGeniePhaseKeys(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	paths := []string{
		".",
		"../../handlers/baseline",
		"../../handlers/workflowreplay",
		"../../../ui/src/features/baselines",
		"../../../ui/src/components/ScenarioReviewPanelWorkflows.tsx",
	}
	forbidden := []string{
		"surfacePhases", "phaseSurface", "ListRunVisuals", "playbooksPhase",
		`Phase: "structure"`, `Phase: "standards"`, `Phase: "unit"`,
		`Phase: "integration"`, `Phase: "smoke"`, `Phase: "playbooks"`,
	}

	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("stat %s: %v", root, err)
		}
		files := []string{root}
		if info.IsDir() {
			files = files[:0]
			err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if !entry.IsDir() && !strings.Contains(path, "_test.go") && !strings.Contains(path, ".test.") {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", root, err)
			}
		}
		for _, path := range files {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			for _, token := range forbidden {
				if strings.Contains(string(content), token) {
					t.Errorf("production phase routing token %q remains in %s", token, path)
				}
			}
		}
	}
}

func TestUnknownPhaseAndTypedReasonsRemainLossless(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	phase := &runspb.PhaseDiff{
		Phase: "future-provider-phase", Verdict: "not-comparable",
		DescriptorB: &runspb.RunPhaseDescriptor{
			Phase: "future-provider-phase", DisplayName: "Future Provider", Provider: "future-health",
			EvidenceKinds: []string{"application/x-future-evidence"},
		},
		Reasons: []*runspb.PhaseComparisonReason{{
			Code:   runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_NEW_PHASE,
			Detail: "introduced after baseline capture",
		}},
	}
	cmp := CompareResult{Verdict: "not-comparable", Phases: []*runspb.PhaseDiff{phase}}
	if cmp.Phases[0] != phase {
		t.Fatal("comparison copied or replaced the Test Genie phase message")
	}
	if got := cmp.Phases[0].GetDescriptorB().GetEvidenceKinds()[0]; got != "application/x-future-evidence" {
		t.Fatalf("unknown evidence kind lost: %q", got)
	}
}
