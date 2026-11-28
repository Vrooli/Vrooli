package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (idx *Indexer) readVersionDependencyLock(versionPath, libraryID, version string) ([]AssetDependency, *ComponentVersionFile, error) {
	lockPath := filepath.ToSlash(filepath.Join(versionPath, "dependencies.json"))
	raw, err := fs.ReadFile(idx.fs, lockPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read dependency lock %s: %w", lockPath, err)
	}
	var lock VersionLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return nil, nil, ErrInvalidHeader{SourcePath: lockPath, Field: "dependencies.json", Reason: err.Error()}
	}
	if (lock.SchemaVersion != 1 && lock.SchemaVersion != 2) || strings.TrimSpace(lock.LibraryID) != libraryID || strings.TrimSpace(lock.Version) != version || strings.TrimSpace(lock.ResolvedAt) == "" {
		return nil, nil, ErrInvalidHeader{SourcePath: lockPath, Field: "identity", Reason: "schemaVersion, libraryId, version and resolvedAt must match the owning version"}
	}
	seen := map[string]struct{}{}
	dependencies := make([]AssetDependency, 0, len(lock.Dependencies))
	for _, rawDependency := range lock.Dependencies {
		observed := rawDependency.Pin()
		major := rawDependency.Major
		if major == 0 && observed != "" {
			_, _ = fmt.Sscanf(observed, "%d.", &major)
		}
		dependency := AssetDependency{LibraryID: strings.TrimSpace(rawDependency.LibraryID), Version: observed}
		if dependency.LibraryID == "" || dependency.Version == "" || rawDependency.Rank < 1 || rawDependency.Rank > 6 {
			return nil, nil, ErrInvalidHeader{SourcePath: lockPath, Field: "dependencies", Reason: "each dependency requires libraryId, version and rank 1-6"}
		}
		key := dependency.LibraryID + "\x00" + dependency.Version
		if _, exists := seen[key]; exists {
			return nil, nil, ErrInvalidHeader{SourcePath: lockPath, Field: "dependencies", Reason: "duplicate dependency " + dependency.LibraryID + "@" + dependency.Version}
		}
		seen[key] = struct{}{}
		dependencies = append(dependencies, dependency)
	}
	file := &ComponentVersionFile{Path: "dependencies.json", Content: string(raw), ContentSHA256: digestBytes(raw)}
	return dependencies, file, nil
}

func (idx *Indexer) catalogPorts(catalogID string) ([]string, []string, bool) {
	repoRoot := filepath.Clean(filepath.Join(idx.root, "..", "..", ".."))
	paths, _ := filepath.Glob(filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
			Expects []struct {
				Capability string `json:"capability"`
			} `json:"expects"`
			Satisfies []string `json:"satisfies"`
		}
		if json.Unmarshal(raw, &doc) != nil || doc.Asset.ID != catalogID {
			continue
		}
		expects := make([]string, 0, len(doc.Expects))
		for _, expect := range doc.Expects {
			expects = append(expects, expect.Capability)
		}
		return expects, append([]string(nil), doc.Satisfies...), true
	}
	return nil, nil, false
}

func (idx *Indexer) readVersionStory(sourcePath, libraryID, version string, assetKind AssetKind, componentSource string) (*ComponentStory, []IndexFinding, []string) {
	// Foundations are source-only closure members. The story-contract schema
	// intentionally has no foundation kind, so legacy story-shaped companions
	// beside them must not be reported as malformed component contracts.
	if assetKind == AssetKindFoundation {
		return nil, nil, nil
	}
	raw, err := fs.ReadFile(idx.fs, sourcePath)
	if errors.Is(err, fs.ErrNotExist) {
		// The complete-catalog conformance audit owns missing-file failures;
		// indexing remains usable while an author is creating a new version.
		return nil, nil, nil
	}
	if err != nil {
		return nil, []IndexFinding{invalidStoryFinding(sourcePath, "/", "readable story.json", "", err.Error())}, nil
	}
	contract, diagnostics := ParseStoryContract(raw)
	if contract == nil || len(storyErrors(diagnostics)) > 0 {
		errors := storyErrors(diagnostics)
		findings := make([]IndexFinding, 0, len(errors))
		for _, diagnostic := range errors {
			findings = append(findings, invalidStoryFinding(sourcePath, diagnostic.Pointer, diagnostic.Rule, "", diagnostic.Detail))
		}
		return nil, findings, nil
	}
	if registry := idx.storyFrameRegistry(); registry != nil {
		diagnostics = append(diagnostics, ValidateStoryFrames(contract, registry)...)
		if len(storyErrors(diagnostics)) > 0 {
			errors := storyErrors(diagnostics)
			findings := make([]IndexFinding, 0, len(errors))
			for _, diagnostic := range errors {
				findings = append(findings, invalidStoryFinding(sourcePath, diagnostic.Pointer, diagnostic.Rule, "", diagnostic.Detail))
			}
			return nil, findings, nil
		}
	}
	if AssetKind(contract.Kind) != assetKind {
		return nil, []IndexFinding{invalidStoryFinding(sourcePath, "/kind", string(assetKind), string(contract.Kind), "story kind must match manifest asset kind")}, nil
	}
	storyPath := filepath.ToSlash(filepath.Join(filepath.Dir(sourcePath), "story.tsx"))
	storySource, storyErr := fs.ReadFile(idx.fs, storyPath)
	harnesses := make(map[string]struct{})
	for _, definition := range contract.Stories {
		if definition.Composition != nil && definition.Composition.Specimen != nil {
			harnesses[definition.Composition.Specimen.Export] = struct{}{}
		}
	}
	var findings []IndexFinding
	if len(harnesses) > 0 && errors.Is(storyErr, fs.ErrNotExist) {
		return nil, []IndexFinding{{Kind: IndexFindingStoryHarnessMissing, SourcePath: storyPath, Field: "/stories", Detail: "a story references a specimen but story.tsx is missing"}}, nil
	}
	if storyErr != nil && !errors.Is(storyErr, fs.ErrNotExist) {
		return nil, []IndexFinding{{Kind: IndexFindingStoryHarnessMissing, SourcePath: storyPath, Field: "/stories", Detail: storyErr.Error()}}, nil
	}
	if storyErr == nil {
		exports := harnessExports(string(storySource))
		if len(harnesses) == 0 {
			// The declarative contract is authoritative. A story.tsx file may
			// contain typed specimen exports for a future composition without
			// making the otherwise valid story contract unindexable. Keep the
			// hygiene finding, but retain the contract projection so component
			// tests can validate the asset's one story record.
			findings = append(findings, IndexFinding{Kind: IndexFindingStoryHarnessOrphan, SourcePath: storyPath, Field: "/stories", Detail: "story.tsx exists but no story references a specimen export"})
		} else {
			for harness := range harnesses {
				if _, found := exports[harness]; !found {
					return nil, []IndexFinding{{Kind: IndexFindingStoryHarnessExport, SourcePath: storyPath, Field: "/stories/composition/specimen/export", Actual: harness, Detail: "referenced specimen export was not found in story.tsx"}}, nil
				}
			}
		}
	}
	derivedArgs := StoryArgsSchema{Fields: DeriveStoryFields(componentSource)}
	args, _ := json.Marshal(derivedArgs)
	environment, _ := json.Marshal(contract.Environment)
	stories, _ := json.Marshal(contract.Stories)
	normalized, _ := json.Marshal(contract)
	return &ComponentStory{LibraryID: libraryID, Version: version, SchemaVersion: contract.SchemaVersion, Kind: contract.Kind, Title: contract.Title, ArgsJSON: string(args), EnvironmentJSON: string(environment), StoriesJSON: string(stories), ContractJSON: string(normalized), SourcePath: sourcePath}, findings, nil
}

func storyErrors(diagnostics []StoryDiagnostic) []StoryDiagnostic {
	return StoryContractErrors(diagnostics)
}

func (idx *Indexer) storyFrameRegistry() CatalogFrameRegistry {
	if strings.TrimSpace(idx.root) == "" {
		return nil
	}
	catalogDir := filepath.Clean(filepath.Join(idx.root, "..", "catalog"))
	if _, err := os.Stat(catalogDir); err != nil {
		return nil
	}
	registry, err := CatalogFrameRegistryFromDir(catalogDir)
	if err != nil {
		return nil
	}
	return registry
}

var (
	storyHarnessExportRE   = regexp.MustCompile(`(?m)^\s*export\s+(?:async\s+)?(?:function|const|let|var|class)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	storyHarnessReexportRE = regexp.MustCompile(`(?m)export\s*\{([^}]*)\}\s*from\s*["'][^"']+["']`)
)

func harnessExports(source string) map[string]struct{} {
	exports := make(map[string]struct{})
	for _, match := range storyHarnessExportRE.FindAllStringSubmatch(source, -1) {
		if len(match) > 1 {
			exports[match[1]] = struct{}{}
		}
	}
	for _, match := range storyHarnessReexportRE.FindAllStringSubmatch(source, -1) {
		for _, item := range strings.Split(match[1], ",") {
			parts := strings.Fields(strings.TrimSpace(item))
			if len(parts) == 0 {
				continue
			}
			name := parts[0]
			if len(parts) >= 3 && parts[1] == "as" {
				name = parts[2]
			}
			if validHarnessExport(name) {
				exports[name] = struct{}{}
			}
		}
	}
	return exports
}

func assetKindForManifestPath(path, declared string) (AssetKind, error) {
	clean := filepath.ToSlash(path)
	var inferred AssetKind
	switch {
	case strings.HasPrefix(clean, "components/"):
		inferred = AssetKindComponent
	case strings.HasPrefix(clean, "primitives/"):
		// Primitives are renderable React assets. They use the component
		// contract and wire shape while retaining their authored root so
		// source files and previews resolve to the real materialized file.
		inferred = AssetKindComponent
	case strings.HasPrefix(clean, "foundations/"):
		inferred = AssetKindFoundation
	case strings.HasPrefix(clean, "hooks/"):
		inferred = AssetKindHook
	case strings.HasPrefix(clean, "services/"):
		inferred = AssetKindComponent
	default:
		return "", ErrInvalidHeader{SourcePath: path, Field: "asset root", Reason: "manifest must be under components/, primitives/, foundations/, or hooks/"}
	}
	if declared == "" {
		return inferred, nil
	}
	kind := AssetKind(strings.ToLower(strings.TrimSpace(declared)))
	// Catalog manifests use the public vocabulary while the registry keeps a
	// smaller internal kind set. Normalize aliases before validating the
	// declared kind so runtime assets are indexed rather than silently omitted.
	if kind == "runtime-hook" {
		kind = AssetKindHook
	}
	if kind == "runtime-service" {
		kind = AssetKindComponent
	}
	// "primitive" is the authored vocabulary for a renderable asset in the
	// primitives root; the registry intentionally normalizes it to the stable
	// component kind used by the API and dependency resolver.
	if kind == "primitive" && inferred == AssetKindComponent && strings.HasPrefix(clean, "primitives/") {
		return inferred, nil
	}
	if kind == "service" && inferred == AssetKindComponent && strings.HasPrefix(clean, "services/") {
		return inferred, nil
	}
	// GestureTokens was historically authored below components/ even though it
	// is a foundation. Keep that materialized release addressable while the
	// library is migrated to the canonical foundations/ root; dependency
	// closures must be able to resolve the released foundation during the
	// migration.
	if kind == AssetKindFoundation && inferred == AssetKindComponent && strings.HasPrefix(clean, "components/") {
		return kind, nil
	}
	if !kind.Valid() || kind != inferred {
		return "", ErrInvalidHeader{SourcePath: path, Field: "assetKind", Reason: "must match authored asset root " + string(inferred)}
	}
	return kind, nil
}

func validEntryExtension(name string, kind AssetKind) bool {
	if kind == AssetKindHook {
		return strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".tsx")
	}
	if kind == AssetKindFoundation {
		return strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx")
	}
	return strings.HasSuffix(name, ".tsx")
}

func isTestArtifact(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return strings.HasSuffix(base, ".test.tsx") || strings.HasSuffix(base, ".spec.tsx") ||
		strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".spec.ts")
}

func entryFileRequirement(kind AssetKind) string {
	if kind == AssetKindHook {
		return "TS file in the version folder"
	}
	if kind == AssetKindFoundation {
		return "TS or TSX file in the version folder"
	}
	return "TSX file in the version folder"
}

func (idx *Indexer) readParityReport(sourcePath string) (*IngestParityReport, error) {
	raw, err := fs.ReadFile(idx.fs, sourcePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", sourcePath, err)
	}
	var report IngestParityReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, ErrInvalidHeader{SourcePath: sourcePath, Field: "parity", Reason: err.Error()}
	}
	return &report, nil
}

func invalidStoryFinding(sourcePath, field, expected, actual, detail string) IndexFinding {
	return IndexFinding{Kind: IndexFindingInvalidStory, SourcePath: sourcePath, Field: field, Expected: expected, Actual: actual, Detail: detail}
}

func headerDisagreementFinding(sourcePath, field, expected, actual, detail string) IndexFinding {
	return IndexFinding{
		Kind:       IndexFindingHeaderDisagreement,
		SourcePath: sourcePath,
		Field:      field,
		Expected:   expected,
		Actual:     actual,
		Detail:     detail,
	}
}

func validateDepsHeaderJSON(raw string) error {
	if !strings.HasPrefix(raw, "{") && !strings.HasPrefix(raw, "[") {
		return fmt.Errorf("@deps must be a JSON object or array")
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return fmt.Errorf("invalid @deps JSON: %w", err)
	}
	return nil
}
