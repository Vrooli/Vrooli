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

func runUpdateRuntime(_ *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps update-runtime")
	var req runtimeDeclaration
	var jsonOutput bool
	fs.StringVar(&req.Scenario, "scenario", "", "Exact consuming scenario")
	fs.StringVar(&req.Dependency, "dependency", "", "Exact existing runtime source scenario")
	fs.StringVar(&req.Description, "description", "", "Purpose of the dependency")
	fs.StringVar(&req.Rationale, "rationale", "", "Why this runtime dependency has no imported scenario package")
	fs.StringVar(&req.DegradedBehavior, "degraded-behavior", "", "Behavior when the optional source is unavailable")
	fs.StringVar(&req.ExpectedSHA256, "expected-sha256", "", "Required with --apply; before_sha256 from preview")
	fs.BoolVar(&req.Apply, "apply", false, "Apply the update shown by preview")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("update-runtime takes named flags only")
	}
	result, err := updateRuntime(cliutil.ResolveRepoRoot(), req)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintReportJSON(result)
	}
	fmt.Printf("Optional runtime declaration %s -> %s: changed=%t applied=%t\nBefore SHA-256: %s\n", req.Scenario, req.Dependency, result.Changed, result.Applied, result.BeforeSHA)
	return nil
}

func updateRuntime(root string, req runtimeDeclaration) (runtimeDeclarationResult, error) {
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
	before, info, err := readRegularFileNoSymlink(path)
	if err != nil {
		return result, err
	}
	if !json.Valid(before) {
		return result, fmt.Errorf("invalid service manifest JSON")
	}
	if err := rejectDuplicateJSONKeys(before); err != nil {
		return result, err
	}
	if req.Apply && !strings.EqualFold(sha256Hex(before), req.ExpectedSHA256) {
		return result, fmt.Errorf("service manifest changed since preview; no write performed")
	}
	start, end, err := jsonObjectAt(before, []string{"dependencies", "scenarios"}, 0)
	if err != nil {
		return result, err
	}
	depStart, depEnd, raw, err := jsonMemberAt(before[start:end], req.Dependency, start)
	if err != nil {
		return result, err
	}
	if depStart < 0 {
		return result, fmt.Errorf("dependency %s is not declared; update-runtime only updates existing dependencies", req.Dependency)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return result, fmt.Errorf("dependency %s must be an object", req.Dependency)
	}
	if !jsonStringEquals(fields["type"], "scenario") || !jsonBoolEquals(fields["enabled"], true) || !jsonBoolEquals(fields["required"], false) || !jsonBoolEquals(fields["runtime_only"], true) || !jsonStringEquals(fields["startup_policy"], "ignore") {
		return result, fmt.Errorf("dependency %s is not exactly optional runtime-only with startup_policy=ignore; refusing structural change", req.Dependency)
	}
	descriptions := map[string]string{"description": req.Description, "runtime_only_rationale": req.Rationale, "degraded_behavior": req.DegradedBehavior}
	unchanged := true
	for key, expected := range descriptions {
		actual, ok := jsonStringValue(fields[key])
		if !ok || actual != expected {
			unchanged = false
			break
		}
	}
	if unchanged {
		var declaration map[string]any
		_ = json.Unmarshal(raw, &declaration)
		digest := sha256Hex(before)
		return runtimeDeclarationResult{Scenario: req.Scenario, Dependency: req.Dependency, Target: path, BeforeSHA: digest, AfterSHA: digest, Declaration: declaration}, nil
	}
	for key, value := range descriptions {
		encoded, _ := json.Marshal(value)
		fields[key] = encoded
	}
	updated, err := json.MarshalIndent(fields, "      ", "  ")
	if err != nil {
		return result, err
	}
	after := append([]byte{}, before[:depStart]...)
	after = append(after, updated...)
	after = append(after, before[depEnd:]...)
	if !json.Valid(after) {
		return result, fmt.Errorf("invalid proposed manifest")
	}
	var declaration map[string]any
	_ = json.Unmarshal(updated, &declaration)
	result = runtimeDeclarationResult{Scenario: req.Scenario, Dependency: req.Dependency, Target: path, Changed: !bytes.Equal(before, after), BeforeSHA: sha256Hex(before), AfterSHA: sha256Hex(after), Declaration: declaration}
	if !result.Changed {
		result.AfterSHA = result.BeforeSHA
		return result, nil
	}
	if !req.Apply {
		return result, nil
	}
	if err := writeAtomicallyIfUnchanged(filepath.Dir(path), path, after, info, req.ExpectedSHA256); err != nil {
		return result, err
	}
	result.Applied = true
	return result, nil
}

func jsonStringEquals(raw json.RawMessage, expected string) bool {
	value, ok := jsonStringValue(raw)
	return ok && value == expected
}

func jsonStringValue(raw json.RawMessage) (string, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return "", false
	}
	var value string
	if json.Unmarshal(trimmed, &value) != nil {
		return "", false
	}
	return value, true
}

func jsonBoolEquals(raw json.RawMessage, expected bool) bool {
	trimmed := bytes.TrimSpace(raw)
	if expected {
		return bytes.Equal(trimmed, []byte("true"))
	}
	return bytes.Equal(trimmed, []byte("false"))
}

// jsonMemberAt locates an exact object member and returns absolute byte offsets.
func jsonMemberAt(data []byte, key string, base int) (int, int, []byte, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	tok, err := d.Token()
	if err != nil || tok != json.Delim('{') {
		return -1, -1, nil, fmt.Errorf("dependency path must contain JSON objects")
	}
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return -1, -1, nil, err
		}
		k := t.(string)
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return -1, -1, nil, err
		}
		end := int(d.InputOffset())
		start := end - len(raw)
		if k == key {
			return base + start, base + end, raw, nil
		}
	}
	return -1, -1, nil, nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(d); err != nil {
		return err
	}
	if _, err := d.Token(); err == nil {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}
func scanJSONValue(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				key := k.(string)
				if seen[key] {
					return fmt.Errorf("duplicate manifest key %q", key)
				}
				seen[key] = true
				if err := scanJSONValue(d); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		case '[':
			for d.More() {
				if err := scanJSONValue(d); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
	}
	return nil
}
