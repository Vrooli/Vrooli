package plans

import "time"

type PlanRecord struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Path        string    `json:"path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Archived    bool      `json:"archived"`
	ArchivedAt  time.Time `json:"archived_at,omitempty"`
	SourcePath  string    `json:"source_path,omitempty"`
	ContentHash string    `json:"content_hash,omitempty"`
}

type AddRequest struct {
	Title   string
	Slug    string
	Repo    string
	Content string
}

type AddOutput struct {
	Success bool       `json:"success"`
	Plan    PlanRecord `json:"plan"`
}

type ListRequest struct {
	Repo            string
	IncludeAll      bool
	IncludeArchived bool
}

type ListOutput struct {
	Success bool         `json:"success"`
	Plans   []PlanRecord `json:"plans"`
}

type ShowRequest struct {
	Ref  string
	Repo string
}

type ShowOutput struct {
	Success bool       `json:"success"`
	Plan    PlanRecord `json:"plan"`
	Content string     `json:"content,omitempty"`
}

type PathOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Path    string `json:"path"`
}

type ArchiveRequest struct {
	Ref  string
	Repo string
}

type ArchiveOutput struct {
	Success bool       `json:"success"`
	Plan    PlanRecord `json:"plan"`
}

type ImportRequest struct {
	Path         string
	Title        string
	Slug         string
	Repo         string
	DeleteSource bool
}

type ImportOutput struct {
	Success bool       `json:"success"`
	Plan    PlanRecord `json:"plan"`
	Deleted bool       `json:"deleted_source"`
}

type ExportRequest struct {
	Ref  string
	Repo string
	To   string
}

type ExportOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Path    string `json:"path"`
}
