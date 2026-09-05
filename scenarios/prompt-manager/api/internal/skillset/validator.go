package skillset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Role struct{ Role, Status, Source, Reason string }
type Result struct {
	Scenario, Status string
	Roles            []Role
	Findings         []string
}

// Validate inspects declaration integrity and sensor-facing structure. Skill
// prose quality remains owned by the dedicated skill-validation flow.
func Validate(repoRoot, scenario string) Result {
	r := Result{Scenario: scenario, Status: "ok"}
	root := filepath.Join(repoRoot, "scenarios", scenario)
	var service struct {
		Skills map[string]json.RawMessage `json:"skills"`
	}
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if err != nil {
		return unavailable(r, "skill-set.service_missing")
	}
	if err := json.Unmarshal(data, &service); err != nil {
		return unavailable(r, "skill-set.service_malformed")
	}
	var skillsBlock struct {
		Waivers map[string]json.RawMessage `json:"waivers"`
	}
	if raw, ok := service.Skills["waivers"]; ok {
		_ = json.Unmarshal(raw, &skillsBlock.Waivers)
	}
	_, improveDeclared := service.Skills["improve"]
	_, improveWaived := skillsBlock.Waivers["improve"]
	_, learningDeclared := service.Skills["learning"]
	owed := map[string]bool{
		"usage":  fileExists(filepath.Join(root, "cli", "manifest.json")),
		"improve": improveDeclared || improveWaived || learningDeclared,
		// Feature skills are opt-in and are never structurally owed.
		"feature": false,
	}
	for _, role := range []string{"usage", "improve", "feature"} {
		if !owed[role] {
			r.Roles = append(r.Roles, Role{Role: role, Status: "not_applicable", Reason: "role is not owed by this scenario's declarations"})
			continue
		}
		var declaration map[string]any
		if raw := service.Skills[role]; len(raw) > 0 {
			_ = json.Unmarshal(raw, &declaration)
		}
		if len(declaration) == 0 {
			if waiver, ok := skillsBlock.Waivers[role]; ok {
				var details struct {
					Reason     string `json:"reason"`
					DeclaredAt string `json:"declared_at"`
				}
				_ = json.Unmarshal(waiver, &details)
				status := "waived"
				if strings.TrimSpace(details.Reason) == "" || strings.TrimSpace(details.DeclaredAt) == "" {
					status = "invalid"
					r.Findings = append(r.Findings, "skill-set.waiver_invalid:"+role)
				}
				r.Roles = append(r.Roles, Role{Role: role, Status: status, Reason: strings.TrimSpace(details.Reason)})
				continue
			}
			r.Roles = append(r.Roles, Role{Role: role, Status: "absent", Reason: "role is owed but has no declaration"})
			r.Findings = append(r.Findings, "skill-set.role_missing:"+role)
			continue
		}
		source, _ := declaration["source"].(string)
		if source == "" {
			r.Roles = append(r.Roles, Role{Role: role, Status: "invalid", Reason: "declaration has no source"})
			r.Findings = append(r.Findings, "skill-set.source_missing:"+role)
			continue
		}
		clean := filepath.Clean(filepath.Join(root, filepath.FromSlash(source)))
		if _, err := os.Stat(clean); err != nil {
			r.Roles = append(r.Roles, Role{Role: role, Status: "invalid", Source: source, Reason: "declared source does not exist"})
			r.Findings = append(r.Findings, "skill-set.source_missing:"+role)
			continue
		}
		content, readErr := os.ReadFile(clean)
		if readErr != nil {
			r.Findings = append(r.Findings, "skill-set.source_unreadable:"+role)
			continue
		}
		text := string(content)
		if !strings.Contains(text, "metadata:") || !strings.Contains(text, "kind:") || !strings.Contains(text, "modes:") {
			r.Findings = append(r.Findings, "skill-set.dialect_invalid:"+role)
		}
		r.Roles = append(r.Roles, Role{Role: role, Status: "present", Source: source})
	}
	if len(r.Findings) > 0 {
		r.Status = "findings"
	}
	return r
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func unavailable(r Result, finding string) Result {
	r.Status = "unavailable"
	r.Findings = []string{finding}
	r.Roles = []Role{{Role: "usage", Status: "unavailable", Reason: fmt.Sprintf("%s cannot be read", r.Scenario)}}
	return r
}
func (r Result) String() string { return strings.TrimSpace(r.Status) }
