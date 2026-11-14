package plans

import "time"

type PlanRecord struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	Path          string    `json:"path"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Archived      bool      `json:"archived"`
	ArchivedAt    time.Time `json:"archived_at,omitempty"`
	SourcePath    string    `json:"source_path,omitempty"`
	ContentHash   string    `json:"content_hash,omitempty"`
	WorkspaceID   string    `json:"workspace_id,omitempty"`
	WorkspaceRoot string    `json:"workspace_root,omitempty"`
}

type WorkspaceScope struct {
	ID   string `json:"id,omitempty"`
	Root string `json:"root,omitempty"`
}

type ListRequest struct {
	Workspace       string
	IncludeArchived bool
}

type ListOutput struct {
	Success  bool         `json:"success"`
	Plans    []PlanRecord `json:"plans"`
	Source   string       `json:"source,omitempty"`
	Degraded bool         `json:"degraded,omitempty"`
	Warning  string       `json:"warning,omitempty"`
}

type ShowRequest struct {
	Ref       string
	Workspace string
}

type ShowOutput struct {
	Success  bool       `json:"success"`
	Plan     PlanRecord `json:"plan"`
	Content  string     `json:"content,omitempty"`
	Source   string     `json:"source,omitempty"`
	Degraded bool       `json:"degraded,omitempty"`
	Warning  string     `json:"warning,omitempty"`
}

type PathOutput struct {
	Success  bool   `json:"success"`
	ID       string `json:"id"`
	Path     string `json:"path"`
	Source   string `json:"source,omitempty"`
	Degraded bool   `json:"degraded,omitempty"`
	Warning  string `json:"warning,omitempty"`
}
