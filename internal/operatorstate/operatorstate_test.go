package operatorstate

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func testService(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	return New(Config{RepoRoot: root, SchemaPath: filepath.Join(repoRoot, SchemaPath), Now: func() time.Time {
		return time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	}}), root
}

func TestApplyPreservesFieldsThisWriterDoesNotModel(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	first, err := service.Apply(ctx, []byte(`{"trust_posture":"shared","core":{"seed":["alpha"],"trusted_base":["alpha"]},"future_permission":{"enabled":true}}`))
	if err != nil {
		t.Fatalf("initial apply: %v", err)
	}
	if first.TrustPosture != "shared" || first.Core == nil {
		t.Fatalf("typed fields were not retained: %#v", first)
	}
	second, err := service.Apply(ctx, []byte(`{"scenarios":{"alpha":{"enabled":true}}}`))
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if _, ok := second.RawFields["future_permission"]; !ok {
		t.Fatal("future field was discarded")
	}
	state, err := service.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(state.RawFields["future_permission"]) != `{"enabled":true}` {
		t.Fatalf("future field bytes = %s", state.RawFields["future_permission"])
	}
}

func TestApplyRejectsInvalidPatchWithoutChangingStoredDocument(t *testing.T) {
	service, root := testService(t)
	ctx := context.Background()
	if _, err := service.Apply(ctx, []byte(`{"trust_posture":"personal"}`)); err != nil {
		t.Fatalf("initial apply: %v", err)
	}
	path := filepath.Join(root, ".vrooli", StateFile)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	if _, err := service.Apply(ctx, []byte(`{"trust_posture":"not-a-posture"}`)); err == nil || !strings.Contains(err.Error(), "/trust_posture") {
		t.Fatalf("invalid patch error = %v, want JSON path", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("invalid patch changed stored state")
	}
}

func TestHostWorkloadPostureDefaultsAndValidates(t *testing.T) {
	service, _ := testService(t)
	doc, err := service.Load(context.Background())
	if err != nil || doc.HostWorkloadPosture != "vrooli_only" { t.Fatalf("default posture = %q, err=%v", doc.HostWorkloadPosture, err) }
	if _, err := service.Apply(context.Background(), []byte(`{"host_workload_posture":"operator_machine"}`)); err == nil || !strings.Contains(err.Error(), "/host_workload_posture") { t.Fatalf("invalid posture error = %v", err) }
}

func TestDisjointConcurrentPatchesBothLand(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	var group sync.WaitGroup
	errs := make(chan error, 2)
	for _, patch := range []string{`{"scenarios":{"alpha":{"enabled":true}}}`, `{"resources":{"ollama":{"enabled":true}}}`} {
		group.Add(1)
		go func(patch string) {
			defer group.Done()
			_, err := service.Apply(ctx, []byte(patch))
			errs <- err
		}(patch)
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent apply: %v", err)
		}
	}
	doc, err := service.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Scenarios["alpha"].Enabled == nil || !*doc.Scenarios["alpha"].Enabled || doc.Resources["ollama"].Enabled == nil || !*doc.Resources["ollama"].Enabled {
		t.Fatalf("disjoint patches did not both land: %#v", doc)
	}
}

func TestOnlyOperatorStatePackageWritesTheOperatorState(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	var writers []string
	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(data)
		if strings.Contains(source, "operator-state.json") && (strings.Contains(source, "WriteFile") || strings.Contains(source, "os.Create") || strings.Contains(source, "os.OpenFile")) && !strings.HasSuffix(path, filepath.Join("internal", "operatorstate", "operatorstate.go")) {
			writers = append(writers, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan source: %v", err)
	}
	if len(writers) != 0 {
		t.Fatalf("operator-state path has writers outside operatorstate: %v", writers)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "internal", "operatorstate", "operatorstate.go"))
	if err != nil {
		t.Fatalf("read operatorstate writer: %v", err)
	}
	if !strings.Contains(string(data), "storage.WriteFileAtomic") {
		t.Fatal("operatorstate package does not contain the atomic writer")
	}
}
