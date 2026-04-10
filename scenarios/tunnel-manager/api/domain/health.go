package domain

// DetailedHealth is the composite health response for cross-scenario consumption. [REQ:OBS-004]
type DetailedHealth struct {
	Status    string            `json:"status"`
	Tunnel    TunnelHealthInfo  `json:"tunnel"`
	Routes    []RouteHealthInfo `json:"routes"`
	Timestamp string            `json:"timestamp"`
}

// TunnelHealthInfo summarizes the tunnel's health state.
type TunnelHealthInfo struct {
	Ready        string `json:"ready"`
	Systemd      string `json:"systemd"`
	Score        int    `json:"score"`
	ReadyLatency int    `json:"ready_latency_ms,omitempty"`
}

// RouteHealthInfo summarizes a route's recent probe status.
type RouteHealthInfo struct {
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	Enabled      bool   `json:"enabled"`
	InternalUp   *bool  `json:"internal_up,omitempty"`
	ExternalUp   *bool  `json:"external_up,omitempty"`
}
