package projects

// InspectProjectRequest is the request for inspecting a project folder.
type InspectProjectRequest struct {
	FolderPath string `json:"folder_path"`
}

// InspectProjectResponse is the response from inspecting a project folder.
type InspectProjectResponse struct {
	FolderPath           string `json:"folder_path"`
	Exists               bool   `json:"exists"`
	IsDir                bool   `json:"is_dir"`
	HasBasMetadata       bool   `json:"has_bas_metadata"`
	MetadataError        string `json:"metadata_error,omitempty"`
	HasWorkflows         bool   `json:"has_workflows"`
	AlreadyIndexed       bool   `json:"already_indexed"`
	IndexedProjectID     string `json:"indexed_project_id,omitempty"`
	SuggestedName        string `json:"suggested_name,omitempty"`
	SuggestedDescription string `json:"suggested_description,omitempty"`
}

// ImportProjectRequest is the request for importing a project.
type ImportProjectRequest struct {
	FolderPath  string `json:"folder_path"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// ScanProjectsRequest is the request for scanning directories for projects.
type ScanProjectsRequest struct {
	Path  string `json:"path,omitempty"`
	Depth int    `json:"depth,omitempty"`
}

// ProjectEntry represents a folder entry when scanning.
type ProjectEntry struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	IsProject     bool   `json:"is_project"`
	IsRegistered  bool   `json:"is_registered"`
	ProjectID     string `json:"project_id,omitempty"`
	SuggestedName string `json:"suggested_name,omitempty"`
}

// ScanProjectsResponse is the response from scanning for projects.
type ScanProjectsResponse struct {
	Path                string         `json:"path"`
	Parent              *string        `json:"parent,omitempty"`
	DefaultProjectsRoot string         `json:"default_projects_root,omitempty"`
	Entries             []ProjectEntry `json:"entries"`
}
