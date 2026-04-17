package teams

import (
	"context"
	"prompt-manager/store"
	"testing"
)

// Test helper to set up permission checker with mock relation store
func setupPermissionChecker(config PermissionConfig) (*PermissionChecker, *MockRelationStore) {
	relationStore := NewMockRelationStore()
	checker := NewPermissionChecker(config, relationStore)
	return checker, relationStore
}

// ============== CanManageTeam Tests ==============

func TestCanManageTeam_AuthDisabled(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  false,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin"},
	}
	checker, _ := setupPermissionChecker(config)

	// Should return true even without any membership
	can, err := checker.CanManageTeam(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !can {
		t.Error("Expected true when auth is disabled")
	}
}

func TestCanManageTeam_WithManagerRole(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin", "lead"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	// Add agent as active member with manager role
	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"manager"},
			Status:  store.MemberStatusActive,
		},
	}

	can, err := checker.CanManageTeam(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !can {
		t.Error("Expected true for active member with manager role")
	}
}

func TestCanManageTeam_WithoutManagerRole(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	// Add agent as active member with developer role (not a manager role)
	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer"},
			Status:  store.MemberStatusActive,
		},
	}

	can, err := checker.CanManageTeam(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if can {
		t.Error("Expected false for member without manager role")
	}
}

func TestCanManageTeam_InactiveMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	// Add agent as inactive member with manager role
	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"manager"},
			Status:  store.MemberStatusInactive,
		},
	}

	can, err := checker.CanManageTeam(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if can {
		t.Error("Expected false for inactive member even with manager role")
	}
}

func TestCanManageTeam_NotAMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin"},
	}
	checker, _ := setupPermissionChecker(config)

	// Agent is not a member of the team
	can, err := checker.CanManageTeam(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if can {
		t.Error("Expected false for non-member")
	}
}

// ============== CanManageMembers Tests ==============

func TestCanManageMembers_WithManagerRole(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"manager"},
			Status:  store.MemberStatusActive,
		},
	}

	can, err := checker.CanManageMembers(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !can {
		t.Error("Expected true for active member with manager role")
	}
}

// ============== CanManageRoles Tests ==============

func TestCanManageRoles_AuthDisabled(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  false,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin"},
	}
	checker, _ := setupPermissionChecker(config)

	can, err := checker.CanManageRoles(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !can {
		t.Error("Expected true when auth is disabled")
	}
}

func TestCanManageRoles_WithAdminRole(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin", "owner"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"admin"},
			Status:  store.MemberStatusActive,
		},
	}

	can, err := checker.CanManageRoles(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !can {
		t.Error("Expected true for active member with admin role")
	}
}

func TestCanManageRoles_ManagerNotAdmin(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin", "owner"},
	}
	checker, relationStore := setupPermissionChecker(config)

	// Manager role can manage team but not roles
	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"manager"},
			Status:  store.MemberStatusActive,
		},
	}

	can, err := checker.CanManageRoles(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if can {
		t.Error("Expected false for manager (not admin) trying to manage roles")
	}
}

func TestCanManageRoles_InactiveMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"admin"},
			Status:  store.MemberStatusInactive,
		},
	}

	can, err := checker.CanManageRoles(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if can {
		t.Error("Expected false for inactive member even with admin role")
	}
}

// ============== CanManageOrgChart Tests ==============

func TestCanManageOrgChart_WithAdminRole(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin", "owner"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"owner"},
			Status:  store.MemberStatusActive,
		},
	}

	can, err := checker.CanManageOrgChart(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !can {
		t.Error("Expected true for active member with owner role")
	}
}

func TestCanManageOrgChart_ManagerNotAdmin(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin"},
		AdminRoles:   []string{"admin", "owner"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"manager"},
			Status:  store.MemberStatusActive,
		},
	}

	can, err := checker.CanManageOrgChart(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if can {
		t.Error("Expected false for manager (not admin) trying to manage org chart")
	}
}

// ============== IsMember Tests ==============

func TestIsMember_AuthDisabled(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  false,
		ManagerRoles: []string{"manager"},
		AdminRoles:   []string{"admin"},
	}
	checker, _ := setupPermissionChecker(config)

	is, err := checker.IsMember(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !is {
		t.Error("Expected true when auth is disabled")
	}
}

func TestIsMember_ActiveMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer"},
			Status:  store.MemberStatusActive,
		},
	}

	is, err := checker.IsMember(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !is {
		t.Error("Expected true for active member")
	}
}

func TestIsMember_InactiveMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	// IsMember returns true even for inactive members
	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer"},
			Status:  store.MemberStatusInactive,
		},
	}

	is, err := checker.IsMember(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !is {
		t.Error("Expected true for inactive member (IsMember checks existence, not status)")
	}
}

func TestIsMember_NotMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager"},
		AdminRoles:   []string{"admin"},
	}
	checker, _ := setupPermissionChecker(config)

	is, err := checker.IsMember(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if is {
		t.Error("Expected false for non-member")
	}
}

// ============== IsActiveMember Tests ==============

func TestIsActiveMember_AuthDisabled(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  false,
		ManagerRoles: []string{"manager"},
		AdminRoles:   []string{"admin"},
	}
	checker, _ := setupPermissionChecker(config)

	is, err := checker.IsActiveMember(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !is {
		t.Error("Expected true when auth is disabled")
	}
}

func TestIsActiveMember_ActiveMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer"},
			Status:  store.MemberStatusActive,
		},
	}

	is, err := checker.IsActiveMember(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !is {
		t.Error("Expected true for active member")
	}
}

func TestIsActiveMember_InactiveMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer"},
			Status:  store.MemberStatusInactive,
		},
	}

	is, err := checker.IsActiveMember(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if is {
		t.Error("Expected false for inactive member")
	}
}

func TestIsActiveMember_PendingMember(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer"},
			Status:  store.MemberStatusPending,
		},
	}

	is, err := checker.IsActiveMember(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if is {
		t.Error("Expected false for pending member")
	}
}

// ============== Multiple Roles Tests ==============

func TestCanManageTeam_MultipleRoles(t *testing.T) {
	config := PermissionConfig{
		RequireAuth:  true,
		ManagerRoles: []string{"manager", "admin", "lead"},
		AdminRoles:   []string{"admin"},
	}
	checker, relationStore := setupPermissionChecker(config)

	// Agent has multiple roles, one of which is a manager role
	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer", "lead"},
			Status:  store.MemberStatusActive,
		},
	}

	can, err := checker.CanManageTeam(context.Background(), "agent-1", "team-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !can {
		t.Error("Expected true when one of multiple roles is a manager role")
	}
}

// ============== Default Config Tests ==============

func TestDefaultPermissionConfig(t *testing.T) {
	config := DefaultPermissionConfig()

	if config.RequireAuth {
		t.Error("Expected RequireAuth to be false by default")
	}

	if len(config.ManagerRoles) == 0 {
		t.Error("Expected ManagerRoles to have default values")
	}

	if len(config.AdminRoles) == 0 {
		t.Error("Expected AdminRoles to have default values")
	}

	// Verify expected default roles
	expectedManagerRoles := []string{"manager", "admin", "lead", "owner"}
	for _, role := range expectedManagerRoles {
		found := false
		for _, r := range config.ManagerRoles {
			if r == role {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' to be in default ManagerRoles", role)
		}
	}

	expectedAdminRoles := []string{"admin", "owner"}
	for _, role := range expectedAdminRoles {
		found := false
		for _, r := range config.AdminRoles {
			if r == role {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' to be in default AdminRoles", role)
		}
	}
}
