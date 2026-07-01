package plans

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	configpkg "github.com/vrooli/vrooli/internal/config"
)

const indexFilename = "_index.json"

// MirrorFallbackReader is the read-only mirror projection used only when Plan
// Manager is unavailable. It must never create, mutate, archive, or import
// plans; canonical writes belong to Plan Manager.
type MirrorFallbackReader interface {
	List(ctx context.Context, includeArchived bool) ([]PlanRecord, error)
	Find(ctx context.Context, ref string) (PlanRecord, error)
	Read(ctx context.Context, ref string) (PlanRecord, string, error)
}

type OSMirrorFallbackReader struct {
	Home     string
	Now      func() time.Time
	ReadFile func(string) ([]byte, error)
}

func (r OSMirrorFallbackReader) List(_ context.Context, includeArchived bool) ([]PlanRecord, error) {
	idx, err := r.loadIndex()
	if err != nil {
		return nil, err
	}
	records := filterArchived(idx.Plans, includeArchived)
	sortRecords(records)
	return records, nil
}

func (r OSMirrorFallbackReader) Find(ctx context.Context, ref string) (PlanRecord, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return PlanRecord{}, fmt.Errorf("plan id or slug is required")
	}
	records, err := r.List(ctx, true)
	if err != nil {
		return PlanRecord{}, err
	}
	for _, record := range records {
		if record.ID == ref || record.Slug == ref {
			return record, nil
		}
	}
	return PlanRecord{}, fmt.Errorf("plan %q not found", ref)
}

func (r OSMirrorFallbackReader) Read(ctx context.Context, ref string) (PlanRecord, string, error) {
	record, err := r.Find(ctx, ref)
	if err != nil {
		return PlanRecord{}, "", err
	}
	content, err := r.readFile(record.Path)
	if err != nil {
		return PlanRecord{}, "", fmt.Errorf("read plan mirror: %w", err)
	}
	return record, string(content), nil
}

func (r OSMirrorFallbackReader) loadIndex() (indexFile, error) {
	path := filepath.Join(r.storageDir(), indexFilename)
	data, err := r.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r.scanMirrorDir()
		}
		return indexFile{}, fmt.Errorf("read plan mirror index: %w", err)
	}
	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return indexFile{}, fmt.Errorf("decode plan mirror index: %w", err)
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	if len(idx.Plans) == 0 {
		scanned, err := r.scanMirrorDir()
		if err == nil && len(scanned.Plans) > 0 {
			return scanned, nil
		}
	}
	sortRecords(idx.Plans)
	return idx, nil
}

func (r OSMirrorFallbackReader) scanMirrorDir() (indexFile, error) {
	dir := r.storageDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return indexFile{Version: 1}, nil
		}
		return indexFile{}, fmt.Errorf("scan plan mirror dir: %w", err)
	}
	now := r.now()
	records := make([]PlanRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		slug := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		content, _ := r.readFile(path)
		records = append(records, PlanRecord{
			ID:          slug,
			Title:       titleFromSlug(slug),
			Slug:        slug,
			Path:        path,
			CreatedAt:   now,
			UpdatedAt:   now,
			ContentHash: contentHash(string(content)),
		})
	}
	sortRecords(records)
	return indexFile{Version: 1, Plans: records}, nil
}

func (r OSMirrorFallbackReader) storageDir() string {
	home := strings.TrimSpace(r.Home)
	if home == "" {
		if resolved, err := configpkg.HomeDir(); err == nil {
			home = resolved
		}
	}
	dir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyPlans)
	if err != nil {
		return ""
	}
	return dir
}

func (r OSMirrorFallbackReader) readFile(path string) ([]byte, error) {
	if r.ReadFile != nil {
		return r.ReadFile(path)
	}
	return os.ReadFile(path)
}

func (r OSMirrorFallbackReader) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}
