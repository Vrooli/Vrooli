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

type stepModelEntry struct {
	ID      string `json:"id"`
	Ordinal int    `json:"ordinal"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "wizard", Description: "Configure an installation through the onboarding API", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show the computed onboarding step and committed state", Run: func(args []string) error { return support.GetJSON(core, "wizard", args, "/v2/session") }},
		{Name: "commit", Description: "Commit a selection document and apply it; top-level apply applies committed state", Run: func(args []string) error { return apply(core, args) }},
		{Name: "export", Description: "Export the current manifest-derived selection", Run: func(args []string) error { return exportSelection(core, args) }},
		{Name: "run", Description: "Walk the same nine capability steps used by the UI", Run: func(args []string) error { return runWizard(core, args) }},
	}}
}

func runWizard(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard run")
	interactive := fs.Bool("interactive", false, "Walk all nine onboarding steps in the terminal")
	acceptRecommendation := fs.Bool("accept-recommendation", false, "Use the manifest-derived starter profile without asking for scenario names")
	nonInteractive := fs.Bool("non-interactive", false, "Never read input; return a typed needs-input error when a decision is required")
	fromStep := fs.String("from-step", "", "Start at a declared step id instead of the session pointer")
	restart := fs.Bool("restart", false, "Restart from the first declared step instead of resuming")
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
		return support.GetJSON(core, "wizard", args, "/v2/scenarios")
	}
	reader := bufio.NewReader(os.Stdin)
	stepsBody, err := core.Get("/v2/steps", nil)
	if err != nil {
		return err
	}
	var stepResponse struct {
		Steps []stepModelEntry `json:"steps"`
	}
	if err := json.Unmarshal(stepsBody, &stepResponse); err != nil {
		return fmt.Errorf("decode step model: %w", err)
	}
	sort.Slice(stepResponse.Steps, func(i, j int) bool { return stepResponse.Steps[i].Ordinal < stepResponse.Steps[j].Ordinal })
	for _, step := range stepResponse.Steps {
		if _, ok := stepHandlers[step.ID]; !ok {
			return unimplementedStepError{ID: step.ID}
		}
	}
	stepIndex := func(id string) (int, error) {
		for _, step := range stepResponse.Steps {
			if step.ID == id {
				return step.Ordinal, nil
			}
		}
		return 0, fmt.Errorf("unknown onboarding step %q", id)
	}
	startIndex := 0
	if !*restart {
		sessionBody, sessionErr := core.Get("/v2/session", nil)
		if sessionErr != nil {
			return sessionErr
		}
		var session struct {
			FirstUnsatisfiedStep int  `json:"first_unsatisfied_step"`
			Completion           bool `json:"completion"`
		}
		if err := json.Unmarshal(sessionBody, &session); err != nil {
			return fmt.Errorf("decode onboarding session: %w", err)
		}
		startIndex = session.FirstUnsatisfiedStep
		if session.Completion {
			_, _ = fmt.Fprintln(os.Stdout, "Onboarding configuration is already applied; use --restart to walk it again.")
			return nil
		}
	}
	if strings.TrimSpace(*fromStep) != "" {
		var stepErr error
		startIndex, stepErr = stepIndex(strings.TrimSpace(*fromStep))
		if stepErr != nil {
			return stepErr
		}
	}
	if startIndex < 0 || startIndex >= len(stepResponse.Steps) {
		startIndex = 0
	}
	if startIndex > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Resuming onboarding at %s; %d step(s) already satisfied.\n", stepResponse.Steps[startIndex].ID, startIndex)
	}
	shouldRun := func(id string) bool { index, err := stepIndex(id); return err == nil && index >= startIndex }
	stepOrdinal := func(id string) (int, error) {
		for _, step := range stepResponse.Steps {
			if step.ID == id {
				return step.Ordinal + 1, nil
			}
		}
		return 0, fmt.Errorf("step model is missing %q", id)
	}
	read := func(stepID, prompt string) (string, error) {
		step, err := stepOrdinal(stepID)
		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(os.Stdout, "Step %d — %s\n> ", step, prompt); err != nil {
			return "", err
		}
		line, err := reader.ReadString('\n')
		return strings.TrimSpace(line), err
	}
	readSecret := func(stepID, prompt string) (string, error) {
		step, err := stepOrdinal(stepID)
		if err != nil {
			return "", err
		}
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
	known := map[string]bool{}
	names := make([]string, 0, len(scenarioResponse.Scenarios))
	for _, scenario := range scenarioResponse.Scenarios {
		known[scenario.Name] = true
		names = append(names, scenario.Name)
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
	knownResources := map[string]bool{}
	for _, name := range optionalNames {
		knownResources[name] = true
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
	var applyResult []byte
	runStep := func(id string) error {
		switch id {
		case "welcome":
			_, err := read("welcome", "welcome; press enter to begin the onboarding steps")
			return err
		case "scenarios":
			_, _ = fmt.Fprintln(os.Stdout, "Available scenarios:", strings.Join(names, ", "))
			selectedLine, err := read("scenarios", "select scenario names (comma separated; press enter to accept the starter profile)")
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
			return nil
		case "resources":
			_, _ = fmt.Fprintln(os.Stdout, "Optional/standalone resources:", strings.Join(optionalNames, ", "))
			resourceLine, err := read("resources", "select optional or standalone resources (comma separated; press enter to keep the recommendation)")
			if err != nil {
				return err
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
			return nil
		case "credentials":
			if err := resolvePendingOperatorInputs(core, func(_ int, prompt string) (string, error) { return read("credentials", prompt) }, func(_ int, prompt string) (string, error) { return readSecret("credentials", prompt) }); err != nil {
				return err
			}
			if _, err := read("credentials", "credentials are listed by the API; provision values with credentials provision, then press enter"); err != nil {
				return err
			}
			for _, credential := range credentialResponse.Credentials {
				if credential.Status == "configured" {
					continue
				}
				label := credential.Label
				if label == "" {
					label = credential.LogicalID + "/" + credential.Field
				}
				value, readErr := readSecret("credentials", fmt.Sprintf("enter %s; leave blank to defer (value is never printed)", label))
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
			return nil
		case "integrations":
			_, err := read("integrations", "integration binding is deferred; press enter to continue")
			return err
		case "host":
			_, _ = fmt.Fprintln(os.Stdout, "Host tools:", describeHostItems(hostResponse.Tools))
			toolLine, err := read("host", "select optional host tools (comma separated; required tools are automatic)")
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
			safeguardLine, err := read("host", "select optional host safeguards (comma separated; press enter for none)")
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
			return nil
		case "operating-mode":
			modeLine, err := read("operating-mode", "choose operating mode: enter scenario names for auto-restart (comma separated)")
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
			return nil
		case "apply":
			// The plan is fetched and shown BEFORE the confirmation prompt.
			// It used to be printed immediately after, which meant the
			// operator answered "apply this selection now?" with nothing
			// disclosed — a consent prompt whose disclosure arrived too late
			// to inform the answer. Persisting the selection first is what
			// makes the plan computable, and it authorizes nothing on its own:
			// selectionPatch carries no apply flag, and the host is only
			// touched by POST /v2/apply below.
			patch, marshalErr := json.Marshal(selectionPatch(selection))
			if marshalErr != nil {
				return marshalErr
			}
			if _, err := core.Request("PATCH", "/v2/operator-state", nil, patch); err != nil {
				return err
			}
			planBody, err := core.Get("/v2/apply/plan", nil)
			if err != nil {
				return err
			}
			if err := printApplyPlan(planBody); err != nil {
				return err
			}
			confirmation, err := read("apply", "apply this selection now? answer yes or no; press enter for yes")
			if err != nil {
				return err
			}
			if strings.TrimSpace(confirmation) != "" && strings.ToLower(strings.TrimSpace(confirmation)) != "yes" {
				return fmt.Errorf("selection not applied; answer yes to commit the wizard selection")
			}
			selection.Apply = true
			applyResult, err = core.Request("POST", "/v2/apply", nil, []byte("{}"))
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
			return nil
		case "validation":
			if _, err := read("validation", "validation will run after apply; press enter to print final status"); err != nil {
				return err
			}
			readinessResult, err := core.Get("/v2/readiness", nil)
			if err != nil {
				return err
			}
			if applyResult != nil {
				if err := printApplyReport(applyResult); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(os.Stdout, "Readiness:", string(readinessResult)); err != nil {
				return err
			}
			return reportReadinessBlockers(readinessResult)
		default:
			return unimplementedStepError{ID: id}
		}
	}
	session := &wizardSession{runStep: runStep}
	for _, step := range stepResponse.Steps {
		if !shouldRun(step.ID) {
			continue
		}
		handler, ok := stepHandlers[step.ID]
		if !ok {
			return unimplementedStepError{ID: step.ID}
		}
		if err := handler(session); err != nil {
			return err
		}
		body, marshalErr := json.Marshal(map[string]int{"step": step.Ordinal})
		if marshalErr != nil {
			return marshalErr
		}
		if _, err := core.Request("POST", "/v2/session/step", nil, body); err != nil {
			return fmt.Errorf("record completed step %s: %w", step.ID, err)
		}
	}
	return nil
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

// printApplyPlan is the operator's disclosure before consent. It must answer
// four questions the flat kind/name list did not: how much is about to happen,
// which items change host state with elevation, which items are already in
// place, and what "apply" actually does to this machine. The plan is a
// desired-state list, so without the state split every entry reads as a
// pending change even when most are already satisfied.
func printApplyPlan(body []byte) error {
	var plan struct {
		Items []planItem `json:"items"`
	}
	if err := json.Unmarshal(body, &plan); err != nil {
		return fmt.Errorf("decode apply plan: %w", err)
	}
	if len(plan.Items) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Apply plan: no changes. Nothing will be executed.")
		return nil
	}

	pending := filterByState(plan.Items, "pending")
	satisfied := filterByState(plan.Items, "satisfied")
	unknown := filterByState(plan.Items, "unknown")
	elevatedPending := 0
	for _, item := range pending {
		if item.Privileged {
			elevatedPending++
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nApply plan — %d selected item(s): %d not yet in place, %d already in place, %d not checkable before applying.\n",
		len(plan.Items), len(pending), len(satisfied), len(unknown))

	_, _ = fmt.Fprintln(os.Stdout, "\nWhat \"apply\" does, per kind:")
	_, _ = fmt.Fprintln(os.Stdout, "  tool       `vrooli host install <name>`   — installs a program on this host")
	_, _ = fmt.Fprintln(os.Stdout, "  safeguard  `vrooli host safeguard <name>` — changes host configuration (sysctl, systemd, sudoers and similar)")
	_, _ = fmt.Fprintln(os.Stdout, "  resource   `vrooli resource enable <name>` — starts a local service")
	_, _ = fmt.Fprintln(os.Stdout, "  scenario   `vrooli scenario start <name>`  — starts an app's processes")
	_, _ = fmt.Fprintln(os.Stdout, "Every item runs even when already in place; those runs converge rather than reinstall.")
	_, _ = fmt.Fprintln(os.Stdout, "Nothing is removed, disabled, or uninstalled by apply. Deselected items are skipped, not reverted.")

	if len(pending) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nNOT YET IN PLACE — these change this host (%d, %d elevated)\n", len(pending), elevatedPending)
		printItemsGrouped(pending, true)
	}
	if len(satisfied) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nALREADY IN PLACE — verified present on this host (%d)\n", len(satisfied))
		printItemsGrouped(satisfied, false)
	}
	if len(unknown) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nNOT CHECKED — state is reported by the handler during apply (%d)\n", len(unknown))
		printItemsGrouped(unknown, false)
	}
	_, _ = fmt.Fprintln(os.Stdout, "\nNothing has been applied yet. Answering no leaves the host unchanged.")
	return nil
}

type planItem struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Required   bool   `json:"required"`
	Privileged bool   `json:"privileged"`
	State      string `json:"state"`
}

func filterByState(items []planItem, state string) []planItem {
	matched := make([]planItem, 0, len(items))
	for _, item := range items {
		itemState := item.State
		if itemState == "" {
			itemState = "unknown"
		}
		if itemState == state {
			matched = append(matched, item)
		}
	}
	return matched
}

// printItemsGrouped lists items by kind. Detailed lines are reserved for the
// group that changes the host; the rest are summarized on one line per kind so
// a long already-satisfied list cannot bury the part that matters.
func printItemsGrouped(items []planItem, detailed bool) {
	byKind := map[string][]planItem{}
	kindOrder := make([]string, 0, 4)
	for _, item := range items {
		if _, seen := byKind[item.Kind]; !seen {
			kindOrder = append(kindOrder, item.Kind)
		}
		byKind[item.Kind] = append(byKind[item.Kind], item)
	}
	for _, kind := range kindOrder {
		entries := byKind[kind]
		_, _ = fmt.Fprintf(os.Stdout, "  %s (%d)\n", pluralKind(kind, len(entries)), len(entries))
		if !detailed {
			names := make([]string, 0, len(entries))
			for _, item := range entries {
				names = append(names, item.Name)
			}
			_, _ = fmt.Fprintf(os.Stdout, "    %s\n", strings.Join(names, ", "))
			continue
		}
		for _, item := range entries {
			markers := []string{"optional"}
			if item.Required {
				markers = []string{"required"}
			}
			marker := "-"
			if item.Privileged {
				markers = append(markers, "elevated")
				marker = "!"
			}
			_, _ = fmt.Fprintf(os.Stdout, "    %s %s (%s)\n", marker, item.Name, strings.Join(markers, ", "))
		}
	}
}

// pluralKind renders an apply-plan kind as a readable section heading.
func pluralKind(kind string, count int) string {
	label := map[string]string{
		"tool":      "Host tool",
		"safeguard": "Host safeguard",
		"resource":  "Resource",
		"scenario":  "Scenario",
	}[kind]
	if label == "" {
		label = strings.ToUpper(kind[:1]) + kind[1:]
	}
	if count == 1 {
		return label
	}
	if strings.HasSuffix(label, "s") {
		return label
	}
	return label + "s"
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

func apply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard commit")
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

// completionBlocker mirrors the API's metadata-only blocker. It never carries a
// credential value.
type completionBlocker struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation"`
}

// reportReadinessBlockers prints the named reasons configuration is not
// complete and fails the command when any remains.
//
// A wizard that prints a blocking verdict and exits zero tells a script that
// the host is configured. The exit code is the only part of that report a
// caller can act on without parsing prose, so it has to carry the verdict.
func reportReadinessBlockers(readiness []byte) error {
	var response struct {
		Status               string              `json:"status"`
		Blockers             []completionBlocker `json:"blockers"`
		Degraded             []completionBlocker `json:"degraded"`
		DegradedDigest       string              `json:"degraded_digest"`
		DegradedAcknowledged bool                `json:"degraded_acknowledged"`
	}
	if err := json.Unmarshal(readiness, &response); err != nil {
		return fmt.Errorf("decode readiness: %w", err)
	}
	for _, blocker := range response.Blockers {
		_, _ = fmt.Fprintf(os.Stdout, "Blocked: %s %s — %s. Next: %s\n", blocker.Kind, blocker.Name, blocker.Reason, blocker.Remediation)
	}
	for _, gap := range response.Degraded {
		_, _ = fmt.Fprintf(os.Stdout, "Degraded: %s %s — %s. Next: %s\n", gap.Kind, gap.Name, gap.Reason, gap.Remediation)
	}
	if len(response.Blockers) > 0 {
		return fmt.Errorf("configuration is not complete: %d blocking item(s) remain", len(response.Blockers))
	}
	if len(response.Degraded) > 0 && !response.DegradedAcknowledged {
		_, _ = fmt.Fprintf(os.Stdout, "Accept the degraded set with: vrooli-onboarding readiness acknowledge-degraded --digest %s\n", response.DegradedDigest)
		return fmt.Errorf("configuration is not complete: %d optional item(s) need an explicit acknowledgement", len(response.Degraded))
	}
	return nil
}
