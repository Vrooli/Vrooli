package reconcile

import (
	"fmt"
	"path/filepath"
	"react-component-library/internal/availability"
	"sort"
	"strings"

	"react-component-library/internal/gates"
)

type Verdict string

const (
	VerdictMatches       Verdict = "matches"
	VerdictResolvedLocal Verdict = "resolved-local"
	VerdictDrifted       Verdict = "drifted"
	VerdictMissing       Verdict = "missing"
	VerdictExtra         Verdict = "extra"
	VerdictUnverifiable  Verdict = "unverifiable"
)

var VerdictVocabulary = [...]Verdict{VerdictMatches, VerdictDrifted, VerdictMissing, VerdictExtra, VerdictUnverifiable, VerdictResolvedLocal}

type Coverage struct {
	Resolved, ResolvedLocal, LibraryBacked, Local         int
	Built, Declared, Invented, Missing, Unresolved, Total int
	BuiltPercent                                          float64
	Status                                                string
}

type RegionVerdict struct {
	Result
	Verdict      Verdict
	Finding      gates.Finding
	Availability availability.Result
}
type Verification struct {
	Regions  []RegionVerdict
	Coverage Coverage
	Passes   bool
}

func Verify(scenariosRoot, scenario, page string, joins []Result, placements map[string]Fill, availabilityByRegion ...map[string]availability.Result) Verification {
	owner := filepath.Join(scenariosRoot, scenario, "experience", "pages", page+".json")
	verification := Verification{Passes: true}
	available := map[string]availability.Result{}
	if len(availabilityByRegion) > 0 && availabilityByRegion[0] != nil {
		available = availabilityByRegion[0]
	}
	// Include every declared region, even when no placement was authored. An
	// unknown placement cannot disappear from the denominator either.
	seen := map[string]int{}
	for _, join := range joins {
		if !join.Extra && join.Region != "" {
			seen[join.Region]++
		}
	}
	var missing []string
	for region := range placements {
		if seen[region] == 0 {
			missing = append(missing, region)
		}
	}
	sort.Strings(missing)
	for _, region := range missing {
		joins = append(joins, Result{Region: region, Required: true, ReasonCode: "undeclared_region", Reason: "placement has no declared semantic region"})
	}
	counted := map[string]bool{}
	for _, join := range joins {
		fill := placements[join.Region]
		join.SelectedAsset = fill.Asset
		join.SelectedVersion = fill.Version
		resolution := available[join.Region]
		verdict := verdictFor(join)
		if verdict == VerdictMatches && (resolution.CatalogID != fill.Asset || resolution.Version != fill.Version) {
			verdict = VerdictUnverifiable
			join.ReasonCode = "availability_identity_mismatch"
			join.Reason = "build evidence does not identify the selected asset and version"
		}
		if verdict == VerdictMatches && !resolution.IsBuilt() {
			verdict = VerdictUnverifiable
			join.ReasonCode = resolution.ReasonCode
			join.Reason = resolution.Reason
		}
		if !join.Extra && join.Region != "" {
			if seen[join.Region] > 1 {
				verdict = VerdictUnverifiable
				join.ReasonCode = "duplicate_region"
				join.Reason = "semantic region appears more than once"
				verification.Passes = false
			}
			if counted[join.Region] {
				verdict = VerdictUnverifiable
				join.ReasonCode = "duplicate_region"
				join.Reason = "semantic region appears more than once"
				verification.Passes = false
			} else {
				counted[join.Region] = true
				verification.Coverage.Total++
				if join.Proven && !join.Heuristic && join.FilePath != "" && seen[join.Region] == 1 {
					verification.Coverage.Resolved++
				}
				if join.LibraryAsset != "" {
					verification.Coverage.LibraryBacked++
				} else if join.LocalComponent != "" {
					verification.Coverage.Local++
				}
				switch {
				case verdict == VerdictMatches && resolution.IsBuilt():
					verification.Coverage.Built++
				case verdict == VerdictResolvedLocal:
					verification.Coverage.ResolvedLocal++
				case fill.Placeholder != "":
					verification.Coverage.Invented++
				case join.Proven && !join.Heuristic && join.FilePath != "":
					verification.Coverage.Unresolved++
				case fill.Asset == "":
					verification.Coverage.Missing++
				case fill.Version == "":
					verification.Coverage.Declared++
				default:
					verification.Coverage.Unresolved++
				}
			}
		}
		if strings.TrimSpace(join.Reason) == "" {
			join.Reason = verdictReason(join, verdict)
		}
		id := join.Region
		if id == "" {
			id = join.FilePath
		}
		finding := gates.Finding{Code: "sketch-region-" + string(verdict), Category: "sketch", AssetID: join.Region, CatalogID: id, Scope: gates.FindingScopeCorpus, Blocking: false, Owner: owner, Severity: gates.FindingSeverityWarning, Message: fmt.Sprintf("scope %q is %s: %s", id, verdict, join.Reason), File: owner}
		verification.Regions = append(verification.Regions, RegionVerdict{Result: join, Verdict: verdict, Finding: finding, Availability: resolution})
		// Scoped verification fails on required unknowns; fleet findings stay
		// advisory and are not silently promoted to global blockers.
		if verdict == VerdictDrifted || (join.Required && verdict != VerdictMatches && verdict != VerdictResolvedLocal) {
			verification.Passes = false
		}
	}
	verification.Coverage.Status = "applicable"
	if verification.Coverage.Total == 0 {
		verification.Coverage.Status = "not_applicable"
	} else {
		verification.Coverage.BuiltPercent = 100 * float64(verification.Coverage.Built) / float64(verification.Coverage.Total)
	}
	verification.Passes = verification.Passes && verification.Coverage.Resolved > 0
	return verification
}

func verdictFor(result Result) Verdict {
	if result.Extra {
		return VerdictExtra
	}
	if result.Heuristic || !result.Proven {
		if result.FilePath == "" && result.ReasonCode != "ambiguous_binding" && result.ReasonCode != "source_unavailable" && result.Reason != "placement is a placeholder" && result.Reason != "scenario has no scannable files in a declared UI slot" {
			return VerdictMissing
		}
		return VerdictUnverifiable
	}
	if result.FilePath == "" {
		return VerdictMissing
	}
	if result.Provenance == ProvenanceUnknown {
		return VerdictUnverifiable
	}
	if result.Provenance == ProvenanceCustom && result.SelectedAsset == "" {
		return VerdictResolvedLocal
	}
	if result.SelectedAsset == "" || result.SelectedVersion == "" {
		return VerdictUnverifiable
	}
	if result.Provenance == ProvenanceCustom {
		return VerdictDrifted
	}
	if result.ObservedAsset != result.SelectedAsset || result.ObservedVersion != result.SelectedVersion {
		return VerdictDrifted
	}
	if result.Provenance == ProvenanceAdoptedModified {
		return VerdictDrifted
	}
	if result.Provenance == ProvenanceAdoptedUnmodified {
		return VerdictMatches
	}
	return VerdictUnverifiable
}

// Every verdict is an actionable observation, including successful joins.
func verdictReason(result Result, verdict Verdict) string {
	switch verdict {
	case VerdictResolvedLocal:
		return fmt.Sprintf("resolved to %s code in %s through %s", result.Provenance, result.FilePath, result.JoinRule)
	case VerdictMatches:
		return fmt.Sprintf("%s implements selected asset %s@%s with verified build evidence", result.FilePath, result.SelectedAsset, result.SelectedVersion)
	case VerdictDrifted:
		return fmt.Sprintf("%s has %s provenance (%s@%s), which differs from selected %s@%s", result.FilePath, result.Provenance, result.ObservedAsset, result.ObservedVersion, result.SelectedAsset, result.SelectedVersion)
	case VerdictExtra:
		return fmt.Sprintf("source %s has no declared page region", result.FilePath)
	case VerdictMissing:
		return "no proven source binding identifies this region"
	default:
		if result.FilePath != "" {
			return fmt.Sprintf("source %s is located but %s provenance or selected-version evidence remains unverified", result.FilePath, result.Provenance)
		}
		return "source binding could not be verified"
	}
}
