package templatecontracts

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

type TemplateVar struct {
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type TemplateHook struct {
	Description string `json:"description,omitempty"`
	Cmd         string `json:"cmd,omitempty"`
	Cwd         string `json:"cwd,omitempty"`
}

// TemplateRelocation declares an out-of-tree placement performed by the
// generator after the in-tree copy. The directory at From (template-relative)
// is rendered (with placeholder substitution applied to both file content
// and path components) into To (repo-root-relative; may contain placeholders).
//
// Post commands run from the repo root after every relocation in the manifest
// has been applied — useful for codegen steps that depend on the relocated
// content (e.g., regenerating proto artifacts in packages/proto/).
//
// The From directory is automatically excluded from the in-tree copy that
// writes into the scenario destination, so the same source folder doesn't
// end up in two places.
type TemplateRelocation struct {
	Description string         `json:"description,omitempty"`
	From        string         `json:"from"`
	To          string         `json:"to"`
	Post        []TemplateHook `json:"post,omitempty"`
}

type TemplateDesign struct {
	Adapter  string `json:"adapter,omitempty"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type TemplateManifest struct {
	Name          string                 `json:"name,omitempty"`
	Version       string                 `json:"version,omitempty"`
	DisplayName   string                 `json:"displayName,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Stack         []string               `json:"stack,omitempty"`
	StartDocument string                 `json:"startDocument,omitempty"`
	Design        TemplateDesign         `json:"design,omitempty"`
	Orientation   *TemplateOrientation   `json:"orientation,omitempty"`
	RequiredVars  map[string]TemplateVar `json:"requiredVars,omitempty"`
	OptionalVars  map[string]TemplateVar `json:"optionalVars,omitempty"`
	Docs          map[string]string      `json:"docs,omitempty"`
	CopyExcludes  []string               `json:"copyExcludes,omitempty"`
	PostHooks     []TemplateHook         `json:"postHooks,omitempty"`
	Relocations   []TemplateRelocation   `json:"relocations,omitempty"`
	ExampleDomain *TemplateExampleDomain `json:"exampleDomain,omitempty"`
}

// TemplateExampleDomain declares the template's illustrative example domain
// (the `notes` worked vertical slice) so it can be removed mechanically and
// verifiably by `template-manager detemplate`. There is exactly one marker
// vocabulary, placed three ways:
//
//   - doc block: <!-- EXAMPLE-DOMAIN:<marker> START --> ... <!-- EXAMPLE-DOMAIN:<marker> END -->
//   - whole file/dir: enumerated in Paths (template-relative)
//   - registration line: a trailing `EXAMPLE-DOMAIN:<marker>` comment marker
//
// Marker is the example domain's slug (e.g. "notes"); detemplate reads it so
// the command stays domain-name-agnostic.
type TemplateExampleDomain struct {
	Marker    string                   `json:"marker,omitempty"`
	Paths     []string                 `json:"paths,omitempty"`
	JSONPrune []TemplateJSONPruneEntry `json:"jsonPrune,omitempty"`
}

// TemplateJSONPruneEntry removes example-domain content from a hand-authored
// JSON file that cannot carry comment markers (i18n locales, the CLI
// manifest). Keys lists dotted object key-paths to delete; ArrayMatch removes
// array elements whose object fields all equal the given values. The file is
// rewritten preserving key order and UTF-8 content.
type TemplateJSONPruneEntry struct {
	File       string                   `json:"file"`
	Keys       []string                 `json:"keys,omitempty"`
	ArrayMatch []TemplateJSONArrayMatch `json:"arrayMatch,omitempty"`
}

// TemplateJSONArrayMatch selects array elements to delete: at the dotted Path
// to an array, remove every element that is an object matching all Where
// field==value pairs.
type TemplateJSONArrayMatch struct {
	Path  string            `json:"path"`
	Where map[string]string `json:"where,omitempty"`
}

type TemplateOrientation struct {
	Version       string                      `json:"version,omitempty"`
	CopyTo        string                      `json:"copyTo,omitempty"`
	StartDocument string                      `json:"startDocument,omitempty"`
	Finalize      TemplateOrientationFinalize `json:"finalize,omitempty"`
	Steps         []TemplateOrientationStep   `json:"steps,omitempty"`
}

type TemplateOrientationFinalize struct {
	Cleanup []string `json:"cleanup,omitempty"`
	Message string   `json:"message,omitempty"`
}

type TemplateOrientationStep struct {
	ID          string                     `json:"id"`
	Title       string                     `json:"title,omitempty"`
	Description string                     `json:"description,omitempty"`
	Docs        []string                   `json:"docs,omitempty"`
	Required    *bool                      `json:"required,omitempty"`
	Checks      []TemplateOrientationCheck `json:"checks,omitempty"`
}

type TemplateOrientationCheck struct {
	Kind     string `json:"kind"`
	Path     string `json:"path,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	Query    string `json:"query,omitempty"`
	Text     string `json:"text,omitempty"`
	MinCount int    `json:"minCount,omitempty"`
	Run      string `json:"run,omitempty"`
	Timeout  string `json:"timeout,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type TemplateInfo struct {
	Name     string
	Path     string
	Manifest TemplateManifest
	Missing  bool
}

type GenerateOptions struct {
	Destination string
	Design      string
	Force       bool
	DryRun      bool
	RunHooks    bool
	Values      map[string]string
}

type (
	TemplateListRequest struct{}
	TemplateShowRequest struct{ Name string }
	GenerateRequest     struct {
		TemplateInfo TemplateInfo
		Options      GenerateOptions
	}
)

type TemplateValidationMode string

const (
	TemplateValidationModeShallow TemplateValidationMode = "shallow"
	TemplateValidationModeDeep    TemplateValidationMode = "deep"
)

type TemplateValidationWarningPolicy string

const (
	TemplateValidationWarningPolicyIgnore TemplateValidationWarningPolicy = "ignore"
	TemplateValidationWarningPolicyReport TemplateValidationWarningPolicy = "report"
	TemplateValidationWarningPolicyFail   TemplateValidationWarningPolicy = "fail"
)

const DefaultTemplateValidationTestPreset = "comprehensive"

type TemplateValidateRequest struct {
	Mode          TemplateValidationMode
	TemplateName  string
	RetainTemp    bool
	TestPreset    string
	WarningPolicy TemplateValidationWarningPolicy
}

// TemplateDriftRequest selects which scenarios to audit and how verbose the
// output should be. Exactly one of Scenario or All is set (parser enforces).
type TemplateDriftRequest struct {
	Scenario string
	All      bool
	Verbose  bool
	JSON     bool
}

// TemplateDriftStatus classifies the outcome for a single scenario. Empty
// values default to "ok" in renderers.
type TemplateDriftStatus string

const (
	TemplateDriftStatusOK            TemplateDriftStatus = "ok"
	TemplateDriftStatusDrifted       TemplateDriftStatus = "drifted"
	TemplateDriftStatusMissingHashes TemplateDriftStatus = "no_recorded_hashes"
	TemplateDriftStatusTemplateGone  TemplateDriftStatus = "template_not_found"
	TemplateDriftStatusNoProvenance  TemplateDriftStatus = "no_provenance"
	TemplateDriftStatusHashError     TemplateDriftStatus = "hash_error"
)

// TemplateDriftFileDiff describes one inherited file whose bytes differ
// between the current template and the scenario's copy. Only populated when
// --verbose was requested and content drift was detected.
type TemplateDriftFileDiff struct {
	Path   string `json:"path"`
	Reason string `json:"reason"` // "modified" | "added_in_template" | "removed_from_scenario"
}

type TemplateDriftScenarioReport struct {
	Scenario         string                  `json:"scenario"`
	TemplateID       string                  `json:"templateId,omitempty"`
	RecordedVersion  string                  `json:"recordedVersion,omitempty"`
	CurrentVersion   string                  `json:"currentVersion,omitempty"`
	RecordedManifest string                  `json:"recordedManifestSha,omitempty"`
	CurrentManifest  string                  `json:"currentManifestSha,omitempty"`
	RecordedContent  string                  `json:"recordedContentSha,omitempty"`
	CurrentContent   string                  `json:"currentContentSha,omitempty"`
	ManifestDrifted  bool                    `json:"manifestDrifted,omitempty"`
	ContentDrifted   bool                    `json:"contentDrifted,omitempty"`
	Status           TemplateDriftStatus     `json:"status"`
	Message          string                  `json:"message,omitempty"`
	FileDiffs        []TemplateDriftFileDiff `json:"fileDiffs,omitempty"`
}

func (r TemplateDriftScenarioReport) Drifted() bool {
	return r.ManifestDrifted || r.ContentDrifted
}

type TemplateDriftReport struct {
	Scenarios []TemplateDriftScenarioReport `json:"scenarios"`
}

func (r TemplateDriftReport) AnyDrifted() bool {
	for _, s := range r.Scenarios {
		if s.Drifted() {
			return true
		}
	}
	return false
}

func RenderTemplateDriftResponse(w io.Writer, format cliout.Format, report TemplateDriftReport) error {
	if format == cliout.FormatJSON {
		return writeScenarioTemplateDriftJSON(w, report)
	}
	if len(report.Scenarios) == 0 {
		_, _ = fmt.Fprintln(w, "No scenarios with template provenance found.")
		return nil
	}
	for i, s := range report.Scenarios {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		writeTemplateDriftScenario(w, s)
	}
	return nil
}

func writeTemplateDriftScenario(w io.Writer, s TemplateDriftScenarioReport) {
	header := s.Scenario
	if s.TemplateID != "" {
		header = fmt.Sprintf("%s (%s)", s.Scenario, s.TemplateID)
	}
	switch s.Status {
	case TemplateDriftStatusNoProvenance:
		_, _ = fmt.Fprintf(w, "%s: no template provenance recorded\n", header)
		return
	case TemplateDriftStatusMissingHashes:
		_, _ = fmt.Fprintf(w, "%s: provenance recorded without hashes (generated before drift tracking)\n", header)
		return
	case TemplateDriftStatusTemplateGone:
		_, _ = fmt.Fprintf(w, "%s: template %q not found in current tree\n", header, s.TemplateID)
		return
	case TemplateDriftStatusHashError:
		_, _ = fmt.Fprintf(w, "%s: %s\n", header, s.Message)
		return
	}
	if !s.Drifted() {
		_, _ = fmt.Fprintf(w, "%s: in sync with template\n", header)
		return
	}
	_, _ = fmt.Fprintf(w, "%s: drifted\n", header)
	if s.RecordedVersion != "" || s.CurrentVersion != "" {
		_, _ = fmt.Fprintf(w, "  version:  recorded=%s current=%s\n", emptyDash(s.RecordedVersion), emptyDash(s.CurrentVersion))
	}
	if s.ManifestDrifted {
		_, _ = fmt.Fprintf(w, "  manifest: recorded=%s current=%s\n", shortSha(s.RecordedManifest), shortSha(s.CurrentManifest))
	}
	if s.ContentDrifted {
		_, _ = fmt.Fprintf(w, "  content:  recorded=%s current=%s\n", shortSha(s.RecordedContent), shortSha(s.CurrentContent))
	}
	for _, fd := range s.FileDiffs {
		_, _ = fmt.Fprintf(w, "    %-22s %s\n", fd.Reason, fd.Path)
	}
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func shortSha(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}

type TemplateCleanupRequest struct {
	DryRun          bool
	OlderThan       string
	IncludeRetained bool
	RunID           string
}

// ResolvedRelocation captures a relocation after placeholder substitution,
// so callers (and dry-run output) can show exactly where each From folder
// would land before any disk writes happen.
type ResolvedRelocation struct {
	Description string         `json:"description,omitempty"`
	From        string         `json:"from"` // template-relative source dir
	To          string         `json:"to"`   // absolute path under repo root after substitution
	Post        []TemplateHook `json:"post,omitempty"`
}

type ResolvedDesignCopy struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ResolvedDesign struct {
	KitID     string               `json:"kitId,omitempty"`
	KitName   string               `json:"kitName,omitempty"`
	Version   string               `json:"version,omitempty"`
	AdapterID string               `json:"adapterId,omitempty"`
	Copies    []ResolvedDesignCopy `json:"copies,omitempty"`
}

type GenerateResult struct {
	TemplateName string
	DisplayName  string
	Destination  string
	Values       map[string]string
	Manifest     TemplateManifest
	Design       ResolvedDesign
	Relocations  []ResolvedRelocation
	Provenance   GenerationProvenance
	DryRun       bool
	RunHooks     bool
}

type GenerationProvenance struct {
	Template    GenerationTemplate `json:"template,omitempty"`
	GeneratedAt string             `json:"generated_at,omitempty"`
	Design      GenerationDesign   `json:"design,omitempty"`
	ManifestSha string             `json:"manifest_sha,omitempty"`
	ContentSha  string             `json:"content_sha,omitempty"`
}

type GenerationTemplate struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

type GenerationDesign struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Adapter string `json:"adapter,omitempty"`
}

type OrientationRequest struct {
	Name     string
	JSON     bool
	Finalize bool
}

type OrientationReport struct {
	Scenario         string                  `json:"scenario"`
	ScenarioPath     string                  `json:"scenarioPath,omitempty"`
	OrientationPath  string                  `json:"orientationPath,omitempty"`
	Finalized        bool                    `json:"finalized,omitempty"`
	Template         GenerationTemplate      `json:"template,omitempty"`
	Design           GenerationDesign        `json:"design,omitempty"`
	StartDocument    string                  `json:"startDocument,omitempty"`
	Completed        int                     `json:"completed"`
	Required         int                     `json:"required"`
	Steps            []OrientationStepReport `json:"steps,omitempty"`
	NextStep         *OrientationStepReport  `json:"nextStep,omitempty"`
	Message          string                  `json:"message,omitempty"`
	FinalizeRequired bool                    `json:"finalizeRequired,omitempty"`
}

type OrientationStepReport struct {
	ID          string                   `json:"id"`
	Title       string                   `json:"title,omitempty"`
	Description string                   `json:"description,omitempty"`
	Docs        []string                 `json:"docs,omitempty"`
	Required    bool                     `json:"required"`
	Complete    bool                     `json:"complete"`
	Checks      []OrientationCheckReport `json:"checks,omitempty"`
}

type OrientationCheckReport struct {
	Kind     string `json:"kind"`
	Label    string `json:"label,omitempty"`
	Passed   bool   `json:"passed"`
	Skipped  bool   `json:"skipped,omitempty"`
	Message  string `json:"message,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type TemplateValidationIssue struct {
	Template string `json:"template"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type TemplateValidationWarning struct {
	Message      string `json:"message"`
	Source       string `json:"source,omitempty"`
	LogPath      string `json:"logPath,omitempty"`
	ArtifactPath string `json:"artifactPath,omitempty"`
}

type TemplateValidationPhaseWarningSummary struct {
	Name     string                      `json:"name"`
	Count    int                         `json:"count"`
	Warnings []TemplateValidationWarning `json:"warnings,omitempty"`
}

type TemplateValidationWarningSummary struct {
	Total  int                                     `json:"total"`
	Phases []TemplateValidationPhaseWarningSummary `json:"phases,omitempty"`
}

type TemplateValidationDeepRun struct {
	Template            string                           `json:"template"`
	RunID               string                           `json:"runId,omitempty"`
	ScenarioID          string                           `json:"scenarioId,omitempty"`
	ScenarioPath        string                           `json:"scenarioPath,omitempty"`
	TempRoot            string                           `json:"tempRoot,omitempty"`
	TestPreset          string                           `json:"testPreset,omitempty"`
	WarningSummary      TemplateValidationWarningSummary `json:"warningSummary"`
	RetainedTemp        bool                             `json:"retainedTemp,omitempty"`
	CleanupStatus       string                           `json:"cleanupStatus,omitempty"`
	RelocationArtifacts []string                         `json:"relocationArtifacts,omitempty"`
	CleanupCommand      string                           `json:"cleanupCommand,omitempty"`
}

type TemplateValidationReport struct {
	Mode           TemplateValidationMode           `json:"mode,omitempty"`
	TemplateName   string                           `json:"templateName,omitempty"`
	TestPreset     string                           `json:"testPreset,omitempty"`
	WarningPolicy  TemplateValidationWarningPolicy  `json:"warningPolicy,omitempty"`
	WarningSummary TemplateValidationWarningSummary `json:"warningSummary"`
	Count          int                              `json:"count"`
	DeepRuns       []TemplateValidationDeepRun      `json:"deepRuns,omitempty"`
	Issues         []TemplateValidationIssue        `json:"issues,omitempty"`
}

type TemplateCleanupResult = templatevalidation.CleanupResult

func RenderTemplateListResponse(w io.Writer, format cliout.Format, templates []TemplateInfo) error {
	if format == cliout.FormatJSON {
		return writeScenarioTemplateListJSON(w, templates)
	}
	rows := make([][]string, 0, len(templates))
	for _, item := range templates {
		required := formatTemplateRequiredVars(item.Manifest)
		if item.Missing {
			required = "?"
		}
		display := item.Manifest.DisplayName
		if display == "" {
			display = "(template.json missing)"
		}
		rows = append(rows, []string{item.Name, display, item.Manifest.Version, required})
	}
	_ = cliout.RenderTable(w, []string{"Name", "Display Name", "Version", "Required Vars"}, rows)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Tip: template-manager registry show <name>")
	return nil
}

func RenderTemplateShowResponse(w io.Writer, format cliout.Format, info TemplateInfo) error {
	_ = format
	manifest := info.Manifest
	title := manifest.DisplayName
	if title == "" {
		title = info.Name
	}
	_, _ = fmt.Fprintf(w, "%s (%s)\n", title, info.Name)
	if manifest.Description != "" {
		_, _ = fmt.Fprintln(w, manifest.Description)
	}
	if len(manifest.Stack) > 0 {
		_, _ = fmt.Fprintf(w, "Stack: %s\n", strings.Join(manifest.Stack, ", "))
	}
	if strings.TrimSpace(manifest.Version) != "" {
		_, _ = fmt.Fprintf(w, "Version: %s\n", manifest.Version)
	}
	if strings.TrimSpace(manifest.StartDocument) != "" {
		_, _ = fmt.Fprintf(w, "Start document: %s\n", manifest.StartDocument)
	}
	writeTemplateDesignSection(w, manifest.Design)
	writeTemplateVarTable(w, "Required Variables", manifest.RequiredVars)
	writeTemplateVarTable(w, "Optional Variables", manifest.OptionalVars)
	writeTemplatePostHooksSection(w, manifest.PostHooks)
	writeTemplateRelocationsSection(w, manifest.Relocations)
	writeTemplateOrientationSection(w, manifest.Orientation)
	writeTemplateDocsSection(w, manifest.Docs)
	writeTemplateFilesSection(w, info.Path)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Tip: template-manager generate %s%s\n", info.Name, FormatScenarioTemplateRequiredFlags(manifest.RequiredVars))
	return nil
}

func writeTemplateDesignSection(w io.Writer, design TemplateDesign) {
	if strings.TrimSpace(design.Default) == "" && strings.TrimSpace(design.Adapter) == "" && !design.Required {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Design:")
	if design.Default != "" {
		_, _ = fmt.Fprintf(w, "  default: %s\n", design.Default)
	}
	if design.Adapter != "" {
		_, _ = fmt.Fprintf(w, "  adapter: %s\n", design.Adapter)
	}
	if design.Required {
		_, _ = fmt.Fprintln(w, "  required: yes")
	}
}

func writeTemplatePostHooksSection(w io.Writer, hooks []TemplateHook) {
	if len(hooks) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Post Hooks:")
	for _, hook := range hooks {
		line := hook.Description
		if line == "" {
			line = hook.Cmd
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", line)
	}
}

func writeTemplateRelocationsSection(w io.Writer, relocations []TemplateRelocation) {
	if len(relocations) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Relocations:")
	for _, reloc := range relocations {
		label := reloc.Description
		if label == "" {
			label = reloc.From + " -> " + reloc.To
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", label)
		_, _ = fmt.Fprintf(w, "      from: %s\n", reloc.From)
		_, _ = fmt.Fprintf(w, "      to:   %s\n", reloc.To)
		writeTemplateRelocationHooks(w, reloc.Post)
	}
}

func writeTemplateRelocationHooks(w io.Writer, hooks []TemplateHook) {
	for _, hook := range hooks {
		cmd := hook.Description
		if cmd == "" {
			cmd = hook.Cmd
		}
		_, _ = fmt.Fprintf(w, "      post: %s\n", cmd)
	}
}

func writeTemplateOrientationSection(w io.Writer, orientation *TemplateOrientation) {
	if orientation == nil {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Orientation:")
	if orientation.CopyTo != "" {
		_, _ = fmt.Fprintf(w, "  copy to: %s\n", orientation.CopyTo)
	}
	if orientation.StartDocument != "" {
		_, _ = fmt.Fprintf(w, "  start document: %s\n", orientation.StartDocument)
	}
	if len(orientation.Steps) > 0 {
		_, _ = fmt.Fprintf(w, "  steps: %d\n", len(orientation.Steps))
	}
}

func writeTemplateDocsSection(w io.Writer, docs map[string]string) {
	if len(docs) == 0 {
		return
	}
	docKeys := make([]string, 0, len(docs))
	for key := range docs {
		docKeys = append(docKeys, key)
	}
	sort.Strings(docKeys)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Docs:")
	for _, key := range docKeys {
		_, _ = fmt.Fprintf(w, "  - %s: %s\n", key, docs[key])
	}
}

func writeTemplateFilesSection(w io.Writer, path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Files:")
	for _, name := range names {
		_, _ = fmt.Fprintf(w, "  - %s\n", name)
	}
}

func RenderGenerateResponse(w io.Writer, format cliout.Format, result GenerateResult) error {
	_ = format
	if result.DryRun {
		_, _ = fmt.Fprintf(w, "[DRY-RUN] Would generate template %s at %s\n", result.TemplateName, result.Destination)
		WriteTemplateValues(w, result.Values)
		WriteTemplateDesign(w, result.Design)
		WriteTemplateProvenance(w, result.Provenance)
		WriteTemplateRelocations(w, result.Relocations)
		return nil
	}
	_, _ = fmt.Fprintf(w, "Created %s at %s\n", result.DisplayName, result.Destination)
	WriteTemplateValues(w, result.Values)
	WriteTemplateDesign(w, result.Design)
	WriteTemplateProvenance(w, result.Provenance)
	WriteTemplateRelocations(w, result.Relocations)
	WriteTemplateNextSteps(w, result.Destination, result.Manifest)
	if !result.RunHooks {
		WriteTemplateHooks(w, result.Manifest)
	}
	return nil
}

func WriteTemplateProvenance(w io.Writer, provenance GenerationProvenance) {
	if provenance.Template.ID == "" && provenance.Design.ID == "" {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Generation:")
	if provenance.Template.ID != "" {
		_, _ = fmt.Fprintf(w, "  template: %s", provenance.Template.ID)
		if provenance.Template.Version != "" {
			_, _ = fmt.Fprintf(w, " (%s)", provenance.Template.Version)
		}
		_, _ = fmt.Fprintln(w)
	}
	if provenance.GeneratedAt != "" {
		_, _ = fmt.Fprintf(w, "  generated_at: %s\n", provenance.GeneratedAt)
	}
	if provenance.ManifestSha != "" {
		_, _ = fmt.Fprintf(w, "  manifest_sha: %s\n", shortSha(provenance.ManifestSha))
	}
	if provenance.ContentSha != "" {
		_, _ = fmt.Fprintf(w, "  content_sha:  %s\n", shortSha(provenance.ContentSha))
	}
	if provenance.Design.ID != "" {
		_, _ = fmt.Fprintf(w, "  design: %s", provenance.Design.ID)
		if provenance.Design.Version != "" {
			_, _ = fmt.Fprintf(w, " (%s)", provenance.Design.Version)
		}
		if provenance.Design.Adapter != "" {
			_, _ = fmt.Fprintf(w, " adapter=%s", provenance.Design.Adapter)
		}
		_, _ = fmt.Fprintln(w)
	}
}

func WriteTemplateDesign(w io.Writer, design ResolvedDesign) {
	if strings.TrimSpace(design.KitID) == "" {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Design:")
	_, _ = fmt.Fprintf(w, "  kit: %s", design.KitID)
	if design.Version != "" {
		_, _ = fmt.Fprintf(w, " (%s)", design.Version)
	}
	_, _ = fmt.Fprintln(w)
	if design.AdapterID != "" {
		_, _ = fmt.Fprintf(w, "  adapter: %s\n", design.AdapterID)
	}
	if len(design.Copies) > 0 {
		_, _ = fmt.Fprintln(w, "  copy:")
		for _, copy := range design.Copies {
			_, _ = fmt.Fprintf(w, "    - %s -> %s\n", copy.From, copy.To)
		}
	}
}

// WriteTemplateRelocations renders the resolved relocations (if any). Used
// by both the dry-run path (so authors can see exactly where each folder
// would land) and the success path (so the destination summary explains
// what happened outside the scenario directory).
func WriteTemplateRelocations(w io.Writer, relocations []ResolvedRelocation) {
	if len(relocations) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Relocations:")
	for _, reloc := range relocations {
		label := reloc.Description
		if label == "" {
			label = reloc.From + " -> " + reloc.To
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", label)
		_, _ = fmt.Fprintf(w, "      from: %s\n", reloc.From)
		_, _ = fmt.Fprintf(w, "      to:   %s\n", reloc.To)
		for _, hook := range reloc.Post {
			cmd := hook.Description
			if cmd == "" {
				cmd = hook.Cmd
			}
			_, _ = fmt.Fprintf(w, "      post: %s\n", cmd)
		}
	}
}

func RenderTemplateValidateResponse(w io.Writer, format cliout.Format, report TemplateValidationReport) error {
	if format == cliout.FormatJSON {
		return writeScenarioTemplateValidateJSON(w, report)
	}
	mode := string(report.Mode)
	if mode == "" {
		mode = string(TemplateValidationModeShallow)
	}
	if len(report.Issues) == 0 {
		if report.WarningSummary.Total > 0 {
			_, _ = fmt.Fprintf(w, "Validated %d scenario templates (%s) with %d warning(s)\n", report.Count, mode, report.WarningSummary.Total)
			writeTemplateValidationWarnings(w, report.WarningSummary)
		} else {
			_, _ = fmt.Fprintf(w, "Validated %d scenario templates (%s)\n", report.Count, mode)
		}
		writeRetainedTemplateValidationPaths(w, report.DeepRuns)
		return nil
	}
	_, _ = fmt.Fprintf(w, "Scenario template validation failed (%d templates checked, %s)\n", report.Count, mode)
	for _, issue := range report.Issues {
		line := issue.Template
		if strings.TrimSpace(issue.Path) != "" {
			line += " [" + issue.Path + "]"
		}
		_, _ = fmt.Fprintf(w, "  - %s: %s\n", line, issue.Message)
	}
	writeTemplateValidationWarnings(w, report.WarningSummary)
	writeRetainedTemplateValidationPaths(w, report.DeepRuns)
	return nil
}

func RenderTemplateCleanupResponse(w io.Writer, format cliout.Format, result TemplateCleanupResult) error {
	if format == cliout.FormatJSON {
		return writeScenarioTemplateCleanupJSON(w, result)
	}
	_, _ = fmt.Fprintln(w, "Template validation cleanup")
	_, _ = fmt.Fprintf(w, "Status: %s\n", result.Message)
	if len(result.Eligible) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Eligible")
		for _, run := range result.Eligible {
			_, _ = fmt.Fprintf(w, "  %s  %s  age=%s  retained=%t  %s\n", run.Marker.RunID, run.Marker.Template, run.Age, run.Marker.Retained, run.Marker.TempRoot)
		}
	}
	if len(result.Removed) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Removed")
		for _, run := range result.Removed {
			_, _ = fmt.Fprintf(w, "  %s  %s\n", run.Marker.RunID, run.Marker.TempRoot)
		}
	}
	if len(result.Skipped) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Skipped")
		for _, skipped := range result.Skipped {
			label := skipped.Path
			if skipped.Run != nil {
				label = skipped.Run.Marker.RunID
			}
			_, _ = fmt.Fprintf(w, "  %s  %s\n", label, skipped.Reason)
		}
	}
	if len(result.Failures) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Failures")
		for _, failure := range result.Failures {
			label := failure.Path
			if failure.Run != nil {
				label = failure.Run.Marker.RunID
			}
			if strings.TrimSpace(label) == "" {
				label = "cleanup"
			}
			_, _ = fmt.Fprintf(w, "  %s  %s\n", label, failure.Error)
		}
	}
	if result.NeedsProtoGenerate {
		_, _ = fmt.Fprintln(w)
		if result.DryRun {
			_, _ = fmt.Fprintln(w, "Next steps: rerun without --dry-run to remove proto artifacts and regenerate packages/proto outputs.")
		} else if !result.ProtoGenerateRan {
			_, _ = fmt.Fprintln(w, "Next steps: run `cd packages/proto && make generate` to refresh proto outputs.")
		}
	}
	return nil
}

func writeRetainedTemplateValidationPaths(w io.Writer, runs []TemplateValidationDeepRun) {
	for _, run := range runs {
		if run.RetainedTemp && strings.TrimSpace(run.TempRoot) != "" {
			_, _ = fmt.Fprintf(w, "Retained temp workspace for %s: %s\n", run.Template, run.TempRoot)
			if strings.TrimSpace(run.CleanupCommand) != "" {
				_, _ = fmt.Fprintf(w, "Cleanup command: %s\n", run.CleanupCommand)
			}
		}
	}
}

func writeTemplateValidationWarnings(w io.Writer, summary TemplateValidationWarningSummary) {
	if summary.Total == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "Warnings:")
	for _, phase := range summary.Phases {
		if phase.Count == 0 {
			continue
		}
		_, _ = fmt.Fprintf(w, "  %s (%d)\n", phase.Name, phase.Count)
		for _, warning := range phase.Warnings {
			_, _ = fmt.Fprintf(w, "    - %s\n", warning.Message)
			if strings.TrimSpace(warning.LogPath) != "" {
				_, _ = fmt.Fprintf(w, "      log: %s\n", warning.LogPath)
			}
			if strings.TrimSpace(warning.ArtifactPath) != "" {
				_, _ = fmt.Fprintf(w, "      artifact: %s\n", warning.ArtifactPath)
			}
		}
	}
}

func WriteTemplateValues(w io.Writer, values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintln(w, "Resolved values:")
	for _, key := range keys {
		_, _ = fmt.Fprintf(w, "  %s=%s\n", key, values[key])
	}
}

func WriteTemplateNextSteps(w io.Writer, destination string, manifest TemplateManifest) {
	_, _ = fmt.Fprintln(w)
	startDocument := strings.TrimSpace(manifest.StartDocument)
	if manifest.Orientation != nil && strings.TrimSpace(manifest.Orientation.StartDocument) != "" {
		startDocument = strings.TrimSpace(manifest.Orientation.StartDocument)
	}
	if startDocument != "" {
		_, _ = fmt.Fprintln(w, "Start here:")
		_, _ = fmt.Fprintf(w, "  %s\n", startDocument)
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, "Next steps:")
	if manifest.Orientation != nil {
		scenarioName := filepath.Base(filepath.Clean(destination))
		_, _ = fmt.Fprintln(w, "  1. Read the start document")
		_, _ = fmt.Fprintf(w, "  2. Track initialization with: template-manager orient %s\n", scenarioName)
		_, _ = fmt.Fprintf(w, "  3. Finalize orientation with: template-manager orient %s --finalize\n", scenarioName)
	} else if startDocument != "" {
		_, _ = fmt.Fprintln(w, "  1. Read the start document")
		_, _ = fmt.Fprintln(w, "  2. Run scenario setup and tests")
	} else {
		_, _ = fmt.Fprintf(w, "  1. Review files in %s\n", destination)
		_, _ = fmt.Fprintln(w, "  2. Run scenario setup and tests")
	}
	if len(manifest.PostHooks) > 0 {
		_, _ = fmt.Fprintln(w, "  3. Consider re-running with --run-hooks")
	}
}

func WriteTemplateHooks(w io.Writer, manifest TemplateManifest) {
	if len(manifest.PostHooks) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Available post hooks:")
	for _, hook := range manifest.PostHooks {
		line := hook.Description
		if line == "" {
			line = hook.Cmd
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", line)
	}
}

func formatTemplateRequiredVars(manifest TemplateManifest) string {
	keys := make([]string, 0, len(manifest.RequiredVars))
	for key := range manifest.RequiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func writeTemplateVarTable(w io.Writer, title string, variables map[string]TemplateVar) {
	if len(variables) == 0 {
		return
	}
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, title+":")
	for _, key := range keys {
		variable := variables[key]
		line := key
		if variable.Flag != "" {
			line = fmt.Sprintf("%s (--%s)", key, variable.Flag)
		}
		if variable.Description != "" {
			line += " - " + variable.Description
		}
		if variable.Default != "" {
			line += " [default: " + variable.Default + "]"
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", line)
	}
}
