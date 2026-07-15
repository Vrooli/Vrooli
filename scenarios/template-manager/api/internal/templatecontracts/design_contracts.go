package templatecontracts

type DesignCopyRule struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type DesignAdapterManifest struct {
	ID       string           `json:"id,omitempty"`
	Copy     []DesignCopyRule `json:"copy,omitempty"`
	Requires string           `json:"requires,omitempty"`
}

type DesignKitAdapter struct {
	Path     string   `json:"path"`
	Supports []string `json:"supports,omitempty"`
}

type DesignKitManifest struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Version     string                      `json:"version"`
	Default     bool                        `json:"default,omitempty"`
	Description string                      `json:"description,omitempty"`
	Tags        []string                    `json:"tags,omitempty"`
	Adapters    map[string]DesignKitAdapter `json:"adapters,omitempty"`
}

type DesignKitInfo struct {
	ID       string
	Path     string
	Manifest DesignKitManifest
	Missing  bool
}

type DesignValidationIssue struct {
	Kit     string `json:"kit,omitempty"`
	Adapter string `json:"adapter,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// DesignKitValidationResult is the per-kit verdict: Status is "pass" when the
// kit produced no issues, "fail" otherwise, and Issues is that kit's subset of
// the flat report.
type DesignKitValidationResult struct {
	Kit    string                  `json:"kit"`
	Status string                  `json:"status"`
	Issues []DesignValidationIssue `json:"issues,omitempty"`
}

type DesignValidationReport struct {
	Count int `json:"count"`
	// Issues is the flat list across every kit plus any fleet-level issue (e.g.
	// more than one kit marked default). Results groups the per-kit verdicts.
	Issues  []DesignValidationIssue     `json:"issues,omitempty"`
	Results []DesignKitValidationResult `json:"results,omitempty"`
}

type (
	DesignListRequest     struct{}
	DesignShowRequest     struct{ ID string }
	DesignValidateRequest struct {
		ID  string
		All bool
	}
)
