package components

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// defaultListLimit caps List rows when callers pass 0. Business
// policy, not transport policy — lives next to the only code that
// applies it.
const defaultListLimit = 200

// maxSlugResolutionScan bounds the catalog scan used to resolve an unqualified
// component name. It sits above the live catalog size so resolution stays
// complete, and exists only so a pathological catalog cannot turn a lookup into
// an unbounded read.
const maxSlugResolutionScan = 2000

// toKebabSlug lowercases and hyphenates a catalog slug so PascalCase library
// names and the kebab-case ids an experience spec is required to use resolve to
// the same component.
func toKebabSlug(name string) string {
	var b strings.Builder
	for i, r := range strings.TrimSpace(name) {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte('-')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return strings.Trim(b.String(), "-")
}

// Service is the application-layer surface handlers and the indexer
// depend on. Owns validation, default substitution, and any cross-
// handler policy that doesn't belong in transport.
type Service interface {
	Upsert(ctx context.Context, in UpsertInput) (Component, error)
	Get(ctx context.Context, id string) (Component, error)
	GetByLibraryID(ctx context.Context, libraryID string) (Component, error)
	List(ctx context.Context, q SearchQuery) ([]Component, error)
	GetContent(ctx context.Context, id string) (Content, error)
	GetContentAt(ctx context.Context, id, path string) (Content, error)
	ListVersions(ctx context.Context, componentID string, limit int) ([]ComponentVersion, error)
	GetVersion(ctx context.Context, componentID, version string) (ComponentVersion, error)
	ListStories(ctx context.Context, q StoryQuery) ([]ComponentStory, error)
	GetVersionContent(ctx context.Context, componentID, version string) (Content, error)
	MaterializeVersion(ctx context.Context, componentID, version, into string) error
	ListDesignStyles(ctx context.Context) ([]DesignStyle, error)
	ValidateDesignStyle(ctx context.Context, id string) error
	ValidateStyleFit(ctx context.Context, componentID, version, scenario string) (StyleFitVerdict, error)
	UpdateContent(ctx context.Context, id string, in WriteContentInput) (Content, error)
	UpdateContentAt(ctx context.Context, id, path string, in WriteContentInput) (Content, error)
	InitializeComponent(ctx context.Context, in InitializeComponentInput) (InitializeComponentResult, error)
	IngestComponent(ctx context.Context, in IngestComponentInput) (IngestComponentResult, error)
	CreateComponentVersion(ctx context.Context, in CreateComponentVersionInput) (CreateComponentVersionResult, error)
	UpdateComponentManifest(ctx context.Context, in UpdateComponentManifestInput) (Component, error)
}

// Materializer is the narrow capability used by package/export consumers to
// restore an immutable version without coupling them to the catalog service.
type Materializer interface {
	EnsureMaterialized(ctx context.Context, componentID, version, into string) (MaterializeResult, error)
}

// PresenceReconciler applies the reachability-derived materialization tier.
// It is intentionally narrow so lifecycle domains can trigger the same
// policy without depending on the versions transport package.
type PresenceReconciler interface {
	ReconcilePresence(ctx context.Context, componentID string, apply bool) error
}

// MaterializeResult reports the outcome of one idempotent restore.
type MaterializeResult struct {
	Version        string
	Directory      string
	FilesWritten   int
	AlreadyPresent bool
	Transient      bool
}

type AuthoringService interface {
	BeginComponentVersion(ctx context.Context, in BeginComponentVersionInput) (AuthoringVersionResult, error)
	CheckComponentVersion(ctx context.Context, componentID, version string) (CheckComponentVersionResult, error)
	PublishComponentVersion(ctx context.Context, in PublishComponentVersionInput) (AuthoringVersionResult, error)
}

// ContentChangeListener is an optional sink the service invokes after a
// successful UpdateContent write. Wired by main.go to drive the
// versions recorder (req 11). Listener errors are logged at the
// handler edge and do not roll back the save — the file on disk is
// already updated. Keep listeners idempotent and fast.
type ContentChangeListener interface {
	OnContentSaved(ctx context.Context, c Component, content Content) error
}

// ServiceJSONReader is the target-scenario-tree seam ValidateStyleFit
// uses to read .vrooli/service.json. Production walks the configured
// scenarios root with a traversal guard; tests inject a fake.
type ServiceJSONReader interface {
	Read(ctx context.Context, scenario string) ([]byte, error)
}

type service struct {
	repo     Repository
	content  ContentStore
	source   SourceStore
	services ServiceJSONReader
	listener ContentChangeListener
	ingest   ScenarioSourceReader
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func NewServiceWithScenarioReader(repo Repository, services ServiceJSONReader) Service {
	return &service{repo: repo, services: services}
}

// NewServiceWithContent wires the optional ContentStore seam. Read
// paths work without it; GetContent/UpdateContent return an error when
// content is nil. Production constructs the store from the configured
// source root; tests inject a fake.
func NewServiceWithContent(repo Repository, content ContentStore) Service {
	s := &service{repo: repo, content: content}
	if source, ok := content.(SourceStore); ok {
		s.source = source
	}
	return s
}

// SetContentChangeListener installs the post-save listener. Designed
// to be called once at boot before any requests land; not concurrency-
// safe with in-flight UpdateContent.
func SetContentChangeListener(svc Service, l ContentChangeListener) {
	if s, ok := svc.(*service); ok {
		s.listener = l
	}
}

func SetServiceJSONReader(svc Service, reader ServiceJSONReader) {
	if s, ok := svc.(*service); ok {
		s.services = reader
	}
}

// SetScenarioSourceReader enables the ingest workflow to read a source file
// from a sibling scenario through the same guarded filesystem seam adoptions
// use. It is installed once during process wiring.
func SetScenarioSourceReader(svc Service, reader ScenarioSourceReader) {
	if s, ok := svc.(*service); ok {
		s.ingest = reader
	}
}

var _ Service = (*service)(nil)

func (s *service) Upsert(ctx context.Context, in UpsertInput) (Component, error) {
	in.LibraryID = strings.TrimSpace(in.LibraryID)
	if in.LibraryID == "" {
		return Component{}, ErrInvalidHeader{SourcePath: in.SourcePath, Field: "libraryId", Reason: "required"}
	}
	return s.repo.Upsert(ctx, in)
}

func (s *service) Get(ctx context.Context, id string) (Component, error) {
	c, err := s.repo.Get(ctx, id)
	if err == nil || !errors.As(err, &ErrComponentNotFound{}) {
		return c, err
	}
	// Public component identifiers are the stable manifest library IDs. Keep
	// accepting the internal UUID as well, but make both identifiers equivalent
	// at the service boundary so sibling domains do not need to guess which one
	// the repository stores.
	c, err = s.repo.GetByLibraryID(ctx, id)
	if err == nil || !errors.As(err, &ErrComponentNotFound{}) {
		return c, err
	}
	// Callers legitimately hold the unqualified slug rather than the
	// library-qualified id: the CLI takes a component name, and an experience
	// spec pins the same component in kebab-case because its schema requires
	// kebab-case ids. Resolve those to the same component instead of reporting
	// it missing, which reads as "this component does not exist".
	return s.getBySlug(ctx, id)
}

// getBySlug resolves an unqualified component name. It matches the catalog slug
// case-insensitively and in kebab-case, and refuses an ambiguous match rather
// than picking one, so the caller is told to qualify the id.
func (s *service) getBySlug(ctx context.Context, name string) (Component, error) {
	wanted := strings.TrimSpace(name)
	if wanted == "" || strings.Contains(wanted, ":") {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: name}
	}
	all, listErr := s.repo.List(ctx, SearchQuery{Limit: maxSlugResolutionScan})
	if listErr != nil {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: name}
	}
	var matches []Component
	for _, candidate := range all {
		slug := strings.TrimSpace(candidate.Slug)
		if slug == "" {
			continue
		}
		if strings.EqualFold(slug, wanted) || strings.EqualFold(toKebabSlug(slug), toKebabSlug(wanted)) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: name}
	}
	return matches[0], nil
}

func (s *service) GetByLibraryID(ctx context.Context, libraryID string) (Component, error) {
	return s.repo.GetByLibraryID(ctx, libraryID)
}

func (s *service) List(ctx context.Context, q SearchQuery) ([]Component, error) {
	if q.Limit <= 0 {
		q.Limit = defaultListLimit
	}
	return s.repo.List(ctx, q)
}

func (s *service) GetContent(ctx context.Context, id string) (Content, error) {
	return s.GetContentAt(ctx, id, "")
}

func (s *service) GetContentAt(ctx context.Context, id, path string) (Content, error) {
	if s.content == nil {
		return Content{}, errNoContentStore
	}
	c, err := s.Get(ctx, id)
	if err != nil {
		return Content{}, err
	}
	draftVersion := strings.TrimSpace(c.DraftVersion)
	if draftVersion != "" {
		draft, draftErr := s.repo.GetVersion(ctx, c.ID, draftVersion)
		if draftErr != nil {
			return Content{}, fmt.Errorf("resolve active draft %s@%s: %w", c.LibraryID, draftVersion, draftErr)
		}
		if draft.Status != VersionStatusDraft {
			return Content{}, ErrInvalidHeader{SourcePath: draft.SourcePath, Field: "draft", Reason: "active draft pointer does not resolve to a mutable draft"}
		}
		c.SourcePath = draft.SourcePath
	}
	if path != "" {
		store, ok := s.content.(PathContentStore)
		if !ok {
			return Content{}, errNoContentStore
		}
		return store.ReadPath(ctx, c, path)
	}
	return s.content.Read(ctx, c)
}

func (s *service) ListVersions(ctx context.Context, componentID string, limit int) ([]ComponentVersion, error) {
	c, err := s.Get(ctx, componentID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListVersions(ctx, c.ID, limit)
}

func (s *service) GetVersion(ctx context.Context, componentID, version string) (ComponentVersion, error) {
	c, err := s.Get(ctx, componentID)
	if err != nil {
		return ComponentVersion{}, err
	}
	return s.repo.GetVersion(ctx, c.ID, version)
}

func (s *service) ListStories(ctx context.Context, q StoryQuery) ([]ComponentStory, error) {
	if q.Limit <= 0 {
		q.Limit = defaultListLimit
	}
	if strings.TrimSpace(q.ComponentID) != "" {
		component, err := s.Get(ctx, q.ComponentID)
		if err != nil {
			return nil, err
		}
		q.ComponentID = component.ID
	}
	return s.repo.ListStories(ctx, q)
}

func (s *service) ValidateStyleFit(ctx context.Context, componentID, version, scenario string) (StyleFitVerdict, error) {
	cid := strings.TrimSpace(componentID)
	if cid == "" {
		return StyleFitVerdict{}, fmt.Errorf("component_id required")
	}
	scn := strings.TrimSpace(scenario)
	if scn == "" {
		return StyleFitVerdict{}, fmt.Errorf("scenario required")
	}
	if s.services == nil {
		return StyleFitVerdict{}, fmt.Errorf("service.json reader not configured")
	}

	component, err := s.Get(ctx, cid)
	if err != nil {
		return StyleFitVerdict{}, err
	}
	if strings.TrimSpace(version) != "" {
		if _, err := s.GetVersion(ctx, cid, strings.TrimSpace(version)); err != nil {
			return StyleFitVerdict{}, err
		}
	}

	styleID, err := readScenarioDesignStyle(ctx, s.services, scn)
	if err != nil {
		return StyleFitVerdict{}, err
	}
	out := StyleFitVerdict{
		Kind:          StyleFitVerdictInfo,
		ComponentID:   cid,
		Version:       strings.TrimSpace(version),
		Scenario:      scn,
		ScenarioStyle: styleID,
		Detail:        fmt.Sprintf("component %q declares no affinity for scenario design style %q", component.LibraryID, styleID),
	}
	if styleID == "" {
		out.Kind = StyleFitVerdictWarn
		out.Detail = fmt.Sprintf("scenario %q does not declare generation.design.id", scn)
		return out, nil
	}
	for _, affinity := range component.DesignStyles {
		if !strings.EqualFold(affinity.StyleID, styleID) {
			continue
		}
		out.Affinity = affinity.Affinity
		out.Detail = styleFitDetail(component.LibraryID, styleID, affinity)
		switch affinity.Affinity {
		case DesignAffinityNative, DesignAffinityCompatible:
			out.Kind = StyleFitVerdictOK
		case DesignAffinityDiscouraged:
			out.Kind = StyleFitVerdictWarn
		default:
			out.Kind = StyleFitVerdictInfo
		}
		return out, nil
	}
	return out, nil
}

func (s *service) GetVersionContent(ctx context.Context, componentID, version string) (Content, error) {
	v, err := s.GetVersion(ctx, componentID, version)
	if err != nil {
		return Content{}, err
	}
	return Content{Body: v.Content, SourcePath: v.SourcePath, SHA256: v.ContentSHA256}, nil
}

func (s *service) MaterializeVersion(ctx context.Context, componentID, version, into string) error {
	_, err := s.EnsureMaterialized(ctx, componentID, version, into)
	return err
}

func (s *service) EnsureMaterialized(ctx context.Context, componentID, version, into string) (MaterializeResult, error) {
	store, ok := s.content.(*FSContentStore)
	if !ok {
		return MaterializeResult{}, fmt.Errorf("materialization requires a filesystem content store")
	}
	c, err := s.Get(ctx, componentID)
	if err != nil {
		return MaterializeResult{}, err
	}
	v, err := s.repo.GetVersion(ctx, c.ID, strings.TrimSpace(version))
	if err != nil {
		return MaterializeResult{}, err
	}
	result, err := store.Materialize(ctx, v, into)
	if err != nil {
		return MaterializeResult{}, err
	}
	if strings.TrimSpace(into) == "" || filepath.Clean(into) == filepath.Clean(store.Root()) {
		if err := s.repo.SetVersionPresence(ctx, c.ID, version, "materialized"); err != nil {
			return MaterializeResult{}, err
		}
	}
	result.Version = v.Version
	result.Transient = strings.TrimSpace(into) != "" && filepath.Clean(into) != filepath.Clean(store.Root())
	return result, nil
}

// GetVersionContentAt reads a companion source file beside an immutable
// version entry. It intentionally is not part of Service: preview is the only
// consumer and probes for this capability so existing service fakes stay small.
func (s *service) GetVersionContentAt(ctx context.Context, componentID, version, path string) (Content, error) {
	if s.content == nil {
		return Content{}, errNoContentStore
	}
	v, err := s.GetVersion(ctx, componentID, version)
	if err != nil {
		return Content{}, err
	}
	c, err := s.Get(ctx, componentID)
	if err != nil {
		return Content{}, err
	}
	entry := filepath.Base(c.SourcePath)
	if entry == "." || entry == string(filepath.Separator) || entry == "" || strings.HasSuffix(entry, ".json") {
		entry = c.Slug + ".tsx"
	}
	c.SourcePath = v.SourcePath
	if v.Presence == "evicted" {
		if _, err := s.EnsureMaterialized(ctx, componentID, version, ""); err != nil {
			return Content{}, err
		}
	}
	if !strings.Contains(filepath.ToSlash(c.SourcePath), "/versions/") {
		c.SourcePath = filepath.ToSlash(filepath.Join("components", c.Slug, "versions", v.Version, entry))
	}
	store, ok := s.content.(PathContentStore)
	if !ok {
		return Content{}, errNoContentStore
	}
	return store.ReadPath(ctx, c, path)
}

func (s *service) UpdateContent(ctx context.Context, id string, in WriteContentInput) (Content, error) {
	return s.UpdateContentAt(ctx, id, "", in)
}

func (s *service) UpdateContentAt(ctx context.Context, id, path string, in WriteContentInput) (Content, error) {
	if s.content == nil {
		return Content{}, errNoContentStore
	}
	c, err := s.Get(ctx, id)
	if err != nil {
		return Content{}, err
	}
	draftVersion := strings.TrimSpace(c.DraftVersion)
	if draftVersion == "" {
		return Content{}, ErrInvalidHeader{SourcePath: c.ManifestPath, Field: "draft", Reason: "released component versions are immutable; begin a draft before writing content"}
	}
	draft, err := s.repo.GetVersion(ctx, c.ID, draftVersion)
	if err != nil {
		return Content{}, fmt.Errorf("resolve active draft %s@%s: %w", c.LibraryID, draftVersion, err)
	}
	if draft.Status != VersionStatusDraft {
		return Content{}, ErrInvalidHeader{SourcePath: draft.SourcePath, Field: "draft", Reason: "active draft pointer does not resolve to a mutable draft"}
	}
	// The component projection intentionally points at latest for consumers.
	// Authoring writes must instead bind to the explicit mutable draft.
	c.SourcePath = draft.SourcePath
	// JSON is a source artifact, not an opaque text blob. Normalize it at the
	// write boundary so story contracts, experience contracts, and companion
	// metadata remain deterministic regardless of which authoring surface wrote
	// them. TypeScript formatting is owned by the source store's version-creation
	// path; JSON can be canonicalized without a toolchain.
	if strings.EqualFold(filepath.Ext(firstNonEmpty(path, c.SourcePath)), ".json") {
		formatted, formatErr := canonicalJSONText(in.Body)
		if formatErr != nil {
			return Content{}, fmt.Errorf("format JSON artifact %q: %w", firstNonEmpty(path, c.SourcePath), formatErr)
		}
		in.Body = formatted
	}
	var written Content
	if path != "" {
		store, ok := s.content.(PathContentStore)
		if !ok {
			return Content{}, errNoContentStore
		}
		written, err = store.WritePath(ctx, c, path, in)
	} else {
		written, err = s.content.Write(ctx, c, in)
	}
	if err != nil {
		return Content{}, err
	}
	if s.listener != nil {
		// Listener errors are non-fatal — the file is already written.
		// Callers swallow + log at the handler edge.
		_ = s.listener.OnContentSaved(ctx, c, written)
	}
	return written, nil
}

// UpdateVersionContentAt writes a companion artifact in an explicitly named
// draft version. Preview authoring uses this narrow seam to persist a frame
// choice without ever mutating a released version or falling back to the
// manifest's moving latest pointer.
func (s *service) UpdateVersionContentAt(ctx context.Context, componentID, version, path string, in WriteContentInput) (Content, error) {
	if s.content == nil {
		return Content{}, errNoContentStore
	}
	if strings.TrimSpace(path) == "" {
		return Content{}, ErrInvalidHeader{SourcePath: path, Field: "path", Reason: "version companion path is required"}
	}
	c, err := s.Get(ctx, componentID)
	if err != nil {
		return Content{}, err
	}
	v, err := s.GetVersion(ctx, c.ID, strings.TrimSpace(version))
	if err != nil {
		return Content{}, err
	}
	if v.Status != VersionStatusDraft {
		return Content{}, ErrInvalidHeader{SourcePath: v.SourcePath, Field: "version", Reason: "released component versions are immutable; create a draft first"}
	}
	store, ok := s.content.(PathContentStore)
	if !ok {
		return Content{}, errNoContentStore
	}
	entry := filepath.Base(c.SourcePath)
	if entry == "." || entry == string(filepath.Separator) || entry == "" || strings.HasSuffix(entry, ".json") {
		entry = c.Slug + ".tsx"
	}
	c.SourcePath = v.SourcePath
	if !strings.Contains(filepath.ToSlash(c.SourcePath), "/versions/") {
		c.SourcePath = filepath.ToSlash(filepath.Join("components", c.Slug, "versions", v.Version, entry))
	}
	if strings.EqualFold(filepath.Ext(path), ".json") {
		formatted, formatErr := canonicalJSONText(in.Body)
		if formatErr != nil {
			return Content{}, fmt.Errorf("format JSON artifact %q: %w", path, formatErr)
		}
		in.Body = formatted
	}
	written, err := store.WritePath(ctx, c, path, in)
	if err != nil {
		return Content{}, fmt.Errorf("write version artifact %q from %q: %w", path, c.SourcePath, err)
	}
	if _, err := NewIndexer(s.repo, s.source.Root(), nil).IndexManifest(ctx, c.ManifestPath); err != nil {
		return Content{}, err
	}
	return written, nil
}

func (s *service) InitializeComponent(ctx context.Context, in InitializeComponentInput) (InitializeComponentResult, error) {
	if s.source == nil {
		return InitializeComponentResult{}, errNoSourceStore
	}
	in.LibraryID = strings.TrimSpace(in.LibraryID)
	in.Slug = normalizeSlug(firstNonEmpty(in.Slug, in.DisplayName, in.LibraryID))
	if in.LibraryID == "" {
		in.LibraryID = "react-component-library:" + in.Slug
	}
	if _, err := s.repo.GetByLibraryID(ctx, in.LibraryID); err == nil {
		return InitializeComponentResult{}, ErrComponentAlreadyExists{LibraryID: in.LibraryID}
	} else if !errors.As(err, &ErrComponentNotFound{}) {
		return InitializeComponentResult{}, err
	}
	manifestPath, sourcePath, err := s.source.InitializeComponent(ctx, in)
	if err != nil {
		return InitializeComponentResult{}, err
	}
	if _, err := NewIndexer(s.repo, s.source.Root(), nil).Run(ctx); err != nil {
		return InitializeComponentResult{}, err
	}
	c, err := s.repo.GetByLibraryID(ctx, in.LibraryID)
	if err != nil {
		return InitializeComponentResult{}, err
	}
	return InitializeComponentResult{Component: c, ManifestPath: manifestPath, SourcePath: sourcePath}, nil
}

func (s *service) BeginComponentVersion(ctx context.Context, in BeginComponentVersionInput) (AuthoringVersionResult, error) {
	c, err := s.Get(ctx, strings.TrimSpace(in.Component))
	if err != nil {
		return AuthoringVersionResult{}, err
	}
	if c.DraftVersion != "" {
		return AuthoringVersionResult{}, ErrInvalidHeader{SourcePath: c.ManifestPath, Field: "draft", Reason: fmt.Sprintf("active draft %s already exists; publish or discard it before beginning another", c.DraftVersion)}
	}
	releaseVersion := strings.TrimSpace(in.Version)
	if releaseVersion == "" {
		releaseVersion, err = bumpReleaseVersion(c.LatestVersion, in.Bump)
		if err != nil {
			return AuthoringVersionResult{}, err
		}
	}
	if err := validateVersionToken(releaseVersion); err != nil {
		return AuthoringVersionResult{}, err
	}
	if strings.Contains(releaseVersion, "-") {
		return AuthoringVersionResult{}, ErrInvalidHeader{SourcePath: c.ManifestPath, Field: "version", Reason: "begin version must identify a release semver without a prerelease suffix"}
	}
	draftVersion := releaseVersion + "-draft.1"
	created, err := s.CreateComponentVersion(ctx, CreateComponentVersionInput{
		ComponentID: c.ID,
		Version:     draftVersion,
		FromVersion: c.LatestVersion,
		Intent:      VersionIntentDraft,
	})
	if err != nil {
		return AuthoringVersionResult{}, err
	}
	return s.authoringResult(created)
}

func (s *service) CheckComponentVersion(ctx context.Context, componentID, version string) (CheckComponentVersionResult, error) {
	if s.source == nil {
		return CheckComponentVersionResult{}, errNoSourceStore
	}
	c, err := s.Get(ctx, strings.TrimSpace(componentID))
	if err != nil {
		return CheckComponentVersionResult{}, err
	}
	// A focused check is also the freshness boundary for authoring. The
	// registry is an index, not the source of truth; re-index this one manifest
	// before resolving its version and dependency closure so edits to a draft
	// cannot be hidden behind a stale catalog row. This deliberately avoids a
	// global walk, keeping the rapid component loop independent of unrelated
	// catalog errors.
	refreshed, refreshErr := NewIndexer(s.repo, s.source.Root(), nil).IndexManifest(ctx, c.ManifestPath)
	if refreshErr == nil {
		c = refreshed
	}
	version = firstNonEmpty(strings.TrimSpace(version), c.DraftVersion, c.LatestVersion)
	result := CheckComponentVersionResult{Component: c, Version: version, Passed: true}
	add := func(stage, verdict, message, remediation string) {
		result.Checks = append(result.Checks, ComponentVersionCheck{Stage: stage, Verdict: verdict, Message: message, Remediation: remediation})
		if verdict != "passed" {
			result.Passed = false
		}
	}
	if refreshErr != nil {
		add("index", "failed", "focused source refresh failed: "+refreshErr.Error(), "repair the selected component before checking it again")
		return result, nil
	}
	v, err := s.repo.GetVersion(ctx, c.ID, version)
	if err != nil {
		return CheckComponentVersionResult{}, err
	}
	if strings.TrimSpace(v.Content) == "" {
		add("source", "failed", "version entry source is empty", "restore the entry source before publishing")
	} else {
		add("source", "passed", "version entry source is present", "")
	}
	if _, err := ResolveDependencyClosure(ctx, s, c.ID, version); err != nil {
		add("dependencies", "failed", err.Error(), "repair the version-pinned dependency closure")
	} else {
		add("dependencies", "passed", "version-pinned dependency closure resolved", "")
	}
	storyPath := filepath.Join(s.source.Root(), componentAssetRoot(c), c.Slug, "versions", version, "story.json")
	story, err := os.ReadFile(storyPath)
	if err != nil {
		add("story", "failed", "story.json is missing or unreadable: "+err.Error(), "restore the version story contract")
		return result, nil
	}
	contract, diagnostics := ParseStoryContract(story)
	if len(StoryContractErrors(diagnostics)) > 0 || contract == nil {
		messages := make([]string, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Error())
		}
		add("story", "failed", strings.Join(messages, "; "), "repair story.json before publishing")
		return result, nil
	}
	if gaps := StoryCoverageGaps(contract); len(gaps) > 0 {
		messages := make([]string, 0, len(gaps))
		for _, gap := range gaps {
			messages = append(messages, gap.Error())
		}
		add("story", "failed", strings.Join(messages, "; "), "add representative stories for the declared enum values")
	} else {
		add("story", "passed", "story contract parsed and declared enum coverage is complete", "")
	}
	contractPath := filepath.Join(s.source.Root(), componentAssetRoot(c), c.Slug, "versions", version, "experience-contract.json")
	contractBytes, contractErr := os.ReadFile(contractPath)
	if os.IsNotExist(contractErr) {
		// Contracts were introduced after the earliest library releases. Keep
		// those immutable historical versions checkable while requiring and
		// formatting contracts whenever a version actually carries one.
		add("experience-contract", "passed", "no experience contract is attached to this legacy version", "")
	} else if contractErr != nil {
		add("experience-contract", "failed", "experience-contract.json is unreadable: "+contractErr.Error(), "restore the version experience contract")
	} else if diagnostics := validateExperienceContract(contractBytes, *contract); len(diagnostics) > 0 {
		add("experience-contract", "failed", strings.Join(diagnostics, "; "), "repair experience-contract.json and align its states with story.json")
	} else {
		add("experience-contract", "passed", "experience contract is readable and its state references match story.json", "")
	}
	return result, nil
}

func (s *service) PublishComponentVersion(ctx context.Context, in PublishComponentVersionInput) (AuthoringVersionResult, error) {
	c, err := s.Get(ctx, strings.TrimSpace(in.Component))
	if err != nil {
		return AuthoringVersionResult{}, err
	}
	draftVersion := firstNonEmpty(strings.TrimSpace(in.DraftVersion), c.DraftVersion)
	if draftVersion == "" || !strings.Contains(draftVersion, "-") {
		return AuthoringVersionResult{}, ErrInvalidHeader{SourcePath: c.ManifestPath, Field: "draft", Reason: "an active prerelease draft is required"}
	}
	check, err := s.CheckComponentVersion(ctx, c.ID, draftVersion)
	if err != nil {
		return AuthoringVersionResult{}, err
	}
	if !check.Passed {
		return AuthoringVersionResult{}, ErrVersionCheckFailed{LibraryID: c.LibraryID, Version: draftVersion, Checks: check.Checks}
	}
	releaseVersion := firstNonEmpty(strings.TrimSpace(in.Version), strings.SplitN(draftVersion, "-", 2)[0])
	created, err := s.CreateComponentVersion(ctx, CreateComponentVersionInput{
		ComponentID:             c.ID,
		Version:                 releaseVersion,
		FromVersion:             draftVersion,
		Intent:                  VersionIntentRelease,
		ChangelogMD:             in.ChangelogMD,
		AcknowledgeParityWaiver: in.AcknowledgeParityWaiver,
	})
	if err != nil {
		return AuthoringVersionResult{}, err
	}
	return s.authoringResult(created)
}

func (s *service) authoringResult(created CreateComponentVersionResult) (AuthoringVersionResult, error) {
	paths, err := versionArtifactPaths(s.source, created.Component, created.Version.Version)
	if err != nil {
		return AuthoringVersionResult{}, err
	}
	return AuthoringVersionResult{Component: created.Component, Version: created.Version, SourcePath: created.SourcePath, ArtifactPaths: paths}, nil
}

func bumpReleaseVersion(current, bump string) (string, error) {
	parts := strings.Split(strings.TrimSpace(current), ".")
	if len(parts) != 3 {
		return "", ErrInvalidHeader{SourcePath: "component.json", Field: "latest", Reason: "latest version must be x.y.z before it can be bumped"}
	}
	values := make([]int, 3)
	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return "", ErrInvalidHeader{SourcePath: "component.json", Field: "latest", Reason: "latest version must be x.y.z before it can be bumped"}
		}
		values[index] = value
	}
	switch firstNonEmpty(strings.ToLower(strings.TrimSpace(bump)), "patch") {
	case "major":
		values[0]++
		values[1], values[2] = 0, 0
	case "minor":
		values[1]++
		values[2] = 0
	case "patch":
		values[2]++
	default:
		return "", ErrInvalidHeader{SourcePath: "component.json", Field: "bump", Reason: "must be major, minor, or patch"}
	}
	return fmt.Sprintf("%d.%d.%d", values[0], values[1], values[2]), nil
}

func versionArtifactPaths(source SourceStore, c Component, version string) ([]string, error) {
	if source == nil {
		return nil, errNoSourceStore
	}
	dir := filepath.Join(source.Root(), componentAssetRoot(c), c.Slug, "versions", version)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list version artifacts: %w", err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		paths = append(paths, filepath.ToSlash(filepath.Join(componentAssetRoot(c), c.Slug, "versions", version, entry.Name())))
	}
	sort.Strings(paths)
	return paths, nil
}

func (s *service) CreateComponentVersion(ctx context.Context, in CreateComponentVersionInput) (CreateComponentVersionResult, error) {
	if s.source == nil {
		return CreateComponentVersionResult{}, errNoSourceStore
	}
	c, err := s.Get(ctx, in.ComponentID)
	if err != nil {
		return CreateComponentVersionResult{}, err
	}
	intent := in.Intent
	if intent == "" && !strings.Contains(in.Version, "-") {
		intent = VersionIntentRelease
	}
	if intent == VersionIntentRelease {
		coverageVersion := firstNonEmpty(in.FromVersion, in.Version)
		if err := releaseStoryCoverage(s.source.Root(), c, coverageVersion); err != nil {
			return CreateComponentVersionResult{}, err
		}
		from := firstNonEmpty(in.FromVersion, c.DraftVersion)
		if from != "" && in.ParityReport == nil {
			if draft, err := s.repo.GetVersion(ctx, c.ID, from); err == nil && draft.ParityReport != nil {
				if len(draft.ParityReport.Findings) > 0 && !in.AcknowledgeParityWaiver {
					return CreateComponentVersionResult{}, ErrParityWaiverRequired{ComponentID: c.ID, Version: from, Findings: draft.ParityReport.Findings}
				}
				report := *draft.ParityReport
				report.Acknowledged = in.AcknowledgeParityWaiver
				in.ParityReport = &report
			}
		}
	}
	previous := c
	sourcePath, err := s.source.CreateVersion(ctx, c, in)
	if err != nil {
		return CreateComponentVersionResult{}, err
	}
	rollback := func(cause error) error {
		if store, ok := s.source.(interface {
			RollbackVersion(context.Context, Component, string) error
		}); ok {
			if rollbackErr := store.RollbackVersion(ctx, previous, in.Version); rollbackErr != nil {
				return errors.Join(cause, rollbackErr)
			}
			// Reconcile any rows written before the index failure. The original
			// cause remains authoritative even when this best-effort sweep fails.
			_, _ = NewIndexer(s.repo, s.source.Root(), nil).IndexManifest(ctx, previous.ManifestPath)
		}
		return cause
	}
	if _, err := NewIndexer(s.repo, s.source.Root(), nil).IndexManifest(ctx, c.ManifestPath); err != nil {
		return CreateComponentVersionResult{}, rollback(err)
	}
	c, err = s.Get(ctx, in.ComponentID)
	if err != nil {
		return CreateComponentVersionResult{}, rollback(err)
	}
	v, err := s.repo.GetVersion(ctx, c.ID, in.Version)
	if err != nil {
		return CreateComponentVersionResult{}, rollback(err)
	}
	if intent == VersionIntentRelease && strings.Contains(strings.TrimSpace(in.FromVersion), "-") {
		if store, ok := s.source.(interface {
			RemoveVersion(context.Context, Component, string) error
		}); ok {
			if err := store.RemoveVersion(ctx, previous, in.FromVersion); err != nil {
				return CreateComponentVersionResult{}, rollback(err)
			}
		}
	}
	return CreateComponentVersionResult{Component: c, Version: v, SourcePath: sourcePath}, nil
}

func (s *service) UpdateComponentManifest(ctx context.Context, in UpdateComponentManifestInput) (Component, error) {
	if s.source == nil {
		return Component{}, errNoSourceStore
	}
	c, err := s.Get(ctx, in.ComponentID)
	if err != nil {
		var notFound ErrComponentNotFound
		if !errors.As(err, &notFound) {
			return Component{}, err
		}
		// A malformed version pointer can make a manifest intentionally fail
		// indexing. Manifest repair must still remain available through the
		// governed command surface, otherwise the only recovery path is a direct
		// catalog-file edit. Resolve the authored identity from disk for this
		// narrow not-indexed case; the normal index below remains the authority
		// that accepts or rejects the repaired manifest.
		c, err = s.resolveAuthoredManifestComponent(in.ComponentID)
		if err != nil {
			return Component{}, err
		}
	}
	metadataOnly := strings.TrimSpace(in.LatestVersion) == "" && strings.TrimSpace(in.DraftVersion) == ""
	in.PreserveVersionPointers = metadataOnly
	// The durable ledger, not a stale manifest marker, owns cold-tier truth.
	// This also makes a metadata-only governed update recover an authored asset
	// whose registry projection was lost: unverifiable evicted markers are
	// cleared instead of permanently preventing the asset from re-indexing.
	in.ReconcileEvictedVersions = true
	if strings.TrimSpace(c.ID) != "" {
		versions, listErr := s.repo.ListVersions(ctx, c.ID, 100000)
		if listErr != nil {
			return Component{}, listErr
		}
		for _, version := range versions {
			if version.Presence == "evicted" {
				in.EvictedVersions = append(in.EvictedVersions, version.Version)
			}
		}
	}
	if err := s.source.UpdateManifest(ctx, c, in); err != nil {
		return Component{}, err
	}
	if metadataOnly {
		headers := make(map[string]string, len(c.Headers)+1)
		for key, value := range c.Headers {
			headers[key] = value
		}
		catalogID := c.CatalogID
		if value := strings.TrimSpace(in.CatalogID); value != "" {
			catalogID = value
		}
		if in.ClearCatalogID {
			catalogID = ""
			delete(headers, "catalogId")
		} else if catalogID != "" {
			headers["catalogId"] = catalogID
		}
		displayName := firstNonEmpty(strings.TrimSpace(in.DisplayName), c.DisplayName, c.Slug)
		description := firstNonEmpty(strings.TrimSpace(in.Description), c.Description)
		tags := append([]string(nil), c.Tags...)
		if in.Tags != nil {
			tags = cleanTags(in.Tags)
		}
		dependencies := append([]AssetDependency(nil), c.Dependencies...)
		if in.Dependencies != nil {
			dependencies = append([]AssetDependency(nil), in.Dependencies...)
		}
		return s.repo.Upsert(ctx, UpsertInput{
			CatalogID: catalogID, LibraryID: c.LibraryID, Slug: c.Slug,
			DisplayName: displayName, Description: description, Slot: c.Slot, Category: c.Category,
			ManifestPath: c.ManifestPath, SourcePath: c.SourcePath, Version: c.Version,
			LatestVersion: c.LatestVersion, DraftVersion: c.DraftVersion, Tags: tags, Headers: headers,
			DesignStyles: c.DesignStyles, AssetKind: c.AssetKind, Dependencies: dependencies,
		})
	}
	if _, err := NewIndexer(s.repo, s.source.Root(), nil).IndexManifest(ctx, c.ManifestPath); err != nil {
		return Component{}, err
	}
	return s.Get(ctx, in.ComponentID)
}

func (s *service) resolveAuthoredManifestComponent(componentID string) (Component, error) {
	wanted := strings.TrimSpace(componentID)
	if wanted == "" {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: componentID}
	}
	type authoredManifest struct {
		CatalogID   string   `json:"catalogId"`
		LibraryID   string   `json:"libraryId"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Slot        string   `json:"slot"`
		Category    string   `json:"category"`
		Latest      string   `json:"latest"`
		Draft       string   `json:"draft"`
		Tags        []string `json:"tags"`
		AssetKind   string   `json:"assetKind"`
	}

	root := s.source.Root()
	var found *Component
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "versions" || entry.Name() == "node_modules" || strings.HasPrefix(entry.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "component.json" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var manifest authoredManifest
		if decodeErr := json.Unmarshal(raw, &manifest); decodeErr != nil || strings.TrimSpace(manifest.LibraryID) != wanted {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		manifestPath := filepath.ToSlash(relative)
		assetKind, kindErr := assetKindForManifestPath(manifestPath, manifest.AssetKind)
		if kindErr != nil {
			return kindErr
		}
		if found != nil {
			return ErrInvalidHeader{SourcePath: manifestPath, Field: "libraryId", Reason: "duplicates another authored manifest"}
		}
		component := Component{
			CatalogID:     strings.TrimSpace(manifest.CatalogID),
			LibraryID:     strings.TrimSpace(manifest.LibraryID),
			Slug:          filepath.Base(filepath.Dir(path)),
			DisplayName:   strings.TrimSpace(manifest.DisplayName),
			Description:   strings.TrimSpace(manifest.Description),
			Slot:          strings.TrimSpace(manifest.Slot),
			Category:      strings.TrimSpace(manifest.Category),
			LatestVersion: strings.TrimSpace(manifest.Latest),
			DraftVersion:  strings.TrimSpace(manifest.Draft),
			ManifestPath:  manifestPath,
			Tags:          append([]string(nil), manifest.Tags...),
			AssetKind:     assetKind,
		}
		found = &component
		return nil
	})
	if err != nil {
		return Component{}, err
	}
	if found == nil {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: componentID}
	}
	return *found, nil
}

// errNoContentStore signals that the service was constructed without a
// ContentStore. Surfaces as Internal at the transport edge — it's a
// wiring bug, not a caller error.
var (
	errNoContentStore = contentStoreUnconfiguredError{}
	errNoSourceStore  = sourceStoreUnconfiguredError{}
)

type contentStoreUnconfiguredError struct{}

func (contentStoreUnconfiguredError) Error() string {
	return "components service: ContentStore not configured"
}

type sourceStoreUnconfiguredError struct{}

func (sourceStoreUnconfiguredError) Error() string {
	return "components service: SourceStore not configured"
}
