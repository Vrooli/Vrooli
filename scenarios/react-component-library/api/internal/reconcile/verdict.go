package reconcile

import (
	"fmt"
	"path/filepath"

	"react-component-library/internal/gates"
)

type Verdict string

const (
	VerdictMatches      Verdict = "matches"
	VerdictDrifted      Verdict = "drifted"
	VerdictMissing      Verdict = "missing"
	VerdictExtra        Verdict = "extra"
	VerdictUnverifiable Verdict = "unverifiable"
)

var VerdictVocabulary = [...]Verdict{VerdictMatches, VerdictDrifted, VerdictMissing, VerdictExtra, VerdictUnverifiable}

type Coverage struct{ Built, Declared, Invented int }

type RegionVerdict struct {
	Result
	Verdict Verdict
	Finding gates.Finding
}

type Verification struct {
	Regions  []RegionVerdict
	Coverage Coverage
	Passes   bool
}

func Verify(scenariosRoot, scenario, page string, joins []Result, sketchPlacements map[string]Fill) Verification {
	owner := filepath.Join(scenariosRoot, scenario, "experience", "pages", page+".json")
	verification := Verification{Passes: true}
	for _, fill := range sketchPlacements {
		switch {
		case fill.Placeholder != "":
			verification.Coverage.Invented++
		case fill.Asset != "" && fill.Version != "":
			verification.Coverage.Built++
		case fill.Asset != "":
			verification.Coverage.Declared++
		}
	}
	for _, join := range joins {
		verdict := verdictFor(join)
		catalogID := join.Region
		if catalogID == "" {
			catalogID = join.FilePath
		}
		finding := gates.Finding{
			Code: "sketch-region-" + string(verdict), Category: "sketch", AssetID: join.Region,
			CatalogID: catalogID, Scope: gates.FindingScopeCorpus, Blocking: false,
			Owner: owner, Severity: gates.FindingSeverityWarning,
			Message: fmt.Sprintf("scope %q is %s", catalogID, verdict), File: owner,
		}
		verification.Regions = append(verification.Regions, RegionVerdict{Result: join, Verdict: verdict, Finding: finding})
		if verdict == VerdictDrifted || (verdict == VerdictMissing && join.Required) {
			verification.Passes = false
		}
	}
	return verification
}

func verdictFor(result Result) Verdict {
	if result.Extra {
		return VerdictExtra
	}
	if result.FilePath == "" {
		if result.Reason == "placement is a placeholder" || result.Reason == "scenario has no scannable files in a declared UI slot" {
			return VerdictUnverifiable
		}
		return VerdictMissing
	}
	switch result.Provenance {
	case ProvenanceAdoptedUnmodified:
		return VerdictMatches
	case ProvenanceAdoptedModified, ProvenanceUnknown:
		return VerdictDrifted
	case ProvenanceCustom:
		return VerdictMatches
	default:
		return VerdictUnverifiable
	}
}
