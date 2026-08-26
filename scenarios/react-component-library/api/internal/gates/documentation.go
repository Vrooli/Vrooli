package gates

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var exportedTypeScriptSymbol = regexp.MustCompile(`^\s*export\s+(?:default\s+)?(?:async\s+)?(?:function|const|let|var|class|interface|type|enum)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)

// ValidateDocumentation is intentionally an absence check. It does not score
// prose quality; it proves the cheaper, falsifiable contract that every
// exported symbol has a TSDoc block immediately above it. A missing block is
// reported, never converted into a pass by a catalog description.
func ValidateDocumentation(root string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		kind := asset.Asset.Kind
		if kind != "foundation" && kind != "runtime-hook" && kind != "runtime-service" && kind != "adapter" && kind != "primitive" && kind != "component" && kind != "pattern" && kind != "page-template" && kind != "navigation" && kind != "generator" {
			continue
		}
		_, source, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if !ok {
			continue
		}
		if isStorySource(source) {
			// story.tsx exports are preview specimens, not public library API.
			// Their behavior is covered by the story contract and component test,
			// so requiring TSDoc on every specimen helper creates noise without
			// improving the published component contract.
			continue
		}
		data, err := os.Open(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		scanner := bufio.NewScanner(data)
		lineNumber := 0
		commented := false
		for scanner.Scan() {
			lineNumber++
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "/**") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "//") {
				if strings.HasPrefix(line, "/**") {
					commented = true
				}
				continue
			}
			match := exportedTypeScriptSymbol.FindStringSubmatch(line)
			if len(match) == 0 {
				if line != "" {
					commented = false
				}
				continue
			}
			if !commented {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.documentation", AssetID: asset.Asset.ID, File: repoRel(root, source), Line: lineNumber,
					Message:     "exported symbol " + match[1] + " has no TSDoc block",
					Remediation: "Add a /** ... */ TSDoc block immediately before the exported symbol describing its purpose, public inputs, behavior, accessibility contract, and important performance constraints.",
					DocsRef:     "docs/internal/TESTING.md",
				})
			}
			commented = false
		}
		if err := scanner.Err(); err != nil {
			_ = data.Close()
			return Result{}, err
		}
		if err := data.Close(); err != nil {
			return Result{}, err
		}
	}
	return nonEmpty(result, "documentation"), nil
}
