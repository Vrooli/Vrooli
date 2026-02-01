// Package teams provides types and operations for team management.
//
// DOC: docs/reference/api-endpoints.md#teams
package teams

// Response is the API response for a team.
type Response struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Mission     string `json:"mission,omitempty"`
	MemberCount int    `json:"memberCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// TeamDetailsResponse includes full team details.
type TeamDetailsResponse struct {
	Response
	Roles    []RoleDTO    `json:"roles"`
	Members  []MemberDTO  `json:"members"`
	Defaults *DefaultsDTO `json:"defaults,omitempty"`
}

// RoleDTO represents a role in a team.
type RoleDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// MemberDTO represents a team member.
type MemberDTO struct {
	AgentID     string   `json:"agentId"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
	Status      string   `json:"status"`
}

// DefaultsDTO represents default team policies.
type DefaultsDTO struct {
	SkillGrantsByRole map[string][]string `json:"skillGrantsByRole,omitempty"`
}

// CreateRequest is the request body for creating a team.
type CreateRequest struct {
	ID          string       `json:"id,omitempty"`
	DisplayName string       `json:"displayName"`
	Mission     string       `json:"mission,omitempty"`
	Defaults    *DefaultsDTO `json:"defaults,omitempty"`
}

// UpdateRequest is the request body for updating a team.
type UpdateRequest struct {
	DisplayName *string      `json:"displayName,omitempty"`
	Mission     *string      `json:"mission,omitempty"`
	Defaults    *DefaultsDTO `json:"defaults,omitempty"`
}

// AddMemberRequest is the request body for adding a member to a team.
type AddMemberRequest struct {
	AgentID string   `json:"agentId"`
	Roles   []string `json:"roles,omitempty"`
}

// UpdateMemberRequest is the request body for updating a team member.
type UpdateMemberRequest struct {
	Roles  []string `json:"roles,omitempty"`
	Status *string  `json:"status,omitempty"`
}

// SetRolesRequest is the request body for setting team roles.
type SetRolesRequest struct {
	Roles []RoleDTO `json:"roles"`
}

// OrgEdgeDTO represents a manager-report relationship in the org chart.
type OrgEdgeDTO struct {
	ManagerAgentID string `json:"managerAgentId"`
	ReportAgentID  string `json:"reportAgentId"`
}

// OrgChartResponse is the API response for a team's org chart.
type OrgChartResponse struct {
	TeamID string       `json:"teamId"`
	Edges  []OrgEdgeDTO `json:"edges"`
}

// SetOrgChartRequest is the request body for setting a team's org chart.
type SetOrgChartRequest struct {
	Edges []OrgEdgeDTO `json:"edges"`
}
