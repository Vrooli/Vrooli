package knowledge

type InventoryRequest struct {
	Roots    []string `json:"roots"`
	Includes []string `json:"includes,omitempty"`
	Excludes []string `json:"excludes,omitempty"`
	MaxFiles int      `json:"max_files,omitempty"`
	MaxBytes int64    `json:"max_bytes,omitempty"`
}

type InventoryResponse struct {
	Candidates []InventoryCandidate `json:"candidates"`
}
type InventoryCandidate struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
}
type ProposalRequest struct {
	Candidates []InventoryCandidate `json:"candidates"`
}
type DispositionProposal struct {
	Source                string   `json:"source"`
	SourceSha256          string   `json:"source_sha256"`
	Action                string   `json:"action"`
	Decision              string   `json:"decision"`
	Confidence            float64  `json:"confidence"`
	Uncertainty           []string `json:"uncertainty"`
	Citations             []string `json:"citations"`
	Preservation          []string `json:"preservation"`
	AuthorizationRequired string   `json:"authorization_required"`
	PromptInjectionSignal bool     `json:"prompt_injection_signal"`
}
type RouteRequest struct {
	Proposal             DispositionProposal `json:"proposal"`
	ExpectedSourceSha256 string              `json:"expected_source_sha256"`
	Authorization        string              `json:"authorization,omitempty"`
	IdempotencyKey       string              `json:"idempotency_key"`
	DryRun               bool                `json:"dry_run"`
}
