// Package system provides system-level services including Wine management.
package system

import "context"

// WineService manages Wine installation for Windows cross-compilation.
type WineService interface {
	// IsInstalled checks if Wine is available.
	IsInstalled() bool

	// GetVersion returns the installed Wine version.
	GetVersion() string

	// CheckStatus returns the current Wine installation status.
	CheckStatus(ctx context.Context) (*WineCheckResponse, error)

	// StartInstallation begins an async Wine installation.
	StartInstallation(ctx context.Context, method string) (string, error)

	// GetInstallStatus returns the status of an installation.
	GetInstallStatus(installID string) (*WineInstallStatus, bool)
}

// TemplateService provides template listing and retrieval.
type TemplateService interface {
	// ListTemplates returns all available templates.
	ListTemplates(ctx context.Context) ([]TemplateInfo, error)

	// GetTemplate retrieves a specific template configuration.
	GetTemplate(ctx context.Context, templateType string) (map[string]interface{}, error)
}

// BuildStore provides access to build statistics.
type BuildStore interface {
	// Snapshot returns a copy of all build statuses.
	Snapshot() map[string]*BuildStatus
}

// BuildStatus represents a build status for statistics.
type BuildStatus struct {
	Status string
}
