// Package types contains the compatibility shape for legacy scenario playbook
// registries. It is deliberately a data-only package: provider execution and
// provider-specific clients do not belong in Test Genie.
package types

type Registry struct {
	Note       string   `json:"note,omitempty"`
	Generated  string   `json:"generated_at,omitempty"`
	Metadata   Metadata `json:"metadata,omitempty"`
	Playbooks  []Entry  `json:"playbooks,omitempty"`
	Deprecated []Entry  `json:"deprecated_playbooks,omitempty"`
	Scenario   string   `json:"scenario,omitempty"`
}

type Metadata struct {
	ExecutionMode string `json:"execution_mode,omitempty"`
}

type Entry struct {
	File         string   `json:"file"`
	Description  string   `json:"description,omitempty"`
	Order        string   `json:"order,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
	Fixtures     []string `json:"fixtures,omitempty"`
	Reset        string   `json:"reset,omitempty"`
}
