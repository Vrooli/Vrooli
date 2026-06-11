package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	pkg "github.com/vrooli/ai-go/search"

	"knowledge-observatory/internal/doccontract"
)

// ErrScenariosRootRequired is returned by NewDocSource when no scenarios root
// is configured (KO cannot locate the repository to index).
var ErrScenariosRootRequired = errors.New("aisearch: scenarios root is required")

// maxDocBytes skips pathologically large files (generated dumps, vendored
// blobs) that are never useful documentation. 1 MiB comfortably covers every
// hand-written doc in the repo.
const maxDocBytes = 1 << 20

// docExtensions are the text/markdown extensions kodocsource indexes. Manifest
// entries pointing at non-prose artifacts (e.g. .json schemas) are skipped.
var docExtensions = map[string]struct{}{
	".md":       {},
	".mdx":      {},
	".markdown": {},
	".txt":      {},
}

// prunedDirs are directory names never descended while discovering manifests —
// VCS metadata, dependency trees, build output, generated code, runtime data,
// and test fixtures. Pruning keeps discovery fast and avoids permission-denied
// runtime data dirs (e.g. resources/*/instances).
var prunedDirs = map[string]struct{}{
	".git": {}, "node_modules": {}, "vendor": {}, "dist": {}, "build": {},
	".next": {}, ".turbo": {}, ".cache": {}, ".vrooli": {}, "coverage": {},
	"instances": {}, "gen": {}, "testdata": {}, "__pycache__": {}, ".venv": {},
}

// DocSource is KO's manifest-driven documentation Source. It walks every
// docs/manifest.json across the repo (root + scenarios + template scenarios),
// emits one SourceDoc per documentation file with manifest metadata, and
// supplements scenario/project README.md and PRD.md files not registered in a
// manifest. It implements github.com/vrooli/ai-go/search.Source.
type DocSource struct {
	repoRoot      string
	scenariosRoot string
}

var _ pkg.Source = (*DocSource)(nil)

// NewDocSource builds a documentation source rooted at the given scenarios
// directory; the repository root is its parent (mirroring docsearch.Service).
func NewDocSource(scenariosRoot string) (*DocSource, error) {
	scenariosRoot = strings.TrimSpace(scenariosRoot)
	if scenariosRoot == "" {
		return nil, ErrScenariosRootRequired
	}
	info, err := os.Stat(scenariosRoot)
	if err != nil || !info.IsDir() {
		return nil, ErrScenariosRootRequired
	}
	repoRoot := filepath.Dir(scenariosRoot)
	if repoRoot == "" || repoRoot == scenariosRoot {
		return nil, ErrScenariosRootRequired
	}
	return &DocSource{repoRoot: repoRoot, scenariosRoot: scenariosRoot}, nil
}

// LoadAll enumerates every documentation file as a SourceDoc. A failure to read
// one manifest or file is logged and skipped — indexing never crashes. Output
// is sorted by ID (repo-relative path) for deterministic planning.
func (s *DocSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	seen := make(map[string]struct{}, 512)
	out := make([]pkg.SourceDoc, 0, 512)

	for _, manifestAbs := range s.findManifests(ctx) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		s.loadManifest(manifestAbs, seen, &out)
	}

	// Manifest-registered docs are added first (richest metadata wins via seen);
	// the sweep then backfills any documentation file a manifest omitted so
	// search coverage is complete, not manifest-gated.
	s.supplementDocsTrees(seen, &out)
	s.supplementReadmesAndPRDs(seen, &out)

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// findManifests collects every docs/manifest.json under the repo root, pruning
// heavy/irrelevant directories.
func (s *DocSource) findManifests(ctx context.Context) []string {
	var manifests []string
	_ = filepath.WalkDir(s.repoRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// Unreadable entry (e.g. permission-denied runtime dir): skip it
			// and, if a directory, don't try to descend.
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if ctx.Err() != nil {
			return filepath.SkipDir
		}
		if d.IsDir() {
			if p != s.repoRoot {
				if _, pruned := prunedDirs[d.Name()]; pruned {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if d.Name() == "manifest.json" && filepath.Base(filepath.Dir(p)) == "docs" {
			manifests = append(manifests, p)
		}
		return nil
	})
	sort.Strings(manifests)
	return manifests
}

// loadManifest parses one manifest and appends a SourceDoc for each of its
// documents (deduped against seen).
func (s *DocSource) loadManifest(manifestAbs string, seen map[string]struct{}, out *[]pkg.SourceDoc) {
	manifest, err := doccontract.LoadManifest(manifestAbs)
	if err != nil {
		log.Printf("[ko/aisearch] load manifest %s: %v", manifestAbs, err)
		return
	}
	// The base directory is the parent of docs/ (the scenario/project root),
	// matching doccontract's NormalizeManifestPath convention (docs/<x>, or a
	// bare path for ../ references that escape docs/).
	baseDirAbs := filepath.Dir(filepath.Dir(manifestAbs))
	scope, scenario := s.deriveScopeScenario(baseDirAbs)

	for _, section := range manifest.Sections {
		for _, doc := range section.Documents {
			normalized := doccontract.NormalizeManifestPath(doc.Path)
			if normalized == "" {
				continue
			}
			fileAbs := filepath.Join(baseDirAbs, filepath.FromSlash(normalized))
			meta := docMeta{
				docType:      strings.TrimSpace(doc.DocType),
				title:        strings.TrimSpace(doc.Title),
				description:  strings.TrimSpace(doc.Description),
				audience:     doc.Audience,
				canonicalFor: doc.CanonicalFor,
				maturity:     strings.TrimSpace(doc.Maturity),
				scope:        scope,
				scenario:     scenario,
				origin:       SourceManifest,
			}
			s.addDoc(fileAbs, meta, seen, out)
		}
	}
}

// supplementDocsTrees indexes every documentation file under each docs/ tree
// (project, scenarios, template scenarios) that a manifest did not already
// register. It runs after the manifest pass and dedups via seen, so
// manifest-registered docs keep their richer metadata while un-listed files
// still become searchable with inferred metadata (title from first heading,
// docType inferred). This makes search coverage a property of the filesystem,
// not of manifest hygiene (plan §3.2 "all-docs discovery").
func (s *DocSource) supplementDocsTrees(seen map[string]struct{}, out *[]pkg.SourceDoc) {
	type tree struct {
		dirAbs   string
		scope    string
		scenario string
	}
	trees := []tree{{dirAbs: filepath.Join(s.repoRoot, "docs"), scope: ScopeProject}}
	addScenarioTrees := func(parent string) {
		entries, err := os.ReadDir(parent)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			trees = append(trees, tree{
				dirAbs:   filepath.Join(parent, e.Name(), "docs"),
				scope:    ScopeScenario,
				scenario: e.Name(),
			})
		}
	}
	addScenarioTrees(s.scenariosRoot)
	addScenarioTrees(filepath.Join(s.repoRoot, "templates", "scenarios"))

	for _, tr := range trees {
		s.sweepDocsTree(tr.dirAbs, tr.scope, tr.scenario, seen, out)
	}
}

// trackerFiles are filenames never indexed for search regardless of discovery
// path (manifest, sweep, or README/PRD supplement): internal status/scratch
// trackers that are not user-facing documentation and otherwise flood results
// (438 chunks — 209 PROGRESS + 229 PROBLEMS — across the repo as of 2026-06).
// They remain valid docs in the manifest/viewer; they are just excluded from the
// semantic index. addDoc enforces this for every path.
var trackerFiles = map[string]struct{}{
	"progress.md":             {},
	"problems.md":             {},
	"refactoring_progress.md": {},
}

// sweepDocsTree walks one docs/ directory, adding every text doc not already
// seen. Pruned subdirectories (build output, runtime data, fixtures) are
// skipped; tracker files are filtered centrally in addDoc.
func (s *DocSource) sweepDocsTree(dirAbs, scope, scenario string, seen map[string]struct{}, out *[]pkg.SourceDoc) {
	info, err := os.Stat(dirAbs)
	if err != nil || !info.IsDir() {
		return
	}
	_ = filepath.WalkDir(dirAbs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if _, pruned := prunedDirs[d.Name()]; pruned {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := docExtensions[strings.ToLower(filepath.Ext(p))]; !ok {
			return nil
		}
		s.addDoc(p, docMeta{scope: scope, scenario: scenario, origin: SourceSweep}, seen, out)
		return nil
	})
}

// supplementReadmesAndPRDs adds the project root and each scenario's README.md
// and PRD.md when they were not already registered by a manifest, so a scenario
// without a docs manifest is still discoverable.
func (s *DocSource) supplementReadmesAndPRDs(seen map[string]struct{}, out *[]pkg.SourceDoc) {
	type loc struct {
		dirAbs   string
		scope    string
		scenario string
	}
	locs := []loc{{dirAbs: s.repoRoot, scope: ScopeProject}}
	if entries, err := os.ReadDir(s.scenariosRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			locs = append(locs, loc{
				dirAbs:   filepath.Join(s.scenariosRoot, e.Name()),
				scope:    ScopeScenario,
				scenario: e.Name(),
			})
		}
	}
	for _, l := range locs {
		for _, f := range []struct {
			name    string
			docType string
			origin  string
		}{
			{"README.md", "readme", SourceReadme},
			{"PRD.md", "prd", SourcePRD},
		} {
			s.addDoc(filepath.Join(l.dirAbs, f.name), docMeta{
				docType:  f.docType,
				scope:    l.scope,
				scenario: l.scenario,
				origin:   f.origin,
			}, seen, out)
		}
	}
}

// docMeta is the metadata carried from discovery into a SourceDoc's payload.
type docMeta struct {
	docType      string
	title        string
	description  string
	audience     []string
	canonicalFor []string
	maturity     string
	scope        string
	scenario     string
	origin       string
}

// addDoc reads one file and appends a SourceDoc, deduping by repo-relative
// path and skipping missing/oversized/non-text/out-of-repo files.
func (s *DocSource) addDoc(fileAbs string, meta docMeta, seen map[string]struct{}, out *[]pkg.SourceDoc) {
	relPath := s.repoRelative(fileAbs)
	if relPath == "" || strings.HasPrefix(relPath, "../") {
		return // escapes the repo root
	}
	if _, ok := seen[relPath]; ok {
		return
	}
	if _, ok := docExtensions[strings.ToLower(filepath.Ext(fileAbs))]; !ok {
		return
	}
	if _, isTracker := trackerFiles[strings.ToLower(filepath.Base(fileAbs))]; isTracker {
		return // internal status tracker: a valid doc, but never search-indexed
	}
	info, err := os.Stat(fileAbs)
	if err != nil || info.IsDir() {
		return // referenced doc not yet written; the manifest validator flags it
	}
	if info.Size() > maxDocBytes {
		log.Printf("[ko/aisearch] skip oversized doc %s (%d bytes)", relPath, info.Size())
		return
	}
	body, err := os.ReadFile(fileAbs)
	if err != nil {
		log.Printf("[ko/aisearch] read doc %s: %v", relPath, err)
		return
	}
	seen[relPath] = struct{}{}

	title := meta.title
	if title == "" {
		title = firstHeading(string(body))
	}
	if title == "" {
		title = titleFromPath(relPath)
	}
	docType := meta.docType
	if docType == "" {
		docType = inferDocType(relPath)
	}

	payload := map[string]any{
		MetaScenario:     meta.scenario,
		MetaRelativePath: relPath,
		MetaPath:         relPath,
		MetaDocType:      docType,
		MetaTitle:        title,
		MetaDescription:  meta.description,
		MetaAudience:     normalizeStrings(meta.audience),
		MetaCanonicalFor: normalizeStrings(meta.canonicalFor),
		MetaMaturity:     meta.maturity,
		MetaScope:        meta.scope,
		MetaSource:       meta.origin,
		MetaPathPrefixes: pathPrefixes(relPath),
	}

	*out = append(*out, pkg.SourceDoc{
		ID:          relPath,
		Kind:        DocKind,
		ContentHash: contentHash(string(body), title, meta.description, docType, meta.maturity, meta.scope, meta.scenario, payload[MetaAudience], payload[MetaCanonicalFor]),
		Body:        string(body),
		Meta:        payload,
	})
}

// pathPrefixes returns the ancestor directory prefixes of a repo-relative path,
// e.g. "docs/concepts/ARCHITECTURE.md" -> ["docs", "docs/concepts"]. Stored in
// the payload so a path scope ("docs/concepts") can filter server-side by exact
// segment match, instead of relying on a global top-K then a client trim (which
// returns nothing when in-prefix docs fall outside the global shortlist).
func pathPrefixes(relPath string) []string {
	parts := strings.Split(relPath, "/")
	if len(parts) <= 1 {
		return nil
	}
	prefixes := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		prefixes = append(prefixes, strings.Join(parts[:i], "/"))
	}
	return prefixes
}

// deriveScopeScenario classifies a manifest base directory (the parent of
// docs/) as a project-level or scenario-level corpus and names the scenario.
func (s *DocSource) deriveScopeScenario(baseDirAbs string) (scope, scenario string) {
	baseRel := s.repoRelative(baseDirAbs)
	baseRel = strings.Trim(baseRel, "/")
	if baseRel == "" || baseRel == "." {
		return ScopeProject, ""
	}
	parts := strings.Split(baseRel, "/")
	switch {
	case parts[0] == "scenarios" && len(parts) >= 2:
		return ScopeScenario, parts[1]
	case parts[0] == "templates" && len(parts) >= 3 && parts[1] == "scenarios":
		return ScopeScenario, parts[2]
	default:
		return ScopeProject, ""
	}
}

// repoRelative returns the forward-slashed repo-relative path of an absolute
// path, or "" if it cannot be made relative.
func (s *DocSource) repoRelative(abs string) string {
	rel, err := filepath.Rel(s.repoRoot, abs)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

// contentHash is the source-level drift gate. It hashes the file body together
// with every embedding- and payload-affecting metadata field, so a manifest
// metadata edit (title/description/facets) busts the source-level skip and
// re-plans the file even though its bytes are unchanged (plan §4.6). The
// contextual composer embeds title+description+heading-path, so those must be
// covered; facets ride the payload, so they are covered too.
func contentHash(parts ...any) string {
	h := sha256.New()
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			h.Write([]byte(v))
		case []string:
			for _, e := range v {
				h.Write([]byte(e))
				h.Write([]byte{0x1e})
			}
		}
		h.Write([]byte{0x1f})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)[:16])
}

// normalizeStrings trims, drops empties, and returns a non-nil slice for stable
// payload encoding.
func normalizeStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// firstHeading returns the text of a leading ATX heading (the document title),
// or "" if the body opens with prose instead of a heading.
func firstHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			return strings.TrimSpace(strings.TrimLeft(trimmed, "# "))
		}
		return "" // prose before any heading: no title heading
	}
	return ""
}

// titleFromPath derives a human title from a file's base name when no manifest
// title or heading is available (e.g. "ERROR-HANDLING.md" -> "Error Handling").
func titleFromPath(relPath string) string {
	base := strings.TrimSuffix(path.Base(relPath), path.Ext(relPath))
	base = strings.NewReplacer("-", " ", "_", " ").Replace(base)
	fields := strings.Fields(base)
	for i, f := range fields {
		fields[i] = strings.ToUpper(f[:1]) + strings.ToLower(f[1:])
	}
	return strings.Join(fields, " ")
}

// inferDocType guesses a docType for files lacking a manifest docType.
func inferDocType(relPath string) string {
	base := strings.ToLower(strings.TrimSuffix(path.Base(relPath), path.Ext(relPath)))
	switch base {
	case "readme":
		return "readme"
	case "prd":
		return "prd"
	default:
		return "doc"
	}
}
