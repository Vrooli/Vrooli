package bundle

// PackageResult represents the result of a bundle packaging operation.
type PackageResult struct {
	BundleDir       string
	ManifestPath    string
	RuntimeBinaries map[string]string
	CopiedArtifacts []string
	TotalSizeBytes  int64
	TotalSizeHuman  string
	SizeWarning     *SizeWarning
}

// SizeWarning represents a warning about bundle size.
type SizeWarning struct {
	Level      string          `json:"level"` // "warning" or "critical"
	Message    string          `json:"message"`
	TotalBytes int64           `json:"total_bytes"`
	TotalHuman string          `json:"total_human"`
	LargeFiles []LargeFileInfo `json:"large_files,omitempty"`
}

// LargeFileInfo describes a file contributing significantly to bundle size.
type LargeFileInfo struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SizeHuman string `json:"size_human"`
}

// PackageRequest represents a request to package a bundle.
type PackageRequest struct {
	AppPath            string   `json:"app_path"`
	BundleManifestPath string   `json:"bundle_manifest_path"`
	Platforms          []string `json:"platforms"`
	Store              string   `json:"store"`
	Enterprise         bool     `json:"enterprise"`
}

// PackageResponse represents the response from a package operation.
type PackageResponse struct {
	Status          string            `json:"status"`
	BundleDir       string            `json:"bundle_dir"`
	Manifest        string            `json:"manifest"`
	RuntimeBinaries map[string]string `json:"runtime_binaries"`
	Artifacts       []string          `json:"artifacts"`
	TotalSizeBytes  int64             `json:"total_size_bytes"`
	TotalSizeHuman  string            `json:"total_size_human"`
	Timestamp       string            `json:"timestamp"`
	SizeWarning     *SizeWarning      `json:"size_warning,omitempty"`
}
