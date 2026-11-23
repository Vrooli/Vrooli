//go:build linux

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type terminalWriteBarrier struct {
	repository.RunRepository
	committed    chan uuid.UUID
	release      chan struct{}
	terminalOnce sync.Once
	releaseOnce  sync.Once
}

func (r *terminalWriteBarrier) Update(ctx context.Context, run *domain.Run) error {
	if err := r.RunRepository.Update(ctx, run); err != nil {
		return err
	}
	if run != nil && run.Status.IsTerminal() {
		r.terminalOnce.Do(func() {
			r.committed <- run.ID
			<-r.release
		})
	}
	return nil
}

func (r *terminalWriteBarrier) unblock() {
	r.releaseOnce.Do(func() { close(r.release) })
}

// TestOrdinaryRunTreasuryKeyLifecycle exercises the real CreateRun → runner →
// Unix credential broker → encrypted run-key authority path. It deliberately
// uses no Treasury service or fork; the CF1 local-fork rehearsal separately
// proves the rotated key against the mandate generation and pins the exact raw
// receipt/nonce/debit record across Treasury restart.
func TestOrdinaryRunTreasuryKeyLifecycle(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "cf1-key-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Setenv("TMPDIR", root)
	t.Setenv("VROOLI_VARIANT", "live")
	socketDir := filepath.Join(root, "vrooli-agent-manager-"+fmt.Sprint(os.Getuid()))
	if err := os.MkdirAll(socketDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(socketDir, 0o700); err != nil {
		t.Fatal(err)
	}

	const secret = "ordinary-run-session-lifecycle-integration-secret"
	const passphrase = "isolated-ordinary-run-key-authority-passphrase"
	authorityPath := filepath.Join(root, "run-keys.enc")
	openAuthority := func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(authorityPath, passphrase)
	}
	fixtures := []string{strings.Repeat("0", 63) + "1", strings.Repeat("0", 63) + "2"}
	fixtureIndex := 0
	baseKeyStore := orchestration.NewCredentialAuthorityTreasurySessionKeysWithGenerator(openAuthority, func() (string, error) {
		if fixtureIndex >= len(fixtures) {
			return "", errors.New("test key fixture exhausted")
		}
		key := fixtures[fixtureIndex]
		fixtureIndex++
		return key, nil
	})
	keyStore := baseKeyStore

	db, cleanupDB := testutil.SetupTestDB(t)
	defer cleanupDB()
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	runRepo := &terminalWriteBarrier{RunRepository: repos.Runs, committed: make(chan uuid.UUID, 1), release: make(chan struct{})}
	t.Cleanup(runRepo.unblock)
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	type lifecycleResult struct {
		initialLaunch string
		initialBroker string
		rotated       string
		replay        string
		current       string
		rotatedLaunch string
		foreignDenied bool
		identityToken string
		err           error
	}
	resultCh := make(chan lifecycleResult, 1)
	mock.ExecuteFunc = func(runCtx context.Context, request runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		result := lifecycleResult{}
		result.identityToken = request.Environment[cliutil.EnvIdentityToken]
		fail := func(err error) (*runner.ExecuteResult, error) {
			result.err = err
			resultCh <- result
			return nil, err
		}
		if request.Environment[cliutil.EnvIdentityToken] == "" || request.Environment["VROOLI_TREASURY_SESSION_PRIVATE_KEY"] != "" || request.PrepareLaunch == nil {
			return fail(errors.New("ordinary run did not keep its Treasury key behind the launch boundary"))
		}
		if request.Transcript == nil || request.Transcript.OnProcessStart == nil {
			return fail(errors.New("ordinary run has no process identity callback for the credential broker"))
		}
		if err := request.Transcript.OnProcessStart(os.Getpid(), os.Getpid()); err != nil {
			return fail(fmt.Errorf("bind runner identity to the current test process: %w", err))
		}
		launch := func() (string, error) {
			child := &runner.LaunchRequest{Env: []string{"SAFE=kept", "VROOLI_TREASURY_SESSION_PRIVATE_KEY=caller-value"}}
			release, err := request.PrepareLaunch(runCtx, child)
			if err != nil {
				return "", err
			}
			defer func() {
				if release != nil {
					release()
				}
			}()
			var found string
			count := 0
			for _, entry := range child.Env {
				if strings.HasPrefix(entry, "VROOLI_TREASURY_SESSION_PRIVATE_KEY=") {
					found = strings.TrimPrefix(entry, "VROOLI_TREASURY_SESSION_PRIVATE_KEY=")
					count++
				}
			}
			if count != 1 || found == "caller-value" || !containsExact(child.Env, "SAFE=kept") {
				return "", fmt.Errorf("launch did not receive exactly one protected session key: %#v", child.Env)
			}
			return found, nil
		}
		var err error
		result.initialLaunch, err = launch()
		if err != nil {
			return fail(fmt.Errorf("prepare first child launch: %w", err))
		}
		result.initialBroker, err = cliutil.ResolveTreasurySessionKey(runCtx, request.Environment[cliutil.EnvIdentityToken], "")
		if err != nil || result.initialBroker != result.initialLaunch {
			return fail(fmt.Errorf("ordinary broker key did not match launch key: equal=%t err=%v", result.initialBroker == result.initialLaunch, err))
		}

		foreignToken, err := identity.GenerateToken(&identity.Claims{
			RunID: uuid.New(), TaskID: request.Task.ID, Scopes: []string{"treasury:write"},
			IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}, []byte(secret))
		if err != nil {
			return fail(fmt.Errorf("create foreign-run test identity: %w", err))
		}
		if _, err := cliutil.ResolveTreasurySessionKey(runCtx, foreignToken, ""); err == nil {
			return fail(errors.New("credential broker returned this run's Treasury key to a foreign run identity"))
		}
		result.foreignDenied = true

		const rotationRequest = "mandate-generation-2-session-key"
		result.rotated, err = cliutil.ResolveTreasurySessionKey(runCtx, request.Environment[cliutil.EnvIdentityToken], rotationRequest)
		if err != nil || result.rotated == "" || result.rotated == result.initialLaunch {
			return fail(fmt.Errorf("ordinary run did not receive a fresh broker key for the rotated mandate: changed=%t err=%v", result.rotated != result.initialLaunch, err))
		}
		result.replay, err = cliutil.ResolveTreasurySessionKey(runCtx, request.Environment[cliutil.EnvIdentityToken], rotationRequest)
		if err != nil || result.replay != result.rotated {
			return fail(fmt.Errorf("rotation request replay did not return the same active key: equal=%t err=%v", result.replay == result.rotated, err))
		}
		result.current, err = cliutil.ResolveTreasurySessionKey(runCtx, request.Environment[cliutil.EnvIdentityToken], "")
		if err != nil || result.current != result.rotated {
			return fail(fmt.Errorf("broker did not rebind this active run to the rotated key: equal=%t err=%v", result.current == result.rotated, err))
		}
		result.rotatedLaunch, err = launch()
		if err != nil || result.rotatedLaunch != result.rotated {
			return fail(fmt.Errorf("next ordinary child launch did not receive the rotated key: equal=%t err=%v", result.rotatedLaunch == result.rotated, err))
		}
		resultCh <- result
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	service := orchestration.New(repos.Profiles, repos.Tasks, runRepo,
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1, RequireSandboxByDefault: false}),
		orchestration.WithEvents(events), orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints), orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithRunStateRoot(filepath.Join(root, "run-state")), orchestration.WithIdentitySecret([]byte(secret)),
		orchestration.WithTreasurySessionKeyStore(keyStore), orchestration.WithOwnerIdentity(cf1TreasuryOperatorVerifier{}),
		orchestration.WithSandbox(mocks.NewFakeSandboxProvider()),
		newTestRolePolicyOption(t),
	)
	stopBroker, err := service.StartAuthorizationBroker()
	if err != nil {
		t.Fatalf("start isolated Agent Manager credential broker: %v", err)
	}
	t.Cleanup(func() { _ = stopBroker() })

	profile, err := service.CreateProfile(context.Background(), &domain.AgentProfile{
		ID: uuid.New(), Name: "ordinary-run-key-lifecycle", RoleRef: "code.default", DeclaredScopes: []string{"treasury:read", "treasury:write"},
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff},
	})
	if err != nil {
		t.Fatalf("create Treasury-scoped profile: %v", err)
	}
	projectRoot := filepath.Join(root, "project")
	if err := os.MkdirAll(projectRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(context.Background(), &domain.Task{
		ID: uuid.New(), Title: "ordinary run key lifecycle", Description: "exercise session-key rotation through the real run broker",
		ScopePath: projectRoot, ProjectRoot: projectRoot, Status: domain.TaskStatusQueued,
	})
	if err != nil {
		t.Fatalf("create integration task: %v", err)
	}
	run, err := service.CreateRun(context.Background(), orchestration.CreateRunRequest{
		OwnerToken: cf1TreasuryOperator, ExpectedOwnerSubject: "cf1-local-operator", TaskID: task.ID, AgentProfileID: &profile.ID,
		Prompt: "Exercise the ordinary-run Treasury session-key lifecycle.", RequestedScopes: []string{"treasury:read", "treasury:write"},
		RunMode: func() *domain.RunMode { mode := domain.RunModeInPlace; return &mode }(),
	})
	if err != nil {
		t.Fatalf("create ordinary Treasury run: %v", err)
	}
	var completed lifecycleResult
	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatal(result.err)
		}
		completed = result
		if result.initialLaunch != fixtures[0] || result.initialBroker != fixtures[0] || result.rotated != fixtures[1] || result.replay != fixtures[1] || result.current != fixtures[1] || result.rotatedLaunch != fixtures[1] || !result.foreignDenied {
			t.Fatalf("ordinary run did not preserve expected key lifecycle: %+v", result)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("ordinary run did not exercise its broker lifecycle")
	}
	select {
	case committedRunID := <-runRepo.committed:
		if committedRunID != run.ID {
			t.Fatalf("terminal write barrier reached unexpected run %s, want %s", committedRunID, run.ID)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("terminal status was not persisted through the run repository")
	}
	visibleRun, err := service.GetRun(context.Background(), run.ID)
	if err != nil || !visibleRun.Status.IsTerminal() {
		t.Fatalf("terminal status was not durably visible before key revocation completed: run=%+v err=%v", visibleRun, err)
	}
	keyRequestDone := make(chan error, 1)
	go func() {
		_, release, err := baseKeyStore.Acquire(context.Background(), run.ID)
		if release != nil {
			release()
		}
		keyRequestDone <- err
	}()
	select {
	case err := <-keyRequestDone:
		t.Fatalf("key request escaped the terminal write gate while revocation was pending: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	runRepo.unblock()
	if err := <-keyRequestDone; err == nil {
		t.Fatal("queued key request returned the run key after terminal revocation")
	}
	waitForCompletedRun(t, context.Background(), service, run.ID, 30*time.Second)
	if completed.identityToken == "" {
		t.Fatal("ordinary run did not retain its test identity token for terminal refusal verification")
	}
	if _, err := cliutil.ResolveTreasurySessionKey(context.Background(), completed.identityToken, ""); err == nil {
		t.Fatal("credential broker returned a run key after terminal revocation")
	}
	identityID, err := credentialauthority.ParseIdentity("vrooli/agent-manager-run-" + run.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	authority, err := openAuthority()
	if err != nil {
		t.Fatalf("reopen isolated encrypted run-key authority: %v", err)
	}
	stored, err := authority.Resolve(identityID, "treasury-agent-session-key")
	if err != nil {
		t.Fatalf("read terminal run-key tombstone: %v", err)
	}
	var state struct {
		Version int    `json:"version"`
		Key     string `json:"key"`
		Revoked bool   `json:"revoked"`
	}
	if err := json.Unmarshal([]byte(stored), &state); err != nil || state.Version != 1 || !state.Revoked || state.Key != "" || strings.Contains(stored, fixtures[0]) || strings.Contains(stored, fixtures[1]) {
		t.Fatalf("terminal authority did not retain a keyless revocation tombstone: state=%+v raw=%q err=%v", state, stored, err)
	}
}

func containsExact(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
