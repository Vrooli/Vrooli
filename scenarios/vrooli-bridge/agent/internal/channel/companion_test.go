package channel

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	companionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion"
)

type companionRunner struct{ argv [][]string }

func (r *companionRunner) Run(_ context.Context, argv []string, _ string, _ func(string)) (int, error) {
	r.argv = append(r.argv, append([]string(nil), argv...))
	return 0, nil
}

func TestValidCompanionCommandRejectsUnknownOperations(t *testing.T) {
	base := &companionv1.CompanionCommand{OperationId: "op-1", NodeId: "node-1", Kind: companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSPECT}
	if !validCompanionCommand(base) {
		t.Fatal("expected valid inspect command")
	}
	for _, candidate := range []*companionv1.CompanionCommand{
		{NodeId: "node-1", Kind: base.GetKind()},
		{OperationId: "op-1", Kind: base.GetKind()},
		{OperationId: "op-1", NodeId: "node-1"},
		{OperationId: "op-1", NodeId: "node-1", Kind: companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_UNSPECIFIED},
	} {
		if validCompanionCommand(candidate) {
			t.Fatalf("accepted invalid command: %+v", candidate)
		}
	}
}

func TestLocalCompanionAdapterNeverClaimsInstallReadyWithoutArtifact(t *testing.T) {
	response, err := (localCompanionLifecycleAdapter{}).HandleCompanionCommand(nil, &companionv1.CompanionCommand{OperationId: "op-1", NodeId: "node-1", Kind: companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSTALL})
	expectedReason := "artifact_not_installed"
	if runtime.GOOS != "darwin" {
		expectedReason = "unsupported_platform"
	}
	if err != nil || response.GetState() != companionv1.CompanionState_COMPANION_STATE_FAILED || response.GetReasonCode() != expectedReason {
		t.Fatalf("response=%+v err=%v", response, err)
	}
}

func TestLocalCompanionAdapterInstallsAndRemovesUserScopedFiles(t *testing.T) {
	home := t.TempDir()
	paths := newCompanionPaths(home)
	if err := os.MkdirAll(filepath.Dir(paths.artifact), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.artifact, []byte("companion"), 0o750); err != nil {
		t.Fatal(err)
	}
	runner := &companionRunner{}
	adapter := localCompanionLifecycleAdapter{homeDir: home, runner: runner}
	response := adapter.install(context.Background(), &companionv1.CompanionResponse{}, paths, "display-1")
	if response.GetState() != companionv1.CompanionState_COMPANION_STATE_READY || !response.GetUserLaunchAgent() {
		t.Fatalf("install response=%+v", response)
	}
	if _, err := os.Stat(paths.config); err != nil {
		t.Fatalf("config missing: %v", err)
	}
	if _, err := os.Stat(paths.plist); err != nil {
		t.Fatalf("plist missing: %v", err)
	}
	if len(runner.argv) != 2 || runner.argv[1][0] != "launchctl" || runner.argv[1][1] != "bootstrap" {
		t.Fatalf("launchctl calls=%v", runner.argv)
	}
	response = adapter.revoke(context.Background(), response, paths, true)
	if response.GetState() != companionv1.CompanionState_COMPANION_STATE_REMOVED {
		t.Fatalf("remove response=%+v", response)
	}
	for _, path := range []string{paths.artifact, paths.config, paths.plist} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists, err=%v", path, err)
		}
	}
}

func TestRenderLaunchAgentEscapesUserPaths(t *testing.T) {
	plist := renderLaunchAgent("/Users/A&B/bin/companion", "/Users/A&B/Library/Application Support/Vrooli/desktop.json")
	if !strings.Contains(plist, "DEVICE_CONTROL_COMPANION_PORT") || !strings.Contains(plist, companionPort) {
		t.Fatalf("launch agent does not pin the companion port: %s", plist)
	}
	if !strings.Contains(plist, "A&amp;B") || strings.Contains(plist, "<string>/Users/A&B/") {
		t.Fatalf("plist paths were not XML escaped: %s", plist)
	}
}
