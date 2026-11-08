package rolepolicy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

type fakeCommandExecutor struct {
	command string
	args    []string
	output  []byte
	err     error
}

func (f *fakeCommandExecutor) Run(_ context.Context, command string, args ...string) ([]byte, error) {
	f.command = command
	f.args = append([]string(nil), args...)
	return f.output, f.err
}

func TestResourceRoleResolverResolvesStrictResourceEvidence(t *testing.T) {
	executor := &fakeCommandExecutor{output: []byte(`{
  "schema_version":"v1", "runner":"codex", "role":"code.default", "model":"gpt-5.4",
  "fallbacks":["gpt-5.5"], "description":"Balanced coding", "capabilities":["code","tools"],
  "provenance":{"source":"Codex catalog","observed_at":"2026-07-10"},
  "enforcement":{"permissions":"intent_only","caveats":["No native command enforcement"]},
  "policy_path":"/catalog/model-policy.json", "policy_digest":"abc123"
}`)}

	resolved, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.default")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if executor.command != "resource-codex" {
		t.Fatalf("command = %q, want resource-codex", executor.command)
	}
	if want := []string{"policy", "resolve", "--role", "code.default", "--json"}; !reflect.DeepEqual(executor.args, want) {
		t.Fatalf("args = %#v, want %#v", executor.args, want)
	}
	if resolved.Runner != domain.RunnerTypeCodex || resolved.Model != "gpt-5.4" || resolved.Enforcement.Permissions != "intent_only" {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResourceRoleResolverCarriesConfiguredModelExclusions(t *testing.T) {
	response := strings.Replace(string(validResponse("\"runner\":\"codex\"")), `"policy_path":"/catalog"`, `"excluded_models":["blocked-model","vendor/blocked"],"policy_path":"/catalog"`, 1)
	executor := &fakeCommandExecutor{output: []byte(response)}
	resolved, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.default")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !reflect.DeepEqual(resolved.ExcludedModels, []string{"blocked-model", "vendor/blocked"}) {
		t.Fatalf("excluded models = %#v", resolved.ExcludedModels)
	}
}

func TestResourceRoleResolverAcceptsUnverifiedHookPosture(t *testing.T) {
	response := strings.Replace(string(validResponse(`"runner":"codex"`)), `"permissions":"intent_only"`, `"permissions":"hook_unverified"`, 1)
	executor := &fakeCommandExecutor{output: []byte(response)}
	resolved, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.default")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Enforcement.Permissions != "hook_unverified" {
		t.Fatalf("permissions = %q, want hook_unverified", resolved.Enforcement.Permissions)
	}
}

func TestResourceRoleResolverAcceptsVerifiedHookPosture(t *testing.T) {
	response := string(validResponse(`"runner":"claude-code"`))
	response = strings.Replace(response, `"role":"code.default"`, `"role":"code.judgment"`, 1)
	response = strings.Replace(response, `"permissions":"intent_only"`, `"permissions":"hook_verified"`, 1)
	executor := &fakeCommandExecutor{output: []byte(response)}
	resolved, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeClaudeCode, "code.judgment")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Enforcement.Permissions != "hook_verified" {
		t.Fatalf("permissions = %q, want hook_verified", resolved.Enforcement.Permissions)
	}
}

func TestResourceRoleResolverRejectsMismatchedIdentity(t *testing.T) {
	executor := &fakeCommandExecutor{output: validResponse(`"runner":"claude-code"`)}
	_, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.default")
	if !errors.Is(err, ErrInvalidResourceResponse) || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("Resolve error = %v, want invalid mismatched response", err)
	}
}

func TestResourceRoleResolverClassifiesUnknownRole(t *testing.T) {
	executor := &fakeCommandExecutor{output: []byte("unknown coding role \"code.future\""), err: errors.New("exit status 1")}
	_, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.future")
	if !errors.Is(err, ErrUnknownResourceRole) {
		t.Fatalf("Resolve error = %v, want ErrUnknownResourceRole", err)
	}
}

func TestResourceRoleResolverClassifiesUnavailableResource(t *testing.T) {
	executor := &fakeCommandExecutor{err: errors.New("executable file not found")}
	_, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.default")
	if !errors.Is(err, ErrResourceUnavailable) {
		t.Fatalf("Resolve error = %v, want ErrResourceUnavailable", err)
	}
}

func TestResourceRoleResolverPreservesCommandDiagnostic(t *testing.T) {
	executor := &fakeCommandExecutor{
		output: []byte(`Error: parse coding role policy: json: unknown field "model_aliases"`),
		err:    errors.New("exit status 1"),
	}
	_, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.smart")
	if !errors.Is(err, ErrResourceUnavailable) {
		t.Fatalf("Resolve error = %v, want ErrResourceUnavailable", err)
	}
	if !strings.Contains(err.Error(), `unknown field "model_aliases"`) {
		t.Fatalf("Resolve error = %v, want resource diagnostic", err)
	}
}

func TestResourceRoleResolverRejectsUnknownResponseFields(t *testing.T) {
	executor := &fakeCommandExecutor{output: validResponse(`"unexpected":true`)}
	_, err := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.default")
	if !errors.Is(err, ErrInvalidResourceResponse) || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Resolve error = %v, want strict JSON rejection", err)
	}
}

func validResponse(replacement string) []byte {
	response := `{"schema_version":"v1","runner":"codex","role":"code.default","model":"gpt-5.4","description":"Balanced coding","capabilities":["code"],"provenance":{"source":"catalog","observed_at":"2026-07-10"},"enforcement":{"permissions":"intent_only"},"policy_path":"/catalog","policy_digest":"digest"}`
	return []byte(strings.Replace(response, `"runner":"codex"`, replacement, 1))
}

func TestResourceRoleResolverReadsThePolicyFileTheResourceReports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(path, []byte(`{"runner":"codex"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	response := fmt.Sprintf(`{"schema_version":"v1","runner":"codex","role":"code.default","model":"m","description":"d","capabilities":["code"],"provenance":{"source":"s","observed_at":"2026-07-10"},"enforcement":{"permissions":"intent_only"},"policy_path":%q,"policy_digest":"d"}`, path)
	executor := &fakeCommandExecutor{output: []byte(response)}
	raw, err := NewResourceRoleResolver(executor).PolicyFile(context.Background(), domain.RunnerTypeCodex)
	if err != nil || string(raw) != `{"runner":"codex"}` {
		t.Fatalf("PolicyFile = %q, %v", raw, err)
	}
	failing := &fakeCommandExecutor{output: []byte("parse error"), err: errors.New("exit 1")}
	if _, err := NewResourceRoleResolver(failing).PolicyFile(context.Background(), domain.RunnerTypeCodex); !errors.Is(err, ErrResourceUnavailable) {
		t.Fatalf("an unavailable resource must not fall back to a guessed path: %v", err)
	}
}

// Exercise the production subprocess seam rather than returning a precombined
// fake result: successful CLI diagnostics belong on stderr, never in JSON.
func TestCommandExecutorResourceResponseStreams(t *testing.T) {
	valid := string(validResponse(`"runner":"codex"`))
	for _, tc := range []struct {
		name, stdout, stderr string
		exit                 int
		wantErr              error
		wantDiagnostic       string
	}{
		{name: "success with rebuild diagnostics", stdout: valid, stderr: "resource-codex CLI rebuilt from current sources; restarting command...\n"},
		{name: "unknown role on stderr", stdout: "partial output", stderr: "unknown coding role code.default", exit: 1, wantErr: ErrUnknownResourceRole, wantDiagnostic: "unknown coding role code.default"},
		{name: "unavailable diagnostic on stderr", stdout: "partial output", stderr: "catalog is unavailable", exit: 2, wantErr: ErrResourceUnavailable, wantDiagnostic: "catalog is unavailable"},
		{name: "malformed stdout stays invalid", stdout: "banner\n" + valid, stderr: "diagnostic", wantErr: ErrInvalidResourceResponse},
		{name: "JSON on stderr cannot replace stdout", stdout: "not JSON", stderr: valid, wantErr: ErrInvalidResourceResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
			script := fmt.Sprintf("#!/bin/sh\nprintf '%%s' %s\nprintf '%%s' %s >&2\nexit %d\n", quote(tc.stdout), quote(tc.stderr), tc.exit)
			if err := os.WriteFile(filepath.Join(bin, "resource-codex"), []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PROJECT_ROOT", bin)
			if tc.wantErr == nil {
				output, err := (commandExecutor{}).Run(context.Background(), "resource-codex", "policy", "resolve", "--role", "code.default", "--json")
				if err != nil || string(output) != tc.stdout {
					t.Fatalf("successful stdout = %q, %v; want exact JSON stdout", output, err)
				}
			}
			resolved, err := NewResourceRoleResolver(nil).Resolve(context.Background(), domain.RunnerTypeCodex, "code.default")
			if tc.wantErr == nil {
				if err != nil || resolved.Model != "gpt-5.4" {
					t.Fatalf("Resolve = %#v, %v", resolved, err)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Resolve error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantDiagnostic != "" && (err == nil || !strings.Contains(err.Error(), tc.wantDiagnostic) || !strings.Contains(err.Error(), "partial output")) {
				t.Fatalf("lost command stderr/stdout diagnostic: %v", err)
			}
		})
	}
}
