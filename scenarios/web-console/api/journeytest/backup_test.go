package journeytest

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// SYNTHETIC CAPTURES: constructed protocol envelopes, not observations from a
// paid provider at a real limit. OPERATING-PROCEDURE §10 permits this check to
// remain synthetic until the operator supplies real captures.
func TestJ6_LimitDetectedPerHarness(t *testing.T) {
	t.Log("SYNTHETIC provider captures; real limit captures remain operator-owned")
	now := time.Now()
	for _, tc := range []struct{ harness, record string }{
		{"codex", `{"type":"error","error":{"code":"usage_limit_reached"}}`},
		{"opencode", `{"type":"error","error":{"data":{"responseBody":"{\"error\":{\"code\":\"insufficient_quota\"}}"}}}`},
		{"claude", `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","utilization":1}}`},
	} {
		t.Run(tc.harness, func(t *testing.T) {
			limit, ok := cliutil.DetectCodingLimit(tc.harness, []byte(tc.record), now)
			if !ok || !limit.Exhausted(now) {
				t.Fatalf("explicit limit was not detected: %+v", limit)
			}
			if limit.Exhausted(now.Add(6 * time.Minute)) {
				t.Fatal("stale event triggered a switch")
			}
			for _, negative := range []string{`{"type":"error","error":{"code":"rate_limit_error","status":429}}`, `{"type":"message","text":"usage_limit_reached"}`, `429 Too Many Requests`, `{"type":"tool_result","error":{"code":"usage_limit_reached"}}`} {
				if _, ok := cliutil.DetectCodingLimit(tc.harness, []byte(negative), now); ok {
					t.Fatalf("non-quota signal triggered: %s", negative)
				}
			}
		})
	}
	for _, h := range []string{"grok", "antigravity", "unknown"} {
		if _, ok := cliutil.DetectCodingLimit(h, []byte(`{"type":"error","error":{"code":"usage_limit_reached"}}`), now); ok {
			t.Fatal("unknown codec inferred a limit")
		}
	}
	reset := cliutil.CodingLimit{Harness: "codex", Kind: "quota_exhausted", ObservedAt: now, ResetsAt: now.Add(-time.Second)}
	if reset.Exhausted(now) {
		t.Fatal("reset quota retained exhaustion")
	}
}

func runBackupJourney(t *testing.T) {
	t.Helper()
	cmd := exec.Command("go", "test", "-tags=journey", ".", "-run", "^TestCodingBackupProductionJourney$", "-count=1", "-v", "-timeout", "15m")
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), "GOWORK=off", "CODING_BACKUP_JOURNEY="+t.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("production Web Console journey: %v\n%s", err, output)
	}
	t.Log(string(output))
}
func TestJ6_OneTapRestartOnCredits(t *testing.T)             { runBackupJourney(t) }
func TestJ6_NewSessionsSwitchWithConsentAndCap(t *testing.T) { runBackupJourney(t) }
func TestJ6_NoCreditsSpentWithoutConsent(t *testing.T)       { runBackupJourney(t) }
func TestJ6_SwitchBackWorks(t *testing.T)                    { runBackupJourney(t) }
func TestJ6_UnattendedRunSwitchesAndNotifies(t *testing.T)   { runBackupJourney(t) }
