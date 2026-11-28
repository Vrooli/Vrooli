package sessionpermission

import (
	"sort"
	"strings"
)

type Capability string

const (
	CapabilityObserve                  Capability = "observe"
	CapabilityAccessibilityTree        Capability = "accessibility_tree"
	CapabilityControl                  Capability = "control"
	CapabilityApprovalReceipt          Capability = "approval_receipt"
	CapabilityHistoryRead              Capability = "history_read"
	CapabilityHistoryExport            Capability = "history_export"
	CapabilityHistoryRetention         Capability = "history_retention"
	CapabilityCredentialUse            Capability = "credential_use"
	CapabilityCreditBudget             Capability = "credit_budget"
	CapabilityByteBudget               Capability = "byte_budget"
	CapabilityExtensionInventory       Capability = "extension_inventory"
	CapabilityExtensionHostPermissions Capability = "extension_host_permissions"
	CapabilityExtensionLoad            Capability = "extension_load"
	CapabilityExtensionInstall         Capability = "extension_install"
	CapabilityHandoff                  Capability = "handoff"
	CapabilityReconnect                Capability = "reconnect"
	CapabilityPreserveUserState        Capability = "preserve_user_state"
	CapabilityAudioCapture             Capability = "audio_capture"
	CapabilityAudioDeviceEvidence      Capability = "audio_device_evidence"
	CapabilityIdentityCustomization    Capability = "identity_customization"
	CapabilityAppMonitorMetrics        Capability = "app_monitor_metrics"
	CapabilityAppMonitorTrace          Capability = "app_monitor_trace"
)

// BackendReport is derived from the adapter's live capability negotiation.
// `Qualified` means the adapter has an owner-backed live receipt identified
// by EvidenceID; source-only flags are not enough.
type BackendReport struct {
	Backend      Backend               `json:"backend"`
	Qualified    bool                  `json:"qualified"`
	Capabilities []Capability          `json:"capabilities,omitempty"`
	Unavailable  map[Capability]string `json:"unavailable,omitempty"`
	EvidenceID   string                `json:"evidence_id,omitempty"`
}

type CapabilityNegotiation struct {
	Backend   Backend                `json:"backend"`
	Qualified bool                   `json:"qualified"`
	Accepted  bool                   `json:"accepted"`
	Available []Capability           `json:"available,omitempty"`
	Missing   []Capability           `json:"missing,omitempty"`
	Reasons   map[Capability]string  `json:"reasons,omitempty"`
	Identity  *IdentityCustomization `json:"identity_customization,omitempty"`
}

// IdentityCustomization distinguishes settings accepted by an adapter from
// values independently observed from its serving browser. Configured fields
// are not evidence that a site's page saw those values; only Observed fields
// have an owner readback in the current session contract.
type IdentityCustomization struct {
	Configured  []string          `json:"configured,omitempty"`
	Observed    []string          `json:"observed,omitempty"`
	Unsupported map[string]string `json:"unsupported,omitempty"`
}

// NegotiateCapabilities returns a truthful row for every backend. A backend
// is never selected from an unqualified, source-only capability declaration.
func NegotiateCapabilities(reports []BackendReport, required []Capability) []CapabilityNegotiation {
	rows := make([]CapabilityNegotiation, 0, len(reports))
	for _, report := range reports {
		available := make(map[Capability]bool, len(report.Capabilities))
		for _, capability := range report.Capabilities {
			available[capability] = true
		}
		qualified := report.Qualified && strings.TrimSpace(report.EvidenceID) != ""
		row := CapabilityNegotiation{
			Backend: report.Backend, Qualified: qualified,
			Reasons: make(map[Capability]string),
		}
		if qualified {
			row.Available = append([]Capability(nil), report.Capabilities...)
		}
		for _, capability := range required {
			if qualified && available[capability] {
				continue
			}
			row.Missing = append(row.Missing, capability)
			if reason := report.Unavailable[capability]; reason != "" {
				row.Reasons[capability] = reason
			} else if !qualified {
				row.Reasons[capability] = "backend capability has no live qualification receipt"
			} else {
				row.Reasons[capability] = "backend does not provide this capability"
			}
		}
		row.Accepted = qualified && len(row.Missing) == 0
		sort.Slice(row.Available, func(i, j int) bool { return row.Available[i] < row.Available[j] })
		sort.Slice(row.Missing, func(i, j int) bool { return row.Missing[i] < row.Missing[j] })
		if len(row.Reasons) == 0 {
			row.Reasons = nil
		}
		rows = append(rows, row)
	}
	return rows
}
