package findings

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/vrooli/maturity-go/dimensions"
)

// TestLiveTestGenieAudit parses a real test-genie audit. It is gated behind
// ECOSYSTEM_MANAGER_LIVE_TESTGENIE so the default suite stays hermetic; set it
// to a real scenario name (and ensure test-genie is on PATH) to exercise the
// production runner end-to-end:
//
//	ECOSYSTEM_MANAGER_LIVE_TESTGENIE=ecosystem-manager \
//	  go test ./pkg/findings/ -run TestLiveTestGenieAudit -timeout 25m
func TestLiveTestGenieAudit(t *testing.T) {
	scenario := os.Getenv("ECOSYSTEM_MANAGER_LIVE_TESTGENIE")
	if scenario == "" {
		t.Skip("set ECOSYSTEM_MANAGER_LIVE_TESTGENIE=<scenario> to run the live audit check")
	}
	if _, err := exec.LookPath("test-genie"); err != nil {
		t.Skipf("test-genie not on PATH: %v", err)
	}

	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		root = "../../../../.." // repo root from pkg/findings
	}
	runner := &TestGenieRunner{ProjectRoot: root, Timeout: 24 * time.Minute}

	preset := os.Getenv("ECOSYSTEM_MANAGER_LIVE_PRESET")
	if preset == "" {
		preset = "quick"
	}

	audit, err := runner.Audit(context.Background(), AuditRequest{Scenario: scenario, Preset: preset})
	if err != nil {
		t.Fatalf("live audit: %v", err)
	}
	if len(audit.Phases) == 0 {
		t.Fatal("live audit returned no phases")
	}
	st := BuildState(ToFindings(audit))
	for _, f := range st.Findings {
		if !dimensions.IsValid(f.Dimension) {
			t.Errorf("live finding %s mapped to invalid dimension %q", f.ID, f.Dimension)
		}
	}
	t.Logf("live audit of %q (preset %s): %d findings, total weighted score %.0f, heaviest %v",
		scenario, preset, len(st.Findings), st.TotalScore, st.HeaviestDimensions())
}
