// Package edge provides edge and TLS management commands for the CLI.
package edge

// DNSCheckResponse is the response from DNS check.
type DNSCheckResponse struct {
	DeploymentID string          `json:"deployment_id"`
	Domain       string          `json:"domain"`
	Healthy      bool            `json:"healthy"`
	Records      []DNSCheckItem  `json:"records"`
	Issues       []string        `json:"issues,omitempty"`
	Timestamp    string          `json:"timestamp"`
}

// DNSCheckItem represents a single DNS record check result.
type DNSCheckItem struct {
	Type     string `json:"type"` // A, AAAA, CNAME, etc.
	Name     string `json:"name"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
}

// DNSRecordsResponse is the response from DNS records listing.
type DNSRecordsResponse struct {
	DeploymentID string      `json:"deployment_id"`
	Domain       string      `json:"domain"`
	Records      []DNSRecord `json:"records"`
	Timestamp    string      `json:"timestamp"`
}

// DNSRecord represents a DNS record.
type DNSRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
}

// CaddyRequest is the request for Caddy control actions.
type CaddyRequest struct {
	Action string `json:"action"` // reload, restart, status, validate
}

// CaddyResponse is the response from Caddy control.
type CaddyResponse struct {
	Success   bool   `json:"success"`
	Action    string `json:"action"`
	Message   string `json:"message,omitempty"`
	Config    string `json:"config,omitempty"` // For validate/status actions
	Timestamp string `json:"timestamp"`
}

// TLSInfoResponse is the response from TLS info.
type TLSInfoResponse struct {
	DeploymentID string            `json:"deployment_id"`
	Certificates []CertificateInfo `json:"certificates"`
	Timestamp    string            `json:"timestamp"`
}

// CertificateInfo represents a TLS certificate.
type CertificateInfo struct {
	Domain     string   `json:"domain"`
	Issuer     string   `json:"issuer"`
	ValidFrom  string   `json:"valid_from"`
	ValidUntil string   `json:"valid_until"`
	DaysLeft   int      `json:"days_left"`
	Expired    bool     `json:"expired"`
	AutoRenew  bool     `json:"auto_renew"`
	SANs       []string `json:"sans,omitempty"` // Subject Alternative Names
}

// TLSRenewRequest is the request for TLS renewal.
type TLSRenewRequest struct {
	Domain string `json:"domain,omitempty"` // Empty = renew all
	Force  bool   `json:"force,omitempty"`
}

// TLSRenewResponse is the response from TLS renewal.
type TLSRenewResponse struct {
	Success   bool            `json:"success"`
	Results   []RenewalResult `json:"results,omitempty"`
	Message   string          `json:"message,omitempty"`
	Timestamp string          `json:"timestamp"`
}

// RenewalResult represents the result of renewing a single certificate.
type RenewalResult struct {
	Domain  string `json:"domain"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
