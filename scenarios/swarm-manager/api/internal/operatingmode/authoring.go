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
	// StartFrom names an existing registered mode to clone as the head start —
	// its phase graph, reads, transitions, and example-runs re-homed under the
	// new id — instead of the blank built-in template. This is the reuse-first
	// path: an author starts from the closest existing methodology (including a
	// composed one that already delegates via executed_by) and edits, rather
	// than assembling a mode from scratch. Empty uses the built-in template.
	StartFrom string
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
	// UncoveredBranches lists the guarded/classified edges no example-run walks —
	// the branch example-runs an author still owes before the simulation
	// walkthrough. A valid mode can still carry uncovered branches; coverage is
	// an authoring-completeness signal, not a load failure.
	UncoveredBranches []string `json:"uncovered_branches,omitempty"`
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

	var files []scaffoldFile
	if startFrom := cleanModeID(req.StartFrom); startFrom != "" {
		if startFrom == id {
			return ScaffoldResult{}, fmt.Errorf("start-from mode %q must differ from the new mode id", id)
		}
		files, err := s.renderCloneScaffold(startFrom, id, label, description)
		if err != nil {
			return ScaffoldResult{}, err
		}
		return s.writeScaffold(id, files, req.Force)
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
	files = []scaffoldFile{
		{RelPath: ModeFileName, Content: modeJSON},
		{RelPath: filepath.ToSlash(filepath.Join(exampleRunsDirName, happyPathPresetID+".json")), Content: exampleJSON},
	}
	return s.writeScaffold(id, files, req.Force)
}

// scaffoldFile is one file a scaffold writes, keyed by its slash-separated path
// relative to the mode folder.
type scaffoldFile struct {
	RelPath string
	Content []byte
}

// writeScaffold refuses to clobber an existing folder (unless Force was honored
// by the caller before this point), creates the folder, and writes every
// scaffold file. It is shared by the template and clone paths so both produce an
// identical on-disk result shape.
func (s *Service) writeScaffold(id string, files []scaffoldFile, force bool) (ScaffoldResult, error) {
	dir := filepath.Join(s.modesDir(), id)
	if _, statErr := os.Stat(dir); statErr == nil {
		if !force {
			return ScaffoldResult{}, fmt.Errorf("mode %q already exists at %s (pass force to overwrite)", id, dir)
		}
		// A re-scaffold replaces the folder wholesale so stale example-runs from
		// a prior scaffold never linger alongside the freshly written set.
		if err := os.RemoveAll(dir); err != nil {
			return ScaffoldResult{}, fmt.Errorf("overwrite mode dir %q: %w", dir, err)
		}
	}
	created := make([]string, 0, len(files))
	for _, file := range files {
		abs := filepath.Join(dir, filepath.FromSlash(file.RelPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return ScaffoldResult{}, fmt.Errorf("create mode dir %q: %w", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, file.Content, 0o644); err != nil {
			return ScaffoldResult{}, fmt.Errorf("write %q: %w", abs, err)
		}
		created = append(created, file.RelPath)
	}
	return ScaffoldResult{Mode: id, Dir: dir, CreatedFiles: created}, nil
}

// renderCloneScaffold builds the scaffold files for a reuse-first scaffold: it
// reads an existing registered mode's folder, re-homes its identity and derived
// fields (id, label, description, prompt catalog prefix, artifact root, event
// sources) under the new id, and re-targets its example-runs at the new mode.
// The rendered set is verified against the full on-disk mode set — the new
// mode's phase graph loads and every re-homed example-run walks the real guards
// (including one-level executed_by delegation) — before it is returned, so a
// clone can never produce a folder that fails to load or simulate.
func (s *Service) renderCloneScaffold(startFrom, id, label, description string) ([]scaffoldFile, error) {
	srcDir := filepath.Join(s.modesDir(), startFrom)
	srcModePath := filepath.Join(srcDir, ModeFileName)
	srcModeJSON, err := os.ReadFile(srcModePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("start-from mode %q not found on disk at %s", startFrom, srcModePath)
		}
		return nil, fmt.Errorf("read start-from mode %q: %w", startFrom, err)
	}
	modeJSON, err := rehomeCloneMode(srcModeJSON, id, label, description)
	if err != nil {
		return nil, fmt.Errorf("clone mode %q: %w", startFrom, err)
	}

	files := []scaffoldFile{{RelPath: ModeFileName, Content: modeJSON}}
	exampleFiles, err := os.ReadDir(filepath.Join(srcDir, exampleRunsDirName))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read start-from example-runs: %w", err)
	}
	for _, entry := range exampleFiles {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(srcDir, exampleRunsDirName, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read start-from example-run %q: %w", entry.Name(), err)
		}
		rehomed, err := rehomeCloneExampleRun(raw, id)
		if err != nil {
			return nil, fmt.Errorf("clone example-run %q: %w", entry.Name(), err)
		}
		files = append(files, scaffoldFile{
			RelPath: filepath.ToSlash(filepath.Join(exampleRunsDirName, entry.Name())),
			Content: rehomed,
		})
	}

	if err := verifyCloneScaffold(s.modesDir(), id, files); err != nil {
		return nil, fmt.Errorf("cloned mode failed self-validation: %w", err)
	}
	return files, nil
}

// rehomeCloneMode rewrites a source mode.json under a new identity: id, label,
// and description become the clone's, and the id-derived fields (prompt catalog
// prefix, artifact root, backlog/metrics event sources) are regenerated from the
// new id so the clone points at its own prompt skills and artifacts rather than
// the source's. Everything else — the phase graph, reads, transitions,
// classification, delegation — is preserved verbatim as the author's head start.
func rehomeCloneMode(srcJSON []byte, id, label, description string) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(srcJSON, &doc); err != nil {
		return nil, fmt.Errorf("decode source mode.json: %w", err)
	}
	newRoot := "modes/" + id
	oldRoot := ""
	if artifact, ok := doc["artifact"].(map[string]any); ok {
		if root, ok := artifact["root"].(string); ok {
			oldRoot = strings.TrimSpace(root)
		}
		artifact["root"] = newRoot
	}
	doc["id"] = id
	doc["label"] = label
	doc["description"] = description
	if prompt, ok := doc["prompt"].(map[string]any); ok {
		prompt["catalog_prefix"] = "swarm-manager-" + id
	}
	if backlogSync, ok := doc["backlog_sync"].(map[string]any); ok {
		if _, has := backlogSync["event_source"]; has {
			backlogSync["event_source"] = id
		}
	}
	if metrics, ok := doc["metrics"].(map[string]any); ok {
		if _, has := metrics["event_source"]; has {
			metrics["event_source"] = id
		}
	}
	// Re-home every artifact path anchored under the source mode's artifact root
	// (phase output_artifacts, result_binding artifacts) so they land under the
	// clone's root — the validator rejects a path outside the mode root.
	if oldRoot != "" && oldRoot != newRoot {
		rehomeRootPrefix(doc, oldRoot, newRoot)
	}
	return marshalScaffoldJSON(doc)
}

// rehomeRootPrefix recursively rewrites every string value equal to oldRoot or
// prefixed by oldRoot+"/" so the prefix becomes newRoot, leaving all other data
// untouched. oldRoot is a distinctive `modes/<id>` path, so the swap is exact.
func rehomeRootPrefix(node any, oldRoot, newRoot string) {
	switch typed := node.(type) {
	case map[string]any:
		for key, value := range typed {
			if str, ok := value.(string); ok {
				typed[key] = swapRootPrefix(str, oldRoot, newRoot)
				continue
			}
			rehomeRootPrefix(value, oldRoot, newRoot)
		}
	case []any:
		for i, value := range typed {
			if str, ok := value.(string); ok {
				typed[i] = swapRootPrefix(str, oldRoot, newRoot)
				continue
			}
			rehomeRootPrefix(value, oldRoot, newRoot)
		}
	}
}

func swapRootPrefix(value, oldRoot, newRoot string) string {
	if value == oldRoot {
		return newRoot
	}
	if strings.HasPrefix(value, oldRoot+"/") {
		return newRoot + strings.TrimPrefix(value, oldRoot)
	}
	return value
}

// rehomeCloneExampleRun re-targets a source example-run at the clone's mode id so
// it walks the new mode. The fixture body — seeded outputs and expected path —
// is preserved, since the phase graph it exercises is preserved.
func rehomeCloneExampleRun(srcJSON []byte, id string) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(srcJSON, &doc); err != nil {
		return nil, fmt.Errorf("decode source example-run: %w", err)
	}
	doc["mode"] = id
	return marshalScaffoldJSON(doc)
}

func marshalScaffoldJSON(doc map[string]any) ([]byte, error) {
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// verifyCloneScaffold proves the rendered clone loads AND passes the full
// semantic validator before it is written: it loads the existing on-disk mode
// set (so executed_by sub-modes resolve), overlays the new mode with its
// re-homed example-runs, and runs ValidateLoadedModes — the same validation the
// registry runs at startup — so a clone can never emit a folder the loader would
// later reject.
func verifyCloneScaffold(modesDir, id string, files []scaffoldFile) error {
	defs, err := LoadModesFromDir(modesDir)
	if err != nil {
		return fmt.Errorf("load existing modes: %w", err)
	}
	var newDef Definition
	for _, file := range files {
		if file.RelPath == ModeFileName {
			newDef, err = LoadModeDefinition(file.Content)
			if err != nil {
				return fmt.Errorf("load cloned mode.json: %w", err)
			}
		}
	}
	if newDef.Mode != Mode(id) {
		return fmt.Errorf("cloned mode.json declares id %q, expected %q", newDef.Mode, id)
	}
	for _, file := range files {
		if !strings.HasPrefix(file.RelPath, exampleRunsDirName+"/") {
			continue
		}
		run, err := LoadExampleRun(file.Content)
		if err != nil {
			return fmt.Errorf("load cloned example-run %q: %w", file.RelPath, err)
		}
		newDef.ExampleRuns = append(newDef.ExampleRuns, run)
	}
	defs[newDef.Mode] = newDef
	if err := ValidateLoadedModes(defs); err != nil {
		return err
	}
	return nil
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
	report := ValidationReport{
		Mode:        id,
		OK:          true,
		PhaseCount:  len(def.PhaseGraph.Phases),
		ExampleRuns: len(def.ExampleRuns),
		Summary:     "valid",
	}
	// Report branch coverage so the author knows which guarded/classified paths
	// still lack a covering example-run before the simulation walkthrough. A
	// coverage-computation failure is not a validity failure — the mode loaded —
	// so it degrades to a note rather than flipping OK.
	if uncovered, covErr := modeBranchCoverage(defs, def); covErr != nil {
		report.Summary = "valid (branch coverage unavailable: " + covErr.Error() + ")"
	} else {
		report.UncoveredBranches = uncovered
		if len(uncovered) > 0 {
			report.Summary = fmt.Sprintf("valid; %d branch(es) not covered by an example-run", len(uncovered))
		}
	}
	return report, nil
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
	// Scaffolded templates contain no delegated phases, so the walk needs no
	// wider mode set than the template itself.
	if _, err := WalkExampleRun(map[Mode]Definition{def.Mode: def}, def, run); err != nil {
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
  "target": { "kind": "initiative" },
  "input_contract": {
    "specs": [
      { "id": "execution.mode_id", "type": "string", "required": true, "sensitivity": "public", "retention": "value", "description": "Stable operating-mode identity for this execution." },
      { "id": "execution.phase_id", "type": "string", "required": true, "sensitivity": "public", "retention": "value", "description": "Current phase identity within the pinned graph." },
      { "id": "execution.round_number", "type": "integer", "required": true, "minimum": 1, "sensitivity": "public", "retention": "value", "description": "One-based round number within the immutable execution." },
      { "id": "execution.operator_note", "type": "string", "max_length": 32768, "sensitivity": "internal", "retention": "value", "description": "Optional operator guidance for the current round." },
      { "id": "execution.prior_rounds", "type": "array", "required": true, "sensitivity": "internal", "retention": "value", "description": "Durable prior round envelopes available to the current phase." },
      { "id": "execution.elastic_slice_guidance", "type": "string", "required": true, "sensitivity": "public", "retention": "digest", "description": "Shared guidance for selecting a bounded implementation slice." },
      { "id": "execution.backlog_sync_guidance", "type": "string", "required": true, "sensitivity": "public", "retention": "digest", "description": "Shared contract for emitting backlog reconciliation proposals." },
      { "id": "initiative.name", "type": "string", "required": true, "sensitivity": "internal", "retention": "value", "description": "Stable initiative identity targeted by this mode." },
      { "id": "initiative.title", "type": "string", "required": true, "sensitivity": "internal", "retention": "value", "description": "Current initiative title." },
      { "id": "initiative.description", "type": "string", "sensitivity": "internal", "retention": "value", "description": "Current initiative description." },
      { "id": "initiative.acceptance_criteria", "type": "string", "sensitivity": "internal", "retention": "value", "description": "Newline-normalized initiative acceptance criteria." },
      { "id": "initiative.member_items", "type": "array", "required": true, "sensitivity": "internal", "retention": "value", "description": "Resolved initiative backlog members and their current status." }
    ],
    "sources": [
      { "input_id": "execution.mode_id", "kind": "generic_provider", "capability": "generic.operating_mode" },
      { "input_id": "execution.phase_id", "kind": "generic_provider", "capability": "generic.phase" },
      { "input_id": "execution.round_number", "kind": "generic_provider", "capability": "generic.round_number" },
      { "input_id": "execution.operator_note", "kind": "generic_provider", "capability": "generic.operator_note" },
      { "input_id": "execution.prior_rounds", "kind": "generic_provider", "capability": "generic.prior_rounds" },
      { "input_id": "execution.elastic_slice_guidance", "kind": "generic_provider", "capability": "generic.elastic_slice" },
      { "input_id": "execution.backlog_sync_guidance", "kind": "generic_provider", "capability": "generic.backlog_sync_proposal" },
      { "input_id": "initiative.name", "kind": "target_adapter", "capability": "target.initiative_name" },
      { "input_id": "initiative.title", "kind": "target_adapter", "capability": "target.initiative_title" },
      { "input_id": "initiative.description", "kind": "target_adapter", "capability": "target.initiative_description" },
      { "input_id": "initiative.acceptance_criteria", "kind": "target_adapter", "capability": "target.acceptance_criteria" },
      { "input_id": "initiative.member_items", "kind": "target_adapter", "capability": "target.member_items" }
    ],
    "aliases": [
      { "name": "OPERATING_MODE", "input_id": "execution.mode_id" },
      { "name": "PHASE", "input_id": "execution.phase_id" },
      { "name": "ROUND_NUMBER", "input_id": "execution.round_number" },
      { "name": "OPERATOR_NOTE", "input_id": "execution.operator_note" },
      { "name": "PRIOR_ROUNDS_JSON", "input_id": "execution.prior_rounds" },
      { "name": "ELASTIC_SLICE_SNIPPET", "input_id": "execution.elastic_slice_guidance" },
      { "name": "BACKLOG_SYNC_PROPOSAL_SNIPPET", "input_id": "execution.backlog_sync_guidance" },
      { "name": "INITIATIVE_NAME", "input_id": "initiative.name" },
      { "name": "INITIATIVE_TITLE", "input_id": "initiative.title" },
      { "name": "INITIATIVE_DESCRIPTION", "input_id": "initiative.description" },
      { "name": "ACCEPTANCE_CRITERIA", "input_id": "initiative.acceptance_criteria" },
      { "name": "MEMBER_ITEMS_JSON", "input_id": "initiative.member_items" }
    ]
  },
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
        "reads": ["OPERATING_MODE", "PHASE", "ROUND_NUMBER", "OPERATOR_NOTE", "PRIOR_ROUNDS_JSON", "INITIATIVE_NAME", "INITIATIVE_TITLE", "INITIATIVE_DESCRIPTION", "MEMBER_ITEMS_JSON", "ELASTIC_SLICE_SNIPPET"],
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
        "reads": ["OPERATING_MODE", "PHASE", "ROUND_NUMBER", "OPERATOR_NOTE", "PRIOR_ROUNDS_JSON", "INITIATIVE_NAME", "ACCEPTANCE_CRITERIA", "MEMBER_ITEMS_JSON"],
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
        "reads": ["OPERATING_MODE", "PHASE", "ROUND_NUMBER", "OPERATOR_NOTE", "PRIOR_ROUNDS_JSON", "INITIATIVE_NAME", "MEMBER_ITEMS_JSON", "BACKLOG_SYNC_PROPOSAL_SNIPPET"],
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
