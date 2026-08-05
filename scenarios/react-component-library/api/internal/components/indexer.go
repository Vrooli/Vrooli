package components

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Indexer walks a configured filesystem root for asset manifests under
// components/*/component.json and hooks/*/component.json and upserts the
// manifest plus its version folders into the Repository. A final DeleteMissing call
// removes rows whose manifests no longer exist, so deleted components
// leave the registry without manual intervention.
//
// The header contract (req CR-002, captured in PRD.md):
//
//	/**
//	 * @libraryId   react-component-library:Button
//	 * @displayName Button
//	 * @description Primary call-to-action button.
//	 * @version     1.0.0
//	 * @tags        ["form", "interactive"]
//	 * @deps        {"react": "^18"}
//	 * @warning     DO NOT REMOVE THIS HEADER...
//	 */
//
// component.json is the source of truth. Source-file headers are
// validation hints for version, status, deps, and readability.
// UpsertObserver is the post-upsert seam other domains hook into. The
// indexer calls Observe after a successful repo.UpsertManifest with the
// parsed manifest input, so cross-domain consumers (currently req 10's deps
// recorder) can re-sync without parsing files themselves. nil observer means
// "no hook"; production wires deps.Service via a small adapter in main.go.
type UpsertObserver interface {
	Observe(ctx context.Context, c Component, in IndexManifestInput) error
}

type Indexer struct {
	repo     Repository
	root     string
	fs       fs.FS // injected for tests; nil means use os.DirFS(root)
	observer UpsertObserver
}

// SetUpsertObserver installs the post-upsert seam. Designed to be
// called once at boot before any Run call; not concurrency-safe with
// in-flight Runs.
func (idx *Indexer) SetUpsertObserver(o UpsertObserver) { idx.observer = o }

// NewIndexer constructs an Indexer rooted at root. The root is the
// absolute path on disk; consumers resolve it via api-core/storage
// before calling. fsys may be nil — production passes nil and the
// indexer wraps os.DirFS(root); tests pass an in-memory fs.FS so they
// don't touch disk.
func NewIndexer(repo Repository, root string, fsys fs.FS) *Indexer {
	if fsys == nil && root != "" {
		fsys = os.DirFS(root)
	}
	return &Indexer{repo: repo, root: root, fs: fsys}
}

// IndexResult summarizes one Run.
type IndexResult struct {
	Scanned    int
	Indexed    int
	Skipped    int
	Deleted    int
	Errors     []error
	Findings   []IndexFinding
	LibraryIDs []string // upserted IDs in walk order — useful for tests
}

// Run walks the root, upserts every manifest with valid version folders,
// and returns a result. Malformed manifests are recorded in Errors but
// do not stop the walk — a single broken component should not hide an
// otherwise healthy run.
func (idx *Indexer) Run(ctx context.Context) (IndexResult, error) {
	var result IndexResult
	if idx.fs == nil {
		return result, fmt.Errorf("indexer has no filesystem configured")
	}

	walkErr := fs.WalkDir(idx.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) != "component.json" {
			return nil
		}
		result.Scanned++
		in, _, perr := idx.buildManifestInput(path)
		if perr != nil {
			result.Errors = append(result.Errors, perr)
			return nil
		}
		result.Findings = append(result.Findings, in.Findings...)
		comp, err := idx.repo.UpsertManifest(ctx, in)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("upsert %s: %w", path, err))
			return nil
		}
		if idx.observer != nil {
			if oerr := idx.observer.Observe(ctx, comp, in); oerr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("observe %s: %w", path, oerr))
				// continue — observer failure must not hide the upsert.
			}
		}
		result.Indexed++
		result.LibraryIDs = append(result.LibraryIDs, in.Manifest.LibraryID)
		return nil
	})
	if walkErr != nil {
		return result, fmt.Errorf("walk %s: %w", idx.root, walkErr)
	}

	// Sweep registry-orphaned rows before DeleteMissing runs. At this
	// point every current component has just been re-upserted and stale
	// components still hold their registry row, so the only orphans here
	// are genuine cruft (a prior re-slug or withdrawal that deleted the
	// registry row without clearing its children). Each swept version
	// row surfaces a conformance finding so the cleanup is never silent.
	orphans, oerr := idx.repo.SweepOrphans(ctx)
	if oerr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("sweep orphans: %w", oerr))
	}
	for _, o := range orphans {
		result.Findings = append(result.Findings, registryOrphanFinding(o))
	}

	// DeleteMissing removes stale registry rows (manifests gone from
	// disk) and cascades their children — expected churn, so it emits no
	// finding.
	deleted, derr := idx.repo.DeleteMissing(ctx, result.LibraryIDs)
	if derr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("delete missing: %w", derr))
	}
	result.Deleted = deleted

	return result, nil
}

func registryOrphanFinding(o OrphanVersion) IndexFinding {
	return IndexFinding{
		Kind:       IndexFindingRegistryOrphan,
		SourcePath: o.SourcePath,
		Field:      "component_id",
		Expected:   "component_versions row with an owning components registry row",
		Actual:     o.ComponentID,
		Detail: fmt.Sprintf("removed registry-orphaned version %s@%s (component_id %s has no registry parent)",
			o.LibraryID, o.Version, o.ComponentID),
	}
}

func missingDesignAffinityFinding(sourcePath, libraryID string) IndexFinding {
	return IndexFinding{
		Kind:       IndexFindingMissingDesignAffinity,
		SourcePath: sourcePath,
		Field:      "designStyles",
		Expected:   "at least one declared design-style affinity",
		Actual:     "none",
		Detail: fmt.Sprintf("component %q declares no design-style affinities; authored catalog components carry 2-3",
			libraryID),
	}
}

type manifestFile struct {
	CatalogID   string `json:"catalogId"`
	LibraryID   string `json:"libraryId"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Slot        string `json:"slot"`
	// Category is catalog metadata, not source-header metadata. Keeping it in
	// the manifest makes it stable across versions and available to list views.
	Category string `json:"category"`
	// AssetKind is optional for backwards compatibility: the authored root is
	// authoritative for legacy manifests. New manifests may state it explicitly
	// as a guard against moving a manifest into the wrong root.
	AssetKind    string            `json:"assetKind"`
	Dependencies []AssetDependency `json:"dependencies"`
	Entry        string            `json:"entry"`
	// FileSlots pins an explicit placement slot per version-unit basename
	// (e.g. {"useFocusTrap.ts": "hook"}). Authoritative over the resolver's
	// extension heuristic. Applies across all versions of the component.
	FileSlots          map[string]string        `json:"fileSlots"`
	Tags               []string                 `json:"tags"`
	DesignStyles       []manifestDesignAffinity `json:"designStyles"`
	Latest             string                   `json:"latest"`
	Draft              string                   `json:"draft"`
	DeprecatedVersions []string                 `json:"deprecatedVersions"`
}

type manifestDesignAffinity struct {
	StyleID  string `json:"styleId"`
	Affinity string `json:"affinity"`
	Reason   string `json:"reason"`
}

func (idx *Indexer) buildManifestInput(path string) (IndexManifestInput, map[string]string, error) {
	raw, err := fs.ReadFile(idx.fs, path)
	if err != nil {
		return IndexManifestInput{}, nil, fmt.Errorf("read %s: %w", path, err)
	}
	var mf manifestFile
	if err := json.Unmarshal(raw, &mf); err != nil {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "component.json", Reason: err.Error()}
	}
	slug := filepath.Base(filepath.Dir(path))
	assetKind, err := assetKindForManifestPath(path, mf.AssetKind)
	if err != nil {
		return IndexManifestInput{}, nil, err
	}
	slot, err := normalizeComponentSlot(mf.Slot)
	if err != nil && assetKind == AssetKindComponent {
		return IndexManifestInput{}, nil, errInvalidHeader(path, "slot", err.Error())
	}
	if assetKind == AssetKindHook && strings.TrimSpace(mf.Slot) != "" {
		return IndexManifestInput{}, nil, errInvalidHeader(path, "slot", "hooks are non-renderable and must not declare a UI slot")
	}
	deps, err := normalizeAssetDependencies(path, mf.Dependencies)
	if err != nil {
		return IndexManifestInput{}, nil, err
	}
	manifest := ComponentManifest{
		CatalogID:          strings.TrimSpace(mf.CatalogID),
		LibraryID:          strings.TrimSpace(mf.LibraryID),
		Slug:               slug,
		DisplayName:        strings.TrimSpace(mf.DisplayName),
		Description:        strings.TrimSpace(mf.Description),
		Slot:               string(slot),
		Category:           strings.TrimSpace(mf.Category),
		ManifestPath:       filepath.ToSlash(path),
		LatestVersion:      strings.TrimSpace(mf.Latest),
		DraftVersion:       strings.TrimSpace(mf.Draft),
		DeprecatedVersions: append([]string(nil), mf.DeprecatedVersions...),
		Tags:               append([]string(nil), mf.Tags...),
		AssetKind:          assetKind,
		Dependencies:       deps,
	}
	designStyles, err := parseManifestDesignStyles(path, mf.DesignStyles)
	if err != nil {
		return IndexManifestInput{}, nil, err
	}
	staleFindings, err := staleDesignStyleFindings(path, designStyles)
	if err != nil {
		return IndexManifestInput{}, nil, err
	}
	manifest.DesignStyles = designStyles
	if manifest.LibraryID == "" {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "libraryId", Reason: "required"}
	}
	if manifest.DisplayName == "" {
		manifest.DisplayName = slug
	}
	if manifest.LatestVersion == "" {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "required"}
	}
	versionRoot := filepath.ToSlash(filepath.Join(filepath.Dir(path), "versions"))
	entries, err := fs.ReadDir(idx.fs, versionRoot)
	if err != nil {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "versions", Reason: err.Error()}
	}
	deprecated := map[string]bool{}
	for _, v := range manifest.DeprecatedVersions {
		deprecated[strings.TrimSpace(v)] = true
	}
	var versions []ComponentVersion
	var stories []ComponentStory
	findings := append([]IndexFinding(nil), staleFindings...)
	// A promoted component with no declared affinities is catalog-incomplete:
	// its detail view reads "No design affinities declared" while authored
	// peers carry 2-3. Surface it as a soft conformance finding (never a hard
	// error) so the gap is visible on every reindex without blocking the walk.
	if len(manifest.DesignStyles) == 0 {
		findings = append(findings, missingDesignAffinityFinding(path, manifest.LibraryID))
	}
	var latestFound bool
	var draftFound bool
	latestHeaders := map[string]string{"libraryId": manifest.LibraryID}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := strings.TrimSpace(entry.Name())
		versionPath := filepath.ToSlash(filepath.Join(versionRoot, version))
		files, err := fs.ReadDir(idx.fs, versionPath)
		if err != nil {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "version", Reason: err.Error()}
		}
		var sourceFiles []fs.DirEntry
		for _, f := range files {
			if f.IsDir() {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "version", Reason: "subdirectories are not supported"}
			}
			// story.tsx is a preview-only artifact. It is validated with the
			// story contract but is never an asset companion or adoption input.
			if f.Name() == "story.tsx" {
				continue
			}
			if strings.HasSuffix(f.Name(), ".tsx") || strings.HasSuffix(f.Name(), ".ts") {
				sourceFiles = append(sourceFiles, f)
			}
		}
		entryName := strings.TrimSpace(mf.Entry)
		if entryName != "" && (filepath.Base(entryName) != entryName || !validEntryExtension(entryName, assetKind)) {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "entry", Reason: entryFileRequirement(assetKind)}
		}
		if entryName == "" {
			var entries []string
			for _, f := range sourceFiles {
				if validEntryExtension(f.Name(), assetKind) {
					entries = append(entries, f.Name())
				}
			}
			if len(entries) != 1 {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "version", Reason: "expected exactly one " + entryFileRequirement(assetKind) + " or component.json entry"}
			}
			entryName = entries[0]
		}
		var entryFound bool
		var versionFiles []ComponentVersionFile
		for _, f := range sourceFiles {
			filePath := filepath.ToSlash(filepath.Join(versionPath, f.Name()))
			body, err := fs.ReadFile(idx.fs, filePath)
			if err != nil {
				return IndexManifestInput{}, nil, fmt.Errorf("read %s: %w", filePath, err)
			}
			isEntry := f.Name() == entryName
			if isEntry {
				entryFound = true
			}
			versionFiles = append(versionFiles, ComponentVersionFile{Path: f.Name(), Content: string(body), ContentSHA256: digestBytes(body), IsEntry: isEntry, Slot: strings.TrimSpace(mf.FileSlots[f.Name()])})
		}
		if !entryFound {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "entry", Reason: "entry file not found"}
		}
		experienceContract := ""
		contractPath := filepath.ToSlash(filepath.Join(versionPath, "experience-contract.json"))
		if rawContract, contractErr := fs.ReadFile(idx.fs, contractPath); contractErr == nil {
			experienceContract = string(rawContract)
			versionFiles = append(versionFiles, ComponentVersionFile{Path: "experience-contract.json", Content: experienceContract, ContentSHA256: digestBytes([]byte(experienceContract))})
		} else if !errors.Is(contractErr, fs.ErrNotExist) {
			return IndexManifestInput{}, nil, fmt.Errorf("read experience contract %s: %w", contractPath, contractErr)
		}
		sourcePath := filepath.ToSlash(filepath.Join(versionPath, entryName))
		src, err := fs.ReadFile(idx.fs, sourcePath)
		if err != nil {
			return IndexManifestInput{}, nil, fmt.Errorf("read %s: %w", sourcePath, err)
		}
		headers := map[string]string{}
		if header, ok := extractHeaderBlock(string(src)); ok {
			headers, err = parseHeader(sourcePath, header)
			if err != nil {
				return IndexManifestInput{}, nil, err
			}
			if hv := strings.TrimSpace(headers["version"]); hv != "" && hv != version {
				findings = append(findings, headerDisagreementFinding(
					sourcePath,
					"version",
					version,
					hv,
					"source header @version does not match the version folder",
				))
			}
			if hid := strings.TrimSpace(headers["libraryId"]); hid != "" && hid != manifest.LibraryID {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: sourcePath, Field: "libraryId", Reason: "does not match manifest"}
			}
			if rawDeps := strings.TrimSpace(headers["deps"]); rawDeps != "" {
				if err := validateDepsHeaderJSON(rawDeps); err != nil {
					findings = append(findings, headerDisagreementFinding(
						sourcePath,
						"deps",
						"JSON object or array",
						rawDeps,
						err.Error(),
					))
				}
			}
		}
		status := VersionStatusReleased
		if strings.Contains(version, "-") {
			status = VersionStatusDraft
		}
		if deprecated[version] {
			status = VersionStatusDeprecated
		}
		if rawStatus := strings.TrimSpace(headers["status"]); rawStatus != "" {
			headerStatus := ComponentVersionStatus(strings.ToLower(rawStatus))
			if !isValidVersionStatus(headerStatus) || headerStatus != status {
				findings = append(findings, headerDisagreementFinding(
					sourcePath,
					"status",
					string(status),
					rawStatus,
					"source header @status does not match manifest-derived version status",
				))
			}
		}
		if version == manifest.LatestVersion {
			latestFound = true
			if status != VersionStatusReleased {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "must point to a released non-deprecated version"}
			}
			for k, v := range headers {
				latestHeaders[k] = v
			}
		}
		if version == manifest.DraftVersion {
			draftFound = true
			if status != VersionStatusDraft {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "draft", Reason: "must point to a draft/pre-release version"}
			}
		}
		parity, err := idx.readParityReport(filepath.ToSlash(filepath.Join(versionPath, "parity.json")))
		if err != nil {
			return IndexManifestInput{}, nil, err
		}
		versions = append(versions, ComponentVersion{
			LibraryID:          manifest.LibraryID,
			Version:            version,
			Status:             status,
			SourcePath:         sourcePath,
			Content:            string(src),
			ContentSHA256:      digestBytes(src),
			Headers:            headers,
			Files:              versionFiles,
			ExperienceContract: experienceContract,
			ParityReport:       parity,
		})
		story, storyFindings := idx.readVersionStory(filepath.ToSlash(filepath.Join(versionPath, "story.json")), manifest.LibraryID, version, manifest.AssetKind)
		if story != nil {
			stories = append(stories, *story)
		}
		findings = append(findings, storyFindings...)
	}
	if !latestFound {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "version folder not found"}
	}
	if manifest.DraftVersion != "" && !draftFound {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "draft", Reason: "version folder not found"}
	}
	// Older catalog entries stored this in the entry header. Keep that as an
	// import-time fallback only; new manifests own the canonical value.
	if manifest.Category == "" {
		manifest.Category = strings.TrimSpace(latestHeaders["category"])
	}
	if manifest.CatalogID != "" {
		latestHeaders["catalogId"] = manifest.CatalogID
		if expects, satisfies, ok := idx.catalogPorts(manifest.CatalogID); ok {
			if raw, err := json.Marshal(expects); err == nil {
				latestHeaders["catalogExpects"] = string(raw)
			}
			if raw, err := json.Marshal(satisfies); err == nil {
				latestHeaders["catalogSatisfies"] = string(raw)
			}
			manifest.Expects = expects
			manifest.Satisfies = satisfies
		}
	}
	return IndexManifestInput{Manifest: manifest, Versions: versions, Stories: stories, Headers: latestHeaders, Findings: findings}, latestHeaders, nil
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

func (idx *Indexer) readVersionStory(sourcePath, libraryID, version string, assetKind AssetKind) (*ComponentStory, []IndexFinding) {
	raw, err := fs.ReadFile(idx.fs, sourcePath)
	if errors.Is(err, fs.ErrNotExist) {
		// The complete-catalog conformance audit owns missing-file failures;
		// indexing remains usable while an author is creating a new version.
		return nil, nil
	}
	if err != nil {
		return nil, []IndexFinding{invalidStoryFinding(sourcePath, "/", "readable story.json", "", err.Error())}
	}
	contract, diagnostics := ParseStoryContract(raw)
	if contract == nil || len(diagnostics) > 0 {
		findings := make([]IndexFinding, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			findings = append(findings, invalidStoryFinding(sourcePath, diagnostic.Pointer, diagnostic.Rule, "", diagnostic.Detail))
		}
		return nil, findings
	}
	if AssetKind(contract.Kind) != assetKind {
		return nil, []IndexFinding{invalidStoryFinding(sourcePath, "/kind", string(assetKind), string(contract.Kind), "story kind must match manifest asset kind")}
	}
	storyPath := filepath.ToSlash(filepath.Join(filepath.Dir(sourcePath), "story.tsx"))
	storySource, storyErr := fs.ReadFile(idx.fs, storyPath)
	harnesses := make(map[string]struct{})
	for _, definition := range contract.Stories {
		if definition.Harness != "" {
			harnesses[definition.Harness] = struct{}{}
		}
	}
	if len(harnesses) > 0 && errors.Is(storyErr, fs.ErrNotExist) {
		return nil, []IndexFinding{{Kind: IndexFindingStoryHarnessMissing, SourcePath: storyPath, Field: "/stories", Detail: "a story references a harness but story.tsx is missing"}}
	}
	if storyErr != nil && !errors.Is(storyErr, fs.ErrNotExist) {
		return nil, []IndexFinding{{Kind: IndexFindingStoryHarnessMissing, SourcePath: storyPath, Field: "/stories", Detail: storyErr.Error()}}
	}
	if storyErr == nil {
		exports := harnessExports(string(storySource))
		if len(harnesses) == 0 {
			return nil, []IndexFinding{{Kind: IndexFindingStoryHarnessOrphan, SourcePath: storyPath, Field: "/stories", Detail: "story.tsx exists but no story references a harness export"}}
		}
		for harness := range harnesses {
			if _, found := exports[harness]; !found {
				return nil, []IndexFinding{{Kind: IndexFindingStoryHarnessExport, SourcePath: storyPath, Field: "/stories/harness", Actual: harness, Detail: "referenced harness export was not found in story.tsx"}}
			}
		}
	}
	args, _ := json.Marshal(contract.Args)
	environment, _ := json.Marshal(contract.Environment)
	stories, _ := json.Marshal(contract.Stories)
	normalized, _ := json.Marshal(contract)
	return &ComponentStory{LibraryID: libraryID, Version: version, SchemaVersion: contract.SchemaVersion, Kind: contract.Kind, Title: contract.Title, ArgsJSON: string(args), EnvironmentJSON: string(environment), StoriesJSON: string(stories), ContractJSON: string(normalized), SourcePath: sourcePath}, nil
}

var storyHarnessExportRE = regexp.MustCompile(`(?m)^\s*export\s+(?:async\s+)?(?:function|const|let|var|class)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)

func harnessExports(source string) map[string]struct{} {
	exports := make(map[string]struct{})
	for _, match := range storyHarnessExportRE.FindAllStringSubmatch(source, -1) {
		if len(match) > 1 {
			exports[match[1]] = struct{}{}
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
	case strings.HasPrefix(clean, "hooks/"):
		inferred = AssetKindHook
	default:
		return "", ErrInvalidHeader{SourcePath: path, Field: "asset root", Reason: "manifest must be under components/ or hooks/"}
	}
	if declared == "" {
		return inferred, nil
	}
	kind := AssetKind(strings.ToLower(strings.TrimSpace(declared)))
	if !kind.Valid() || kind != inferred {
		return "", ErrInvalidHeader{SourcePath: path, Field: "assetKind", Reason: "must match authored asset root " + string(inferred)}
	}
	return kind, nil
}

func normalizeAssetDependencies(path string, in []AssetDependency) ([]AssetDependency, error) {
	seen := map[string]struct{}{}
	out := make([]AssetDependency, 0, len(in))
	for _, dep := range in {
		dep.LibraryID = strings.TrimSpace(dep.LibraryID)
		dep.Version = strings.TrimSpace(dep.Version)
		if dep.LibraryID == "" || dep.Version == "" {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "dependencies", Reason: "each dependency requires libraryId and version"}
		}
		key := dep.LibraryID + "\x00" + dep.Version
		if _, exists := seen[key]; exists {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "dependencies", Reason: "duplicate dependency " + dep.LibraryID + "@" + dep.Version}
		}
		seen[key] = struct{}{}
		out = append(out, dep)
	}
	return out, nil
}

func validEntryExtension(name string, kind AssetKind) bool {
	if kind == AssetKindHook {
		return strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".tsx")
	}
	return strings.HasSuffix(name, ".tsx")
}

func entryFileRequirement(kind AssetKind) string {
	if kind == AssetKindHook {
		return "TS file in the version folder"
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

func parseManifestDesignStyles(path string, raw []manifestDesignAffinity) ([]ComponentDesignAffinity, error) {
	out := make([]ComponentDesignAffinity, 0, len(raw))
	seen := map[string]struct{}{}
	for _, entry := range raw {
		styleID := strings.TrimSpace(entry.StyleID)
		affinity := DesignAffinity(strings.ToLower(strings.TrimSpace(entry.Affinity)))
		if styleID == "" {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "designStyles", Reason: "styleId required"}
		}
		if affinity == "" {
			affinity = DesignAffinityCompatible
		}
		if !isValidDesignAffinity(affinity) {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "designStyles", Reason: fmt.Sprintf("invalid affinity %q for style %q", affinity, styleID)}
		}
		if _, ok := seen[styleID]; ok {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "designStyles", Reason: fmt.Sprintf("duplicate style id %q", styleID)}
		}
		seen[styleID] = struct{}{}
		out = append(out, ComponentDesignAffinity{StyleID: styleID, Affinity: affinity, Reason: strings.TrimSpace(entry.Reason)})
	}
	return out, nil
}

func staleDesignStyleFindings(path string, affinities []ComponentDesignAffinity) ([]IndexFinding, error) {
	if len(affinities) == 0 {
		return nil, nil
	}
	root, err := defaultDesignRoot()
	if err != nil {
		return nil, err
	}
	styles, err := LoadDesignStyles(context.Background(), root)
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(styles))
	for _, style := range styles {
		known[style.ID] = struct{}{}
	}
	var findings []IndexFinding
	for _, affinity := range affinities {
		if _, ok := known[affinity.StyleID]; !ok {
			findings = append(findings, IndexFinding{
				Kind:       IndexFindingStaleDesignStyle,
				SourcePath: path,
				Field:      "designStyles",
				Expected:   "known design style",
				Actual:     affinity.StyleID,
				Detail:     fmt.Sprintf("component design affinity references unknown style id %q", affinity.StyleID),
			})
		}
	}
	return findings, nil
}

func isValidDesignAffinity(affinity DesignAffinity) bool {
	switch affinity {
	case DesignAffinityNative, DesignAffinityCompatible, DesignAffinityDiscouraged:
		return true
	default:
		return false
	}
}

func normalizeComponentSlot(raw string) (ComponentSlot, error) {
	slot := ComponentSlot(strings.ToLower(strings.TrimSpace(raw)))
	if slot == "" {
		return "", nil
	}
	switch slot {
	case ComponentSlotUIPrimitive, ComponentSlotUIPattern, ComponentSlotLayoutNav:
		return slot, nil
	default:
		return "", fmt.Errorf("invalid slot %q", raw)
	}
}

func isValidVersionStatus(status ComponentVersionStatus) bool {
	switch status {
	case VersionStatusDraft, VersionStatusReleased, VersionStatusDeprecated, VersionStatusArchived:
		return true
	default:
		return false
	}
}

func errInvalidHeader(path, field, reason string) ErrInvalidHeader {
	return ErrInvalidHeader{SourcePath: path, Field: field, Reason: reason}
}

// headerBlockRe captures the first /** … */ comment block in a file.
// (?s) flag makes . match newlines.
var headerBlockRe = regexp.MustCompile(`(?s)/\*\*(.*?)\*/`)

// headerFieldRe captures @field-name value pairs on each header line.
// The leading `*` and surrounding whitespace are stripped by the
// caller before this matches.
var headerFieldRe = regexp.MustCompile(`^@([A-Za-z][A-Za-z0-9_-]*)\s+(.*)$`)

// extractHeaderBlock returns the inner text of the first JSDoc-style
// header block, or ok=false if no such block exists. Only blocks that
// contain `@libraryId` are treated as library headers; otherwise the
// indexer would claim every JSDoc-commented file.
func extractHeaderBlock(src string) (string, bool) {
	matches := headerBlockRe.FindStringSubmatch(src)
	if len(matches) < 2 {
		return "", false
	}
	body := matches[1]
	if !strings.Contains(body, "@libraryId") {
		return "", false
	}
	return body, true
}

// parseHeader extracts field/value pairs from a header block. Multi-
// line values are folded onto the @field line until the next @field
// or end-of-block (so JSDoc continuation lines are tolerated).
func parseHeader(path, body string) (map[string]string, error) {
	out := map[string]string{}
	var currentField string
	var currentValue strings.Builder

	flush := func() {
		if currentField == "" {
			return
		}
		out[currentField] = strings.TrimSpace(currentValue.String())
		currentField = ""
		currentValue.Reset()
	}

	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := headerFieldRe.FindStringSubmatch(line); m != nil {
			flush()
			currentField = m[1]
			currentValue.WriteString(m[2])
			continue
		}
		if currentField != "" {
			currentValue.WriteByte(' ')
			currentValue.WriteString(line)
		}
	}
	flush()

	if _, ok := out["libraryId"]; !ok {
		return nil, ErrInvalidHeader{SourcePath: path, Field: "libraryId", Reason: "required"}
	}
	return out, nil
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
