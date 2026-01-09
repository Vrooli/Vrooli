package policy

import (
	"context"
	"fmt"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/types"
)

// DefaultApprovalPolicy implements ApprovalPolicy using configuration.
type DefaultApprovalPolicy struct {
	config config.PolicyConfig
}

// NewDefaultApprovalPolicy creates an approval policy from config.
func NewDefaultApprovalPolicy(cfg config.PolicyConfig) *DefaultApprovalPolicy {
	return &DefaultApprovalPolicy{config: cfg}
}

// CanAutoApprove decides if changes can be automatically approved.
func (p *DefaultApprovalPolicy) CanAutoApprove(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange) (bool, string) {
	// If human approval is required, never auto-approve
	if p.config.RequireHumanApproval {
		return false, "human approval is required by policy"
	}

	// Check file count threshold
	if p.config.AutoApproveThresholdFiles > 0 && len(changes) > p.config.AutoApproveThresholdFiles {
		return false, fmt.Sprintf("change count (%d) exceeds auto-approve threshold (%d files)",
			len(changes), p.config.AutoApproveThresholdFiles)
	}

	// Note: Line count threshold would require diff generation here.
	// For now, we only check file count. Line count check should be done
	// after diff is generated if needed.

	return true, "changes meet auto-approval criteria"
}

// ValidateApproval performs additional validation before approval.
func (p *DefaultApprovalPolicy) ValidateApproval(ctx context.Context, sandbox *types.Sandbox, req *types.ApprovalRequest) error {
	// Default implementation: no additional validation
	return nil
}

// RequireHumanApprovalPolicy always requires human approval.
type RequireHumanApprovalPolicy struct{}

// NewRequireHumanApprovalPolicy creates a policy that always requires human approval.
func NewRequireHumanApprovalPolicy() *RequireHumanApprovalPolicy {
	return &RequireHumanApprovalPolicy{}
}

// CanAutoApprove always returns false.
func (p *RequireHumanApprovalPolicy) CanAutoApprove(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange) (bool, string) {
	return false, "human approval is always required"
}

// ValidateApproval checks that an actor is provided.
func (p *RequireHumanApprovalPolicy) ValidateApproval(ctx context.Context, sandbox *types.Sandbox, req *types.ApprovalRequest) error {
	if req.Actor == "" {
		return fmt.Errorf("actor is required for approval")
	}
	return nil
}

// Verify interfaces are implemented.
var (
	_ ApprovalPolicy = (*DefaultApprovalPolicy)(nil)
	_ ApprovalPolicy = (*RequireHumanApprovalPolicy)(nil)
)
