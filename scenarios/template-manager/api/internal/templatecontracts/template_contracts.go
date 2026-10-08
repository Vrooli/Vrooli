package templatecontracts

import (
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

type TemplateVar struct {
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type TemplateHook struct {
	Description string            `json:"description,omitempty"`
	Argv        []string          `json:"argv"`
	Env         map[string]string `json:"env,omitempty"`
	Cwd         string            `json:"cwd,omitempty"`
}

// TemplateRelocation declares an out-of-tree placement performed by the
// generator after the in-tree copy. The directory at From (template-relative)
// is rendered (with placeholder substitution applied to both file content
// and path components) into To (repo-root-relative; may contain placeholders).
//
// Post hooks run from the repo root after every relocation in the manifest
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
	Name string `json:"name,omitempty"`
	// BaseTemplate lets a focused template inherit a maintained scenario
	// skeleton while replacing selected slots. The base is resolved beside the
	// current template under templates/scenarios.
	BaseTemplate     string                 `json:"baseTemplate,omitempty"`
	BaseCopyExcludes []string               `json:"baseCopyExcludes,omitempty"`
	Version          string                 `json:"version,omitempty"`
	DisplayName      string                 `json:"displayName,omitempty"`
	Description      string                 `json:"description,omitempty"`
	Stack            []string               `json:"stack,omitempty"`
	StartDocument    string                 `json:"startDocument,omitempty"`
	Design           TemplateDesign         `json:"design,omitempty"`
	Orientation      *TemplateOrientation   `json:"orientation,omitempty"`
	RequiredVars     map[string]TemplateVar `json:"requiredVars,omitempty"`
	OptionalVars     map[string]TemplateVar `json:"optionalVars,omitempty"`
	Docs             map[string]string      `json:"docs,omitempty"`
	CopyExcludes     []string               `json:"copyExcludes,omitempty"`
	PostHooks        []TemplateHook         `json:"postHooks,omitempty"`
	Relocations      []TemplateRelocation   `json:"relocations,omitempty"`
	ExampleDomain    *TemplateExampleDomain `json:"exampleDomain,omitempty"`
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
	Kind     string   `json:"kind"`
	Path     string   `json:"path,omitempty"`
	Pattern  string   `json:"pattern,omitempty"`
	Query    string   `json:"query,omitempty"`
	Text     string   `json:"text,omitempty"`
	MinCount int      `json:"minCount,omitempty"`
	Exec     []string `json:"exec,omitempty"`
	Timeout  string   `json:"timeout,omitempty"`
	Optional bool     `json:"optional,omitempty"`
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
	From   string `json:"from"`
	To     string `json:"to"`
	Append bool   `json:"append,omitempty"`
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
