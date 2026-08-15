package wizard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"golang.org/x/term"
)

// Selection is the stable automation document. It names operator intent and
// deliberately does not mirror the internal operator-state schema.
type Selection struct {
	ActiveProfile     string          `json:"active_profile,omitempty"`
	Scenarios         []string        `json:"scenarios"`
	ScenarioState     map[string]bool `json:"scenario_state,omitempty"`
	OptionalResources []string        `json:"optional_resources,omitempty"`
	Resources         map[string]bool `json:"resources,omitempty"`
	Host              struct {
		Tools      []string `json:"tools,omitempty"`
		Safeguards []string `json:"safeguards,omitempty"`
	} `json:"host,omitempty"`
	HostTools      map[string]bool `json:"host_tools,omitempty"`
	HostSafeguards map[string]bool `json:"host_safeguards,omitempty"`
	OperatingMode  map[string]struct {
		AutoRestart bool `json:"auto_restart"`
	} `json:"operating_mode,omitempty"`
	Apply bool `json:"apply,omitempty"`
}

// selectionPatch is the one translation from the stable automation document
// to operator-state fields. Keeping it pure makes the interactive and
// declarative surfaces provably share the same write contract.
func selectionPatch(selection Selection) map[string]any {
	patch := map[string]any{"scenarios": map[string]any{}}
	if strings.TrimSpace(selection.ActiveProfile) != "" {
		patch["active_profile"] = selection.ActiveProfile
	}
	scenarios := patch["scenarios"].(map[string]any)
	for name, enabled := range selection.ScenarioState {
		scenarios[name] = map[string]any{"enabled": enabled}
	}
	for _, name := range selection.Scenarios {
		scenarios[name] = map[string]any{"enabled": true}
	}
	if len(selection.Resources) > 0 || len(selection.OptionalResources) > 0 {
		resources := map[string]any{}
		for name, enabled := range selection.Resources {
			resources[name] = map[string]any{"enabled": enabled}
		}
		for _, name := range selection.OptionalResources {
			resources[name] = map[string]any{"enabled": true}
		}
		patch["resources"] = resources
	}
	if len(selection.Host.Tools) > 0 || len(selection.HostTools) > 0 {
		hostTools := map[string]any{}
		for _, name := range selection.Host.Tools {
			hostTools[name] = map[string]any{"opted_in": true}
		}
		for name, optedIn := range selection.HostTools {
			hostTools[name] = map[string]any{"opted_in": optedIn}
		}
		patch["host_tools"] = hostTools
	}
	if len(selection.Host.Safeguards) > 0 || len(selection.HostSafeguards) > 0 {
		hostSafeguards := map[string]any{}
		for _, name := range selection.Host.Safeguards {
			hostSafeguards[name] = map[string]any{"opted_in": true}
		}
		for name, optedIn := range selection.HostSafeguards {
			hostSafeguards[name] = map[string]any{"opted_in": optedIn}
		}
		patch["host_safeguards"] = hostSafeguards
	}
	for name, mode := range selection.OperatingMode {
		scenarios[name] = map[string]any{"enabled": true, "auto_restart": mode.AutoRestart}
	}
	return patch
}

func ensureModeMap(value map[string]struct {
	AutoRestart bool `json:"auto_restart"`
}) map[string]struct {
	AutoRestart bool `json:"auto_restart"`
} {
	if value == nil {
		return make(map[string]struct {
			AutoRestart bool `json:"auto_restart"`
		})
	}
	return value
}

func hostNames(names []string) string { return strings.Join(names, ", ") }

type hostItem struct {
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Risk      string `json:"risk"`
	Privilege string `json:"privilege"`
}

type operatorInputRequest struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Default     string   `json:"default"`
	Options     []string `json:"options"`
	Required    bool     `json:"required"`
}

type operatorInputQueue struct {
	Requests []operatorInputRequest `json:"requests"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "wizard", Description: "Configure an installation through the onboarding API", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show the computed onboarding step and committed state", Run: func(args []string) error { return get(core, args, "/v2/session") }},
		{Name: "apply", Description: "Commit a selection document and apply it", Run: func(args []string) error { return apply(core, args) }},
		{Name: "export", Description: "Export the current manifest-derived selection", Run: func(args []string) error { return exportSelection(core, args) }},
		{Name: "run", Description: "Walk the same nine capability steps used by the UI", Run: func(args []string) error { return runWizard(core, args) }},
	}}
}

func runWizard(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard run")
	interactive := fs.Bool("interactive", false, "Walk all nine onboarding steps in the terminal")
	acceptRecommendation := fs.Bool("accept-recommendation", false, "Use the manifest-derived starter profile without asking for scenario names")
	nonInteractive := fs.Bool("non-interactive", false, "Never read input; return a typed needs-input error when a decision is required")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *nonInteractive && !*acceptRecommendation {
		_, _ = fmt.Fprintln(os.Stdout, `{"status":"needs_input","reason":"use --accept-recommendation or run with --interactive"}`)
		return fmt.Errorf("wizard needs operator input; rerun with --accept-recommendation or --interactive")
	}
	if *acceptRecommendation {
		return applyRecommendation(core)
	}
	if !*interactive {
		return get(core, args, "/v2/scenarios")
	}
	reader := bufio.NewReader(os.Stdin)
	read := func(step int, prompt string) (string, error) {
		if _, err := fmt.Fprintf(os.Stdout, "Step %d — %s\n> ", step, prompt); err != nil {
			return "", err
		}
		line, err := reader.ReadString('\n')
		return strings.TrimSpace(line), err
	}
	readSecret := func(step int, prompt string) (string, error) {
		if _, err := fmt.Fprintf(os.Stdout, "Step %d — %s\n> ", step, prompt); err != nil {
			return "", err
		}
		if term.IsTerminal(int(os.Stdin.Fd())) {
			value, err := term.ReadPassword(int(os.Stdin.Fd()))
			_, _ = fmt.Fprintln(os.Stdout)
			return strings.TrimSpace(string(value)), err
		}
		line, err := reader.ReadString('\n')
		return strings.TrimSpace(line), err
	}
	scenarios, err := core.Get("/v2/scenarios", nil)
	if err != nil {
		return err
	}
	var scenarioResponse struct {
		Scenarios []struct {
			Name string `json:"name"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(scenarios, &scenarioResponse); err != nil {
		return fmt.Errorf("decode scenarios: %w", err)
	}
	selection := Selection{ScenarioState: map[string]bool{}, ActiveProfile: "starter"}
	recommendationBody, err := core.Get("/v2/recommendation", nil)
	if err != nil {
		return err
	}
	var recommendation struct {
		Profile   string   `json:"profile"`
		Scenarios []string `json:"scenarios"`
		Resources []string `json:"resources"`
	}
	if err := json.Unmarshal(recommendationBody, &recommendation); err != nil {
		return fmt.Errorf("decode recommendation: %w", err)
	}
	for _, name := range recommendation.Scenarios {
		selection.ScenarioState[name] = true
	}
	selection.Scenarios = append(selection.Scenarios, recommendation.Scenarios...)
	selection.OptionalResources = append(selection.OptionalResources, recommendation.Resources...)
	selection.Resources = map[string]bool{}
	for _, name := range recommendation.Resources {
		selection.Resources[name] = true
	}
	if _, err := read(1, "welcome; press enter to begin the onboarding steps"); err != nil {
		return err
	}
	known := map[string]bool{}
	names := make([]string, 0, len(scenarioResponse.Scenarios))
	for _, scenario := range scenarioResponse.Scenarios {
		known[scenario.Name] = true
		names = append(names, scenario.Name)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Available scenarios:", strings.Join(names, ", "))
	selectedLine, err := read(2, "select scenario names (comma separated; press enter to accept the starter profile)")
	if err != nil {
		return err
	}
	if strings.TrimSpace(selectedLine) != "" {
		selection.Scenarios = nil
		selection.ScenarioState = map[string]bool{}
	}
	for _, name := range strings.Split(selectedLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" && !known[name] {
			return fmt.Errorf("unknown scenario %q; choose from %s", name, strings.Join(names, ", "))
		}
		if name != "" {
			selection.Scenarios = append(selection.Scenarios, name)
			selection.ScenarioState[name] = true
		}
	}
	resourcesBody, err := core.Get("/v2/resources", nil)
	if err != nil {
		return err
	}
	var resourceResponse struct {
		Optional []struct {
			Name string `json:"name"`
		} `json:"optional"`
		Standalone []struct {
			Name string `json:"name"`
		} `json:"standalone"`
	}
	if err := json.Unmarshal(resourcesBody, &resourceResponse); err != nil {
		return fmt.Errorf("decode resources: %w", err)
	}
	optionalNames := make([]string, 0, len(resourceResponse.Optional)+len(resourceResponse.Standalone))
	for _, resource := range resourceResponse.Optional {
		optionalNames = append(optionalNames, resource.Name)
	}
	for _, resource := range resourceResponse.Standalone {
		optionalNames = append(optionalNames, resource.Name)
	}
	sort.Strings(optionalNames)
	_, _ = fmt.Fprintln(os.Stdout, "Optional/standalone resources:", strings.Join(optionalNames, ", "))
	resourceLine, err := read(3, "select optional or standalone resources (comma separated; press enter to keep the recommendation)")
	if err != nil {
		return err
	}
	knownResources := map[string]bool{}
	for _, name := range optionalNames {
		knownResources[name] = true
	}
	if strings.TrimSpace(resourceLine) != "" {
		selection.OptionalResources = nil
		selection.Resources = map[string]bool{}
	}
	for _, name := range strings.Split(resourceLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" && !knownResources[name] {
			return fmt.Errorf("unknown resource %q; choose from %s", name, strings.Join(optionalNames, ", "))
		}
		if name != "" {
			selection.OptionalResources = append(selection.OptionalResources, name)
			if selection.Resources == nil {
				selection.Resources = map[string]bool{}
			}
			selection.Resources[name] = true
		}
	}
	if err := resolvePendingOperatorInputs(core, read, readSecret); err != nil {
		return err
	}
	if _, err := read(4, "credentials are listed by the API; provision values with credentials provision, then press enter"); err != nil {
		return err
	}
	credentialsBody, err := core.Get("/v2/credentials", nil)
	if err != nil {
		return err
	}
	var credentialResponse struct {
		Credentials []struct {
			LogicalID string `json:"logical_id"`
			Field     string `json:"field"`
			Label     string `json:"label"`
			Required  bool   `json:"required"`
			Status    string `json:"status"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(credentialsBody, &credentialResponse); err != nil {
		return fmt.Errorf("decode credentials: %w", err)
	}
	for _, credential := range credentialResponse.Credentials {
		if credential.Status == "configured" {
			continue
		}
		label := credential.Label
		if label == "" {
			label = credential.LogicalID + "/" + credential.Field
		}
		value, readErr := readSecret(4, fmt.Sprintf("enter %s; leave blank to defer (value is never printed)", label))
		if readErr != nil {
			return readErr
		}
		if value == "" {
			if credential.Required {
				_, _ = fmt.Fprintln(os.Stdout, "Required credential deferred; readiness will remain blocked.")
			}
			continue
		}
		body, marshalErr := json.Marshal(map[string]string{"logical_id": credential.LogicalID, "field": credential.Field, "value": value})
		if marshalErr != nil {
			return marshalErr
		}
		if _, requestErr := core.Request("POST", "/v2/credentials/provision", nil, body); requestErr != nil {
			return fmt.Errorf("provision %s: %w", label, requestErr)
		}
		_, _ = fmt.Fprintln(os.Stdout, "Credential stored through the native authority:", label)
	}
	if _, err := read(5, "integration binding is deferred; press enter to continue"); err != nil {
		return err
	}
	hostBody, err := core.Get("/v2/host-requirements", nil)
	if err != nil {
		return err
	}
	var hostResponse struct {
		Tools      []hostItem `json:"tools"`
		Safeguards []hostItem `json:"safeguards"`
	}
	if err := json.Unmarshal(hostBody, &hostResponse); err != nil {
		return fmt.Errorf("decode host requirements: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Host tools:", describeHostItems(hostResponse.Tools))
	toolLine, err := read(6, "select optional host tools (comma separated; required tools are automatic)")
	if err != nil {
		return err
	}
	for _, name := range strings.Split(toolLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			if !containsHost(hostResponse.Tools, name) {
				return fmt.Errorf("unknown host tool %q", name)
			}
			selection.Host.Tools = append(selection.Host.Tools, name)
			if selection.HostTools == nil {
				selection.HostTools = map[string]bool{}
			}
			selection.HostTools[name] = true
		}
	}
	_, _ = fmt.Fprintln(os.Stdout, "Host safeguards:", describeHostItems(hostResponse.Safeguards))
	safeguardLine, err := read(6, "select optional host safeguards (comma separated; press enter for none)")
	if err != nil {
		return err
	}
	for _, name := range strings.Split(safeguardLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			if !containsSafeguard(hostResponse.Safeguards, name) {
				return fmt.Errorf("unknown host safeguard %q", name)
			}
			selection.Host.Safeguards = append(selection.Host.Safeguards, name)
			if selection.HostSafeguards == nil {
				selection.HostSafeguards = map[string]bool{}
			}
			selection.HostSafeguards[name] = true
		}
	}
	modeLine, err := read(7, "choose operating mode: enter scenario names for auto-restart (comma separated)")
	if err != nil {
		return err
	}
	for _, name := range strings.Split(modeLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" && known[name] {
			selection.OperatingMode = ensureModeMap(selection.OperatingMode)
			selection.OperatingMode[name] = struct {
				AutoRestart bool `json:"auto_restart"`
			}{AutoRestart: true}
		}
	}
	confirmation, err := read(8, "apply this selection now? answer yes or no; press enter for yes")
	if err != nil {
		return err
	}
	if strings.TrimSpace(confirmation) != "" && strings.ToLower(strings.TrimSpace(confirmation)) != "yes" {
		return fmt.Errorf("selection not applied; answer yes to commit the wizard selection")
	}
	if _, err := read(9, "validation will run after apply; press enter to print final status"); err != nil {
		return err
	}
	selection.Apply = true
	patch := selectionPatch(selection)
	patchBody, _ := json.Marshal(patch)
	if _, err := core.Request("PATCH", "/v2/operator-state", nil, patchBody); err != nil {
		return err
	}
	planBody, err := core.Get("/v2/apply/plan", nil)
	if err != nil {
		return err
	}
	if err := printApplyPlan(planBody); err != nil {
		return err
	}
	applyResult, err := core.Request("POST", "/v2/apply", nil, []byte("{}"))
	if err != nil {
		return err
	}
	var applyResponse struct {
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(applyResult, &applyResponse); err == nil && applyResponse.RunID != "" {
		applyResult, err = waitForApply(core, applyResponse.RunID, applyResponse.Status)
		if err != nil {
			return err
		}
	}
	readinessResult, err := core.Get("/v2/readiness", nil)
	if err != nil {
		return err
	}
	if err := printApplyReport(applyResult); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Readiness:", string(readinessResult))
	return err
}

func applyRecommendation(core *cliapp.ScenarioApp) error {
	if err := rejectPendingOperatorInputs(core); err != nil {
		return err
	}
	body, err := core.Get("/v2/recommendation", nil)
	if err != nil {
		return err
	}
	var recommendation struct {
		Profile   string   `json:"profile"`
		Scenarios []string `json:"scenarios"`
		Resources []string `json:"resources"`
	}
	if err := json.Unmarshal(body, &recommendation); err != nil {
		return fmt.Errorf("decode recommendation: %w", err)
	}
	scenarios := map[string]any{}
	for _, name := range recommendation.Scenarios {
		scenarios[name] = map[string]any{"enabled": true}
	}
	resources := map[string]any{}
	for _, name := range recommendation.Resources {
		resources[name] = map[string]any{"enabled": true}
	}
	patch := map[string]any{
		"active_profile": recommendation.Profile,
		"scenarios":      scenarios,
		"resources":      resources,
	}
	patchBody, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	if _, err := core.Request("PATCH", "/v2/operator-state", nil, patchBody); err != nil {
		return err
	}
	planBody, err := core.Get("/v2/apply/plan", nil)
	if err != nil {
		return err
	}
	if err := printApplyPlan(planBody); err != nil {
		return err
	}
	applyResult, err := core.Request("POST", "/v2/apply", nil, []byte("{}"))
	if err != nil {
		return err
	}
	var applyResponse struct {
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(applyResult, &applyResponse); err == nil && applyResponse.RunID != "" {
		applyResult, err = waitForApply(core, applyResponse.RunID, applyResponse.Status)
		if err != nil {
			return err
		}
	}
	readiness, err := core.Get("/v2/readiness", nil)
	if err != nil {
		return err
	}
	if err := printApplyReport(applyResult); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Readiness:", string(readiness))
	return err
}

func readOperatorInputQueue(core *cliapp.ScenarioApp) (operatorInputQueue, error) {
	body, err := core.Get("/v2/operator-inputs", nil)
	if err != nil {
		return operatorInputQueue{}, err
	}
	var queue operatorInputQueue
	if err := json.Unmarshal(body, &queue); err != nil {
		return operatorInputQueue{}, fmt.Errorf("decode operator input queue: %w", err)
	}
	return queue, nil
}

func rejectPendingOperatorInputs(core *cliapp.ScenarioApp) error {
	queue, err := readOperatorInputQueue(core)
	if err != nil {
		return err
	}
	if len(queue.Requests) == 0 {
		return nil
	}
	ids := make([]string, 0, len(queue.Requests))
	for _, request := range queue.Requests {
		ids = append(ids, request.ID)
	}
	encodedIDs, _ := json.Marshal(ids)
	_, _ = fmt.Fprintf(os.Stdout, `{"status":"needs_input","requests":%s}
`, encodedIDs)
	return fmt.Errorf("wizard has pending operator input; resolve it through onboarding before applying the recommendation")
}

func resolvePendingOperatorInputs(core *cliapp.ScenarioApp, read, readSecret func(int, string) (string, error)) error {
	queue, err := readOperatorInputQueue(core)
	if err != nil {
		return err
	}
	if len(queue.Requests) == 0 {
		return nil
	}
	answers := make([]map[string]string, 0, len(queue.Requests))
	for _, request := range queue.Requests {
		prompt := request.Title
		if request.Description != "" {
			prompt += " — " + request.Description
		}
		if len(request.Options) > 0 {
			prompt += " [" + strings.Join(request.Options, ", ") + "]"
		}
		if request.Default != "" {
			prompt += " (default: " + request.Default + ")"
		}
		readAnswer := read
		if request.Kind == "secret" {
			readAnswer = readSecret
		}
		value, err := readAnswer(4, prompt)
		if err != nil {
			return err
		}
		if value == "" {
			value = request.Default
		}
		answers = append(answers, map[string]string{"request_id": request.ID, "value": value})
	}
	body, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	if _, err := core.Request("POST", "/v2/operator-inputs/resolve", nil, body); err != nil {
		return fmt.Errorf("resolve onboarding operator input: %w", err)
	}
	return nil
}

func describeHostItems(items []hostItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		detail := item.Name
		if item.Risk != "" || item.Privilege != "" {
			risk := item.Risk
			if risk == "" {
				risk = "not declared"
			}
			privilege := item.Privilege
			if privilege == "" {
				privilege = "not declared"
			}
			detail = fmt.Sprintf("%s (risk=%s, privilege=%s)", detail, risk, privilege)
		}
		parts = append(parts, detail)
	}
	return strings.Join(parts, ", ")
}

func printApplyPlan(body []byte) error {
	var plan struct {
		Items []struct {
			Kind     string `json:"kind"`
			Name     string `json:"name"`
			Required bool   `json:"required"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &plan); err != nil {
		return fmt.Errorf("decode apply plan: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Apply plan (the exact list to be executed):")
	if len(plan.Items) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "  (no changes)")
		return nil
	}
	for _, item := range plan.Items {
		required := "optional"
		if item.Required {
			required = "required"
		}
		_, _ = fmt.Fprintf(os.Stdout, "  - %s: %s (%s)\n", item.Kind, item.Name, required)
	}
	return nil
}

func printApplyReport(body []byte) error {
	var report struct {
		Status string `json:"status"`
		Items  []struct {
			Name      string `json:"name"`
			Outcome   string `json:"outcome"`
			Error     string `json:"error"`
			BlockedBy string `json:"blocked_by"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return fmt.Errorf("decode apply report: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Apply report:", report.Status)
	for _, item := range report.Items {
		detail := item.Error
		if detail == "" {
			detail = item.BlockedBy
		}
		if detail == "" {
			detail = "completed"
		}
		_, _ = fmt.Fprintf(os.Stdout, "  - %s: %s (%s)\n", item.Name, item.Outcome, detail)
	}
	return nil
}

func containsHost(items []hostItem, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func containsSafeguard(items []hostItem, name string) bool {
	return containsHost(items, name)
}

func waitForApply(core *cliapp.ScenarioApp, runID, status string) ([]byte, error) {
	result := []byte(`{"run_id":"` + runID + `","status":"` + status + `"}`)
	for status == "pending" || status == "applying" {
		time.Sleep(100 * time.Millisecond)
		var err error
		result, err = core.Get("/v2/apply/"+runID, nil)
		if err != nil {
			return nil, err
		}
		var current struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(result, &current); err != nil {
			return nil, err
		}
		status = current.Status
	}
	return result, nil
}

func get(core *cliapp.ScenarioApp, args []string, path string) error {
	fs := support.NewFlagSet("wizard")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(body, '\n'))
		return err
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("decode wizard response: %w", err)
	}
	pretty, _ := json.MarshalIndent(value, "", "  ")
	_, err = fmt.Fprintln(os.Stdout, string(pretty))
	return err
}

func apply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard apply")
	selectionPath := fs.String("selection", "", "Path to a selection document")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*selectionPath) == "" {
		return fmt.Errorf("--selection is required")
	}
	body, err := support.ReadJSONFile(*selectionPath, true)
	if err != nil {
		return err
	}
	var selection Selection
	if err := json.Unmarshal(body, &selection); err != nil {
		return fmt.Errorf("decode selection: %w", err)
	}
	patch := selectionPatch(selection)
	patchBody, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	if _, err := core.Request("PATCH", "/v2/operator-state", nil, patchBody); err != nil {
		return err
	}
	result, err := core.Request("POST", "/v2/apply", nil, []byte("{}"))
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(result, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Selection committed and applied"}, NextCommand: []string{support.CLIName + " wizard status"}})
}

func exportSelection(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard export")
	outputPath := fs.String("output", "", "Output selection document path")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*outputPath) == "" {
		return fmt.Errorf("--output is required")
	}
	body, err := core.Get("/v2/scenarios", nil)
	if err != nil {
		return err
	}
	var response struct {
		Scenarios []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode scenarios: %w", err)
	}
	selection := Selection{Scenarios: []string{}, ScenarioState: map[string]bool{}}
	for _, scenario := range response.Scenarios {
		selection.ScenarioState[scenario.Name] = scenario.Enabled
		if scenario.Enabled {
			selection.Scenarios = append(selection.Scenarios, scenario.Name)
		}
	}
	stateBody, err := core.Get("/operator-state", nil)
	if err == nil {
		var state struct {
			Resources map[string]struct {
				Enabled *bool `json:"enabled"`
			} `json:"resources"`
			HostTools map[string]struct {
				OptedIn *bool `json:"opted_in"`
			} `json:"host_tools"`
			HostSafeguards map[string]struct {
				OptedIn *bool `json:"opted_in"`
			} `json:"host_safeguards"`
		}
		if json.Unmarshal(stateBody, &state) == nil {
			selection.Resources = map[string]bool{}
			for name, choice := range state.Resources {
				if choice.Enabled != nil {
					selection.Resources[name] = *choice.Enabled
				}
			}
			selection.HostTools = map[string]bool{}
			for name, choice := range state.HostTools {
				if choice.OptedIn != nil {
					selection.HostTools[name] = *choice.OptedIn
				}
			}
			selection.HostSafeguards = map[string]bool{}
			for name, choice := range state.HostSafeguards {
				if choice.OptedIn != nil {
					selection.HostSafeguards[name] = *choice.OptedIn
				}
			}
		}
	}
	data, _ := json.MarshalIndent(selection, "", "  ")
	if err := os.WriteFile(*outputPath, append(data, '\n'), 0o600); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Selection exported to", *outputPath)
	return err
}
