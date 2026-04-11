package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

type scenarioListPortOutput struct {
	Key  string `json:"key"`
	Step string `json:"step,omitempty"`
	Port int    `json:"port"`
}

type scenarioListItemOutput struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Version     string                   `json:"version,omitempty"`
	Status      string                   `json:"status"`
	Tags        []string                 `json:"tags"`
	Path        string                   `json:"path"`
	Ports       []scenarioListPortOutput `json:"ports"`
}

type scenarioStatusItemOutput struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name,omitempty"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags"`
	Status      string         `json:"status"`
	Processes   int            `json:"processes"`
	Runtime     string         `json:"runtime"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	Ports       map[string]int `json:"ports"`
	Health      any            `json:"health_status"`
}

type scenarioInfoOutput struct {
	Success  bool                     `json:"success"`
	Scenario scenarioInfoScenarioData `json:"scenario"`
	Runtime  scenarioInfoRuntimeData  `json:"runtime"`
}

type scenarioInfoScenarioData struct {
	Name             string                  `json:"name"`
	DisplayName      string                  `json:"display_name,omitempty"`
	Description      string                  `json:"description,omitempty"`
	Version          string                  `json:"version,omitempty"`
	Type             string                  `json:"type,omitempty"`
	Category         string                  `json:"category,omitempty"`
	Tags             []string                `json:"tags"`
	Path             string                  `json:"path"`
	ServicePath      string                  `json:"service_path"`
	SandboxRedirect  bool                    `json:"sandbox_redirected"`
	ConfigVersion    string                  `json:"config_version,omitempty"`
	LifecycleVersion string                  `json:"lifecycle_version,omitempty"`
	Ports            []scenario.PortSummary  `json:"ports"`
	Phases           []scenario.PhaseSummary `json:"phases"`
}

type scenarioInfoRuntimeData struct {
	Status      string                   `json:"status"`
	Processes   int                      `json:"processes"`
	Runtime     string                   `json:"runtime"`
	StartedAt   *time.Time               `json:"started_at,omitempty"`
	Ports       map[string]int           `json:"ports"`
	ProcessInfo []process.Record         `json:"process_records"`
	ListPorts   []scenarioListPortOutput `json:"list_ports"`
}

type scenarioStatusSingleOutput struct {
	Success  bool                     `json:"success"`
	Scenario scenarioStatusItemOutput `json:"scenario"`
	Info     scenarioInfoScenarioData `json:"info"`
	Runtime  scenarioInfoRuntimeData  `json:"runtime"`
}

type scenarioLifecycleItemOutput struct {
	Name               string         `json:"name"`
	Status             string         `json:"status"`
	Health             string         `json:"health,omitempty"`
	Ports              map[string]int `json:"ports,omitempty"`
	FailedDependencies []string       `json:"failed_dependencies,omitempty"`
}

func runScenarioCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		showScenarioHelp(stdout)
		return nil
	}

	handler, ok := scenarioCommands[args[0]]
	if !ok {
		return newUnknownScenarioCommandError(args[0])
	}
	return handler(root, globals, args[1:], stdout, stderr)
}

func runScenarioStartCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli scenario start <name> [name2...] [--path <path>] [--best-effort] [--clean-stale] [--open] [--json]")
			return nil
		}
	}

	names, opts, jsonFlag, openAfter, err := parseScenarioStartArgs(globals.json, args)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("scenario start requires at least one scenario name")
	}
	if opts.CustomPath != "" && len(names) != 1 {
		return fmt.Errorf("scenario start with --path accepts exactly one scenario name")
	}

	format, err := cliout.ParseFormat("", jsonFlag)
	if err != nil {
		return err
	}

	runnerOut := stdout
	if format == cliout.FormatJSON {
		runnerOut = stderr
	}

	service, err := newScenarioService(root, runnerOut, stderr)
	if err != nil {
		return err
	}

	items := make([]scenarioLifecycleItemOutput, 0, len(names))
	for _, name := range names {
		result, err := service.StartDetailed(name, opts)
		if err != nil {
			return err
		}

		status := "started"
		if result.AlreadyRunning {
			status = "already_running"
		}
		items = append(items, scenarioLifecycleItemOutput{
			Name:               result.Scenario.Slug,
			Status:             status,
			Health:             result.Details.Health,
			Ports:              envPortMap(result.Scenario.Manifest, result.AllocatedPorts),
			FailedDependencies: append([]string(nil), result.FailedDependencies...),
		})

		if openAfter {
			resolved, err := service.ResolvePort(name, "UI_PORT")
			if err != nil {
				return err
			}
			if err := scenarioOpenURLFn(resolved.URL); err != nil {
				return err
			}
		}
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":   true,
			"scenarios": items,
		})
	}

	for _, item := range items {
		if item.Status == "already_running" {
			_, _ = fmt.Fprintf(stdout, "Scenario '%s' is already running", item.Name)
		} else {
			_, _ = fmt.Fprintf(stdout, "Started scenario '%s'", item.Name)
		}
		if item.Health != "" {
			_, _ = fmt.Fprintf(stdout, " (%s)", item.Health)
		}
		_, _ = fmt.Fprintln(stdout)
		if len(item.Ports) > 0 {
			_, _ = fmt.Fprintf(stdout, "  Ports: %s\n", formatPortMap(item.Ports))
		}
		if len(item.FailedDependencies) > 0 {
			_, _ = fmt.Fprintf(stdout, "  Failed dependencies: %s\n", strings.Join(item.FailedDependencies, ", "))
		}
	}
	return nil
}

func runScenarioStopCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	name, jsonFlag, err := parseScenarioNameAndJSON("stop", globals.json, args)
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", jsonFlag)
	if err != nil {
		return err
	}

	runnerOut := stdout
	if format == cliout.FormatJSON {
		runnerOut = stderr
	}

	runner, err := newScenarioLifecycleRunner(root, runnerOut, stderr)
	if err != nil {
		return err
	}
	if err := runner.Stop(name, lifecycle.StopOptions{}); err != nil {
		return err
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":  true,
			"scenario": name,
			"status":   "stopped",
		})
	}

	_, _ = fmt.Fprintf(stdout, "Stopped scenario '%s'\n", name)
	return nil
}

func runScenarioRestartCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli scenario restart <name> [--path <path>] [--best-effort] [--clean-stale] [--open] [--json]")
			return nil
		}
	}

	name, opts, jsonFlag, openAfter, err := parseScenarioSingleStartArgs("restart", globals.json, args)
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", jsonFlag)
	if err != nil {
		return err
	}

	runnerOut := stdout
	if format == cliout.FormatJSON {
		runnerOut = stderr
	}

	service, err := newScenarioService(root, runnerOut, stderr)
	if err != nil {
		return err
	}
	result, err := service.RestartDetailed(name, opts)
	if err != nil {
		return err
	}

	item := scenarioLifecycleItemOutput{
		Name:               result.Scenario.Slug,
		Status:             "restarted",
		Health:             result.Details.Health,
		Ports:              envPortMap(result.Scenario.Manifest, result.AllocatedPorts),
		FailedDependencies: append([]string(nil), result.FailedDependencies...),
	}

	if openAfter {
		resolved, err := service.ResolvePort(name, "UI_PORT")
		if err != nil {
			return err
		}
		if err := scenarioOpenURLFn(resolved.URL); err != nil {
			return err
		}
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":  true,
			"scenario": item,
		})
	}

	_, _ = fmt.Fprintf(stdout, "Restarted scenario '%s'", item.Name)
	if item.Health != "" {
		_, _ = fmt.Fprintf(stdout, " (%s)", item.Health)
	}
	_, _ = fmt.Fprintln(stdout)
	if len(item.Ports) > 0 {
		_, _ = fmt.Fprintf(stdout, "  Ports: %s\n", formatPortMap(item.Ports))
	}
	if len(item.FailedDependencies) > 0 {
		_, _ = fmt.Fprintf(stdout, "  Failed dependencies: %s\n", strings.Join(item.FailedDependencies, ", "))
	}
	return nil
}

func newScenarioLifecycleRunner(root string, stdout, stderr io.Writer) (*lifecycle.Runner, error) {
	home, err := process.HomeDir()
	if err != nil {
		return nil, err
	}
	return lifecycle.NewRunner(root, home, stdout, stderr)
}

func newScenarioService(root string, stdout, stderr io.Writer) (*orchestrator.Service, error) {
	home, err := process.HomeDir()
	if err != nil {
		return nil, err
	}
	return orchestrator.New(root, home, stdout, stderr), nil
}

func runScenarioListCommand(root string, globals globalOptions, args []string, stdout io.Writer) error {
	includePorts := false
	jsonFlag := globals.json
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonFlag = true
		case "--include-ports":
			includePorts = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli scenario list [--json] [--include-ports]")
			return nil
		default:
			return fmt.Errorf("unknown option for scenario list: %s", arg)
		}
	}

	format, err := cliout.ParseFormat("", jsonFlag)
	if err != nil {
		return err
	}

	service, err := newScenarioService(root, io.Discard, io.Discard)
	if err != nil {
		return err
	}
	inventory, err := service.Inventory()
	if err != nil {
		return err
	}

	items := make([]scenarioListItemOutput, 0, len(inventory))
	runningCount := 0
	for _, item := range inventory {
		status := "available"
		if item.Details.Status == "running" {
			status = item.Details.Status
			runningCount++
		}

		listPorts := []scenarioListPortOutput{}
		if includePorts && item.Details.Status == "running" {
			listPorts = runtimePortOutputs(item.Details.PortBindings)
		}

		items = append(items, scenarioListItemOutput{
			Name:        item.Scenario.Slug,
			Description: item.Scenario.Manifest.Service.Description,
			Version:     item.Scenario.Manifest.Service.Version,
			Status:      status,
			Tags:        copyStrings(item.Scenario.Manifest.Service.Tags),
			Path:        item.Scenario.Path + string(os.PathSeparator),
			Ports:       listPorts,
		})
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success": true,
			"summary": map[string]int{
				"total_scenarios": len(items),
				"running":         runningCount,
				"available":       len(items) - runningCount,
			},
			"scenarios": items,
		})
	}

	_, _ = fmt.Fprintln(stdout, "[INFO]    Available scenarios:")
	for _, item := range items {
		line := "  • " + item.Name
		if item.Description != "" {
			line += " - " + item.Description
		}
		if includePorts && len(item.Ports) > 0 {
			portParts := make([]string, 0, len(item.Ports))
			for _, port := range item.Ports {
				portParts = append(portParts, fmt.Sprintf("%s=%d", port.Key, port.Port))
			}
			line += " (ports: " + strings.Join(portParts, ", ") + ")"
		}
		_, _ = fmt.Fprintln(stdout, line)
	}
	return nil
}

func runScenarioInfoCommand(root string, globals globalOptions, args []string, stdout io.Writer) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli scenario info <name> [--json]")
			return nil
		}
	}

	name, jsonFlag, err := parseScenarioNameAndJSON("info", globals.json, args)
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", jsonFlag)
	if err != nil {
		return err
	}

	service, err := newScenarioService(root, io.Discard, io.Discard)
	if err != nil {
		return err
	}
	detail, err := service.Detail(name)
	if err != nil {
		return err
	}

	output := scenarioInfoOutput{
		Success:  true,
		Scenario: buildScenarioInfoData(detail.Scenario),
		Runtime:  buildScenarioRuntimeData(detail.Scenario.Manifest, detail.Runtime),
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, output)
	}

	writeScenarioInfoHuman(stdout, output.Scenario, output.Runtime)
	return nil
}

func runScenarioStatusCommand(root string, globals globalOptions, args []string, stdout io.Writer) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli scenario status [name] [--json]")
			return nil
		}
	}

	name, jsonFlag, err := parseOptionalScenarioNameAndJSON("status", globals.json, args)
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", jsonFlag)
	if err != nil {
		return err
	}

	if name == "" {
		service, err := newScenarioService(root, io.Discard, io.Discard)
		if err != nil {
			return err
		}
		inventory, err := service.Inventory()
		if err != nil {
			return err
		}

		items := make([]scenarioStatusItemOutput, 0, len(inventory))
		runningCount := 0
		for _, item := range inventory {
			statusItem := buildScenarioStatusDetail(item)
			if statusItem.Status == "running" {
				runningCount++
			}
			items = append(items, statusItem)
		}

		if format == cliout.FormatJSON {
			return cliout.WriteJSON(stdout, map[string]any{
				"success": true,
				"summary": map[string]int{
					"total_scenarios": len(items),
					"running":         runningCount,
					"stopped":         len(items) - runningCount,
				},
				"scenarios": items,
			})
		}

		writeScenarioStatusTable(stdout, items)
		return nil
	}

	service, err := newScenarioService(root, io.Discard, io.Discard)
	if err != nil {
		return err
	}
	detail, err := service.Detail(name)
	if err != nil {
		return err
	}

	output := scenarioStatusSingleOutput{
		Success:  true,
		Scenario: buildScenarioStatusDetail(detail),
		Info:     buildScenarioInfoData(detail.Scenario),
		Runtime:  buildScenarioRuntimeData(detail.Scenario.Manifest, detail.Runtime),
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, output)
	}

	writeScenarioStatusHuman(stdout, output)
	return nil
}

func loadScenarioState(root string) ([]scenario.Scenario, map[string]process.ScenarioRuntime, error) {
	service, err := newScenarioService(root, io.Discard, io.Discard)
	if err != nil {
		return nil, nil, err
	}
	inventory, err := service.Inventory()
	if err != nil {
		return nil, nil, err
	}

	items := make([]scenario.Scenario, 0, len(inventory))
	runtimes := make(map[string]process.ScenarioRuntime, len(inventory))
	for _, item := range inventory {
		items = append(items, item.Scenario)
		if item.Runtime.ProcessCount > 0 {
			runtimes[item.Scenario.Slug] = item.Runtime
		}
	}
	return items, runtimes, nil
}

func loadScenarioDetail(root, name string) (scenario.Scenario, process.ScenarioRuntime, string, error) {
	service, err := newScenarioService(root, io.Discard, io.Discard)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}
	detail, err := service.Detail(name)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}
	return detail.Scenario, detail.Runtime, detail.Details.Health, nil
}

func buildScenarioStatusItem(item scenario.Scenario, runtime process.ScenarioRuntime) scenarioStatusItemOutput {
	return buildScenarioStatusDetail(orchestrator.Detail{
		Scenario: item,
		Runtime:  runtime,
		Details:  scenario.DescribeRuntime(item.Manifest, runtime),
	})
}

func buildScenarioStatusDetail(detail orchestrator.Detail) scenarioStatusItemOutput {
	health := any(nil)
	if detail.Details.Health != "" {
		health = detail.Details.Health
	}
	return scenarioStatusItemOutput{
		Name:        detail.Scenario.Slug,
		DisplayName: detail.Scenario.Manifest.Service.DisplayName,
		Description: detail.Scenario.Manifest.Service.Description,
		Tags:        copyStrings(detail.Scenario.Manifest.Service.Tags),
		Status:      detail.Details.Status,
		Processes:   detail.Details.Processes,
		Runtime:     detail.Details.Runtime,
		StartedAt:   detail.Details.StartedAt,
		Ports:       copyIntMap(detail.Details.Ports),
		Health:      health,
	}
}

func buildScenarioInfoData(item scenario.Scenario) scenarioInfoScenarioData {
	return scenarioInfoScenarioData{
		Name:             item.Slug,
		DisplayName:      item.Manifest.Service.DisplayName,
		Description:      item.Manifest.Service.Description,
		Version:          item.Manifest.Service.Version,
		Type:             item.Manifest.Service.Type,
		Category:         item.Manifest.Service.Category,
		Tags:             copyStrings(item.Manifest.Service.Tags),
		Path:             item.Path,
		ServicePath:      item.ServicePath,
		SandboxRedirect:  item.Redirected,
		ConfigVersion:    item.Manifest.Version,
		LifecycleVersion: item.Manifest.Lifecycle.Version,
		Ports:            item.Manifest.SortedPorts(),
		Phases:           item.Manifest.PhaseSummaries(),
	}
}

func buildScenarioRuntimeData(manifest scenario.ServiceManifest, runtime process.ScenarioRuntime) scenarioInfoRuntimeData {
	details := scenario.DescribeRuntime(manifest, runtime)
	return scenarioInfoRuntimeData{
		Status:      details.Status,
		Processes:   details.Processes,
		Runtime:     details.Runtime,
		StartedAt:   details.StartedAt,
		Ports:       copyIntMap(details.Ports),
		ProcessInfo: copyProcessRecords(details.ProcessInfo),
		ListPorts:   runtimePortOutputs(details.PortBindings),
	}
}

func runtimePortOutputs(bindings []scenario.RuntimePortBinding) []scenarioListPortOutput {
	listPorts := make([]scenarioListPortOutput, 0, len(bindings))
	for _, binding := range bindings {
		listPorts = append(listPorts, scenarioListPortOutput{
			Key:  binding.Key,
			Step: binding.Step,
			Port: binding.Port,
		})
	}
	return listPorts
}

// buildListPorts remains as a thin adapter because the CLI tests and week-4
// wrappers still assert the legacy output contract while the orchestration
// logic now lives in internal/scenario and internal/orchestrator.
func buildListPorts(manifest scenario.ServiceManifest, records []process.Record) ([]scenarioListPortOutput, map[string]int) {
	bindings, ports := scenario.RuntimePortBindings(manifest, records)
	return runtimePortOutputs(bindings), copyIntMap(ports)
}

func inferPortEnvVar(manifest scenario.ServiceManifest, step string) string {
	return scenario.InferPortEnvVar(manifest, step)
}

func copyIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return map[string]int{}
	}
	dst := make(map[string]int, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func parseScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name, jsonFlag, err := parseOptionalScenarioNameAndJSON(command, defaultJSON, args)
	if err != nil {
		return "", false, err
	}
	if name == "" {
		return "", false, fmt.Errorf("scenario %s requires a scenario name", command)
	}
	return name, jsonFlag, nil
}

func parseOptionalScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name := ""
	jsonFlag := defaultJSON
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonFlag = true
		case "--help", "-h":
			return "", false, fmt.Errorf("usage requested")
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("unknown option for scenario %s: %s", command, arg)
			}
			if name != "" {
				return "", false, fmt.Errorf("scenario %s accepts at most one scenario name", command)
			}
			name = arg
		}
	}
	return name, jsonFlag, nil
}

func parseScenarioStartArgs(defaultJSON bool, args []string) ([]string, lifecycle.StartOptions, bool, bool, error) {
	names := []string{}
	jsonFlag := defaultJSON
	openAfter := false
	opts := lifecycle.StartOptions{}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--json":
			jsonFlag = true
		case "--open":
			openAfter = true
		case "--best-effort":
			opts.BestEffort = true
		case "--clean-stale":
			opts.CleanStale = true
		case "--path":
			if index+1 >= len(args) {
				return nil, lifecycle.StartOptions{}, false, false, fmt.Errorf("scenario start --path requires a value")
			}
			index++
			opts.CustomPath = args[index]
		case "--help", "-h":
			return nil, lifecycle.StartOptions{}, false, false, fmt.Errorf("usage requested")
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, lifecycle.StartOptions{}, false, false, fmt.Errorf("unknown option for scenario start: %s", arg)
			}
			names = append(names, arg)
		}
	}
	return names, opts, jsonFlag, openAfter, nil
}

func parseScenarioSingleStartArgs(command string, defaultJSON bool, args []string) (string, lifecycle.StartOptions, bool, bool, error) {
	names, opts, jsonFlag, openAfter, err := parseScenarioStartArgs(defaultJSON, args)
	if err != nil {
		return "", lifecycle.StartOptions{}, false, false, err
	}
	if len(names) == 0 {
		return "", lifecycle.StartOptions{}, false, false, fmt.Errorf("scenario %s requires a scenario name", command)
	}
	if len(names) > 1 {
		return "", lifecycle.StartOptions{}, false, false, fmt.Errorf("scenario %s accepts exactly one scenario name", command)
	}
	return names[0], opts, jsonFlag, openAfter, nil
}

func showScenarioHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Vrooli Scenario Commands")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli scenario <subcommand> [options]")
	_, _ = fmt.Fprintln(w)
	renderCommandGroups(w, groupedScenarioCommands())
}

func writeScenarioInfoHuman(w io.Writer, info scenarioInfoScenarioData, runtime scenarioInfoRuntimeData) {
	_, _ = fmt.Fprintf(w, "Scenario: %s\n", info.Name)
	if info.DisplayName != "" {
		_, _ = fmt.Fprintf(w, "Display name: %s\n", info.DisplayName)
	}
	if info.Description != "" {
		_, _ = fmt.Fprintf(w, "Description: %s\n", info.Description)
	}
	if info.Version != "" {
		_, _ = fmt.Fprintf(w, "Version: %s\n", info.Version)
	}
	if info.Type != "" {
		_, _ = fmt.Fprintf(w, "Type: %s\n", info.Type)
	}
	if info.Category != "" {
		_, _ = fmt.Fprintf(w, "Category: %s\n", info.Category)
	}
	if len(info.Tags) > 0 {
		_, _ = fmt.Fprintf(w, "Tags: %s\n", strings.Join(info.Tags, ", "))
	}
	_, _ = fmt.Fprintf(w, "Path: %s\n", info.Path)
	if info.SandboxRedirect {
		_, _ = fmt.Fprintln(w, "Sandbox: using redirected scenario path")
	}
	if info.LifecycleVersion != "" {
		_, _ = fmt.Fprintf(w, "Lifecycle version: %s\n", info.LifecycleVersion)
	}
	_, _ = fmt.Fprintf(w, "Runtime status: %s\n", runtime.Status)
	if runtime.StartedAt != nil {
		_, _ = fmt.Fprintf(w, "Started at: %s\n", runtime.StartedAt.UTC().Format(time.RFC3339))
	}
	if len(runtime.Ports) > 0 {
		_, _ = fmt.Fprintf(w, "Ports: %s\n", formatPortMap(runtime.Ports))
	}

	if len(info.Ports) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Configured ports:")
		for _, port := range info.Ports {
			line := fmt.Sprintf("  %s (%s)", port.EnvVar, port.Name)
			if port.FixedPort != nil {
				line += fmt.Sprintf(" fixed=%d", *port.FixedPort)
			}
			if port.Range != "" {
				line += fmt.Sprintf(" range=%s", port.Range)
			}
			_, _ = fmt.Fprintln(w, line)
		}
	}
}

func writeScenarioStatusTable(w io.Writer, items []scenarioStatusItemOutput) {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		health := ""
		if item.Health != nil {
			health = fmt.Sprint(item.Health)
		}
		rows = append(rows, []string{
			item.Name,
			item.Status,
			health,
			fmt.Sprintf("%d", item.Processes),
			item.Runtime,
			formatPortMap(item.Ports),
		})
	}
	_ = cliout.RenderTable(w, []string{"Name", "Status", "Health", "Processes", "Runtime", "Ports"}, rows)
}

func writeScenarioStatusHuman(w io.Writer, output scenarioStatusSingleOutput) {
	info := output.Info
	runtime := output.Runtime
	status := output.Scenario

	_, _ = fmt.Fprintf(w, "Scenario: %s\n", info.Name)
	if info.DisplayName != "" {
		_, _ = fmt.Fprintf(w, "Display name: %s\n", info.DisplayName)
	}
	_, _ = fmt.Fprintf(w, "Status: %s\n", status.Status)
	if status.Health != nil {
		_, _ = fmt.Fprintf(w, "Health: %v\n", status.Health)
	}
	if info.Description != "" {
		_, _ = fmt.Fprintf(w, "Description: %s\n", info.Description)
	}
	_, _ = fmt.Fprintf(w, "Path: %s\n", info.Path)
	if runtime.StartedAt != nil {
		_, _ = fmt.Fprintf(w, "Started at: %s\n", runtime.StartedAt.UTC().Format(time.RFC3339))
	}
	_, _ = fmt.Fprintf(w, "Runtime: %s\n", runtime.Runtime)
	if len(runtime.Ports) > 0 {
		_, _ = fmt.Fprintf(w, "Ports: %s\n", formatPortMap(runtime.Ports))
	}
	if len(runtime.ProcessInfo) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Processes:")
		for _, record := range runtime.ProcessInfo {
			line := fmt.Sprintf("  %s pid=%d", record.Step, record.PID)
			if record.Port > 0 {
				line += fmt.Sprintf(" port=%d", record.Port)
			}
			if !record.StartedAt.IsZero() {
				line += fmt.Sprintf(" started=%s", record.StartedAt.UTC().Format(time.RFC3339))
			}
			_, _ = fmt.Fprintln(w, line)
		}
	}
}

func formatPortMap(ports map[string]int) string {
	if len(ports) == 0 {
		return ""
	}

	keys := make([]string, 0, len(ports))
	for key := range ports {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, ports[key]))
	}
	return strings.Join(parts, ", ")
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func copyProcessRecords(values []process.Record) []process.Record {
	if len(values) == 0 {
		return []process.Record{}
	}
	return append([]process.Record(nil), values...)
}
