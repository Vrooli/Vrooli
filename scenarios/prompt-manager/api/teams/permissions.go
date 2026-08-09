package teams

import (
	"context"
	"slices"

	"prompt-manager/store"
)

// PermissionConfig configures permission checking behavior.
type PermissionConfig struct {
	// RequireAuth toggles authentication requirement.
	// When false, all permission checks return true (open access for development).
	RequireAuth bool

	// ManagerRoles lists roles that can manage team and members.
	ManagerRoles []string

	// AdminRoles lists roles that can manage roles and org chart.
	AdminRoles []string
}

// DefaultPermissionConfig returns the default permission configuration.
// RequireAuth is false to allow open access during development.
func DefaultPermissionConfig() PermissionConfig {
	return PermissionConfig{
		RequireAuth:  false,
		ManagerRoles: []string{"manager", "admin", "lead", "owner"},
		AdminRoles:   []string{"admin", "owner"},
	}
}

// PermissionChecker provides permission decision methods for team operations.
type PermissionChecker struct {
	config        PermissionConfig
	relationStore store.RelationStore
}

// NewPermissionChecker creates a new permission checker with the given configuration.
func NewPermissionChecker(config PermissionConfig, relationStore store.RelationStore) *PermissionChecker {
	return &PermissionChecker{
		config:        config,
		relationStore: relationStore,
	}
}

// CanManageTeam checks if an agent can manage team settings (update, delete).
// Returns true if:
// - RequireAuth is disabled (open access)
// - Agent is an active member with a manager or admin role
func (p *PermissionChecker) CanManageTeam(ctx context.Context, agentID, teamID string) (bool, error) {
	if !p.config.RequireAuth {
		return true, nil
	}

	membership, err := p.relationStore.GetTeamMember(ctx, teamID, agentID)
	if err != nil {
		return false, nil // Not a member
	}

	if membership.Status != store.MemberStatusActive {
		return false, nil // Inactive member
	}

	return p.hasAnyRole(membership.Roles, p.config.ManagerRoles), nil
}

// CanManageMembers checks if an agent can add/update/remove team members.
// Returns true if:
// - RequireAuth is disabled (open access)
// - Agent is an active member with a manager or admin role
func (p *PermissionChecker) CanManageMembers(ctx context.Context, agentID, teamID string) (bool, error) {
	// Same permissions as team management
	return p.CanManageTeam(ctx, agentID, teamID)
}

// CanManageRoles checks if an agent can modify team role definitions.
// Returns true if:
// - RequireAuth is disabled (open access)
// - Agent is an active member with an admin role
func (p *PermissionChecker) CanManageRoles(ctx context.Context, agentID, teamID string) (bool, error) {
	if !p.config.RequireAuth {
		return true, nil
	}

	membership, err := p.relationStore.GetTeamMember(ctx, teamID, agentID)
	if err != nil {
		return false, nil // Not a member
	}

	if membership.Status != store.MemberStatusActive {
		return false, nil // Inactive member
	}

	return p.hasAnyRole(membership.Roles, p.config.AdminRoles), nil
}

// CanManageOrgChart checks if an agent can modify the team org chart.
// Returns true if:
// - RequireAuth is disabled (open access)
// - Agent is an active member with an admin role
func (p *PermissionChecker) CanManageOrgChart(ctx context.Context, agentID, teamID string) (bool, error) {
	// Same permissions as role management
	return p.CanManageRoles(ctx, agentID, teamID)
}

// IsMember checks if an agent is a member of the team.
// Returns true if:
// - RequireAuth is disabled (open access)
// - Agent has any membership record (active, inactive, or pending)
func (p *PermissionChecker) IsMember(ctx context.Context, agentID, teamID string) (bool, error) {
	if !p.config.RequireAuth {
		return true, nil
	}

	_, err := p.relationStore.GetTeamMember(ctx, teamID, agentID)
	return err == nil, nil
}

// IsActiveMember checks if an agent is an active member of the team.
// Returns true if:
// - RequireAuth is disabled (open access)
// - Agent is a member with active status
func (p *PermissionChecker) IsActiveMember(ctx context.Context, agentID, teamID string) (bool, error) {
	if !p.config.RequireAuth {
		return true, nil
	}

	membership, err := p.relationStore.GetTeamMember(ctx, teamID, agentID)
	if err != nil {
		return false, nil
	}

	return membership.Status == store.MemberStatusActive, nil
}

// hasAnyRole checks if any of the agent's roles are in the allowed list.
func (p *PermissionChecker) hasAnyRole(agentRoles, allowedRoles []string) bool {
	for _, role := range agentRoles {
		if slices.Contains(allowedRoles, role) {
			return true
		}
	}
	return false
}
