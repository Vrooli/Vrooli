package phases

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	"test-genie/cli/execute"
	"test-genie/cli/internal/apijson"
)

const usage = `usage: test-genie phases <list|inspect|applicability|plan> ...
  list [--json]
  inspect <phase> [--json]
  applicability <target> [--phase <phase>] [--preset <preset>] [--json]
  plan <target> [--preset <preset>] [--phase <phase>] [--skip <phase>] [--json]`

type phaseList struct {
	Items []phaseDescriptor `json:"items"`
	Count int               `json:"count"`
}

type phaseDescriptor struct {
	Name                  string         `json:"name"`
	Provider              string         `json:"provider,omitempty"`
	Source                string         `json:"source"`
	Description           string         `json:"description,omitempty"`
	Optional              bool           `json:"optional"`
	DefaultTimeoutSeconds int            `json:"defaultTimeoutSeconds,omitempty"`
	DocPath               string         `json:"docPath,omitempty"`
	DescriptorPath        string         `json:"descriptorPath,omitempty"`
	Policy                map[string]any `json:"policy,omitempty"`
	Runnability           map[string]any `json:"runnability,omitempty"`
	FindingSource         string         `json:"findingSource,omitempty"`
}

type inspectResponse struct {
	Phase phaseDescriptor `json:"phase"`
}

// Run executes the phase inspection command group.
func Run(api *cliutil.APIClient, args []string) error {
	return run(api, args, os.Stdout)
}

func run(api *cliutil.APIClient, args []string, w io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "list":
		return runList(api, args[1:], w)
	case "inspect":
		return runInspect(api, args[1:], w)
	case "applicability":
		return runApplicability(api, args[1:], w)
	case "plan":
		return runPlan(api, args[1:], w)
	default:
		return errors.New(usage)
	}
}

func runList(api *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("phases list", flag.ContinueOnError)
	asJSON := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return errors.New(usage)
	}
	raw, err := api.Get("/api/v1/phases", nil)
	if err != nil {
		return err
	}
	if *asJSON {
		writeJSON(w, raw)
		return nil
	}
	payload, err := apijson.Parse[phaseList](raw, "parse phases list")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Test Genie phases (%d)\n", payload.Count)
	for _, phase := range payload.Items {
		provider := phase.Provider
		if provider == "" {
			provider = phase.Source
		}
		fmt.Fprintf(w, "  %-16s %-28s %s\n", phase.Name, provider, phase.Description)
	}
	return nil
}

func runInspect(api *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("phases inspect", flag.ContinueOnError)
	asJSON := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 || strings.TrimSpace(rest[0]) == "" {
		return errors.New(usage)
	}
	raw, err := api.Get("/api/v1/phases/"+url.PathEscape(rest[0]), nil)
	if err != nil {
		return err
	}
	if *asJSON {
		writeJSON(w, raw)
		return nil
	}
	payload, err := apijson.Parse[inspectResponse](raw, "parse phase inspection")
	if err != nil {
		return err
	}
	printPhase(w, payload.Phase)
	return nil
}

func runApplicability(api *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("phases applicability", flag.ContinueOnError)
	phase := fs.String("phase", "", "Restrict output to one phase")
	preset := fs.String("preset", "", "Preset to use while previewing applicability")
	asJSON := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 || strings.TrimSpace(rest[0]) == "" {
		return errors.New(usage)
	}
	query := url.Values{"target": []string{strings.TrimSpace(rest[0])}}
	if strings.TrimSpace(*phase) != "" {
		query.Set("phase", strings.TrimSpace(*phase))
	}
	if strings.TrimSpace(*preset) != "" {
		query.Set("preset", strings.TrimSpace(*preset))
	}
	raw, err := api.Get("/api/v1/phases/applicability", query)
	if err != nil {
		return err
	}
	if *asJSON {
		writeJSON(w, raw)
		return nil
	}
	var payload struct {
		ScenarioName        string         `json:"scenarioName"`
		Phase               plannedPhase   `json:"phase"`
		Phases              []plannedPhase `json:"phases"`
		NotApplicablePhases []plannedPhase `json:"notApplicablePhases"`
		Warnings            []string       `json:"warnings"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("parse applicability preview: %w", err)
	}
	fmt.Fprintf(w, "Applicability for %s\n", payload.ScenarioName)
	if payload.Phase.Name != "" {
		printPlannedPhase(w, payload.Phase)
		return nil
	}
	for _, phase := range payload.Phases {
		printPlannedPhase(w, phase)
	}
	for _, phase := range payload.NotApplicablePhases {
		printPlannedPhase(w, phase)
	}
	for _, warning := range payload.Warnings {
		fmt.Fprintf(w, "  warning: %s\n", warning)
	}
	return nil
}

func runPlan(api *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("phases plan", flag.ContinueOnError)
	preset := fs.String("preset", "", "Preset name")
	phaseCSV := fs.String("phase", "", "Comma-separated phase selection")
	skipCSV := fs.String("skip", "", "Comma-separated phases to skip")
	asJSON := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 || strings.TrimSpace(rest[0]) == "" {
		return errors.New(usage)
	}
	req := execute.Request{
		ScenarioName: strings.TrimSpace(rest[0]),
		Preset:       strings.TrimSpace(*preset),
		Phases:       cliutil.ParseCSV(*phaseCSV),
		Skip:         cliutil.ParseCSV(*skipCSV),
	}
	raw, err := api.Request("POST", "/api/v1/executions/plan", nil, req)
	if err != nil {
		return err
	}
	if *asJSON {
		writeJSON(w, raw)
		return nil
	}
	var payload struct {
		ScenarioName        string         `json:"scenarioName"`
		PresetUsed          string         `json:"presetUsed"`
		Phases              []plannedPhase `json:"phases"`
		NotApplicablePhases []plannedPhase `json:"notApplicablePhases"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("parse execution plan: %w", err)
	}
	fmt.Fprintf(w, "Plan for %s", payload.ScenarioName)
	if payload.PresetUsed != "" {
		fmt.Fprintf(w, " (%s)", payload.PresetUsed)
	}
	fmt.Fprintln(w)
	for _, phase := range payload.Phases {
		printPlannedPhase(w, phase)
	}
	for _, phase := range payload.NotApplicablePhases {
		printPlannedPhase(w, phase)
	}
	return nil
}

type plannedPhase struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description,omitempty"`
	SelectionStatus      string   `json:"selectionStatus,omitempty"`
	ApplicabilityStatus  string   `json:"applicabilityStatus,omitempty"`
	ApplicabilityReasons []reason `json:"applicabilityReasons,omitempty"`
	ProviderReadiness    string   `json:"providerReadiness,omitempty"`
	Freshness            string   `json:"freshness,omitempty"`
	TimeoutSeconds       int      `json:"timeoutSeconds,omitempty"`
	DescriptorPath       string   `json:"descriptorPath,omitempty"`
}

type reason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func printPhase(w io.Writer, phase phaseDescriptor) {
	fmt.Fprintf(w, "%s\n", phase.Name)
	if phase.Provider != "" {
		fmt.Fprintf(w, "  provider: %s\n", phase.Provider)
	}
	if phase.Source != "" {
		fmt.Fprintf(w, "  source: %s\n", phase.Source)
	}
	if phase.Description != "" {
		fmt.Fprintf(w, "  description: %s\n", phase.Description)
	}
	if phase.FindingSource != "" {
		fmt.Fprintf(w, "  findingSource: %s\n", phase.FindingSource)
	}
	if phase.DescriptorPath != "" {
		fmt.Fprintf(w, "  descriptor: %s\n", phase.DescriptorPath)
	}
}

func printPlannedPhase(w io.Writer, phase plannedPhase) {
	status := phase.ApplicabilityStatus
	if status == "" {
		status = phase.SelectionStatus
	}
	fmt.Fprintf(w, "  %-16s %s", phase.Name, status)
	if phase.SelectionStatus != "" && phase.SelectionStatus != status {
		fmt.Fprintf(w, " selection=%s", phase.SelectionStatus)
	}
	if phase.ProviderReadiness != "" {
		fmt.Fprintf(w, " readiness=%s", phase.ProviderReadiness)
	}
	if phase.Freshness != "" {
		fmt.Fprintf(w, " freshness=%s", phase.Freshness)
	}
	fmt.Fprintln(w)
	for _, reason := range phase.ApplicabilityReasons {
		fmt.Fprintf(w, "      - %s: %s\n", reason.Code, reason.Message)
	}
}

func writeJSON(w io.Writer, raw []byte) {
	_, _ = w.Write(raw)
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		_, _ = fmt.Fprintln(w)
	}
}
