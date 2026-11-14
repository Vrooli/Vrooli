package depsapproved

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"scenario-dependency-analyzer/cli/internal/support"
)

// Runtime declarations are explicit operator contracts, not inferred edges.
// This offline owner operation cannot launch the dependency or call its API.
type runtimeDeclaration struct {
	Scenario, Dependency, Description, Rationale, DegradedBehavior, ExpectedSHA256 string
	Apply                                                                          bool
}
type runtimeDeclarationResult struct {
	Scenario    string         `json:"scenario"`
	Dependency  string         `json:"dependency"`
	Target      string         `json:"target"`
	Applied     bool           `json:"applied"`
	Changed     bool           `json:"changed"`
	BeforeSHA   string         `json:"before_sha256"`
	AfterSHA    string         `json:"after_sha256"`
	Declaration map[string]any `json:"declaration"`
}

func runDeclareRuntime(_ *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps declare-runtime")
	var req runtimeDeclaration
	var jsonOutput bool
	fs.StringVar(&req.Scenario, "scenario", "", "Exact consuming scenario")
	fs.StringVar(&req.Dependency, "dependency", "", "Exact existing runtime source scenario")
	fs.StringVar(&req.Description, "description", "", "Purpose of the dependency")
	fs.StringVar(&req.Rationale, "rationale", "", "Why this runtime dependency has no imported scenario package")
	fs.StringVar(&req.DegradedBehavior, "degraded-behavior", "", "Behavior when the optional source is unavailable")
	fs.StringVar(&req.ExpectedSHA256, "expected-sha256", "", "Required with --apply; before_sha256 from preview")
	fs.BoolVar(&req.Apply, "apply", false, "Apply the exact optional runtime-only ignore declaration")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("declare-runtime takes named flags only")
	}
	result, err := declareRuntime(cliutil.ResolveRepoRoot(), req)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintReportJSON(result)
	}
	fmt.Printf("Optional runtime declaration %s -> %s (startup_policy=ignore): changed=%t applied=%t\nBefore SHA-256: %s\n", req.Scenario, req.Dependency, result.Changed, result.Applied, result.BeforeSHA)
	return nil
}

func declareRuntime(root string, req runtimeDeclaration) (runtimeDeclarationResult, error) {
	var result runtimeDeclarationResult
	slug := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	if !slug.MatchString(req.Scenario) || !slug.MatchString(req.Dependency) || req.Scenario == req.Dependency {
		return result, fmt.Errorf("two distinct exact scenario slugs are required")
	}
	for _, v := range []string{req.Description, req.Rationale, req.DegradedBehavior} {
		if strings.TrimSpace(v) == "" {
			return result, fmt.Errorf("description, rationale and degraded-behavior are required")
		}
	}
	if req.Apply && !validSHA256(req.ExpectedSHA256) {
		return result, fmt.Errorf("--apply requires --expected-sha256 from preview")
	}
	if !req.Apply && req.ExpectedSHA256 != "" {
		return result, fmt.Errorf("--expected-sha256 requires --apply")
	}
	// Reject symlinked paths rather than editing a different checkout or scenario.
	for _, name := range []string{req.Scenario, req.Dependency} {
		for _, part := range []string{"scenarios", filepath.Join("scenarios", name), filepath.Join("scenarios", name, ".vrooli"), filepath.Join("scenarios", name, ".vrooli", "service.json")} {
			info, err := os.Lstat(filepath.Join(root, part))
			if err != nil {
				return result, err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return result, fmt.Errorf("symlinked scenario path is not supported: %s", part)
			}
		}
	}
	path := filepath.Join(root, "scenarios", req.Scenario, ".vrooli", "service.json")
	before, info, err := readRegularGoMod(path)
	if err != nil {
		return result, err
	}
	if !json.Valid(before) {
		return result, fmt.Errorf("invalid service manifest JSON")
	}
	if req.Apply && !strings.EqualFold(sha256Hex(before), req.ExpectedSHA256) {
		return result, fmt.Errorf("service manifest changed since preview; no write performed")
	}
	entry := map[string]any{"type": "scenario", "enabled": true, "required": false, "startup_policy": "ignore", "runtime_only": true, "description": req.Description, "runtime_only_rationale": req.Rationale, "degraded_behavior": req.DegradedBehavior}
	// Locate the authoritative nested object, rejecting duplicate JSON keys along
	// the path. Insert only the selected member; retain every existing byte.
	start, end, err := jsonObjectAt(before, []string{"dependencies", "scenarios"}, 0)
	if err != nil {
		return result, err
	}
	var existing map[string]json.RawMessage
	if err = json.Unmarshal(before[start:end], &existing); err != nil {
		return result, err
	}
	if raw, ok := existing[req.Dependency]; ok {
		var current map[string]any
		if err = json.Unmarshal(raw, &current); err != nil {
			return result, err
		}
		a, _ := json.Marshal(current)
		b, _ := json.Marshal(entry)
		if !bytes.Equal(a, b) {
			return result, fmt.Errorf("dependency %s already exists with a different contract; refusing to overwrite", req.Dependency)
		}
		return runtimeDeclarationResult{Scenario: req.Scenario, Dependency: req.Dependency, Target: path, BeforeSHA: sha256Hex(before), AfterSHA: sha256Hex(before), Declaration: entry}, nil
	}
	encoded, err := json.MarshalIndent(entry, "      ", "  ")
	if err != nil {
		return result, err
	}
	key, _ := json.Marshal(req.Dependency)
	insert := bytes.TrimRight(before[start:end-1], " \t\r\n")
	suffix := ""
	if len(existing) > 0 {
		suffix = ","
	}
	after := append([]byte{}, before[:start]...)
	after = append(after, insert...)
	after = append(after, []byte(suffix+"\n      "+string(key)+": "+string(encoded)+"\n    ")...)
	after = append(after, before[end-1:]...)
	if !json.Valid(after) {
		return result, fmt.Errorf("invalid proposed manifest")
	}
	result = runtimeDeclarationResult{Scenario: req.Scenario, Dependency: req.Dependency, Target: path, Changed: true, BeforeSHA: sha256Hex(before), AfterSHA: sha256Hex(after), Declaration: entry}
	if !req.Apply {
		return result, nil
	}
	if err = writeGoModIfUnchanged(filepath.Dir(path), path, after, info, req.ExpectedSHA256); err != nil {
		return result, err
	}
	result.Applied = true
	return result, nil
}

// jsonObjectAt returns absolute byte bounds, preserving formatting and unknown
// manifest fields. Decode consumes each complete value, so nested similarly
// named keys cannot redirect the selected dependency object.
func jsonObjectAt(data []byte, path []string, base int) (int, int, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return 0, 0, fmt.Errorf("dependency path must contain JSON objects")
	}
	seen := map[string]bool{}
	foundStart, foundEnd := -1, -1
	for d.More() {
		keyToken, err := d.Token()
		if err != nil {
			return 0, 0, err
		}
		key := keyToken.(string)
		if seen[key] {
			return 0, 0, fmt.Errorf("duplicate manifest key %q", key)
		}
		seen[key] = true
		var raw json.RawMessage
		if err = d.Decode(&raw); err != nil {
			return 0, 0, err
		}
		end := int(d.InputOffset())
		start := end - len(raw)
		if len(path) > 0 && key == path[0] {
			foundStart, foundEnd, err = jsonObjectAt(raw, path[1:], base+start)
			if err != nil {
				return 0, 0, err
			}
		}
	}
	if _, err = d.Token(); err != nil {
		return 0, 0, err
	}
	if len(path) == 0 {
		return base, base + int(d.InputOffset()), nil
	}
	if foundStart < 0 {
		return 0, 0, fmt.Errorf("manifest is missing dependency object %q", strings.Join(path, "."))
	}
	return foundStart, foundEnd, nil
}
