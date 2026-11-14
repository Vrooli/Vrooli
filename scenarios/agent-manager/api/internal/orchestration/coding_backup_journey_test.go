//go:build journey

package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/fixtures"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// Both invocations execute the real native runner. The first uses a synthetic
// provider refusal and the labeled capture because real limit captures are
// operator-owned. The second uses the production credit source unchanged.
type syntheticLimitNativeRunner struct {
	runner.Runner
	baseURL string
	calls   atomic.Int32
}

func (r *syntheticLimitNativeRunner) Execute(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
	if r.calls.Add(1) != 1 {
		return r.Runner.Execute(ctx, req)
	}
	env := make(map[string]string, len(req.Environment)+2)
	for key, value := range req.Environment {
		env[key] = value
	}
	env[cliutil.ModelSourceEnvironment] = `{"id":"synthetic subscription","kind":"own_key","model":"subscription-model","base_url":"` + r.baseURL + `","secret_env":"SYNTHETIC_SUBSCRIPTION_KEY"}`
	env["SYNTHETIC_SUBSCRIPTION_KEY"] = "synthetic"
	req.Environment = env
	result, err := r.Runner.Execute(ctx, req)
	if err != nil {
		return result, err
	}
	if result == nil || result.Success {
		return nil, fmt.Errorf("synthetic subscription did not fail")
	}
	limit, ok := cliutil.DetectCodingLimit("codex", []byte(`{"type":"error","error":{"code":"usage_limit_reached"}}`), time.Now())
	if !ok {
		return nil, fmt.Errorf("synthetic limit was not recognized")
	}
	result.CodingLimit = &limit
	return result, nil
}

// The limit capture is synthetic. The restart below uses the production AM
// coordinator, SQLite, encrypted credential authority, native Codex process,
// LPBS, Gateway, wallet, and eventbus producer. The calling journey checks the
// wallet debit; this helper checks the launched session and notification.
func TestCodingBackupUnattendedJourney(t *testing.T) {
	if os.Getenv("AM_BACKUP_JOURNEY") != "1" {
		t.Skip("journey fixture not requested")
	}
	t.Log("SYNTHETIC subscription provider refusal and structured limit capture; both launches use native Codex")
	owner := os.Getenv("AM_BACKUP_OWNER")
	path := os.Getenv("AM_BACKUP_AUTHORITY_PATH")
	if owner == "" || !filepath.IsAbs(path) {
		t.Fatal("fixture owner and absolute authority path are required")
	}
	authority, err := credentialauthority.OpenEncryptedFile(path, "synthetic-fixture-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := credentialauthority.ParseIdentity(cliutil.CodingBackupIdentity)
	if err != nil {
		t.Fatal(err)
	}
	read := func(owner, harness string) (cliutil.CodingBackupRecord, error) {
		var record cliutil.CodingBackupRecord
		raw, err := authority.Resolve(identity, cliutil.CodingBackupField(owner, harness))
		if err != nil {
			return record, err
		}
		err = json.Unmarshal([]byte(raw), &record)
		return record, err
	}
	if _, err := read(owner, "codex"); err != nil {
		t.Fatalf("fixture consent unavailable: %v", err)
	}

	var notice []byte
	eventsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/events" {
			http.NotFound(w, r)
			return
		}
		notice, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer eventsServer.Close()
	repos, store, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	profile, task := fixtures.NewAgentProfile(t), fixtures.NewTask(t)
	task.ProjectRoot, task.ScopePath = t.TempDir(), "."
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatal(err)
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := fixtures.NewRun(t, task.ID, profile.ID)
	run.RunMode = domain.RunModeInPlace
	run.OwnerSubject = owner
	run.ResolvedConfig = domain.DefaultRunConfig()
	run.ResolvedConfig.RunnerType = domain.RunnerTypeCodex
	run.ResolvedConfig.Model = "subscription-model"
	run.ResolvedConfig.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeOff}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	codec, err := codecs.NewCodex()
	if err != nil {
		t.Fatal(err)
	}
	native := core.NewRunner(codec, nil, nil)
	if available, why := native.IsAvailable(ctx); !available {
		t.Fatalf("native Codex unavailable: %s", why)
	}
	capture := `{"type":"error","error":{"code":"usage_limit_reached"}}`
	parsed := codec.NewTranscriptParser().ParseTranscriptLine(run.ID, capture)
	if parsed.Err != nil {
		t.Fatalf("native transcript parser rejected synthetic limit capture: %v", parsed.Err)
	}
	limit, ok := cliutil.DetectCodingLimit("codex", []byte(capture), time.Now())
	if !ok || !limit.Exhausted(time.Now()) {
		t.Fatal("synthetic explicit limit was not recognized")
	}
	refusal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"code":"usage_limit_reached","message":"synthetic subscription limit","type":"invalid_request_error"}}`)
	}))
	defer refusal.Close()
	selected := &syntheticLimitNativeRunner{Runner: native, baseURL: refusal.URL}
	registry := runner.NewRegistry()
	if err := registry.Register(selected); err != nil {
		t.Fatal(err)
	}
	executor := NewRunExecutor(repos.Runs, registry, nil, store, run, task, profile, "Reply with the requested result.", "")
	executor.runStateRoot = t.TempDir()
	executor.sessionEnv = map[string]string{"CODEX_HOME": t.TempDir()}
	executor.codingBackupReader = read
	executor.hooks = &Orchestrator{approvalPublisher: eventbus.Client{BaseURL: eventsServer.URL, HTTPClient: eventsServer.Client()}}
	executor.Execute(ctx)
	if selected.calls.Load() != 2 || executor.execErr != nil || executor.result == nil || !executor.result.Success {
		t.Fatalf("native backup failed: launches=%d result=%+v error=%v", selected.calls.Load(), executor.result, executor.execErr)
	}
	retained, err := store.Get(ctx, run.ID, event.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range retained {
		if message, ok := messageEvent(ev); ok && strings.Contains(message.Content, "Backup journey completed.") {
			found = true
		}
	}
	if !found {
		t.Fatal("native Codex response missing from AM events")
	}
	if !strings.Contains(string(notice), "agent_manager.coding_backup.notification.v1") || !strings.Contains(string(notice), "coding-backup-switched") {
		t.Fatalf("notification contract missing: %s", notice)
	}
	t.Log("real native Codex completed through LPBS and Gateway; AM emitted eventbus credit-switch notification")
}
