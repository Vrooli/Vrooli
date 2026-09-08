package testquality

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed catalog.json
var catalogJSON []byte

type Rule struct {
	ID                     string      `json:"id"`
	Version                string      `json:"version"`
	Intent                 string      `json:"intent"`
	Implementation         string      `json:"implementation"`
	SupportProfiles        []string    `json:"supportProfiles"`
	TestKinds              []string    `json:"testKinds"`
	RequiredEvidence       []string    `json:"requiredEvidence"`
	DefaultEnforcement     Enforcement `json:"defaultEnforcement"`
	Severity               Severity    `json:"severity"`
	CanBlock               bool        `json:"canBlock"`
	PromotionPrerequisites []string    `json:"promotionPrerequisites"`
	CalibrationCaseIDs     []string    `json:"calibrationCaseIds"`
	DocumentationAnchor    string      `json:"documentationAnchor"`
	FailureSemantics       string      `json:"failureSemantics"`
}

type Catalog struct {
	Version string `json:"version"`
	Rules   []Rule `json:"rules"`
}

// LoadCatalog returns a fresh copy so callers cannot change catalog policy.
func LoadCatalog() (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(catalogJSON, &c); err != nil {
		return c, err
	}
	return c, c.Validate()
}

func (c Catalog) Validate() error {
	if c.Version == "" || len(c.Rules) == 0 {
		return fmt.Errorf("catalog version and rules required")
	}
	seen := map[string]bool{}
	for _, r := range c.Rules {
		if r.ID == "" || seen[r.ID] {
			return fmt.Errorf("missing or duplicate rule identity %q", r.ID)
		}
		seen[r.ID] = true
		if r.Version == "" || r.Intent == "" || r.FailureSemantics == "" || r.DocumentationAnchor != "rule-"+r.ID {
			return fmt.Errorf("rule %s lacks version, intent, failure semantics or canonical anchor", r.ID)
		}
		if len(r.SupportProfiles) == 0 || len(r.TestKinds) == 0 || len(r.RequiredEvidence) == 0 || len(r.CalibrationCaseIDs) == 0 {
			return fmt.Errorf("rule %s lacks applicability, evidence or calibration contract", r.ID)
		}
		if r.Implementation != "planned" && r.Implementation != "implemented" {
			return fmt.Errorf("rule %s has unknown implementation state", r.ID)
		}
		if r.DefaultEnforcement != Advisory {
			return fmt.Errorf("rule %s: blocking requires a separate profile promotion decision", r.ID)
		}
		if r.Severity != Info && r.Severity != Warning && r.Severity != Error {
			return fmt.Errorf("rule %s has unknown severity", r.ID)
		}
		for name, values := range map[string][]string{"profiles": r.SupportProfiles, "test kinds": r.TestKinds, "evidence": r.RequiredEvidence, "calibration": r.CalibrationCaseIDs, "promotion": r.PromotionPrerequisites} {
			seenValues := map[string]bool{}
			for _, value := range values {
				if strings.TrimSpace(value) == "" || seenValues[value] {
					return fmt.Errorf("rule %s has empty or duplicate %s value", r.ID, name)
				}
				seenValues[value] = true
			}
		}
		if r.CanBlock {
			for _, required := range []string{"reviewed-calibration", "reviewed-holdout", "native-profile-conformance", "false-positive-budget", "owner-promotion-decision"} {
				found := false
				for _, prerequisite := range r.PromotionPrerequisites {
					if prerequisite == required {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("rule %s lacks promotion prerequisite %s", r.ID, required)
				}
			}
		}
	}
	return nil
}

// EnforcementReference is generated solely from catalog metadata. It does not
// claim that planned capabilities or listed calibration cases have been proven.
func (c Catalog) EnforcementReference() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Test-quality enforcement reference\n\nGenerated from testquality/catalog.json (catalog %s). Do not edit by hand.\n\nDeclared profiles and calibration cases are not certification. Checked clean applies only to the named supported check; unknown is never a pass. Promotion requires separate recorded evidence.\n", c.Version)
	for _, r := range c.Rules {
		fmt.Fprintf(&b, "\n<a id=%q></a>\n\n## %s\n\n%s\n\n- Version: %s\n- Implementation: %s\n- Profiles: %s\n- Test kinds: %s\n- Required evidence: %s\n- Severity: %s\n- Default enforcement: %s\n- Eligible for promotion: %t\n- Promotion prerequisites: %s\n- Calibration cases: %s\n\n%s\n", r.DocumentationAnchor, r.ID, r.Intent, r.Version, r.Implementation, strings.Join(r.SupportProfiles, ", "), strings.Join(r.TestKinds, ", "), strings.Join(r.RequiredEvidence, ", "), r.Severity, r.DefaultEnforcement, r.CanBlock, strings.Join(r.PromotionPrerequisites, ", "), strings.Join(r.CalibrationCaseIDs, ", "), r.FailureSemantics)
	}
	return b.String()
}
