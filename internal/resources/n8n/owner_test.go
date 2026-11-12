package n8n

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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
	ephemeralCredentialStore := false
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
	if containerExists || networkExists || running {
		t.Fatalf("partial owned runtime was not cleaned: container=%v network=%v running=%v", containerExists, networkExists, running)
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
	wantImport := []string{"exec", "--user", "node", "vrooli-n8n-5e33a6f0b1bf4f5d931595ba60d9e025", "n8n", "import:workflow", "--input=/tmp/.vrooli-workflow-" + receipt.WorkflowID + ".json", "--activeState=false"}
	if strings.Join(importArgs, "\x00") != strings.Join(wantImport, "\x00") {
		t.Fatalf("import command = %#v, want %#v", importArgs, wantImport)
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
	for _, output := range [][]byte{nil, []byte(`not json`), []byte(`{"data":{}}`), []byte(`{"data":{"resultData":{"error":{"message":"private failure"},"runData":{}}}}`)} {
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
