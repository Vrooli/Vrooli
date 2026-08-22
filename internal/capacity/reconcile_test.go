package capacity

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// fakeAttributor maps PIDs to fixed attributions for deterministic tests.
type fakeAttributor map[int]Attribution

func (f fakeAttributor) Attribute(_ context.Context, pid int) Attribution {
	if a, ok := f[pid]; ok {
		a.PID = pid
		return a
	}
	return Attribution{PID: pid, OwnerID: OwnerUnknown}
}

func snapshotWithProcs(procs ...hostinventory.GPUProcess) hostinventory.Snapshot {
	return hostinventory.Snapshot{
		GPUs:         []hostinventory.GPU{{Index: 0, Name: "Test", Source: "nvidia-smi", VRAMBytes: 16 * uint64(gib)}},
		GPUProcesses: procs,
	}
}

func TestReconcileFlagsUnclaimedConsumer(t *testing.T) {
	ctx := context.Background()
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "python", UsedBytes: 7 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}

	findings := Reconcile(ctx, snap, nil, attr, DefaultPolicy())
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}
	f := findings[0]
	if f.Class != FindingUnclaimed || f.Severity != "warn" {
		t.Errorf("class/severity = %s/%s, want unclaimed/warn", f.Class, f.Severity)
	}
	if f.OwnerID != "whisper" {
		t.Errorf("owner = %q, want whisper (normalized)", f.OwnerID)
	}
}

func TestReconcileClassifiesClaimed(t *testing.T) {
	ctx := context.Background()
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "whisper", UsedBytes: 7 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}
	ledger := []CapacityClaim{{
		ClaimID: "clm-w", OwnerID: "whisper", OwnerKind: OwnerKindResource, ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted,
	}}

	findings := Reconcile(ctx, snap, ledger, attr, DefaultPolicy())
	if len(findings) != 1 || findings[0].Class != FindingClaimed {
		t.Fatalf("findings = %+v, want one claimed", findings)
	}
	if findings[0].Severity != "info" || findings[0].ClaimID != "clm-w" {
		t.Errorf("claimed finding = %+v", findings[0])
	}
}

func TestReconcileFlagsOverClaim(t *testing.T) {
	ctx := context.Background()
	// Uses 8 GiB but only claimed 2 GiB; drift well over the 512 MiB default.
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "sd", UsedBytes: 8 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/image-tools", OwnerID: "image-tools"}}
	ledger := []CapacityClaim{{
		ClaimID: "clm-it", OwnerID: "image-tools:job-1", OwnerKind: OwnerKindOp, ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 2 * gib, Status: StatusGranted,
	}}

	findings := Reconcile(ctx, snap, ledger, attr, DefaultPolicy())
	if len(findings) != 1 || findings[0].Class != FindingOverClaim {
		t.Fatalf("findings = %+v, want over_claim (op-owner should match scenario prefix)", findings)
	}
	if findings[0].Severity != "warn" {
		t.Errorf("over_claim severity = %q, want warn", findings[0].Severity)
	}
}

func TestReconcileIgnoresBelowThreshold(t *testing.T) {
	ctx := context.Background()
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, UsedBytes: 64 * 1024 * 1024}) // 64 MiB < 256 MiB
	if findings := Reconcile(ctx, snap, nil, fakeAttributor{}, DefaultPolicy()); len(findings) != 0 {
		t.Errorf("findings = %+v, want none below tracking threshold", findings)
	}
}

func TestReconcileNilAttributorDegradesToUnknown(t *testing.T) {
	ctx := context.Background()
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "ollama", UsedBytes: 3 * uint64(gib)})
	findings := Reconcile(ctx, snap, nil, nil, DefaultPolicy())
	if len(findings) != 1 || findings[0].Class != FindingUnclaimed {
		t.Fatalf("findings = %+v, want one unclaimed", findings)
	}
	if findings[0].OwnerID != "ollama" { // falls back to process name
		t.Errorf("owner = %q, want process-name fallback ollama", findings[0].OwnerID)
	}
}

func TestNormalizeOwnerName(t *testing.T) {
	cases := map[string]string{
		"/vrooli-whisper-1":  "whisper",
		"vrooli_kyutai-stt":  "kyutai-stt",
		"/resource-ollama-2": "ollama",
		"image-tools":        "image-tools",
		"":                   OwnerUnknown,
	}
	for in, want := range cases {
		if got := NormalizeOwnerName(in); got != want {
			t.Errorf("NormalizeOwnerName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeProcessOwnerRecognizesNativeOllamaWorker(t *testing.T) {
	if got := NormalizeProcessOwner("/opt/ollama/lib/ollama/llama-server"); got != "ollama" {
		t.Fatalf("owner = %q, want ollama", got)
	}
	if got := NormalizeProcessOwner("/usr/bin/unrelated-worker"); got != "" {
		t.Fatalf("unrelated owner = %q, want empty", got)
	}
}

func TestNormalizeProcessOwnerRecognizesNativeRerankerWorker(t *testing.T) {
	if got := NormalizeProcessOwner("/home/user/.vrooli/artifacts/reranker/1.7.4/reranker_linux_amd64"); got != "reranker" {
		t.Fatalf("NormalizeProcessOwner(reranker artifact) = %q, want reranker", got)
	}
}

func TestDeclaredGPUWithoutClaimFindingsCatchesIdleDeclaration(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "ollama")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"name":"ollama","acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{}}}`)
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}

	findings, err := DeclaredGPUWithoutClaimFindings(root, nil, nil)
	if err != nil {
		t.Fatalf("DeclaredGPUWithoutClaimFindings() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %+v, want one declaration finding", findings)
	}
	if got := findings[0]; got.Class != FindingDeclaredUnclaimed || got.OwnerID != "ollama" || got.Severity != "warn" {
		t.Fatalf("finding = %+v, want declared_unclaimed ollama warn", got)
	}

	// And a resource that declares only the cpu backend is not reported: it
	// asked for no device, so holding no claim is correct rather than a
	// mismatch. This is the inversion the single declaration removes — the
	// previous shape reported a resource for merely having a gpu block, while
	// missing every managed-service resource entirely.
	cpuOnly := filepath.Join(root, "resources", "sherpa-onnx")
	if err := os.MkdirAll(cpuOnly, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cpuOnly, "resource.json"), []byte(`{"name":"sherpa-onnx","acceleration":{"backends":["cpu"],"cpu":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err = DeclaredGPUWithoutClaimFindings(root, nil, nil)
	if err != nil {
		t.Fatalf("DeclaredGPUWithoutClaimFindings() error = %v", err)
	}
	for _, finding := range findings {
		if finding.OwnerID == "sherpa-onnx" {
			t.Fatalf("a cpu-only resource was reported as declaring an accelerator: %+v", finding)
		}
	}

	claim := CapacityClaim{OwnerKind: OwnerKindResource, OwnerID: "ollama", Status: StatusGranted}
	findings, err = DeclaredGPUWithoutClaimFindings(root, []CapacityClaim{claim}, nil)
	if err != nil {
		t.Fatalf("claimed check error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings with active claim = %+v, want none", findings)
	}
}

// Scenario: a resource that is not installed here holds no claim, and that is
// not a mismatch.
//
// Without this filter every accelerator-declaring resource an operator has
// never installed raises a permanent warning, which is what made this finding
// class ignorable.
func TestDeclaredUnclaimedSkipsResourcesThatAreNotInstalled(t *testing.T) {
	// Given two accelerator-declaring resources, only one of them installed
	root := t.TempDir()
	for _, name := range []string{"ollama", "kokoro"} {
		dir := filepath.Join(root, "resources", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := []byte(`{"name":"` + name + `","acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{}}}`)
		if err := os.WriteFile(filepath.Join(dir, "resource.json"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	installed := func(resource string) bool { return resource == "ollama" }

	// When findings are computed with the installed predicate
	findings, err := DeclaredGPUWithoutClaimFindings(root, nil, installed)
	if err != nil {
		t.Fatalf("DeclaredGPUWithoutClaimFindings() error = %v", err)
	}

	// Then only the installed one is reported
	if len(findings) != 1 || findings[0].OwnerID != "ollama" {
		t.Fatalf("findings = %+v, want only the installed ollama", findings)
	}

	// And with no predicate every declaration is reported, because the caller
	// has no way to tell
	all, err := DeclaredGPUWithoutClaimFindings(root, nil, nil)
	if err != nil {
		t.Fatalf("DeclaredGPUWithoutClaimFindings() error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("findings without a predicate = %+v, want both declarations", all)
	}
}
