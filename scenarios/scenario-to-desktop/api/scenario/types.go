package scenario

import "time"

// DesktopBuildArtifact describes an installer produced for a specific platform.
type DesktopBuildArtifact struct {
	Platform     string `json:"platform"`
	FileName     string `json:"file_name"`
	SizeBytes    int64  `json:"size_bytes"`
	ModifiedAt   string `json:"modified_at"`
	AbsolutePath string `json:"absolute_path"`
	RelativePath string `json:"relative_path"`
}

// DesktopConnectionConfig holds scenario connection settings.
type DesktopConnectionConfig struct {
	Mode     string `json:"mode,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

// ScenarioDesktopStatus represents the aggregated desktop build status for a scenario.
type ScenarioDesktopStatus struct {
	Name                  string                   `json:"name"`
	DisplayName           string                   `json:"display_name,omitempty"`
	ServiceDisplay        string                   `json:"service_display_name,omitempty"`
	ServiceDesc           string                   `json:"service_description,omitempty"`
	ServiceIconPath       string                   `json:"service_icon_path,omitempty"`
	HasDesktop            bool                     `json:"has_desktop"`
	DesktopPath           string                   `json:"desktop_path,omitempty"`
	Version               string                   `json:"version,omitempty"`
	Platforms             []string                 `json:"platforms,omitempty"`
	Built                 bool                     `json:"built,omitempty"`
	DistPath              string                   `json:"dist_path,omitempty"`
	LastModified          string                   `json:"last_modified,omitempty"`
	PackageSize           int64                    `json:"package_size,omitempty"`
	ConnectionConfig      *DesktopConnectionConfig `json:"connection_config,omitempty"`
	BuildArtifacts        []DesktopBuildArtifact   `json:"build_artifacts,omitempty"`
	ArtifactsSource       string                   `json:"artifacts_source,omitempty"`
	ArtifactsPath         string                   `json:"artifacts_path,omitempty"`
	ArtifactsExpectedPath string                   `json:"artifacts_expected_path,omitempty"`
	RecordID              string                   `json:"record_id,omitempty"`
	RecordOutputPath      string                   `json:"record_output_path,omitempty"`
	RecordLocationMode    string                   `json:"record_location_mode,omitempty"`
	RecordUpdatedAt       string                   `json:"record_updated_at,omitempty"`
}

// ScenarioStats contains aggregate statistics about scenarios.
type ScenarioStats struct {
	Total       int `json:"total"`
	WithDesktop int `json:"with_desktop"`
	Built       int `json:"built"`
	WebOnly     int `json:"web_only"`
}

// ListResponse is the HTTP response for listing scenarios.
type ListResponse struct {
	Scenarios []ScenarioDesktopStatus `json:"scenarios"`
	Stats     *ScenarioStats          `json:"stats"`
}

// scenarioServiceInfo holds service.json metadata.
type scenarioServiceInfo struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// distArtifactScan holds results of scanning a dist directory.
type distArtifactScan struct {
	artifacts    []DesktopBuildArtifact
	platforms    []string
	totalSize    int64
	lastModified string
}

// latestRecordTime returns the most recent timestamp from a record.
func latestRecordTime(rec *DesktopAppRecord) time.Time {
	if rec == nil {
		return time.Time{}
	}
	if !rec.UpdatedAt.IsZero() {
		return rec.UpdatedAt
	}
	return rec.CreatedAt
}

// recordTimestamp formats the record's latest timestamp.
func recordTimestamp(rec *DesktopAppRecord) string {
	ts := latestRecordTime(rec)
	if ts.IsZero() {
		return ""
	}
	return ts.Format("2006-01-02 15:04:05")
}

// recordOutputPath returns the best available output path from a record.
func recordOutputPath(rec *DesktopAppRecord) string {
	if rec == nil {
		return ""
	}
	if rec.OutputPath != "" {
		return rec.OutputPath
	}
	if rec.StagingPath != "" {
		return rec.StagingPath
	}
	if rec.CustomPath != "" {
		return rec.CustomPath
	}
	return rec.DestinationPath
}
