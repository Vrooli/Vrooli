package assets

// UploadAssetRequest is the request for uploading an asset.
// Note: This is typically sent as multipart/form-data with the file.
type UploadAssetRequest struct {
	ProjectID string `json:"project_id"`
	Path      string `json:"path"` // Relative path within assets directory
}

// UploadAssetResponse is the response from uploading an asset.
type UploadAssetResponse struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type,omitempty"`
}

// ListAssetsRequest is the request for listing assets.
type ListAssetsRequest struct {
	ProjectID string `json:"project_id"`
	Path      string `json:"path,omitempty"` // Subdirectory to list (default: root)
}

// AssetEntry represents an asset file.
type AssetEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type,omitempty"`
	IsDir     bool   `json:"is_dir"`
}

// ListAssetsResponse is the response from listing assets.
type ListAssetsResponse struct {
	Path    string       `json:"path"`
	Entries []AssetEntry `json:"entries"`
}

// DeleteAssetRequest is the request for deleting an asset.
type DeleteAssetRequest struct {
	ProjectID string `json:"project_id"`
	Path      string `json:"path"`
}
