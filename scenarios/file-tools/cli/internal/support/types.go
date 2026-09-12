package support

import "time"

// HealthStatus mirrors a minimal slice of the /health response fields that
// the CLI surfaces directly. Unknown fields are collected via MapRows in the
// decoded map form for the operational report's triage section.
type HealthStatus struct {
	Status   string `json:"status,omitempty"`
	Version  string `json:"version,omitempty"`
	Database string `json:"database,omitempty"`
}

// CompressResponse mirrors the POST /api/v1/files/compress response.
type CompressResponse struct {
	OperationID       string  `json:"operation_id"`
	ArchivePath       string  `json:"archive_path"`
	OriginalSizeBytes int64   `json:"original_size_bytes"`
	CompressedSize    int64   `json:"compressed_size_bytes"`
	CompressionRatio  float64 `json:"compression_ratio"`
	FilesIncluded     int     `json:"files_included"`
	Checksum          string  `json:"checksum"`
}

// ExtractResponse mirrors the POST /api/v1/files/extract response.
type ExtractResponse struct {
	OperationID    string   `json:"operation_id"`
	ExtractedFiles []string `json:"extracted_files"`
	TotalFiles     int      `json:"total_files"`
	TotalSizeBytes int64    `json:"total_size_bytes"`
}

// ChecksumEntry is one file/checksum pair returned by /api/v1/files/checksum.
type ChecksumEntry struct {
	File      string `json:"file"`
	Checksum  string `json:"checksum"`
	Algorithm string `json:"algorithm"`
}

// ChecksumResponse mirrors the POST /api/v1/files/checksum response.
type ChecksumResponse struct {
	Results []ChecksumEntry `json:"results"`
	Total   int             `json:"total"`
}

// SplitResponse mirrors the POST /api/v1/files/split response.
type SplitResponse struct {
	OperationID string   `json:"operation_id"`
	Parts       []string `json:"parts"`
	TotalParts  int      `json:"total_parts"`
	ChunkSize   int64    `json:"chunk_size"`
}

// MergeResponse mirrors the POST /api/v1/files/merge response.
type MergeResponse struct {
	OperationID string `json:"operation_id"`
	OutputFile  string `json:"output_file"`
	MergedParts int    `json:"merged_parts"`
	TotalSize   int64  `json:"total_size"`
	Checksum    string `json:"checksum"`
}

// MetadataResponse mirrors the GET /api/v1/files/metadata?path= response.
type MetadataResponse struct {
	FilePath string                 `json:"file_path"`
	Size     int64                  `json:"size_bytes"`
	MimeType string                 `json:"mime_type"`
	ModTime  time.Time              `json:"modified_time"`
	Checksum map[string]string      `json:"checksums"`
	Metadata map[string]interface{} `json:"metadata"`
}

// FileOperationResponse mirrors the POST /api/v1/files/operation response.
type FileOperationResponse struct {
	OperationID string `json:"operation_id"`
	Operation   string `json:"operation"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	Status      string `json:"status"`
}
