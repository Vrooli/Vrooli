package components

import (
	"context"
	"fmt"
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
	raw, err := s.ingest.Read(ctx, in.Scenario, in.SourceFile)
	if err != nil {
		return IngestComponentResult{}, fmt.Errorf("read ingest source: %w", err)
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = in.Slug
	}
	description := strings.TrimSpace(in.Description)
	if description == "" {
		description = fmt.Sprintf("Ingested from %s:%s", in.Scenario, in.SourceFile)
	}
	libraryID := "react-component-library:" + in.Slug
	findings := inspectIngestSource(string(raw), in.SourceFile)
	releaseSource := withIngestHeader(string(raw), libraryID, displayName, description, ingestReleaseVersion, in.Tags, in.Scenario, in.SourceFile)
	created, err := s.InitializeComponent(ctx, InitializeComponentInput{
		LibraryID: libraryID, Slug: in.Slug, DisplayName: displayName,
		Description: description, Tags: in.Tags, Slot: in.Slot, InitialVersion: ingestReleaseVersion,
		FileName: in.Slug + ".tsx", InitialSource: releaseSource,
	})
	if err != nil {
		return IngestComponentResult{}, err
	}
	draftSource := withIngestHeader(string(raw), libraryID, displayName, description, ingestDraftVersion, in.Tags, in.Scenario, in.SourceFile)
	draft, err := s.CreateComponentVersion(ctx, CreateComponentVersionInput{
		ComponentID: created.Component.ID, Version: ingestDraftVersion, Intent: VersionIntentDraft,
		FileName: in.Slug + ".tsx", Source: draftSource,
	})
	if err != nil {
		return IngestComponentResult{}, err
	}
	return IngestComponentResult{
		Component: draft.Component, ManifestPath: created.ManifestPath, SourcePath: draft.SourcePath,
		DraftVersion: ingestDraftVersion, Findings: findings, ChecklistPath: ingestChecklistPath,
	}, nil
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
