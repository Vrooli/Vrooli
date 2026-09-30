package gates

import (
	"os"
)

func ValidateSurfaceDiscipline(scope Scope) (Result, error) {
	root := scope.Root
	assets, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	captures, err := loadSurfaceCaptures(root)
	if err != nil {
		return Result{}, err
	}
	tokens, err := loadElevationTokens(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{SurfaceCounts: map[string]int{}}
	for _, asset := range assets {
		manifest, source, found, sourceErr := implementationSource(root, asset.Asset.ID)
		if sourceErr != nil {
			return Result{}, sourceErr
		}
		if !found {
			continue
		}
		sourceData, err := os.ReadFile(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		result.InspectedAssets = appendUnique(result.InspectedAssets, asset.Asset.ID)
		staticApplied := sourceUsesSurfaceRamp(string(sourceData), asset.Asset.Surface)
		captured, ok := captures[asset.Asset.ID]
		if !ok {
			result.UnmeasuredAssets = append(result.UnmeasuredAssets, asset.Asset.ID)
		}
		verdict := classifySurface(staticApplied, ok, ok && renderedSurfaceMatches(captured, asset.Asset.Surface, tokens))
		result.SurfaceCounts[string(verdict)]++
		switch verdict {
		case SurfaceRenderedWrong:
			result.Findings = append(result.Findings, Finding{Code: "catalog.surface_wrong_elevation", AssetID: asset.Asset.ID, File: repoRel(root, manifest), Message: "renders the wrong elevation", Remediation: "Apply the declared SURFACE_ELEVATIONS value to the stamped asset root so its computed box-shadow matches the declared surface ramp."})
		case SurfaceHardCoded:
			result.Findings = append(result.Findings, Finding{Code: "catalog.surface_hard_coded", AssetID: asset.Asset.ID, File: repoRel(root, manifest), Message: "correct pixels, hard-coded path", Remediation: "Use SURFACE_ELEVATIONS in the rendered JSX instead of reproducing the ramp value through a raw class or style."})
		case SurfaceBothMismatch:
			result.Findings = append(result.Findings, Finding{Code: "catalog.surface_unreconciled", AssetID: asset.Asset.ID, File: repoRel(root, manifest), Message: "static ramp usage and rendered elevation both disagree with the declared surface", Remediation: "Declare the intended surface and make the rendered root consume the matching SURFACE_ELEVATIONS value."})
		}
	}
	if result.Inspected == 0 || len(captures) == 0 {
		result.Status = "unmeasured"
	}
	stampReport, stampErr := LoadStampReport(root)
	if stampErr != nil {
		return Result{}, stampErr
	}
	result.UnstampedAssets, result.UncapturedAssets = classifyUnmeasured(stampReport, result.UnmeasuredAssets)
	result.SurfaceCounts["unstamped"] = len(result.UnstampedAssets)
	result.SurfaceCounts["uncaptured"] = len(result.UncapturedAssets)
	return nonEmpty(result, "surface-discipline"), nil
}
