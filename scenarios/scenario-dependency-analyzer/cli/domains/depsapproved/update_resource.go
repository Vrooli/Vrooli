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

type resourceUpdate struct {
	Scenario, Resource, Description, DegradedBehavior, StartupPolicy, ExpectedSHA256 string
	Apply                                                                            bool
}

type resourceUpdateResult struct {
	Scenario    string         `json:"scenario"`
	Resource    string         `json:"resource"`
	Target      string         `json:"target"`
	Applied     bool           `json:"applied"`
	Changed     bool           `json:"changed"`
	BeforeSHA   string         `json:"before_sha256"`
	AfterSHA    string         `json:"after_sha256"`
	Declaration map[string]any `json:"declaration"`
}

func runUpdateResource(_ *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps update-resource")
	var req resourceUpdate
	var jsonOutput bool
	fs.StringVar(&req.Scenario, "scenario", "", "Exact consuming scenario")
	fs.StringVar(&req.Resource, "resource", "", "Exact existing resource slug")
	fs.StringVar(&req.Description, "description", "", "Purpose of the optional resource")
	fs.StringVar(&req.DegradedBehavior, "degraded-behavior", "", "Behavior when the resource is absent or unhealthy")
	fs.StringVar(&req.StartupPolicy, "startup-policy", "", "Optional resource start policy: ignore or try_start")
	fs.StringVar(&req.ExpectedSHA256, "expected-sha256", "", "Required with --apply; before_sha256 from preview")
	fs.BoolVar(&req.Apply, "apply", false, "Apply the update shown by preview")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("update-resource takes named flags only")
	}
	result, err := updateResource(cliutil.ResolveRepoRoot(), req)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintReportJSON(result)
	}
	fmt.Printf("Optional resource dependency %s -> %s: changed=%t applied=%t\nBefore SHA-256: %s\n", req.Scenario, req.Resource, result.Changed, result.Applied, result.BeforeSHA)
	return nil
}

func updateResource(root string, req resourceUpdate) (resourceUpdateResult, error) {
	var result resourceUpdateResult
	slug := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	if !slug.MatchString(req.Scenario) || !slug.MatchString(req.Resource) {
		return result, fmt.Errorf("exact scenario and resource slugs are required")
	}
	if req.StartupPolicy != "" && req.StartupPolicy != "ignore" && req.StartupPolicy != "try_start" {
		return result, fmt.Errorf("startup-policy must be ignore or try_start")
	}
	if strings.TrimSpace(req.Description) == "" && strings.TrimSpace(req.DegradedBehavior) == "" && req.StartupPolicy == "" {
		return result, fmt.Errorf("at least one update field is required")
	}
	if (req.Description != "" && strings.TrimSpace(req.Description) == "") || (req.DegradedBehavior != "" && strings.TrimSpace(req.DegradedBehavior) == "") {
		return result, fmt.Errorf("provided descriptive fields must not be blank")
	}
	if req.Apply && !validSHA256(req.ExpectedSHA256) {
		return result, fmt.Errorf("--apply requires --expected-sha256 from preview")
	}
	if !req.Apply && req.ExpectedSHA256 != "" {
		return result, fmt.Errorf("--expected-sha256 requires --apply")
	}
	for _, part := range []string{"scenarios", filepath.Join("scenarios", req.Scenario), filepath.Join("scenarios", req.Scenario, ".vrooli"), filepath.Join("scenarios", req.Scenario, ".vrooli", "service.json"), "resources", filepath.Join("resources", req.Resource), filepath.Join("resources", req.Resource, "resource.json")} {
		info, err := os.Lstat(filepath.Join(root, part))
		if err != nil {
			return result, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return result, fmt.Errorf("symlinked dependency path is not supported: %s", part)
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
	start, end, err := jsonObjectAt(before, []string{"dependencies", "resources"}, 0)
	if err != nil {
		return result, err
	}
	depStart, depEnd, raw, err := jsonMemberAt(before[start:end], req.Resource, start)
	if err != nil {
		return result, err
	}
	if depStart < 0 {
		return result, fmt.Errorf("resource %s is not declared; update-resource only updates existing dependencies", req.Resource)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return result, fmt.Errorf("resource dependency %s must be an object", req.Resource)
	}
	if (len(fields["enabled"]) > 0 && !jsonBoolEquals(fields["enabled"], true)) || !jsonBoolEquals(fields["required"], false) || (!jsonStringEquals(fields["startup_policy"], "ignore") && !jsonStringEquals(fields["startup_policy"], "try_start")) {
		return result, fmt.Errorf("resource %s is not an enabled optional dependency with a supported startup_policy; refusing structural change", req.Resource)
	}
	values := map[string]string{}
	if req.Description != "" {
		values["description"] = req.Description
	}
	if req.DegradedBehavior != "" {
		values["degraded_behavior"] = req.DegradedBehavior
	}
	if req.StartupPolicy != "" {
		values["startup_policy"] = req.StartupPolicy
	}
	unchanged := true
	for key, value := range values {
		actual, ok := jsonStringValue(fields[key])
		if !ok || actual != value {
			unchanged = false
			break
		}
	}
	if unchanged {
		var declaration map[string]any
		_ = json.Unmarshal(raw, &declaration)
		digest := sha256Hex(before)
		return resourceUpdateResult{Scenario: req.Scenario, Resource: req.Resource, Target: path, BeforeSHA: digest, AfterSHA: digest, Declaration: declaration}, nil
	}
	for key, value := range values {
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
	result = resourceUpdateResult{Scenario: req.Scenario, Resource: req.Resource, Target: path, Changed: !bytes.Equal(before, after), BeforeSHA: sha256Hex(before), AfterSHA: sha256Hex(after), Declaration: declaration}
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
