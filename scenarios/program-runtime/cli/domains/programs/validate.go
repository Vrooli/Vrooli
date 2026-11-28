package programs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/cli-core/cliapp"
)

// validate performs the offline authoring checks a program author otherwise
// only discovers at submit time: the contract is validated against the
// canonical schema (resolved drift-free from the contract's own $schema
// reference), the name matches the file stem, and the program source is syntax
// checked. Deep binding-call resolution stays owned by the server preflight
// (analyze.py) at submit; validate is honest about that boundary.

type validateResult struct {
	ContractPath string   `json:"contract_path"`
	ProgramPath  string   `json:"program_path"`
	SchemaPath   string   `json:"schema_path"`
	OK           bool     `json:"ok"`
	Errors       []string `json:"errors"`
	Warnings     []string `json:"warnings"`
	Checked      []string `json:"checked"`
}

func (h *handlers) validatePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ActionWithExit(h.validate, validateReport, func(_ cliapp.OperationContext, r validateResult) error {
		if r.OK {
			return nil
		}
		return fmt.Errorf("contract validation found %d problem(s)", len(r.Errors))
	})
}

func (h *handlers) validate(ctx cliapp.OperationContext) (validateResult, error) {
	contractPath := strings.TrimSpace(ctx.Positional("contract"))
	if contractPath == "" {
		return validateResult{}, fmt.Errorf("a contract .json path is required")
	}
	programPath := strings.TrimSpace(ctx.Positional("program"))
	if programPath == "" {
		programPath = strings.TrimSuffix(contractPath, filepath.Ext(contractPath)) + ".py"
	}

	res := validateResult{ContractPath: contractPath, ProgramPath: programPath, OK: true}

	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		return validateResult{}, fmt.Errorf("read contract: %w", err)
	}
	var contract map[string]any
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		res.OK = false
		res.Errors = append(res.Errors, "contract is not valid JSON: "+err.Error())
		return res, nil
	}
	res.Checked = append(res.Checked, "contract parses as JSON")

	// Resolve the canonical schema from the contract's own $schema reference,
	// so validate uses the live schema and never a drifting embedded copy.
	schemaPath := resolveSchemaPath(contractPath, contract)
	res.SchemaPath = schemaPath
	if schemaPath == "" {
		res.OK = false
		res.Errors = append(res.Errors, "could not resolve the contract schema; set $schema in the contract or run from the repo")
	} else if schemaErrs := validateAgainstSchema(schemaPath, contract); len(schemaErrs) > 0 {
		res.OK = false
		res.Errors = append(res.Errors, schemaErrs...)
	} else {
		res.Checked = append(res.Checked, "contract validates against program-contract.schema.json")
	}

	// name must match the file stem (the loader keys programs by stem).
	if name, _ := contract["name"].(string); name != "" {
		_, program, _ := strings.Cut(name, ".")
		stem := strings.TrimSuffix(filepath.Base(contractPath), filepath.Ext(contractPath))
		if program != stem {
			res.OK = false
			res.Errors = append(res.Errors, fmt.Sprintf("contract name %q implies file stem %q but the file is %q.json", name, program, stem))
		} else {
			res.Checked = append(res.Checked, "contract name matches the file stem")
		}
	}

	// Program source: must exist, and is syntax checked when python3 is present.
	if _, statErr := os.Stat(programPath); statErr != nil {
		res.OK = false
		res.Errors = append(res.Errors, "program source not found at "+programPath)
	} else if warn := pythonSyntaxCheck(programPath); warn != "" {
		if strings.HasPrefix(warn, "skip:") {
			res.Warnings = append(res.Warnings, strings.TrimPrefix(warn, "skip:"))
		} else {
			res.OK = false
			res.Errors = append(res.Errors, warn)
		}
	} else {
		res.Checked = append(res.Checked, "program source compiles")
	}

	return res, nil
}

// resolveSchemaPath resolves the schema from the contract's relative $schema
// reference, falling back to a walk up to scenarios/program-runtime/schemas.
func resolveSchemaPath(contractPath string, contract map[string]any) string {
	dir := filepath.Dir(contractPath)
	if ref, _ := contract["$schema"].(string); ref != "" {
		candidate := filepath.Clean(filepath.Join(dir, ref))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	cur, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(cur, "scenarios", "program-runtime", "schemas", "program-contract.schema.json")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

func validateAgainstSchema(schemaPath string, instance map[string]any) []string {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return []string{"could not read schema " + schemaPath + ": " + err.Error()}
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaPath, bytes.NewReader(schemaBytes)); err != nil {
		return []string{"could not load schema " + schemaPath + ": " + err.Error()}
	}
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		return []string{"could not compile schema " + schemaPath + ": " + err.Error()}
	}
	if err := schema.Validate(instance); err != nil {
		var ve *jsonschema.ValidationError
		if asValidation(err, &ve) {
			out := make([]string, 0, len(ve.Causes)+1)
			for _, cause := range flattenCauses(ve) {
				out = append(out, "schema: "+cause)
			}
			if len(out) == 0 {
				out = append(out, "schema: "+ve.Error())
			}
			return out
		}
		return []string{"schema: " + err.Error()}
	}
	return nil
}

func asValidation(err error, target **jsonschema.ValidationError) bool {
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		*target = ve
		return true
	}
	return false
}

// flattenCauses turns the nested validation tree into one line per leaf
// violation, each naming the failing instance location.
func flattenCauses(ve *jsonschema.ValidationError) []string {
	if len(ve.Causes) == 0 {
		loc := ve.InstanceLocation
		if loc == "" {
			loc = "(root)"
		}
		return []string{loc + ": " + ve.Message}
	}
	var out []string
	for _, c := range ve.Causes {
		out = append(out, flattenCauses(c)...)
	}
	return out
}

// pythonSyntaxCheck compiles the program with python3 when available. A missing
// interpreter is a skip (returned with a "skip:" prefix), not a failure.
func pythonSyntaxCheck(programPath string) string {
	python, err := exec.LookPath("python3")
	if err != nil {
		return "skip:python3 not found; program syntax was not checked"
	}
	cmd := exec.Command(python, "-m", "py_compile", programPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "program does not compile: " + msg
	}
	return ""
}

func validateReport(_ cliapp.OperationContext, r validateResult) cliapp.MutationReport {
	if r.OK {
		result := []string{fmt.Sprintf("OK — %s is valid.", filepath.Base(r.ContractPath))}
		for _, c := range r.Checked {
			result = append(result, "  ✓ "+c)
		}
		for _, w := range r.Warnings {
			result = append(result, "  ! "+w)
		}
		return cliapp.MutationReport{Result: result}
	}
	result := []string{fmt.Sprintf("INVALID — %s has %d problem(s):", filepath.Base(r.ContractPath), len(r.Errors))}
	for _, e := range r.Errors {
		result = append(result, "  ✗ "+e)
	}
	for _, w := range r.Warnings {
		result = append(result, "  ! "+w)
	}
	return cliapp.MutationReport{Result: result}
}
