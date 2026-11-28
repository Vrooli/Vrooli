package installgateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPythonLockStagePublishesExactStagedOutput(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "requirements.lock")
	if err := os.WriteFile(target, []byte("old==1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, ".stage")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "new.lock")
	if err := os.WriteFile(source, []byte("new==2.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	base, _ := os.ReadFile(target)
	stage := &PythonLockStage{Dir: dir, items: []pythonLockPublication{{source: source, output: target}}, bases: map[string][32]byte{target: sha256.Sum256(base)}, missing: map[string]bool{}}
	if err := stage.Publish(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "new==2.0\n" {
		t.Fatalf("published %q, err=%v", got, err)
	}
}

func TestPythonLockStageRefusesStaleTargetAndReleases(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "requirements.lock")
	if err := os.WriteFile(target, []byte("old==1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, ".stage")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "new.lock")
	if err := os.WriteFile(source, []byte("new==2.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage := &PythonLockStage{Dir: dir, items: []pythonLockPublication{{source: source, output: target}}, bases: map[string][32]byte{target: sha256.Sum256([]byte("old==1.0\n"))}, missing: map[string]bool{}}
	if err := os.WriteFile(target, []byte("concurrent==3.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := stage.Publish()
	if err == nil || !strings.Contains(err.Error(), "changed after admission") {
		t.Fatalf("expected stale-stage refusal, got %v", err)
	}
	stage.Release()
	stage.Release()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("stage directory remains: %v", err)
	}
}

func TestPythonLockStageRefusesChangedIndexManifest(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "requirements.lock")
	manifest := filepath.Join(root, "resource.json")
	if err := os.WriteFile(target, []byte("old==1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"index_url":"https://pypi.org/simple"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, ".stage")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "new.lock")
	if err := os.WriteFile(source, []byte("new==2.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, _ := os.ReadFile(target)
	indexes, _ := os.ReadFile(manifest)
	stage := &PythonLockStage{Dir: dir, items: []pythonLockPublication{{source: source, output: target}}, bases: map[string][32]byte{target: sha256.Sum256(lock), manifest: sha256.Sum256(indexes)}, missing: map[string]bool{}}
	if err := os.WriteFile(manifest, []byte(`{"index_url":"https://mirror.example/simple"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stage.Publish(); err == nil || !strings.Contains(err.Error(), "changed after admission") {
		t.Fatalf("expected manifest-change refusal, got %v", err)
	}
	stage.Release()
}

func TestDiskAdmissionBoundsPythonLockStageBeforeResolver(t *testing.T) {
	root := t.TempDir()
	locks := make([]string, pythonLockStageMaxFiles+1)
	for i := range locks {
		locks[i] = filepath.Join(root, "requirements"+string(rune('a'+i))+".lock")
	}
	plan, err := PlanDiskAdmission(context.Background(), Resolution{SurfaceRoot: root, RepositoryRoot: root, PackageManager: "uv-lock", PythonLockfiles: locks, PackageName: "onnxruntime-gpu", Argv: []string{"onnxruntime-gpu==1.27.0"}})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Release()
	if plan.Unsupported == "" || !strings.Contains(plan.Unsupported, "lockfiles") {
		t.Fatalf("expected closed lock-count bound, got %q", plan.Unsupported)
	}
}

func TestPythonLockStagePeakCoversAllBoundedLockCopies(t *testing.T) {
	// Every admitted lock can coexist as the copied resolver input, uv output,
	// canonical output and canonical .in file. Include bounded metadata overhead.
	minimum := int64(pythonLockStageMaxFiles)*4*pythonLockStageFileBytes + 2*(1<<20)
	if pythonLockStagePeakBytes < minimum {
		t.Fatalf("declared stage peak %d is below bounded copies %d", pythonLockStagePeakBytes, minimum)
	}
	if pythonLockStagePeakBytes > DiskJobPlanningCeilingBytes {
		t.Fatalf("declared stage peak %d exceeds per-job planning ceiling", pythonLockStagePeakBytes)
	}
}

func TestPythonLockStageGuardAbortsLiveGrowthAndAllowsCleanup(t *testing.T) {
	dir := t.TempDir()
	stage := filepath.Join(dir, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := runPythonLockStageGuarded(context.Background(), stage, 64<<10, 16, 16<<10,
		func(ctx context.Context) (string, error) {
			path := filepath.Join(stage, "growing.tmp")
			file, err := os.Create(path)
			if err != nil {
				return "", err
			}
			defer file.Close()
			chunk := make([]byte, 4<<10)
			for i := 0; i < 20; i++ {
				if _, err := file.Write(chunk); err != nil {
					return "", err
				}
				time.Sleep(40 * time.Millisecond)
				if ctx.Err() != nil {
					return "stopped", nil
				}
			}
			return "completed", nil
		})
	if err == nil || !strings.Contains(err.Error(), "live growth guard") || !strings.Contains(err.Error(), "above 16384-byte limit") {
		t.Fatalf("expected live per-file refusal, output=%q err=%v", output, err)
	}
	if output != "stopped" {
		t.Fatalf("resolver did not observe cancellation: %q", output)
	}
	(&PythonLockStage{Dir: stage}).Release()
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatalf("refused stage was not cleaned up: %v", err)
	}
}

func TestPythonLockStageGuardDeniesExcessEntriesBeforeResolver(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d", i)), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ran := false
	_, err := runPythonLockStageGuarded(context.Background(), dir, 1<<20, 2, 1<<10,
		func(context.Context) (string, error) {
			ran = true
			return "unexpected", nil
		})
	if err == nil || !strings.Contains(err.Error(), "exceeded 2 filesystem entries") {
		t.Fatalf("expected entry-count refusal, got %v", err)
	}
	if ran {
		t.Fatal("resolver ran despite an already oversized stage")
	}
}

func TestPythonLockStageGuardAbortsAggregateGrowthDuringResolver(t *testing.T) {
	dir := t.TempDir()
	stopped := false
	output, err := runPythonLockStageGuarded(context.Background(), dir, 32<<10, 16, 24<<10,
		func(ctx context.Context) (string, error) {
			chunk := make([]byte, 4<<10)
			for i := 0; i < 9; i++ {
				path := filepath.Join(dir, fmt.Sprintf("scratch-%d", i))
				if err := os.WriteFile(path, chunk, 0o600); err != nil {
					return "", err
				}
				time.Sleep(40 * time.Millisecond)
				if ctx.Err() != nil {
					stopped = true
					return "stopped", nil
				}
			}
			return "completed", nil
		})
	if err == nil || !strings.Contains(err.Error(), "live growth guard") || !strings.Contains(err.Error(), "beyond 32768-byte peak") {
		t.Fatalf("expected live aggregate refusal, output=%q err=%v", output, err)
	}
	if !stopped {
		t.Fatal("resolver completed without observing aggregate-growth cancellation")
	}
}

func TestOnnxGPUProviderReplacesCPUDistributionInput(t *testing.T) {
	input := "onnxruntime>=1.17\nonnx==1.19.0\n"
	got, err := removePythonRequirement(input, "onnxruntime")
	if err != nil {
		t.Fatal(err)
	}
	got, err = updatePythonInput(got, "onnxruntime-gpu", "1.27.0")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "onnxruntime>=") || !strings.Contains(got, "onnxruntime-gpu==1.27.0") {
		t.Fatalf("provider replacement not reflected in canonical input: %q", got)
	}
	lock := "onnxruntime==1.27.0 \\\n    --hash=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nonnx==1.19.0 \\\n    --hash=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"
	lock, err = removePythonRequirement(lock, "onnxruntime")
	if err != nil {
		t.Fatal(err)
	}
	lock, err = replacePythonRequirement(lock, "onnxruntime-gpu", "1.27.0")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(lock, "onnxruntime==") || strings.Contains(lock, "aaaaaaaa") || !strings.Contains(lock, "onnxruntime-gpu==1.27.0") {
		t.Fatalf("provider replacement not reflected in lock resolver input: %q", lock)
	}
}
