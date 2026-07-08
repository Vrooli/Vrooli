package operatingmode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"swarm-manager/internal/pathutil"
)

// This file is the self-serve authoring surface of the data-driven engine: it
// scaffolds a new mode folder from a built-in template, validates an on-disk
// mode by loading it through the real loader/validator, and simulates a mode
// straight from disk (before it is registered). Together these make authoring a
// mode a data task — scaffold, validate, simulate, then restart to execute —
// with zero Go edits and no rebuild. Each operation resolves the scenario's
// modes/ directory and reuses the same loader/validator/guard machinery the
// runtime does, so what an author validates is exactly what the runtime loads.

// modeIDPattern is the accepted shape of a mode id (also its folder name):
// lowercase alphanumeric segments joined by single hyphens, matching the
// existing item-level / holistic-loop / phased-plan-drain ids.
var modeIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ScaffoldRequest describes a new mode to write to disk from the built-in
// template. Only ID is required; Label/Description default from the ID.
type ScaffoldRequest struct {
	ID          string
	Label       string
	Description string
	// Force overwrites an existing mode folder instead of refusing. Off by
	// default so an author never silently clobbers a mode they are editing.
	Force bool
}

// ScaffoldResult reports the folder a scaffold wrote.
type ScaffoldResult struct {
	Mode         string   `json:"mode"`
	Dir          string   `json:"dir"`
	CreatedFiles []string `json:"created_files"`
}

// ValidationReport is the typed result of validating an on-disk mode. An invalid
// mode is a normal, expected outcome of a validate request (OK=false with typed
// errors), not a transport error — only infrastructure failures return an error.
type ValidationReport struct {
	Mode        string   `json:"mode"`
	OK          bool     `json:"ok"`
	Errors      []string `json:"errors,omitempty"`
	PhaseCount  int      `json:"phase_count"`
	ExampleRuns int      `json:"example_runs"`
	Summary     string   `json:"summary"`
}

// modesDir resolves the scenario's modes/ directory. It honors the service's
// configured scenario root (set in production wiring) and falls back to the
// resolved scenario root otherwise, so scaffolding and draft validation/
// simulation operate on the same folder the registry loads at startup.
func (s *Service) modesDir() string {
	root := strings.TrimSpace(s.scenarioRoot)
	if root == "" {
		root = pathutil.ResolveScenarioRoot("swarm-manager")
	}
	return filepath.Join(root, modesDirName)
}

func cleanModeID(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// ScaffoldMode writes a new mode folder (mode.json + example-runs/happy-path.json)
// from the built-in initiative-scoped phase-mode template. The rendered template
// is validated in memory (loader + example-run walk) before anything is written,
// so a scaffold either produces a folder that loads and simulates cleanly or
// fails without touching disk.
func (s *Service) ScaffoldMode(req ScaffoldRequest) (ScaffoldResult, error) {
	id := cleanModeID(req.ID)
	if id == "" {
		return ScaffoldResult{}, fmt.Errorf("mode id is required")
	}
	if !modeIDPattern.MatchString(id) {
		return ScaffoldResult{}, fmt.Errorf("mode id %q must be lowercase alphanumeric segments joined by single hyphens (e.g. my-mode)", id)
	}

	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = humanizeToken(id)
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = fmt.Sprintf("TODO: describe the %s methodology and when to use it.", label)
	}

	modeJSON, exampleJSON, err := renderScaffold(id, label, description)
	if err != nil {
		return ScaffoldResult{}, err
	}
	// Guard the built-in template against regressions: prove the rendered mode
	// loads and its happy-path example-run walks the real guards before writing.
	if err := verifyScaffoldTemplate(modeJSON, exampleJSON); err != nil {
		return ScaffoldResult{}, fmt.Errorf("scaffold template failed self-validation: %w", err)
	}

	dir := filepath.Join(s.modesDir(), id)
	if !req.Force {
		if _, statErr := os.Stat(dir); statErr == nil {
			return ScaffoldResult{}, fmt.Errorf("mode %q already exists at %s (pass force to overwrite)", id, dir)
		}
	}
	exampleDir := filepath.Join(dir, exampleRunsDirName)
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		return ScaffoldResult{}, fmt.Errorf("create mode dir %q: %w", dir, err)
	}
	modePath := filepath.Join(dir, ModeFileName)
	examplePath := filepath.Join(exampleDir, happyPathPresetID+".json")
	if err := os.WriteFile(modePath, modeJSON, 0o644); err != nil {
		return ScaffoldResult{}, fmt.Errorf("write %q: %w", modePath, err)
	}
	if err := os.WriteFile(examplePath, exampleJSON, 0o644); err != nil {
		return ScaffoldResult{}, fmt.Errorf("write %q: %w", examplePath, err)
	}
	return ScaffoldResult{
		Mode: id,
		Dir:  dir,
		CreatedFiles: []string{
			ModeFileName,
			filepath.ToSlash(filepath.Join(exampleRunsDirName, happyPathPresetID+".json")),
		},
	}, nil
}

// ValidateModeDraft loads the whole modes/ directory fresh from disk (so on-disk
// edits and cross-mode references both resolve) and reports whether the named
// mode is valid. It returns a typed report rather than an error for an invalid
// mode; an error is returned only when the mode folder is absent or the modes
// directory cannot be read.
func (s *Service) ValidateModeDraft(modeID string) (ValidationReport, error) {
	id := cleanModeID(modeID)
	if id == "" {
		return ValidationReport{}, fmt.Errorf("mode id is required")
	}
	modePath := filepath.Join(s.modesDir(), id, ModeFileName)
	if _, err := os.Stat(modePath); err != nil {
		if os.IsNotExist(err) {
			return ValidationReport{
				Mode:    id,
				OK:      false,
				Errors:  []string{fmt.Sprintf("no %s found for mode %q under %s", ModeFileName, id, s.modesDir())},
				Summary: "not found on disk",
			}, nil
		}
		return ValidationReport{}, fmt.Errorf("stat %q: %w", modePath, err)
	}

	defs, err := LoadModesFromDir(s.modesDir())
	if err != nil {
		// A load/validation failure over the full set is the expected shape of an
		// invalid draft; surface it as a typed report focused on the request.
		return ValidationReport{
			Mode:    id,
			OK:      false,
			Errors:  []string{err.Error()},
			Summary: "invalid",
		}, nil
	}
	def, ok := defs[Mode(id)]
	if !ok {
		return ValidationReport{
			Mode:    id,
			OK:      false,
			Errors:  []string{fmt.Sprintf("mode %q did not load from disk (folder name / declared id mismatch?)", id)},
			Summary: "invalid",
		}, nil
	}
	return ValidationReport{
		Mode:        id,
		OK:          true,
		PhaseCount:  len(def.PhaseGraph.Phases),
		ExampleRuns: len(def.ExampleRuns),
		Summary:     "valid",
	}, nil
}

// SimulateModeDraft simulates a mode straight from disk, loading the whole
// modes/ directory fresh so an author can preview a scaffolded mode's flow
// before it is registered (no restart). It delegates to the same simulation core
// SimulateMode uses, so a draft walk and a live walk are identical mechanics.
func (s *Service) SimulateModeDraft(ctx context.Context, modeID, presetID string) (SimulationResponse, error) {
	id := cleanModeID(modeID)
	if id == "" {
		return SimulationResponse{}, fmt.Errorf("mode id is required")
	}
	defs, err := LoadModesFromDir(s.modesDir())
	if err != nil {
		return SimulationResponse{}, err
	}
	def, ok := defs[Mode(id)]
	if !ok {
		return SimulationResponse{}, fmt.Errorf("unknown operating mode %q", id)
	}
	return s.simulateDefinition(ctx, def, presetID)
}

// verifyScaffoldTemplate loads the rendered template through the real loader and
// walks its happy-path example-run against the generic guards, so a scaffold can
// never emit a folder that would fail validation or simulation.
func verifyScaffoldTemplate(modeJSON, exampleJSON []byte) error {
	def, err := LoadModeDefinition(modeJSON)
	if err != nil {
		return fmt.Errorf("load mode.json: %w", err)
	}
	run, err := LoadExampleRun(exampleJSON)
	if err != nil {
		return fmt.Errorf("load example-run: %w", err)
	}
	if _, err := WalkExampleRun(def, run); err != nil {
		return fmt.Errorf("walk example-run %q: %w", run.ID, err)
	}
	return nil
}

// scaffoldFields are the substitutions the built-in template consumes.
type scaffoldFields struct {
	ID              string
	Snake           string
	Title           string
	LabelJSON       string
	DescriptionJSON string
}

func renderScaffold(id, label, description string) (modeJSON, exampleJSON []byte, err error) {
	labelJSON, err := json.Marshal(label)
	if err != nil {
		return nil, nil, fmt.Errorf("encode label: %w", err)
	}
	descJSON, err := json.Marshal(description)
	if err != nil {
		return nil, nil, fmt.Errorf("encode description: %w", err)
	}
	fields := scaffoldFields{
		ID:              id,
		Snake:           strings.ReplaceAll(id, "-", "_"),
		Title:           humanizeToken(id),
		LabelJSON:       string(labelJSON),
		DescriptionJSON: string(descJSON),
	}
	modeJSON, err = renderTemplate("scaffold-mode", scaffoldModeTemplate, fields)
	if err != nil {
		return nil, nil, err
	}
	exampleJSON, err = renderTemplate("scaffold-example-run", scaffoldExampleRunTemplate, fields)
	if err != nil {
		return nil, nil, err
	}
	return modeJSON, exampleJSON, nil
}

func renderTemplate(name, body string, fields scaffoldFields) ([]byte, error) {
	tmpl, err := template.New(name).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse %s template: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, fields); err != nil {
		return nil, fmt.Errorf("render %s template: %w", name, err)
	}
	return buf.Bytes(), nil
}

// scaffoldModeTemplate is the built-in starting point for a new mode: a minimal
// but complete initiative-scoped phase mode (execute → review → reconcile) with
// a verdict-guarded branch, so an author edits a working methodology rather than
// assembling one from scratch. TODO markers flag the prose an author must fill.
const scaffoldModeTemplate = `{
  "kind": "operating-mode",
  "id": "{{.ID}}",
  "label": {{.LabelJSON}},
  "description": {{.DescriptionJSON}},
  "best_for": [
    "TODO: describe when this methodology is the right choice."
  ],
  "not_for": [
    "TODO: describe when a different mode fits better."
  ],
  "tradeoffs": [
    "TODO: state the cost this mode pays for its benefit."
  ],
  "scope": { "kind": "initiative" },
  "run_strategy": { "kind": "operator_gated_loop" },
  "prompt": { "catalog_prefix": "swarm-manager-{{.ID}}" },
  "artifact": { "root": "modes/{{.ID}}" },
  "profile": { "default_profile_key": "swarm-manager/deep-work" },
  "backlog_sync": {
    "capabilities": ["read_only"],
    "event_source": "{{.ID}}",
    "apply_mode": "operator-gated"
  },
  "metrics": {
    "event_source": "{{.ID}}",
    "accepted_verdicts": ["accepted"]
  },
  "lock": { "initiative_exclusive": true },
  "ui": { "workspace_tab_id": "operating-mode" },
  "phase_graph": {
    "start_phase": "execute",
    "terminal": ["reconcile"],
    "phases": [
      {
        "id": "execute",
        "kind": "execute",
        "activity_purpose": "{{.Snake}}_execute",
        "profile_key": "swarm-manager/deep-work",
        "writes_repo": true,
        "prompt": {
          "title": "{{.Title}} Execute",
          "trigger": "Operator starts {{.ID}} execute phase",
          "purpose": "TODO: describe what the execute phase does and what it must emit."
        },
        "transitions": [
          { "when": { "op": "always" }, "to": ["review"] }
        ]
      },
      {
        "id": "review",
        "kind": "review",
        "activity_purpose": "{{.Snake}}_review",
        "profile_key": "swarm-manager/analysis",
        "requires_criteria": true,
        "prompt": {
          "title": "{{.Title}} Review",
          "trigger": "Operator starts {{.ID}} review phase",
          "purpose": "TODO: evaluate the work against acceptance criteria and emit a verdict."
        },
        "declared_output": {
          "fields": [
            {
              "name": "verdict",
              "type": "string",
              "required": true,
              "enum": ["accepted", "changes_requested"],
              "description": "Acceptance verdict for the work against the initiative's criteria."
            }
          ]
        },
        "metrics": { "counts_acceptance_sample": true },
        "transitions": [
          { "when": { "op": "eq", "field": "verdict", "value": "accepted" }, "to": ["reconcile"] },
          { "when": { "op": "always" }, "to": ["execute"] }
        ]
      },
      {
        "id": "reconcile",
        "kind": "reconcile",
        "activity_purpose": "{{.Snake}}_reconcile",
        "profile_key": "swarm-manager/analysis",
        "auto_start_after": ["review"],
        "prompt": {
          "suffix": "reconcile",
          "title": "{{.Title}} Reconcile",
          "trigger": "Round refresher auto-starts {{.ID}} reconcile after review accepts",
          "purpose": "TODO: reconcile the backlog with the work just completed."
        }
      }
    ]
  }
}
`

// scaffoldExampleRunTemplate is the reserved happy-path fixture every phase mode
// ships: it walks execute → review (accepted) → reconcile through the real
// guards and asserts that path, so the mode is testable the moment it is
// scaffolded. Additional branch example-runs are an author's next data edit.
const scaffoldExampleRunTemplate = `{
  "kind": "operating-mode-example-run",
  "id": "happy-path",
  "mode": "{{.ID}}",
  "label": "Clean pass",
  "description": "Execute → review (accepted) → reconcile with no rework.",
  "branch": "review → reconcile (verdict = accepted)",
  "scenario": "TODO: describe the initiative shape where the first pass succeeds and review accepts.",
  "initiative": {
    "title": "{{.Title}} sandbox",
    "description": "TODO: describe a representative initiative for this mode.",
    "items": [
      { "ref": "execute/sample-item", "title": "Representative scoped work item", "status": "in_progress", "priority": 1, "effort": "M" }
    ],
    "criteria": [
      "TODO: an acceptance criterion the review phase evaluates against."
    ]
  },
  "steps": [
    { "phase": "execute", "note": "always -> review" },
    { "phase": "review", "output": { "verdict": "accepted" }, "note": "verdict=accepted -> reconcile" }
  ],
  "expected_path": ["execute", "review", "reconcile"]
}
`
