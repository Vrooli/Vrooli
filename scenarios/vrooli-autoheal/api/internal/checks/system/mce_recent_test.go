package system

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func newMCEWithMock(installed bool) (*MCERecentCheck, *testutil.MockExecutor) {
	mock := testutil.NewMockExecutor()
	probe := func(ctx context.Context, _ checks.CommandExecutor) (string, bool) {
		if !installed {
			return "", false
		}
		return "ras-mc-ctl", true
	}
	c := NewMCERecentCheck(WithMCEExecutor(mock))
	c.rasCommandProbe = probe
	c.serviceStateProbe = func(ctx context.Context, exec checks.CommandExecutor) (string, bool) {
		return "active", true
	}
	return c, mock
}

func TestMCEWarningWhenRasdaemonMissing(t *testing.T) {
	c, _ := newMCEWithMock(false)
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING", r.Status)
	}
	if r.Details["rasdaemonAvailable"] != false {
		t.Errorf("rasdaemonAvailable = %v", r.Details["rasdaemonAvailable"])
	}
}

func TestMCEOKWhenNoErrors(t *testing.T) {
	c, mock := newMCEWithMock(true)
	mock.DefaultResponse = testutil.MockResponse{Output: []byte(
		"  0 Corrected errors\n  0 Uncorrected errors\n",
	)}
	r := c.Run(context.Background())
	if r.Status != checks.StatusOK {
		t.Errorf("Status = %s, want OK", r.Status)
	}
}

func TestMCEWarningOnExcessCorrected(t *testing.T) {
	c, mock := newMCEWithMock(true)
	mock.DefaultResponse = testutil.MockResponse{Output: []byte(
		"  17 Corrected errors\n  0 Uncorrected errors\n",
	)}
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING for >5 corrected", r.Status)
	}
	if r.Details["correctedErrors"] != 17 {
		t.Errorf("correctedErrors = %v", r.Details["correctedErrors"])
	}
}

func TestMCECriticalOnUncorrected(t *testing.T) {
	c, mock := newMCEWithMock(true)
	mock.DefaultResponse = testutil.MockResponse{Output: []byte(
		"  0 Corrected errors\n  3 Uncorrected errors\n",
	)}
	r := c.Run(context.Background())
	if r.Status != checks.StatusCritical {
		t.Errorf("Status = %s, want CRITICAL", r.Status)
	}
	if r.Details["uncorrectedErrors"] != 3 {
		t.Errorf("uncorrectedErrors = %v", r.Details["uncorrectedErrors"])
	}
}

func TestMCEWarningOnCommandError(t *testing.T) {
	c, mock := newMCEWithMock(true)
	mock.DefaultResponse = testutil.MockResponse{Error: errors.New("permission denied")}
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING on error", r.Status)
	}
}

func TestMCERecoveryActionsAreDiagnostic(t *testing.T) {
	c := NewMCERecentCheck()
	actions := c.RecoveryActions(nil)
	if len(actions) != 4 {
		t.Errorf("got %d actions, want 4", len(actions))
	}
	for _, a := range actions {
		if a.Dangerous {
			t.Errorf("action %s should not be dangerous (forensic preservation)", a.ID)
		}
	}
}

func TestMCEExecuteUnknownAction(t *testing.T) {
	c := NewMCERecentCheck(WithMCEExecutor(testutil.NewMockExecutor()))
	r := c.ExecuteAction(context.Background(), "destroy-data")
	if r.Success {
		t.Error("unknown action should fail")
	}
}

func TestMCEMetadata(t *testing.T) {
	c := NewMCERecentCheck()
	if c.ID() != "system-mce-recent" {
		t.Errorf("ID = %s", c.ID())
	}
	if c.IntervalSeconds() != 300 {
		t.Errorf("Interval = %d", c.IntervalSeconds())
	}
}
