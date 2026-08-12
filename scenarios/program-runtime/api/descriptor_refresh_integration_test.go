package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"program-runtime/handlers/health"
	"program-runtime/internal/bindings"
	"program-runtime/internal/programs"
	"program-runtime/internal/testutil/mocks"
)

// TestProgramRuntimeDescriptorRefreshUpdatesRegistryKernelAndHealth exercises
// the complete no-restart contract at the HTTP/runner composition boundary:
// one descriptor publication advances the registry and health metadata, new
// kernels receive the new binding projection, and an existing kernel retains
// its original projection for request consistency.
func TestProgramRuntimeDescriptorRefreshUpdatesRegistryKernelAndHealth(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	descriptorPath := filepath.Join(dir, "image.binpb")
	original, err := os.ReadFile(filepath.Join(root, "packages", "proto", "gen", "descriptor", "image.binpb"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(descriptorPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, runtimeRefreshManifest(false), 0o644); err != nil {
		t.Fatal(err)
	}
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: descriptorPath, ManifestPaths: []string{manifestPath}})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := bindings.LoadFromSource(source)
	if err != nil {
		t.Fatal(err)
	}

	pinger := &mocks.FakePinger{}
	router := mux.NewRouter()
	health.ModuleWithDescriptor(pinger, "program-runtime-api", "1.0.0", registry.SkippedManifestCount, registry.SnapshotMetadata).Mount(router)
	healthServer := httptest.NewServer(router)
	t.Cleanup(healthServer.Close)

	runner := programs.NewSubprocessRunnerWithBindings(writeBindingCountKernel(t), nil, "")
	runner.SetBindingProvider(func() []programs.BindingSpec {
		current := registry.List("", "")
		out := make([]programs.BindingSpec, 0, len(current))
		for _, binding := range current {
			out = append(out, programs.BindingSpec{ID: binding.GetId(), Scenario: binding.GetScenario(), Group: binding.GetGroup(), Command: binding.GetCommand(), Effect: binding.GetEffect()})
		}
		return out
	})
	t.Cleanup(func() { _ = runner.Close() })

	initialHealth := getRuntimeHealth(t, healthServer.URL)
	initialDigest := metricString(t, initialHealth, "proto_descriptor_digest")
	initialGeneration := metricNumber(t, initialHealth, "proto_descriptor_generation")
	if got := len(registry.List("", "")); got != 1 {
		t.Fatalf("initial registry binding count = %d, want 1", got)
	}
	if got := runBindingCount(t, runner, "stable"); got != 1 {
		t.Fatalf("initial kernel binding count = %d, want 1", got)
	}

	updatedDescriptor := appendDescriptorFile(t, original)
	writeAtomic(t, descriptorPath, updatedDescriptor)
	writeAtomic(t, manifestPath, runtimeRefreshManifest(true))

	if got := len(registry.List("", "")); got != 2 {
		t.Fatalf("refreshed registry binding count = %d, want 2", got)
	}
	refreshedHealth := getRuntimeHealth(t, healthServer.URL)
	if got := metricString(t, refreshedHealth, "proto_descriptor_digest"); got == initialDigest {
		t.Fatalf("health digest did not change: %s", got)
	}
	if got := metricNumber(t, refreshedHealth, "proto_descriptor_generation"); got <= initialGeneration {
		t.Fatalf("health generation did not advance: initial=%v refreshed=%v", initialGeneration, got)
	}
	if got := runBindingCount(t, runner, "stable"); got != 1 {
		t.Fatalf("in-flight kernel binding count = %d, want original 1", got)
	}
	if got := runBindingCount(t, runner, "fresh"); got != 2 {
		t.Fatalf("new kernel binding count = %d, want refreshed 2", got)
	}

	// A malformed publication must not erase the last-known-good registry or
	// health surface, but the failed reload must remain observable.
	writeAtomic(t, descriptorPath, []byte("not a descriptor image"))
	if got := len(registry.List("", "")); got != 2 {
		t.Fatalf("registry lost last-known-good bindings after failed reload: %d", got)
	}
	failedHealth := getRuntimeHealth(t, healthServer.URL)
	if got := metricString(t, failedHealth, "proto_descriptor_digest"); got == "" || got != metricString(t, refreshedHealth, "proto_descriptor_digest") {
		t.Fatalf("health lost last-known-good descriptor digest: %q", got)
	}
	if got := metricString(t, failedHealth, "proto_descriptor_reload_error"); got == "" {
		t.Fatal("health did not expose the failed descriptor reload")
	}
}

func runtimeRefreshManifest(withDescribe bool) []byte {
	commands := `{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}`
	if withDescribe {
		commands += `,{"name":"describe","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"DescribeBinding"},"governance":{"effect":"read","run_eligible":true}}`
	}
	return []byte(fmt.Sprintf(`{"name":"program-runtime","groups":[{"name":"records","commands":[%s]}]}`, commands))
}

func writeBindingCountKernel(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "binding-count-kernel.py")
	source := `import json, os, sys
for line in sys.stdin:
    with open(os.environ["PROGRAM_RUNTIME_BINDINGS_FILE"], "r") as f:
        bindings = json.load(f)
    print(json.dumps({"ok": True, "stdout": str(len(bindings)) + "\n"}), flush=True)
`
	if err := os.WriteFile(path, []byte(source), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func runBindingCount(t *testing.T, runner *programs.SubprocessRunner, session string) int {
	t.Helper()
	result, err := runner.Execute(context.Background(), session, "print(1)", false)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(result.Stdout), "%d", &count); err != nil {
		t.Fatalf("decode kernel binding count %q: %v", result.Stdout, err)
	}
	return count
}

func getRuntimeHealth(t *testing.T, baseURL string) map[string]any {
	t.Helper()
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("health status = %d: %s", resp.StatusCode, raw)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func metricString(t *testing.T, body map[string]any, name string) string {
	t.Helper()
	metrics, ok := body["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("health metrics missing: %#v", body)
	}
	value, ok := metrics[name].(string)
	if !ok {
		t.Fatalf("health metric %q missing: %#v", name, metrics)
	}
	return value
}

func metricNumber(t *testing.T, body map[string]any, name string) float64 {
	t.Helper()
	metrics, ok := body["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("health metrics missing: %#v", body)
	}
	value, ok := metrics[name].(float64)
	if !ok {
		t.Fatalf("health metric %q missing: %#v", name, metrics)
	}
	return value
}

func appendDescriptorFile(t *testing.T, raw []byte) []byte {
	t.Helper()
	set := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(raw, set); err != nil {
		t.Fatal(err)
	}
	set.File = append(set.File, &descriptorpb.FileDescriptorProto{
		Name:    proto.String("descriptor-refresh/v1/refresh.proto"),
		Package: proto.String("descriptor_refresh.v1"),
		Syntax:  proto.String("proto3"),
	})
	updated, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func writeAtomic(t *testing.T, target string, raw []byte) {
	t.Helper()
	stage := target + ".stage"
	if err := os.WriteFile(stage, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(stage, target); err != nil {
		t.Fatal(err)
	}
}
