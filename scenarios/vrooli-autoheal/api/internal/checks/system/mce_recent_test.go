package system

import (
	"context"
	"errors"
	"testing"
	"vrooli-autoheal/internal/checks"
)

func newMCEWithMock(installed bool) (*MCERecentCheck, *checks.MockExecutor) {
	mock := checks.NewMockExecutor()
	probe := func(ctx context.Context, _ checks.CommandExecutor) bool { return installed }
	c := NewMCERecentCheck(WithMCEExecutor(mock))
	c.rasdaemonProbe = probe
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
	mock.DefaultResponse = checks.MockResponse{Output: []byte(
		"  0 Corrected errors\n  0 Uncorrected errors\n",
	)}
	r := c.Run(context.Background())
	if r.Status != checks.StatusOK {
		t.Errorf("Status = %s, want OK", r.Status)
	}
}

func TestMCEWarningOnExcessCorrected(t *testing.T) {
	c, mock := newMCEWithMock(true)
	mock.DefaultResponse = checks.MockResponse{Output: []byte(
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
	mock.DefaultResponse = checks.MockResponse{Output: []byte(
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
	mock.DefaultResponse = checks.MockResponse{Error: errors.New("permission denied")}
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
	c := NewMCERecentCheck(WithMCEExecutor(checks.NewMockExecutor()))
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
