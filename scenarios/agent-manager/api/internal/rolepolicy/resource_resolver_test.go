package rolepolicy

import (
	"context"
	"errors"
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
