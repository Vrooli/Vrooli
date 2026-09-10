package wizard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	onboardingselection "vrooli-onboarding/cli/domains/selection"
	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply"
	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials"
	credentialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials/credentialsv1connect"
	operatorinputsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	"golang.org/x/term"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"vrooli-onboarding/cli/internal/clock"
)

// Selection is the stable automation document. It names operator intent and
// deliberately does not mirror the internal operator-state schema.
type Selection struct {
	ActiveProfile     string          `json:"active_profile,omitempty"`
	Scenarios         []string        `json:"scenarios"`
	CoreSeed          []string        `json:"core_seed,omitempty"`
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
	if selection.CoreSeed != nil {
		patch["core"] = map[string]any{"seed": normalizedNames(selection.CoreSeed)}
	}
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
	Declinable  bool     `json:"declinable"`
	Decision    string   `json:"decision"`
}

type operatorInputQueue struct {
	Version          int32                  `json:"version"`
	ExpectedRevision string                 `json:"expected_revision,omitempty"`
	Requests         []operatorInputRequest `json:"requests"`
}

const (
	applyStartProcedure            = "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply"
	applyReviewProcedure           = "/vrooli.vrooli_onboarding.v1.apply.ApplyService/ReviewApply"
	applyRunProcedure              = "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyRun"
	applyPlanProcedure             = "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyPlan"
	sessionGetProcedure            = "/vrooli.vrooli_onboarding.v1.session.SessionService/GetSession"
	sessionAdvanceProcedure        = "/vrooli.vrooli_onboarding.v1.session.SessionService/AdvanceSessionStep"
	sessionModelProcedure          = "/vrooli.vrooli_onboarding.v1.session.SessionService/GetStepModel"
	operatorInputsListProcedure    = "/vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService/ListOperatorInputs"
	operatorInputsResolveProcedure = "/vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService/ResolveOperatorInputs"
	operatorStateGetProcedure      = "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/GetOperatorState"
	operatorStatePatchProcedure    = "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState"
	readinessGetProcedure          = "/vrooli.vrooli_onboarding.v1.readiness.ReadinessService/GetReadiness"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "wizard", Description: "Configure an installation through the onboarding API", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show the computed onboarding step and committed state", Run: func(args []string) error { return sessionStatus(core, args) }},
		{Name: "commit", Description: "Commit a selection document and apply it; top-level apply applies committed state", Run: func(args []string) error { return apply(core, args) }},
		{Name: "export", Description: "Export the current manifest-derived selection", Run: func(args []string) error { return exportSelection(core, args) }},
		{Name: "support-export", Description: "Write an explicit, metadata-only support diagnostic", Run: func(args []string) error { return supportExport(core, args) }},
		{Name: "run", Description: "Walk the same ten capability steps used by the UI", Run: func(args []string) error { return runWizard(core, args) }},
		{Name: "core-set", Description: "Preview and update the operator core seed", Run: func(args []string) error { return runCoreSet(core, args) }},
		{Name: "advance-step", Description: "Advance the shared onboarding step pointer", Run: func(args []string) error { return advanceSessionStep(core, args) }},
	}}
}

// ManifestHandlers retains the workflow commands that need terminal and file
// behavior while the surrounding command tree is loaded from the manifest.
func ManifestHandlers(core *cliapp.ScenarioApp) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"wizard.export": cliapp.ExternalDelegation(func(ctx cliapp.RunContext) error {
			return exportSelection(core, contextArgs(ctx))
		}),
		"wizard.support-export": cliapp.ExternalDelegation(func(ctx cliapp.RunContext) error {
			return supportExport(core, contextArgs(ctx))
		}),
		"wizard.run": cliapp.ExternalDelegation(func(ctx cliapp.RunContext) error {
			return runWizard(core, contextArgs(ctx))
		}),
	}
}

func contextArgs(ctx cliapp.RunContext) []string {
	args := append([]string(nil), ctx.Args()...)
	for _, flag := range ctx.Schema().Flags {
		if flag.Bool {
			if ctx.BoolFlag(flag.Name) {
				args = append(args, "--"+flag.Name)
			}
			continue
		}
		if ctx.FlagProvided(flag.Name) {
			args = append(args, "--"+flag.Name, ctx.Flag(flag.Name))
		}
	}
	return args
}

func runWizard(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard run")
	interactive := fs.Bool("interactive", false, "Walk all nine onboarding steps in the terminal")
	acceptRecommendation := fs.Bool("accept-recommendation", false, "Use the manifest-derived starter profile without asking for scenario names")
	nonInteractive := fs.Bool("non-interactive", false, "Never read input; return a typed needs-input error when a decision is required")
	target := fs.String("target", "local", "Onboarding target scenario")
	fromStep := fs.String("from-step", "", "Start at a declared step id instead of the session pointer")
	restart := fs.Bool("restart", false, "Restart from the first declared step instead of resuming")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*target) == "" {
		return fmt.Errorf("--target is required")
	}
	if *nonInteractive && !*acceptRecommendation {
		_, _ = fmt.Fprintln(os.Stdout, `{"status":"needs_input","reason":"use --accept-recommendation or run with --interactive"}`)
		return fmt.Errorf("wizard needs operator input; rerun with --accept-recommendation or --interactive")
	}
	if *acceptRecommendation {
		return applyRecommendation(core, strings.TrimSpace(*target))
	}
	if !*interactive {
		return onboardingselection.List(core, args)
	}
	reader := bufio.NewReader(os.Stdin)
	stepResponse := &sessionv1.GetStepModelResponse{}
	if err := requestSession(core, sessionModelProcedure, &sessionv1.GetStepModelRequest{Target: strings.TrimSpace(*target)}, stepResponse); err != nil {
		return err
	}
	sort.Slice(stepResponse.Steps, func(i, j int) bool { return stepResponse.Steps[i].GetOrdinal() < stepResponse.Steps[j].GetOrdinal() })
	for _, step := range stepResponse.Steps {
		if _, ok := stepHandlers[step.GetId()]; !ok {
			return unimplementedStepError{ID: step.GetId()}
		}
	}
	stepIndex := func(id string) (int, error) {
		for _, step := range stepResponse.Steps {
			if step.GetId() == id {
				return int(step.GetOrdinal()), nil
			}
		}
		return 0, fmt.Errorf("unknown onboarding step %q", id)
	}
	startIndex := 0
	if !*restart {
		sessionResponse := &sessionv1.GetSessionResponse{}
		if sessionErr := requestSession(core, sessionGetProcedure, &sessionv1.GetSessionRequest{Target: strings.TrimSpace(*target)}, sessionResponse); sessionErr != nil {
			return sessionErr
		}
		startIndex = int(sessionResponse.GetFirstUnsatisfiedStep())
		if sessionResponse.GetCompletion() {
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
		_, _ = fmt.Fprintf(os.Stdout, "Resuming onboarding at %s; %d step(s) already satisfied.\n", stepResponse.Steps[startIndex].GetId(), startIndex)
	}
	shouldRun := func(id string) bool { index, err := stepIndex(id); return err == nil && index >= startIndex }
	stepOrdinal := func(id string) (int, error) {
		for _, step := range stepResponse.Steps {
			if step.GetId() == id {
				return int(step.GetOrdinal()) + 1, nil
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
	scenarioResponse := &selectionv1.ListScenariosResponse{}
	if err := onboardingselection.Request(core, onboardingselection.ListScenariosProcedure, &selectionv1.ListScenariosRequest{Target: strings.TrimSpace(*target)}, scenarioResponse); err != nil {
		return err
	}
	selection := Selection{ScenarioState: map[string]bool{}, ActiveProfile: "starter"}
	currentCore, err := onboardingselection.CoreSetForTarget(core, nil, strings.TrimSpace(*target))
	if err != nil {
		return err
	}
	selection.CoreSeed = append([]string(nil), currentCore.Seed...)
	recommendation := &selectionv1.GetRecommendationResponse{}
	if err := onboardingselection.Request(core, onboardingselection.GetRecommendationProcedure, &selectionv1.GetRecommendationRequest{Target: strings.TrimSpace(*target)}, recommendation); err != nil {
		return err
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
	union := &selectionv1.GetUnionResponse{}
	if err := onboardingselection.Request(core, onboardingselection.GetUnionProcedure, &selectionv1.GetUnionRequest{Target: strings.TrimSpace(*target)}, union); err != nil {
		return err
	}
	optionalNames := make([]string, 0, len(union.OptionalResources)+len(union.StandaloneResources))
	for _, resource := range union.OptionalResources {
		optionalNames = append(optionalNames, resource.Name)
	}
	for _, resource := range union.StandaloneResources {
		optionalNames = append(optionalNames, resource.Name)
	}
	sort.Strings(optionalNames)
	knownResources := map[string]bool{}
	for _, name := range optionalNames {
		knownResources[name] = true
	}
	credentialResponse := &credentialsv1.ListCredentialsResponse{}
	if err := requestSession(core, credentialsconnect.CredentialsServiceListCredentialsProcedure, &credentialsv1.ListCredentialsRequest{Target: strings.TrimSpace(*target)}, credentialResponse); err != nil {
		return fmt.Errorf("list credentials: %w", err)
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
	var applyResult *applyv1.GetApplyRunResponse
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
		case "core-set":
			_, _ = fmt.Fprintln(os.Stdout, "Current core seed:", strings.Join(selection.CoreSeed, ", "))
			if currentCore.Available {
				_, _ = fmt.Fprintf(os.Stdout, "Computed closure: %d scenario(s), %d resource(s)\n", currentCore.MemberCounts["scenario"], currentCore.MemberCounts["resource"])
			} else {
				_, _ = fmt.Fprintln(os.Stdout, "Closure unavailable; the seed remains authoritative:", currentCore.Error)
			}
			seedLine, err := read("core-set", "enter the complete core seed (comma separated; press enter to keep it)")
			if err != nil {
				return err
			}
			if strings.TrimSpace(seedLine) != "" {
				selection.CoreSeed = normalizedNames(strings.Split(seedLine, ","))
			}
			preview, err := fetchCoreSet(core, selection.CoreSeed)
			if err != nil {
				return err
			}
			currentCore = preview
			if preview.Available {
				_, _ = fmt.Fprintf(os.Stdout, "Updated closure: %d scenario(s), %d resource(s)\n", preview.MemberCounts["scenario"], preview.MemberCounts["resource"])
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
			if err := resolvePendingOperatorInputs(core, strings.TrimSpace(*target), func(_ int, prompt string) (string, error) { return read("credentials", prompt) }, func(_ int, prompt string) (string, error) { return readSecret("credentials", prompt) }); err != nil {
				return err
			}
			if _, err := read("credentials", "credentials are listed by the API; provision values with credentials provision, then press enter"); err != nil {
				return err
			}
			for _, credential := range credentialResponse.GetCredentials() {
				if credential.GetStatus() == "configured" {
					continue
				}
				label := credential.GetLabel()
				if label == "" {
					label = credential.GetLogicalId() + "/" + credential.GetField()
				}
				value, readErr := readSecret("credentials", fmt.Sprintf("enter %s; leave blank to defer (value is never printed)", label))
				if readErr != nil {
					return readErr
				}
				if value == "" {
					if credential.GetRequired() {
						_, _ = fmt.Fprintln(os.Stdout, "Required credential deferred; readiness will remain blocked.")
					}
					continue
				}
				provisioned := &credentialsv1.ProvisionCredentialResponse{}
				if requestErr := requestSession(core, credentialsconnect.CredentialsServiceProvisionCredentialProcedure, &credentialsv1.ProvisionCredentialRequest{Target: strings.TrimSpace(*target), LogicalId: credential.GetLogicalId(), Field: credential.GetField(), Value: value}, provisioned); requestErr != nil {
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
			// touched by StartApply below.
			patch, marshalErr := json.Marshal(selectionPatch(selection))
			if marshalErr != nil {
				return marshalErr
			}
			if err := patchOperatorState(core, strings.TrimSpace(*target), patch); err != nil {
				return err
			}
			planResponse := &applyv1.GetApplyPlanResponse{}
			if err := requestApply(core, applyPlanProcedure, &applyv1.GetApplyPlanRequest{Target: strings.TrimSpace(*target)}, planResponse); err != nil {
				return err
			}
			if err := renderApplyPlan(planResponse); err != nil {
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
			applyResponse, err := startApplyWithConsent(core, strings.TrimSpace(*target), planResponse)
			if err != nil {
				return err
			}
			if applyResponse.GetRun() != nil {
				applyResult, err = waitForApply(core, strings.TrimSpace(*target), applyResponse.GetRun())
				if err != nil {
					return err
				}
			}
			return nil
		case "validation":
			if _, err := read("validation", "validation will run after apply; press enter to print final status"); err != nil {
				return err
			}
			readinessResponse := &readinessv1.GetReadinessResponse{}
			if err := requestOperator(core, readinessGetProcedure, &readinessv1.GetReadinessRequest{Target: strings.TrimSpace(*target)}, readinessResponse); err != nil {
				return err
			}
			readinessResult, err := protojson.Marshal(readinessResponse)
			if err != nil {
				return fmt.Errorf("encode readiness: %w", err)
			}
			if applyResult != nil {
				if err := renderApplyReport(applyResult); err != nil {
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
		if !shouldRun(step.GetId()) {
			continue
		}
		handler, ok := stepHandlers[step.GetId()]
		if !ok {
			return unimplementedStepError{ID: step.GetId()}
		}
		if err := handler(session); err != nil {
			return err
		}
		if err := requestSession(core, sessionAdvanceProcedure, &sessionv1.AdvanceSessionStepRequest{Target: strings.TrimSpace(*target), StepId: step.GetId()}, &sessionv1.GetSessionResponse{}); err != nil {
			return fmt.Errorf("record completed step %s: %w", step.GetId(), err)
		}
	}
	return nil
}

type coreSetView = selectionv1.GetCoreSetResponse

func runCoreSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard core-set")
	target := fs.String("target", "local", "Onboarding target scenario")
	add := fs.String("add", "", "Comma-separated scenarios to add to core.seed")
	remove := fs.String("remove", "", "Comma-separated scenarios to remove from core.seed")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*target) == "" {
		return fmt.Errorf("--target is required")
	}
	current, err := fetchCoreSetForTarget(core, nil, strings.TrimSpace(*target))
	if err != nil {
		return err
	}
	seed := map[string]bool{}
	for _, name := range current.Seed {
		seed[name] = true
	}
	for _, name := range normalizedNames(strings.Split(*add, ",")) {
		seed[name] = true
	}
	for _, name := range normalizedNames(strings.Split(*remove, ",")) {
		delete(seed, name)
	}
	proposed := make([]string, 0, len(seed))
	for name := range seed {
		proposed = append(proposed, name)
	}
	sort.Strings(proposed)
	preview, err := fetchCoreSetForTarget(core, proposed, strings.TrimSpace(*target))
	if err != nil {
		return err
	}
	if strings.TrimSpace(*add) != "" || strings.TrimSpace(*remove) != "" {
		if !preview.Available {
			return fmt.Errorf("core-set closure unavailable; seed remains unchanged: %s", preview.Error)
		}
		body, err := json.Marshal(map[string]any{"core": map[string]any{"seed": proposed}})
		if err != nil {
			return err
		}
		if err := patchOperatorState(core, strings.TrimSpace(*target), body); err != nil {
			return err
		}
	}
	encoded, err := (protojson.MarshalOptions{Multiline: true, Indent: "  "}).Marshal(preview)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(encoded, '\n'))
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "Core seed: %s\nClosure: %d scenario(s), %d resource(s)\n", strings.Join(preview.Seed, ", "), preview.MemberCounts["scenario"], preview.MemberCounts["resource"])
	return err
}

func fetchCoreSet(core *cliapp.ScenarioApp, seed []string) (*coreSetView, error) {
	return fetchCoreSetForTarget(core, seed, "local")
}

func fetchCoreSetForTarget(core *cliapp.ScenarioApp, seed []string, target string) (*coreSetView, error) {
	return onboardingselection.CoreSetForTarget(core, normalizedNames(seed), target)
}

func normalizedNames(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value = strings.ToLower(strings.TrimSpace(value)); value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func applyRecommendation(core *cliapp.ScenarioApp, target string) error {
	if err := rejectPendingOperatorInputs(core, target); err != nil {
		return err
	}
	accepted := &selectionv1.AcceptRecommendationResponse{}
	if err := onboardingselection.Request(core, onboardingselection.AcceptRecommendationProcedure, &selectionv1.AcceptRecommendationRequest{Target: target}, accepted); err != nil {
		return err
	}
	var err error
	planResponse := &applyv1.GetApplyPlanResponse{}
	if err := requestApply(core, applyPlanProcedure, &applyv1.GetApplyPlanRequest{Target: target}, planResponse); err != nil {
		return err
	}
	if err := renderApplyPlan(planResponse); err != nil {
		return err
	}
	applyResponse, err := startApplyWithConsent(core, target, planResponse)
	if err != nil {
		return err
	}
	var applyResult *applyv1.GetApplyRunResponse
	if applyResponse.GetRun() != nil {
		applyResult, err = waitForApply(core, target, applyResponse.GetRun())
		if err != nil {
			return err
		}
	}
	readinessResponse := &readinessv1.GetReadinessResponse{}
	if err := requestOperator(core, readinessGetProcedure, &readinessv1.GetReadinessRequest{Target: target}, readinessResponse); err != nil {
		return err
	}
	readiness, err := protojson.Marshal(readinessResponse)
	if err != nil {
		return err
	}
	if err := renderApplyReport(applyResult); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Readiness:", string(readiness))
	return err
}

func startApplyWithConsent(core *cliapp.ScenarioApp, target string, plan *applyv1.GetApplyPlanResponse) (*applyv1.StartApplyResponse, error) {
	if plan == nil || strings.TrimSpace(plan.GetPlanId()) == "" || strings.TrimSpace(plan.GetPlanDigest()) == "" || strings.TrimSpace(plan.GetRevision()) == "" {
		return nil, fmt.Errorf("apply plan is incomplete; reload the target plan before applying")
	}
	review := &applyv1.ReviewApplyResponse{}
	if err := requestApply(core, applyReviewProcedure, &applyv1.ReviewApplyRequest{
		Target: target, PlanId: plan.GetPlanId(), PlanDigest: plan.GetPlanDigest(), ExpectedRevision: plan.GetRevision(),
	}, review); err != nil {
		return nil, err
	}
	result := &applyv1.StartApplyResponse{}
	if err := requestApply(core, applyStartProcedure, &applyv1.StartApplyRequest{
		Target: target, PlanId: review.GetPlanId(), PlanDigest: review.GetPlanDigest(), ExpectedRevision: review.GetRevision(),
		ConsentReceiptId: review.GetConsentReceiptId(), IdempotencyKey: "wizard:" + target + ":" + plan.GetPlanDigest(),
	}, result); err != nil {
		return nil, err
	}
	return result, nil
}

func readOperatorInputQueue(core *cliapp.ScenarioApp, target string) (operatorInputQueue, error) {
	response := &operatorinputsv1.ListOperatorInputsResponse{}
	if err := requestOperator(core, operatorInputsListProcedure, &operatorinputsv1.ListOperatorInputsRequest{Target: target}, response); err != nil {
		return operatorInputQueue{}, err
	}
	queue := operatorInputQueue{Version: response.GetVersion()}
	for _, request := range response.GetRequests() {
		if request == nil {
			continue
		}
		kind := strings.ToLower(strings.TrimPrefix(request.GetKind().String(), "OPERATOR_INPUT_KIND_"))
		queue.Requests = append(queue.Requests, operatorInputRequest{ID: request.GetId(), Kind: kind, Title: request.GetTitle(), Description: request.GetDescription(), Default: request.GetDefaultValue(), Options: request.GetOptions(), Required: request.GetRequired(), Declinable: request.GetDeclinable()})
	}
	readiness := &readinessv1.GetReadinessResponse{}
	if err := requestOperator(core, readinessGetProcedure, &readinessv1.GetReadinessRequest{Target: target}, readiness); err != nil {
		return operatorInputQueue{}, err
	}
	queue.ExpectedRevision = readiness.GetConfigurationRevision()
	return queue, nil
}

func rejectPendingOperatorInputs(core *cliapp.ScenarioApp, target string) error {
	queue, err := readOperatorInputQueue(core, target)
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

func resolvePendingOperatorInputs(core *cliapp.ScenarioApp, target string, read, readSecret func(int, string) (string, error)) error {
	queue, err := readOperatorInputQueue(core, target)
	if err != nil {
		return err
	}
	if len(queue.Requests) == 0 {
		return nil
	}
	answers := make([]*operatorinputsv1.Answer, 0, len(queue.Requests))
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
		if request.Declinable {
			prompt += " (type decline to keep this optional control declined)"
		}
		readAnswer := read
		if request.Kind == "secret" {
			readAnswer = readSecret
		}
		value, err := readAnswer(4, prompt)
		if err != nil {
			return err
		}
		if request.Declinable && strings.EqualFold(strings.TrimSpace(value), "decline") {
			answers = append(answers, &operatorinputsv1.Answer{RequestId: request.ID, Declined: true})
			continue
		}
		if value == "" {
			value = request.Default
		}
		answers = append(answers, &operatorinputsv1.Answer{RequestId: request.ID, Value: value})
	}
	response := &operatorinputsv1.ResolveOperatorInputsResponse{}
	if err := requestOperator(core, operatorInputsResolveProcedure, &operatorinputsv1.ResolveOperatorInputsRequest{Target: target, ExpectedRevision: queue.ExpectedRevision, Answers: answers}, response); err != nil {
		return fmt.Errorf("resolve onboarding operator input: %w", err)
	}
	return nil
}

func requestOperator(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "operator inputs")
}

func patchOperatorState(core *cliapp.ScenarioApp, target string, body []byte) error {
	state := &operatorstatev1.OperatorState{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, state); err != nil {
		return fmt.Errorf("decode operator-state patch: %w", err)
	}
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(body, &patch); err != nil {
		return fmt.Errorf("decode operator-state patch fields: %w", err)
	}
	paths := make([]string, 0, len(patch))
	for path := range patch {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	response := &operatorstatev1.PatchOperatorStateResponse{}
	if err := support.RequestProto(core, operatorStatePatchProcedure, &operatorstatev1.PatchOperatorStateRequest{
		Target:     target,
		State:      state,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
	}, response, "operator state"); err != nil {
		return err
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

// printApplyPlan decodes a Connect response for compatibility with focused
// renderer tests. The CLI request path uses renderApplyPlan directly, so the
// production renderer only receives generated protobuf messages.
func printApplyPlan(body []byte) error {
	plan := &applyv1.GetApplyPlanResponse{}
	if err := protojson.Unmarshal(body, plan); err != nil {
		return fmt.Errorf("decode apply plan: %w", err)
	}
	return renderApplyPlan(plan)
}

// renderApplyPlan is the operator's disclosure before consent. It must answer
// four questions the flat kind/name list did not: how much is about to happen,
// which items change host state with elevation, which items are already in
// place, and what "apply" actually does to this machine. The plan is a
// desired-state list, so without the state split every entry reads as a
// pending change even when most are already satisfied.
func renderApplyPlan(response *applyv1.GetApplyPlanResponse) error {
	items := make([]planItem, 0, len(response.GetItems()))
	for _, item := range response.GetItems() {
		items = append(items, planItem{Kind: item.GetKind(), Name: item.GetName(), Required: item.GetRequired(), Privileged: item.GetPrivileged(), State: item.GetObservedState()})
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Apply plan: no changes. Nothing will be executed.")
		return nil
	}

	pending := filterByState(items, "pending")
	satisfied := filterByState(items, "satisfied")
	unknown := filterByState(items, "unknown")
	elevatedPending := 0
	for _, item := range pending {
		if item.Privileged {
			elevatedPending++
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nApply plan — %d selected item(s): %d not yet in place, %d already in place, %d not sampled.\n",
		len(items), len(pending), len(satisfied), len(unknown))

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
		_, _ = fmt.Fprintf(os.Stdout, "\nNOT SAMPLED — checking these costs a control-plane round trip each, so this plan did not (%d)\n", len(unknown))
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
	report := &applyv1.GetApplyRunResponse{}
	if err := protojson.Unmarshal(body, report); err != nil {
		return fmt.Errorf("decode apply report: %w", err)
	}
	return renderApplyReport(report)
}

func renderApplyReport(report *applyv1.GetApplyRunResponse) error {
	status := report.GetLegacyStatus()
	if status == "" {
		status = report.GetStatus().String()
	}
	_, _ = fmt.Fprintln(os.Stdout, "Apply report:", status)
	// A run can end without applying anything -- refused up front, for example.
	// Printing only the status would tell the operator that something went
	// wrong while withholding the one line that says what to do about it.
	for _, blocker := range report.GetBlockers() {
		_, _ = fmt.Fprintf(os.Stdout, "  ! %s: %s\n", blocker.GetName(), blocker.GetReason())
		if strings.TrimSpace(blocker.GetRemediation()) != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    fix: %s\n", blocker.GetRemediation())
		}
	}
	for _, item := range report.GetSteps() {
		outcome := item.GetLegacyOutcome()
		if outcome == "" {
			outcome = item.GetState().String()
		}
		detail := item.GetError()
		if detail == "" {
			detail = item.GetBlockedBy()
		}
		// An item can carry a reason without having failed -- a skipped item is
		// the plain case. Falling straight through to "completed" reported those
		// as if they had run.
		if detail == "" {
			detail = item.GetRemediation()
		}
		if detail == "" {
			detail = "completed"
		}
		_, _ = fmt.Fprintf(os.Stdout, "  - %s: %s (%s)\n", item.GetName(), outcome, detail)
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

// applyReconnectWindow is how long the wizard keeps trying to reach the API
// while an apply is in flight before it gives up on reconnecting.
//
// Applying a selection starts scenarios, and starting a scenario can restart
// the onboarding API -- so the API going away mid-apply is an expected event on
// this path, not a failure. The run itself is executed by a separate process
// and continues across that restart; the only thing that breaks is this
// client's connection. Treating the first refused connection as fatal is what
// made the operator's wizard vanish moments after they answered the consent
// prompt, with the run still progressing invisibly behind it.
//
// The window is generous because the restart it absorbs includes a rebuild: a
// cold scenario stage can take tens of seconds before the port is listening.
const (
	applyReconnectWindow = 5 * time.Minute
	applyPollInterval    = 500 * time.Millisecond
)

func requestApply(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "apply")
}

func requestSession(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "session")
}

func printSessionResponse(response proto.Message) error {
	return support.PrintProto(os.Stdout, response, "session")
}

func sessionStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard status")
	target := fs.String("target", "local", "Onboarding target scenario")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*target) == "" {
		return fmt.Errorf("--target is required")
	}
	response := &sessionv1.GetSessionResponse{}
	if err := requestSession(core, sessionGetProcedure, &sessionv1.GetSessionRequest{Target: strings.TrimSpace(*target)}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printSessionResponse(response)
	}
	_, err := fmt.Fprintf(os.Stdout, "Onboarding step %s (%d), first unsatisfied step %d, completion=%t\n", response.GetStepId(), response.GetStep(), response.GetFirstUnsatisfiedStep(), response.GetCompletion())
	return err
}

func advanceSessionStep(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard advance-step")
	target := fs.String("target", "local", "Onboarding target scenario")
	stepID := fs.String("step-id", "", "Stable onboarding step id")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*target) == "" || strings.TrimSpace(*stepID) == "" {
		return fmt.Errorf("--target and --step-id are required")
	}
	response := &sessionv1.GetSessionResponse{}
	if err := requestSession(core, sessionAdvanceProcedure, &sessionv1.AdvanceSessionStepRequest{Target: strings.TrimSpace(*target), StepId: strings.TrimSpace(*stepID)}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printSessionResponse(response)
	}
	_, err := fmt.Fprintf(os.Stdout, "Onboarding step advanced to %s (%d)\n", response.GetStepId(), response.GetStep())
	return err
}

func applyItems(steps []*applyv1.ApplyStep) []applyRunItem {
	items := make([]applyRunItem, 0, len(steps))
	for _, step := range steps {
		if step == nil {
			continue
		}
		outcome := step.GetLegacyOutcome()
		if outcome == "" {
			outcome = strings.ToLower(strings.TrimPrefix(step.GetState().String(), "APPLY_STEP_STATE_"))
		}
		items = append(items, applyRunItem{ID: step.GetId(), Kind: step.GetKind(), Name: step.GetName(), Outcome: outcome, Error: step.GetError()})
	}
	return items
}

func waitForApply(core *cliapp.ScenarioApp, target string, current *applyv1.GetApplyRunResponse) (*applyv1.GetApplyRunResponse, error) {
	if current == nil {
		return nil, fmt.Errorf("apply start returned no run")
	}
	var (
		unreachableSince time.Time
		announced        bool
	)
	reporter := newApplyProgressReporter(os.Stdout, clock.Real{}.Now)
	reporter.Observe(applyItems(current.GetSteps()))
	for current.GetStatus() == applyv1.ApplyRunState_APPLY_RUN_STATE_PENDING || current.GetStatus() == applyv1.ApplyRunState_APPLY_RUN_STATE_APPLYING {
		time.Sleep(applyPollInterval)

		response := &applyv1.GetApplyRunResponse{}
		err := requestApply(core, applyRunProcedure, &applyv1.GetApplyRunRequest{Target: target, RunId: current.GetRunId()}, response)
		if err != nil {
			// The run is server-owned. Keep waiting for it rather than
			// reporting a failure this client cannot actually observe.
			if unreachableSince.IsZero() {
				unreachableSince = clock.Real{}.Now()
			}
			if !announced {
				announced = true
				_, _ = fmt.Fprintln(os.Stdout, "The onboarding API restarted while applying; the run continues. Reconnecting...")
			}
			// A restarted scenario comes back on a freshly assigned port, and
			// the base this CLI resolved at startup points at the old one.
			// Without rebinding, every reconnection attempt would politely
			// retry an address nothing is listening on until the window ran out.
			if core.RebindToDetectedAPI() {
				_, _ = fmt.Fprintln(os.Stdout, "The onboarding API moved to", core.APIRootBase())
			}
			if time.Since(unreachableSince) > applyReconnectWindow {
				return nil, fmt.Errorf("apply run %s is still in progress but the onboarding API has been unreachable for %s: %w", current.GetRunId(), applyReconnectWindow, err)
			}
			continue
		}
		if !unreachableSince.IsZero() {
			unreachableSince = time.Time{}
			_, _ = fmt.Fprintln(os.Stdout, "Reconnected to the onboarding API.")
			reporter.Reconnected()
		}

		*current = *response
		reporter.Observe(applyItems(current.GetSteps()))
	}
	items := applyItems(current.GetSteps())
	reporter.Finish(current.GetLegacyStatus(), items)
	return current, nil
}

func apply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard commit")
	selectionPath := fs.String("selection", "", "Path to a selection document")
	target := fs.String("target", "local", "Onboarding target scenario")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*selectionPath) == "" {
		return fmt.Errorf("--selection is required")
	}
	if strings.TrimSpace(*target) == "" {
		return fmt.Errorf("--target is required")
	}
	body, err := support.ReadJSONFile(*selectionPath, true)
	if err != nil {
		return err
	}
	var selection Selection
	if err := json.Unmarshal(body, &selection); err != nil {
		return fmt.Errorf("decode selection: %w", err)
	}
	selection, err = expandSpecialSelection(core, selection, strings.TrimSpace(*target))
	if err != nil {
		return err
	}
	accepted := &selectionv1.AcceptRecommendationResponse{}
	if err := onboardingselection.Request(core, onboardingselection.AcceptRecommendationProcedure, &selectionv1.AcceptRecommendationRequest{Target: strings.TrimSpace(*target), Profile: selection.ActiveProfile, Selection: selectionDocument(selection)}, accepted); err != nil {
		return err
	}
	plan := &applyv1.GetApplyPlanResponse{}
	if err := requestApply(core, applyPlanProcedure, &applyv1.GetApplyPlanRequest{Target: strings.TrimSpace(*target)}, plan); err != nil {
		return err
	}
	if err := renderApplyPlan(plan); err != nil {
		return err
	}
	result, err := startApplyWithConsent(core, strings.TrimSpace(*target), plan)
	if err != nil {
		return err
	}
	run := result.GetRun()
	if run == nil {
		return fmt.Errorf("apply start returned no run")
	}
	final, err := waitForApply(core, strings.TrimSpace(*target), run)
	if err != nil {
		return err
	}
	if *jsonOutput {
		body, marshalErr := (protojson.MarshalOptions{Multiline: true, Indent: "  "}).Marshal(final)
		if marshalErr != nil {
			return marshalErr
		}
		_, err = fmt.Fprintln(os.Stdout, string(body))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Selection committed and applied"}, NextCommand: []string{support.CLIName + " wizard status"}})
}

func selectionDocument(selection Selection) *setupv1.Selection {
	document := &setupv1.Selection{SchemaVersion: "v1", Scenarios: append([]string(nil), selection.Scenarios...), CoreSeed: normalizedNames(selection.CoreSeed), OptionalResources: append([]string(nil), selection.OptionalResources...), Apply: selection.Apply}
	for _, name := range selection.Host.Tools {
		document.HostTools = append(document.HostTools, name)
	}
	for _, name := range selection.Host.Safeguards {
		document.HostSafeguards = append(document.HostSafeguards, name)
	}
	for name, optedIn := range selection.HostTools {
		if optedIn {
			document.HostTools = append(document.HostTools, name)
		}
	}
	for name, optedIn := range selection.HostSafeguards {
		if optedIn {
			document.HostSafeguards = append(document.HostSafeguards, name)
		}
	}
	document.OperatingMode = map[string]string{}
	for name, mode := range selection.OperatingMode {
		if mode.AutoRestart {
			document.OperatingMode[name] = "auto-restart"
		} else {
			document.OperatingMode[name] = "manual"
		}
	}
	if len(document.OperatingMode) == 0 {
		document.OperatingMode = nil
	}
	sort.Strings(document.Scenarios)
	sort.Strings(document.CoreSeed)
	sort.Strings(document.OptionalResources)
	sort.Strings(document.HostTools)
	sort.Strings(document.HostSafeguards)
	return document
}

// expandSpecialSelection resolves the small set of bridge profile aliases at
// the node where the selection is committed. The profile document remains
// portable, while the applied operator state contains concrete names rather
// than pretending that "all" is a scenario.
func expandSpecialSelection(core *cliapp.ScenarioApp, selection Selection, target string) (Selection, error) {
	var err error
	selection.Scenarios, err = expandScenarioNames(core, selection.Scenarios, target)
	if err != nil {
		return Selection{}, err
	}
	selection.OptionalResources, err = expandResourceNames(core, selection.OptionalResources, target)
	if err != nil {
		return Selection{}, err
	}
	return selection, nil
}

func expandScenarioNames(core *cliapp.ScenarioApp, names []string, target string) ([]string, error) {
	if !containsAlias(names, "all") {
		return removeAlias(names, "none"), nil
	}
	response := &selectionv1.ListScenariosResponse{}
	if err := onboardingselection.Request(core, onboardingselection.ListScenariosProcedure, &selectionv1.ListScenariosRequest{Target: target}, response); err != nil {
		return nil, fmt.Errorf("expand all scenarios: %w", err)
	}
	result := removeAlias(names, "all")
	for _, scenario := range response.Scenarios {
		if scenario.Enabled {
			result = append(result, scenario.Name)
		}
	}
	return uniqueNames(result), nil
}

func expandResourceNames(core *cliapp.ScenarioApp, names []string, target string) ([]string, error) {
	if !containsAlias(names, "enabled") {
		return removeAlias(names, "none"), nil
	}
	response := &selectionv1.GetUnionResponse{}
	if err := onboardingselection.Request(core, onboardingselection.GetUnionProcedure, &selectionv1.GetUnionRequest{Target: target}, response); err != nil {
		return nil, fmt.Errorf("expand enabled resources: %w", err)
	}
	result := removeAlias(names, "enabled")
	for _, resource := range response.ResourceModels {
		if resource.Enabled {
			result = append(result, resource.Name)
		}
	}
	return uniqueNames(result), nil
}

func containsAlias(names []string, alias string) bool {
	for _, name := range names {
		if name == alias {
			return true
		}
	}
	return false
}

func removeAlias(names []string, alias string) []string {
	result := make([]string, 0, len(names))
	for _, name := range names {
		if name != alias {
			result = append(result, name)
		}
	}
	return result
}

func uniqueNames(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	result := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok || strings.TrimSpace(name) == "" {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func exportSelection(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard export")
	outputPath := fs.String("output", "", "Output selection document path")
	target := fs.String("target", "local", "Onboarding target scenario")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*outputPath) == "" || strings.TrimSpace(*target) == "" {
		return fmt.Errorf("--output and --target are required")
	}
	selection, err := currentSelectionForTarget(core, strings.TrimSpace(*target))
	if err != nil {
		return err
	}
	data, _ := json.MarshalIndent(selection, "", "  ")
	if err := writePrivateExport(*outputPath, append(data, '\n')); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Selection exported to", *outputPath)
	return err
}

func currentSelection(core *cliapp.ScenarioApp) (Selection, error) {
	return currentSelectionForTarget(core, "local")
}

func currentSelectionForTarget(core *cliapp.ScenarioApp, target string) (Selection, error) {
	response := &selectionv1.ListScenariosResponse{}
	if err := onboardingselection.Request(core, onboardingselection.ListScenariosProcedure, &selectionv1.ListScenariosRequest{Target: target}, response); err != nil {
		return Selection{}, err
	}
	selection := Selection{Scenarios: []string{}, ScenarioState: map[string]bool{}}
	for _, scenario := range response.Scenarios {
		selection.ScenarioState[scenario.Name] = scenario.Enabled
		if scenario.Enabled {
			selection.Scenarios = append(selection.Scenarios, scenario.Name)
		}
	}
	stateResponse := &operatorstatev1.GetOperatorStateResponse{}
	if err := support.RequestProto(core, operatorStateGetProcedure, &operatorstatev1.GetOperatorStateRequest{Target: target}, stateResponse, "operator state"); err != nil {
		return selection, nil
	}
	stateBody, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(stateResponse.GetState())
	if err != nil {
		return selection, nil
	}
	var state exportedOperatorState
	if json.Unmarshal(stateBody, &state) == nil {
		applyExportedOperatorState(&selection, state)
	}
	return selection, nil
}

type supportExportDocument struct {
	SchemaVersion string            `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	Target        string            `json:"target"`
	Included      []string          `json:"included"`
	Selection     *Selection        `json:"selection,omitempty"`
	Readiness     *supportReadiness `json:"readiness,omitempty"`
	Session       *supportSession   `json:"session,omitempty"`
}

type supportReadiness struct {
	Target                string              `json:"target"`
	ConfigurationRevision string              `json:"configuration_revision"`
	ExpiresAt             string              `json:"expires_at"`
	Status                string              `json:"status"`
	Scenarios             []string            `json:"scenarios"`
	Resources             []string            `json:"resources"`
	Credentials           []supportCredential `json:"credentials"`
	Hosts                 []supportHost       `json:"hosts"`
	Blockers              []completionBlocker `json:"blockers"`
	Degraded              []completionBlocker `json:"degraded"`
	DegradedDigest        string              `json:"degraded_digest,omitempty"`
	DegradedAcknowledged  bool                `json:"degraded_acknowledged"`
	CheckedAt             string              `json:"checked_at"`
	Recovery              supportRecovery     `json:"recovery"`
}

type supportCredential struct {
	Resource     string `json:"resource"`
	LogicalID    string `json:"logical_id"`
	Field        string `json:"field"`
	Label        string `json:"label"`
	Required     bool   `json:"required"`
	Status       string `json:"status"`
	Provisioning string `json:"provisioning,omitempty"`
	DerivedFrom  string `json:"derived_from,omitempty"`
}

type supportHost struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
}

type supportRecovery struct {
	ReceiptExists  bool     `json:"receipt_exists"`
	ExportedAt     string   `json:"exported_at,omitempty"`
	EntryCount     int      `json:"entry_count"`
	Uncovered      []string `json:"uncovered"`
	RequiredAbsent []string `json:"required_absent"`
}

type supportSession struct {
	Step                 int32  `json:"step"`
	StepID               string `json:"step_id"`
	FirstUnsatisfiedStep int32  `json:"first_unsatisfied_step"`
	Completion           bool   `json:"completion"`
}

func supportExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard support-export")
	outputPath := fs.String("output", "", "Output diagnostic path")
	include := fs.String("include", "", "Comma-separated metadata sections: selection,readiness,session")
	target := fs.String("target", "local", "Onboarding target scenario")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*outputPath) == "" || strings.TrimSpace(*target) == "" {
		return fmt.Errorf("--output and --target are required")
	}
	included, err := supportExportSections(*include)
	if err != nil {
		return err
	}
	document := supportExportDocument{SchemaVersion: "vrooli-onboarding-support.v1", GeneratedAt: clock.Real{}.Now().UTC().Format(time.RFC3339Nano), Target: strings.TrimSpace(*target), Included: included}
	for _, section := range included {
		switch section {
		case "selection":
			value, err := currentSelectionForTarget(core, strings.TrimSpace(*target))
			if err != nil {
				return fmt.Errorf("collect selection: %w", err)
			}
			document.Selection = &value
		case "readiness":
			response := &readinessv1.GetReadinessResponse{}
			if err := requestOperator(core, readinessGetProcedure, &readinessv1.GetReadinessRequest{Target: strings.TrimSpace(*target)}, response); err != nil {
				return fmt.Errorf("collect readiness: %w", err)
			}
			body, err := protojson.Marshal(response)
			if err != nil {
				return fmt.Errorf("encode readiness: %w", err)
			}
			var value supportReadiness
			if err := json.Unmarshal(body, &value); err != nil {
				return fmt.Errorf("decode readiness: %w", err)
			}
			document.Readiness = &value
		case "session":
			response := &sessionv1.GetSessionResponse{}
			if err := requestSession(core, sessionGetProcedure, &sessionv1.GetSessionRequest{Target: strings.TrimSpace(*target)}, response); err != nil {
				return fmt.Errorf("collect session: %w", err)
			}
			document.Session = &supportSession{Step: response.GetStep(), StepID: response.GetStepId(), FirstUnsatisfiedStep: response.GetFirstUnsatisfiedStep(), Completion: response.GetCompletion()}
		}
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode support export: %w", err)
	}
	if err := writePrivateExport(*outputPath, append(data, '\n')); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Metadata-only support diagnostic exported to", *outputPath)
	return err
}

func supportExportSections(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("--include is required; choose one or more of selection,readiness,session")
	}
	valid := map[string]bool{"selection": true, "readiness": true, "session": true}
	seen := map[string]bool{}
	sections := make([]string, 0, 3)
	for _, value := range strings.Split(raw, ",") {
		section := strings.TrimSpace(value)
		if section == "" {
			continue
		}
		if !valid[section] {
			return nil, fmt.Errorf("unsupported support-export section %q; choose selection,readiness,session", section)
		}
		if !seen[section] {
			seen[section] = true
			sections = append(sections, section)
		}
	}
	if len(sections) == 0 {
		return nil, fmt.Errorf("--include must name at least one section")
	}
	return sections, nil
}

func writePrivateExport(path string, data []byte) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("output path is required")
	}
	absolutePath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("resolve support-export path: %w", err)
	}
	directory := filepath.Dir(absolutePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create support-export directory: %w", err)
	}
	resolvedDirectory, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return fmt.Errorf("resolve support-export directory: %w", err)
	}
	if filepath.Clean(resolvedDirectory) != filepath.Clean(directory) {
		return fmt.Errorf("refuse support-export directory through symlink: %s", directory)
	}
	if info, err := os.Lstat(absolutePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse support-export symlink: %s", path)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect support-export path: %w", err)
	}
	file, err := os.CreateTemp(directory, ".vrooli-onboarding-export-*")
	if err != nil {
		return fmt.Errorf("create support-export staging file: %w", err)
	}
	temporaryPath := file.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("protect support export: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write support export: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync support export: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close support export: %w", err)
	}
	if err := os.Rename(temporaryPath, absolutePath); err != nil {
		return fmt.Errorf("install support export: %w", err)
	}
	removeTemporary = false
	return nil
}

// exportedOperatorState is deliberately metadata-only. Pointer fields retain
// an explicit false choice instead of collapsing it into an omitted value,
// while no credential or private completion field is copied into the
// automation document.
type exportedOperatorState struct {
	ActiveProfile *string `json:"active_profile"`
	Core          struct {
		Seed []string `json:"seed"`
	} `json:"core"`
	Scenarios map[string]struct {
		Enabled     *bool `json:"enabled"`
		AutoRestart *bool `json:"auto_restart"`
	} `json:"scenarios"`
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

func applyExportedOperatorState(selection *Selection, state exportedOperatorState) {
	if state.ActiveProfile != nil {
		selection.ActiveProfile = *state.ActiveProfile
	}
	if len(state.Core.Seed) > 0 {
		selection.CoreSeed = append([]string(nil), state.Core.Seed...)
	}
	if len(state.Scenarios) > 0 {
		if selection.ScenarioState == nil {
			selection.ScenarioState = map[string]bool{}
		}
		selection.OperatingMode = map[string]struct {
			AutoRestart bool `json:"auto_restart"`
		}{}
		for name, choice := range state.Scenarios {
			if choice.Enabled != nil {
				selection.ScenarioState[name] = *choice.Enabled
				if *choice.Enabled && !containsName(selection.Scenarios, name) {
					selection.Scenarios = append(selection.Scenarios, name)
				}
			}
			if choice.AutoRestart != nil {
				selection.OperatingMode[name] = struct {
					AutoRestart bool `json:"auto_restart"`
				}{AutoRestart: *choice.AutoRestart}
			}
		}
	}
	selection.Resources = boolChoices(state.Resources, func(choice struct {
		Enabled *bool `json:"enabled"`
	},
	) *bool {
		return choice.Enabled
	})
	selection.HostTools = boolChoices(state.HostTools, func(choice struct {
		OptedIn *bool `json:"opted_in"`
	},
	) *bool {
		return choice.OptedIn
	})
	selection.HostSafeguards = boolChoices(state.HostSafeguards, func(choice struct {
		OptedIn *bool `json:"opted_in"`
	},
	) *bool {
		return choice.OptedIn
	})
}

func containsName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func boolChoices[T any](choices map[string]T, value func(T) *bool) map[string]bool {
	if len(choices) == 0 {
		return nil
	}
	result := make(map[string]bool, len(choices))
	for name, choice := range choices {
		if enabled := value(choice); enabled != nil {
			result[name] = *enabled
		}
	}
	return result
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
		return fmt.Errorf("configuration is not complete: %d blocking item(s) remain; blockers: %s", len(response.Blockers), completionBlockerDetails(response.Blockers))
	}
	if len(response.Degraded) > 0 && !response.DegradedAcknowledged {
		_, _ = fmt.Fprintf(os.Stdout, "Accept the degraded set with: vrooli-onboarding readiness acknowledge-degraded --digest %s\n", response.DegradedDigest)
		return fmt.Errorf("configuration is not complete: %d optional item(s) need an explicit acknowledgement; degraded: %s", len(response.Degraded), completionBlockerDetails(response.Degraded))
	}
	return nil
}

// completionBlockerDetails is deliberately metadata-only. The wizard error
// is also consumed by remote orchestration, so it must retain the same safe
// diagnostic context as the human-readable lines written above.
func completionBlockerDetails(items []completionBlocker) string {
	details := make([]string, 0, len(items))
	for _, item := range items {
		details = append(details, fmt.Sprintf("%s %s — %s; next: %s", item.Kind, item.Name, item.Reason, item.Remediation))
	}
	return strings.Join(details, " | ")
}
