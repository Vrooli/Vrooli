package policy

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeRunner struct {
	name string
	args []string
	out  []byte
	err  error
}

const (
	fixtureEmbeddingModel      = "fixture-embed-model:latest"
	fixtureEmbeddingDimensions = 1234
)

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.name = name
	f.args = append([]string(nil), args...)
	return f.out, f.err
}

func TestResolveRoleShellsOutToPolicyCommand(t *testing.T) {
	runner := &fakeRunner{out: []byte(`{
		"schema_version":"2026-06-10",
		"policy_path":"/repo/resources/ollama/model-policy.json",
		"role":"embedding.default",
		"source":"role",
		"model":"fixture-embed-model:latest",
		"required_capabilities":["embedding"],
		"capabilities":["embedding"],
		"context_window_tokens":8192,
		"embedding_dimensions":1234,
		"disk_size_gb_estimate":0.3,
		"ram_gb_estimate":0.8,
		"vram_gb_estimate":0.8,
		"default_eligible":true
	}`)}
	resolved, err := (Resolver{Run: runner}).ResolveRole(context.Background(), "embedding.default")
	if err != nil {
		t.Fatalf("ResolveRole returned error: %v", err)
	}
	if runner.name != DefaultBin {
		t.Fatalf("command name = %q, want %q", runner.name, DefaultBin)
	}
	wantArgs := []string{"policy", "resolve", "--role", "embedding.default", "--json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", runner.args, wantArgs)
	}
	if resolved.Model != fixtureEmbeddingModel || resolved.EmbeddingDimensions != fixtureEmbeddingDimensions {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveModelShellsOutToPolicyCommand(t *testing.T) {
	runner := &fakeRunner{out: []byte(`{"model":"fixture-embed-model:latest","source":"model","capabilities":["embedding"]}`)}
	_, err := (Resolver{Bin: "custom-ollama", Run: runner}).ResolveModel(context.Background(), fixtureEmbeddingModel)
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}
	if runner.name != "custom-ollama" {
		t.Fatalf("command name = %q", runner.name)
	}
	wantArgs := []string{"policy", "resolve", "--model", fixtureEmbeddingModel, "--json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", runner.args, wantArgs)
	}
}

func TestResolveRoleRejectsBlankRole(t *testing.T) {
	if _, err := (Resolver{}).ResolveRole(context.Background(), " "); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveIncludesCommandOutputOnFailure(t *testing.T) {
	runner := &fakeRunner{out: []byte("unknown role"), err: errors.New("exit status 1")}
	_, err := (Resolver{Run: runner}).ResolveRole(context.Background(), "embedding.missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" || !containsAll(got, []string{"resource-ollama policy resolve --role embedding.missing --json", "unknown role"}) {
		t.Fatalf("error = %q", got)
	}
}

func TestResolveRejectsMalformedJSON(t *testing.T) {
	runner := &fakeRunner{out: []byte(`not json`)}
	_, err := (Resolver{Run: runner}).ResolveRole(context.Background(), "embedding.default")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveRejectsMissingModel(t *testing.T) {
	runner := &fakeRunner{out: []byte(`{"role":"embedding.default"}`)}
	_, err := (Resolver{Run: runner}).ResolveRole(context.Background(), "embedding.default")
	if err == nil {
		t.Fatal("expected error")
	}
}

func containsAll(s string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(s, needle) {
			return false
		}
	}
	return true
}
