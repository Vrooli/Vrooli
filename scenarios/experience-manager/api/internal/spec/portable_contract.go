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
	Claims []json.RawMessage `json:"claims"`
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
	_ = filepath.WalkDir(libraryRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != "experience-contract.json" {
			return nil
		}
		checkPortableContract(report, scenarioDir, path)
		return nil
	})
}

func checkPortableContract(report *Report, scenarioDir, path string) {
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
			claim.Type = "custom"
		}
		if claim.Tier == "" {
			claim.Tier = "manual"
		}
		checkClaimShape(report, location, claim)
		claims = append(claims, claim)
	}
	if !hasSubstantiveClaim(claims) {
		// A vacuous portable contract is valid catalog data but cannot earn a
		// machine-validated rung. Keep it visible as a warning here; the catalog
		// coverage engine owns rung eligibility, while route readiness and the
		// experience phase remain able to validate registered authored surfaces.
		report.add(CodeVacuousContract, SeverityWarning,
			fmt.Sprintf("portable contract %q is vacuous: no claim has an implemented evaluator", strings.TrimPrefix(location, string(filepath.Separator))),
			location, "Add a claim with an implemented evaluator or lower the contract to an explicitly unproven scaffold.")
	}
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
