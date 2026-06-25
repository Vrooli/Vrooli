package system

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
)

func newPMRuntimeWithMock(avail bool, count int, queryErr error) (*PMRuntimeHogCheck, *checks.MockExecutor) {
	mock := checks.NewMockExecutor()
	if avail {
		mock.Responses["journalctl --version"] = checks.MockResponse{Output: []byte("systemd 254\n")}
	} else {
		mock.Responses["journalctl --version"] = checks.MockResponse{Error: errors.New("not found")}
	}

	queryKey := "journalctl --no-pager -o json -k --since 1 hour ago -g pm_runtime_work .* hogged CPU -n 1000"
	if queryErr != nil {
		mock.Responses[queryKey] = checks.MockResponse{Error: queryErr}
	} else {
		var lines []string
		for i := 0; i < count; i++ {
			lines = append(lines, `{"__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":"pm_runtime_work hogged CPU"}`)
		}
		mock.Responses[queryKey] = checks.MockResponse{Output: []byte(strings.Join(lines, "\n"))}
	}
	mock.DefaultResponse = checks.MockResponse{Output: []byte("")}

	c := NewPMRuntimeHogCheck(
		WithPMRuntimeExecutor(mock),
		WithPMRuntimeReader(journal.NewReader(mock)),
	)
	return c, mock
}

func TestPMRuntimeJournalUnavailable(t *testing.T) {
	c, _ := newPMRuntimeWithMock(false, 0, nil)
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING", r.Status)
	}
}

func TestPMRuntimeOKBelowWarn(t *testing.T) {
	c, _ := newPMRuntimeWithMock(true, 10, nil)
	r := c.Run(context.Background())
	if r.Status != checks.StatusOK {
		t.Errorf("Status = %s, want OK for low count", r.Status)
	}
}

func TestPMRuntimeEmptyJournalGrepExitOneIsOK(t *testing.T) {
	c, mock := newPMRuntimeWithMock(true, 0, nil)
	queryKey := "journalctl --no-pager -o json -k --since 1 hour ago -g pm_runtime_work .* hogged CPU -n 1000"
	mock.Responses[queryKey] = checks.MockResponse{Error: errors.New("exit status 1")}

	r := c.Run(context.Background())
	if r.Status != checks.StatusOK {
		t.Errorf("Status = %s, want OK for empty grep result", r.Status)
	}
	if got := r.Details["count"]; got != 0 {
		t.Errorf("count = %v, want 0", got)
	}
	if _, ok := r.Details["error"]; ok {
		t.Errorf("error detail should be absent for empty grep result: %+v", r.Details)
	}
}

func TestPMRuntimeWarningAtThreshold(t *testing.T) {
	c, _ := newPMRuntimeWithMock(true, 75, nil)
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING at 75/hr", r.Status)
	}
}

func TestPMRuntimeCriticalAtFlood(t *testing.T) {
	c, _ := newPMRuntimeWithMock(true, 250, nil)
	r := c.Run(context.Background())
	if r.Status != checks.StatusCritical {
		t.Errorf("Status = %s, want CRITICAL at 250/hr", r.Status)
	}
}

func TestPMRuntimeWarningOnQueryError(t *testing.T) {
	c, _ := newPMRuntimeWithMock(true, 0, errors.New("permission denied"))
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING on error", r.Status)
	}
}

func TestPMRuntimeRecoveryActionsAreDiagnostic(t *testing.T) {
	c := NewPMRuntimeHogCheck()
	for _, a := range c.RecoveryActions(nil) {
		if a.Dangerous {
			t.Errorf("action %s should not be dangerous", a.ID)
		}
	}
}

func TestPMRuntimeMetadata(t *testing.T) {
	c := NewPMRuntimeHogCheck()
	if c.ID() != "system-pm-runtime-hog" {
		t.Errorf("ID = %s", c.ID())
	}
	if c.IntervalSeconds() != 300 {
		t.Errorf("Interval = %d", c.IntervalSeconds())
	}
}
