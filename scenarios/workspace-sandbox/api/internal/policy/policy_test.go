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
	hooks := []ValidationHook{
		{Name: "lint", Command: "true", Required: true},
	}

	policy := NewHookValidationPolicy(hooks)
	sandbox := &types.Sandbox{ID: uuid.New()}
	changes := []*types.FileChange{}

	// Current implementation is a placeholder that always succeeds
	err := policy.ValidateBeforeApply(ctx, sandbox, changes)
	if err != nil {
		t.Errorf("placeholder should succeed: %v", err)
	}
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

func TestDefaultApprovalPolicy_NilContext(t *testing.T) {
	policy := NewDefaultApprovalPolicy(config.PolicyConfig{
		RequireHumanApproval: false,
	})
	sandbox := &types.Sandbox{ID: uuid.New()}
	changes := []*types.FileChange{}

	// Should not panic with nil context
	canApprove, _ := policy.CanAutoApprove(nil, sandbox, changes)
	if !canApprove {
		t.Error("should still work with nil context")
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
