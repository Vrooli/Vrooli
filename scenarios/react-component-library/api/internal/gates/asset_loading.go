// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"react-component-library/internal/librarywalk"
	"sort"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/utilityclass"
)

func validateReleasedVersionHashLedger(root string) (Result, error) {
	ledgerPath := filepath.Join(root, "scenarios", "react-component-library", "library", "released-version-hashes.json")
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		return Result{RunnerError: []Finding{{
			Code: "catalog.released_version_immutable", AssetID: "__corpus__.released-version-hashes", File: repoRel(root, ledgerPath),
			Message:     "component index is unavailable or empty and the released-version hash ledger cannot be read",
			Remediation: "Restore library/released-version-hashes.json; an immutability gate must never pass without a byte oracle.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning",
		}}}, nil
	}
	var ledger releasedVersionHashLedger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return Result{}, fmt.Errorf("decode released-version hash ledger: %w", err)
	}
	result := Result{Inspected: len(ledger.Entries)}
	if len(ledger.Entries) == 0 {
		result.RunnerError = append(result.RunnerError, Finding{
			Code: "catalog.released_version_immutable", AssetID: "__corpus__.released-version-hashes", File: repoRel(root, ledgerPath),
			Message: "released-version hash ledger contains zero entries", Remediation: "Regenerate the ledger from every released version file; zero inspected inputs is not immutability evidence.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning",
		})
		return result, nil
	}
	for _, entry := range ledger.Entries {
		// dependencies.json is generated from authored source and may be
		// regenerated when a major-line dependency resolves to a new release.
		// Released immutability protects authored source, not this derived lock.
		if !components.IsAuthoredReleaseFile(entry.Path) {
			result.Inspected--
			continue
		}
		path := filepath.Join(root, "scenarios", "react-component-library", "library", filepath.FromSlash(entry.Path))
		raw, err := os.ReadFile(path)
		if err != nil {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.released_version_immutable", AssetID: implementationName(path), File: filepath.ToSlash(filepath.Join("library", entry.Path)),
				Message: fmt.Sprintf("released file cannot be read: %v", err), Remediation: "Restore the released file or retire the version through the governed lifecycle.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning",
			})
			continue
		}
		sum := sha256.Sum256(raw)
		current := hex.EncodeToString(sum[:])
		if current == entry.SHA256 {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.released_version_immutable", AssetID: implementationName(path), File: filepath.ToSlash(filepath.Join("library", entry.Path)),
			Message:     fmt.Sprintf("released file changed after ledger capture: recorded %s, current %s", shortHash(entry.SHA256), shortHash(current)),
			Remediation: "Revert the released file and publish the change as a new version; never update the ledger to bless an in-place mutation.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning",
		})
	}
	return nonEmpty(result, "released-version-immutable"), nil
}

func shortHash(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

type assetDoc struct {
	Asset struct {
		ID, Kind, Name, Surface, Description string
		Target                               struct {
			Maturity string `json:"maturity"`
		} `json:"target"`
	} `json:"asset"`
	Budgets struct {
		MountMS float64 `json:"mountMs"`
	} `json:"budgets"`
	API *struct {
		Variants map[string][]string `json:"variants"`
		Modes    []string            `json:"modes"`
		Parts    []json.RawMessage   `json:"parts"`
	} `json:"api"`
	Fixture *struct {
		DataShapes []string `json:"dataShapes"`
		Satisfies  *struct {
			Capability    string   `json:"capability"`
			TypeArguments []string `json:"typeArguments"`
		} `json:"satisfies"`
	} `json:"fixture"`
	Provides map[string]json.RawMessage `json:"provides"`
	Consumes map[string]json.RawMessage `json:"consumes"`
}

func loadAssets(scope Scope) ([]assetDoc, error) {
	root := scope.Root
	selected := make(map[string]bool, len(scope.Assets))
	for _, assetID := range scope.Assets {
		selected[assetID] = true
	}
	paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []assetDoc
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc assetDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if !scope.IsFullCorpus() && !selected[doc.Asset.ID] {
			continue
		}
		out = append(out, doc)
	}
	return out, nil
}

// loadLibraryAssets is the manifest-backed view used by source gates. The
// catalog projection can legitimately lag a newly authored manifest; source
// quality gates must not turn that lag into an uninspected implementation.
func loadLibraryAssets(scope Scope) ([]assetDoc, error) {
	root := scope.Root
	catalog, err := loadAssets(scope)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]assetDoc, len(catalog))
	for _, asset := range catalog {
		byID[asset.Asset.ID] = asset
	}
	var result []assetDoc
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		manifests, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		sort.Strings(manifests)
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return nil, err
			}
			var metadata struct {
				CatalogID    string `json:"catalogId"`
				LibraryID    string `json:"libraryId"`
				DisplayName  string `json:"displayName"`
				AssetKind    string `json:"assetKind"`
				Supplemental bool   `json:"supplemental"`
			}
			if err := json.Unmarshal(data, &metadata); err != nil {
				return nil, fmt.Errorf("parse %s: %w", manifest, err)
			}
			id := metadata.CatalogID
			if id == "" && metadata.Supplemental {
				id = "supplemental." + strings.ReplaceAll(metadata.LibraryID, ":", ".")
			}
			if projected, ok := byID[id]; ok {
				result = append(result, projected)
				continue
			}
			assetKind := metadata.AssetKind
			result = append(result, assetDoc{Asset: struct {
				ID, Kind, Name, Surface, Description string
				Target                               struct {
					Maturity string `json:"maturity"`
				} `json:"target"`
			}{ID: id, Kind: assetKind, Name: metadata.DisplayName}})
		}
	}
	return result, nil
}

// ValidateAPI checks declared API vocabulary against the implementation
// source selected by catalogId. Missing implementations are not failures of
// this runner; coverage keeps those assets at missing/scaffolded.
func validateNoUtilityClasses(scope Scope, gate string) (Result, error) {
	root := scope.Root
	allowlistPath := filepath.Join(root, "scenarios", "react-component-library", "library", "utility-class-allowlist.json")
	data, err := os.ReadFile(allowlistPath)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	var allowlist utilityClassAllowance
	if len(data) > 0 {
		if err := json.Unmarshal(data, &allowlist); err != nil {
			return Result{}, fmt.Errorf("decode utility-class allowlist: %w", err)
		}
	}
	allowed := make(map[string]bool, len(allowlist.Entries))
	for _, entry := range allowlist.Entries {
		allowed[filepath.ToSlash(filepath.Clean(entry.Path))] = true
	}
	sources, err := allLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	runtimeSources := sources[:0]
	for _, path := range sources {
		base := strings.ToLower(filepath.Base(path))
		if strings.HasPrefix(base, "story.") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
			continue
		}
		runtimeSources = append(runtimeSources, path)
	}
	result := Result{Inspected: len(runtimeSources)}
	observedAllowed := map[string]bool{}
	for _, path := range runtimeSources {
		source, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		rel := repoRel(root, path)
		versionPath := libraryVersionPath(rel)
		for _, hit := range utilityclass.EmitsAny(string(source)) {
			finding := Finding{
				Code:        "catalog." + gate,
				AssetID:     implementationName(path),
				File:        rel,
				Line:        lineOf(source, hit.Class),
				Category:    hit.Category,
				Message:     fmt.Sprintf("emits utility class %q (%s)", hit.Class, hit.Category),
				Remediation: "Replace library-owned utility classes with a module stylesheet and semantic custom properties; consumer-supplied className values may pass through unchanged.",
				DocsRef:     "docs/reference/style-ownership.md",
			}
			if allowed[versionPath] {
				if !observedAllowed[versionPath] {
					observedAllowed[versionPath] = true
					finding.Message = fmt.Sprintf("allow-listed utility-class debt remains in %s", versionPath)
					result.InformationalFindings = append(result.InformationalFindings, finding)
				}
				continue
			}
			result.Findings = append(result.Findings, finding)
		}
	}
	for path := range allowed {
		if observedAllowed[path] {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog." + gate + ".stale-allowance",
			AssetID:     "__corpus__.utility-class-allowlist",
			File:        repoRel(root, allowlistPath),
			Message:     fmt.Sprintf("allowlist entry %q no longer covers an emitted utility class", path),
			Remediation: "Remove the stale entry and lower utility-class-allowlist.max.",
			DocsRef:     "docs/reference/style-ownership.md",
		})
	}
	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].File == result.Findings[j].File {
			return result.Findings[i].Line < result.Findings[j].Line
		}
		return result.Findings[i].File < result.Findings[j].File
	})
	return nonEmpty(result, gate), nil
}

func libraryVersionPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for index := 0; index+4 < len(parts); index++ {
		if parts[index] == "library" && parts[index+3] == "versions" {
			return strings.Join(parts[index:index+5], "/")
		}
	}
	return ""
}

// ValidateDeprecatedImports checks the active catalog source against its own
// manifest's deprecatedVersions list. Released historical versions remain
// immutable and may still contain the dependency pins that were valid when
// they were published; active sources are the surface this gate governs.
func consumerPinManifests(root string) (map[string]consumerPinManifest, error) {
	result := map[string]consumerPinManifest{}
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, err := librarywalk.Glob(filepath.Join(libraryRoot, kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			var manifest consumerPinManifest
			if err := json.Unmarshal(raw, &manifest); err != nil {
				return nil, err
			}
			manifest.Root = filepath.Dir(path)
			result[filepath.Base(filepath.Dir(path))] = manifest
		}
	}
	return result, nil
}
