package policy

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
	"workspace-sandbox/internal/types"
)

func testSandbox() *types.Sandbox {
	return &types.Sandbox{
		ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Owner:       "agent-one",
		ScopePath:   "src",
		ProjectRoot: "/workspace/project",
		UpperDir:    "/workspace/upper",
		MergedDir:   "/workspace/merged",
	}
}

func TestDefaultAttributionPolicy_GetCommitAuthor(t *testing.T) {
	sandbox := testSandbox()
	cases := []struct {
		name  string
		mode  string
		actor string
		want  string
	}{
		{name: "agent", mode: "agent", want: "agent-one <noreply@workspace-sandbox.local>"},
		{name: "reviewer", mode: "reviewer", actor: "reviewer", want: "reviewer <noreply@workspace-sandbox.local>"},
		{name: "coauthored", mode: "coauthored", want: "agent-one <noreply@workspace-sandbox.local>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewDefaultAttributionPolicy(config.PolicyConfig{
				CommitMessageTemplate: "Apply sandbox changes",
				CommitAuthorMode:      tc.mode,
			})
			if err != nil {
				t.Fatalf("NewDefaultAttributionPolicy: %v", err)
			}
			if got := p.GetCommitAuthor(context.Background(), sandbox, tc.actor); got != tc.want {
				t.Errorf("GetCommitAuthor() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDefaultAttributionPolicy_GetCommitMessage(t *testing.T) {
	p, err := NewDefaultAttributionPolicy(config.PolicyConfig{
		CommitMessageTemplate: "Apply {{.FileCount}} files in {{.ScopePath}}",
		CommitAuthorMode:      "agent",
	})
	if err != nil {
		t.Fatalf("NewDefaultAttributionPolicy: %v", err)
	}
	sandbox := testSandbox()
	changes := []*types.FileChange{{FilePath: "a.go"}, {FilePath: "b.go"}}
	if got := p.GetCommitMessage(context.Background(), sandbox, changes, ""); got != "Apply 2 files in src" {
		t.Errorf("generated message = %q", got)
	}
	if got := p.GetCommitMessage(context.Background(), sandbox, changes, "operator message"); got != "operator message" {
		t.Errorf("user message = %q", got)
	}
}

func TestDefaultAttributionPolicy_GetCoAuthors(t *testing.T) {
	sandbox := testSandbox()
	p, err := NewDefaultAttributionPolicy(config.PolicyConfig{
		CommitMessageTemplate: "Apply sandbox changes",
		CommitAuthorMode:      "coauthored",
	})
	if err != nil {
		t.Fatalf("NewDefaultAttributionPolicy: %v", err)
	}
	want := []string{"Co-authored-by: reviewer <noreply@workspace-sandbox.local>"}
	if got := p.GetCoAuthors(context.Background(), sandbox, "reviewer"); !reflect.DeepEqual(got, want) {
		t.Errorf("GetCoAuthors() = %v, want %v", got, want)
	}
	if got := p.GetCoAuthors(context.Background(), sandbox, sandbox.Owner); got != nil {
		t.Errorf("owner should not be added as co-author: %v", got)
	}
}

func TestFormatCommitMessage(t *testing.T) {
	if got := FormatCommitMessage("message", nil); got != "message" {
		t.Errorf("without co-authors = %q", got)
	}
	got := FormatCommitMessage("message", []string{"Co-authored-by: reviewer <r@example.com>"})
	if got != "message\n\nCo-authored-by: reviewer <r@example.com>" {
		t.Errorf("with co-authors = %q", got)
	}
}

func TestNewHookValidationPolicy_EmptyHooks(t *testing.T) {
	p := NewHookValidationPolicy(procmocks.NewFakeStarter(), nil)
	if got := p.GetValidationHooks(); len(got) != 0 {
		t.Fatalf("empty hooks = %v", got)
	}
}

func TestNewHookValidationPolicy_WithHooks(t *testing.T) {
	hooks := []ValidationHook{{Name: "lint", Command: "lint", Required: true}}
	p := NewHookValidationPolicy(procmocks.NewFakeStarter(), hooks)
	if got := p.GetValidationHooks(); !reflect.DeepEqual(got, hooks) {
		t.Fatalf("configured hooks = %v, want %v", got, hooks)
	}
}

func TestHookValidationPolicy_GetValidationHooks(t *testing.T) {
	hooks := []ValidationHook{{Name: "test", Command: "test"}}
	p := NewHookValidationPolicy(procmocks.NewFakeStarter(), hooks)
	if len(p.GetValidationHooks()) != 1 || p.GetValidationHooks()[0].Name != "test" {
		t.Fatalf("GetValidationHooks() = %v", p.GetValidationHooks())
	}
}

func TestHookValidationPolicy_ValidateBeforeApply(t *testing.T) {
	sandbox := testSandbox()
	changes := []*types.FileChange{{FilePath: "src/main.go"}}

	t.Run("passes successful required hook", func(t *testing.T) {
		starter := procmocks.NewFakeStarter()
		starter.AddCommand("check", procmocks.CommandBehavior{Stdout: []byte("ok")})
		p := NewHookValidationPolicy(starter, []ValidationHook{{Name: "check", Command: "check", Required: true}})
		if err := p.ValidateBeforeApply(context.Background(), sandbox, changes); err != nil {
			t.Fatalf("ValidateBeforeApply: %v", err)
		}
		calls := starter.MatchedCalls("check")
		if len(calls) != 1 || calls[0].Dir != sandbox.MergedDir {
			t.Fatalf("hook call = %v", calls)
		}
	})

	t.Run("returns typed error for failed required hook", func(t *testing.T) {
		starter := procmocks.NewFakeStarter()
		starter.AddCommand("check", procmocks.CommandBehavior{
			Exit:   process.ProcessExit{ExitCode: 3},
			Stderr: []byte("lint failed"),
		})
		p := NewHookValidationPolicy(starter, []ValidationHook{{Name: "check", Command: "check", Required: true}})
		var hookErr *ValidationHookError
		if err := p.ValidateBeforeApply(context.Background(), sandbox, changes); !errors.As(err, &hookErr) {
			t.Fatalf("error = %v, want ValidationHookError", err)
		}
		if hookErr.HookName != "check" || !strings.Contains(hookErr.Output, "lint failed") {
			t.Errorf("typed error = %+v", hookErr)
		}
	})
}

func TestValidationHookError(t *testing.T) {
	underlying := errors.New("boom")
	err := &ValidationHookError{HookName: "lint", Output: "details", Err: underlying}
	if !strings.Contains(err.Error(), "lint") || !errors.Is(err, underlying) {
		t.Errorf("error = %q, unwrap = %v", err.Error(), errors.Unwrap(err))
	}
	var domainErr types.DomainError = err
	if domainErr.HTTPStatus() != 422 || domainErr.IsRetryable() {
		t.Errorf("domain error status/retry = %d/%v", domainErr.HTTPStatus(), domainErr.IsRetryable())
	}
	if err.Hint() == "" {
		t.Error("Hint() must be actionable")
	}
}

func TestBuildHookEnv(t *testing.T) {
	sandbox := testSandbox()
	env := buildHookEnv(sandbox, []*types.FileChange{{FilePath: "a.go"}, {FilePath: "b.go"}})
	values := make(map[string]string, len(env))
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		values[parts[0]] = parts[1]
	}
	for key, want := range map[string]string{
		"SANDBOX_ID":            sandbox.ID.String(),
		"SANDBOX_SCOPE_PATH":    sandbox.ScopePath,
		"SANDBOX_PROJECT_ROOT":  sandbox.ProjectRoot,
		"SANDBOX_UPPER_DIR":     sandbox.UpperDir,
		"SANDBOX_MERGED_DIR":    sandbox.MergedDir,
		"SANDBOX_CHANGE_COUNT":  "2",
		"SANDBOX_CHANGED_FILES": "a.go,b.go",
	} {
		if values[key] != want {
			t.Errorf("%s = %q, want %q", key, values[key], want)
		}
	}
}
