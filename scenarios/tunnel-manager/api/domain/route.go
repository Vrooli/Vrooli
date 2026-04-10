package domain

import "time"

// Route represents a published tunnel route mapping a subdomain to a local scenario port.
type Route struct {
	ID           int       `json:"id"`
	Subdomain    string    `json:"subdomain"`
	ScenarioName string    `json:"scenario_name"`
	LocalPort    int       `json:"local_port"`
	HealthPath   string    `json:"health_path"`
	PublicURL    string    `json:"public_url"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RouteInput is the create/update payload for routes.
type RouteInput struct {
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	LocalPort    int    `json:"local_port"`
	HealthPath   string `json:"health_path,omitempty"`
	PublicURL    string `json:"public_url,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

// CloudflaredIngress represents a single ingress rule in cloudflared config.
type CloudflaredIngress struct {
	Hostname string `yaml:"hostname,omitempty"`
	Service  string `yaml:"service"`
}

// CloudflaredConfig represents a cloudflared config.yml.
type CloudflaredConfig struct {
	Tunnel          string               `yaml:"tunnel,omitempty"`
	CredentialsFile string               `yaml:"credentials-file,omitempty"`
	WarpRouting     map[string]any       `yaml:"warp-routing,omitempty"`
	Ingress         []CloudflaredIngress `yaml:"ingress"`
}

// ManagementMode represents the tunnel management mode. [REQ:CFAPI-006]
type ManagementMode string

const (
	ModeLocal  ManagementMode = "local"
	ModeRemote ManagementMode = "remote"
)
