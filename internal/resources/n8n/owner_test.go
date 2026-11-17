package n8n

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/vrooli/internal/resources/n8n/virtualapi"
)

const testSandboxID = "5e33a6f0-b1bf-4f5d-9315-95ba60d9e025"

func TestWaitReadyReportsLastProbeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "starting", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	owner := &Owner{HTTPClient: server.Client(), PollEvery: time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := owner.waitReady(ctx, server.URL)
	if err == nil || !strings.Contains(err.Error(), "last probe: n8n readiness returned HTTP 503") {
		t.Fatalf("expected final readiness probe cause in error, got %v", err)
	}
}

func TestRuntimeFromContainerUsesOwnerCLIWhenInternalBridgeOmitsPublishedPort(t *testing.T) {
	_, networkName := objectNames(testSandboxID)
	output := []byte(`[ {"Id":"owned-n8n","Config":{"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"container"}},"State":{"Running":true},"NetworkSettings":{"Ports":{},"Networks":{"` + networkName + `":{"Aliases":["` + n8nAlias(testSandboxID) + `"]}}},"HostConfig":{"PortBindings":{"5678/tcp":[{"HostIp":"127.0.0.1","HostPort":"15678"}]}}} ]`)
	container, found, err := parseContainerInspect(output)
	if err != nil || !found {
		t.Fatalf("parse container inspection: found=%t err=%v", found, err)
	}
	owner := &Owner{}
	runtime, err := owner.runtimeFromContainer(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), container)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.AccessMode != "owner-cli" || runtime.Endpoint != "" || runtime.Alias != n8nAlias(testSandboxID) {
		t.Fatalf("runtime = %#v, want private owner-cli access and no host endpoint", runtime)
	}
}

func TestStatusProbesOwnerCLIRuntimeInsideContainer(t *testing.T) {
	containerName, networkName := objectNames(testSandboxID)
	runtimeID := runtimeIDFor(testSandboxID)
	runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch joined {
		case "info --format {{.DockerRootDir}}":
			return []byte("/var/lib/docker"), nil
		case "container inspect " + containerName:
			return []byte(`[ {"Id":"owned-n8n","Config":{"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"` + runtimeID + `","com.vrooli.object-kind":"container"}},"State":{"Running":true},"NetworkSettings":{"Ports":{},"Networks":{"` + networkName + `":{"Aliases":["` + n8nAlias(testSandboxID) + `"]}}},"HostConfig":{"PortBindings":{"5678/tcp":[{"HostIp":"127.0.0.1","HostPort":"15678"}]}}} ]`), nil
		case "exec --user node " + containerName + " node -e " + n8nHealthProbeProgram:
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", joined)
		}
	}}
	owner := &Owner{Runner: runner}
	runtime, err := owner.Status(context.Background(), testSandboxID)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Status != "running" || runtime.AccessMode != "owner-cli" || runtime.Endpoint != "" {
		t.Fatalf("runtime = %#v, want running private owner-cli mode", runtime)
	}
}

func TestRuntimeFromContainerRejectsNonLoopbackConfiguredPortWhenMappingIsAbsent(t *testing.T) {
	_, networkName := objectNames(testSandboxID)
	output := []byte(`[ {"Id":"owned-n8n","Config":{"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"container"}},"State":{"Running":true},"NetworkSettings":{"Ports":{},"Networks":{"` + networkName + `":{"Aliases":["` + n8nAlias(testSandboxID) + `"]}}},"HostConfig":{"PortBindings":{"5678/tcp":[{"HostIp":"0.0.0.0","HostPort":"15678"}]}}} ]`)
	container, found, err := parseContainerInspect(output)
	if err != nil || !found {
		t.Fatalf("parse container inspection: found=%t err=%v", found, err)
	}
	owner := &Owner{}
	if _, err := owner.runtimeFromContainer(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), container); err == nil || !strings.Contains(err.Error(), "not bound to loopback") {
		t.Fatalf("runtime error = %v, want non-loopback configuration rejection", err)
	}
}

func TestProbeRuntimeReadyUsesOwnerExecWhenNoHostEndpointExists(t *testing.T) {
	runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) != 7 || strings.Join(args[:6], " ") != "exec --user node task-container node -e" || args[6] != n8nHealthProbeProgram {
			return nil, fmt.Errorf("unexpected health probe: %s", strings.Join(args, " "))
		}
		return nil, nil
	}}
	owner := &Owner{Runner: runner}
	if err := owner.probeRuntimeReady(context.Background(), "task-container", Runtime{AccessMode: "owner-cli"}); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("health probe calls = %d, want 1", len(runner.calls))
	}
}

type fakeRunner struct {
	call      func(context.Context, ...string) ([]byte, error)
	inputCall func(context.Context, []byte, ...string) ([]byte, error)
	calls     [][]string
	stdin     [][]byte
}

func (f *fakeRunner) RunQuiet(ctx context.Context, args ...string) ([]byte, error) {
	return f.Run(ctx, args...)
}

func (f *fakeRunner) RunInputQuiet(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	f.stdin = append(f.stdin, append([]byte(nil), input...))
	f.calls = append(f.calls, append([]string(nil), args...))
	if f.inputCall != nil {
		return f.inputCall(ctx, input, args...)
	}
	if f.call != nil {
		return f.call(ctx, args...)
	}
	return nil, fmt.Errorf("unexpected Docker command: %s", strings.Join(args, " "))
}

func (f *fakeRunner) RunStdoutLimitQuiet(ctx context.Context, limit int, args ...string) ([]byte, bool, error) {
	output, err := f.Run(ctx, args...)
	if len(output) > limit {
		return output[:limit], true, err
	}
	return output, false, err
}

func (f *fakeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if f.call == nil {
		return nil, fmt.Errorf("unexpected Docker command: %s", strings.Join(args, " "))
	}
	return f.call(ctx, args...)
}

func TestInvalidSandboxIDDoesNotCallDocker(t *testing.T) {
	runner := &fakeRunner{}
	owner := &Owner{Runner: runner}
	if _, err := owner.Start(context.Background(), "not-a-uuid"); err == nil {
		t.Fatal("Start accepted an invalid Sandbox ID")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("invalid Sandbox ID reached Docker: %#v", runner.calls)
	}
}

func TestLimitedCaptureCapsRetainedWorkflowOutput(t *testing.T) {
	capture := &limitedCapture{limit: 5}
	if count, err := capture.Write([]byte("1234")); err != nil || count != 4 {
		t.Fatalf("first write = %d, %v", count, err)
	}
	if count, err := capture.Write([]byte("567890")); err != nil || count != 6 {
		t.Fatalf("second write = %d, %v", count, err)
	}
	if capture.buffer.String() != "12345" || !capture.exceeded {
		t.Fatalf("capture retained unbounded output or missed truncation: content=%q exceeded=%t", capture.buffer.String(), capture.exceeded)
	}
}

func TestClassifyWorkflowExecutionFailureDoesNotReturnEngineContent(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty diagnostic", input: "", want: "engine-exit-no-diagnostic"},
		{name: "missing workflow", input: "Could not find workflow with ID secret-workflow-id", want: "workflow-not-found"},
		{name: "missing workflow from n8n CLI", input: "The workflow with the id \"secret-workflow-id\" does not exist.", want: "workflow-not-found"},
		{name: "owner bootstrap", input: "Instance owner not found", want: "instance-owner-not-found"},
		{name: "user management uninitialized", input: "User management is not initialized", want: "user-management-uninitialized"},
		{name: "missing owner record", input: "No owner is configured", want: "instance-owner-missing"},
		{name: "missing owner user", input: "User not found", want: "owner-user-not-found"},
		{name: "database", input: "database connection failed", want: "database-unavailable"},
		{name: "missing node", input: "Unknown node type n8n-nodes-base.privateNode", want: "node-type-unavailable"},
		{name: "timeout", input: "execution timed out", want: "execution-timeout"},
		{name: "ambiguous permission text", input: "permission denied", want: "engine-execution-error"},
		{name: "authorization denied", input: "request unauthorized", want: "execution-permission-denied"},
		{name: "task runner broker port conflict", input: "listen EADDRINUSE: address already in use 127.0.0.1:5679", want: "task-runner-broker-port-conflict"},
		{name: "unassigned project", input: "The workflow project is not assigned.", want: "workflow-project-unavailable"},
		{name: "missing cli command", input: "unknown command execute", want: "cli-command-unavailable"},
		{name: "unknown engine error", input: "private response body secret-value", want: "engine-execution-error"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := classifyWorkflowExecutionFailure([]byte(test.input))
			if got != test.want {
				t.Fatalf("failure class = %q, want %q", got, test.want)
			}
			if strings.Contains(got, "secret") || strings.Contains(got, "private") {
				t.Fatalf("failure class leaked engine content: %q", got)
			}
		})
	}
}

func TestClassifyWorkflowJSONFailureUsesOnlySafeTypeEnum(t *testing.T) {
	input := []byte(`{"data":{"resultData":{"error":{"name":"NodeApiError","message":"private response body secret"}}}}`)
	if got := classifyWorkflowExecutionFailure(input); got != "node-api-error" {
		t.Fatalf("failure class = %q, want node-api-error", got)
	}
}

func TestClassifySyntheticWorkflowImportFailureRetainsOnlyClassAndHash(t *testing.T) {
	stdout := []byte(`No node type found: private node data`)
	class, digest := classifySyntheticWorkflowImportFailure(stdout, nil)
	if class != "node-type-unavailable" || len(digest) != 64 || strings.Contains(class, "private") || strings.Contains(digest, "private") {
		t.Fatalf("import diagnostic class=%q digest=%q", class, digest)
	}
	class, digest = classifySyntheticWorkflowImportFailure(nil, []byte(`private engine output`))
	if class != "server-cli-import-error" || len(digest) != 64 {
		t.Fatalf("unknown import diagnostic class=%q digest=%q", class, digest)
	}
	if class, digest = classifySyntheticWorkflowImportFailure(nil, nil); class != "server-cli-import-error" || digest != "" {
		t.Fatalf("empty import diagnostic class=%q digest=%q", class, digest)
	}
}

func TestRecordNonzeroExecutionOutputKeepsOnlyAggregateNodeFailureEvidence(t *testing.T) {
	output := []byte(`{"data":{"resultData":{"runData":{"HTTP Request":[{"error":{"message":"HTTP request failed: secret response"},"executionStatus":"error","data":{"main":[[{"json":{"secret":"raw-output"}}]]}}]}}}}`)
	receipt := WorkflowExecutionReceipt{Status: "failed"}
	if !recordNonzeroExecutionOutput(&receipt, output) {
		t.Fatal("valid failed-node output was not recorded")
	}
	if receipt.Status != "completed-with-node-errors" || receipt.FailureClass != "node-execution-error" || receipt.Evidence.FailedNodeExecutionCount != 1 || receipt.Evidence.ItemCount != 1 {
		t.Fatalf("receipt = %#v, want aggregate failed-node evidence", receipt)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "raw-output") {
		t.Fatalf("receipt leaked raw execution content: %s", encoded)
	}
}

func TestSummarizeWorkflowExecutionPreservesWorkflowErrorWithoutRunData(t *testing.T) {
	output := []byte(`{"data":{"resultData":{"error":{"name":"WorkflowRunError","message":"secret response body"}}}}`)
	evidence, err := summarizeWorkflowExecution(output)
	if err != nil {
		t.Fatalf("summarize workflow-level failure: %v", err)
	}
	if evidence.ExecutionCount != 1 || evidence.FailedNodeExecutionCount != 1 || evidence.OutputBytes != len(output) {
		t.Fatalf("workflow failure evidence = %#v", evidence)
	}
}

func TestStartRefusesMissingDaemonAndImageWithoutPull(t *testing.T) {
	t.Run("daemon unavailable", func(t *testing.T) {
		runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
			return nil, errors.New("daemon unavailable")
		}}
		owner := &Owner{Runner: runner}
		if _, err := owner.Start(context.Background(), testSandboxID); !errors.Is(err, ErrDockerUnavailable) {
			t.Fatalf("Start error = %v, want ErrDockerUnavailable", err)
		}
		if len(runner.calls) != 1 || strings.Join(runner.calls[0], " ") != "info --format {{.DockerRootDir}}" {
			t.Fatalf("daemon failure issued unexpected commands: %#v", runner.calls)
		}
	})

	t.Run("pinned image absent", func(t *testing.T) {
		runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
			if strings.Join(args, " ") == "info --format {{.DockerRootDir}}" {
				return []byte("/var/lib/docker"), nil
			}
			if len(args) >= 2 && args[0] == "image" && args[1] == "inspect" {
				return nil, errors.New("No such image")
			}
			return nil, fmt.Errorf("unexpected command: %s", strings.Join(args, " "))
		}}
		owner := &Owner{Runner: runner, Image: ImageRef}
		if _, err := owner.Start(context.Background(), testSandboxID); !errors.Is(err, ErrImageMissing) {
			t.Fatalf("Start error = %v, want ErrImageMissing", err)
		}
		for _, call := range runner.calls {
			if len(call) > 0 && call[0] == "pull" {
				t.Fatalf("Start implicitly pulled image: %#v", runner.calls)
			}
		}
	})
}

func TestSyntheticPostgresRefusesMissingPinnedImageWithoutPull(t *testing.T) {
	imageInspected := false
	runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case joined == "info --format {{.DockerRootDir}}":
			return []byte("/var/lib/docker"), nil
		case len(args) == 3 && args[0] == "image" && args[1] == "inspect":
			if args[2] != SyntheticPostgresImage {
				t.Fatalf("unexpected image reference: %v", args)
			}
			imageInspected = true
			return nil, errors.New("No such image")
		default:
			return nil, fmt.Errorf("unexpected Docker command: %s", joined)
		}
	}}
	owner := &Owner{Runner: runner, Image: ImageRef}
	if _, err := owner.StartSyntheticPostgres(context.Background(), testSandboxID, SyntheticPostgresName, SyntheticPostgresVersion, SyntheticPostgresDB); err == nil || !strings.Contains(err.Error(), "image is not installed") {
		t.Fatalf("missing pinned Postgres image error = %v", err)
	}
	if !imageInspected {
		t.Fatal("did not verify the pinned Postgres image")
	}
	for _, args := range runner.calls {
		if len(args) > 0 && args[0] == "pull" {
			t.Fatalf("companion start pulled an image: %#v", runner.calls)
		}
		if len(args) > 0 && args[0] == "run" {
			t.Fatalf("companion started without its pinned image: %#v", runner.calls)
		}
	}
}

func TestSyntheticPostgresRejectsCallerChosenImageProfileBeforeDocker(t *testing.T) {
	runner := &fakeRunner{}
	owner := &Owner{Runner: runner}
	if _, err := owner.StartSyntheticPostgres(context.Background(), testSandboxID, "pg", "latest", "client-db"); err == nil {
		t.Fatal("accepted caller-selected Postgres profile")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("invalid profile reached Docker: %#v", runner.calls)
	}
}

func TestSyntheticPostgresRequiresOnlyTmpfsForWritableState(t *testing.T) {
	var inspect containerInspect
	inspect.Mounts = []struct {
		Type        string `json:"Type"`
		Destination string `json:"Destination"`
	}{{Type: "tmpfs", Destination: "/var/lib/postgresql/data"}, {Type: "tmpfs", Destination: "/var/run/postgresql"}, {Type: "tmpfs", Destination: "/tmp"}}
	if err := verifySyntheticPostgresTmpfs(inspect); err != nil {
		t.Fatalf("valid tmpfs mounts rejected: %v", err)
	}
	inspect.Mounts[0].Type = "volume"
	if err := verifySyntheticPostgresTmpfs(inspect); err == nil {
		t.Fatal("image-declared disk volume was accepted as ephemeral tmpfs")
	}
}

func TestSyntheticPostgresCreatesNoneNetworkContainerThenOwnerAttachesIt(t *testing.T) {
	created, running, attached := false, false, false
	_, taskNetwork := objectNames(testSandboxID)
	runner := &fakeRunner{}
	runner.call = func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case joined == "info --format {{.DockerRootDir}}":
			return []byte("/var/lib/docker"), nil
		case joined == "image inspect "+SyntheticPostgresImage:
			return []byte(SyntheticPostgresImage), nil
		case len(args) == 3 && args[0] == "container" && args[1] == "inspect":
			if args[2] == postgresContainerName(testSandboxID) {
				if !created {
					return nil, errors.New("No such container")
				}
				return syntheticPostgresInspect(testSandboxID, created, running, attached), nil
			}
			if args[2] == func() string { name, _ := objectNames(testSandboxID); return name }() {
				return syntheticN8nInspect(testSandboxID), nil
			}
			if args[2] == "postgres-container" && created {
				return syntheticPostgresInspect(testSandboxID, created, running, attached), nil
			}
		case len(args) >= 2 && args[0] == "run":
			if !containsArgPair(args, "--network", "none") {
				t.Fatalf("Postgres container not created network=none: %#v", args)
			}
			for _, forbidden := range []string{"--publish", "--volume", "-v", "--network-alias"} {
				if containsArg(args, forbidden) {
					t.Fatalf("unexpected egress/persistence option %q: %#v", forbidden, args)
				}
			}
			if !containsArg(args, "--read-only") || !containsArgPair(args, "--cap-drop", "ALL") || !containsArg(args, "--security-opt") {
				t.Fatalf("Postgres container missing hardening flags: %#v", args)
			}
			if args[len(args)-1] != SyntheticPostgresImage {
				t.Fatalf("caller-selected Postgres image: %#v", args)
			}
			created, running = true, true
			return []byte("postgres-container"), nil
		case len(args) > 0 && args[0] == "exec":
			if strings.Join(args[1:], " ") != "postgres-container pg_isready -U synthetic -d workflow" {
				t.Fatalf("unexpected fixed readiness command: %#v", args)
			}
			return []byte("accepting connections"), nil
		case len(args) >= 2 && args[0] == "network" && args[1] == "inspect":
			return []byte(`[ {"Name":"` + taskNetwork + `","Driver":"bridge","Internal":true,"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"network"},"Containers":{}} ]`), nil
		case len(args) > 0 && args[0] == "ps":
			return []byte("postgres-container\n"), nil
		case len(args) >= 2 && args[0] == "network" && args[1] == "connect":
			attached = true
			return nil, nil
		case len(args) >= 2 && args[0] == "network" && args[1] == "disconnect":
			attached = false
			return nil, nil
		case len(args) >= 2 && args[0] == "stop":
			running = false
			return nil, nil
		case len(args) >= 2 && args[0] == "rm":
			created, running, attached = false, false, false
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected Docker command: %s", joined)
		}
		return nil, fmt.Errorf("unexpected Docker command: %s", joined)
	}
	owner := &Owner{Runner: runner, Image: ImageRef}
	receipt, err := owner.StartSyntheticPostgres(context.Background(), testSandboxID, SyntheticPostgresName, SyntheticPostgresVersion, SyntheticPostgresDB)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.NoEgressVerified || receipt.Status != "attached" || receipt.CompanionRuntimeID != postgresRuntimeIDFor(testSandboxID) || !attached {
		t.Fatalf("unexpected owner receipt or attach state: %+v attached=%t", receipt, attached)
	}
	container, found, err := owner.inspectContainer(context.Background(), postgresContainerName(testSandboxID))
	if err != nil || !found {
		t.Fatalf("inspect after start: found=%t err=%v", found, err)
	}
	if err := verifyNoEgressExceptTask(container, taskNetwork); err != nil {
		t.Fatalf("Postgres output did not retain only none and the owned task network: %v", err)
	}
	if err := owner.StopSyntheticPostgres(context.Background(), testSandboxID, receipt.CompanionRuntimeID); err != nil {
		t.Fatalf("stop owned companion: %v", err)
	}
	if created || running || attached {
		t.Fatalf("stop did not remove only the companion: created=%t running=%t attached=%t", created, running, attached)
	}
	if err := owner.StopSyntheticPostgres(context.Background(), testSandboxID, receipt.CompanionRuntimeID); err != nil {
		t.Fatalf("idempotent stop: %v", err)
	}
	var removedAnonymousVolumes bool
	for _, args := range runner.calls {
		if len(args) >= 3 && args[0] == "rm" && args[1] == "--volumes" && args[2] == "postgres-container" {
			removedAnonymousVolumes = true
		}
	}
	if !removedAnonymousVolumes {
		t.Fatalf("stop did not clean the matching image-declared anonymous volume: %#v", runner.calls)
	}
	if len(runner.calls) == 0 {
		t.Fatal("owner performed no Docker operations")
	}
}

func syntheticPostgresInspect(sandboxID string, created, running, attached bool) []byte {
	if !created {
		return []byte("[]")
	}
	networks := `"none":{"Aliases":[]}`
	_, taskNetwork := objectNames(sandboxID)
	if attached {
		networks += `,"` + taskNetwork + `":{"Aliases":["` + companionAlias(sandboxID, postgresRuntimeIDFor(sandboxID)) + `"]}`
	}
	return []byte(`[ {"Id":"postgres-container","Config":{"Labels":{"com.vrooli.resource-companion":"n8n-postgres","com.vrooli.workspace-sandbox.sandbox-id":"` + sandboxID + `","com.vrooli.workspace-sandbox.runtime-id":"` + postgresRuntimeIDFor(sandboxID) + `","com.vrooli.workspace-sandbox.companion-declared":"true"}},"State":{"Running":` + fmt.Sprint(running) + `},"Mounts":[{"Type":"tmpfs","Destination":"/var/lib/postgresql/data"},{"Type":"tmpfs","Destination":"/var/run/postgresql"},{"Type":"tmpfs","Destination":"/tmp"}],"NetworkSettings":{"Networks":{` + networks + `}}} ]`)
}

func syntheticN8nInspect(sandboxID string) []byte {
	_, taskNetwork := objectNames(sandboxID)
	return []byte(`[ {"Id":"n8n-container","Config":{"Image":"` + ImageRef + `","Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + sandboxID + `","com.vrooli.runtime-id":"n8n:` + sandboxID + `","com.vrooli.object-kind":"container"}},"State":{"Running":true},"NetworkSettings":{"Ports":{"5678/tcp":[{"HostIp":"127.0.0.1","HostPort":"15678"}]},"Networks":{"` + taskNetwork + `":{"Aliases":["` + n8nAlias(sandboxID) + `"]}}}} ]`)
}

func containsArg(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted {
			return true
		}
	}
	return false
}

func containsArgPair(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func TestSyntheticPostgresStopRejectsForeignRuntimeBeforeDocker(t *testing.T) {
	runner := &fakeRunner{}
	owner := &Owner{Runner: runner}
	if err := owner.StopSyntheticPostgres(context.Background(), testSandboxID, "sandbox.postgres:foreign"); !errors.Is(err, ErrForeignObject) {
		t.Fatalf("stop error = %v, want foreign object", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("foreign runtime ID reached Docker: %#v", runner.calls)
	}
}

func TestAcquireImageReportsSizeAndRetentionForPinnedReuse(t *testing.T) {
	runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch joined {
		case "info --format {{.DockerRootDir}}":
			return []byte("/var/lib/docker"), nil
		case "image inspect " + ImageRef:
			return []byte(ImageRef), nil
		case "image inspect --format {{.Size}} " + ImageRef:
			return []byte("1234567"), nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", joined)
		}
	}}
	owner := &Owner{Runner: runner, Image: ImageRef}
	receipt, err := owner.AcquireImage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.AlreadyPresent || receipt.Acquired || receipt.ImageSizeBytes != 1234567 || receipt.ReusePolicy == "" || receipt.EvictionRule == "" {
		t.Fatalf("retained image receipt is incomplete: %#v", receipt)
	}
}

func TestAttachCompanionRejectsForeignAndEgressCapableRuntime(t *testing.T) {
	for _, tc := range []struct {
		name         string
		companionIDs map[string]string
		networks     []string
		wantErr      error
		wantConnect  bool
	}{
		{name: "foreign Sandbox", companionIDs: map[string]string{labelSandboxOwner: "another-sandbox", labelSandboxRunID: "db:runtime-1", labelSandboxDecl: "true"}, wantErr: ErrForeignObject},
		{name: "undeclared companion", companionIDs: map[string]string{labelSandboxOwner: testSandboxID, labelSandboxRunID: "db:runtime-1", labelSandboxDecl: "false"}, wantErr: ErrCompanionDenied},
		{name: "additional egress network", companionIDs: map[string]string{labelSandboxOwner: testSandboxID, labelSandboxRunID: "db:runtime-1", labelSandboxDecl: "true"}, networks: []string{"bridge"}, wantErr: ErrCompanionDenied},
		{name: "isolated none network", companionIDs: map[string]string{labelSandboxOwner: testSandboxID, labelSandboxRunID: "db:runtime-1", labelSandboxDecl: "true"}, networks: []string{"none"}, wantConnect: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connected := false
			runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
				joined := strings.Join(args, " ")
				switch {
				case len(args) >= 2 && args[0] == "network" && args[1] == "inspect":
					return []byte(`[ {"Name":"vrooli-n8n-net-5e33a6f0b1bf4f5d931595ba60d9e025","Driver":"bridge","Internal":true,"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"network"},"Containers":{}} ]`), nil
				case len(args) >= 2 && args[0] == "container" && args[1] == "inspect" && args[len(args)-1] == "vrooli-n8n-5e33a6f0b1bf4f5d931595ba60d9e025":
					return []byte(`[ {"Id":"n8n-container","Config":{"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"container"}},"State":{"Running":true},"NetworkSettings":{"Ports":{"5678/tcp":[{"HostIp":"127.0.0.1","HostPort":"15678"}]},"Networks":{"vrooli-n8n-net-5e33a6f0b1bf4f5d931595ba60d9e025":{"Aliases":["n8n-5e33a6f0b1bf4f5d931595ba60d9e025"]}}}} ]`), nil
				case len(args) > 0 && args[0] == "ps":
					return []byte("companion-container\n"), nil
				case len(args) >= 2 && args[0] == "container" && args[1] == "inspect":
					var networkJSON string
					if len(tc.networks) > 0 {
						networkJSON = `,"` + tc.networks[0] + `":{"Aliases":[]}`
					}
					labels := fmt.Sprintf(`"%s":"%s","%s":"%s","%s":"%s"`, labelSandboxOwner, tc.companionIDs[labelSandboxOwner], labelSandboxRunID, tc.companionIDs[labelSandboxRunID], labelSandboxDecl, tc.companionIDs[labelSandboxDecl])
					return []byte(`[ {"Id":"companion-container","Config":{"Labels":{` + labels + `}},"State":{"Running":true},"NetworkSettings":{"Networks":{` + strings.TrimPrefix(networkJSON, ",") + `}}} ]`), nil
				case len(args) > 0 && args[0] == "network" && args[1] == "connect":
					connected = true
					return nil, nil
				default:
					return nil, fmt.Errorf("unexpected command: %s", joined)
				}
			}}
			owner := &Owner{Runner: runner}
			_, err := owner.AttachCompanion(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), "db:runtime-1")
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("AttachCompanion error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && err != nil {
				t.Fatalf("AttachCompanion error = %v, want success", err)
			}
			if connected != tc.wantConnect {
				t.Fatalf("network connect = %v, want %v", connected, tc.wantConnect)
			}
		})
	}
}

func TestStartCleansPartialOwnedContainerAndNetwork(t *testing.T) {
	containerExists, networkExists, running := false, false, false
	ephemeralCredentialStore, correctUserFolder, correctDatabasePath := false, false, false
	runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case joined == "info --format {{.DockerRootDir}}":
			return []byte("/var/lib/docker"), nil
		case len(args) >= 2 && args[0] == "image" && args[1] == "inspect":
			return []byte(ImageRef), nil
		case len(args) >= 2 && args[0] == "network" && args[1] == "inspect":
			if !networkExists {
				return nil, errors.New("No such network")
			}
			return []byte(`[ {"Name":"vrooli-n8n-net-5e33a6f0b1bf4f5d931595ba60d9e025","Driver":"bridge","Internal":true,"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"network"},"Containers":{}} ]`), nil
		case len(args) > 0 && args[0] == "network" && args[1] == "create":
			networkExists = true
			return []byte("network-id"), nil
		case len(args) > 0 && args[0] == "run":
			if !containsArgPair(args, "--env", "N8N_RUNNERS_BROKER_PORT=0") {
				t.Fatal("owner runtime did not reserve an ephemeral task-runner broker port")
			}
			correctUserFolder = containsArgPair(args, "--env", "N8N_USER_FOLDER=/home/node")
			correctDatabasePath = containsArgPair(args, "--env", "DB_SQLITE_DATABASE=/home/node/.n8n/database.sqlite")
			for i := 0; i+1 < len(args); i++ {
				if args[i] == "--tmpfs" && strings.HasPrefix(args[i+1], "/home/node/.n8n:") {
					ephemeralCredentialStore = true
				}
			}
			containerExists, running = true, true
			return []byte("partial create"), errors.New("daemon interrupted after create")
		case len(args) >= 2 && args[0] == "container" && args[1] == "inspect":
			if !containerExists {
				return nil, errors.New("No such container")
			}
			return []byte(`[ {"Id":"n8n-container","Config":{"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"container"}},"State":{"Running":true}} ]`), nil
		case len(args) > 0 && args[0] == "stop":
			running = false
			return nil, nil
		case len(args) > 0 && args[0] == "rm":
			containerExists = false
			return nil, nil
		case len(args) >= 2 && args[0] == "network" && args[1] == "rm":
			networkExists = false
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", joined)
		}
	}}
	owner := &Owner{Runner: runner, Image: ImageRef}
	if _, err := owner.Start(context.Background(), testSandboxID); err == nil {
		t.Fatal("Start unexpectedly succeeded after Docker reported a partial create error")
	}
	if !ephemeralCredentialStore {
		t.Fatal("n8n credential database is not on the owner-managed ephemeral tmpfs")
	}
	if !correctUserFolder {
		t.Fatal("n8n user folder must be the parent of the owner-managed /home/node/.n8n tmpfs")
	}
	if !correctDatabasePath {
		t.Fatal("server and CLI must use the same absolute SQLite database inside the owner-managed tmpfs")
	}
	if containerExists || networkExists || running {
		t.Fatalf("partial owned runtime was not cleaned: container=%v network=%v running=%v", containerExists, networkExists, running)
	}
}

func TestInitializeSyntheticOwnerUsesOnlyTaskLoopbackAndStdin(t *testing.T) {
	var fakeFailure string
	runner := &fakeRunner{inputCall: func(_ context.Context, input []byte, args ...string) ([]byte, error) {
		wantArgs := []string{"exec", "--interactive", "--user", "node", "task-container", "node", "-e", initializeSyntheticOwnerProgram, testSandboxID, runtimeIDFor(testSandboxID)}
		if strings.Join(args, "\x00") != strings.Join(wantArgs, "\x00") {
			fakeFailure = "unexpected command arguments"
			return nil, errors.New("unexpected command arguments")
		}
		var body map[string]string
		if err := json.Unmarshal(input, &body); err != nil {
			fakeFailure = "malformed JSON payload"
			return nil, errors.New("malformed JSON payload")
		}
		password := body["password"]
		if body["email"] != "owner@synthetic.invalid" || body["firstName"] != "Synthetic" || body["lastName"] != "Rehearsal" || len(password) < 8 || len(password) > 64 || !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") {
			fakeFailure = "payload or password policy mismatch"
			return nil, errors.New("payload or password policy mismatch")
		}
		if bytes.Contains([]byte(strings.Join(args, " ")), []byte(password)) {
			fakeFailure = "password leaked into arguments"
			return nil, errors.New("password leaked into arguments")
		}
		return []byte("201"), nil
	}}
	if err := initializeSyntheticOwner(context.Background(), runner, "task-container", testSandboxID, runtimeIDFor(testSandboxID)); err != nil {
		t.Fatalf("initialize synthetic owner: %v (%s)", err, fakeFailure)
	}
	if !strings.Contains(initializeSyntheticOwnerProgram, "ownerId:id") || !strings.Contains(initializeSyntheticOwnerProgram, "flag:'wx'") || !strings.Contains(initializeSyntheticOwnerProgram, syntheticOwnerIdentityPath) {
		t.Fatal("owner bootstrap does not persist only the validated task-scoped owner identity")
	}
}

func TestSyntheticOwnerIDRequiresMatchingTaskAndBoundedValue(t *testing.T) {
	valid := `{"sandboxId":"` + testSandboxID + `","runtimeId":"` + runtimeIDFor(testSandboxID) + `","ownerId":"synthetic-owner_7"}`
	cases := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{name: "valid task identity", output: valid, want: "synthetic-owner_7"},
		{name: "missing owner identity", output: "__missing__", wantErr: true},
		{name: "wrong sandbox", output: strings.Replace(valid, testSandboxID, "6e33a6f0-b1bf-4f5d-9315-95ba60d9e025", 1), wantErr: true},
		{name: "wrong runtime", output: strings.Replace(valid, runtimeIDFor(testSandboxID), "n8n:6e33a6f0-b1bf-4f5d-9315-95ba60d9e025", 1), wantErr: true},
		{name: "invalid owner ID", output: strings.Replace(valid, "synthetic-owner_7", "owner/id", 1), wantErr: true},
		{name: "oversized owner ID", output: strings.Replace(valid, "synthetic-owner_7", strings.Repeat("x", 129), 1), wantErr: true},
		{name: "invalid JSON", output: `{"ownerId":`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
				if len(args) < 7 || args[0] != "exec" || args[len(args)-2] != "-e" || args[len(args)-1] != readSyntheticOwnerIdentityProgram {
					return nil, fmt.Errorf("unexpected owner identity read command")
				}
				return []byte(tc.output), nil
			}}
			got, err := syntheticOwnerID(context.Background(), runner, "task-container", testSandboxID, runtimeIDFor(testSandboxID))
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("owner ID = %q err=%v, want ID %q error=%t", got, err, tc.want, tc.wantErr)
			}
			if strings.Contains(fmt.Sprint(err), tc.output) {
				t.Fatal("owner identity value escaped diagnostic")
			}
		})
	}
}

func TestValidateTaskTrustCARequiresOneSigningCA(t *testing.T) {
	t.Run("rejects malformed input", func(t *testing.T) {
		if _, _, err := validateTaskTrustCA([]byte("not a certificate")); err == nil {
			t.Fatal("accepted malformed task trust input")
		}
	})
	t.Run("rejects non CA certificate", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := validateTaskTrustCA(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})); err == nil {
			t.Fatal("accepted a leaf certificate as a task trust CA")
		}
	})
	t.Run("canonicalizes one signing CA", func(t *testing.T) {
		_, gatewayReceipt, err := virtualapi.New(virtualapi.Contract{
			TaskID: testSandboxID,
			Hosts:  []string{"api.notion.com"},
			Routes: []virtualapi.Route{{Host: "api.notion.com", Method: "GET", Path: "/v1/users", Status: 200, ContentType: "application/json", Body: []byte(`{"results":[]}`)}},
		})
		if err != nil {
			t.Fatal(err)
		}
		canonical, fingerprint, err := validateTaskTrustCA(gatewayReceipt.CACertPEM)
		if err != nil {
			t.Fatalf("valid generated task CA rejected: %v", err)
		}
		if len(canonical) == 0 || len(fingerprint) != 64 || fingerprint != gatewayReceipt.CAFingerprint {
			t.Fatalf("CA was not canonically identified: bytes=%d fingerprint=%q", len(canonical), fingerprint)
		}
	})
}

func TestPrepareSyntheticWorkflowAcceptsNativeExportMetadataSafely(t *testing.T) {
	const source = `{"name":"public export","nodeGroups":[{"name":"Display group","nodeIds":["node-1"]}],"nodes":[{"id":"node-1","name":"Manual Start","type":"n8n-nodes-base.manualTrigger","typeVersion":1,"position":[0,0],"parameters":{}}],"connections":{},"settings":{"executionOrder":"v1","binaryMode":"separate"}}`
	prepared, _, err := prepareSyntheticWorkflow([]byte(source), "0123456789abcdefghij", testSandboxID, runtimeIDFor(testSandboxID), "")
	if err != nil {
		t.Fatalf("prepare public n8n export: %v", err)
	}
	var workflow map[string]any
	if err := json.Unmarshal(prepared, &workflow); err != nil {
		t.Fatal(err)
	}
	if _, ok := workflow["nodeGroups"]; ok {
		t.Fatal("editor-only nodeGroups metadata was retained")
	}
	settings, ok := workflow["settings"].(map[string]any)
	if !ok || settings["binaryMode"] != "separate" {
		t.Fatalf("n8n-supported binaryMode was not preserved: %#v", workflow["settings"])
	}
	unsafe := strings.Replace(source, `"binaryMode":"separate"`, `"binaryMode":"external"`, 1)
	if _, _, err := prepareSyntheticWorkflow([]byte(unsafe), "0123456789abcdefghij", testSandboxID, runtimeIDFor(testSandboxID), ""); err == nil {
		t.Fatal("accepted an unsupported binary mode")
	}
}

func TestStopIsSafeAndIdempotentWhenRuntimeIsAbsent(t *testing.T) {
	runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "container" && args[1] == "inspect" {
			return nil, errors.New("No such container")
		}
		if len(args) >= 2 && args[0] == "network" && args[1] == "inspect" {
			return nil, errors.New("No such network")
		}
		return nil, fmt.Errorf("unexpected mutation command: %s", strings.Join(args, " "))
	}}
	owner := &Owner{Runner: runner}
	for i := 0; i < 2; i++ {
		result, err := owner.Stop(context.Background(), testSandboxID, runtimeIDFor(testSandboxID))
		if err != nil || result.Status != "stopped" {
			t.Fatalf("Stop attempt %d = %#v, %v", i+1, result, err)
		}
	}
	for _, call := range runner.calls {
		if len(call) > 0 && (call[0] == "rm" || call[0] == "stop" || (call[0] == "network" && len(call) > 1 && call[1] != "inspect")) {
			t.Fatalf("absent runtime triggered a Docker mutation: %#v", call)
		}
	}
}

func TestStopNeverMutatesForeignContainer(t *testing.T) {
	runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "container" && args[1] == "inspect" {
			return []byte(`[ {"Id":"foreign","Config":{"Labels":{"com.vrooli.resource":"somebody-else"}},"State":{"Running":true}} ]`), nil
		}
		return nil, fmt.Errorf("unexpected command after foreign inspection: %s", strings.Join(args, " "))
	}}
	owner := &Owner{Runner: runner}
	if _, err := owner.Stop(context.Background(), testSandboxID, runtimeIDFor(testSandboxID)); !errors.Is(err, ErrForeignObject) {
		t.Fatalf("Stop error = %v, want ErrForeignObject", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("foreign object caused additional Docker operations: %#v", runner.calls)
	}
}

func TestStopRemovesOnlyOwnedRuntimeAndNetwork(t *testing.T) {
	containerExists, networkExists, running := true, true, true
	runner := &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case len(args) >= 2 && args[0] == "container" && args[1] == "inspect":
			if !containerExists {
				return nil, errors.New("No such container")
			}
			status := "false"
			if running {
				status = "true"
			}
			return []byte(`[ {"Id":"owned","Config":{"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"container"}},"State":{"Running":` + status + `}} ]`), nil
		case len(args) > 0 && args[0] == "stop":
			running = false
			return nil, nil
		case len(args) > 0 && args[0] == "rm":
			containerExists = false
			return nil, nil
		case len(args) >= 2 && args[0] == "network" && args[1] == "inspect":
			if !networkExists {
				return nil, errors.New("No such network")
			}
			return []byte(`[ {"Name":"vrooli-n8n-net-5e33a6f0b1bf4f5d931595ba60d9e025","Driver":"bridge","Internal":true,"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"n8n:` + testSandboxID + `","com.vrooli.object-kind":"network"},"Containers":{}} ]`), nil
		case len(args) >= 2 && args[0] == "network" && args[1] == "rm":
			networkExists = false
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", joined)
		}
	}}
	owner := &Owner{Runner: runner}
	if _, err := owner.Stop(context.Background(), testSandboxID, runtimeIDFor(testSandboxID)); err != nil {
		t.Fatal(err)
	}
	if containerExists || networkExists || running {
		t.Fatalf("owned runtime not removed: container=%v network=%v running=%v", containerExists, networkExists, running)
	}
}

const workflowOperationID = "f0a7d3e6-2a6b-4f8a-9714-1bb8be3eb1e2"

const syntheticWorkflowJSON = `{"id":"client-id-to-discard","name":"synthetic rehearsal","nodes":[{"id":"node-1","name":"Manual Start","type":"n8n-nodes-base.manualTrigger","typeVersion":1,"position":[0,0],"parameters":{}}],"connections":{},"active":true,"settings":{"executionOrder":"v1"}}`

func ownedRuntimeRunner(t *testing.T, runtimeID string) *fakeRunner {
	t.Helper()
	_, networkName := objectNames(testSandboxID)
	containerName, _ := objectNames(testSandboxID)
	return &fakeRunner{call: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case joined == "info --format {{.DockerRootDir}}":
			return []byte("/var/lib/docker"), nil
		case len(args) >= 2 && args[0] == "container" && args[1] == "inspect":
			if args[len(args)-1] != containerName {
				return nil, fmt.Errorf("unexpected container: %s", joined)
			}
			return []byte(`[ {"Id":"owned-n8n","Config":{"Image":"` + ImageRef + `","Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"` + runtimeID + `","com.vrooli.object-kind":"container"}},"State":{"Running":true},"NetworkSettings":{"Ports":{"5678/tcp":[{"HostIp":"127.0.0.1","HostPort":"15678"}]},"Networks":{"` + networkName + `":{"Aliases":["` + n8nAlias(testSandboxID) + `"]}}}} ]`), nil
		case len(args) >= 2 && args[0] == "network" && args[1] == "inspect":
			return []byte(`[ {"Name":"` + networkName + `","Driver":"bridge","Internal":true,"Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"` + runtimeID + `","com.vrooli.object-kind":"network"},"Containers":{}} ]`), nil
		case strings.Contains(joined, readSyntheticOwnerIdentityProgram):
			return []byte(`{"sandboxId":"` + testSandboxID + `","runtimeId":"` + runtimeID + `","ownerId":"synthetic-owner-7"}`), nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", joined)
		}
	}}
}

func TestImportSyntheticWorkflowUsesPinnedCLIAndNodeTmpfsStage(t *testing.T) {
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	var importArgs []string
	runner.inputCall = func(_ context.Context, workflowBytes []byte, args ...string) ([]byte, error) {
		if strings.Join(args, "\x00") != strings.Join([]string{"exec", "--interactive", "--user", "node", "vrooli-n8n-5e33a6f0b1bf4f5d931595ba60d9e025", "node", "-e", stageWorkflowFileProgram, "/tmp/.vrooli-workflow-" + workflowIDFor(testSandboxID, runtimeIDFor(testSandboxID), uuid.MustParse(workflowOperationID)) + ".json"}, "\x00") {
			return nil, fmt.Errorf("unexpected input command: %s", strings.Join(args, " "))
		}
		var workflow map[string]json.RawMessage
		if err := json.Unmarshal(workflowBytes, &workflow); err != nil {
			return nil, err
		}
		var id string
		_ = json.Unmarshal(workflow["id"], &id)
		if len(id) == 0 || len(id) > 21 || id == "client-id-to-discard" || string(workflow["active"]) != "false" {
			return nil, fmt.Errorf("workflow ID/active state was not owner-sanitized")
		}
		if strings.Contains(string(workflowBytes), "client-id-to-discard") {
			return nil, fmt.Errorf("caller workflow ID was not removed")
		}
		if !strings.Contains(stageWorkflowFileProgram, "O_NOFOLLOW") || !strings.Contains(stageWorkflowFileProgram, "0o600") || strings.Contains(strings.Join(args, " "), "sh -c") {
			return nil, fmt.Errorf("workflow stage is not fixed, no-follow and private")
		}
		return nil, nil
	}
	runner.call = func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, readSyntheticOwnerIdentityProgram) {
			return []byte(`{"sandboxId":"` + testSandboxID + `","runtimeId":"` + runtimeIDFor(testSandboxID) + `","ownerId":"synthetic-owner-7"}`), nil
		}
		if strings.Contains(joined, " n8n import:workflow ") {
			importArgs = append([]string(nil), args...)
			return []byte("Successfully imported 1 workflow."), nil
		}
		if strings.Contains(joined, " rm -f -- /tmp/.vrooli-workflow-") {
			return nil, nil
		}
		return baseCall(context.Background(), args...)
	}
	owner := &Owner{Runner: runner}
	operationID := uuid.MustParse(workflowOperationID)
	receipt, err := owner.ImportSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), SyntheticWorkflowImport{OperationID: operationID, JSON: []byte(syntheticWorkflowJSON)})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "imported-inactive" || receipt.WorkflowID != workflowIDFor(testSandboxID, runtimeIDFor(testSandboxID), operationID) || receipt.InputRetained {
		t.Fatalf("unexpected import receipt: %#v", receipt)
	}
	wantImport := []string{"exec", "--user", "node", "vrooli-n8n-5e33a6f0b1bf4f5d931595ba60d9e025", "n8n", "import:workflow", "--input=/tmp/.vrooli-workflow-" + receipt.WorkflowID + ".json", "--activeState=false", "--userId=synthetic-owner-7"}
	if strings.Join(importArgs, "\x00") != strings.Join(wantImport, "\x00") {
		t.Fatalf("import command = %#v, want %#v", importArgs, wantImport)
	}
	encodedReceipt, _ := json.Marshal(receipt)
	if strings.Contains(string(encodedReceipt), "synthetic-owner-7") {
		t.Fatal("internal owner identity leaked in workflow import receipt")
	}
	if len(runner.stdin) != 1 {
		t.Fatalf("workflow stdin count = %d, want 1", len(runner.stdin))
	}
}

func TestPrepareSyntheticWebhookAlwaysStaysInactiveAndKeepsItsRouteBoundToInput(t *testing.T) {
	const source = `{"name":"synthetic webhook","active":true,"nodes":[{"id":"source-node","name":"Webhook","type":"n8n-nodes-base.webhook","typeVersion":2.2,"position":[0,0],"parameters":{"httpMethod":"POST","path":"synthetic-event","responseMode":"onReceived","options":{}}}],"connections":{}}`
	const workflowID = "0123456789abcdefghij"
	prepared, credentials, err := prepareSyntheticWorkflow([]byte(source), workflowID, testSandboxID, runtimeIDFor(testSandboxID), "")
	if err != nil {
		t.Fatalf("prepare synthetic webhook: %v", err)
	}
	if len(credentials) != 0 {
		t.Fatalf("webhook with no credentials generated unexpected credential stand-ins: %d", len(credentials))
	}
	var workflow struct {
		ID     string `json:"id"`
		Active bool   `json:"active"`
		Nodes  []struct {
			Type       string `json:"type"`
			Parameters struct {
				Method string `json:"httpMethod"`
				Path   string `json:"path"`
			} `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(prepared, &workflow); err != nil {
		t.Fatal(err)
	}
	if workflow.ID != workflowID || workflow.Active || len(workflow.Nodes) != 1 {
		t.Fatalf("webhook import was not owner-bound and inactive: %#v", workflow)
	}
	if workflow.Nodes[0].Type != "n8n-nodes-base.webhook" || workflow.Nodes[0].Parameters.Method != "POST" || workflow.Nodes[0].Parameters.Path != "synthetic-event" {
		t.Fatalf("webhook route/method did not remain attached to sanitized workflow: %#v", workflow.Nodes[0])
	}
}

func TestImportRejectsCredentialsBeforeDockerAndCleansPartialInput(t *testing.T) {
	credentialWorkflow := `{"name":"unsafe","nodes":[{"name":"node","type":"n8n-nodes-base.httpRequest","credentials":{"httpBasicAuth":{"id":"client-secret-id"}}}],"connections":{}}`
	runner := &fakeRunner{}
	owner := &Owner{Runner: runner}
	_, err := owner.ImportSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), SyntheticWorkflowImport{OperationID: uuid.MustParse(workflowOperationID), JSON: []byte(credentialWorkflow)})
	if err == nil || len(runner.calls) != 0 {
		t.Fatalf("credential-bearing workflow should be refused before Docker calls, err=%v calls=%#v", err, runner.calls)
	}

	runner = ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	runner.inputCall = func(context.Context, []byte, ...string) ([]byte, error) { return nil, errors.New("stage interrupted") }
	runner.call = func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, " rm -f -- /tmp/.vrooli-workflow-") {
			return nil, nil
		}
		return baseCall(context.Background(), args...)
	}
	owner = &Owner{Runner: runner}
	_, err = owner.ImportSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), SyntheticWorkflowImport{OperationID: uuid.MustParse(workflowOperationID), JSON: []byte(syntheticWorkflowJSON)})
	if err == nil {
		t.Fatal("partial Docker copy unexpectedly succeeded")
	}
	for _, call := range runner.calls {
		if strings.Contains(strings.Join(call, " "), " n8n import:workflow ") {
			t.Fatalf("import ran after partial copy failed: %#v", runner.calls)
		}
	}
}

func TestSyntheticCredentialProfilesAreAllowlistedAndIdempotent(t *testing.T) {
	workflow := `{"name":"with stand-ins","nodes":[{"name":"GitHub","type":"n8n-nodes-base.github","credentials":{"githubApi":{"id":"client-github-id","name":"Client GitHub"}}},{"name":"Notion","type":"n8n-nodes-base.notion","credentials":{"notionApi":{"id":"client-notion-id","name":"Client Notion"}}},{"name":"Supabase","type":"n8n-nodes-base.supabase","credentials":{"supabaseApi":{"id":"client-supabase-id","name":"Client Supabase"}}},{"name":"WooCommerce","type":"n8n-nodes-base.wooCommerce","credentials":{"wooCommerceApi":{"id":"client-woo-id","name":"Client Woo"}}},{"name":"HTTP","type":"n8n-nodes-base.httpRequest","credentials":{"httpHeaderAuth":{"id":"client-header-id","name":"Client Header"}}},{"name":"Postgres","type":"n8n-nodes-base.postgres","credentials":{"postgres":{"id":"client-postgres-id","name":"Client Postgres"}}},{"name":"OpenAI metadata","type":"n8n-nodes-base.httpRequest","parameters":{"nodeCredentialType":"openAiApi"}},{"name":"Pinecone metadata","type":"n8n-nodes-base.httpRequest","parameters":{"nodeCredentialType":"pineconeApi"}},{"name":"OpenAI chat","type":"@n8n/n8n-nodes-langchain.lmChatOpenAi","parameters":{}},{"name":"OpenAI embedding","type":"@n8n/n8n-nodes-langchain.embeddingsOpenAi","parameters":{}},{"name":"Pinecone vector store","type":"@n8n/n8n-nodes-langchain.vectorStorePinecone","parameters":{}}],"connections":{}}`
	companionRuntimeID := "sandbox.postgres"
	prepared, credentials, err := prepareSyntheticWorkflow([]byte(workflow), "workflow-id", testSandboxID, runtimeIDFor(testSandboxID), companionRuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(credentials) != 11 {
		t.Fatalf("credential count = %d, want 11", len(credentials))
	}
	wantTypes := map[string]bool{"githubApi": true, "notionApi": true, "supabaseApi": true, "wooCommerceApi": true, "httpHeaderAuth": true, "postgres": true, "openAiApi": true, "pineconeApi": true}
	gotTypes := map[string]bool{}
	for _, record := range credentials {
		gotTypes[record.Type] = true
		if len(record.ID) != 20 || !strings.HasPrefix(record.Name, "Vrooli synthetic ") {
			t.Fatalf("credential was not assigned a synthetic owner identity: %#v", record)
		}
		if strings.Contains(record.Name, "Client") || strings.Contains(record.ID, "client-") {
			t.Fatalf("client credential identity leaked into synthetic record: %#v", record)
		}
		if strings.Contains(fmt.Sprint(record.Data), "Client") {
			t.Fatalf("client credential name leaked into stand-in data: %#v", record.Data)
		}
	}
	for credentialType := range wantTypes {
		if !gotTypes[credentialType] {
			t.Errorf("synthetic profile %q was not created", credentialType)
		}
	}
	var imported map[string]any
	if err := json.Unmarshal(prepared, &imported); err != nil {
		t.Fatal(err)
	}
	preparedNodes := imported["nodes"].([]any)
	for _, value := range preparedNodes {
		node := value.(map[string]any)
		refs := node["credentials"].(map[string]any)
		for _, refValue := range refs {
			ref := refValue.(map[string]any)
			if strings.Contains(ref["name"].(string), "Client") || strings.Contains(ref["id"].(string), "client-") {
				t.Fatalf("source credential identity survived remapping: %#v", ref)
			}
		}
	}
	_, credentialsAgain, err := prepareSyntheticWorkflow([]byte(workflow), "workflow-id-2", testSandboxID, runtimeIDFor(testSandboxID), companionRuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, record := range credentials {
		ids[record.Type+":"+record.ID] = true
	}
	for _, record := range credentialsAgain {
		if !ids[record.Type+":"+record.ID] {
			t.Fatalf("same Sandbox source credential did not reuse owner ID: %#v", record)
		}
	}
	var postgres syntheticCredentialRecord
	for _, record := range credentials {
		if record.Type == "postgres" {
			postgres = record
		}
	}
	if postgres.Data["host"] != companionAlias(testSandboxID, companionRuntimeID) || postgres.Data["password"] != "vrooli-workflow-rehearsal-only" || postgres.Data["database"] != SyntheticPostgresDB {
		t.Fatalf("Postgres stand-in does not use owner alias and synthetic secret: %#v", postgres.Data)
	}
	unsupported := strings.Replace(workflow, "githubApi", "unapprovedOAuthType", 1)
	if _, _, err := prepareSyntheticWorkflow([]byte(unsupported), "workflow-id", testSandboxID, runtimeIDFor(testSandboxID), ""); err == nil {
		t.Fatal("credential type outside the synthetic allowlist was accepted")
	}
}

func TestSyntheticPublicWorkflowCredentialProfilesHaveNoExternalIdentity(t *testing.T) {
	workflow := `{"name":"public integration profiles","nodes":[{"name":"Gmail","type":"n8n-nodes-base.gmail","parameters":{}},{"name":"Sheets","type":"n8n-nodes-base.googleSheets","parameters":{}},{"name":"GitLab","type":"n8n-nodes-base.gitlab","parameters":{"authentication":"accessToken"},"credentials":{"gitlabApi":{"id":"public-source-id","name":"Public GitLab"}}},{"name":"GitLab OAuth","type":"n8n-nodes-base.gitlab","parameters":{"authentication":"oAuth2"}},{"name":"n8n API","type":"n8n-nodes-base.n8n","parameters":{},"credentials":{"n8nApi":{"id":"public-source-api-id","name":"Public n8n"}}},{"name":"Notion","type":"n8n-nodes-base.notion","parameters":{}},{"name":"Slack","type":"n8n-nodes-base.slack","parameters":{}},{"name":"Telegram","type":"n8n-nodes-base.telegram","parameters":{}},{"name":"Discord webhook","type":"n8n-nodes-base.discord","parameters":{"authentication":"webhook"}}],"connections":{}}`
	prepared, credentials, err := prepareSyntheticWorkflow([]byte(workflow), "workflow-id", testSandboxID, runtimeIDFor(testSandboxID), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(credentials) != 9 {
		t.Fatalf("credentials = %d, want nine shared integration profiles", len(credentials))
	}
	want := map[string]bool{"gmailOAuth2": false, "googleSheetsOAuth2Api": false, "gitlabApi": false, "gitlabOAuth2Api": false, "n8nApi": false, "notionApi": false, "slackApi": false, "telegramApi": false, "discordWebhookApi": false}
	for _, credential := range credentials {
		if _, ok := want[credential.Type]; !ok {
			t.Fatalf("unexpected profile type %q", credential.Type)
		}
		want[credential.Type] = true
		if strings.Contains(credential.Name, "Public") || strings.Contains(fmt.Sprint(credential.Data), "Public") || strings.Contains(fmt.Sprint(credential.Data), "public-source") {
			t.Fatalf("public workflow credential identity leaked into profile: %#v", credential)
		}
	}
	for credentialType, found := range want {
		if !found {
			t.Errorf("missing synthetic profile %q", credentialType)
		}
	}
	var exported map[string]any
	if err := json.Unmarshal(prepared, &exported); err != nil {
		t.Fatal(err)
	}
	nodes := exported["nodes"].([]any)
	for index, credentialType := range []string{"gmailOAuth2", "googleSheetsOAuth2Api", "gitlabApi", "gitlabOAuth2Api", "n8nApi", "notionApi", "slackApi", "telegramApi", "discordWebhookApi"} {
		references := nodes[index].(map[string]any)["credentials"].(map[string]any)
		if _, exists := references[credentialType]; !exists {
			t.Errorf("node %d did not get owner-issued %s reference", index, credentialType)
		}
	}
	for _, credentialType := range []string{"gmailOAuth2", "googleSheetsOAuth2Api"} {
		profile, ok := syntheticCredentialProfile(credentialType)
		if !ok {
			t.Fatalf("missing OAuth profile %q", credentialType)
		}
		data, err := profile(testSandboxID, "")
		if err != nil {
			t.Fatal(err)
		}
		tokens, ok := data["oauthTokenData"].(map[string]any)
		if !ok || tokens["access_token"] != "vrooli-synthetic-no-egress" || tokens["refresh_token"] != "vrooli-synthetic-no-egress" {
			t.Fatalf("OAuth profile %q lacks inert owner tokens: %#v", credentialType, data)
		}
	}
	unsupported := strings.Replace(workflow, `"n8nApi"`, `"unapprovedOAuthType"`, 1)
	if _, _, err := prepareSyntheticWorkflow([]byte(unsupported), "workflow-id", testSandboxID, runtimeIDFor(testSandboxID), ""); err == nil {
		t.Fatal("unknown public credential profile was accepted")
	}
}

func TestCredentialSourceValuesCannotReachRunnerOrError(t *testing.T) {
	secret := "client-secret-must-not-be-logged-or-imported"
	workflow := `{"name":"unsafe","nodes":[{"name":"GitHub","type":"n8n-nodes-base.github","credentials":{"githubApi":{"id":"source-id","name":"source","data":{"accessToken":"` + secret + `"}}}}],"connections":{}}`
	runner := &fakeRunner{}
	owner := &Owner{Runner: runner}
	_, err := owner.ImportSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), SyntheticWorkflowImport{OperationID: uuid.MustParse(workflowOperationID), JSON: []byte(workflow)})
	if err == nil || strings.Contains(err.Error(), secret) || len(runner.calls) != 0 {
		t.Fatalf("credential source data was accepted or surfaced: err=%v calls=%#v", err, runner.calls)
	}
}

func TestImportSyntheticCredentialStandInsUsesPinnedCLIAndCleansTmpfs(t *testing.T) {
	workflow := `{"name":"with GitHub stand-in","nodes":[{"name":"GitHub","type":"n8n-nodes-base.github","credentials":{"githubApi":{"id":"client-export-id","name":"Client GitHub"}}}],"connections":{}}`
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	credentialJSON := []byte(nil)
	workflowJSON := []byte(nil)
	runner.inputCall = func(_ context.Context, input []byte, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, ".vrooli-credentials-"):
			credentialJSON = append([]byte(nil), input...)
		case strings.Contains(joined, ".vrooli-workflow-"):
			workflowJSON = append([]byte(nil), input...)
		default:
			return nil, fmt.Errorf("unexpected staged path: %s", joined)
		}
		if strings.Contains(string(input), "Client GitHub") || strings.Contains(string(input), "client-export-id") {
			return nil, fmt.Errorf("source credential identity was staged")
		}
		return nil, nil
	}
	runner.call = func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "n8n import:credentials"):
			return []byte("Successfully imported 1 credential."), nil
		case strings.Contains(joined, "n8n import:workflow"):
			return []byte("Successfully imported 1 workflow."), nil
		case strings.Contains(joined, " rm -f -- /tmp/.vrooli-"):
			return nil, nil
		default:
			return baseCall(context.Background(), args...)
		}
	}
	owner := &Owner{Runner: runner}
	receipt, err := owner.ImportSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), SyntheticWorkflowImport{OperationID: uuid.MustParse(workflowOperationID), JSON: []byte(workflow)})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.CredentialCount != 1 || receipt.Status != "imported-inactive" || receipt.InputRetained {
		t.Fatalf("unexpected import receipt: %#v", receipt)
	}
	for _, payload := range [][]byte{credentialJSON, workflowJSON} {
		if len(payload) == 0 || strings.Contains(string(payload), "client-export-id") || strings.Contains(string(payload), "Client GitHub") {
			t.Fatalf("source credential identity reached staged JSON: %s", payload)
		}
	}
	if !strings.Contains(string(credentialJSON), "vrooli-synthetic-no-egress") {
		t.Fatalf("synthetic credential payload is missing inert stand-in value: %s", credentialJSON)
	}
	if !strings.Contains(string(workflowJSON), "Vrooli synthetic githubApi") {
		t.Fatalf("workflow credential ref was not mapped to owner-issued ID: %s", workflowJSON)
	}
	credentialImport, workflowImport, cleanupCount := false, false, 0
	for _, call := range runner.calls {
		joined := strings.Join(call, " ")
		credentialImport = credentialImport || strings.Contains(joined, "n8n import:credentials --input=/tmp/.vrooli-credentials-")
		workflowImport = workflowImport || strings.Contains(joined, "n8n import:workflow --input=/tmp/.vrooli-workflow-")
		if strings.Contains(joined, "rm -f -- /tmp/.vrooli-") {
			cleanupCount++
		}
		if strings.Contains(joined, "Client GitHub") || strings.Contains(joined, "client-export-id") {
			t.Fatalf("source credential identity reached command/log arguments: %s", joined)
		}
	}
	if !credentialImport || !workflowImport || cleanupCount != 2 {
		t.Fatalf("missing import or tmpfs cleanup command: credential=%t workflow=%t cleanup=%d calls=%#v", credentialImport, workflowImport, cleanupCount, runner.calls)
	}
}

func TestPostgresStandInRequiresOwnedAttachedCompanionBeforeStaging(t *testing.T) {
	workflow := `{"name":"Postgres","nodes":[{"name":"Postgres","type":"n8n-nodes-base.postgres","credentials":{"postgres":{"id":"source-db-id","name":"Client database"}}}],"connections":{}}`
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	owner := &Owner{Runner: runner}
	_, err := owner.ImportSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), SyntheticWorkflowImport{
		OperationID: uuid.MustParse(workflowOperationID), JSON: []byte(workflow), PostgresCompanionRuntimeID: "sandbox.postgres",
	})
	if err == nil {
		t.Fatal("Postgres stand-in imported without proving its declared companion attachment")
	}
	if len(runner.stdin) != 0 {
		t.Fatalf("credential data staged before attached companion proof: %#v", runner.stdin)
	}
	for _, call := range runner.calls {
		if strings.Contains(strings.Join(call, " "), "n8n import:") {
			t.Fatalf("credential import ran without attached companion: %#v", runner.calls)
		}
	}
}

func TestCredentialImportFailureCleansInputsAndPreventsWorkflowImport(t *testing.T) {
	workflow := `{"name":"GitHub","nodes":[{"name":"GitHub","type":"n8n-nodes-base.github","credentials":{"githubApi":{"id":"source-id","name":"Source"}}}],"connections":{}}`
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	runner.inputCall = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		if strings.Contains(strings.Join(args, " "), ".vrooli-credentials-") {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected staged path: %s", strings.Join(args, " "))
	}
	runner.call = func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "n8n import:credentials") {
			return []byte("Skipped 1 credential blocked by policy"), nil
		}
		if strings.Contains(joined, " rm -f -- /tmp/.vrooli-credentials-") {
			return nil, nil
		}
		if strings.Contains(joined, " rm -f -- /tmp/.vrooli-workflow-") {
			return nil, nil
		}
		return baseCall(context.Background(), args...)
	}
	owner := &Owner{Runner: runner}
	receipt, err := owner.ImportSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), SyntheticWorkflowImport{OperationID: uuid.MustParse(workflowOperationID), JSON: []byte(workflow)})
	if err == nil || receipt.Status != "credential-import-failed" {
		t.Fatalf("policy-skipped credentials were reported as successful: receipt=%#v err=%v", receipt, err)
	}
	for _, call := range runner.calls {
		if strings.Contains(strings.Join(call, " "), "n8n import:workflow") {
			t.Fatalf("workflow import proceeded after credentials were skipped: %#v", runner.calls)
		}
	}
}

func TestExecuteSyntheticWorkflowUsesOnlyOwnerDerivedWorkflowID(t *testing.T) {
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	var executeArgs []string
	secretSentinel := "private-source-value-81"
	runner.call = func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, " n8n execute ") {
			executeArgs = append([]string(nil), args...)
			return []byte(`{"data":{"resultData":{"runData":{"Create contact":[{"data":{"main":[[{"json":{"id":"` + secretSentinel + `","token":"secret-value"}},{"json":{"id":"2"}}]]}}],"Read rows":[{"data":{"main":[[{"json":{"email":"customer@example.invalid"}}]],"ai_tool":[[{"json":{"input":"other connection"}}]]}}]}}}}`), nil
		}
		return baseCall(context.Background(), args...)
	}
	owner := &Owner{Runner: runner}
	operationID := uuid.MustParse(workflowOperationID)
	receipt, err := owner.ExecuteSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), operationID)
	if err != nil {
		t.Fatal(err)
	}
	wantID := workflowIDFor(testSandboxID, runtimeIDFor(testSandboxID), operationID)
	want := []string{"exec", "--user", "node", "vrooli-n8n-5e33a6f0b1bf4f5d931595ba60d9e025", "n8n", "execute", "--id", wantID, "--rawOutput"}
	if receipt.WorkflowID != wantID || receipt.Status != "completed" || receipt.Evidence.NodeCount != 2 || receipt.Evidence.ExecutionCount != 2 || receipt.Evidence.ItemCount != 4 || receipt.Evidence.OutputBytes == 0 || receipt.Evidence.OutputSHA256 == "" {
		t.Fatalf("unexpected execute receipt: %#v", receipt)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secretSentinel) || strings.Contains(string(encoded), "secret-value") || strings.Contains(string(encoded), "customer@example.invalid") || strings.Contains(string(encoded), "Create contact") {
		t.Fatalf("execution receipt exposed output values or node names: %s", encoded)
	}
	if strings.Join(executeArgs, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("execute command = %#v, want %#v", executeArgs, want)
	}
}

func TestExecuteSyntheticWorkflowDoesNotExposeRuntimeErrorOutput(t *testing.T) {
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	runner.call = func(ctx context.Context, args ...string) ([]byte, error) {
		if strings.Contains(strings.Join(args, " "), " n8n execute ") {
			return []byte("raw runtime log containing private-value-99"), errors.New("runtime error private-value-99")
		}
		return baseCall(ctx, args...)
	}
	owner := &Owner{Runner: runner}
	receipt, err := owner.ExecuteSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), uuid.MustParse(workflowOperationID))
	if err == nil || receipt.Status != "failed" || strings.Contains(err.Error(), "private-value-99") {
		t.Fatalf("execution failure leaked runtime output: receipt=%+v err=%v", receipt, err)
	}
}

func TestSummarizeWorkflowExecutionRejectsUnsupportedShapes(t *testing.T) {
	for _, output := range [][]byte{nil, []byte(`not json`), []byte(`{"data":{}}`)} {
		if evidence, err := summarizeWorkflowExecution(output); err == nil || evidence.OutputSHA256 != "" {
			t.Fatalf("unsupported result shape accepted: evidence=%+v err=%v", evidence, err)
		}
	}
}

func TestSummarizeWorkflowExecutionReportsNodeErrorWithoutLeakingIt(t *testing.T) {
	secret := "node error private payload"
	output := []byte(`{"data":{"resultData":{"runData":{"HTTP Request":[{"error":{"message":"` + secret + `"}},{"executionStatus":"crashed","data":{"main":[[]]}}]}}}}`)
	evidence, err := summarizeWorkflowExecution(output)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.NodeCount != 1 || evidence.ExecutionCount != 2 || evidence.FailedNodeExecutionCount != 2 || strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), "HTTP Request") {
		t.Fatalf("node failure evidence leaked output or lost failure count: %+v %s", evidence, encoded)
	}
}

func TestSummarizeWorkflowExecutionAcceptsCompleteEnvelopeAfterCLIText(t *testing.T) {
	output := []byte("n8n cli status line\n" + `{"data":{"resultData":{"runData":{"Set":[{"data":{"main":[[{"json":{"marker":"synthetic"}}]]}}]}}}}`)
	evidence, err := summarizeWorkflowExecution(output)
	if err != nil {
		t.Fatalf("summarize CLI output envelope: %v", err)
	}
	if evidence.NodeCount != 1 || evidence.ExecutionCount != 1 || evidence.ItemCount != 1 || len(evidence.NodeOutputItemCounts) != 1 || evidence.NodeOutputItemCounts[0] != 1 {
		t.Fatalf("CLI output aggregate evidence = %+v", evidence)
	}
}

func TestExecuteSyntheticWorkflowRejectsNonPinnedRuntimeImage(t *testing.T) {
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	executed := false
	runner.call = func(ctx context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if len(args) >= 2 && args[0] == "container" && args[1] == "inspect" {
			data, err := baseCall(ctx, args...)
			return []byte(strings.Replace(string(data), `"Image":"`+ImageRef+`"`, `"Image":"n8nio/n8n:other"`, 1)), err
		}
		if strings.Contains(joined, " n8n execute ") {
			executed = true
		}
		return baseCall(ctx, args...)
	}
	owner := &Owner{Runner: runner}
	if _, err := owner.ExecuteSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), uuid.MustParse(workflowOperationID)); err == nil {
		t.Fatal("execution accepted a task runtime on a non-pinned image")
	}
	if executed {
		t.Fatal("execution command ran before pinned image identity was verified")
	}
	customOwnerRunner := &fakeRunner{}
	customOwner := &Owner{Runner: customOwnerRunner, Image: "n8nio/n8n:custom"}
	if _, err := customOwner.ExecuteSyntheticWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), uuid.MustParse(workflowOperationID)); err == nil || len(customOwnerRunner.calls) != 0 {
		t.Fatalf("custom owner image reached Docker or was accepted: calls=%#v err=%v", customOwnerRunner.calls, err)
	}
}
