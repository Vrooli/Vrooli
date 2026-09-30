package spec

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"experience-manager/internal/claimtypes"
)

// portableContract is the catalog form of an experience contract. It uses the
// same Claim shape and validation rules as registered scenario documents, but
// deliberately keeps the catalog envelope independent from the scenario index.
type portableContract struct {
	Kind   string            `json:"kind"`
	Ref    string            `json:"ref"`
	Claims []json.RawMessage `json:"claims"`
}

type vacuousAllowlist struct {
	Entries []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"entries"`
}

// checkPortableLibraryContracts makes the catalog part of the parser's input
// surface. A contract is not trustworthy merely because it is not registered
// in a scenario index; malformed and vacuous catalog contracts must remain
// visible at their real library path.
func checkPortableLibraryContracts(report *Report, scenarioDir string) {
	libraryRoot := filepath.Join(scenarioDir, "library")
	if info, err := os.Stat(libraryRoot); err != nil || !info.IsDir() {
		return
	}
	allowlisted := loadVacuousAllowlist(scenarioDir)
	_ = filepath.WalkDir(libraryRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Retired versions remain on disk as provenance backups, but they are no
		// longer active catalog inputs. Parsing them here turns historical
		// contracts into current maturity debt and makes retirement non-terminal.
		if entry.IsDir() {
			if entry.Name() == ".retired" {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Name() != "experience-contract.json" {
			return nil
		}
		checkPortableContract(report, scenarioDir, path, allowlisted)
		return nil
	})
}

func checkPortableContract(report *Report, scenarioDir, path string, allowlisted map[string]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("read portable contract: %v", err), rel(scenarioDir, path), "Repair or remove the unreadable catalog contract.")
		return
	}
	var contract portableContract
	if err := json.Unmarshal(data, &contract); err != nil {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("invalid portable contract JSON: %v", err), rel(scenarioDir, path), "Repair the JSON syntax.")
		return
	}
	location := rel(scenarioDir, path)
	if contract.Kind == "experience-reference" {
		ref := strings.TrimSpace(contract.Ref)
		if ref == "" {
			report.add(CodeSchemaInvalid, SeverityError, "experience reference is missing ref", location, "Point the version-scoped reference at the canonical experience component spec.")
			return
		}
		target := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(ref)))
		relTarget, relErr := filepath.Rel(scenarioDir, target)
		if relErr != nil || relTarget == ".." || strings.HasPrefix(relTarget, ".."+string(filepath.Separator)) {
			report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("experience reference %q escapes the scenario root", ref), location, "Point the reference at a canonical experience component under experience/components.")
			return
		}
		if _, statErr := os.Stat(target); statErr != nil {
			report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("experience reference %q is unreadable: %v", ref, statErr), location, "Restore the canonical experience component spec or repair the reference.")
		}
		return
	}
	claims := make([]Claim, 0, len(contract.Claims))
	for _, rawClaim := range contract.Claims {
		var claim Claim
		if err := json.Unmarshal(rawClaim, &claim); err != nil {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("invalid portable claim: %v", err), location, "Repair the claim JSON.")
			continue
		}
		// Older catalog contracts used description-only manual claims. Normalize
		// that envelope at the parser boundary so the shared Claim validator can
		// still enforce the same tier and evaluator rules without losing intent.
		var legacy struct {
			Description string `json:"description"`
		}
		_ = json.Unmarshal(rawClaim, &legacy)
		if claim.Statement == "" {
			claim.Statement = legacy.Description
		}
		if claim.Type == "" {
			// Description-only legacy catalog claims remain visible as explicitly
			// aspirational review items. They must not silently re-enter the
			// retired custom vocabulary.
			claim.Type = "visual-review"
		}
		if claim.Tier == "" {
			claim.Tier = "manual"
		}
		if claim.Type == "custom" {
			report.add(CodeTierViolation, SeverityError, fmt.Sprintf("custom claim %q is retired", claim.ID), location, "Replace it with an implemented claim type or the explicit manual-review type.")
		}
		checkClaimShape(report, location, claim)
		claims = append(claims, claim)
	}
	if !hasSubstantiveClaim(claims) {
		severity := SeverityError
		reason := "Remove this entry from the legacy allowlist by adding an implemented machine claim."
		if allowlistedReason, ok := allowlisted[filepath.ToSlash(rel(scenarioDir, path))]; ok {
			severity = SeverityWarning
			reason = allowlistedReason
		}
		// A legacy allowlisted contract remains visible as a warning. A new or
		// changed contract is an error, so contract debt cannot enter through the
		// normal authoring path.
		report.add(CodeVacuousContract, severity,
			fmt.Sprintf("portable contract %q is vacuous: no claim has an implemented evaluator", strings.TrimPrefix(location, string(filepath.Separator))),
			location, reason)
	}
}

func loadVacuousAllowlist(scenarioDir string) map[string]string {
	path := filepath.Join(scenarioDir, "library", "vacuous-allowlist.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	var document vacuousAllowlist
	if json.Unmarshal(data, &document) != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(document.Entries))
	for _, entry := range document.Entries {
		entryPath := filepath.ToSlash(strings.TrimSpace(entry.Path))
		if entryPath != "" {
			out[entryPath] = strings.TrimSpace(entry.Reason)
		}
	}
	return out
}

func hasSubstantiveClaim(claims []Claim) bool {
	for _, claim := range claims {
		if claim.ID == "contract-present" || !claimtypes.IsImplemented(claim.Type) {
			continue
		}
		return true
	}
	return false
}
