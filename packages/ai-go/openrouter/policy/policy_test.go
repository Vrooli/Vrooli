package policy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.name = name
	f.args = append([]string(nil), args...)
	return f.out, f.err
}

func TestResolveRoleShellsOutToPolicyCommand(t *testing.T) {
	runner := &fakeRunner{out: []byte(`{
		"schema_version":"2026-06-30",
		"policy_path":"/repo/resources/openrouter/model-policy.json",
		"role":"image.generate.logo",
		"source":"role",
		"model":"recraft/recraft-v4.1-vector",
		"endpoint":"images",
		"fallbacks":["recraft/recraft-v4.1-pro-vector"],
		"required_capabilities":["image_output"],
		"capabilities":["text_input","image_output","svg_output"],
		"modalities":{"input":["text"],"output":["image","vector"]},
		"request_defaults":{"output_format":"svg","background":"transparent"},
		"default_eligible":true
	}`)}
	resolved, err := (Resolver{Run: runner}).ResolveRole(context.Background(), "image.generate.logo")
	if err != nil {
		t.Fatalf("ResolveRole: %v", err)
	}
	if runner.name != DefaultBin {
		t.Fatalf("command name = %q", runner.name)
	}
	wantArgs := []string{"policy", "resolve", "--role", "image.generate.logo", "--json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v", runner.args)
	}
	if resolved.Model != "recraft/recraft-v4.1-vector" || resolved.Endpoint != "images" {
		t.Fatalf("resolved = %#v", resolved)
	}
	if resolved.RequestDefaults == nil || resolved.RequestDefaults.OutputFormat != "svg" {
		t.Fatalf("request_defaults = %#v", resolved.RequestDefaults)
	}
	cands := resolved.ModelCandidates()
	want := []string{"recraft/recraft-v4.1-vector", "recraft/recraft-v4.1-pro-vector"}
	if !reflect.DeepEqual(cands, want) {
		t.Fatalf("candidates = %#v", cands)
	}
}

func TestResolveModelShellsOutToPolicyCommand(t *testing.T) {
	runner := &fakeRunner{out: []byte(`{"model":"vendor/x","source":"model","endpoint":"chat","capabilities":["chat"]}`)}
	_, err := (Resolver{Bin: "custom-openrouter", Run: runner}).ResolveModel(context.Background(), "vendor/x")
	if err != nil {
		t.Fatalf("ResolveModel: %v", err)
	}
	if runner.name != "custom-openrouter" {
		t.Fatalf("command name = %q", runner.name)
	}
	wantArgs := []string{"policy", "resolve", "--model", "vendor/x", "--json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v", runner.args)
	}
}

func TestResolveRoleRejectsBlank(t *testing.T) {
	if _, err := (Resolver{}).ResolveRole(context.Background(), "  "); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveIncludesCommandOutputOnFailure(t *testing.T) {
	runner := &fakeRunner{out: []byte("unknown role"), err: errors.New("exit status 1")}
	_, err := (Resolver{Run: runner}).ResolveRole(context.Background(), "missing.role")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, needle := range []string{"resource-openrouter policy resolve --role missing.role --json", "unknown role"} {
		if !strings.Contains(err.Error(), needle) {
			t.Fatalf("error %q missing %q", err.Error(), needle)
		}
	}
}

func TestResolveRejectsMalformedJSON(t *testing.T) {
	runner := &fakeRunner{out: []byte("not json")}
	if _, err := (Resolver{Run: runner}).ResolveRole(context.Background(), "chat.default"); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveRejectsMissingModel(t *testing.T) {
	runner := &fakeRunner{out: []byte(`{"role":"chat.default"}`)}
	if _, err := (Resolver{Run: runner}).ResolveRole(context.Background(), "chat.default"); err == nil {
		t.Fatal("expected error")
	}
}

func TestExecRunnerIgnoresStderrNoticesOnSuccess(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "resource-cli")
	script := "#!/bin/sh\necho '⚠ notice: rebuilt resource CLI' >&2\necho '{\"model\":\"m:latest\"}'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := (Resolver{Bin: bin}).ResolveRole(context.Background(), "embedding.default")
	if err != nil {
		t.Fatalf("ResolveRole with stderr notice: %v", err)
	}
	if resolved.Model != "m:latest" {
		t.Fatalf("model = %q, want m:latest", resolved.Model)
	}
}

func TestExecRunnerReportsStderrOnFailure(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "resource-cli")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 'unknown model role' >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := (Resolver{Bin: bin}).ResolveRole(context.Background(), "nope")
	if err == nil || !strings.Contains(err.Error(), "unknown model role") {
		t.Fatalf("error = %v, want stderr text", err)
	}
}
