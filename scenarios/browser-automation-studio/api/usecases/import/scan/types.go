package scan

// ScanMode defines the type of BAS object to scan for.
type ScanMode string

const (
	ScanModeProjects  ScanMode = "projects"
	ScanModeWorkflows ScanMode = "workflows"
	ScanModeAssets    ScanMode = "assets"
	ScanModeFiles     ScanMode = "files"
)

// ScanRequest describes a filesystem scan request.
type ScanRequest struct {
	Mode      string `json:"mode"`
	Path      string `json:"path,omitempty"`
	Depth     int    `json:"depth,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

// ScanEntry represents a scan entry for folders/files.
type ScanEntry struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	IsDir         bool   `json:"is_dir"`
	IsTarget      bool   `json:"is_target"`
	IsRegistered  bool   `json:"is_registered"`
	RegisteredID  string `json:"registered_id,omitempty"`
	SuggestedName string `json:"suggested_name,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	SizeBytes     int64  `json:"size_bytes,omitempty"`
}

// ScanResponse contains scan results for a directory.
type ScanResponse struct {
	Path        string      `json:"path"`
	Parent      *string     `json:"parent,omitempty"`
	DefaultRoot string      `json:"default_root,omitempty"`
	Entries     []ScanEntry `json:"entries"`
}
