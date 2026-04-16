package support

import (
	"encoding/json"
	"time"
)

// Project mirrors the API's Project shape returned by /api/v1/projects.
type Project struct {
	ID                string                 `json:"id"`
	Path              string                 `json:"path"`
	Name              string                 `json:"name"`
	Type              string                 `json:"type"`
	VrooliVersion     *string                `json:"vrooli_version,omitempty"`
	BridgeVersion     *string                `json:"bridge_version,omitempty"`
	IntegrationStatus string                 `json:"integration_status"`
	LastUpdated       time.Time              `json:"last_updated"`
	CreatedAt         time.Time              `json:"created_at"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ProjectsResponse is the shape returned by GET /api/v1/projects.
type ProjectsResponse struct {
	Projects []Project `json:"projects"`
}

// ScanResponse is the shape returned by POST /api/v1/projects/scan.
type ScanResponse struct {
	Found    int       `json:"found"`
	New      int       `json:"new"`
	Projects []Project `json:"projects"`
}

// IntegrateResponse is the shape returned by POST /api/v1/projects/{id}/integrate.
type IntegrateResponse struct {
	Success      bool            `json:"success"`
	FilesCreated []string        `json:"files_created,omitempty"`
	FilesUpdated []string        `json:"files_updated,omitempty"`
	Message      string          `json:"message,omitempty"`
	Extra        json.RawMessage `json:"-"`
}
