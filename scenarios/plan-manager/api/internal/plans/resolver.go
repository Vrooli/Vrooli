package plans

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"plan-manager/internal/planmodel"
)

// SourceReader is the filesystem seam for the plan-source resolver. It reads
// markdown plan sources from the hygiene-blessed fallback read locations
// (~/.vrooli/plans, <repo>/docs/plans, <repo>/plans). Production wires the
// os-backed reader; tests inject a fake. Kept narrow so the domain never imports
// path/filesystem concerns beyond a byte read.
type SourceReader interface {
	ReadFile(path string) ([]byte, error)
}

// OSSourceReader is the production SourceReader (reads from disk).
type OSSourceReader struct{}

// ReadFile reads the named file from disk.
func (OSSourceReader) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

var _ SourceReader = OSSourceReader{}

// FallbackReadLocations are the ordered fallback read/import locations the
// resolver treats as valid plan sources (the canonical write location is the
// ~/.vrooli home store owned by the repository). These are relative hints; the
// import path accepts an explicit source path. The order encodes precedence.
var FallbackReadLocations = []string{
	"~/.vrooli/plans",
	"docs/plans",
	"plans",
}

const fallbackIndexFilename = "_index.json"

// Import adopts a markdown plan (from one of the fallback read locations) into
// the structured model and persists it through Create. When markdown is empty,
// the source is read from sourcePath via the SourceReader seam. References are
// parsed from the [CODE:]/[REQ:]/[DOC:] grammar. The fallback source is NOT
// mutated (non-destructive import — see docs/concepts/DATA.md).
func (s *service) Import(ctx context.Context, sourcePath, markdown string) (Plan, error) {
	if strings.TrimSpace(markdown) == "" {
		if strings.TrimSpace(sourcePath) == "" {
			return Plan{}, ErrInvalidPlan{Reason: "import requires markdown or a source path"}
		}
		if s.reader == nil {
			return Plan{}, ErrInvalidPlan{Reason: "no source reader configured; pass inline markdown"}
		}
		raw, err := s.reader.ReadFile(sourcePath)
		if err != nil {
			return Plan{}, fmt.Errorf("read plan source %q: %w", sourcePath, err)
		}
		markdown = string(raw)
	}
	parsed, err := ParsePlanMarkdown(markdown)
	if err != nil {
		return Plan{}, err
	}
	return s.Create(ctx, parsed)
}

// Migrate ensures a plan resolved from a fallback location is present in the
// canonical home store. If the plan is already canonical it is idempotently
// touched; otherwise the resolver reads the legacy ~/.vrooli/plans index (plus
// the documented fallback locations), parses the referenced markdown, and
// imports it as a structured plan. The fallback source is never destructively
// removed here.
func (s *service) Migrate(ctx context.Context, idOrSlug string) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug)
	if err == nil {
		p.UpdatedAt = s.now()
		if err := s.repo.Save(ctx, p); err != nil {
			return Plan{}, err
		}
		return p, nil
	}
	var notFound ErrPlanNotFound
	if !errors.As(err, &notFound) {
		return Plan{}, err
	}
	sourcePath, markdown, err := s.resolveFallbackPlanSource(strings.TrimSpace(idOrSlug))
	if err != nil {
		return Plan{}, err
	}
	return s.Import(ctx, sourcePath, markdown)
}

type fallbackIndex struct {
	Version int                  `json:"version"`
	Plans   []fallbackPlanRecord `json:"plans"`
}

type fallbackPlanRecord struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Path        string    `json:"path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Archived    bool      `json:"archived"`
	ContentHash string    `json:"content_hash"`
}

func (s *service) resolveFallbackPlanSource(idOrSlug string) (string, string, error) {
	if strings.TrimSpace(idOrSlug) == "" {
		return "", "", ErrInvalidPlan{Reason: "migrate requires a plan id or slug"}
	}
	if s.reader == nil {
		return "", "", ErrInvalidPlan{Reason: "no source reader configured; cannot read fallback plans"}
	}
	for _, location := range FallbackReadLocations {
		dir := expandPlanLocation(location)
		if sourcePath, markdown, ok, err := s.resolveFallbackFromIndex(dir, idOrSlug); ok || err != nil {
			return sourcePath, markdown, err
		}
		sourcePath := filepath.Join(dir, idOrSlug+".md")
		raw, err := s.reader.ReadFile(sourcePath)
		if err == nil {
			return sourcePath, string(raw), nil
		}
		if !os.IsNotExist(err) {
			continue
		}
	}
	return "", "", ErrPlanNotFound{ID: idOrSlug}
}

func (s *service) resolveFallbackFromIndex(dir, idOrSlug string) (string, string, bool, error) {
	indexPath := filepath.Join(dir, fallbackIndexFilename)
	raw, err := s.reader.ReadFile(indexPath)
	if err != nil {
		return "", "", false, nil
	}
	var idx fallbackIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return "", "", true, fmt.Errorf("decode fallback plan index %q: %w", indexPath, err)
	}
	for _, record := range idx.Plans {
		if record.Archived {
			continue
		}
		if record.ID != idOrSlug && record.Slug != idOrSlug {
			continue
		}
		sourcePath := strings.TrimSpace(record.Path)
		if sourcePath == "" {
			sourcePath = filepath.Join(dir, record.Slug+".md")
		}
		raw, err := s.reader.ReadFile(sourcePath)
		if err != nil {
			return "", "", true, fmt.Errorf("read fallback plan %q: %w", sourcePath, err)
		}
		return sourcePath, string(raw), true, nil
	}
	return "", "", false, nil
}

func expandPlanLocation(location string) string {
	if strings.HasPrefix(location, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, strings.TrimPrefix(location, "~/"))
		}
	}
	return filepath.Clean(location)
}

// ParsePlanMarkdown parses a markdown plan into the structured model.
func ParsePlanMarkdown(markdown string) (Plan, error) {
	return planmodel.ParsePlanMarkdown(markdown)
}
