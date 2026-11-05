package rendering

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"device-studio/internal/geometry"
)

func TestGeometryCompileManifestProducesOnlyBoundedCanonicalConstruct(t *testing.T) {
	workspace := t.TempDir()
	recipe := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{{ID: "housing", Kind: "rounded_box", Role: "housing", Size: [3]float64{78, 148, 10}, Bevel: 1, Material: "matte_polymer"}, {ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5.2}, Size: [3]float64{64, 124, 1}, Material: "glass"}}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	recipeBytes, err := json.Marshal(recipe)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, GeometryRecipeName), append(recipeBytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executableBytes, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(executableBytes)
	manifest := GeometryWorkerManifest{Version: 1, JobID: "01234567-89ab-cdef-0123-456789abcdef", AttemptID: "11234567-89ab-cdef-0123-456789abcdef", WorkerSHA256: hex.EncodeToString(digest[:]), Envelope: recipe.Envelope}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, GeometryManifestName), append(manifestBytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RunGeometryCompileManifest(context.Background(), filepath.Join(workspace, GeometryManifestName)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace, GeometryOutputName)); err != nil {
		t.Fatalf("bounded GLB was not written: %v", err)
	}
}

func TestGeometryCompileSandboxRequiresNetworkAndWriteContainment(t *testing.T) {
	workspace := t.TempDir()
	recipe := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{{ID: "housing", Kind: "rounded_box", Role: "housing", Size: [3]float64{78, 148, 10}, Bevel: 1, Material: "matte_polymer"}, {ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5.2}, Size: [3]float64{64, 124, 1}, Material: "glass"}}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	glb, err := geometry.Compile(recipe)
	if err != nil {
		t.Fatal(err)
	}
	var deleteCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes":
			var body geometrySandboxCreate
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.ProjectRoot != workspace || body.ScopePath != workspace || body.NoLock == nil || !*body.NoLock {
				t.Errorf("unexpected sandbox scope: %+v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(geometrySandbox{ID: "21234567-89ab-cdef-0123-456789abcdef", WorkspacePath: workspace, Containment: &geometryContainment{Backend: "bwrap", Enforcements: []string{"filesystem-write-containment", "network-deny"}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/exec"):
			var body geometrySandboxExec
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.IsolationLevel != "full" || body.ExecutionMode != "protected" || body.AllowNetwork == nil || *body.AllowNetwork || body.Command != "/workspace/worker" || body.WorkingDir != "/workspace" || len(body.Args) != 3 || body.Args[0] != "geometry-compile-worker" || body.Args[2] != "/workspace/"+GeometryManifestName || body.Env["GOMAXPROCS"] != "2" || body.Env["GOMEMLIMIT"] != "256MiB" || body.MemoryLimitMB != 4096 || body.CPUTimeSec != 45 || body.TimeoutSec != 60 || body.MaxProcesses != 32 || body.MaxOpenFiles != 64 {
				t.Errorf("unsafe compiler invocation: %+v", body)
			}
			_ = json.NewEncoder(w).Encode(geometrySandboxExecResult{ExitCode: 0, Containment: &geometryContainment{Backend: "bwrap", Enforcements: []string{"filesystem-write-containment"}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/processes/kill-all"):
			_ = json.NewEncoder(w).Encode(geometrySandboxKillAllResult{})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/processes"):
			_ = json.NewEncoder(w).Encode(geometrySandboxProcesses{})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/files/content"):
			_ = json.NewEncoder(w).Encode(geometrySandboxFile{Encoding: "base64", Content: base64.StdEncoding.EncodeToString(glb), Size: int64(len(glb))})
		case r.Method == http.MethodDelete:
			deleteCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected workspace-sandbox request %s %s", r.Method, r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	oldResolve := resolveGeometrySandbox
	resolveGeometrySandbox = func(context.Context) (string, error) { return server.URL, nil }
	defer func() { resolveGeometrySandbox = oldResolve }()
	err = runGeometryCompileSandbox(context.Background(), workspace, GeometryWorkerManifest{AttemptID: "31234567-89ab-cdef-0123-456789abcdef"})
	if err == nil || !strings.Contains(err.Error(), "did not enforce") {
		t.Fatalf("error=%v, want fail-closed containment error", err)
	}
	if !deleteCalled {
		t.Fatal("sandbox was not deleted after containment failure")
	}
	if _, err := os.Stat(filepath.Join(workspace, GeometryOutputName)); !os.IsNotExist(err) {
		t.Fatalf("output copied despite failed containment: %v", err)
	}
}

func TestCleanupGeometrySandboxStopsAndObservesBeforeDelete(t *testing.T) {
	var runningReads int
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/processes/kill-all"):
			_ = json.NewEncoder(w).Encode(geometrySandboxKillAllResult{Killed: 1})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/processes"):
			runningReads++
			if runningReads == 1 {
				_ = json.NewEncoder(w).Encode(geometrySandboxProcesses{Running: 1})
				return
			}
			_ = json.NewEncoder(w).Encode(geometrySandboxProcesses{})
		case r.Method == http.MethodDelete:
			if runningReads < 2 {
				t.Error("sandbox deleted without checking process quiescence")
			}
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected cleanup request %s %s", r.Method, r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := &http.Client{Timeout: time.Second}
	if err := cleanupGeometrySandbox(context.Background(), client, server.URL, "21234567-89ab-cdef-0123-456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("sandbox was not deleted after process quiescence")
	}
}

func TestGeometryCompileSandboxCleansCreatedScopeWhenWorkspaceMetadataIsInvalid(t *testing.T) {
	workspace := t.TempDir()
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(geometrySandbox{ID: "21234567-89ab-cdef-0123-456789abcdef"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/processes/kill-all"):
			_ = json.NewEncoder(w).Encode(geometrySandboxKillAllResult{})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/processes"):
			_ = json.NewEncoder(w).Encode(geometrySandboxProcesses{})
		case r.Method == http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected cleanup request %s %s", r.Method, r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	oldResolve := resolveGeometrySandbox
	resolveGeometrySandbox = func(context.Context) (string, error) { return server.URL, nil }
	defer func() { resolveGeometrySandbox = oldResolve }()
	err := runGeometryCompileSandbox(context.Background(), workspace, GeometryWorkerManifest{AttemptID: "31234567-89ab-cdef-0123-456789abcdef"})
	if err == nil || !strings.Contains(err.Error(), "no workspace path") {
		t.Fatalf("error=%v, want invalid-workspace-metadata refusal", err)
	}
	if !deleted {
		t.Fatal("sandbox created with malformed metadata was leaked instead of cleaned")
	}
}

func TestCleanupGeometrySandboxFailureDoesNotDeleteUnquiescentSandbox(t *testing.T) {
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/processes/kill-all"):
			_ = json.NewEncoder(w).Encode(geometrySandboxKillAllResult{Errors: []string{"permission denied"}})
		case r.Method == http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected cleanup request %s %s", r.Method, r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := &http.Client{Timeout: time.Second}
	err := cleanupGeometrySandbox(context.Background(), client, server.URL, "21234567-89ab-cdef-0123-456789abcdef")
	if err == nil || !strings.Contains(err.Error(), "1 error(s)") {
		t.Fatalf("cleanup error=%v, want retained stop failure", err)
	}
	if deleted {
		t.Fatal("sandbox deleted despite process stop failure")
	}
}

func TestGeometryCompileSandboxLiveSmoke(t *testing.T) {
	worker := os.Getenv("DEVICE_STUDIO_GEOMETRY_SMOKE_WORKER")
	if worker == "" {
		t.Skip("set DEVICE_STUDIO_GEOMETRY_SMOKE_WORKER to a freshly built Device Studio API executable")
	}
	workerBytes, err := os.ReadFile(worker)
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "worker"), workerBytes, 0o500); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(workerBytes)
	recipe := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{{ID: "housing", Kind: "rounded_box", Role: "housing", Size: [3]float64{78, 148, 10}, Bevel: 1, Material: "matte_polymer"}, {ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5.2}, Size: [3]float64{64, 124, 1}, Material: "glass"}}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	recipeBytes, err := json.Marshal(recipe)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, GeometryRecipeName), append(recipeBytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := GeometryWorkerManifest{Version: 1, JobID: "41234567-89ab-cdef-0123-456789abcdef", AttemptID: "51234567-89ab-cdef-0123-456789abcdef", WorkerSHA256: hex.EncodeToString(digest[:]), Envelope: recipe.Envelope}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, GeometryManifestName), append(manifestBytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGeometryCompileSandbox(context.Background(), workspace, manifest); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(filepath.Join(workspace, GeometryOutputName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InspectPhysicalGLB(result); err != nil {
		t.Fatalf("sandbox output is not a valid physical GLB: %v", err)
	}
}
