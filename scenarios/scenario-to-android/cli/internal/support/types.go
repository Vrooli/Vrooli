package support

// SDKStatus mirrors the shape returned by GET /api/v1/status.
type SDKStatus struct {
	AndroidSDK  string `json:"android_sdk"`
	Java        string `json:"java"`
	Gradle      string `json:"gradle"`
	Ready       bool   `json:"ready"`
	BuildSystem string `json:"build_system"`
}

// BuildMetrics mirrors the shape returned by GET /api/v1/metrics.
type BuildMetrics struct {
	TotalBuilds      int64   `json:"total_builds"`
	SuccessfulBuilds int64   `json:"successful_builds"`
	FailedBuilds     int64   `json:"failed_builds"`
	ActiveBuilds     int64   `json:"active_builds"`
	SuccessRate      float64 `json:"success_rate"`
	AverageDuration  float64 `json:"average_duration_seconds"`
	Uptime           float64 `json:"uptime_seconds"`
}

// BuildCreateResponse mirrors the shape returned by POST /api/v1/android/build.
type BuildCreateResponse struct {
	Success  bool              `json:"success"`
	APKPath  string            `json:"apk_path,omitempty"`
	BuildID  string            `json:"build_id"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    string            `json:"error,omitempty"`
}

// BuildStatus mirrors the shape returned by GET /api/v1/android/status/{buildID}.
type BuildStatus struct {
	Status   string   `json:"status"`
	Progress int      `json:"progress"`
	Logs     []string `json:"logs,omitempty"`
}
