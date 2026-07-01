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

	repocontract "github.com/vrooli/repo-contract-go"
)

// SourceReader is the filesystem seam for the plan-source resolver. It reads
// markdown plan sources from the hygiene-blessed fallback read locations
// (~/.vrooli/plans, <repo>/docs/plans, <repo>/plans). Production wires the
// os-backed reader; tests inject a fake. Kept narrow so the domain never imports
// path/filesystem concerns beyond a byte read.
type SourceReader interface {
	ReadFile(path string) ([]byte, error)
	ListMarkdownFiles(dir string) ([]string, error)
}

// OSSourceReader is the production SourceReader (reads from disk).
type OSSourceReader struct{}

// ReadFile reads the named file from disk.
func (OSSourceReader) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

// ListMarkdownFiles returns direct child markdown files in dir, excluding the
// fallback projection index. Missing directories are treated as empty.
func (OSSourceReader) ListMarkdownFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == fallbackIndexFilename || !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out, nil
}

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
func (s *service) Import(ctx context.Context, sourcePath, markdown, title, slug string, workspace WorkspaceScope) (Plan, error) {
	sourcePath, err := resolveWorkspacePath(sourcePath, workspace)
	if err != nil {
		return Plan{}, err
	}
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
	if override := strings.TrimSpace(title); override != "" {
		parsed.Title = override
	}
	if override := slugify(slug); override != "" {
		parsed.Slug = override
	}
	// Record import provenance (adoption bookkeeping) unless the markdown already
	// carried it. Import is non-destructive: the source file is never mutated.
	if parsed.ImportProvenance == nil {
		parsed.ImportProvenance = &ImportProvenance{
			SourcePath:     strings.TrimSpace(sourcePath),
			ImportedAt:     s.now(),
			OriginalFormat: OriginalFormatLegacyMarkdown,
			Note:           "Adopted from markdown via plans import (non-destructive).",
			WorkspaceID:    strings.TrimSpace(workspace.ID),
			WorkspaceRoot:  strings.TrimSpace(workspace.Root),
		}
	}
	stampCanonicalWorkspace(&parsed, workspace)
	return s.Create(ctx, parsed)
}

// Migrate ensures a plan resolved from a fallback location is present in the
// canonical home store. If the plan is already canonical it is idempotently
// touched; otherwise the resolver reads the legacy ~/.vrooli/plans index (plus
// the documented fallback locations), parses the referenced markdown, and
// imports it as a structured plan. The fallback source is never destructively
// removed here.
func (s *service) Migrate(ctx context.Context, idOrSlug string) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug, WorkspaceScope{})
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
	return s.Import(ctx, sourcePath, markdown, "", "", WorkspaceScope{})
}

// Reconcile repairs rendered mirrors and optionally adopts legacy markdown
// sources in bulk. Legacy sources are never deleted, moved, or overwritten.
func (s *service) Reconcile(ctx context.Context, req ReconcileRequest) (ReconcileResult, error) {
	if !req.RepairMirrors && !req.AdoptLegacy {
		req.RepairMirrors = true
		req.AdoptLegacy = true
	}
	if req.ConflictPolicy == "" {
		req.ConflictPolicy = ReconcileConflictSkipExisting
	}
	if !req.SourceRuntimeHomePlans && !req.SourceDocsPlans && !req.SourceRepoPlans {
		req.SourceRuntimeHomePlans = true
		req.SourceDocsPlans = true
		req.SourceRepoPlans = true
	}
	result := ReconcileResult{DryRun: req.DryRun}
	existing, err := s.repo.List(ctx, ListFilter{IncludeArchived: true})
	if err != nil {
		return result, err
	}
	if req.RepairMirrors {
		for _, p := range existing {
			if p.Status == PlanStatusArchived && !req.IncludeArchived {
				continue
			}
			result.Items = append(result.Items, s.reconcileMirrorItem(ctx, p, req.DryRun))
		}
	}
	if req.AdoptLegacy {
		items, err := s.reconcileLegacyItems(ctx, req, existing)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, items...)
	}
	return result, nil
}

func (s *service) reconcileMirrorItem(ctx context.Context, p Plan, dryRun bool) ReconcileItem {
	item := ReconcileItem{
		PlanID: p.ID,
		Slug:   p.Slug,
		Title:  p.Title,
		Mirror: s.ensureMirrorPath(ctx, p),
	}
	_, meta, err := s.mirror.Read(ctx, p)
	if err == nil && meta.Status == RenderedMirrorStatusFresh {
		item.Action = ReconcileActionMirrorFresh
		item.Mirror = meta
		return item
	}
	if meta.Status != "" {
		item.Mirror = meta
	}
	if dryRun {
		item.Action = ReconcileActionMirrorRepairNeeded
		if err != nil {
			item.Error = err.Error()
		}
		return item
	}
	repaired, repairErr := s.publishMirror(ctx, p)
	item.Mirror = repaired.Mirror
	if repairErr != nil {
		item.Action = ReconcileActionMirrorRepairNeeded
		item.Error = repairErr.Error()
		return item
	}
	item.Action = ReconcileActionMirrorRepaired
	return item
}

func (s *service) reconcileLegacyItems(ctx context.Context, req ReconcileRequest, existing []Plan) ([]ReconcileItem, error) {
	if s.reader == nil {
		return nil, ErrInvalidPlan{Reason: "no source reader configured; cannot scan legacy plans"}
	}
	mirrorPaths := make(map[string]bool, len(existing))
	importedSources := make(map[string]Plan, len(existing))
	contentHashes := make(map[string]Plan, len(existing))
	for _, p := range existing {
		if meta, err := s.mirror.PathFor(ctx, p); err == nil && strings.TrimSpace(meta.Path) != "" {
			mirrorPaths[filepath.Clean(meta.Path)] = true
		}
		if p.ImportProvenance != nil && strings.TrimSpace(p.ImportProvenance.SourcePath) != "" {
			importedSources[filepath.Clean(p.ImportProvenance.SourcePath)] = p
		}
		if strings.TrimSpace(p.ContentHash) != "" {
			contentHashes[p.ContentHash] = p
		}
	}
	var out []ReconcileItem
	for _, dir := range reconcileSourceDirs(req) {
		root := expandPlanLocation(dir)
		paths, err := s.listLegacySourcePaths(root, req.IncludeArchivedLegacy)
		if err != nil {
			out = append(out, ReconcileItem{Action: ReconcileActionConflict, SourcePath: root, SourceUntouched: true, Error: err.Error()})
			continue
		}
		for _, sourcePath := range paths {
			clean := filepath.Clean(sourcePath)
			if mirrorPaths[clean] {
				out = append(out, ReconcileItem{Action: ReconcileActionAlreadyCanonical, SourcePath: clean, SourceUntouched: true})
				continue
			}
			if p, ok := importedSources[clean]; ok {
				out = append(out, ReconcileItem{Action: ReconcileActionAlreadyCanonical, PlanID: p.ID, Slug: p.Slug, Title: p.Title, SourcePath: clean, Mirror: p.Mirror, SourceUntouched: true})
				continue
			}
			raw, err := s.reader.ReadFile(clean)
			if err != nil {
				out = append(out, ReconcileItem{Action: ReconcileActionConflict, SourcePath: clean, SourceUntouched: true, Error: err.Error()})
				continue
			}
			parsed, err := ParsePlanMarkdown(string(raw))
			if err != nil {
				out = append(out, ReconcileItem{Action: ReconcileActionParseFailed, SourcePath: clean, SourceUntouched: true, Error: err.Error()})
				continue
			}
			parsed.ContentHash = contentHash(parsed)
			if match, ok := contentHashes[parsed.ContentHash]; ok {
				action := ReconcileActionSkippedDuplicate
				if req.ConflictPolicy == ReconcileConflictReportOnly {
					action = ReconcileActionConflict
				}
				out = append(out, ReconcileItem{Action: action, PlanID: match.ID, Slug: match.Slug, Title: parsed.Title, SourcePath: clean, Mirror: match.Mirror, SourceUntouched: true})
				continue
			}
			item := ReconcileItem{Action: ReconcileActionImportPlanned, Title: parsed.Title, SourcePath: clean, SourceUntouched: true}
			if !req.DryRun {
				imported, err := s.Import(ctx, clean, string(raw), "", "", req.Workspace)
				if err != nil {
					item.Action = ReconcileActionConflict
					item.Error = err.Error()
				} else {
					item.Action = ReconcileActionImported
					item.PlanID = imported.ID
					item.Slug = imported.Slug
					item.Title = imported.Title
					item.Mirror = imported.Mirror
					contentHashes[imported.ContentHash] = imported
					importedSources[clean] = imported
				}
			}
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *service) listLegacySourcePaths(root string, includeArchived bool) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	indexPath := filepath.Join(root, fallbackIndexFilename)
	if raw, err := s.reader.ReadFile(indexPath); err == nil {
		var idx fallbackIndex
		if err := json.Unmarshal(raw, &idx); err != nil {
			return nil, fmt.Errorf("decode fallback plan index %q: %w", indexPath, err)
		}
		for _, record := range idx.Plans {
			if record.Archived && !includeArchived {
				continue
			}
			sourcePath := strings.TrimSpace(record.Path)
			if sourcePath == "" {
				sourcePath = filepath.Join(root, record.Slug+".md")
			}
			clean := filepath.Clean(sourcePath)
			if !seen[clean] {
				seen[clean] = true
				out = append(out, clean)
			}
		}
	}
	paths, err := s.reader.ListMarkdownFiles(root)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		clean := filepath.Clean(path)
		if !seen[clean] {
			seen[clean] = true
			out = append(out, clean)
		}
	}
	return out, nil
}

func reconcileSourceDirs(req ReconcileRequest) []string {
	var dirs []string
	if req.SourceRuntimeHomePlans {
		dirs = append(dirs, FallbackReadLocations[0])
	}
	if req.SourceDocsPlans {
		dirs = append(dirs, workspaceLocation(FallbackReadLocations[1], req.Workspace))
	}
	if req.SourceRepoPlans {
		dirs = append(dirs, workspaceLocation(FallbackReadLocations[2], req.Workspace))
	}
	return dirs
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
	if location == "~/.vrooli/plans" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			if path, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyPlans); err == nil {
				return path
			}
		}
	}
	if strings.HasPrefix(location, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, strings.TrimPrefix(location, "~/"))
		}
	}
	return filepath.Clean(location)
}

func workspaceLocation(location string, workspace WorkspaceScope) string {
	root := strings.TrimSpace(workspace.Root)
	if root == "" || filepath.IsAbs(location) || strings.HasPrefix(location, "~/") {
		return location
	}
	return filepath.Join(root, location)
}

func resolveWorkspacePath(path string, workspace WorkspaceScope) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	root := strings.TrimSpace(workspace.Root)
	if root == "" {
		return filepath.Clean(path), nil
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", ErrInvalidPlan{Reason: fmt.Sprintf("invalid workspace root %q: %v", root, err)}
	}
	cleanRoot = filepath.Clean(cleanRoot)
	if _, err := repocontract.LoadDefault(cleanRoot); err != nil {
		return "", ErrInvalidPlan{Reason: fmt.Sprintf("invalid workspace root %q: %v", cleanRoot, err)}
	}
	var resolved string
	if filepath.IsAbs(path) {
		resolved = filepath.Clean(path)
	} else {
		resolved = filepath.Clean(filepath.Join(cleanRoot, path))
	}
	if !pathWithin(cleanRoot, resolved) {
		return "", ErrInvalidPlan{Reason: fmt.Sprintf("import source %q is outside workspace root %q", resolved, cleanRoot)}
	}
	return resolved, nil
}

func pathWithin(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if root == path {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// ParsePlanMarkdown parses a markdown plan into the structured model.
func ParsePlanMarkdown(markdown string) (Plan, error) {
	return planmodel.ParsePlanMarkdown(markdown)
}
