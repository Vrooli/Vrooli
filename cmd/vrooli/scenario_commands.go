package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
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

func runScenarioCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		showScenarioHelp(stdout)
		return nil
	}

	switch args[0] {
	case "list":
		return runScenarioListCommand(root, globals, args[1:], stdout)
	case "info":
		return runScenarioInfoCommand(root, globals, args[1:], stdout)
	case "status":
		return runScenarioStatusCommand(root, globals, args[1:], stdout)
	default:
		return runBashScript(root, globals, "cli/commands/scenario/scenario-commands.sh", args...)
	}
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

	scenarios, runtimes, err := loadScenarioState(root)
	if err != nil {
		return err
	}

	items := make([]scenarioListItemOutput, 0, len(scenarios))
	runningCount := 0
	for _, item := range scenarios {
		runtime, running := runtimes[item.Slug]
		status := "available"
		if running {
			status = "running"
			runningCount++
		}

		listPorts := []scenarioListPortOutput{}
		if includePorts && running {
			listPorts, _ = buildListPorts(item.Manifest, runtime.Records)
		}

		items = append(items, scenarioListItemOutput{
			Name:        item.Slug,
			Description: item.Manifest.Service.Description,
			Version:     item.Manifest.Service.Version,
			Status:      status,
			Tags:        copyStrings(item.Manifest.Service.Tags),
			Path:        item.Path + string(os.PathSeparator),
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

	scenarioItem, runtime, _, err := loadScenarioDetail(root, name)
	if err != nil {
		return err
	}

	output := scenarioInfoOutput{
		Success:  true,
		Scenario: buildScenarioInfoData(scenarioItem),
		Runtime:  buildScenarioRuntimeData(scenarioItem.Manifest, runtime),
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
		scenarios, runtimes, err := loadScenarioState(root)
		if err != nil {
			return err
		}

		items := make([]scenarioStatusItemOutput, 0, len(scenarios))
		runningCount := 0
		for _, item := range scenarios {
			runtime := runtimes[item.Slug]
			statusItem := buildScenarioStatusItem(item, runtime)
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

	scenarioItem, runtime, health, err := loadScenarioDetail(root, name)
	if err != nil {
		return err
	}

	output := scenarioStatusSingleOutput{
		Success:  true,
		Scenario: buildScenarioStatusItemWithHealth(scenarioItem, runtime, health),
		Info:     buildScenarioInfoData(scenarioItem),
		Runtime:  buildScenarioRuntimeData(scenarioItem.Manifest, runtime),
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, output)
	}

	writeScenarioStatusHuman(stdout, output)
	return nil
}

func loadScenarioState(root string) ([]scenario.Scenario, map[string]process.ScenarioRuntime, error) {
	home, err := process.HomeDir()
	if err != nil {
		return nil, nil, err
	}

	scenarios, err := scenario.Discover(root, scenario.SandboxEnvFromEnv())
	if err != nil {
		return nil, nil, err
	}

	valid := make(map[string]struct{}, len(scenarios))
	for _, item := range scenarios {
		valid[item.Slug] = struct{}{}
	}

	running, err := process.DiscoverRunningScenarios(home, func(name string) bool {
		_, ok := valid[name]
		return ok
	})
	if err != nil {
		return nil, nil, err
	}

	runtimes := make(map[string]process.ScenarioRuntime, len(running))
	for _, runtime := range running {
		runtimes[runtime.Name] = runtime
	}

	return scenarios, runtimes, nil
}

func loadScenarioDetail(root, name string) (scenario.Scenario, process.ScenarioRuntime, string, error) {
	item, err := scenario.Load(root, name, scenario.SandboxEnvFromEnv())
	if err != nil {
		if err == scenario.ErrNotFound {
			return scenario.Scenario{}, process.ScenarioRuntime{}, "", fmt.Errorf("scenario %q not found", name)
		}
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}

	home, err := process.HomeDir()
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}
	records, err := process.ReadScenarioRecords(home, name)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}
	runtime := process.SummarizeScenario(name, records)

	health := ""
	if runtime.ProcessCount > 0 {
		_, ports := buildListPorts(item.Manifest, runtime.Records)
		health = scenario.EvaluateHealth(item.Manifest.HealthConfig(), ports)
	}

	return item, runtime, health, nil
}

func buildScenarioStatusItem(item scenario.Scenario, runtime process.ScenarioRuntime) scenarioStatusItemOutput {
	health := any(nil)
	if runtime.ProcessCount > 0 {
		_, ports := buildListPorts(item.Manifest, runtime.Records)
		health = scenario.EvaluateHealth(item.Manifest.HealthConfig(), ports)
	}
	return buildScenarioStatusItemWithHealth(item, runtime, health)
}

func buildScenarioStatusItemWithHealth(item scenario.Scenario, runtime process.ScenarioRuntime, health any) scenarioStatusItemOutput {
	status := "stopped"
	if runtime.ProcessCount > 0 {
		status = "running"
	}
	_, ports := buildListPorts(item.Manifest, runtime.Records)
	return scenarioStatusItemOutput{
		Name:        item.Slug,
		DisplayName: item.Manifest.Service.DisplayName,
		Description: item.Manifest.Service.Description,
		Tags:        copyStrings(item.Manifest.Service.Tags),
		Status:      status,
		Processes:   runtime.ProcessCount,
		Runtime:     runtime.Runtime,
		StartedAt:   runtime.StartedAt,
		Ports:       ports,
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
	listPorts, ports := buildListPorts(manifest, runtime.Records)
	status := "stopped"
	if runtime.ProcessCount > 0 {
		status = "running"
	}
	return scenarioInfoRuntimeData{
		Status:      status,
		Processes:   runtime.ProcessCount,
		Runtime:     runtime.Runtime,
		StartedAt:   runtime.StartedAt,
		Ports:       ports,
		ProcessInfo: copyProcessRecords(runtime.Records),
		ListPorts:   listPorts,
	}
}

// buildListPorts prefers explicit process-record ports and only falls back to
// live process environment inspection for manifest keys that were not captured
// in the runtime JSON metadata.
func buildListPorts(manifest scenario.ServiceManifest, records []process.Record) ([]scenarioListPortOutput, map[string]int) {
	ports := make(map[string]int)
	listPorts := make([]scenarioListPortOutput, 0, len(records))
	seen := make(map[string]struct{})

	for _, record := range records {
		if record.Port <= 0 {
			continue
		}

		key := inferPortEnvVar(manifest, record.Step)
		if key == "" {
			continue
		}

		ports[key] = record.Port
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		listPorts = append(listPorts, scenarioListPortOutput{
			Key:  key,
			Step: record.Step,
			Port: record.Port,
		})
	}

	envPorts := process.ReadEnvironmentPorts(records, manifest.PortEnvVars())
	for key, port := range envPorts {
		if _, exists := ports[key]; !exists {
			ports[key] = port
		}
	}

	sort.Slice(listPorts, func(i, j int) bool {
		if listPorts[i].Key == listPorts[j].Key {
			return listPorts[i].Step < listPorts[j].Step
		}
		return listPorts[i].Key < listPorts[j].Key
	})

	return listPorts, ports
}

// inferPortEnvVar maps lifecycle step names like "start-api" or "launch-ui"
// back to manifest port keys. The process metadata is historical and not fully
// normalized, so the mapping intentionally uses a few lightweight heuristics.
func inferPortEnvVar(manifest scenario.ServiceManifest, step string) string {
	step = strings.ToLower(strings.TrimSpace(step))
	step = strings.TrimPrefix(step, "start-")
	step = strings.TrimPrefix(step, "run-")
	step = strings.TrimPrefix(step, "serve-")
	step = strings.TrimPrefix(step, "launch-")

	if step != "" {
		if envVar := manifest.PortEnvVar(step); envVar != "" {
			return envVar
		}
	}

	for _, definition := range manifest.SortedPorts() {
		name := strings.ToLower(definition.Name)
		if step == name || strings.Contains(step, name) || strings.Contains(name, step) {
			return definition.EnvVar
		}
		normalizedEnv := strings.TrimSuffix(strings.ToLower(definition.EnvVar), "_port")
		if normalizedEnv != "" && (step == normalizedEnv || strings.Contains(step, normalizedEnv)) {
			return definition.EnvVar
		}
	}

	return ""
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

func showScenarioHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Vrooli Scenario Commands")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli scenario <subcommand> [options]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Read-only commands:")
	_, _ = fmt.Fprintln(w, "  list [--json] [--include-ports]   List discovered scenarios")
	_, _ = fmt.Fprintln(w, "  info <name> [--json]              Show scenario metadata and runtime summary")
	_, _ = fmt.Fprintln(w, "  status [name] [--json]            Show scenario runtime status")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Lifecycle and utility commands:")
	_, _ = fmt.Fprintln(w, "  start <name> [options]            Start a scenario")
	_, _ = fmt.Fprintln(w, "  start-all                         Start all available scenarios")
	_, _ = fmt.Fprintln(w, "  setup <name>                      Run the setup lifecycle")
	_, _ = fmt.Fprintln(w, "  restart <name> [options]          Restart a scenario")
	_, _ = fmt.Fprintln(w, "  stop <name>                       Stop a running scenario")
	_, _ = fmt.Fprintln(w, "  stop-all                          Stop all running scenarios")
	_, _ = fmt.Fprintln(w, "  test <name> [phase|all|e2e]       Run scenario tests")
	_, _ = fmt.Fprintln(w, "  logs <name> [options]             View logs for a scenario")
	_, _ = fmt.Fprintln(w, "  open <name> [options]             Open scenario in the browser")
	_, _ = fmt.Fprintln(w, "  port <name> [port]                Show running port assignments")
	_, _ = fmt.Fprintln(w, "  ui-smoke <name> [--json]          Run the Browserless UI smoke harness")
	_, _ = fmt.Fprintln(w, "  requirements <subcommand>         Manage scenario requirements")
	_, _ = fmt.Fprintln(w, "  template [cmd]                    Manage scenario templates")
	_, _ = fmt.Fprintln(w, "  generate <template>               Scaffold a scenario from a template")
	_, _ = fmt.Fprintln(w, "  completeness <name>               Calculate completeness score")
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
