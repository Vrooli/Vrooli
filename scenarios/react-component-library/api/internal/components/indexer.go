package components

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Indexer walks a configured filesystem root for registry-backed manifests under
// components/*/component.json, foundations/*/component.json, and
// hooks/*/component.json and upserts the manifest plus its version folders
// into the Repository. Catalog services are intentionally owned by catalog
// coverage rather than this component registry, so they are skipped during
// indexing. Renderable primitives are part of the registry even though they
// live in their own authored root; a final DeleteMissing call
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
	diskMutationRoot string
	repo             Repository
	root             string
	fs               fs.FS // injected for tests; nil means use os.DirFS(root)
	observer         UpsertObserver
	assetDirectories []string
	kitVocabularies  map[string][]string
	kitVocabularyErr error
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
	diskMutationRoot := ""
	if fsys == nil && root != "" {
		diskMutationRoot = root
		fsys = os.DirFS(root)
	}
	idx := &Indexer{diskMutationRoot: diskMutationRoot, repo: repo, root: root, fs: fsys, assetDirectories: CatalogKindDirectories(root)}
	if repoRoot := repositoryRootFromLibrary(root); repoRoot != "" {
		idx.kitVocabularies, idx.kitVocabularyErr = loadKitTokenVocabularies(repoRoot)
	}
	return idx
}

// IndexResult summarizes one Run.
type IndexResult struct {
	Scanned    int
	Indexed    int
	Skipped    int
	Deleted    int
	Errors     []error
	Findings   []IndexFinding
	Warnings   []string
	LibraryIDs []string // upserted IDs in walk order — useful for tests
}

// IndexManifest validates and upserts one known registry manifest without
// walking or sweeping the rest of the catalog. Authoring operations use this
// path so an unrelated component cannot block a draft or publication.
func (idx *Indexer) IndexManifest(ctx context.Context, path string) (Component, error) {
	return idx.indexManifest(ctx, path, "")
}

// IndexAuthoringManifest refreshes a selected draft or new release. Historical
// releases retain their recorded bytes; creating a successor must not rewrite
// history or require repairing unrelated historical files. IndexManifest and
// Run remain the integrity checks for the full on-disk manifest.
func (idx *Indexer) IndexAuthoringManifest(ctx context.Context, path, targetVersion string) (Component, error) {
	return idx.indexManifest(ctx, path, targetVersion)
}

func (idx *Indexer) indexManifest(ctx context.Context, path, targetVersion string) (Component, error) {
	if idx.diskMutationRoot != "" {
		var release func()
		var err error
		ctx, release, err = AcquireLibraryMutation(ctx, idx.diskMutationRoot)
		if err != nil {
			return Component{}, err
		}
		defer release()
	}
	if idx.kitVocabularyErr != nil {
		return Component{}, idx.kitVocabularyErr
	}
	if idx.fs == nil {
		return Component{}, fmt.Errorf("indexer has no filesystem configured")
	}
	path = filepath.ToSlash(strings.TrimSpace(path))
	if filepath.Base(path) != "component.json" || !idx.registryAssetPath(path) {
		return Component{}, fmt.Errorf("index manifest %q: expected a registry component.json path", path)
	}
	in, _, err := idx.buildManifestInput(path)
	if err != nil {
		return Component{}, err
	}
	// Authoring takes this path, so it needs the same attestations the full
	// walk uses; without them a publish would be refused by a stale index row
	// that the committed registry already supersedes.
	if in.ReleaseAttestations, err = loadReleaseAttestations(idx.fs); err != nil {
		return Component{}, fmt.Errorf("read released version hash registry: %w", err)
	}
	if targetVersion != "" {
		if err := idx.preserveHistoricalReleases(ctx, &in, targetVersion); err != nil {
			return Component{}, err
		}
	}
	component, upsertErr, observeErr := idx.upsertManifest(ctx, path, in)
	if upsertErr != nil {
		return Component{}, upsertErr
	}
	if observeErr != nil {
		return Component{}, observeErr
	}
	return component, nil
}

// upsertManifest is the one persistence boundary for both targeted authoring
// refreshes and whole-library indexing. It keeps repository and observer
// failures distinct because Run records observer failures while preserving a
// successful manifest upsert; the authoring path fails the operation closed.
func (idx *Indexer) upsertManifest(ctx context.Context, path string, in IndexManifestInput) (Component, error, error) {
	component, err := idx.repo.UpsertManifest(ctx, in)
	if err != nil {
		return Component{}, fmt.Errorf("upsert %s: %w", path, err), nil
	}
	if idx.observer == nil {
		return component, nil, nil
	}
	if err := idx.observer.Observe(ctx, component, in); err != nil {
		return component, nil, fmt.Errorf("observe %s: %w", path, err)
	}
	return component, nil, nil
}

func (idx *Indexer) preserveHistoricalReleases(ctx context.Context, in *IndexManifestInput, target string) error {
	component, err := idx.repo.GetByLibraryID(ctx, in.Manifest.LibraryID)
	if err != nil {
		return err
	}
	versions, err := idx.repo.ListVersions(ctx, component.ID, 10000)
	if err != nil {
		return err
	}
	if len(versions) >= 10000 {
		return fmt.Errorf("historical release preservation exceeds version limit")
	}
	released := map[string]bool{}
	for _, v := range versions {
		released[v.Version] = v.Status == VersionStatusReleased
	}
	for i, incoming := range in.Versions {
		if incoming.Version == target || !released[incoming.Version] {
			continue
		}
		recorded, err := idx.repo.GetVersion(ctx, component.ID, incoming.Version)
		if err != nil {
			return err
		}
		if releaseHashMatches(recorded.ContentSHA256, incoming.Content, incoming.ContentSHA256) ||
			attestsRelease(in.ReleaseAttestations, incoming.SourcePath, incoming.ContentSHA256) {
			continue
		}
		in.Versions[i] = recorded
		stories, err := idx.repo.ListStories(ctx, StoryQuery{ComponentID: component.ID, Version: incoming.Version, Limit: 10000})
		if err != nil {
			return err
		}
		if len(stories) >= 10000 {
			return fmt.Errorf("historical release preservation exceeds story limit")
		}
		kept := in.Stories[:0]
		for _, story := range in.Stories {
			if story.Version != incoming.Version {
				kept = append(kept, story)
			}
		}
		in.Stories = append(kept, stories...)
	}
	return nil
}

// Run walks the root, upserts every manifest with valid version folders,
// and returns a result. Malformed manifests are recorded in Errors but
// do not stop the walk — a single broken component should not hide an
// otherwise healthy run.
func (idx *Indexer) Run(ctx context.Context) (IndexResult, error) {
	if idx.diskMutationRoot != "" {
		var release func()
		var err error
		ctx, release, err = AcquireLibraryMutation(ctx, idx.diskMutationRoot)
		if err != nil {
			return IndexResult{}, err
		}
		defer release()
	}
	var result IndexResult
	if idx.kitVocabularyErr != nil {
		return result, idx.kitVocabularyErr
	}
	// A manifest that was discovered but failed validation/upsert is still a
	// live filesystem asset. Protect its last-known registry row from the
	// end-of-run stale sweep; immutable-release failures must not look like
	// deletions to adopters.
	keepLibraryIDs := map[string]struct{}{}
	if idx.fs == nil {
		return result, fmt.Errorf("indexer has no filesystem configured")
	}
	if _, err := fs.Stat(idx.fs, "released-version-hashes.json"); err == nil {
		if err := ValidateVersionDependencyLocks(idx.fs); err != nil {
			return result, err
		}
	}
	if _, err := idx.repo.RestoreEvictedStories(ctx); err != nil {
		return result, fmt.Errorf("restore evicted story projections: %w", err)
	}
	// Read once per run rather than per manifest: the registry describes the
	// whole library and the walk visits 236 manifests.
	releaseAttestations, err := loadReleaseAttestations(idx.fs)
	if err != nil {
		return result, fmt.Errorf("read released version hash registry: %w", err)
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
		if !idx.registryAssetPath(path) {
			result.Skipped++
			return nil
		}
		result.Scanned++
		in, _, perr := idx.buildManifestInput(path)
		if perr != nil {
			result.Errors = append(result.Errors, perr)
			return nil
		}
		result.Findings = append(result.Findings, in.Findings...)
		result.Warnings = append(result.Warnings, in.Warnings...)
		in.ReleaseAttestations = releaseAttestations
		keepLibraryIDs[in.Manifest.LibraryID] = struct{}{}
		_, upsertErr, observeErr := idx.upsertManifest(ctx, path, in)
		if upsertErr != nil {
			result.Errors = append(result.Errors, upsertErr)
			return nil
		}
		if observeErr != nil {
			result.Errors = append(result.Errors, observeErr)
			// continue — observer failure must not hide the upsert.
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
	keep := make([]string, 0, len(keepLibraryIDs))
	for libraryID := range keepLibraryIDs {
		keep = append(keep, libraryID)
	}
	sort.Strings(keep)
	deleted, derr := idx.repo.DeleteMissing(ctx, keep)
	if derr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("delete missing: %w", derr))
	}
	result.Deleted = deleted

	return result, nil
}

func (idx *Indexer) registryAssetPath(path string) bool {
	clean := filepath.ToSlash(path)
	for _, directory := range idx.assetDirectories {
		if strings.HasPrefix(clean, directory+"/") {
			return true
		}
	}
	return false
}

// sortedVersions gives the evicted set a stable order so findings and the
// version list do not vary between runs over the same tree.
func sortedVersions(evicted map[string]struct{}) []string {
	out := make([]string, 0, len(evicted))
	for version := range evicted {
		out = append(out, version)
	}
	sort.Strings(out)
	return out
}

func unrecoverableEvictedFinding(sourcePath, libraryID string, versions []string, detail string) IndexFinding {
	return IndexFinding{
		Kind:       IndexFindingUnrecoverableEvictedVersion,
		SourcePath: sourcePath,
		Field:      "evictedVersions",
		Expected:   "a durable file mirror for every declared evicted version",
		Actual:     strings.Join(versions, ", "),
		Detail:     fmt.Sprintf("%s: %s (%s); the version is omitted from the index and the rest of the asset is indexed", libraryID, detail, strings.Join(versions, ", ")),
	}
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
