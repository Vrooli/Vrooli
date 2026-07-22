package components

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// defaultListLimit caps List rows when callers pass 0. Business
// policy, not transport policy — lives next to the only code that
// applies it.
const defaultListLimit = 200

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
	return s.repo.Get(ctx, id)
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
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return Content{}, err
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
	return s.repo.ListVersions(ctx, componentID, limit)
}

func (s *service) GetVersion(ctx context.Context, componentID, version string) (ComponentVersion, error) {
	return s.repo.GetVersion(ctx, componentID, version)
}

func (s *service) ListStories(ctx context.Context, q StoryQuery) ([]ComponentStory, error) {
	if q.Limit <= 0 {
		q.Limit = defaultListLimit
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

	component, err := s.repo.Get(ctx, cid)
	if err != nil {
		return StyleFitVerdict{}, err
	}
	if strings.TrimSpace(version) != "" {
		if _, err := s.repo.GetVersion(ctx, cid, strings.TrimSpace(version)); err != nil {
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
	v, err := s.repo.GetVersion(ctx, componentID, version)
	if err != nil {
		return Content{}, err
	}
	return Content{Body: v.Content, SourcePath: v.SourcePath, SHA256: v.ContentSHA256}, nil
}

func (s *service) UpdateContent(ctx context.Context, id string, in WriteContentInput) (Content, error) {
	return s.UpdateContentAt(ctx, id, "", in)
}

func (s *service) UpdateContentAt(ctx context.Context, id, path string, in WriteContentInput) (Content, error) {
	if s.content == nil {
		return Content{}, errNoContentStore
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return Content{}, err
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

func (s *service) CreateComponentVersion(ctx context.Context, in CreateComponentVersionInput) (CreateComponentVersionResult, error) {
	if s.source == nil {
		return CreateComponentVersionResult{}, errNoSourceStore
	}
	c, err := s.repo.Get(ctx, in.ComponentID)
	if err != nil {
		return CreateComponentVersionResult{}, err
	}
	intent := in.Intent
	if intent == "" && !strings.Contains(in.Version, "-") {
		intent = VersionIntentRelease
	}
	if intent == VersionIntentRelease {
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
	sourcePath, err := s.source.CreateVersion(ctx, c, in)
	if err != nil {
		return CreateComponentVersionResult{}, err
	}
	if _, err := NewIndexer(s.repo, s.source.Root(), nil).Run(ctx); err != nil {
		return CreateComponentVersionResult{}, err
	}
	c, err = s.repo.Get(ctx, in.ComponentID)
	if err != nil {
		return CreateComponentVersionResult{}, err
	}
	v, err := s.repo.GetVersion(ctx, c.ID, in.Version)
	if err != nil {
		return CreateComponentVersionResult{}, err
	}
	return CreateComponentVersionResult{Component: c, Version: v, SourcePath: sourcePath}, nil
}

func (s *service) UpdateComponentManifest(ctx context.Context, in UpdateComponentManifestInput) (Component, error) {
	if s.source == nil {
		return Component{}, errNoSourceStore
	}
	c, err := s.repo.Get(ctx, in.ComponentID)
	if err != nil {
		return Component{}, err
	}
	if err := s.source.UpdateManifest(ctx, c, in); err != nil {
		return Component{}, err
	}
	if _, err := NewIndexer(s.repo, s.source.Root(), nil).Run(ctx); err != nil {
		return Component{}, err
	}
	return s.repo.Get(ctx, in.ComponentID)
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
