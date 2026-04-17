package preflight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"scenario-to-cloud/ssh"
	"strings"
	"testing"
)

type fakeCleanupRunner struct {
	dfCalls int
}

func (f *fakeCleanupRunner) Run(_ context.Context, _ ssh.Config, command string, _ ssh.RunOptions) (ssh.Result, error) {
	switch {
	case strings.Contains(command, "awk '{print $4}'"):
		f.dfCalls++
		if f.dfCalls == 1 {
			return ssh.Result{Stdout: "1000\n", ExitCode: 0}, nil
		}
		return ssh.Result{Stdout: "1200\n", ExitCode: 0}, nil
	case strings.Contains(command, "apt-get clean"):
		return ssh.Result{Stdout: "apt cleaned", ExitCode: 0}, nil
	case strings.Contains(command, "journalctl --vacuum-size=100M"):
		return ssh.Result{Stderr: "journal lock busy", ExitCode: 100}, fmt.Errorf("command failed")
	default:
		return ssh.Result{ExitCode: 0}, nil
	}
}

func TestHandleDiskCleanupIncludesPerActionResults(t *testing.T) {
	runner := &fakeCleanupRunner{}
	handler := HandleDiskCleanup(runner)

	body := map[string]interface{}{
		"host":     "203.0.113.10",
		"key_path": "/tmp/id_rsa",
		"actions":  []string{"apt_clean", "journal_vacuum"},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/preflight/disk/cleanup", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp DiskCleanupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.OK {
		t.Fatal("expected overall cleanup to fail due journal_vacuum failure")
	}
	if len(resp.ActionsRun) != 1 || resp.ActionsRun[0] != "apt_clean" {
		t.Fatalf("unexpected actions_run: %#v", resp.ActionsRun)
	}
	if len(resp.ActionsFailed) != 1 || resp.ActionsFailed[0] != "journal_vacuum" {
		t.Fatalf("unexpected actions_failed: %#v", resp.ActionsFailed)
	}
	if len(resp.ActionResults) != 2 {
		t.Fatalf("expected 2 action_results, got %d", len(resp.ActionResults))
	}

	var foundFailure bool
	for _, result := range resp.ActionResults {
		if result.Action != "journal_vacuum" {
			continue
		}
		foundFailure = true
		if result.OK {
			t.Fatal("journal_vacuum should be marked failed")
		}
		if result.ExitCode != 100 {
			t.Fatalf("expected exit_code=100, got %d", result.ExitCode)
		}
		if strings.TrimSpace(result.Summary) == "" {
			t.Fatal("expected non-empty summary for failed action")
		}
		if strings.TrimSpace(result.Hint) == "" {
			t.Fatal("expected non-empty hint for failed action")
		}
	}
	if !foundFailure {
		t.Fatal("missing action_results entry for journal_vacuum")
	}
}
