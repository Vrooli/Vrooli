package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"react-component-library/internal/themes"
)

func ValidateTokens(scope Scope) (Result, error) {
	root := scope.Root
	paths, err := filepath.Glob(filepath.Join(root, "templates", "design", "*", "adapters", "react-vite-tailwind", "tokens.css"))
	if err != nil {
		return Result{}, err
	}
	shared := []string{"space-3xs", "space-2xs", "space-xs", "space-sm", "space-md", "space-lg", "space-xl", "space-2xl", "text-display", "text-title", "text-heading", "text-body", "elev-flat", "elev-raised", "layer-base", "layer-modal", "dur-instant"}
	result := Result{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		kit := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
		resolved, err := themes.ComposeKitCSS(root, kit)
		if err != nil {
			return Result{}, fmt.Errorf("resolve design kit %q: %w", kit, err)
		}
		for _, token := range shared {
			if !strings.Contains(resolved, "--"+token) {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.tokens_missing", AssetID: "", File: repoRel(root, path),
					Message:     fmt.Sprintf("design kit %q does not resolve shared token --%s", kit, token),
					Remediation: fmt.Sprintf("Declare --%s in the shared base or this kit's override file. Every kit must resolve the full shared vocabulary, because a component authored against one kit is expected to render in all of them; a missing step silently drops that declaration to its initial value wherever the component lands.", token),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
				})
			}
		}
		for _, loc := range pxValue.FindAllSubmatchIndex(data, -1) {
			raw := string(data[loc[2]:loc[3]])
			value, _ := strconv.ParseFloat(raw, 64)
			if int(value)%4 != 0 {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.tokens_grid", AssetID: "", File: repoRel(root, path), Line: lineAt(data, loc[0]),
					Message:     fmt.Sprintf("design kit %q declares a spacing token at %spx, off the 4px grid", kit, raw),
					Remediation: fmt.Sprintf("Round %spx to the nearest multiple of 4. The grid is what makes steps from different kits interchangeable; an off-grid step produces half-pixel seams where a tokenized element sits next to an off-grid one, which reads as the blurry-edge misalignment that is very hard to attribute back to a token definition.", raw),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
				})
			}
		}
	}
	sources, err := activeLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		result.Findings = append(result.Findings, literalDimensionFindings(root, path, data)...)
	}
	for index := range result.Findings {
		result.Findings[index].Category = "conformance"
	}
	return nonEmpty(result, "tokens"), nil
}

type tokenFallback struct {
	Property string
	Value    string
	Offset   int
}

// ValidateFallbackParity makes authored fallbacks an honest view of the
// canonical base vocabulary. Properties owned inside the same version and
// host-computed --rcl-* contracts are intentionally outside this comparison.
