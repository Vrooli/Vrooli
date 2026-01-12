package system

import "time"

// WineCheckResponse describes Wine installation status.
type WineCheckResponse struct {
	Installed         bool                `json:"installed"`
	Version           string              `json:"version,omitempty"`
	Platform          string              `json:"platform"`
	RequiredFor       []string            `json:"required_for"`
	InstallMethods    []WineInstallMethod `json:"install_methods,omitempty"`
	RecommendedMethod string              `json:"recommended_method,omitempty"`
}

// WineInstallMethod describes an installation method for Wine.
type WineInstallMethod struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	RequiresSudo bool     `json:"requires_sudo"`
	Estimated    string   `json:"estimated"`
	Steps        []string `json:"steps"`
}

// WineInstallStatus tracks an ongoing Wine installation.
type WineInstallStatus struct {
	InstallID   string     `json:"install_id"`
	Status      string     `json:"status"` // pending, installing, completed, failed
	Method      string     `json:"method"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Log         []string   `json:"log"`
	ErrorLog    []string   `json:"error_log,omitempty"`
}

// WineInstallRequest is the request to start Wine installation.
type WineInstallRequest struct {
	Method string `json:"method"`
}

// WineInstallResponse is the response after starting installation.
type WineInstallResponse struct {
	InstallID string `json:"install_id"`
	Status    string `json:"status"`
	Method    string `json:"method"`
	StatusURL string `json:"status_url"`
}

// TemplateInfo describes an available template.
type TemplateInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Framework   string   `json:"framework"`
	UseCases    []string `json:"use_cases"`
	Features    []string `json:"features"`
	Complexity  string   `json:"complexity"`
	Examples    []string `json:"examples"`
}

// StatusResponse provides system status information.
type StatusResponse struct {
	Service             map[string]interface{} `json:"service"`
	Statistics          map[string]interface{} `json:"statistics"`
	Capabilities        []string               `json:"capabilities"`
	SupportedFrameworks []string               `json:"supported_frameworks"`
	SupportedTemplates  []string               `json:"supported_templates"`
	Endpoints           []map[string]string    `json:"endpoints"`
}

// TemplatesResponse is the response for listing templates.
type TemplatesResponse struct {
	Templates []TemplateInfo `json:"templates"`
	Count     int            `json:"count"`
}
