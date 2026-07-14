package components

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	ingestReleaseVersion = "0.1.0"
	ingestDraftVersion   = "0.1.0-draft.1"
	ingestChecklistPath  = "docs/guides/de-scenario-ification-checklist.md"
)

// IngestComponent turns a scenario-local TSX file into an indexed catalog
// entry. A release baseline is created because the manifest/index contract
// requires `latest` to target a released version; callers work on the returned
// draft version until it is ready to promote.
func (s *service) IngestComponent(ctx context.Context, in IngestComponentInput) (IngestComponentResult, error) {
	if s.ingest == nil {
		return IngestComponentResult{}, fmt.Errorf("components service: scenario source reader not configured")
	}
	in.Scenario = strings.TrimSpace(in.Scenario)
	in.SourceFile = strings.TrimSpace(in.SourceFile)
	in.Slug = normalizeSlug(in.Slug)
	if in.Scenario == "" || in.SourceFile == "" || in.Slug == "" {
		return IngestComponentResult{}, fmt.Errorf("scenario, source_file, and slug are required")
	}
	if !strings.HasSuffix(in.SourceFile, ".tsx") {
		return IngestComponentResult{}, fmt.Errorf("source_file must be a .tsx file")
	}
	unit, err := readIngestUnit(ctx, s.ingest, in.Scenario, in.SourceFile, in.SourceFiles)
	if err != nil {
		return IngestComponentResult{}, err
	}
	raw := unit[0].Content
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = in.Slug
	}
	description := strings.TrimSpace(in.Description)
	if description == "" {
		description = fmt.Sprintf("Ingested from %s:%s", in.Scenario, in.SourceFile)
	}
	libraryID := "react-component-library:" + in.Slug
	releaseVersion := strings.TrimSpace(in.Version)
	if releaseVersion == "" {
		releaseVersion = ingestReleaseVersion
	}
	if err := validateVersionToken(releaseVersion); err != nil || strings.Contains(releaseVersion, "-") {
		return IngestComponentResult{}, fmt.Errorf("ingest version must be a released semver-like value")
	}
	draftVersion := releaseVersion + "-draft.1"
	findings := inspectIngestSource(raw, in.SourceFile)
	releaseFiles := ingestFilesForVersion(unit, libraryID, displayName, description, releaseVersion, in.Tags, in.Scenario, in.SourceFile)
	releaseSource := releaseFiles[0].Content
	findings = append(findings, BehaviorLossFindings(joinIngestUnit(unit), joinIngestUnit(releaseFiles), in.SourceFile)...)
	report := IngestParityReport{OriginFiles: ingestFilePaths(unit), HarvestedFiles: ingestFilePaths(releaseFiles), Findings: behaviorFindings(findings)}
	draftFiles := ingestFilesForVersion(unit, libraryID, displayName, description, draftVersion, in.Tags, in.Scenario, in.SourceFile)
	draftSource := draftFiles[0].Content
	existing, existingErr := s.repo.GetByLibraryID(ctx, libraryID)
	if existingErr != nil && !errors.As(existingErr, &ErrComponentNotFound{}) {
		return IngestComponentResult{}, existingErr
	}
	if existingErr == nil {
		draft, err := s.CreateComponentVersion(ctx, CreateComponentVersionInput{ComponentID: existing.ID, Version: draftVersion, Intent: VersionIntentDraft,
			FileName: draftFiles[0].Path, Source: draftSource, Files: draftFiles, ParityReport: &report})
		if err != nil {
			return IngestComponentResult{}, err
		}
		return IngestComponentResult{Component: draft.Component, ManifestPath: existing.ManifestPath, SourcePath: draft.SourcePath, DraftVersion: draftVersion, Findings: findings, ParityReport: report, ChecklistPath: ingestChecklistPath}, nil
	}
	created, err := s.InitializeComponent(ctx, InitializeComponentInput{
		LibraryID: libraryID, Slug: in.Slug, DisplayName: displayName,
		Description: description, Tags: in.Tags, Slot: in.Slot, InitialVersion: releaseVersion,
		FileName: releaseFiles[0].Path, InitialSource: releaseSource, InitialFiles: releaseFiles,
	})
	if err != nil {
		return IngestComponentResult{}, err
	}
	draft, err := s.CreateComponentVersion(ctx, CreateComponentVersionInput{
		ComponentID: created.Component.ID, Version: draftVersion, Intent: VersionIntentDraft,
		FileName: draftFiles[0].Path, Source: draftSource, Files: draftFiles, ParityReport: &report,
	})
	if err != nil {
		return IngestComponentResult{}, err
	}
	return IngestComponentResult{
		Component: draft.Component, ManifestPath: created.ManifestPath, SourcePath: draft.SourcePath,
		DraftVersion: draftVersion, Findings: findings, ParityReport: report, ChecklistPath: ingestChecklistPath,
	}, nil
}

func ingestFilePaths(files []ComponentVersionFile) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func behaviorFindings(findings []IngestFinding) []IngestFinding {
	var out []IngestFinding
	for _, finding := range findings {
		if finding.Code == "behavior-lost" {
			out = append(out, finding)
		}
	}
	return out
}

var relativeImportRE = regexp.MustCompile(`(?:\bfrom\s*|\bimport\s*\()\s*["'](\.[^"']+)["']`)

// readIngestUnit reads the entry plus a transitive closure of sibling TS/TSX
// imports. Library version folders are deliberately flat, therefore a source
// dependency in another directory is rejected instead of silently copied to a
// different module path and broken at preview/adoption time.
func readIngestUnit(ctx context.Context, reader ScenarioSourceReader, scenario, entry string, explicit []string) ([]ComponentVersionFile, error) {
	queued := append([]string{entry}, explicit...)
	seen := map[string]bool{}
	byName := map[string]ComponentVersionFile{}
	for len(queued) > 0 {
		sourcePath := path.Clean(strings.TrimSpace(queued[0]))
		queued = queued[1:]
		if sourcePath == "." || strings.HasPrefix(sourcePath, "../") || seen[sourcePath] {
			continue
		}
		seen[sourcePath] = true
		if !strings.HasSuffix(sourcePath, ".ts") && !strings.HasSuffix(sourcePath, ".tsx") {
			return nil, fmt.Errorf("ingest source %q must be a .ts or .tsx file", sourcePath)
		}
		raw, err := reader.Read(ctx, scenario, sourcePath)
		if err != nil {
			return nil, fmt.Errorf("read ingest source %q: %w", sourcePath, err)
		}
		name := path.Base(sourcePath)
		if previous, ok := byName[name]; ok && previous.Content != string(raw) {
			return nil, fmt.Errorf("ingest source %q collides with another companion named %q; nested version folders are not supported", sourcePath, name)
		}
		body := string(raw)
		for _, match := range relativeImportRE.FindAllStringSubmatch(string(raw), -1) {
			resolved, err := resolveRelativeIngestImport(ctx, reader, scenario, path.Dir(sourcePath), match[1])
			if err != nil {
				return nil, err
			}
			// Version folders intentionally have no subdirectories. Preserve the
			// dependency while rewriting its module specifier to the flattened
			// sibling path used in the catalog/adoption unit.
			module := strings.TrimSuffix(path.Base(resolved), path.Ext(resolved))
			body = strings.ReplaceAll(body, match[1], "./"+module)
			queued = append(queued, resolved)
		}
		byName[name] = ComponentVersionFile{Path: name, Content: body, IsEntry: sourcePath == entry}
	}
	unit := make([]ComponentVersionFile, 0, len(byName))
	for _, file := range byName {
		unit = append(unit, file)
	}
	sort.Slice(unit, func(i, j int) bool { return unit[i].Path < unit[j].Path })
	for i := range unit {
		if unit[i].IsEntry {
			unit[0], unit[i] = unit[i], unit[0]
			break
		}
	}
	return unit, nil
}

func resolveRelativeIngestImport(ctx context.Context, reader ScenarioSourceReader, scenario, dir, specifier string) (string, error) {
	base := path.Clean(path.Join(dir, specifier))
	if strings.HasPrefix(base, "../") {
		return "", fmt.Errorf("relative import %q escapes its scenario", specifier)
	}
	candidates := []string{base}
	if path.Ext(base) == "" {
		candidates = append(candidates, base+".ts", base+".tsx", path.Join(base, "index.ts"), path.Join(base, "index.tsx"))
	}
	for _, candidate := range candidates {
		if _, err := reader.Read(ctx, scenario, candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("relative import %q from %q could not be read as a .ts/.tsx companion", specifier, dir)
}

func ingestFilesForVersion(unit []ComponentVersionFile, libraryID, displayName, description, version string, tags []string, scenario, sourceFile string) []ComponentVersionFile {
	files := append([]ComponentVersionFile(nil), unit...)
	for i := range files {
		if files[i].IsEntry {
			files[i].Content = withIngestHeader(files[i].Content, libraryID, displayName, description, version, tags, scenario, sourceFile)
		}
	}
	return files
}

func joinIngestUnit(files []ComponentVersionFile) string {
	parts := make([]string, 0, len(files))
	for _, file := range files {
		parts = append(parts, file.Content)
	}
	return strings.Join(parts, "\n")
}

func withIngestHeader(source, libraryID, displayName, description, version string, tags []string, scenario, sourceFile string) string {
	clean := func(value string) string { return strings.ReplaceAll(strings.TrimSpace(value), "*/", "") }
	header := fmt.Sprintf("/**\n * @libraryId %s\n * @displayName %s\n * @description %s\n * @version %s\n * @tags %s\n * @originScenario %s\n * @originPath %s\n * @warning Ingested by React Component Library. Preserve this provenance header.\n */\n",
		clean(libraryID), clean(displayName), clean(description), clean(version), tagsJSON(cleanTags(tags)), clean(scenario), clean(sourceFile))
	return header + strings.TrimLeft(source, "\n")
}

func inspectIngestSource(source, sourceFile string) []IngestFinding {
	var findings []IngestFinding
	add := func(code, message string) {
		findings = append(findings, IngestFinding{Code: code, Message: message, SourceFile: sourceFile})
	}
	if strings.Contains(source, "from \"@/") || strings.Contains(source, "from '../") || strings.Contains(source, "from \"../") {
		add("app-import", "Replace scenario-relative or application-alias imports with component-local props or declared dependencies.")
	}
	if strings.Contains(source, "#") || strings.Contains(source, "rgb(") || strings.Contains(source, "bg-red-") || strings.Contains(source, "text-red-") {
		add("token-violation", "Replace hard-coded colors with design-system token classes or CSS variables.")
	}
	if strings.Contains(source, "useNavigate(") || strings.Contains(source, "useRouter(") || strings.Contains(source, "useLocation(") {
		add("app-context", "Move router or scenario-context behavior behind explicit component props.")
	}
	if !strings.Contains(source, "export default") && !strings.Contains(source, "export function") {
		add("missing-export", "Expose the component through a default or named React component export.")
	}
	return findings
}
