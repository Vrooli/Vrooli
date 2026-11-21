package rendering

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestPhysicalGPUUUIDUsesExplicitConfiguredIdentity(t *testing.T) {
	const want = "GPU-test-exact-identity"
	t.Setenv("DEVICE_STUDIO_GPU_UUID", "  "+want+"  ")
	got, err := physicalGPUUUID(context.Background())
	if err != nil {
		t.Fatalf("resolve pinned GPU identity: %v", err)
	}
	if got != want {
		t.Fatalf("resolved GPU UUID = %q, want exact configured UUID %q", got, want)
	}
}

func TestPhysicalGPUUUIDFromInventoryRequiresExactlyOneIdentifiedDevice(t *testing.T) {
	const want = "GPU-test-exact-identity"
	tests := []struct {
		name     string
		gpus     []*cliv1.HostGPU
		want     string
		contains string
	}{
		{name: "one identified GPU", gpus: []*cliv1.HostGPU{{Name: "Test GPU", Uuid: want}}, want: want},
		{name: "unknown inventory is reported as zero records", contains: "observed 0 GPU records"},
		{name: "multiple GPUs identify the actual entries", gpus: []*cliv1.HostGPU{{Name: "GPU A", Uuid: "GPU-a"}, {Name: "GPU B", Uuid: "GPU-b"}}, contains: `observed 2 GPU records: [0]{name:"GPU A",uuid:"GPU-a"}, [1]{name:"GPU B",uuid:"GPU-b"}`},
		{name: "unidentified device is surfaced", gpus: []*cliv1.HostGPU{{Name: "GPU without identity"}}, contains: `observed 1 GPU records: [0]{name:"GPU without identity",uuid:""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := physicalGPUUUIDFromInventory(tt.gpus)
			if tt.want != "" {
				if err != nil || got != tt.want {
					t.Fatalf("physicalGPUUUIDFromInventory() = %q, %v; want %q", got, err, tt.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("inventory failure = %v, want diagnostic containing %q", err, tt.contains)
			}
		})
	}
}

func TestPhysicalGPUSelectionProofIsUsefulBeforeRenderCompletes(t *testing.T) {
	proof := physicalGPUDiagnostics{
		Backend: "OPTIX", UUID: "GPU-test-exact-identity", SelectedDevice: "NVIDIA Test GPU",
		VisibleDevices: []string{"NVIDIA Test GPU"}, RenderCompleted: false,
	}
	if err := validatePhysicalGPUSelectionProof(proof, proof.UUID, proof.Backend); err != nil {
		t.Fatalf("valid selection proof from an interrupted render was rejected: %v", err)
	}
	proof.UUID = "GPU-other"
	if err := validatePhysicalGPUSelectionProof(proof, "GPU-test-exact-identity", "OPTIX"); err == nil {
		t.Fatal("GPU selection proof for a different leased UUID was accepted")
	}
	proof.UUID = "GPU-test-exact-identity"
	proof.VisibleDevices = []string{"NVIDIA Test GPU", "NVIDIA Other GPU"}
	if err := validatePhysicalGPUSelectionProof(proof, proof.UUID, proof.Backend); err == nil {
		t.Fatal("GPU selection proof exposing multiple devices was accepted")
	}
}

func TestPhysicalBlenderProcessAbortReapsBeforeWritingFailureDiagnostics(t *testing.T) {
	if os.Getenv("GO_WANT_PHYSICAL_BLENDER_PROCESS_HELPER") == "1" {
		select {}
	}
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	cmd := exec.Command(os.Args[0], "-test.run=^TestPhysicalBlenderProcessAbortReapsBeforeWritingFailureDiagnostics$")
	cmd.Env = append(os.Environ(), "GO_WANT_PHYSICAL_BLENDER_PROCESS_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &physicalBlenderLogBuffer{limit: 1024}, &physicalBlenderLogBuffer{limit: 1024}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	process := &physicalBlenderProcess{
		ctx: ctx, cancel: cancel, root: root, cmd: cmd,
		stdout: stdout, stderr: stderr, started: time.Now(), startedProc: true,
		manifest: PhysicalRenderManifest{Profile: PhysicalRenderProfile{Backend: "CPU"}, Output: PhysicalOutputPlan{OutputName: "physical-output.png"}},
	}
	if err := process.abort(); err == nil {
		t.Fatal("aborting a live Blender child did not return its terminal process error")
	}
	if !process.reaped || cmd.ProcessState == nil {
		t.Fatal("Blender child was not reaped before finish")
	}
	data, err := os.ReadFile(filepath.Join(rootPath, physicalBlenderLogName))
	if err != nil {
		t.Fatalf("failed process diagnostics were not persisted: %v", err)
	}
	var diagnostics physicalBlenderDiagnostics
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		t.Fatal(err)
	}
	if diagnostics.ExitError == "" || diagnostics.ElapsedMillis < 0 {
		t.Fatalf("cancellation diagnostics are not actionable: %+v", diagnostics)
	}
}
