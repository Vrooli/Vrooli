// Package teams provides types and operations for team management.
//
// DOC: docs/reference/api-endpoints.md#teams
package teams

// Response is the API response for a team.
type Response struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Mission     string `json:"mission,omitempty"`
	Enabled     bool   `json:"enabled"`
	SpawnMode   string `json:"spawnMode,omitempty"`
	MemberCount int    `json:"memberCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// TeamDetailsResponse includes full team details.
type TeamDetailsResponse struct {
	Response
	Roles   []RoleDTO   `json:"roles"`
	Members []MemberDTO `json:"members"`
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

// TeamMessageDTO represents a message sent between team members.
type TeamMessageDTO struct {
	ID          string `json:"id"`
	TeamID      string `json:"teamId"`
	FromAgentID string `json:"fromAgentId"`
	ToAgentID   string `json:"toAgentId"`
	Content     string `json:"content"`
	CreatedAt   string `json:"createdAt"`
}

// TeamInboxResponse is the API response for a team member inbox.
type TeamInboxResponse struct {
	TeamID   string           `json:"teamId"`
	AgentID  string           `json:"agentId"`
	Messages []TeamMessageDTO `json:"messages"`
}

// SendTeamMessageRequest is the request body for sending a team message.
type SendTeamMessageRequest struct {
	FromAgentID string `json:"fromAgentId"`
	Content     string `json:"content"`
}

// CreateRequest is the request body for creating a team.
type CreateRequest struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"displayName"`
	Mission     string `json:"mission,omitempty"`
	SpawnMode   string `json:"spawnMode,omitempty"`
}

// UpdateRequest is the request body for updating a team.
type UpdateRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Mission     *string `json:"mission,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	SpawnMode   *string `json:"spawnMode,omitempty"`
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

// UpdateOrgEdgeRequest is the request body for updating a single org chart edge.
type UpdateOrgEdgeRequest struct {
	ManagerAgentID string `json:"managerAgentId"`
}

// TeamSharedFileEntry represents a file or directory in a team's shared folder.
type TeamSharedFileEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size,omitempty"`
}

// TeamSharedFileListResponse is the response for listing shared files.
type TeamSharedFileListResponse struct {
	TeamID string                `json:"teamId"`
	Files  []TeamSharedFileEntry `json:"files"`
}

// TeamSharedFileContentResponse is the response for shared file content.
type TeamSharedFileContentResponse struct {
	TeamID  string `json:"teamId"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// TeamSharedFileWriteRequest is the request body for writing shared file content.
type TeamSharedFileWriteRequest struct {
	Content string `json:"content"`
}

// TeamSharedFileCreateRequest is the request body for creating a shared file or directory.
type TeamSharedFileCreateRequest struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	IsDir   bool   `json:"isDir,omitempty"`
}

// TeamSharedFileRenameRequest is the request body for renaming a shared file.
type TeamSharedFileRenameRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}
