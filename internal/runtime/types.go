package runtime

import "github.com/vrooli/vrooli/internal/hostreq"

type SupportClass string

const (
	SupportSupported     SupportClass = "supported"
	SupportUnsupported   SupportClass = "unsupported"
	SupportNotApplicable SupportClass = "not_applicable"
	SupportManualOnly    SupportClass = "manual_only"
)

type ItemStatus struct {
	Name             string               `json:"name"`
	Kind             hostreq.Kind         `json:"kind"`
	Command          string               `json:"command,omitempty"`
	Version          string               `json:"version,omitempty"`
	Installed        bool                 `json:"installed"`
	Applied          bool                 `json:"applied,omitempty"`
	Required         bool                 `json:"required"`
	InstallSupported bool                 `json:"install_supported"`
	PackageName      string               `json:"package_name,omitempty"`
	SupportClass     SupportClass         `json:"support_class"`
	Manual           bool                 `json:"manual"`
	Notes            []string             `json:"notes,omitempty"`
	Provenance       []hostreq.Provenance `json:"provenance,omitempty"`
}

type ToolStatus = ItemStatus
type SafeguardStatus = ItemStatus

type Report struct {
	Environment     string            `json:"environment"`
	Host            Host              `json:"host"`
	Tools           []ToolStatus      `json:"tools"`
	Safeguards      []SafeguardStatus `json:"safeguards,omitempty"`
	MissingRequired []string          `json:"missing_required,omitempty"`
	MissingOptional []string          `json:"missing_optional,omitempty"`
}
