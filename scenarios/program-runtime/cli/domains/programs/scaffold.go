package programs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// scaffold generates a minimal, schema-valid contract + program pair so an
// author starts from a working skeleton instead of hand-copying another
// program. It is a local, side-effect-optional command: without --out-dir it
// only prints, so it is safe to run anywhere.

var scaffoldNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*\.[a-z0-9]+(?:-[a-z0-9]+)*$`)

type scaffoldResult struct {
	Name         string `json:"name"`
	Scenario     string `json:"scenario"`
	ContractJSON string `json:"contract_json"`
	ProgramPy    string `json:"program_py"`
	ContractPath string `json:"contract_path,omitempty"`
	ProgramPath  string `json:"program_path,omitempty"`
	Written      bool   `json:"written"`
}

type scaffoldBinding struct {
	ID     string `json:"id"`
	Effect string `json:"effect"`
}

type scaffoldOutputs struct {
	Status  map[string]any `json:"status"`
	Phase   map[string]any `json:"phase"`
	Signals map[string]any `json:"signals"`
	Errors  map[string]any `json:"errors"`
}

type scaffoldBudget struct {
	WallMs         int `json:"wall_ms"`
	InferenceCalls int `json:"inference_calls"`
	DelegatedRuns  int `json:"delegated_runs"`
}

type scaffoldFixture struct {
	ID     string         `json:"id"`
	Inputs map[string]any `json:"inputs"`
	Expect map[string]any `json:"expect"`
}

type scaffoldContract struct {
	Schema      string            `json:"$schema"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Purpose     string            `json:"purpose"`
	Helpers     map[string]string `json:"helpers"`
	Inputs      map[string]any    `json:"inputs"`
	Assumptions []string          `json:"assumptions"`
	Invariants  []string          `json:"invariants"`
	Bindings    []scaffoldBinding `json:"bindings"`
	Verbs       []string          `json:"verbs,omitempty"`
	Outputs     scaffoldOutputs   `json:"outputs"`
	Budget      scaffoldBudget    `json:"budget"`
	Fixtures    []scaffoldFixture `json:"fixtures"`
	Rung        string            `json:"rung"`
}

func (h *handlers) scaffoldPrimitive() cliapp.PrimitiveHandler {
	return cliapp.Action(h.scaffold, scaffoldReport)
}

func (h *handlers) scaffold(ctx cliapp.OperationContext) (scaffoldResult, error) {
	result, err := buildScaffold(strings.TrimSpace(ctx.Flag("name")), strings.TrimSpace(ctx.Flag("purpose")), strings.TrimSpace(ctx.Flag("binding")))
	if err != nil {
		return scaffoldResult{}, err
	}

	if outDir := strings.TrimSpace(ctx.Flag("out-dir")); outDir != "" {
		_, program, _ := strings.Cut(result.Name, ".")
		contractPath := filepath.Join(outDir, program+".json")
		programPath := filepath.Join(outDir, program+".py")
		if !ctx.BoolFlag("force") {
			for _, p := range []string{contractPath, programPath} {
				if _, statErr := os.Stat(p); statErr == nil {
					return scaffoldResult{}, fmt.Errorf("%s already exists; pass --force to overwrite", p)
				}
			}
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return scaffoldResult{}, fmt.Errorf("create out-dir: %w", err)
		}
		if err := os.WriteFile(contractPath, []byte(result.ContractJSON), 0o644); err != nil {
			return scaffoldResult{}, fmt.Errorf("write contract: %w", err)
		}
		if err := os.WriteFile(programPath, []byte(result.ProgramPy), 0o644); err != nil {
			return scaffoldResult{}, fmt.Errorf("write program: %w", err)
		}
		result.ContractPath, result.ProgramPath, result.Written = contractPath, programPath, true
	}

	return result, nil
}

// buildScaffold is the pure generator: it validates the inputs and renders the
// contract + program skeleton with no file I/O, so it is directly testable.
func buildScaffold(name, purpose, binding string) (scaffoldResult, error) {
	if !scaffoldNamePattern.MatchString(name) {
		return scaffoldResult{}, fmt.Errorf("--name must be <scenario>.<program> in kebab-case, e.g. device-control.volume")
	}
	if n := len(purpose); n < 20 || n > 240 {
		return scaffoldResult{}, fmt.Errorf("--purpose must be 20-240 characters (got %d)", n)
	}
	scenario, _, _ := strings.Cut(name, ".")

	bindingPattern := regexp.MustCompile(`^[a-z0-9-]+/[a-z0-9-]+/[a-z0-9-]+$`)
	var bindings []scaffoldBinding
	var verbs []string
	// errors.classes must be non-empty and list exactly the classes the program
	// can emit. The validate-phase path always emits invalid_input; a guarded
	// read classified by program.classify adds the transport classes.
	errorClasses := []string{"invalid_input"}
	if binding != "" {
		if !bindingPattern.MatchString(binding) {
			return scaffoldResult{}, fmt.Errorf("--binding must be <scenario>/<group>/<command>, e.g. vrooli/scenario/status")
		}
		bindings = []scaffoldBinding{{ID: binding, Effect: "read"}}
		verbs = []string{"gather"}
		errorClasses = []string{"invalid_input", "scenario_unreachable", "remote_error", "binding_error"}
	} else {
		bindings = []scaffoldBinding{}
	}

	contract := scaffoldContract{
		Schema:  schemaRelPath(scenario),
		Name:    name,
		Version: "1",
		Purpose: purpose,
		Helpers: map[string]string{"program": "1"},
		Inputs:  map[string]any{},
		Assumptions: []string{
			fmt.Sprintf("the %s scenario is running when a binding is read", scenario),
		},
		Invariants: []string{"the envelope is printed exactly once on every path"},
		Bindings:   bindings,
		Verbs:      verbs,
		Outputs: scaffoldOutputs{
			Status:  map[string]any{"enum": []string{"ok", "partial", "unavailable", "failed"}},
			Phase:   map[string]any{"enum": []string{"validate", "collect", "report"}},
			Signals: map[string]any{},
			Errors:  map[string]any{"classes": errorClasses},
		},
		Budget:   scaffoldBudget{WallMs: 30000, InferenceCalls: 0, DelegatedRuns: 0},
		Fixtures: []scaffoldFixture{{ID: "live", Inputs: map[string]any{}, Expect: map[string]any{"status": []string{"ok", "partial", "unavailable"}}}},
		Rung:     "S3",
	}

	contractBytes, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return scaffoldResult{}, fmt.Errorf("render contract: %w", err)
	}

	return scaffoldResult{
		Name:         name,
		Scenario:     scenario,
		ContractJSON: string(contractBytes) + "\n",
		ProgramPy:    scaffoldProgram(name, purpose, binding),
	}, nil
}

// schemaRelPath is the `$schema` reference from a program file at
// scenarios/<owner>/.vrooli/program-runtime/ to program-runtime's contract
// schema. program-runtime's own programs reach it in two hops; every other
// scenario reaches across the fleet in three.
func schemaRelPath(scenario string) string {
	if scenario == "program-runtime" {
		return "../../schemas/program-contract.schema.json"
	}
	return "../../../program-runtime/schemas/program-contract.schema.json"
}

func scaffoldProgram(name, purpose, binding string) string {
	var read string
	if binding != "" {
		// vrooli/scenario/status -> vrooli_scenario.status
		owner, rest, _ := strings.Cut(binding, "/")
		group, command, _ := strings.Cut(rest, "/")
		call := fmt.Sprintf("%s_%s.%s", strings.ReplaceAll(owner, "-", "_"), strings.ReplaceAll(group, "-", "_"), strings.ReplaceAll(command, "-", "_"))
		read = fmt.Sprintf(`    # One governed read, isolated so a failure degrades to unavailable
    # instead of aborting the program.
    result = program.guarded(lambda: %s(rows="..."))()
    if isinstance(result, Exception):
        status, klass = program.classify(result)
        return program.fail(status, klass, result, "collect")
    # envelope["signals"]["..."] = program.coerce_int(program.select_field(result, "count"))
`, call)
	} else {
		read = "    # Read governed bindings here (declare each in the contract's bindings).\n"
	}

	return fmt.Sprintf(`"""%s v1 — %s."""

# The bound `+"`program`"+` helper provides inputs/envelope/fail/guarded/classify/
# run/report and the coerce_int/coerce_float/select_field accessors. Do not
# import vrooli; call bindings through their injected scenario namespace.

envelope = program.envelope(%q, "1")
envelope["signals"] = {}


def step_validate():
    program.inputs()
    # On a bad input:
    #   return program.fail("failed", "invalid_input", "<detail>", "validate")
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
%s    envelope["status"] = "ok"
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()


STATES = {"validate": step_validate, "collect": step_collect, "report": step_report}
program.run(STATES, "validate")
`, name, purpose, name, read)
}

func scaffoldReport(_ cliapp.OperationContext, r scaffoldResult) cliapp.MutationReport {
	if r.Written {
		return cliapp.MutationReport{
			Result:      []string{fmt.Sprintf("Scaffolded %s", r.Name)},
			Changes:     []string{"Wrote " + r.ContractPath, "Wrote " + r.ProgramPath},
			NextCommand: []string{fmt.Sprintf("program-runtime library run %s --json", r.Name)},
		}
	}
	return cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Scaffold for %s (pass --out-dir to write the pair):", r.Name),
			"",
			"--- contract (" + r.Name + ".json) ---",
			r.ContractJSON,
			"--- program (" + r.Name + ".py) ---",
			r.ProgramPy,
		},
	}
}
