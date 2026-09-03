package plans

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

const (
	// RendererVersion v2: the nine-cluster section model (Purpose/Problem/Outcome/
	// Approach & Decisions/Boundaries/Assumptions & Risks/Verification/Execution
	// Setup/Phases), compact NO_CONTEXT phase lines, decisions +
	// assumption/mitigation rendering. Bumping this marks every mirror rendered
	// by the previous version stale so reconcile re-renders it.
	//
	// v3: Execution Feedback now stamps the pre-filled completion-record
	// command (`swarm-manager records create ...` with scenario/title filled),
	// so end-of-plan agents paste instead of reconstructing it from memory.
	//
	// v4: Execution Feedback spells out concrete log add commands, phase ordinal
	// support, current-phase inference, and the log reassign recovery command.
	//
	// v5: Execution Feedback replaces compact command alternatives with a
	// concrete decision-add example and a short variant list.
	RendererVersion     = "plan-manager-renderer-v6"
	mirrorIndexFilename = "_index.json"
	// mirrorIndexVersion is the schema version this build writes. It is stamped
	// after decoding so an older on-disk version is upgraded in place rather
	// than pinned forever by the decode.
	mirrorIndexVersion = 2
)

// RenderResult is the read model for a rendered markdown request.
type RenderResult struct {
	Markdown        string
	Mirror          RenderedPlanMirror
	Repaired        bool
	Plan            Plan
	QualityStatus   string
	QualityFindings []string
}

// MirrorStore owns the durable file projection of canonical structured plans.
// The file is read/repair infrastructure; SQLite remains the source of truth.
type MirrorStore interface {
	PathFor(ctx context.Context, p Plan) (RenderedPlanMirror, error)
	Read(ctx context.Context, p Plan) ([]byte, RenderedPlanMirror, error)
	Publish(ctx context.Context, p Plan, markdown []byte, renderedAt string) (RenderedPlanMirror, error)
}

type noMirrorStore struct{}

func (noMirrorStore) PathFor(context.Context, Plan) (RenderedPlanMirror, error) {
	return RenderedPlanMirror{Status: RenderedMirrorStatusUnknown, RenderVersion: RendererVersion}, nil
}

func (noMirrorStore) Read(context.Context, Plan) ([]byte, RenderedPlanMirror, error) {
	return nil, RenderedPlanMirror{Status: RenderedMirrorStatusUnknown, RenderVersion: RendererVersion}, errMirrorUnavailable
}

func (noMirrorStore) Publish(context.Context, Plan, []byte, string) (RenderedPlanMirror, error) {
	return RenderedPlanMirror{Status: RenderedMirrorStatusUnknown, RenderVersion: RendererVersion}, nil
}

var errMirrorUnavailable = errors.New("plan mirror store is not configured")

// OSMirrorStore writes rendered markdown under the repo-contract runtime-home
// plans entry. Home is the resolved OS home dir, not the runtime-home root.
type OSMirrorStore struct {
	Home string
}

func NewOSMirrorStore(home string) OSMirrorStore {
	return OSMirrorStore{Home: strings.TrimSpace(home)}
}

func NewDefaultOSMirrorStore() OSMirrorStore {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		if resolved, err := os.UserHomeDir(); err == nil {
			home = resolved
		}
	}
	return NewOSMirrorStore(home)
}

func (s OSMirrorStore) PathFor(_ context.Context, p Plan) (RenderedPlanMirror, error) {
	root, err := s.root()
	if err != nil {
		return RenderedPlanMirror{}, err
	}
	name := slugify(p.Slug)
	if name == "" {
		name = slugify(p.Title)
	}
	if name == "" {
		name = strings.TrimSpace(p.ID)
	}
	if name == "" {
		return RenderedPlanMirror{}, ErrInvalidPlan{Reason: "plan slug or id is required for mirror path"}
	}
	rel := name + ".md"
	return RenderedPlanMirror{
		Path:          filepath.Join(root, rel),
		RelativePath:  rel,
		RenderVersion: RendererVersion,
		Status:        RenderedMirrorStatusUnknown,
	}, nil
}

func (s OSMirrorStore) Read(ctx context.Context, p Plan) ([]byte, RenderedPlanMirror, error) {
	meta, err := s.PathFor(ctx, p)
	if err != nil {
		return nil, meta, err
	}
	data, err := os.ReadFile(meta.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			meta.Status = RenderedMirrorStatusMissing
			return nil, meta, err
		}
		meta.Status = RenderedMirrorStatusUnknown
		meta.LastError = err.Error()
		return nil, meta, err
	}
	meta.ContentHash = renderedContentHash(data)
	meta.RenderedAt = p.Mirror.RenderedAt
	expectedHash := strings.TrimSpace(p.Mirror.ContentHash)
	if expectedHash != "" && expectedHash == meta.ContentHash && p.Mirror.RenderVersion == RendererVersion {
		meta.Status = RenderedMirrorStatusFresh
		return data, meta, nil
	}
	meta.Status = RenderedMirrorStatusStale
	return data, meta, nil
}

func (s OSMirrorStore) Publish(ctx context.Context, p Plan, markdown []byte, renderedAt string) (RenderedPlanMirror, error) {
	meta, err := s.PathFor(ctx, p)
	if err != nil {
		return meta, err
	}
	if err := os.MkdirAll(filepath.Dir(meta.Path), 0o755); err != nil {
		meta.Status = RenderedMirrorStatusWriteFailed
		meta.LastError = err.Error()
		return meta, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(meta.Path), "."+filepath.Base(meta.Path)+".tmp-*")
	if err != nil {
		meta.Status = RenderedMirrorStatusWriteFailed
		meta.LastError = err.Error()
		return meta, err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(markdown); err != nil {
		_ = tmp.Close()
		meta.Status = RenderedMirrorStatusWriteFailed
		meta.LastError = err.Error()
		return meta, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		meta.Status = RenderedMirrorStatusWriteFailed
		meta.LastError = err.Error()
		return meta, err
	}
	if err := tmp.Close(); err != nil {
		meta.Status = RenderedMirrorStatusWriteFailed
		meta.LastError = err.Error()
		return meta, err
	}
	if err := os.Rename(tmpPath, meta.Path); err != nil {
		meta.Status = RenderedMirrorStatusWriteFailed
		meta.LastError = err.Error()
		return meta, err
	}
	cleanup = false
	_ = fsyncDir(filepath.Dir(meta.Path))
	meta.ContentHash = renderedContentHash(markdown)
	meta.RenderVersion = RendererVersion
	meta.RenderedAt = renderedAt
	meta.Status = RenderedMirrorStatusFresh
	if err := s.upsertIndex(p, meta); err != nil {
		meta.Status = RenderedMirrorStatusWriteFailed
		meta.LastError = err.Error()
		return meta, err
	}
	return meta, nil
}

func (s OSMirrorStore) upsertIndex(p Plan, meta RenderedPlanMirror) error {
	root, err := s.root()
	if err != nil {
		return err
	}
	path := filepath.Join(root, mirrorIndexFilename)
	idx := mirrorIndex{Version: mirrorIndexVersion}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &idx); err != nil {
			return fmt.Errorf("decode mirror index: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read mirror index: %w", err)
	}
	// Decoding overwrites Version with whatever the file carried, so an index
	// first written at v1 would report v1 forever and any version-keyed
	// migration would never fire. Restamp it after the decode.
	idx.Version = mirrorIndexVersion
	record := mirrorIndexPlanRecord{
		ID:            p.ID,
		Title:         p.Title,
		Slug:          p.Slug,
		Path:          meta.Path,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		Archived:      p.Status == PlanStatusArchived,
		ContentHash:   meta.ContentHash,
		WorkspaceID:   p.WorkspaceID,
		WorkspaceRoot: p.WorkspaceRoot,
	}
	replaced := false
	for i, existing := range idx.Plans {
		if existing.ID == p.ID || (existing.ID == "" && existing.Slug == p.Slug) {
			idx.Plans[i] = record
			replaced = true
			break
		}
	}
	if !replaced {
		idx.Plans = append(idx.Plans, record)
	}
	return s.writeIndex(root, path, idx)
}

// writeIndex atomically replaces the mirror index. Extracted so upsertIndex and
// PruneIndex share one durable-write path rather than two copies that can drift.
func (s OSMirrorStore) writeIndex(root, path string, idx mirrorIndex) error {
	raw, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("encode mirror index: %w", err)
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(root, "."+mirrorIndexFilename+".tmp-*")
	if err != nil {
		return fmt.Errorf("create mirror index temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write mirror index temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync mirror index temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close mirror index temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace mirror index: %w", err)
	}
	cleanup = false
	return fsyncDir(root)
}

type mirrorIndex struct {
	Version int                     `json:"version"`
	Plans   []mirrorIndexPlanRecord `json:"plans"`
}

type mirrorIndexPlanRecord struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Slug          string `json:"slug"`
	Path          string `json:"path"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
	Archived      bool   `json:"archived,omitempty"`
	ContentHash   string `json:"content_hash,omitempty"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}

func (s OSMirrorStore) root() (string, error) {
	if strings.TrimSpace(s.Home) == "" {
		return "", fmt.Errorf("resolve plan mirror root: home directory is empty")
	}
	return repocontract.RuntimeHomeEntryPath(s.Home, repocontract.HomeKeyPlans)
}

func renderedContentHash(markdown []byte) string {
	sum := sha256.Sum256(markdown)
	return hex.EncodeToString(sum[:])
}

func fsyncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// mirrorMarkerRe matches the self-identifying comment stamped at the top of
// every rendered mirror by renderHeader.
var mirrorMarkerRe = regexp.MustCompile(`<!--\s*plan-manager:mirror\s+id=([^\s]+)(?:\s+slug=([^\s]*))?\s*-->`)

// FormatMirrorMarker renders the marker comment for a plan identity.
func FormatMirrorMarker(id, slug string) string {
	return fmt.Sprintf("<!-- plan-manager:mirror id=%s slug=%s -->", id, slug)
}

// ParseMirrorMarker reports whether markdown is a mirror this system rendered,
// and returns the plan identity it was rendered from. A document without the
// marker is either a hand-authored plan or a mirror written before the marker
// existed; both are treated as un-marked, so callers must keep their existing
// database-keyed guards rather than relying on this alone.
func ParseMirrorMarker(markdown string) (id string, slug string, ok bool) {
	m := mirrorMarkerRe.FindStringSubmatch(markdown)
	if m == nil {
		return "", "", false
	}
	return strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), true
}

// MirrorIndexPruner is implemented by mirror stores that keep a durable index
// beside the rendered files. It is an optional interface rather than a method
// on MirrorStore so stores that hold no index (and test fakes) are unaffected.
type MirrorIndexPruner interface {
	PruneIndex() (MirrorIndexPruneResult, error)
}

// MirrorIndexPruneResult reports what a prune removed.
type MirrorIndexPruneResult struct {
	Before      int
	After       int
	MissingFile int
	DuplicateID int
}

// PruneIndex drops index records that can no longer describe anything: records
// whose rendered file is gone, and duplicate records for one plan id.
//
// The index had no removal path at all. upsertIndex only ever appends or
// replaces in place, so a rendered file that is deleted, or a plan re-rendered
// under a second identity, leaves its record behind forever. Because every plan
// write rewrites the whole index, that dead weight is also a direct per-write
// cost, not just wasted bytes.
//
// This is deliberately keyed on the filesystem rather than on the database: an
// index entry whose file exists is kept even when the plan row is missing,
// because that is exactly the orphaned-mirror case an operator may still want
// to recover from.
func (s OSMirrorStore) PruneIndex() (MirrorIndexPruneResult, error) {
	var res MirrorIndexPruneResult
	root, err := s.root()
	if err != nil {
		return res, err
	}
	path := filepath.Join(root, mirrorIndexFilename)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return res, nil
		}
		return res, fmt.Errorf("read mirror index: %w", err)
	}
	var idx mirrorIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return res, fmt.Errorf("decode mirror index: %w", err)
	}
	res.Before = len(idx.Plans)
	idx.Version = mirrorIndexVersion

	seen := make(map[string]int, len(idx.Plans))
	kept := make([]mirrorIndexPlanRecord, 0, len(idx.Plans))
	for _, record := range idx.Plans {
		recordPath := strings.TrimSpace(record.Path)
		if recordPath == "" {
			recordPath = filepath.Join(root, record.Slug+".md")
		}
		if _, statErr := os.Stat(recordPath); statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				res.MissingFile++
				continue
			}
			return res, fmt.Errorf("stat indexed mirror %q: %w", recordPath, statErr)
		}
		key := strings.TrimSpace(record.ID)
		if key == "" {
			key = "slug:" + record.Slug
		}
		if at, dup := seen[key]; dup {
			// Keep the later record; upsert semantics already treat the most
			// recent write as authoritative.
			kept[at] = record
			res.DuplicateID++
			continue
		}
		seen[key] = len(kept)
		kept = append(kept, record)
	}
	idx.Plans = kept
	res.After = len(kept)
	if res.MissingFile == 0 && res.DuplicateID == 0 {
		return res, nil
	}
	return res, s.writeIndex(root, path, idx)
}
