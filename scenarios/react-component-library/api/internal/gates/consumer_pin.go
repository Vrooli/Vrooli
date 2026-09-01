package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/librarywalk"
	"react-component-library/internal/utilityclass"
)

func ValidateConsumerPins(scope Scope) (Result, error) {
	root := scope.Root
	manifests, err := consumerPinManifests(root)
	if err != nil {
		return Result{}, err
	}
	pins, err := scenarioConsumerPins(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(pins)}
	for _, pin := range pins {
		manifest, exists := manifests[pin.Asset]
		assetID := "__corpus__.consumer-pins"
		if exists {
			assetID = manifest.CatalogID
			if assetID == "" {
				assetID = manifest.LibraryID
			}
		}
		scenarios := sortedStringMapKeys(pin.Scenarios)
		location := ""
		if len(pin.Files) > 0 {
			location = pin.Files[0]
		}
		add := func(suffix, message, remediation string) {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.consumer-pin." + suffix, AssetID: assetID, File: location,
				Message:     fmt.Sprintf("%s@%s consumed by %s: %s", pin.Asset, pin.Version, strings.Join(scenarios, ", "), message),
				Remediation: remediation, DocsRef: "docs/reference/style-ownership.md",
			})
		}
		if !exists {
			add("missing", "asset manifest does not exist", "Restore the published asset or migrate every named consumer to an existing asset.")
			continue
		}
		resolved, versionRoot := resolveConsumerPinVersion(manifest, pin.Version)
		if resolved == "" {
			add("missing", "published version does not exist", "Migrate every named consumer to a version present in the asset manifest and version tree.")
			continue
		}
		if contains(manifest.DeprecatedVersions, resolved) {
			add("deprecated", "version is deprecated", "Migrate every named consumer to a non-deprecated version in the same supported major.")
		}
		if versionMajor(resolved) < versionMajor(manifest.Latest) {
			add("stale-major", fmt.Sprintf("latest is %s", manifest.Latest), "Migrate every named consumer to the current supported major, then use a major-scoped import.")
		}
		emits := false
		if err := librarywalk.Walk(versionRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || emits || entry.IsDir() {
				return walkErr
			}
			base := strings.ToLower(filepath.Base(path))
			ext := strings.ToLower(filepath.Ext(path))
			if (ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx") || strings.HasPrefix(base, "story.") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			emits = len(utilityclass.EmitsAny(string(raw))) > 0
			return nil
		}); err != nil {
			return Result{}, err
		}
		if emits {
			add("utility-class", "version emits library-owned utility classes", "Publish a token-bound version and migrate every named consumer; a Tailwind content glob is only a temporary bridge.")
		}
	}
	return nonEmpty(result, "consumer-pin"), nil
}
