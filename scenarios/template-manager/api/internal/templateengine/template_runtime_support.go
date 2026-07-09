package templateengine

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	. "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts" //nolint:revive // templateengine is a thin glue layer over templatecontracts; dot-import keeps wiring readable.
)

func loadTemplates(root string) ([]TemplateInfo, error) {
	baseDir := config.TemplateBaseDir(root)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	templates := make([]TemplateInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		info, err := loadTemplate(root, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				templates = append(templates, TemplateInfo{Name: name, Path: filepath.Join(baseDir, name), Missing: true})
				continue
			}
			return nil, err
		}
		templates = append(templates, info)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
	return templates, nil
}

func loadTemplate(root, name string) (TemplateInfo, error) {
	templateDir := filepath.Join(config.TemplateBaseDir(root), name)
	info, err := os.Stat(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return TemplateInfo{}, fmt.Errorf("template not found: %s", name)
		}
		return TemplateInfo{}, err
	}
	if !info.IsDir() {
		return TemplateInfo{}, fmt.Errorf("template path is not a directory: %s", templateDir)
	}
	data, err := os.ReadFile(filepath.Join(templateDir, "template.json"))
	if err != nil {
		return TemplateInfo{}, err
	}
	var manifest TemplateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return TemplateInfo{}, err
	}
	if manifest.Name == "" {
		manifest.Name = name
	}
	if manifest.RequiredVars == nil {
		manifest.RequiredVars = map[string]TemplateVar{}
	}
	if manifest.OptionalVars == nil {
		manifest.OptionalVars = map[string]TemplateVar{}
	}
	if manifest.Docs == nil {
		manifest.Docs = map[string]string{}
	}
	return TemplateInfo{Name: name, Path: templateDir, Manifest: manifest}, nil
}

func buildTemplateValues(root, destination, templateName string, manifest TemplateManifest, baseValues map[string]string) (map[string]string, error) {
	currentDate := time.Now().UTC().Format("2006-01-02")
	randomToken, err := randomTemplateToken()
	if err != nil {
		return nil, err
	}
	values := copyStringMap(baseValues)
	values["CURRENT_DATE"] = currentDate
	values["RANDOM_TOKEN"] = randomToken
	if err := populateTemplatePathValues(root, destination, values); err != nil {
		return nil, fmt.Errorf("resolve template path placeholders for %s: %w", templateName, err)
	}
	optionalKeys := make([]string, 0, len(manifest.OptionalVars))
	for key := range manifest.OptionalVars {
		optionalKeys = append(optionalKeys, key)
	}
	sort.Strings(optionalKeys)
	for _, key := range optionalKeys {
		if strings.TrimSpace(values[key]) == "" {
			values[key] = renderTemplateString(manifest.OptionalVars[key].Default, values)
		}
	}
	// Derive snake_case identifiers from kebab-case scenario IDs so proto
	// package directives (which forbid hyphens), Go package aliases, and
	// Python module names get a valid identifier without each template
	// having to re-implement the conversion.
	if id, ok := values["SCENARIO_ID"]; ok && strings.TrimSpace(id) != "" {
		values["SCENARIO_ID_SNAKE"] = strings.ReplaceAll(id, "-", "_")
	}
	return values, nil
}

func populateTemplatePathValues(root, destination string, values map[string]string) error {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return err
	}
	repoRoot := filepath.Clean(root)
	packagesDir, err := contract.TopLevelDir(root, "packages")
	if err != nil {
		return err
	}
	for key, dir := range map[string]string{
		"API":     filepath.Join(destination, "api"),
		"CLI":     filepath.Join(destination, "cli"),
		"RUNTIME": filepath.Join(destination, "runtime"),
	} {
		repoRel, err := filepath.Rel(dir, repoRoot)
		if err != nil {
			return err
		}
		packagesRel, err := filepath.Rel(dir, packagesDir)
		if err != nil {
			return err
		}
		values["REPO_ROOT_REL_FROM_"+key] = filepath.ToSlash(repoRel)
		values["PACKAGES_REL_FROM_"+key] = filepath.ToSlash(packagesRel)
	}
	return nil
}

func copyTemplate(templateDir, destination string, values map[string]string, manifest TemplateManifest) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	return walkTemplateEmissions(templateDir, manifest, func(relPath, absPath string, entry fs.DirEntry) error {
		targetPath := filepath.Join(destination, renderTemplateString(relPath, values))
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		if looksLikeTextFile(data) {
			data = []byte(renderTemplateString(string(data), values))
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

// walkTemplateEmissions iterates the template tree, invoking fn for every
// entry (file or directory) that would be emitted into a generated scenario
// destination. The skip rules — relocation sources, manifest.CopyExcludes,
// hard-coded dirs (node_modules/dist/build/coverage/.turbo/.vite), and the
// always-skipped files (.DS_Store, template.json) — are applied once here so
// copyTemplate and the content-hash walker stay in lockstep.
//
// relPath is the template-relative path (filepath-style separators). For
// directories, fn is called before its children are visited.
func walkTemplateEmissions(templateDir string, manifest TemplateManifest, fn func(relPath, absPath string, entry fs.DirEntry) error) error {
	relocSources := make(map[string]struct{}, len(manifest.Relocations))
	for _, reloc := range manifest.Relocations {
		from := strings.TrimSpace(reloc.From)
		if from == "" {
			continue
		}
		relocSources[filepath.Clean(filepath.FromSlash(from))] = struct{}{}
	}
	copyExcludes := make(map[string]struct{}, len(manifest.CopyExcludes))
	for _, exclude := range manifest.CopyExcludes {
		exclude = strings.TrimSpace(exclude)
		if exclude == "" {
			continue
		}
		copyExcludes[filepath.Clean(filepath.FromSlash(exclude))] = struct{}{}
	}
	return filepath.WalkDir(templateDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == templateDir {
			return nil
		}
		if entry.IsDir() && shouldSkipTemplateCopyDir(entry.Name()) {
			return filepath.SkipDir
		}
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		cleanRel := filepath.Clean(relPath)
		if entry.IsDir() {
			if _, skip := relocSources[cleanRel]; skip {
				return filepath.SkipDir
			}
			if _, skip := copyExcludes[cleanRel]; skip {
				return filepath.SkipDir
			}
		}
		if filepath.Base(path) == ".DS_Store" || relPath == "template.json" {
			return nil
		}
		if _, skip := copyExcludes[cleanRel]; skip {
			return nil
		}
		return fn(relPath, path, entry)
	})
}

func buildGenerationProvenance(info TemplateInfo, design ResolvedDesign, now time.Time) GenerationProvenance {
	// Hash failures are non-fatal: a scenario must still generate even if the
	// hasher has a bug, and a stale/missing hash just makes drift unmeasurable
	// for that scenario — the surface degrades gracefully.
	manifestSha, contentSha, _ := computeTemplateHashes(info)
	return GenerationProvenance{
		Template: GenerationTemplate{
			ID:      info.Name,
			Version: strings.TrimSpace(info.Manifest.Version),
		},
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Design: GenerationDesign{
			ID:      strings.TrimSpace(design.KitID),
			Version: strings.TrimSpace(design.Version),
			Adapter: strings.TrimSpace(design.AdapterID),
		},
		ManifestSha: manifestSha,
		ContentSha:  contentSha,
	}
}

func injectScenarioProvenance(destination string, provenance GenerationProvenance) error {
	servicePath := filepath.Join(destination, ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read service manifest: %w", err)
	}
	var manifest scenariomodel.ServiceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse service manifest: %w", err)
	}
	manifest.Generation = &scenariomodel.GenerationMetadata{
		Template: scenariomodel.GenerationTemplate{
			ID:      provenance.Template.ID,
			Version: provenance.Template.Version,
		},
		GeneratedAt: provenance.GeneratedAt,
		Design: scenariomodel.GenerationDesign{
			ID:      provenance.Design.ID,
			Version: provenance.Design.Version,
			Adapter: provenance.Design.Adapter,
		},
		ManifestSha: provenance.ManifestSha,
		ContentSha:  provenance.ContentSha,
	}
	rendered, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("render service manifest: %w", err)
	}
	rendered = append(rendered, '\n')
	return os.WriteFile(servicePath, rendered, 0o644)
}

func renderOrientationManifest(destination string, manifest TemplateManifest, values map[string]string) error {
	if manifest.Orientation == nil {
		return nil
	}
	copyTo := strings.TrimSpace(manifest.Orientation.CopyTo)
	if copyTo == "" {
		return fmt.Errorf("orientation.copyTo is required")
	}
	cleanPath, err := cleanScenarioRelativePath(copyTo)
	if err != nil {
		return fmt.Errorf("orientation.copyTo: %w", err)
	}
	data, err := json.MarshalIndent(manifest.Orientation, "", "  ")
	if err != nil {
		return fmt.Errorf("render orientation manifest: %w", err)
	}
	data = []byte(renderTemplateString(string(data), values))
	target := filepath.Join(destination, cleanPath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(target, data, 0o644)
}

func verifyTemplate(destination string) error {
	var unresolved []string
	err := filepath.WalkDir(destination, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if unresolvedTemplatePattern.MatchString(path) {
			unresolved = append(unresolved, path)
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if looksLikeTextFile(data) && unresolvedTemplatePattern.Match(data) {
			unresolved = append(unresolved, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(unresolved) == 0 {
		return nil
	}
	sort.Strings(unresolved)
	return fmt.Errorf("unresolved placeholders remain in: %s", strings.Join(unresolved, ", "))
}

func shouldSkipTemplateCopyDir(name string) bool {
	switch strings.TrimSpace(name) {
	case "node_modules", "dist", "build", "coverage", ".turbo", ".vite":
		return true
	default:
		return false
	}
}

func looksLikeTextFile(data []byte) bool {
	return len(data) == 0 || (bytes.IndexByte(data, 0) < 0 && utf8.Valid(data))
}

func LooksLikeTextFile(data []byte) bool {
	return looksLikeTextFile(data)
}

func renderTemplateString(value string, values map[string]string) string {
	rendered := value
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", values[key])
	}
	return rendered
}

func templateValidationSeedValues(info TemplateInfo) map[string]string {
	values := map[string]string{}
	requiredKeys := make([]string, 0, len(info.Manifest.RequiredVars))
	for key := range info.Manifest.RequiredVars {
		requiredKeys = append(requiredKeys, key)
	}
	sort.Strings(requiredKeys)
	for _, key := range requiredKeys {
		switch key {
		case "SCENARIO_ID":
			values[key] = "template-validation-" + info.Name
		case "SCENARIO_DISPLAY_NAME":
			values[key] = coalesce(info.Manifest.DisplayName, info.Name+" Validation")
		case "SCENARIO_DESCRIPTION":
			values[key] = coalesce(info.Manifest.Description, "Validation scenario generated from "+info.Name)
		default:
			if fallback := strings.TrimSpace(info.Manifest.RequiredVars[key].Default); fallback != "" {
				values[key] = fallback
			} else {
				values[key] = strings.ToLower(strings.ReplaceAll(key, "_", "-"))
			}
		}
	}
	return values
}

func templateValidationSeedValuesForScenarioID(info TemplateInfo, scenarioID string) map[string]string {
	values := templateValidationSeedValues(info)
	if strings.TrimSpace(scenarioID) != "" {
		values["SCENARIO_ID"] = scenarioID
	}
	return values
}

func truncateForIssue(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "... (truncated)"
}

func validateTemplateSource(info TemplateInfo) []TemplateValidationIssue {
	if info.Missing {
		return nil
	}
	var issues []TemplateValidationIssue
	issues = append(issues, validateOrientationSource(info)...)
	_ = filepath.WalkDir(info.Path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Base(path) != "go.mod" {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(filepath.Base(path)),
				Message:  fmt.Sprintf("read go.mod: %v", readErr),
			})
			return nil
		}
		for _, target := range parseLocalReplaceTargets(string(data)) {
			if strings.Contains(target, "{{") {
				continue
			}
			rel, relErr := filepath.Rel(info.Path, path)
			if relErr != nil {
				rel = path
			}
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(rel),
				Message:  fmt.Sprintf("go.mod local replace target %q must use generator-computed placeholders", target),
			})
		}
		return nil
	})
	return issues
}

func validateOrientationSource(info TemplateInfo) []TemplateValidationIssue {
	if info.Manifest.Orientation == nil {
		return nil
	}
	orientation := info.Manifest.Orientation
	var issues []TemplateValidationIssue
	add := func(path, message string) {
		issues = append(issues, TemplateValidationIssue{Template: info.Name, Path: path, Message: message})
	}
	if strings.TrimSpace(info.Manifest.Version) == "" {
		add("template.json", "version is required when orientation is declared")
	}
	validateOrientationPaths(orientation, info.Manifest.StartDocument, add)
	seen := map[string]struct{}{}
	for index, step := range orientation.Steps {
		validateOrientationStepSource(index, step, seen, add)
	}
	validateOrientationCleanupSource(orientation.Finalize.Cleanup, add)
	return issues
}

type orientationSourceIssue func(path, message string)

func validateOrientationPaths(orientation *TemplateOrientation, manifestStartDocument string, add orientationSourceIssue) {
	if _, err := cleanScenarioRelativePath(orientation.CopyTo); err != nil {
		add("orientation.copyTo", err.Error())
	}
	startDocument := strings.TrimSpace(orientation.StartDocument)
	if startDocument == "" {
		startDocument = strings.TrimSpace(manifestStartDocument)
	}
	if startDocument == "" {
		return
	}
	if _, err := cleanScenarioRelativePath(startDocument); err != nil {
		add("orientation.startDocument", err.Error())
	}
}

func validateOrientationStepSource(index int, step TemplateOrientationStep, seen map[string]struct{}, add orientationSourceIssue) {
	stepPath := fmt.Sprintf("orientation.steps[%d]", index)
	id := strings.TrimSpace(step.ID)
	if id == "" {
		add(stepPath, "step id is required")
	} else if _, ok := seen[id]; ok {
		add(stepPath, fmt.Sprintf("duplicate step id %q", id))
	}
	seen[id] = struct{}{}
	if orientationStepRequired(step) && len(step.Checks) == 0 {
		add(stepPath, "required step must declare at least one check")
	}
	for checkIndex, check := range step.Checks {
		validateOrientationCheckSource(fmt.Sprintf("%s.checks[%d]", stepPath, checkIndex), check, add)
	}
}

func validateOrientationCheckSource(checkPath string, check TemplateOrientationCheck, add orientationSourceIssue) {
	if !validOrientationCheckKind(check.Kind) {
		add(checkPath, fmt.Sprintf("unknown check kind %q", check.Kind))
	}
	switch check.Kind {
	case "file_exists", "file_absent", "directory_exists":
		validateOrientationCheckPath(checkPath, check.Path, add)
	case "glob_present", "glob_absent", "glob_min_count":
		validateOrientationGlobCheckSource(checkPath, check, add)
	case "json_path_exists", "json_min_entries":
		validateOrientationJSONCheckSource(checkPath, check, add)
	case "text_contains", "text_absent":
		validateOrientationCheckPath(checkPath, check.Path, add)
		validateOrientationRequiredText(checkPath, check.Text, add)
	case "text_absent_tree":
		validateOrientationRequiredText(checkPath, check.Text, add)
	case "command":
		validateOrientationCommandSource(checkPath, check, add)
	}
}

func validateOrientationCheckPath(checkPath, path string, add orientationSourceIssue) {
	if _, err := cleanScenarioRelativePath(path); err != nil {
		add(checkPath+".path", err.Error())
	}
}

func validateOrientationGlobCheckSource(checkPath string, check TemplateOrientationCheck, add orientationSourceIssue) {
	if strings.TrimSpace(check.Pattern) == "" {
		add(checkPath+".pattern", "pattern is required")
	} else if _, err := cleanScenarioRelativePath(check.Pattern); err != nil {
		add(checkPath+".pattern", err.Error())
	}
	if check.Kind == "glob_min_count" && check.MinCount < 1 {
		add(checkPath+".minCount", "minCount must be greater than zero")
	}
}

func validateOrientationJSONCheckSource(checkPath string, check TemplateOrientationCheck, add orientationSourceIssue) {
	validateOrientationCheckPath(checkPath, check.Path, add)
	if strings.TrimSpace(check.Query) == "" {
		add(checkPath+".query", "query is required")
	}
	if check.Kind == "json_min_entries" && check.MinCount < 1 {
		add(checkPath+".minCount", "minCount must be greater than zero")
	}
}

func validateOrientationRequiredText(checkPath, text string, add orientationSourceIssue) {
	if strings.TrimSpace(text) == "" {
		add(checkPath+".text", "text is required")
	}
}

func validateOrientationCommandSource(checkPath string, check TemplateOrientationCheck, add orientationSourceIssue) {
	if strings.TrimSpace(check.Run) == "" {
		add(checkPath+".run", "run is required")
	}
	if strings.TrimSpace(check.Timeout) == "" {
		add(checkPath+".timeout", "timeout is required")
		return
	}
	if _, err := time.ParseDuration(check.Timeout); err != nil {
		add(checkPath+".timeout", fmt.Sprintf("invalid timeout: %v", err))
	}
}

func validateOrientationCleanupSource(cleanups []string, add orientationSourceIssue) {
	for _, cleanup := range cleanups {
		clean, err := cleanScenarioRelativePath(cleanup)
		if err != nil {
			add("orientation.finalize.cleanup", err.Error())
			continue
		}
		if isDurableOrientationCleanupTarget(clean) {
			add("orientation.finalize.cleanup", fmt.Sprintf("cleanup path %q targets durable scenario content", cleanup))
		}
	}
}

func orientationStepRequired(step TemplateOrientationStep) bool {
	return step.Required == nil || *step.Required
}

func validOrientationCheckKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "file_exists", "file_absent", "directory_exists", "glob_present", "glob_absent", "glob_min_count", "json_path_exists", "json_min_entries", "text_contains", "text_absent", "text_absent_tree", "command":
		return true
	default:
		return false
	}
}

func cleanScenarioRelativePath(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	clean := filepath.Clean(filepath.FromSlash(trimmed))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%q must be a scenario-relative path", value)
	}
	return clean, nil
}

func isDurableOrientationCleanupTarget(path string) bool {
	slash := filepath.ToSlash(path)
	if slash == "docs" || strings.HasPrefix(slash, "docs/") || slash == "DESIGN.md" || slash == "requirements" || strings.HasPrefix(slash, "requirements/") {
		return true
	}
	for _, prefix := range []string{"api/", "cli/", "ui/", "proto/", "runtime/"} {
		if strings.HasPrefix(slash, prefix) {
			return true
		}
	}
	return false
}

// validateRelocationProtoSources runs `buf lint` against template-side
// proto source folders so schema-level mistakes (missing package
// directive, syntax errors, naming convention violations) surface in
// template validation rather than after a real scenario generation.
//
// The "is this proto?" decision is heuristic: any relocation source that
// contains a .proto file is treated as one. Future non-proto relocations
// (e.g., scripts) won't be confused for protos because they won't have
// .proto files inside.
//
// Implementation note: `buf lint --path` only accepts paths inside the
// buf module (packages/proto/schemas/). The template's protos live
// outside that module pre-substitution, so we copy them into a temp
// subdirectory under schemas/ with template-validation seed values
// applied, lint there, and clean up. The temp directory name is
// prefixed with `.tmp-validate-` so it can never collide with a real
// scenario schema directory.
//
// Skipped entirely when deps.RunSubprocess is nil (mirrors the pattern
// used by validateGeneratedScenario for `go mod tidy`).
func validateRelocationProtoSources[C any](deps HandlerDeps[C], ctx C, info TemplateInfo) []TemplateValidationIssue {
	if deps.RunSubprocess == nil {
		return nil
	}
	if len(info.Manifest.Relocations) == 0 {
		return nil
	}
	repoRoot := deps.Root(ctx)
	protoPackageDir := filepath.Join(repoRoot, "packages", "proto")
	schemasDir := filepath.Join(protoPackageDir, "schemas")
	if _, err := os.Stat(schemasDir); err != nil {
		// No proto module in this repo (e.g., test fixtures with a
		// minimal repo-contract). The template's claim is that protos
		// belong here, so absence isn't a per-template issue — the
		// generator would fail at make-generate time, which is a
		// separate failure mode.
		return nil
	}
	var issues []TemplateValidationIssue
	values := templateValidationSeedValues(info)
	if id, ok := values["SCENARIO_ID"]; ok && strings.TrimSpace(id) != "" {
		values["SCENARIO_ID_SNAKE"] = strings.ReplaceAll(id, "-", "_")
	}
	for _, reloc := range info.Manifest.Relocations {
		from := strings.TrimSpace(reloc.From)
		if from == "" {
			continue
		}
		srcDir := filepath.Join(info.Path, filepath.FromSlash(from))
		if !directoryContainsProto(srcDir) {
			continue
		}
		tmpDir, err := os.MkdirTemp(schemasDir, ".tmp-validate-"+info.Name+"-")
		if err != nil {
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("create lint temp dir: %v", err),
			})
			continue
		}
		// Best-effort cleanup; lint failures are surfaced through
		// `issues` regardless of whether the cleanup succeeds.
		shouldClean := true
		defer func(path string, doClean *bool) {
			if *doClean {
				_ = os.RemoveAll(path)
			}
		}(tmpDir, &shouldClean)
		if err := copyRelocationTree(srcDir, tmpDir, values); err != nil {
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("substitute proto sources for lint: %v", err),
			})
			continue
		}
		// `buf lint --path` is now scoped to the temp dir which lives
		// inside the buf module, so the lint succeeds.
		//
		// `buf lint` writes lint diagnostics to stdout (one per line) and
		// exits non-zero. We capture both streams and prefer stdout for the
		// surfaced message because that's where the actionable detail lives.
		// The temp-dir path prefix in each diagnostic line is also stripped
		// so the surfaced message matches what an author would see if they
		// ran `buf lint` directly against the template's proto/.
		var stdout, stderr bytes.Buffer
		relTmp, err := filepath.Rel(protoPackageDir, tmpDir)
		if err != nil {
			relTmp = tmpDir
		}
		err = deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   "bash",
			Args:   []string{"-lc", fmt.Sprintf("buf lint --path %s", shellQuote(relTmp))},
			Dir:    protoPackageDir,
			Env:    deps.CommandEnv(ctx),
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if err != nil {
			msg := strings.TrimSpace(stdout.String())
			if msg == "" {
				msg = strings.TrimSpace(stderr.String())
			}
			if msg == "" {
				msg = err.Error()
			}
			// Strip the temp-dir prefix so diagnostics read as if buf lint
			// had been run against the template's source proto/ directly.
			fromPrefix := strings.TrimRight(filepath.ToSlash(from), "/") + "/"
			msg = strings.ReplaceAll(msg, filepath.ToSlash(relTmp)+"/", fromPrefix)
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("buf lint failed: %s", msg),
			})
		}
	}
	return issues
}

// directoryContainsProto reports whether the directory tree rooted at
// path contains any .proto files. Walks until the first match.
func directoryContainsProto(path string) bool {
	stat, err := os.Stat(path)
	if err != nil || !stat.IsDir() {
		return false
	}
	found := false
	_ = filepath.WalkDir(path, func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(p), ".proto") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// shellQuote returns a single-quoted shell argument that survives buf's
// `bash -lc` invocation. Used for absolute paths that may contain
// shell-special characters; deliberately conservative.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func validateGeneratedScenario(destination string, runCommands bool, run func(scenarioexec.SubprocessSpec) error, templateName string, manifest TemplateManifest) []TemplateValidationIssue {
	var issues []TemplateValidationIssue
	issues = append(issues, validateGeneratedStartDocument(destination, templateName, manifest)...)
	_ = filepath.WalkDir(destination, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Base(path) != "go.mod" {
			return err
		}
		moduleDir := filepath.Dir(path)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, TemplateValidationIssue{
				Template: templateName,
				Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
				Message:  fmt.Sprintf("read generated go.mod: %v", readErr),
			})
			return nil
		}
		for _, target := range parseLocalReplaceTargets(string(data)) {
			resolved := target
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(moduleDir, filepath.FromSlash(target))
			}
			if _, statErr := os.Stat(filepath.Clean(resolved)); statErr != nil {
				issues = append(issues, TemplateValidationIssue{
					Template: templateName,
					Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
					Message:  fmt.Sprintf("go.mod replace target %q does not resolve from generated module: %v", target, statErr),
				})
			}
		}
		if runCommands && moduleHasGoFiles(moduleDir) {
			if execErr := run(scenarioexec.SubprocessSpec{
				Name: "bash",
				Args: []string{"-lc", "GOWORK=off go mod tidy"},
				Dir:  moduleDir,
			}); execErr != nil {
				issues = append(issues, TemplateValidationIssue{
					Template: templateName,
					Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
					Message:  fmt.Sprintf("generated module validation failed: %v", execErr),
				})
			}
		}
		return nil
	})
	return issues
}

func validateGeneratedStartDocument(destination, templateName string, manifest TemplateManifest) []TemplateValidationIssue {
	startDocument := strings.TrimSpace(manifest.StartDocument)
	if startDocument == "" {
		return nil
	}
	cleanPath := filepath.Clean(filepath.FromSlash(startDocument))
	if cleanPath == "." || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || cleanPath == ".." {
		return []TemplateValidationIssue{{
			Template: templateName,
			Path:     startDocument,
			Message:  "startDocument must be a scenario-relative path",
		}}
	}
	stat, err := os.Stat(filepath.Join(destination, cleanPath))
	if err != nil {
		return []TemplateValidationIssue{{
			Template: templateName,
			Path:     filepath.ToSlash(cleanPath),
			Message:  fmt.Sprintf("startDocument is declared but missing from generated scenario: %v", err),
		}}
	}
	if stat.IsDir() {
		return []TemplateValidationIssue{{
			Template: templateName,
			Path:     filepath.ToSlash(cleanPath),
			Message:  "startDocument must point to a file, not a directory",
		}}
	}
	return nil
}

var goModReplaceLinePattern = regexp.MustCompile(`^\s*([A-Za-z0-9._/\-{}]+)(?:\s+[^\s]+)?\s*=>\s*([^\s]+)(?:\s+[^\s]+)?\s*(?://.*)?$`)

func parseLocalReplaceTargets(content string) []string {
	var targets []string
	var inReplaceBlock bool
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		switch line {
		case "replace (":
			inReplaceBlock = true
			continue
		case ")":
			inReplaceBlock = false
			continue
		}
		switch {
		case strings.HasPrefix(line, "replace "):
			if target, ok := parseGoReplaceTarget(strings.TrimSpace(strings.TrimPrefix(line, "replace "))); ok {
				targets = append(targets, target)
			}
		case inReplaceBlock:
			if target, ok := parseGoReplaceTarget(line); ok {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

func parseGoReplaceTarget(line string) (string, bool) {
	matches := goModReplaceLinePattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return "", false
	}
	target := strings.TrimSpace(matches[2])
	if target == "" {
		return "", false
	}
	if !(strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/") || strings.Contains(target, "{{")) {
		return "", false
	}
	return target, true
}

func moduleHasGoFiles(moduleDir string) bool {
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

func formatTemplateValidationIssues(issues []TemplateValidationIssue) string {
	if len(issues) == 0 {
		return ""
	}
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		line := issue.Template
		if strings.TrimSpace(issue.Path) != "" {
			line += " [" + issue.Path + "]"
		}
		line += ": " + issue.Message
		lines = append(lines, line)
	}
	return strings.Join(lines, "; ")
}

func runTemplateHooks[C any](deps HandlerDeps[C], ctx C, destination string, manifest TemplateManifest, output io.Writer) error {
	if output == nil {
		output = io.Discard
	}
	if len(manifest.PostHooks) == 0 {
		_, _ = fmt.Fprintln(output, "No post hooks defined for this template")
		return nil
	}
	for index, hook := range manifest.PostHooks {
		description := strings.TrimSpace(hook.Description)
		if description == "" {
			description = hook.Cmd
		}
		_, _ = fmt.Fprintf(output, "[Hook %d] %s\n", index+1, description)
		cwd := destination
		if strings.TrimSpace(hook.Cwd) != "" && hook.Cwd != "." {
			cwd = filepath.Join(destination, filepath.FromSlash(hook.Cwd))
		}
		if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   "bash",
			Args:   []string{"-lc", hook.Cmd},
			Dir:    cwd,
			Env:    deps.CommandEnv(ctx),
			Stdout: output,
			Stderr: deps.Stderr(ctx),
		}); err != nil {
			return err
		}
	}
	return nil
}

func randomTemplateToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func copyStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func FormatTemplateRequiredFlags(manifest TemplateManifest) string {
	keys := make([]string, 0, len(manifest.RequiredVars))
	for key := range manifest.RequiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return " --id <slug>"
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		flag := manifest.RequiredVars[key].Flag
		if flag == "" {
			flag = strings.ToLower(key)
		}
		parts = append(parts, fmt.Sprintf(" --%s <%s>", flag, strings.ToLower(key)))
	}
	return strings.Join(parts, "")
}
