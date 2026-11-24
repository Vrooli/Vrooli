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

type resourceDeclaration struct {
	Scenario, Resource, Description, DegradedBehavior, StartupPolicy, ExpectedSHA256 string
	Apply                                                                            bool
}

type resourceDeclarationResult struct {
	Scenario    string         `json:"scenario"`
	Resource    string         `json:"resource"`
	Target      string         `json:"target"`
	Applied     bool           `json:"applied"`
	Changed     bool           `json:"changed"`
	BeforeSHA   string         `json:"before_sha256"`
	AfterSHA    string         `json:"after_sha256"`
	Declaration map[string]any `json:"declaration"`
}

func runDeclareResource(_ *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps declare-resource")
	var req resourceDeclaration
	var jsonOutput bool
	fs.StringVar(&req.Scenario, "scenario", "", "Exact consuming scenario")
	fs.StringVar(&req.Resource, "resource", "", "Exact existing resource slug")
	fs.StringVar(&req.Description, "description", "", "Purpose of the optional resource")
	fs.StringVar(&req.DegradedBehavior, "degraded-behavior", "", "Behavior when the resource is absent or unhealthy")
	fs.StringVar(&req.StartupPolicy, "startup-policy", "ignore", "Resource start policy: ignore or try_start")
	fs.StringVar(&req.ExpectedSHA256, "expected-sha256", "", "Required with --apply; before_sha256 from preview")
	fs.BoolVar(&req.Apply, "apply", false, "Apply the exact optional resource declaration")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("declare-resource takes named flags only")
	}
	result, err := declareResource(cliutil.ResolveRepoRoot(), req)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintReportJSON(result)
	}
	fmt.Printf("Optional resource declaration %s -> %s (startup_policy=%s): changed=%t applied=%t\nBefore SHA-256: %s\n", req.Scenario, req.Resource, req.StartupPolicy, result.Changed, result.Applied, result.BeforeSHA)
	return nil
}

func declareResource(root string, req resourceDeclaration) (resourceDeclarationResult, error) {
	var result resourceDeclarationResult
	slug := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	if !slug.MatchString(req.Scenario) || !slug.MatchString(req.Resource) {
		return result, fmt.Errorf("exact scenario and resource slugs are required")
	}
	if req.StartupPolicy != "ignore" && req.StartupPolicy != "try_start" {
		return result, fmt.Errorf("startup-policy must be ignore or try_start")
	}
	if strings.TrimSpace(req.Description) == "" || strings.TrimSpace(req.DegradedBehavior) == "" {
		return result, fmt.Errorf("description and degraded-behavior are required")
	}
	if req.Apply && !validSHA256(req.ExpectedSHA256) {
		return result, fmt.Errorf("--apply requires --expected-sha256 from preview")
	}
	if !req.Apply && req.ExpectedSHA256 != "" {
		return result, fmt.Errorf("--expected-sha256 requires --apply")
	}
	for _, part := range []string{
		"scenarios",
		filepath.Join("scenarios", req.Scenario),
		filepath.Join("scenarios", req.Scenario, ".vrooli"),
		filepath.Join("scenarios", req.Scenario, ".vrooli", "service.json"),
		"resources",
		filepath.Join("resources", req.Resource),
		filepath.Join("resources", req.Resource, "resource.json"),
	} {
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
	if req.Apply && !strings.EqualFold(sha256Hex(before), req.ExpectedSHA256) {
		return result, fmt.Errorf("service manifest changed since preview; no write performed")
	}
	entry := map[string]any{
		"enabled":           true,
		"required":          false,
		"startup_policy":    req.StartupPolicy,
		"description":       req.Description,
		"degraded_behavior": req.DegradedBehavior,
	}
	depsStart, depsEnd, err := jsonObjectAt(before, []string{"dependencies"}, 0)
	if err != nil {
		return result, err
	}
	var dependencies map[string]json.RawMessage
	if err = json.Unmarshal(before[depsStart:depsEnd], &dependencies); err != nil {
		return result, err
	}
	start, end, hasResources := 0, 0, false
	if _, ok := dependencies["resources"]; ok {
		start, end, err = jsonObjectAt(before, []string{"dependencies", "resources"}, 0)
		if err != nil {
			return result, err
		}
		hasResources = true
	}
	var existing map[string]json.RawMessage
	if hasResources {
		if err = json.Unmarshal(before[start:end], &existing); err != nil {
			return result, err
		}
	}
	if raw, ok := existing[req.Resource]; ok {
		var current map[string]any
		if err = json.Unmarshal(raw, &current); err != nil {
			return result, err
		}
		a, _ := json.Marshal(current)
		b, _ := json.Marshal(entry)
		if !bytes.Equal(a, b) {
			return result, fmt.Errorf("resource %s already exists with a different contract; refusing to overwrite", req.Resource)
		}
		return resourceDeclarationResult{Scenario: req.Scenario, Resource: req.Resource, Target: path, BeforeSHA: sha256Hex(before), AfterSHA: sha256Hex(before), Declaration: entry}, nil
	}
	encoded, err := json.MarshalIndent(entry, "      ", "  ")
	if err != nil {
		return result, err
	}
	key, _ := json.Marshal(req.Resource)
	var after []byte
	if hasResources {
		insert := bytes.TrimRight(before[start:end-1], " \t\r\n")
		suffix := ""
		if len(existing) > 0 {
			suffix = ","
		}
		after = append([]byte{}, before[:start]...)
		after = append(after, insert...)
		after = append(after, []byte(suffix+"\n      "+string(key)+": "+string(encoded)+"\n    ")...)
		after = append(after, before[end-1:]...)
	} else {
		insert := bytes.TrimRight(before[depsStart:depsEnd-1], " \t\r\n")
		suffix := ""
		if len(dependencies) > 0 {
			suffix = ","
		}
		after = append([]byte{}, before[:depsStart]...)
		after = append(after, insert...)
		after = append(after, []byte(suffix+"\n    \"resources\": {\n      "+string(key)+": "+string(encoded)+"\n    }\n  ")...)
		after = append(after, before[depsEnd-1:]...)
	}
	if !json.Valid(after) {
		return result, fmt.Errorf("invalid proposed manifest")
	}
	result = resourceDeclarationResult{Scenario: req.Scenario, Resource: req.Resource, Target: path, Changed: true, BeforeSHA: sha256Hex(before), AfterSHA: sha256Hex(after), Declaration: entry}
	if !req.Apply {
		return result, nil
	}
	if err = writeAtomicallyIfUnchanged(filepath.Dir(path), path, after, info, req.ExpectedSHA256); err != nil {
		return result, err
	}
	result.Applied = true
	return result, nil
}
