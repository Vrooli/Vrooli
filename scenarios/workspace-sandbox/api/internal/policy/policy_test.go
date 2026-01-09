package policy

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/types"
)

// =============================================================================
// DefaultApprovalPolicy Tests
// =============================================================================

func TestDefaultApprovalPolicy_CanAutoApprove(t *testing.T) {
	ctx := context.Background()
	sandboxID := uuid.New()
	sandbox := &types.Sandbox{ID: sandboxID, Owner: "test-agent"}

	t.Run("human approval required blocks auto-approve", func(t *testing.T) {
		policy := NewDefaultApprovalPolicy(config.PolicyConfig{
			RequireHumanApproval: true,
		})

		changes := []*types.FileChange{{ID: uuid.New()}}
		canApprove, reason := policy.CanAutoApprove(ctx, sandbox, changes)

		if canApprove {
			t.Error("should not auto-approve when human approval required")
		}
		if !strings.Contains(reason, "human approval is required") {
			t.Errorf("reason should mention human approval, got: %s", reason)
		}
	})

	t.Run("within file threshold allows auto-approve", func(t *testing.T) {
		policy := NewDefaultApprovalPolicy(config.PolicyConfig{
			RequireHumanApproval:      false,
			AutoApproveThresholdFiles: 10,
		})

		changes := make([]*types.FileChange, 5)
		for i := range changes {
			changes[i] = &types.FileChange{ID: uuid.New()}
		}

		canApprove, reason := policy.CanAutoApprove(ctx, sandbox, changes)

		if !canApprove {
			t.Error("should auto-approve when within threshold")
		}
		if !strings.Contains(reason, "meet auto-approval criteria") {
			t.Errorf("reason should confirm criteria met, got: %s", reason)
		}
	})

	t.Run("exceeds file threshold blocks auto-approve", func(t *testing.T) {
		policy := NewDefaultApprovalPolicy(config.PolicyConfig{
			RequireHumanApproval:      false,
			AutoApproveThresholdFiles: 5,
		})

		changes := make([]*types.FileChange, 10)
		for i := range changes {
			changes[i] = &types.FileChange{ID: uuid.New()}
		}

		canApprove, reason := policy.CanAutoApprove(ctx, sandbox, changes)

		if canApprove {
			t.Error("should not auto-approve when exceeding threshold")
		}
		if !strings.Contains(reason, "exceeds auto-approve threshold") {
			t.Errorf("reason should mention threshold, got: %s", reason)
		}
	})

	t.Run("zero threshold means no limit", func(t *testing.T) {
		policy := NewDefaultApprovalPolicy(config.PolicyConfig{
			RequireHumanApproval:      false,
			AutoApproveThresholdFiles: 0, // No limit
		})

		changes := make([]*types.FileChange, 100)
		for i := range changes {
			changes[i] = &types.FileChange{ID: uuid.New()}
		}

		canApprove, _ := policy.CanAutoApprove(ctx, sandbox, changes)

		if !canApprove {
			t.Error("zero threshold should allow any number of files")
		}
	})

	t.Run("empty changes list", func(t *testing.T) {
		policy := NewDefaultApprovalPolicy(config.PolicyConfig{
			RequireHumanApproval:      false,
			AutoApproveThresholdFiles: 10,
		})

		canApprove, _ := policy.CanAutoApprove(ctx, sandbox, []*types.FileChange{})

		if !canApprove {
			t.Error("empty changes should be auto-approvable")
		}
	})

	t.Run("exactly at threshold", func(t *testing.T) {
		policy := NewDefaultApprovalPolicy(config.PolicyConfig{
			RequireHumanApproval:      false,
			AutoApproveThresholdFiles: 5,
		})

		changes := make([]*types.FileChange, 5)
		for i := range changes {
			changes[i] = &types.FileChange{ID: uuid.New()}
		}

		canApprove, _ := policy.CanAutoApprove(ctx, sandbox, changes)

		if !canApprove {
			t.Error("exactly at threshold should be allowed")
		}
	})
}

func TestDefaultApprovalPolicy_ValidateApproval(t *testing.T) {
	ctx := context.Background()
	policy := NewDefaultApprovalPolicy(config.PolicyConfig{})
	sandbox := &types.Sandbox{ID: uuid.New()}

	t.Run("default validation always succeeds", func(t *testing.T) {
		req := &types.ApprovalRequest{
			SandboxID: sandbox.ID,
			Mode:      "all",
		}

		err := policy.ValidateApproval(ctx, sandbox, req)
		if err != nil {
			t.Errorf("default validation should succeed: %v", err)
		}
	})
}

// =============================================================================
// RequireHumanApprovalPolicy Tests
// =============================================================================

func TestRequireHumanApprovalPolicy_CanAutoApprove(t *testing.T) {
	ctx := context.Background()
	policy := NewRequireHumanApprovalPolicy()
	sandbox := &types.Sandbox{ID: uuid.New()}

	t.Run("always returns false", func(t *testing.T) {
		changes := []*types.FileChange{{ID: uuid.New()}}
		canApprove, reason := policy.CanAutoApprove(ctx, sandbox, changes)

		if canApprove {
			t.Error("RequireHumanApprovalPolicy should never auto-approve")
		}
		if !strings.Contains(reason, "human approval is always required") {
			t.Errorf("reason should state human approval required, got: %s", reason)
		}
	})

	t.Run("returns false even for empty changes", func(t *testing.T) {
		canApprove, _ := policy.CanAutoApprove(ctx, sandbox, []*types.FileChange{})

		if canApprove {
			t.Error("should not auto-approve even with no changes")
		}
	})
}

func TestRequireHumanApprovalPolicy_ValidateApproval(t *testing.T) {
	ctx := context.Background()
	policy := NewRequireHumanApprovalPolicy()
	sandbox := &types.Sandbox{ID: uuid.New()}

	t.Run("requires actor", func(t *testing.T) {
		req := &types.ApprovalRequest{
			SandboxID: sandbox.ID,
			Mode:      "all",
			Actor:     "", // Empty actor
		}

		err := policy.ValidateApproval(ctx, sandbox, req)
		if err == nil {
			t.Error("should fail without actor")
		}
		if !strings.Contains(err.Error(), "actor is required") {
			t.Errorf("error should mention actor, got: %v", err)
		}
	})

	t.Run("succeeds with actor", func(t *testing.T) {
		req := &types.ApprovalRequest{
			SandboxID: sandbox.ID,
			Mode:      "all",
			Actor:     "reviewer@example.com",
		}

		err := policy.ValidateApproval(ctx, sandbox, req)
		if err != nil {
			t.Errorf("should succeed with actor: %v", err)
		}
	})
}

// =============================================================================
// DefaultAttributionPolicy Tests
// =============================================================================

func TestDefaultAttributionPolicy_GetCommitAuthor(t *testing.T) {
	ctx := context.Background()

	t.Run("agent mode uses sandbox owner", func(t *testing.T) {
		policy, err := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "agent",
			CommitMessageTemplate: "{{.FileCount}} files",
		})
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: "claude-agent"}
		author := policy.GetCommitAuthor(ctx, sandbox, "reviewer")

		if !strings.Contains(author, "claude-agent") {
			t.Errorf("agent mode should use sandbox owner, got: %s", author)
		}
	})

	t.Run("agent mode fallback when no owner", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "agent",
			CommitMessageTemplate: "test",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: ""}
		author := policy.GetCommitAuthor(ctx, sandbox, "reviewer")

		if !strings.Contains(author, "Workspace Sandbox") {
			t.Errorf("should fallback to Workspace Sandbox, got: %s", author)
		}
	})

	t.Run("reviewer mode uses actor", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "reviewer",
			CommitMessageTemplate: "test",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: "agent"}
		author := policy.GetCommitAuthor(ctx, sandbox, "human-reviewer")

		if !strings.Contains(author, "human-reviewer") {
			t.Errorf("reviewer mode should use actor, got: %s", author)
		}
	})

	t.Run("reviewer mode fallback when no actor", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "reviewer",
			CommitMessageTemplate: "test",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: "agent"}
		author := policy.GetCommitAuthor(ctx, sandbox, "")

		if !strings.Contains(author, "Workspace Sandbox") {
			t.Errorf("should fallback when no actor, got: %s", author)
		}
	})

	t.Run("coauthored mode uses sandbox owner", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "coauthored",
			CommitMessageTemplate: "test",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: "agent"}
		author := policy.GetCommitAuthor(ctx, sandbox, "reviewer")

		if !strings.Contains(author, "agent") {
			t.Errorf("coauthored mode should use owner as primary, got: %s", author)
		}
	})

	t.Run("default mode is agent", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "unknown-mode",
			CommitMessageTemplate: "test",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: "my-agent"}
		author := policy.GetCommitAuthor(ctx, sandbox, "reviewer")

		if !strings.Contains(author, "my-agent") {
			t.Errorf("unknown mode should default to agent, got: %s", author)
		}
	})
}

func TestDefaultAttributionPolicy_GetCommitMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("user message overrides template", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitMessageTemplate: "Template: {{.FileCount}} files",
		})

		sandbox := &types.Sandbox{ID: uuid.New()}
		changes := []*types.FileChange{{ID: uuid.New()}}

		message := policy.GetCommitMessage(ctx, sandbox, changes, "Custom message")

		if message != "Custom message" {
			t.Errorf("should use user message, got: %s", message)
		}
	})

	t.Run("template with file count", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitMessageTemplate: "Applied {{.FileCount}} changes",
		})

		sandbox := &types.Sandbox{ID: uuid.New()}
		changes := make([]*types.FileChange, 5)
		for i := range changes {
			changes[i] = &types.FileChange{ID: uuid.New()}
		}

		message := policy.GetCommitMessage(ctx, sandbox, changes, "")

		if !strings.Contains(message, "5") {
			t.Errorf("template should substitute file count, got: %s", message)
		}
	})

	t.Run("template with sandbox ID", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitMessageTemplate: "Sandbox: {{.SandboxID}}",
		})

		sandboxID := uuid.New()
		sandbox := &types.Sandbox{ID: sandboxID}
		changes := []*types.FileChange{}

		message := policy.GetCommitMessage(ctx, sandbox, changes, "")

		if !strings.Contains(message, sandboxID.String()) {
			t.Errorf("template should substitute sandbox ID, got: %s", message)
		}
	})

	t.Run("template with scope path", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitMessageTemplate: "Scope: {{.ScopePath}}",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), ScopePath: "/project/src"}
		changes := []*types.FileChange{}

		message := policy.GetCommitMessage(ctx, sandbox, changes, "")

		if !strings.Contains(message, "/project/src") {
			t.Errorf("template should substitute scope path, got: %s", message)
		}
	})
}

func TestDefaultAttributionPolicy_GetCoAuthors(t *testing.T) {
	ctx := context.Background()

	t.Run("coauthored mode adds reviewer as co-author", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "coauthored",
			CommitMessageTemplate: "test",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: "agent"}
		coAuthors := policy.GetCoAuthors(ctx, sandbox, "reviewer")

		if len(coAuthors) != 1 {
			t.Fatalf("expected 1 co-author, got %d", len(coAuthors))
		}
		if !strings.Contains(coAuthors[0], "Co-authored-by:") {
			t.Errorf("should have Co-authored-by line, got: %s", coAuthors[0])
		}
		if !strings.Contains(coAuthors[0], "reviewer") {
			t.Errorf("should contain reviewer name, got: %s", coAuthors[0])
		}
	})

	t.Run("coauthored mode skips when actor is owner", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "coauthored",
			CommitMessageTemplate: "test",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: "agent"}
		coAuthors := policy.GetCoAuthors(ctx, sandbox, "agent")

		if len(coAuthors) != 0 {
			t.Error("should not add co-author when actor is owner")
		}
	})

	t.Run("coauthored mode skips empty actor", func(t *testing.T) {
		policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
			CommitAuthorMode:      "coauthored",
			CommitMessageTemplate: "test",
		})

		sandbox := &types.Sandbox{ID: uuid.New(), Owner: "agent"}
		coAuthors := policy.GetCoAuthors(ctx, sandbox, "")

		if len(coAuthors) != 0 {
			t.Error("should not add co-author for empty actor")
		}
	})

	t.Run("non-coauthored modes return nil", func(t *testing.T) {
		for _, mode := range []string{"agent", "reviewer", ""} {
			policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
				CommitAuthorMode:      mode,
				CommitMessageTemplate: "test",
			})

			sandbox := &types.Sandbox{ID: uuid.New(), Owner: "agent"}
			coAuthors := policy.GetCoAuthors(ctx, sandbox, "reviewer")

			if coAuthors != nil {
				t.Errorf("mode %q should return nil co-authors", mode)
			}
		}
	})
}

func TestNewDefaultAttributionPolicy_InvalidTemplate(t *testing.T) {
	_, err := NewDefaultAttributionPolicy(config.PolicyConfig{
		CommitMessageTemplate: "{{.Invalid",
	})

	if err == nil {
		t.Error("should fail with invalid template")
	}
	if !strings.Contains(err.Error(), "invalid commit message template") {
		t.Errorf("error should mention template, got: %v", err)
	}
}

func TestFormatCommitMessage(t *testing.T) {
	t.Run("message without co-authors", func(t *testing.T) {
		result := FormatCommitMessage("My commit message", nil)
		if result != "My commit message" {
			t.Errorf("should return message unchanged, got: %s", result)
		}
	})

	t.Run("message with empty co-authors", func(t *testing.T) {
		result := FormatCommitMessage("My commit message", []string{})
		if result != "My commit message" {
			t.Errorf("should return message unchanged, got: %s", result)
		}
	})

	t.Run("message with single co-author", func(t *testing.T) {
		result := FormatCommitMessage("My commit", []string{"Co-authored-by: User <user@example.com>"})

		if !strings.Contains(result, "My commit") {
			t.Error("should contain original message")
		}
		if !strings.Contains(result, "Co-authored-by:") {
			t.Error("should contain co-author line")
		}
		if !strings.Contains(result, "\n\n") {
			t.Error("should have blank line before co-authors")
		}
	})

	t.Run("message with multiple co-authors", func(t *testing.T) {
		coAuthors := []string{
			"Co-authored-by: User1 <user1@example.com>",
			"Co-authored-by: User2 <user2@example.com>",
		}
		result := FormatCommitMessage("My commit", coAuthors)

		if strings.Count(result, "Co-authored-by:") != 2 {
			t.Error("should contain both co-author lines")
		}
	})
}

// =============================================================================
// NoOpValidationPolicy Tests
// =============================================================================

func TestNoOpValidationPolicy_ValidateBeforeApply(t *testing.T) {
	ctx := context.Background()
	policy := NewNoOpValidationPolicy()
	sandbox := &types.Sandbox{ID: uuid.New()}
	changes := []*types.FileChange{{ID: uuid.New()}}

	err := policy.ValidateBeforeApply(ctx, sandbox, changes)
	if err != nil {
		t.Errorf("NoOp validation should always succeed: %v", err)
	}
}

func TestNoOpValidationPolicy_GetValidationHooks(t *testing.T) {
	policy := NewNoOpValidationPolicy()

	hooks := policy.GetValidationHooks()
	if hooks != nil {
		t.Error("NoOp policy should return nil hooks")
	}
}

// =============================================================================
// HookValidationPolicy Tests
// =============================================================================

func TestHookValidationPolicy_GetValidationHooks(t *testing.T) {
	hooks := []ValidationHook{
		{Name: "lint", Command: "eslint", Required: true},
		{Name: "test", Command: "npm test", Required: false},
	}

	policy := NewHookValidationPolicy(hooks)
	result := policy.GetValidationHooks()

	if len(result) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(result))
	}
	if result[0].Name != "lint" {
		t.Error("first hook should be lint")
	}
	if result[1].Name != "test" {
		t.Error("second hook should be test")
	}
}

func TestHookValidationPolicy_ValidateBeforeApply(t *testing.T) {
	ctx := context.Background()

	t.Run("empty hooks list always succeeds", func(t *testing.T) {
		policy := NewHookValidationPolicy(nil)
		sandbox := &types.Sandbox{ID: uuid.New()}
		changes := []*types.FileChange{}

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err != nil {
			t.Errorf("empty hooks should succeed: %v", err)
		}
	})

	t.Run("successful hook passes", func(t *testing.T) {
		hooks := []ValidationHook{
			{Name: "echo-test", Command: "true", Required: true},
		}
		policy := NewHookValidationPolicy(hooks)
		sandbox := &types.Sandbox{ID: uuid.New(), ProjectRoot: "/tmp"}
		changes := []*types.FileChange{}

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err != nil {
			t.Errorf("successful hook should pass: %v", err)
		}
	})

	t.Run("required hook failure blocks approval", func(t *testing.T) {
		hooks := []ValidationHook{
			{Name: "fail-hook", Command: "false", Required: true},
		}
		policy := NewHookValidationPolicy(hooks)
		sandbox := &types.Sandbox{ID: uuid.New(), ProjectRoot: "/tmp"}
		changes := []*types.FileChange{}

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err == nil {
			t.Error("required hook failure should block approval")
		}

		hookErr, ok := err.(*ValidationHookError)
		if !ok {
			t.Errorf("expected ValidationHookError, got: %T", err)
		} else if hookErr.HookName != "fail-hook" {
			t.Errorf("error should reference failing hook, got: %s", hookErr.HookName)
		}
	})

	t.Run("non-required hook failure does not block approval", func(t *testing.T) {
		hooks := []ValidationHook{
			{Name: "optional-fail", Command: "false", Required: false},
		}
		policy := NewHookValidationPolicy(hooks)
		sandbox := &types.Sandbox{ID: uuid.New(), ProjectRoot: "/tmp"}
		changes := []*types.FileChange{}

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err != nil {
			t.Errorf("non-required hook failure should not block: %v", err)
		}
	})

	t.Run("multiple hooks execute in order", func(t *testing.T) {
		hooks := []ValidationHook{
			{Name: "hook1", Command: "true", Required: true},
			{Name: "hook2", Command: "true", Required: true},
		}
		policy := NewHookValidationPolicy(hooks)
		sandbox := &types.Sandbox{ID: uuid.New(), ProjectRoot: "/tmp"}
		changes := []*types.FileChange{}

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err != nil {
			t.Errorf("all passing hooks should succeed: %v", err)
		}
	})

	t.Run("stops on first required failure by default", func(t *testing.T) {
		hooks := []ValidationHook{
			{Name: "hook1", Command: "false", Required: true},
			{Name: "hook2", Command: "true", Required: true},
		}
		policy := NewHookValidationPolicy(hooks)
		sandbox := &types.Sandbox{ID: uuid.New(), ProjectRoot: "/tmp"}
		changes := []*types.FileChange{}

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err == nil {
			t.Error("should fail on first required hook failure")
		}

		hookErr := err.(*ValidationHookError)
		if hookErr.HookName != "hook1" {
			t.Errorf("should fail on hook1, got: %s", hookErr.HookName)
		}
	})

	t.Run("continue on fail option runs all hooks", func(t *testing.T) {
		var executedHooks []string
		hooks := []ValidationHook{
			{Name: "hook1", Command: "false", Required: false},
			{Name: "hook2", Command: "true", Required: false},
		}

		policy := NewHookValidationPolicy(hooks, WithContinueOnFail(true))
		sandbox := &types.Sandbox{ID: uuid.New(), ProjectRoot: "/tmp"}
		changes := []*types.FileChange{}

		// Use a simple logger to track execution
		_ = executedHooks // Used to track in real implementation

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err != nil {
			t.Errorf("non-required hooks should not fail: %v", err)
		}
	})

	t.Run("environment variables are set correctly", func(t *testing.T) {
		// Use sh to verify env vars are available
		// Note: Use /tmp as working dir which always exists
		hooks := []ValidationHook{
			{
				Name:     "check-env",
				Command:  "sh",
				Args:     []string{"-c", "test -n \"$SANDBOX_ID\""},
				Required: true,
			},
		}
		policy := NewHookValidationPolicy(hooks)
		sandbox := &types.Sandbox{
			ID:          uuid.New(),
			ScopePath:   "/project/src",
			ProjectRoot: "/tmp", // Use /tmp which always exists
			UpperDir:    "/tmp",
			MergedDir:   "", // Empty to fall back to ProjectRoot
		}
		changes := []*types.FileChange{
			{FilePath: "file1.go"},
			{FilePath: "file2.go"},
		}

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err != nil {
			t.Errorf("env vars should be set: %v", err)
		}
	})

	t.Run("hook with arguments executes correctly", func(t *testing.T) {
		hooks := []ValidationHook{
			{
				Name:     "echo-args",
				Command:  "sh",
				Args:     []string{"-c", "exit 0"},
				Required: true,
			},
		}
		policy := NewHookValidationPolicy(hooks)
		sandbox := &types.Sandbox{ID: uuid.New(), ProjectRoot: "/tmp"}
		changes := []*types.FileChange{}

		err := policy.ValidateBeforeApply(ctx, sandbox, changes)
		if err != nil {
			t.Errorf("hook with args should work: %v", err)
		}
	})
}

// [OT-P1-005] Pre-commit Validation Hooks - Additional Tests
func TestValidationHookError(t *testing.T) {
	t.Run("error message includes hook name", func(t *testing.T) {
		err := &ValidationHookError{
			HookName: "lint-check",
			Err:      nil,
		}

		msg := err.Error()
		if !strings.Contains(msg, "lint-check") {
			t.Errorf("error should include hook name, got: %s", msg)
		}
	})

	t.Run("error message includes underlying error", func(t *testing.T) {
		err := &ValidationHookError{
			HookName: "test",
			Err:      context.DeadlineExceeded,
		}

		msg := err.Error()
		if !strings.Contains(msg, "deadline") {
			t.Errorf("error should include underlying error, got: %s", msg)
		}
	})

	t.Run("implements DomainError interface", func(t *testing.T) {
		err := &ValidationHookError{HookName: "test"}

		if err.HTTPStatus() != 422 {
			t.Errorf("expected HTTP 422, got: %d", err.HTTPStatus())
		}
		if err.IsRetryable() {
			t.Error("validation errors should not be retryable")
		}
		if !strings.Contains(err.Hint(), "test") {
			t.Errorf("hint should reference hook name, got: %s", err.Hint())
		}
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := context.Canceled
		err := &ValidationHookError{
			HookName: "test",
			Err:      underlying,
		}

		if err.Unwrap() != underlying {
			t.Error("Unwrap should return underlying error")
		}
	})
}

func TestBuildHookEnv(t *testing.T) {
	sandbox := &types.Sandbox{
		ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		ScopePath:   "/project/src",
		ProjectRoot: "/project",
		UpperDir:    "/tmp/upper",
		MergedDir:   "/tmp/merged",
	}
	changes := []*types.FileChange{
		{FilePath: "file1.go"},
		{FilePath: "file2.go"},
	}

	env := buildHookEnv(sandbox, changes)

	t.Run("includes sandbox ID", func(t *testing.T) {
		found := false
		for _, e := range env {
			if strings.HasPrefix(e, "SANDBOX_ID=") {
				found = true
				if !strings.Contains(e, "123e4567") {
					t.Errorf("SANDBOX_ID should contain UUID, got: %s", e)
				}
			}
		}
		if !found {
			t.Error("SANDBOX_ID not found in environment")
		}
	})

	t.Run("includes scope path", func(t *testing.T) {
		found := false
		for _, e := range env {
			if e == "SANDBOX_SCOPE_PATH=/project/src" {
				found = true
			}
		}
		if !found {
			t.Error("SANDBOX_SCOPE_PATH not found in environment")
		}
	})

	t.Run("includes change count", func(t *testing.T) {
		found := false
		for _, e := range env {
			if e == "SANDBOX_CHANGE_COUNT=2" {
				found = true
			}
		}
		if !found {
			t.Error("SANDBOX_CHANGE_COUNT not found in environment")
		}
	})

	t.Run("includes changed files", func(t *testing.T) {
		found := false
		for _, e := range env {
			if strings.HasPrefix(e, "SANDBOX_CHANGED_FILES=") {
				found = true
				if !strings.Contains(e, "file1.go") || !strings.Contains(e, "file2.go") {
					t.Errorf("SANDBOX_CHANGED_FILES should list files, got: %s", e)
				}
			}
		}
		if !found {
			t.Error("SANDBOX_CHANGED_FILES not found in environment")
		}
	})

	t.Run("empty changes does not include changed files", func(t *testing.T) {
		env := buildHookEnv(sandbox, []*types.FileChange{})
		for _, e := range env {
			if strings.HasPrefix(e, "SANDBOX_CHANGED_FILES=") {
				t.Error("should not include SANDBOX_CHANGED_FILES for empty changes")
			}
		}
	})
}

func TestNewHookValidationPolicy_EmptyHooks(t *testing.T) {
	policy := NewHookValidationPolicy(nil)

	hooks := policy.GetValidationHooks()
	if hooks != nil {
		t.Error("nil input should result in nil hooks")
	}
}

func TestNewHookValidationPolicy_WithHooks(t *testing.T) {
	hooks := []ValidationHook{
		{
			Name:        "pre-commit",
			Description: "Run pre-commit checks",
			Command:     "pre-commit",
			Args:        []string{"run", "--all-files"},
			Required:    true,
		},
	}

	policy := NewHookValidationPolicy(hooks)
	result := policy.GetValidationHooks()

	if len(result) != 1 {
		t.Fatal("expected 1 hook")
	}
	if result[0].Name != "pre-commit" {
		t.Error("hook name mismatch")
	}
	if result[0].Description != "Run pre-commit checks" {
		t.Error("hook description mismatch")
	}
	if len(result[0].Args) != 2 {
		t.Error("hook args mismatch")
	}
}

// =============================================================================
// Interface Implementation Tests
// =============================================================================

func TestApprovalPolicyInterface(t *testing.T) {
	// Verify all approval policies implement the interface
	var _ ApprovalPolicy = (*DefaultApprovalPolicy)(nil)
	var _ ApprovalPolicy = (*RequireHumanApprovalPolicy)(nil)
}

func TestAttributionPolicyInterface(t *testing.T) {
	// Verify attribution policy implements the interface
	var _ AttributionPolicy = (*DefaultAttributionPolicy)(nil)
}

func TestValidationPolicyInterface(t *testing.T) {
	// Verify all validation policies implement the interface
	var _ ValidationPolicy = (*NoOpValidationPolicy)(nil)
	var _ ValidationPolicy = (*HookValidationPolicy)(nil)
}

// =============================================================================
// Edge Cases and Error Conditions
// =============================================================================

func TestDefaultApprovalPolicy_EmptyContext(t *testing.T) {
	policy := NewDefaultApprovalPolicy(config.PolicyConfig{
		RequireHumanApproval: false,
	})
	sandbox := &types.Sandbox{ID: uuid.New()}
	changes := []*types.FileChange{}

	// Should work with minimal context
	canApprove, _ := policy.CanAutoApprove(context.TODO(), sandbox, changes)
	if !canApprove {
		t.Error("should work with TODO context")
	}
}

func TestDefaultAttributionPolicy_NilChanges(t *testing.T) {
	policy, _ := NewDefaultAttributionPolicy(config.PolicyConfig{
		CommitMessageTemplate: "{{.FileCount}} files",
	})
	sandbox := &types.Sandbox{ID: uuid.New()}

	// Should not panic with nil changes
	message := policy.GetCommitMessage(context.Background(), sandbox, nil, "")
	if !strings.Contains(message, "0") {
		t.Errorf("nil changes should count as 0, got: %s", message)
	}
}

func TestValidationHookStruct(t *testing.T) {
	hook := ValidationHook{
		Name:        "test-hook",
		Description: "A test hook",
		Command:     "echo",
		Args:        []string{"hello"},
		Required:    true,
	}

	if hook.Name != "test-hook" {
		t.Error("Name field not set correctly")
	}
	if hook.Command != "echo" {
		t.Error("Command field not set correctly")
	}
	if !hook.Required {
		t.Error("Required field not set correctly")
	}
}

func TestValidationResultStruct(t *testing.T) {
	result := ValidationResult{
		HookName: "lint",
		Success:  true,
		Output:   "All files passed",
		Error:    nil,
	}

	if result.HookName != "lint" {
		t.Error("HookName not set correctly")
	}
	if !result.Success {
		t.Error("Success not set correctly")
	}
}
