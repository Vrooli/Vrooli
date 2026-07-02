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
	RendererVersion     = "plan-manager-renderer-v3"
	mirrorIndexFilename = "_index.json"
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
	idx := mirrorIndex{Version: 2}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &idx); err != nil {
			return fmt.Errorf("decode mirror index: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read mirror index: %w", err)
	}
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
