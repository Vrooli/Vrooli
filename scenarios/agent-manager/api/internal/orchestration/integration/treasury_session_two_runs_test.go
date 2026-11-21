//go:build linux

package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const treasuryKeyChildEnv = "CF1_TREASURY_SESSION_KEY_CHILD"
const treasuryKeyBrokerSocketEnv = "CF1_TREASURY_BROKER_SOCKET"

type treasuryKeyChildCommand struct {
	ForeignIdentityToken string `json:"foreign_identity_token"`
}

type treasuryKeyChildResult struct {
	RunID          string `json:"run_id"`
	LaunchKey      string `json:"launch_key"`
	InitialKey     string `json:"initial_key"`
	ForeignDenied  bool   `json:"foreign_denied"`
	RotatedKey     string `json:"rotated_key"`
	RotationReplay string `json:"rotation_replay"`
	CurrentKey     string `json:"current_key"`
}

type treasuryKeyChildStart struct {
	RunID uuid.UUID
	Token string
	Stdin io.WriteCloser
}

// TestCF1TreasurySessionKeyBrokerChild runs in a separate OS process for each
// ordinary run. The parent keeps both helpers alive until it can give each the
// other's real run identity token, making the cross-run refusal discriminating.
func TestCF1TreasurySessionKeyBrokerChild(t *testing.T) {
	if os.Getenv(treasuryKeyChildEnv) != "1" {
		return
	}
	var command treasuryKeyChildCommand
	if err := json.NewDecoder(bufio.NewReader(os.Stdin)).Decode(&command); err != nil {
		t.Fatalf("read cross-run identity test command: %v", err)
	}
	token := strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken))
	runID := strings.TrimSpace(os.Getenv("CF1_TREASURY_RUN_ID"))
	launchKey := strings.TrimSpace(os.Getenv("VROOLI_TREASURY_SESSION_PRIVATE_KEY"))
	if token == "" || runID == "" || launchKey == "" || command.ForeignIdentityToken == "" {
		t.Fatal("ordinary-run child identity command is incomplete")
	}
	if brokerSocket := strings.TrimSpace(os.Getenv(treasuryKeyBrokerSocketEnv)); brokerSocket != "" {
		childSocket := cliutil.AgentCredentialSocket()
		if err := os.MkdirAll(filepath.Dir(childSocket), 0o700); err != nil {
			t.Fatalf("create isolated child broker socket directory: %v", err)
		}
		if err := os.Symlink(brokerSocket, childSocket); err != nil {
			t.Fatalf("link isolated child broker socket: %v", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	initialKey, err := cliutil.ResolveTreasurySessionKey(ctx, token, "")
	if err != nil || initialKey != launchKey {
		t.Fatalf("child broker key does not match its protected launch key: equal=%t err=%v", initialKey == launchKey, err)
	}
	if _, err := cliutil.ResolveTreasurySessionKey(ctx, command.ForeignIdentityToken, ""); err == nil {
		t.Fatal("Agent Manager broker returned this run's Treasury key to the other active run")
	}
	rotationRequest := "cf1-two-run-mandate-rotation-" + runID
	rotated, err := cliutil.ResolveTreasurySessionKey(ctx, token, rotationRequest)
	if err != nil || rotated == "" || rotated == initialKey {
		t.Fatalf("run did not receive a replacement session key: changed=%t err=%v", rotated != initialKey, err)
	}
	replay, err := cliutil.ResolveTreasurySessionKey(ctx, token, rotationRequest)
	if err != nil || replay != rotated {
		t.Fatalf("rotation replay changed this run's key: equal=%t err=%v", replay == rotated, err)
	}
	current, err := cliutil.ResolveTreasurySessionKey(ctx, token, "")
	if err != nil || current != rotated {
		t.Fatalf("ordinary run did not rebind to the rotated key: equal=%t err=%v", current == rotated, err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(treasuryKeyChildResult{
		RunID: runID, LaunchKey: launchKey, InitialKey: initialKey, ForeignDenied: true,
		RotatedKey: rotated, RotationReplay: replay, CurrentKey: current,
	}); err != nil {
		t.Fatalf("write run-key continuity result: %v", err)
	}
}

func TestTwoOrdinaryRunsHaveDistinctTreasurySessionKeysAndRejectCrossRunIdentity(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "cf1-two-key-")
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

	const secret = "two-independent-run-session-key-integration-secret"
	const passphrase = "two-independent-run-test-authority-passphrase"
	authorityPath := filepath.Join(root, "run-keys.enc")
	openAuthority := func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(authorityPath, passphrase)
	}
	fixtures := []string{
		strings.Repeat("0", 63) + "1", strings.Repeat("0", 63) + "2",
		strings.Repeat("0", 63) + "3", strings.Repeat("0", 63) + "4",
	}
	fixtureIndex := 0
	var fixtureMu sync.Mutex
	keyStore := orchestration.NewCredentialAuthorityTreasurySessionKeysWithGenerator(openAuthority, func() (string, error) {
		fixtureMu.Lock()
		defer fixtureMu.Unlock()
		if fixtureIndex >= len(fixtures) {
			return "", errors.New("two-run session-key fixture exhausted")
		}
		key := fixtures[fixtureIndex]
		fixtureIndex++
		return key, nil
	})
	db, cleanupDB := testutil.SetupTestDB(t)
	defer cleanupDB()
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	started := make(chan treasuryKeyChildStart, 2)
	completed := make(chan treasuryKeyChildResult, 2)
	runErrors := make(chan error, 2)
	mock.ExecuteFunc = func(runCtx context.Context, request runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		if request.Environment[cliutil.EnvIdentityToken] == "" || request.Environment["VROOLI_TREASURY_SESSION_PRIVATE_KEY"] != "" || request.PrepareLaunch == nil {
			err := errors.New("ordinary run exposed or omitted its launch-scoped Treasury key")
			runErrors <- err
			return nil, err
		}
		if request.Transcript == nil || request.Transcript.OnProcessStart == nil {
			err := errors.New("ordinary run omitted its child process identity callback")
			runErrors <- err
			return nil, err
		}
		childEnv := append([]string(nil), os.Environ()...)
		setEnv := func(key, value string) {
			prefix := key + "="
			filtered := childEnv[:0]
			for _, entry := range childEnv {
				if !strings.HasPrefix(entry, prefix) {
					filtered = append(filtered, entry)
				}
			}
			childEnv = append(filtered, prefix+value)
		}
		for key, value := range request.Environment {
			setEnv(key, value)
		}
		setEnv("TMPDIR", root)
		setEnv(treasuryKeyChildEnv, "1")
		setEnv(treasuryKeyBrokerSocketEnv, cliutil.AgentCredentialSocket())
		setEnv("CF1_TREASURY_RUN_ID", request.RunID.String())
		launch := &runner.LaunchRequest{Env: childEnv}
		release, err := request.PrepareLaunch(runCtx, launch)
		if err != nil {
			runErrors <- fmt.Errorf("prepare actual ordinary-run child launch: %w", err)
			return nil, err
		}
		cmd := exec.CommandContext(runCtx, os.Args[0], "-test.run=^TestCF1TreasurySessionKeyBrokerChild$")
		cmd.Env = launch.Env
		stdin, err := cmd.StdinPipe()
		if err != nil {
			if release != nil {
				release()
			}
			runErrors <- err
			return nil, err
		}
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		if err := cmd.Start(); err != nil {
			if release != nil {
				release()
			}
			runErrors <- err
			return nil, err
		}
		if err := request.Transcript.OnProcessStart(cmd.Process.Pid, cmd.Process.Pid); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			if release != nil {
				release()
			}
			runErrors <- fmt.Errorf("bind child to its ordinary run: %w", err)
			return nil, err
		}
		if release != nil {
			release()
		}
		started <- treasuryKeyChildStart{RunID: request.RunID, Token: request.Environment[cliutil.EnvIdentityToken], Stdin: stdin}
		if err := cmd.Wait(); err != nil {
			err = fmt.Errorf("ordinary-run broker child failed: %w: %s", err, strings.TrimSpace(output.String()))
			runErrors <- err
			return nil, err
		}
		var result treasuryKeyChildResult
		if err := json.NewDecoder(bytes.NewReader(output.Bytes())).Decode(&result); err != nil {
			err = fmt.Errorf("decode ordinary-run child result: %w: %s", err, strings.TrimSpace(output.String()))
			runErrors <- err
			return nil, err
		}
		completed <- result
		return &runner.ExecuteResult{}, nil
	}
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	service := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: 2 * time.Minute, MaxConcurrentRuns: 2, RequireSandboxByDefault: false}),
		orchestration.WithEvents(events), orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints), orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithRunStateRoot(filepath.Join(root, "run-state")), orchestration.WithIdentitySecret([]byte(secret)),
		orchestration.WithTreasurySessionKeyStore(keyStore), orchestration.WithOwnerIdentity(cf1TreasuryOperatorVerifier{}),
		orchestration.WithSandbox(mocks.NewFakeSandboxProvider()), newTestRolePolicyOption(t),
	)
	stopBroker, err := service.StartAuthorizationBroker()
	if err != nil {
		t.Fatalf("start isolated Agent Manager credential broker: %v", err)
	}
	t.Cleanup(func() { _ = stopBroker() })
	createProfile := func(label string) *domain.AgentProfile {
		t.Helper()
		profile, err := service.CreateProfile(context.Background(), &domain.AgentProfile{
			ID: uuid.New(), Name: label + "-treasury-identity", RoleRef: "code.default",
			DeclaredScopes: []string{"treasury:read", "treasury:write"}, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff},
		})
		if err != nil {
			t.Fatalf("create %s run profile: %v", label, err)
		}
		return profile
	}
	profiles := []*domain.AgentProfile{createProfile("agent-one"), createProfile("agent-two")}
	createRun := func(label string, profile *domain.AgentProfile) *domain.Run {
		t.Helper()
		projectRoot := filepath.Join(root, label)
		if err := os.MkdirAll(projectRoot, 0o700); err != nil {
			t.Fatal(err)
		}
		task, err := service.CreateTask(context.Background(), &domain.Task{ID: uuid.New(), Title: label,
			Description: "independent Treasury run key identity", ScopePath: projectRoot, ProjectRoot: projectRoot, Status: domain.TaskStatusQueued})
		if err != nil {
			t.Fatalf("create %s task: %v", label, err)
		}
		run, err := service.CreateRun(context.Background(), orchestration.CreateRunRequest{
			OwnerToken: cf1TreasuryOperator, ExpectedOwnerSubject: "cf1-local-operator", TaskID: task.ID, AgentProfileID: &profile.ID,
			Prompt: "Use only this ordinary run's Treasury session key.", RequestedScopes: []string{"treasury:read", "treasury:write"},
		})
		if err != nil {
			t.Fatalf("create %s ordinary run: %v", label, err)
		}
		return run
	}
	runs := []*domain.Run{createRun("agent-one", profiles[0]), createRun("agent-two", profiles[1])}
	starts := make(map[uuid.UUID]treasuryKeyChildStart, 2)
	for range runs {
		select {
		case child := <-started:
			starts[child.RunID] = child
		case err := <-runErrors:
			t.Fatal(err)
		case <-time.After(30 * time.Second):
			t.Fatal("both independent ordinary-run child processes did not start")
		}
	}
	if len(starts) != 2 || starts[runs[0].ID].Token == "" || starts[runs[1].ID].Token == "" || starts[runs[0].ID].Token == starts[runs[1].ID].Token {
		t.Fatalf("the two active runs did not have distinct signed identities: run_ids=%s,%s", runs[0].ID, runs[1].ID)
	}
	for index, run := range runs {
		foreign := runs[1-index]
		if err := json.NewEncoder(starts[run.ID].Stdin).Encode(treasuryKeyChildCommand{ForeignIdentityToken: starts[foreign.ID].Token}); err != nil {
			t.Fatalf("send foreign run identity to child %s: %v", run.ID, err)
		}
		if err := starts[run.ID].Stdin.Close(); err != nil {
			t.Fatalf("close child %s input: %v", run.ID, err)
		}
	}
	results := make(map[string]treasuryKeyChildResult, 2)
	for range runs {
		select {
		case result := <-completed:
			results[result.RunID] = result
		case err := <-runErrors:
			t.Fatal(err)
		case <-time.After(30 * time.Second):
			t.Fatal("both ordinary-run key identity checks did not finish")
		}
	}
	// The mock runner sends completed evidence after the actual helper process
	// exits; guard against a helper returning output for the wrong run.
	for _, run := range runs {
		result := results[run.ID.String()]
		if result.RunID != run.ID.String() || !result.ForeignDenied || result.LaunchKey == "" || result.LaunchKey != result.InitialKey || result.RotatedKey == result.InitialKey || result.RotationReplay != result.RotatedKey || result.CurrentKey != result.RotatedKey {
			t.Fatalf("ordinary run %s did not preserve its isolated identity/rotation: %+v", run.ID, result)
		}
		for _, other := range runs {
			if other.ID == run.ID {
				continue
			}
			otherResult := results[other.ID.String()]
			if result.InitialKey == otherResult.InitialKey {
				t.Fatalf("independent runs shared a Treasury session identity: %s and %s", run.ID, other.ID)
			}
			if result.RotatedKey == otherResult.RotatedKey || result.RotatedKey == otherResult.InitialKey || otherResult.RotatedKey == result.InitialKey {
				t.Fatalf("rotation reused another run's Treasury session identity: %s and %s", run.ID, other.ID)
			}
		}
	}
	for _, run := range runs {
		final, err := waitForTerminal(t, context.Background(), service, run.ID, 30*time.Second)
		if err != nil || final.Status != domain.RunStatusComplete {
			t.Fatalf("ordinary run %s did not complete: run=%+v err=%v", run.ID, final, err)
		}
		if _, err := keyStore.ResolveOrMint(context.Background(), run.ID); err == nil {
			t.Fatalf("terminal ordinary run %s retained its Treasury key", run.ID)
		}
	}
}
