package support

import (
	"encoding/json"
	"time"
)

// HTTPResponse mirrors the shape returned by POST /api/v1/network/http.
type HTTPResponse struct {
	StatusCode     int               `json:"status_code"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	ResponseTimeMs int64             `json:"response_time_ms"`
	FinalURL       string            `json:"final_url,omitempty"`
	SSLInfo        json.RawMessage   `json:"ssl_info,omitempty"`
	RedirectChain  []string          `json:"redirect_chain,omitempty"`
}

// DNSAnswer mirrors an entry in DNSResponse.Answers.
type DNSAnswer struct {
	Name string `json:"name"`
	Type string `json:"type"`
	TTL  int    `json:"ttl"`
	Data string `json:"data"`
}

// DNSResponse mirrors the shape returned by POST /api/v1/network/dns.
type DNSResponse struct {
	Query          string      `json:"query"`
	RecordType     string      `json:"record_type"`
	Answers        []DNSAnswer `json:"answers"`
	ResponseTimeMs int64       `json:"response_time_ms"`
	Authoritative  bool        `json:"authoritative"`
	DNSSECValid    bool        `json:"dnssec_valid"`
}

// ConnectivityStatistics mirrors ConnectivityResponse.Statistics.
type ConnectivityStatistics struct {
	PacketsSent       int     `json:"packets_sent"`
	PacketsReceived   int     `json:"packets_received"`
	PacketLossPercent float64 `json:"packet_loss_percent"`
	MinRTTMs          float64 `json:"min_rtt_ms"`
	AvgRTTMs          float64 `json:"avg_rtt_ms"`
	MaxRTTMs          float64 `json:"max_rtt_ms"`
	StdDevRTTMs       float64 `json:"stddev_rtt_ms"`
}

// ConnectivityResponse mirrors POST /api/v1/network/test/connectivity.
type ConnectivityResponse struct {
	Target     string                 `json:"target"`
	TestType   string                 `json:"test_type"`
	Statistics ConnectivityStatistics `json:"statistics"`
	RouteHops  []string               `json:"route_hops,omitempty"`
}

// ScanPort is one port/result entry from POST /api/v1/network/scan.
type ScanPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service,omitempty"`
}

// ScanResponse mirrors POST /api/v1/network/scan.
type ScanResponse struct {
	Target  string     `json:"target"`
	Results []ScanPort `json:"results"`
}

// APIDefinition mirrors the shape returned by the /api/v1/api/definitions endpoints.
type APIDefinition struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	BaseURL               string                 `json:"base_url"`
	Version               string                 `json:"version,omitempty"`
	Specification         string                 `json:"specification,omitempty"`
	SpecDocument          map[string]interface{} `json:"spec_document,omitempty"`
	AuthenticationMethods []string               `json:"authentication_methods,omitempty"`
	RateLimits            map[string]interface{} `json:"rate_limits,omitempty"`
	EndpointsCount        int                    `json:"endpoints_count,omitempty"`
	LastValidated         *time.Time             `json:"last_validated,omitempty"`
	ValidationStatus      string                 `json:"validation_status,omitempty"`
	DocumentationURL      string                 `json:"documentation_url,omitempty"`
	CreatedAt             *time.Time             `json:"created_at,omitempty"`
	UpdatedAt             *time.Time             `json:"updated_at,omitempty"`
}

// Target mirrors one row from GET /api/v1/network/targets.
type Target struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	TargetType string     `json:"target_type"`
	Address    string     `json:"address"`
	Port       *int       `json:"port,omitempty"`
	Protocol   string     `json:"protocol,omitempty"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
}

// Alert mirrors one row from GET /api/v1/network/alerts.
type Alert struct {
	ID         string     `json:"id"`
	AlertType  string     `json:"alert_type"`
	Severity   string     `json:"severity"`
	Title      string     `json:"title"`
	Message    string     `json:"message,omitempty"`
	TargetName string     `json:"target_name,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
}
