package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"business-health/internal/extraction"

	intent "intent-go"
)

// readmePhrases are the registry-contract concepts requirements/README.md
// must still explain (absorbed from the retired PRD control-tower's requirements_readme
// rule). The check is textual, not a parse — the README is prose.
var readmePhrases = []string{"operational target", "auto-sync", "validation"}

// readmeCheck emits requirements_readme when requirements/README.md is
// missing or has drifted away from explaining the registry contract.
type readmeCheck struct{}

func (readmeCheck) Name() string { return "requirements-readme" }

func (readmeCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if !c.Registry.Present {
		return nil // prd_missing_requirements covers the whole tree
	}
	path := filepath.Join(c.ScenarioDir, "requirements", "README.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return []intent.Finding{{
			Code:       "requirements_readme",
			Severity:   "warning",
			Message:    "requirements/README.md is missing — the registry ships without its contract explanation.",
			Suggestion: "Run `business-health fix " + c.Scenario + " --apply` to restore the canonical README.",
			Locations:  []string{"requirements/README.md"},
			Provenance: "business-health",
		}}
	}
	content := strings.ToLower(string(data))
	var missing []string
	for _, phrase := range readmePhrases {
		if !strings.Contains(content, phrase) {
			missing = append(missing, phrase)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []intent.Finding{{
		Code:       "requirements_readme",
		Severity:   "warning",
		Message:    fmt.Sprintf("requirements/README.md no longer explains: %s.", strings.Join(missing, ", ")),
		Suggestion: "Restore the canonical registry contract explanation (or re-run the fixer) and keep scenario-specific notes below it.",
		Locations:  []string{"requirements/README.md"},
		Provenance: "business-health",
	}}
}
