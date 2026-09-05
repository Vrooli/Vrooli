package support

import "time"

// GenerateResponse is returned by POST /api/v1/extension/generate.
type GenerateResponse struct {
	BuildID             string `json:"build_id"`
	ExtensionPath       string `json:"extension_path"`
	InstallInstructions string `json:"install_instructions"`
	TestCommand         string `json:"test_command"`
	Status              string `json:"status"`
}

// BuildStatus is returned by GET /api/v1/extension/status/{build_id}.
type BuildStatus struct {
	BuildID       string     `json:"build_id"`
	ScenarioName  string     `json:"scenario_name"`
	Status        string     `json:"status"`
	ExtensionPath string     `json:"extension_path"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	BuildLog      []string   `json:"build_log,omitempty"`
	ErrorLog      []string   `json:"error_log,omitempty"`
}

// BuildSummary is one entry in the /api/v1/extension/builds list.
type BuildSummary struct {
	BuildID       string     `json:"build_id"`
	ScenarioName  string     `json:"scenario_name"`
	TemplateType  string     `json:"template_type,omitempty"`
	Status        string     `json:"status"`
	ExtensionPath string     `json:"extension_path,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// BuildsResponse wraps the /api/v1/extension/builds response.
type BuildsResponse struct {
	Builds []BuildSummary `json:"builds"`
	Count  int            `json:"count"`
}

// Template is one template entry from /api/v1/extension/templates.
type Template struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name,omitempty"`
	Description string   `json:"description,omitempty"`
	Files       []string `json:"files,omitempty"`
	Source      string   `json:"source,omitempty"`
}

// TemplatesResponse wraps the /api/v1/extension/templates response.
type TemplatesResponse struct {
	Templates []Template `json:"templates"`
	Count     int        `json:"count"`
}

// TestSiteResult mirrors the ExtensionSiteResult API shape.
type TestSiteResult struct {
	Site           string   `json:"site"`
	Loaded         bool     `json:"loaded"`
	Errors         []string `json:"errors,omitempty"`
	ScreenshotPath string   `json:"screenshot_path,omitempty"`
	LoadTime       int      `json:"load_time_ms"`
}

// TestSummary mirrors ExtensionTestSummary.
type TestSummary struct {
	TotalTests  int     `json:"total_tests"`
	Passed      int     `json:"passed"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// TestResult mirrors ExtensionTestResult.
type TestResult struct {
	Success     bool             `json:"success"`
	TestResults []TestSiteResult `json:"test_results"`
	Summary     TestSummary      `json:"summary"`
	ReportTime  *time.Time       `json:"report_time,omitempty"`
}
