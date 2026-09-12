// Package availability resolves exact published library implementations. It
// combines the canonical version graph with the existing preview compiler;
// catalog declarations and version strings never stand in for build evidence.
package availability

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"react-component-library/internal/components"
)

type State string

const (
	Built       State = "built"
	Declared    State = "declared"
	Draft       State = "draft"
	Retired     State = "retired"
	Invalid     State = "invalid"
	Unavailable State = "unavailable"
)

type Result struct {
	CatalogID       string `json:"catalogId"`
	LibraryID       string `json:"libraryId,omitempty"`
	Version         string `json:"version,omitempty"`
	State           State  `json:"state"`
	ReasonCode      string `json:"reasonCode"`
	Reason          string `json:"reason,omitempty"`
	BuildHash       string `json:"buildHash,omitempty"`
	SourceHash      string `json:"sourceHash,omitempty"`
	DependencyCount int    `json:"dependencyCount"`
}

func (r Result) IsBuilt() bool { return r.State == Built }

type Catalog interface {
	components.DependencyReader
	List(context.Context, components.SearchQuery) ([]components.Component, error)
}

// Compile uses the existing version-pinned preview path. It must reject
// unresolved imports/exports and return an identity for the compiled artifact.
type Compile func(context.Context, string, string) (string, error)

// Snapshot is scoped to one verification or refresh, not retained across
// changes in lifecycle state. Each immutable version is read at most once.
type Snapshot struct {
	mu        sync.Mutex
	reader    Catalog
	compile   Compile
	assets    map[string]components.Component
	ambiguous map[string]bool
	versions  map[string]versionResult
	resolved  map[string]Result
}
type versionResult struct {
	value components.ComponentVersion
	err   error
}

func NewSnapshot(ctx context.Context, reader Catalog, compile Compile) (*Snapshot, error) {
	if reader == nil {
		return nil, fmt.Errorf("canonical asset catalog is unavailable")
	}
	assets, err := reader.List(ctx, components.SearchQuery{Limit: 100000})
	if err != nil {
		return nil, err
	}
	snapshot := &Snapshot{reader: reader, compile: compile, assets: map[string]components.Component{}, ambiguous: map[string]bool{}, versions: map[string]versionResult{}, resolved: map[string]Result{}}
	for _, asset := range assets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, id := range []string{asset.ID, asset.CatalogID, asset.LibraryID} {
			if id == "" {
				continue
			}
			if old, ok := snapshot.assets[id]; ok && old.ID != asset.ID {
				snapshot.ambiguous[id] = true
			}
			snapshot.assets[id] = asset
		}
	}
	return snapshot, nil
}

func (s *Snapshot) Get(ctx context.Context, id string) (components.Component, error) {
	if err := ctx.Err(); err != nil {
		return components.Component{}, err
	}
	if s.ambiguous[id] {
		return components.Component{}, fmt.Errorf("ambiguous asset identity %s", id)
	}
	asset, ok := s.assets[id]
	if !ok {
		return components.Component{}, fmt.Errorf("asset identity %s is not indexed", id)
	}
	return asset, nil
}
func (s *Snapshot) GetByLibraryID(ctx context.Context, id string) (components.Component, error) {
	return s.Get(ctx, id)
}
func (s *Snapshot) GetVersion(ctx context.Context, id, version string) (components.ComponentVersion, error) {
	if err := ctx.Err(); err != nil {
		return components.ComponentVersion{}, err
	}
	key := id + "@" + version
	if cached, ok := s.versions[key]; ok {
		return cached.value, cached.err
	}
	value, err := s.reader.GetVersion(ctx, id, version)
	if err == nil && (value.Version != version || value.ComponentID != id) {
		err = fmt.Errorf("version record identity differs from requested %s@%s", id, version)
	}
	if ctx.Err() == nil {
		s.versions[key] = versionResult{value, err}
	}
	return value, err
}

func (s *Snapshot) Resolve(ctx context.Context, catalogID, version string) Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := Result{CatalogID: catalogID, Version: version, State: Unavailable}
	fail := func(state State, code, reason string) Result {
		result.State = state
		result.ReasonCode = code
		result.Reason = reason
		return result
	}
	if err := ctx.Err(); err != nil {
		return fail(Unavailable, "cancelled", err.Error())
	}
	key := catalogID + "@" + version
	if cached, ok := s.resolved[key]; ok {
		return cached
	}
	asset, err := s.Get(ctx, catalogID)
	if err != nil {
		return fail(Unavailable, "asset_unresolved", err.Error())
	}
	// A catalog selection is never matched by a similarly named implementation.
	if asset.CatalogID != catalogID {
		return fail(Invalid, "catalog_identity_mismatch", "selected identity is not the implementation's catalog ID")
	}
	result.LibraryID = asset.LibraryID
	if version == "" {
		return fail(Declared, "version_not_selected", "select an exact published version")
	}
	closure, err := components.ResolveDependencyClosure(ctx, s, asset.ID, version)
	if err != nil {
		return fail(Unavailable, "dependency_unresolved", err.Error())
	}
	for _, entry := range closure {
		v := entry.Version
		if strings.TrimSpace(v.Version) == "" || v.ComponentID != entry.Asset.ID {
			return fail(Invalid, "version_identity_mismatch", "version record does not identify its selected component")
		}
		if v.Status == components.VersionStatusDraft {
			return fail(Draft, "unpublished_version", entry.Asset.LibraryID+"@"+v.Version+" is a draft")
		}
		if string(v.Status) == "retired" {
			return fail(Retired, "retired_version", entry.Asset.LibraryID+"@"+v.Version+" is retired")
		}
		if v.Status != components.VersionStatusReleased && v.Status != components.VersionStatusDeprecated && v.Status != components.VersionStatusArchived {
			return fail(Invalid, "unknown_lifecycle", "version has no recognized published lifecycle state")
		}
		if !v.DependencyLockPresent {
			return fail(Invalid, "dependency_lock_missing", "exact version lacks its source-derived dependency lock")
		}
		if v.SourcePath == "" || v.Content == "" || v.ContentSHA256 == "" {
			return fail(Unavailable, "source_missing", "published entry source or its digest is missing")
		}
		digest := sha256.Sum256([]byte(v.Content))
		if hex.EncodeToString(digest[:]) != v.ContentSHA256 {
			return fail(Invalid, "source_hash_mismatch", "published entry bytes differ from their recorded digest")
		}
		for _, file := range v.Files {
			digest := sha256.Sum256([]byte(file.Content))
			if file.Path == "" || file.ContentSHA256 == "" || hex.EncodeToString(digest[:]) != file.ContentSHA256 {
				return fail(Invalid, "source_hash_mismatch", "published companion bytes differ from their recorded digest")
			}
		}
		if entry.Asset.ID == asset.ID {
			if v.Version != version {
				return fail(Invalid, "version_identity_mismatch", "resolved root version differs from the selected version")
			}
			result.SourceHash = v.ContentSHA256
		}
	}
	if s.compile == nil {
		return fail(Unavailable, "compiler_unavailable", "published source has no build/export evidence")
	}
	artifact, err := s.compile(ctx, asset.ID, version)
	if err != nil {
		return fail(Unavailable, "build_failed", err.Error())
	}
	if artifact == "" {
		return fail(Unavailable, "build_evidence_missing", "compiler returned no artifact identity")
	}
	result.BuildHash = artifact
	result.State = Built
	result.ReasonCode = "published_build_resolved"
	result.DependencyCount = len(closure) - 1
	s.resolved[key] = result
	return result
}

func (s *Snapshot) Latest(catalogID string) string {
	asset, ok := s.assets[catalogID]
	if !ok || s.ambiguous[catalogID] || asset.CatalogID != catalogID {
		return ""
	}
	return asset.LatestVersion
}
