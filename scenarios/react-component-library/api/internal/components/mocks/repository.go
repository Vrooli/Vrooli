// Package mocks holds components-domain test fakes co-located with
// the domain they double for. Deleting the domain folder takes its
// mocks with it; package graph reflects ownership (mocks imports
// components; components does not import mocks).
package mocks

import (
	"context"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"react-component-library/internal/components"
)

// FakeRepository satisfies components.Repository for service and
// indexer tests that don't want the sqlite round-trip. In-memory map
// keyed by ID with a parallel libraryID → id index.
//
// Per-method error knobs (UpsertErr, GetErr, …) let tests drive the
// failure paths without faking sqlite. Atomic call counters keep
// -race quiet under fan-out.
type FakeRepository struct {
	mu          sync.Mutex
	items       map[string]components.Component // by ID
	versions    map[string]map[string]components.ComponentVersion
	stories     map[string][]components.ComponentStory
	libToID     map[string]string // library_id → id
	UpsertErr   error
	GetErr      error
	ListErr     error
	DeleteErr   error
	UpsertCalls atomic.Int64
	GetCalls    atomic.Int64
	ListCalls   atomic.Int64
	DeleteCalls atomic.Int64
	NowFn       func() time.Time
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		items:    map[string]components.Component{},
		versions: map[string]map[string]components.ComponentVersion{},
		stories:  map[string][]components.ComponentStory{},
		libToID:  map[string]string{},
		NowFn:    func() time.Time { return time.Now().UTC() },
	}
}

var _ components.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) Upsert(ctx context.Context, in components.UpsertInput) (components.Component, error) {
	f.UpsertCalls.Add(1)
	if f.UpsertErr != nil {
		return components.Component{}, f.UpsertErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.NowFn()
	id, existed := f.libToID[in.LibraryID]
	indexedAt := now
	if existed {
		indexedAt = f.items[id].IndexedAt
	} else {
		id = uuid.NewString()
		f.libToID[in.LibraryID] = id
	}
	c := components.Component{
		ID:            id,
		LibraryID:     in.LibraryID,
		Slug:          in.Slug,
		DisplayName:   in.DisplayName,
		Description:   in.Description,
		Slot:          in.Slot,
		Category:      firstNonEmpty(in.Category, in.Headers["category"]),
		SourcePath:    in.SourcePath,
		Version:       in.Version,
		LatestVersion: in.LatestVersion,
		DraftVersion:  in.DraftVersion,
		ManifestPath:  in.ManifestPath,
		Tags:          append([]string(nil), in.Tags...),
		IndexedAt:     indexedAt,
		UpdatedAt:     now,
		Headers:       copyHeaders(in.Headers),
		DesignStyles:  append([]components.ComponentDesignAffinity(nil), in.DesignStyles...),
		AssetKind:     in.AssetKind,
		Dependencies:  append([]components.AssetDependency(nil), in.Dependencies...),
	}
	f.items[id] = c
	return c, nil
}

func (f *FakeRepository) UpsertManifest(ctx context.Context, in components.IndexManifestInput) (components.Component, error) {
	c, err := f.Upsert(ctx, components.UpsertInput{
		LibraryID: in.Manifest.LibraryID, Slug: in.Manifest.Slug, DisplayName: in.Manifest.DisplayName,
		Description: in.Manifest.Description, Slot: in.Manifest.Slot, Category: in.Manifest.Category, ManifestPath: in.Manifest.ManifestPath,
		Version: in.Manifest.LatestVersion, LatestVersion: in.Manifest.LatestVersion, DraftVersion: in.Manifest.DraftVersion,
		Tags: in.Manifest.Tags, Headers: in.Headers, DesignStyles: in.Manifest.DesignStyles,
		AssetKind: in.Manifest.AssetKind, Dependencies: in.Manifest.Dependencies,
	})
	if err != nil {
		return components.Component{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.versions[c.ID] = map[string]components.ComponentVersion{}
	f.stories[c.ID] = nil
	for _, v := range in.Versions {
		if v.ID == "" {
			v.ID = uuid.NewString()
		}
		v.ComponentID = c.ID
		v.LibraryID = c.LibraryID
		f.versions[c.ID][v.Version] = v
		if v.Version == c.LatestVersion {
			c.SourcePath = v.SourcePath
			f.items[c.ID] = c
		}
	}
	for _, story := range in.Stories {
		if story.ID == "" {
			story.ID = uuid.NewString()
		}
		story.ComponentID = c.ID
		story.LibraryID = c.LibraryID
		f.stories[c.ID] = append(f.stories[c.ID], story)
	}
	return c, nil
}

func (f *FakeRepository) Get(ctx context.Context, id string) (components.Component, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return components.Component{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.items[id]
	if !ok {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	return c, nil
}

func (f *FakeRepository) GetByLibraryID(ctx context.Context, libraryID string) (components.Component, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return components.Component{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.libToID[libraryID]
	if !ok {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: libraryID}
	}
	return f.items[id], nil
}

func (f *FakeRepository) RestoreEvictedStories(context.Context) (int, error) { return 0, nil }

func (f *FakeRepository) List(ctx context.Context, q components.SearchQuery) ([]components.Component, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		return nil, nil
	}
	matchL := strings.ToLower(strings.TrimSpace(q.Match))
	tagL := strings.ToLower(strings.TrimSpace(q.Tag))
	categoryL := strings.ToLower(strings.TrimSpace(q.Category))
	styleID := strings.ToLower(strings.TrimSpace(q.StyleID))
	affinity := strings.ToLower(strings.TrimSpace(q.Affinity))
	multiTags := make([]string, 0, len(q.Tags))
	for _, t := range q.Tags {
		trimmed := strings.ToLower(strings.TrimSpace(t))
		if trimmed == "" {
			continue
		}
		multiTags = append(multiTags, trimmed)
	}
	var out []components.Component
	for _, c := range f.items {
		if q.AssetKind.Valid() && c.AssetKind != q.AssetKind {
			continue
		}
		if !q.AssetKind.Valid() && c.AssetKind == components.AssetKindFoundation {
			continue
		}
		if matchL != "" {
			hay := strings.ToLower(c.LibraryID + " " + c.DisplayName + " " + c.Description + " " + c.Slot + " " + c.SourcePath)
			if !strings.Contains(hay, matchL) {
				continue
			}
		}
		if tagL != "" {
			hit := false
			for _, t := range c.Tags {
				if strings.EqualFold(t, tagL) {
					hit = true
					break
				}
			}
			if !hit {
				continue
			}
		}
		if len(multiTags) > 0 {
			hit := false
			for _, want := range multiTags {
				for _, t := range c.Tags {
					if strings.EqualFold(t, want) {
						hit = true
						break
					}
				}
				if hit {
					break
				}
			}
			if !hit {
				continue
			}
		}
		if categoryL != "" {
			if !strings.EqualFold(c.Category, categoryL) {
				continue
			}
		}
		if styleID != "" || affinity != "" {
			hit := false
			for _, got := range c.DesignStyles {
				if styleID != "" && !strings.EqualFold(got.StyleID, styleID) {
					continue
				}
				if affinity != "" && !strings.EqualFold(string(got.Affinity), affinity) {
					continue
				}
				hit = true
				break
			}
			if !hit {
				continue
			}
		}
		out = append(out, c)
	}
	matchSet := matchL != ""
	// Mirrors sqlite ORDER BY: match-mode → display_name COLLATE NOCASE;
	// otherwise newest indexed first, then library_id asc.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			less := false
			if matchSet {
				li, lj := strings.ToLower(out[i].DisplayName), strings.ToLower(out[j].DisplayName)
				if lj < li || (lj == li && out[j].LibraryID < out[i].LibraryID) {
					less = true
				}
			} else if out[j].IndexedAt.After(out[i].IndexedAt) ||
				(out[j].IndexedAt.Equal(out[i].IndexedAt) && out[j].LibraryID < out[i].LibraryID) {
				less = true
			}
			if less {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (f *FakeRepository) DeleteMissing(ctx context.Context, keep []string) (int, error) {
	f.DeleteCalls.Add(1)
	if f.DeleteErr != nil {
		return 0, f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	keepSet := map[string]struct{}{}
	for _, k := range keep {
		keepSet[k] = struct{}{}
	}
	deleted := 0
	for lib, id := range f.libToID {
		if _, ok := keepSet[lib]; !ok {
			delete(f.items, id)
			delete(f.libToID, lib)
			// Cascade: mirror the soft-FK cleanup the sqlite repo does
			// so deleting a registry row leaves no orphaned children.
			delete(f.versions, id)
			delete(f.stories, id)
			deleted++
		}
	}
	return deleted, nil
}

func (f *FakeRepository) SweepOrphans(ctx context.Context) ([]components.OrphanVersion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var orphans []components.OrphanVersion
	for cid, vers := range f.versions {
		if _, ok := f.items[cid]; ok {
			continue
		}
		for _, v := range vers {
			orphans = append(orphans, components.OrphanVersion{
				ComponentID: cid,
				LibraryID:   v.LibraryID,
				Version:     v.Version,
				SourcePath:  v.SourcePath,
			})
		}
		delete(f.versions, cid)
		delete(f.stories, cid)
	}
	sort.Slice(orphans, func(i, j int) bool {
		if orphans[i].LibraryID != orphans[j].LibraryID {
			return orphans[i].LibraryID < orphans[j].LibraryID
		}
		return orphans[i].Version < orphans[j].Version
	})
	return orphans, nil
}

func (f *FakeRepository) ListVersions(ctx context.Context, componentID string, limit int) ([]components.ComponentVersion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []components.ComponentVersion
	for _, v := range f.versions[componentID] {
		out = append(out, v)
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeRepository) GetVersion(ctx context.Context, componentID, version string) (components.ComponentVersion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.versions[componentID][version]; ok {
		return v, nil
	}
	return components.ComponentVersion{}, components.ErrComponentNotFound{IDOrLibraryID: componentID + "@" + version}
}

func (f *FakeRepository) SetVersionPresence(ctx context.Context, componentID, version, presence string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.versions[componentID][version]
	if !ok {
		return components.ErrComponentNotFound{IDOrLibraryID: componentID + "@" + version}
	}
	v.Presence = presence
	f.versions[componentID][version] = v
	return nil
}

func (f *FakeRepository) ListStories(ctx context.Context, q components.StoryQuery) ([]components.ComponentStory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	var out []components.ComponentStory
	for componentID, rows := range f.stories {
		if q.ComponentID != "" && q.ComponentID != componentID {
			continue
		}
		for _, story := range rows {
			if q.Version == "" || q.Version == story.Version {
				out = append(out, story)
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

func copyHeaders(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if isStructuredHeaderField(k) {
			continue
		}
		out[k] = v
	}
	return out
}

func isStructuredHeaderField(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "libraryid", "version", "deps", "category":
		return true
	default:
		return false
	}
}
