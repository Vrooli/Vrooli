package templateengine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	. "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
)

// DetemplateHandler wires `vrooli scenario detemplate <name>`.
func DetemplateHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return bindGlobal(deps.Stdout,
		func(ctx C, args []string) (DetemplateRequest, error) {
			return ParseDetemplateRequest(deps.Globals(ctx).JSON, args)
		},
		func(ctx C, req DetemplateRequest) (cliout.Format, DetemplateResult, error) {
			return runDetemplate(deps, ctx, req)
		},
		RenderDetemplateResponse,
	)
}

// detemplateTextExtensions are the file types detemplate rewrites in place
// (fenced doc blocks + registration-line markers). Whole-file example
// artifacts are deleted via the manifest, not edited.
var detemplateTextExtensions = map[string]struct{}{
	".md": {}, ".go": {}, ".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {},
	".mjs": {}, ".cjs": {}, ".json": {}, ".css": {}, ".scss": {}, ".sql": {},
	".sh": {}, ".yaml": {}, ".yml": {}, ".proto": {}, ".txt": {}, ".html": {},
}

// detemplateCodeExtensions are scanned for dangling import references to
// deleted example code (the refuse-on-dangling-ref guard).
var detemplateCodeExtensions = map[string]struct{}{
	".go": {}, ".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {}, ".mjs": {}, ".cjs": {},
}

// detemplateSkipDirs are never walked when editing scenario files or scanning
// for residue. `.vrooli` is skipped because the rendered orientation tracker
// (`.vrooli/orientation.json`) embeds the residue check's own marker text and
// would otherwise self-trip the gate.
var detemplateSkipDirs = map[string]struct{}{
	".git": {}, "node_modules": {}, "dist": {}, "build": {}, ".next": {},
	"coverage": {}, "vendor": {}, ".turbo": {}, ".vrooli": {},
}

type detemplateDeletion struct {
	Display string // scenario-relative, or repo-relative for the relocated proto tree
	Abs     string
	IsProto bool
}

type detemplateEdit struct {
	Abs     string
	Content []byte
	Mode    fs.FileMode
	Summary FileMarkerSummary
}

type detemplateFinalizerPlan struct {
	Description string
	Name        string
	Args        []string
	Dir         string
}

func runDetemplate[C any](deps HandlerDeps[C], ctx C, req DetemplateRequest) (cliout.Format, DetemplateResult, error) {
	format := cliout.FormatHuman
	if req.JSON {
		format = cliout.FormatJSON
	}
	root := deps.Root(ctx)
	item, err := scenariomodel.Load(root, req.Name, scenariomodel.SandboxEnvFromEnv())
	if err != nil {
		return "", DetemplateResult{}, err
	}
	if item.Manifest.Generation == nil || strings.TrimSpace(item.Manifest.Generation.Template.ID) == "" {
		return "", DetemplateResult{}, fmt.Errorf("scenario %s has no template provenance; cannot determine its example domain", item.Slug)
	}
	templateID := item.Manifest.Generation.Template.ID
	info, err := loadTemplate(root, templateID)
	if err != nil {
		return "", DetemplateResult{}, fmt.Errorf("load template %q for scenario %s: %w", templateID, item.Slug, err)
	}
	ex := info.Manifest.ExampleDomain
	if ex == nil || strings.TrimSpace(ex.Marker) == "" {
		return "", DetemplateResult{}, fmt.Errorf("template %q declares no exampleDomain; nothing to detemplate", templateID)
	}
	marker := ex.Marker

	result := DetemplateResult{
		Scenario:     item.Slug,
		ScenarioPath: item.Path,
		Marker:       marker,
		DryRun:       req.DryRun,
	}

	deletions := resolveDetemplateDeletions(root, item, info, ex.Paths)
	edits, dangling, err := planDetemplateEdits(item.Path, marker, deletions)
	if err != nil {
		return "", DetemplateResult{}, err
	}
	if len(dangling) > 0 {
		return "", DetemplateResult{}, &DetemplateDanglingRefError{Marker: marker, References: dangling}
	}
	jsonEdits, err := planDetemplateJSONPrunes(item.Path, ex.JSONPrune)
	if err != nil {
		return "", DetemplateResult{}, err
	}
	edits = append(edits, jsonEdits...)
	sort.Slice(edits, func(i, j int) bool { return edits[i].Summary.Path < edits[j].Summary.Path })

	for _, e := range edits {
		result.FilesEdited = append(result.FilesEdited, e.Summary)
		result.BlocksRemoved += e.Summary.BlocksRemoved
		result.LinesStripped += e.Summary.LinesStripped
	}
	for _, d := range deletions {
		result.PathsDeleted = append(result.PathsDeleted, d.Display)
	}

	protoTouched := false
	for _, d := range deletions {
		if d.IsProto {
			protoTouched = true
			break
		}
	}
	plans := planDetemplateFinalizers(root, item, protoTouched)

	if req.DryRun {
		for _, p := range plans {
			result.Finalizers = append(result.Finalizers, DetemplateFinalizer{
				Description: p.Description,
				Command:     p.commandLine(),
				Ran:         false,
			})
		}
		result.Message = "Dry run: no files were written, deleted, or finalized."
		return format, result, nil
	}

	if len(edits) == 0 && len(deletions) == 0 {
		result.Message = fmt.Sprintf("No %q example-domain residue found; scenario is already detemplated.", marker)
		return format, result, nil
	}

	for _, e := range edits {
		if err := os.WriteFile(e.Abs, e.Content, e.Mode); err != nil {
			return "", DetemplateResult{}, fmt.Errorf("rewrite %s: %w", e.Summary.Path, err)
		}
	}
	for _, d := range deletions {
		if err := os.RemoveAll(d.Abs); err != nil {
			return "", DetemplateResult{}, fmt.Errorf("delete %s: %w", d.Display, err)
		}
	}

	for _, p := range plans {
		fin := DetemplateFinalizer{Description: p.Description, Command: p.commandLine()}
		if deps.RunSubprocess == nil {
			fin.Message = "skipped (no subprocess runner)"
			result.Finalizers = append(result.Finalizers, fin)
			continue
		}
		fin.Ran = true
		if runErr := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   p.Name,
			Args:   p.Args,
			Dir:    p.Dir,
			Env:    deps.CommandEnv(ctx),
			Stdout: deps.Stderr(ctx),
			Stderr: deps.Stderr(ctx),
		}); runErr != nil {
			fin.OK = false
			fin.Message = runErr.Error()
		} else {
			fin.OK = true
		}
		result.Finalizers = append(result.Finalizers, fin)
	}

	result.Message = fmt.Sprintf("Removed the %q example domain. Run `vrooli scenario orient %s` to confirm the example-domain-removed gate.", marker, item.Slug)
	return format, result, nil
}

// resolveDetemplateDeletions maps the manifest's example-domain paths onto
// absolute paths in the generated scenario. Scenario-local paths join the
// scenario root; paths under a relocation source (proto/) are mapped through
// the relocation target, and the corresponding generated proto artifacts are
// added so no orphan gen residue survives. Only paths that exist are returned,
// which makes a second run a no-op (idempotent).
func resolveDetemplateDeletions(root string, item scenariomodel.Scenario, info TemplateInfo, paths []string) []detemplateDeletion {
	var out []detemplateDeletion
	seen := map[string]struct{}{}
	add := func(abs, display string, isProto bool) {
		if _, dup := seen[abs]; dup {
			return
		}
		if _, err := os.Stat(abs); err != nil {
			return
		}
		seen[abs] = struct{}{}
		out = append(out, detemplateDeletion{Display: display, Abs: abs, IsProto: isProto})
	}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		reloc, rel, ok := matchRelocation(info, p)
		if !ok {
			add(filepath.Join(item.Path, filepath.FromSlash(p)), p, false)
			continue
		}
		// Proto schema source, mapped through the relocation target.
		toBase := strings.ReplaceAll(reloc.To, "{{SCENARIO_ID}}", item.Slug)
		schemaRel := filepath.Join(filepath.FromSlash(strings.TrimSuffix(toBase, "/")), filepath.FromSlash(rel))
		add(filepath.Join(root, schemaRel), filepath.ToSlash(schemaRel), true)
		// Generated proto artifacts for the same module (so make generate only
		// has to refresh the descriptor, not race orphaned outputs).
		for _, gen := range generatedProtoDirs(item.Slug, rel) {
			add(filepath.Join(root, filepath.FromSlash(gen)), gen, true)
		}
	}
	return out
}

// matchRelocation returns the relocation whose From is a path prefix of p,
// along with p relative to that From.
func matchRelocation(info TemplateInfo, p string) (TemplateRelocation, string, bool) {
	for _, r := range info.Manifest.Relocations {
		from := strings.TrimSuffix(r.From, "/") + "/"
		if strings.HasPrefix(p, from) {
			return r, strings.TrimPrefix(p, from), true
		}
	}
	return TemplateRelocation{}, "", false
}

// generatedProtoDirs returns the repo-relative generated artifact directories
// for one relocated proto module (rel is e.g. "v1/notes").
func generatedProtoDirs(scenarioID, rel string) []string {
	snake := strings.ReplaceAll(scenarioID, "-", "_")
	rel = filepath.ToSlash(rel)
	return []string{
		"packages/proto/gen/go/" + scenarioID + "/" + rel,
		"packages/proto/gen/typescript/" + scenarioID + "/" + rel,
		"packages/proto/gen/typescript/js/" + scenarioID + "/" + rel,
		"packages/proto/gen/python/" + snake + "/" + rel,
	}
}

// planDetemplateEdits walks the scenario tree, computes the rewritten content
// for every text file carrying example markers, and detects dangling
// references — kept code files that still import a to-be-deleted example
// package after marker removal.
func planDetemplateEdits(scenarioRoot, marker string, deletions []detemplateDeletion) ([]detemplateEdit, []DetemplateDanglingRef, error) {
	deletedAbs := make([]string, 0, len(deletions))
	for _, d := range deletions {
		deletedAbs = append(deletedAbs, d.Abs)
	}
	tokens := danglingTokens(scenarioRoot, deletions)

	var edits []detemplateEdit
	var dangling []DetemplateDanglingRef
	err := filepath.WalkDir(scenarioRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if _, skip := detemplateSkipDirs[entry.Name()]; skip {
				return fs.SkipDir
			}
			if underDeleted(path, deletedAbs) {
				return fs.SkipDir
			}
			return nil
		}
		if underDeleted(path, deletedAbs) {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := detemplateTextExtensions[ext]; !ok {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(scenarioRoot, path)
		rel = filepath.ToSlash(rel)
		out, summary, changed := StripExampleDomainFile(rel, content, marker)
		if changed {
			info, statErr := entry.Info()
			mode := fs.FileMode(0o644)
			if statErr == nil {
				mode = info.Mode().Perm()
			}
			edits = append(edits, detemplateEdit{Abs: path, Content: out, Mode: mode, Summary: summary})
		}
		// Dangling-ref scan on the post-strip content of kept code files.
		if _, isCode := detemplateCodeExtensions[ext]; isCode && len(tokens) > 0 {
			for _, ref := range danglingReferences(out, tokens) {
				dangling = append(dangling, DetemplateDanglingRef{File: rel, Reference: ref})
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].Summary.Path < edits[j].Summary.Path })
	sort.Slice(dangling, func(i, j int) bool {
		if dangling[i].File != dangling[j].File {
			return dangling[i].File < dangling[j].File
		}
		return dangling[i].Reference < dangling[j].Reference
	})
	return edits, dangling, nil
}

// planDetemplateJSONPrunes computes order-preserving prunes for hand-authored
// JSON files (i18n locales, the CLI manifest) that cannot carry comment
// markers. Absent files are skipped, which keeps a second run a no-op.
func planDetemplateJSONPrunes(scenarioRoot string, entries []TemplateJSONPruneEntry) ([]detemplateEdit, error) {
	var edits []detemplateEdit
	for _, e := range entries {
		abs := filepath.Join(scenarioRoot, filepath.FromSlash(e.File))
		info, statErr := os.Stat(abs)
		if statErr != nil {
			continue
		}
		content, readErr := os.ReadFile(abs)
		if readErr != nil {
			return nil, readErr
		}
		out, removed, pruneErr := PruneJSON(content, e.Keys, e.ArrayMatch)
		if pruneErr != nil {
			return nil, fmt.Errorf("prune %s: %w", e.File, pruneErr)
		}
		if removed == 0 {
			continue
		}
		edits = append(edits, detemplateEdit{
			Abs:     abs,
			Content: out,
			Mode:    info.Mode().Perm(),
			Summary: FileMarkerSummary{Path: e.File, LinesStripped: removed},
		})
	}
	return edits, nil
}

func underDeleted(path string, deletedAbs []string) bool {
	for _, d := range deletedAbs {
		if path == d || strings.HasPrefix(path, d+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// danglingTokens builds import-path tokens (last two segments, extension
// stripped) for scenario-local deletions. Each token always contains a slash,
// so single-word incidental mentions can't match.
func danglingTokens(scenarioRoot string, deletions []detemplateDeletion) []*regexp.Regexp {
	var tokens []*regexp.Regexp
	seen := map[string]struct{}{}
	for _, d := range deletions {
		if d.IsProto {
			continue
		}
		rel := strings.TrimSuffix(filepath.ToSlash(d.Display), filepath.Ext(d.Display))
		segs := strings.Split(rel, "/")
		if len(segs) < 2 {
			continue
		}
		token := strings.Join(segs[len(segs)-2:], "/")
		if _, dup := seen[token]; dup {
			continue
		}
		seen[token] = struct{}{}
		// Match the token only inside a single-line double-quoted import path.
		// `[^"\n]` (not `[^"]`) keeps the match on one line so it cannot span
		// from one quote, across a comment mentioning the deleted package, to
		// an unrelated later quote — the bucket-3 incidental-comment trap.
		tokens = append(tokens, regexp.MustCompile(`"[^"\n]*`+regexp.QuoteMeta(token)+`[^"\n]*"`))
	}
	return tokens
}

func danglingReferences(content []byte, tokens []*regexp.Regexp) []string {
	var refs []string
	for _, re := range tokens {
		if m := re.Find(content); m != nil {
			refs = append(refs, strings.Trim(string(m), `"`))
		}
	}
	return refs
}

// planDetemplateFinalizers lists the post-strip commands that refresh
// generated artifacts and tidy/format the scenario. Each is conditional on the
// relevant surface existing, so a CLI-only or UI-only scenario only runs what
// applies.
func planDetemplateFinalizers(root string, item scenariomodel.Scenario, protoTouched bool) []detemplateFinalizerPlan {
	var plans []detemplateFinalizerPlan
	exists := func(parts ...string) bool {
		_, err := os.Stat(filepath.Join(parts...))
		return err == nil
	}
	if protoTouched && exists(root, "packages", "proto", "Makefile") {
		plans = append(plans, detemplateFinalizerPlan{
			Description: "Regenerate proto artifacts (make generate)",
			Name:        "make", Args: []string{"generate"},
			Dir: filepath.Join(root, "packages", "proto"),
		})
	}
	if exists(item.Path, "ui", "package.json") {
		plans = append(plans, detemplateFinalizerPlan{
			Description: "Regenerate UI strings (pnpm strings:gen)",
			Name:        "corepack", Args: []string{"pnpm", "run", "strings:gen"},
			Dir: filepath.Join(item.Path, "ui"),
		})
	}
	if exists(item.Path, "api", "go.mod") {
		plans = append(plans, detemplateFinalizerPlan{
			Description: "Tidy API Go module (go mod tidy)",
			Name:        "go", Args: []string{"mod", "tidy"},
			Dir: filepath.Join(item.Path, "api"),
		})
		plans = append(plans, detemplateFinalizerPlan{
			Description: "Format API Go sources (gofumpt)",
			Name:        "gofumpt", Args: []string{"-w", "."},
			Dir: filepath.Join(item.Path, "api"),
		})
	}
	if exists(item.Path, "cli", "go.mod") {
		plans = append(plans, detemplateFinalizerPlan{
			Description: "Tidy CLI Go module (go mod tidy)",
			Name:        "go", Args: []string{"mod", "tidy"},
			Dir: filepath.Join(item.Path, "cli"),
		})
		plans = append(plans, detemplateFinalizerPlan{
			Description: "Format CLI Go sources (gofumpt)",
			Name:        "gofumpt", Args: []string{"-w", "."},
			Dir: filepath.Join(item.Path, "cli"),
		})
	}
	return plans
}

func (p detemplateFinalizerPlan) commandLine() string {
	return strings.TrimSpace(p.Name + " " + strings.Join(p.Args, " "))
}

// scanTreeForText walks root (text files only, skipping vendored/build dirs)
// and returns the scenario-relative paths of files containing text. Used by
// the example-domain residue gate (orient text_absent_tree check) and the
// deep-validation round-trip.
func scanTreeForText(root, text string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	needle := []byte(text)
	var hits []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if _, skip := detemplateSkipDirs[entry.Name()]; skip {
				return fs.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := detemplateTextExtensions[ext]; !ok {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytesContains(content, needle) {
			rel, _ := filepath.Rel(root, path)
			hits = append(hits, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(hits)
	return hits, nil
}

func bytesContains(haystack, needle []byte) bool {
	return strings.Contains(string(haystack), string(needle))
}
