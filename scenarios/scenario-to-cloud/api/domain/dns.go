// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

// DNSPolicy controls how strictly DNS checks are enforced.
type DNSPolicy string

const (
	DNSPolicyRequired DNSPolicy = "required"
	DNSPolicyWarn     DNSPolicy = "warn"
	DNSPolicySkip     DNSPolicy = "skip"
)

// DNSLookupErrorKind categorizes DNS lookup failures.
type DNSLookupErrorKind string

const (
	DNSLookupNotFound    DNSLookupErrorKind = "not_found"
	DNSLookupTimeout     DNSLookupErrorKind = "timeout"
	DNSLookupInvalidHost DNSLookupErrorKind = "invalid_host"
	DNSLookupUnknown     DNSLookupErrorKind = "unknown"
)

// DNSLookupError captures a classified DNS lookup error.
type DNSLookupError struct {
	Kind    DNSLookupErrorKind `json:"kind"`
	Message string             `json:"message"`
}

func (e DNSLookupError) Error() string {
	return e.Message
}

// DNSLookupResult captures a DNS lookup attempt.
type DNSLookupResult struct {
	Host  string          `json:"host"`
	IPs   []string        `json:"ips,omitempty"`
	Error *DNSLookupError `json:"error,omitempty"`
}

// DNSComparisonResult describes a DNS comparison between a domain and VPS host.
type DNSComparisonResult struct {
	Domain      DNSLookupResult `json:"domain"`
	VPS         DNSLookupResult `json:"vps"`
	PointsToVPS bool            `json:"points_to_vps"`
}

// DNSARecordHint provides structured guidance for A record updates.
type DNSARecordHint struct {
	Domain          string   `json:"domain"`
	TargetIP        string   `json:"target_ip"`
	Providers       []string `json:"providers,omitempty"`
	PropagationNote string   `json:"propagation_note,omitempty"`
}

// DNSRecordValue captures a DNS record value with TTL.
type DNSRecordValue struct {
	Value string `json:"value"`
	TTL   uint32 `json:"ttl,omitempty"`
}

// DNSMXRecord captures an MX record with priority.
type DNSMXRecord struct {
	Host     string `json:"host"`
	Priority uint16 `json:"priority"`
	TTL      uint32 `json:"ttl,omitempty"`
}

// DNSRecordSet groups records by type for a domain.
type DNSRecordSet struct {
	Domain string           `json:"domain"`
	A      []DNSRecordValue `json:"a,omitempty"`
	AAAA   []DNSRecordValue `json:"aaaa,omitempty"`
	CNAME  []DNSRecordValue `json:"cname,omitempty"`
	MX     []DNSMXRecord    `json:"mx,omitempty"`
	TXT    []DNSRecordValue `json:"txt,omitempty"`
	NS     []DNSRecordValue `json:"ns,omitempty"`
}

// ReachabilityResult represents the result of a single reachability check.
type ReachabilityResult struct {
	Target    string             `json:"target"`
	Type      string             `json:"type"` // "host" or "domain"
	Reachable bool               `json:"reachable"`
	Message   string             `json:"message,omitempty"`
	Hint      string             `json:"hint,omitempty"`
	ErrorKind DNSLookupErrorKind `json:"error_kind,omitempty"`
}
