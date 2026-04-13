package scenariocli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type ListPortOutput struct {
	Key  string `json:"key"`
	Step string `json:"step,omitempty"`
	Port int    `json:"port"`
}

type ListItemOutput struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Version     string           `json:"version,omitempty"`
	Status      string           `json:"status"`
	Tags        []string         `json:"tags"`
	Path        string           `json:"path"`
	Ports       []ListPortOutput `json:"ports"`
}

type StatusItemOutput struct {
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

type InfoOutput struct {
	Success  bool             `json:"success"`
	Scenario InfoScenarioData `json:"scenario"`
	Runtime  InfoRuntimeData  `json:"runtime"`
}

type InfoScenarioData struct {
	Name             string                       `json:"name"`
	DisplayName      string                       `json:"display_name,omitempty"`
	Description      string                       `json:"description,omitempty"`
	Version          string                       `json:"version,omitempty"`
	Type             string                       `json:"type,omitempty"`
	Category         string                       `json:"category,omitempty"`
	Tags             []string                     `json:"tags"`
	Path             string                       `json:"path"`
	ServicePath      string                       `json:"service_path"`
	SandboxRedirect  bool                         `json:"sandbox_redirected"`
	ConfigVersion    string                       `json:"config_version,omitempty"`
	LifecycleVersion string                       `json:"lifecycle_version,omitempty"`
	Ports            []scenariomodel.PortSummary  `json:"ports"`
	Phases           []scenariomodel.PhaseSummary `json:"phases"`
}

type InfoRuntimeData struct {
	Status      string           `json:"status"`
	Processes   int              `json:"processes"`
	Runtime     string           `json:"runtime"`
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	Ports       map[string]int   `json:"ports"`
	ProcessInfo []process.Record `json:"process_records"`
	ListPorts   []ListPortOutput `json:"list_ports"`
}

type StatusSingleOutput struct {
	Success  bool             `json:"success"`
	Scenario StatusItemOutput `json:"scenario"`
	Info     InfoScenarioData `json:"info"`
	Runtime  InfoRuntimeData  `json:"runtime"`
}

type LifecycleItemOutput struct {
	Name               string         `json:"name"`
	Status             string         `json:"status"`
	Health             string         `json:"health,omitempty"`
	Ports              map[string]int `json:"ports,omitempty"`
	FailedDependencies []string       `json:"failed_dependencies,omitempty"`
}

type BatchFailure struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

type BatchResponse struct {
	Verb    string
	Started []LifecycleItemOutput
	Stopped []string
	Failed  []BatchFailure
}

type ListResponse struct {
	Items        []ListItemOutput
	RunningCount int
}

type StatusResponse struct {
	Single *StatusSingleOutput
	List   []StatusItemOutput
}

type PortSingleOutput struct {
	Success  bool   `json:"success"`
	Scenario string `json:"scenario"`
	PortName string `json:"port_name"`
	Step     string `json:"step,omitempty"`
	Port     int    `json:"port,omitempty"`
	Error    string `json:"error,omitempty"`
}

type PortListOutput struct {
	Success  bool             `json:"success"`
	Scenario string           `json:"scenario"`
	Ports    []ListPortOutput `json:"ports"`
	Metadata map[string]int   `json:"metadata,omitempty"`
	Error    string           `json:"error,omitempty"`
}

type PortResponse struct {
	Single *PortSingleOutput
	List   *PortListOutput
}

type OpenOutput struct {
	Success  bool   `json:"success"`
	Scenario string `json:"scenario"`
	PortName string `json:"port_name"`
	Port     int    `json:"port"`
	URL      string `json:"url"`
}

func BuildStatusItem(item scenariomodel.Scenario, runtime process.ScenarioRuntime) StatusItemOutput {
	return BuildStatusDetail(orchestrator.Detail{
		Scenario: item,
		Runtime:  runtime,
		Details:  scenariomodel.DescribeRuntime(item.Manifest, runtime),
	})
}

func BuildStatusDetail(detail orchestrator.Detail) StatusItemOutput {
	health := any(nil)
	if detail.Details.Health != "" {
		health = detail.Details.Health
	}
	return StatusItemOutput{
		Name:        detail.Scenario.Slug,
		DisplayName: detail.Scenario.Manifest.Service.DisplayName,
		Description: detail.Scenario.Manifest.Service.Description,
		Tags:        CopyStrings(detail.Scenario.Manifest.Service.Tags),
		Status:      detail.Details.Status,
		Processes:   detail.Details.Processes,
		Runtime:     detail.Details.Runtime,
		StartedAt:   detail.Details.StartedAt,
		Ports:       CopyIntMap(detail.Details.Ports),
		Health:      health,
	}
}

func BuildInfoData(item scenariomodel.Scenario) InfoScenarioData {
	return InfoScenarioData{
		Name:             item.Slug,
		DisplayName:      item.Manifest.Service.DisplayName,
		Description:      item.Manifest.Service.Description,
		Version:          item.Manifest.Service.Version,
		Type:             item.Manifest.Service.Type,
		Category:         item.Manifest.Service.Category,
		Tags:             CopyStrings(item.Manifest.Service.Tags),
		Path:             item.Path,
		ServicePath:      item.ServicePath,
		SandboxRedirect:  item.Redirected,
		ConfigVersion:    item.Manifest.Version,
		LifecycleVersion: item.Manifest.Lifecycle.Version,
		Ports:            item.Manifest.SortedPorts(),
		Phases:           item.Manifest.PhaseSummaries(),
	}
}

func BuildRuntimeData(manifest scenariomodel.ServiceManifest, runtime process.ScenarioRuntime) InfoRuntimeData {
	details := scenariomodel.DescribeRuntime(manifest, runtime)
	return InfoRuntimeData{
		Status:      details.Status,
		Processes:   details.Processes,
		Runtime:     details.Runtime,
		StartedAt:   details.StartedAt,
		Ports:       CopyIntMap(details.Ports),
		ProcessInfo: CopyProcessRecords(details.ProcessInfo),
		ListPorts:   RuntimePortOutputs(details.PortBindings),
	}
}

func RuntimePortOutputs(bindings []scenariomodel.RuntimePortBinding) []ListPortOutput {
	listPorts := make([]ListPortOutput, 0, len(bindings))
	for _, binding := range bindings {
		listPorts = append(listPorts, ListPortOutput{
			Key:  binding.Key,
			Step: binding.Step,
			Port: binding.Port,
		})
	}
	return listPorts
}

func BuildListPorts(manifest scenariomodel.ServiceManifest, records []process.Record) ([]ListPortOutput, map[string]int) {
	bindings, ports := scenariomodel.RuntimePortBindings(manifest, records)
	return RuntimePortOutputs(bindings), CopyIntMap(ports)
}

func CopyIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return map[string]int{}
	}
	dup := make(map[string]int, len(src))
	for key, value := range src {
		dup[key] = value
	}
	return dup
}

func CopyStrings(values []string) []string {
	return append([]string(nil), values...)
}

func CopyProcessRecords(values []process.Record) []process.Record {
	return append([]process.Record(nil), values...)
}

func WriteLifecycleItems(w io.Writer, format cliout.Format, items []LifecycleItemOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"scenarios": items,
		})
	}

	for _, item := range items {
		switch item.Status {
		case "already_running":
			_, _ = fmt.Fprintf(w, "Scenario '%s' is already running", item.Name)
		case "stopped":
			_, _ = fmt.Fprintf(w, "Stopped scenario '%s'", item.Name)
		default:
			_, _ = fmt.Fprintf(w, "Started scenario '%s'", item.Name)
		}
		if item.Health != "" {
			_, _ = fmt.Fprintf(w, " (%s)", item.Health)
		}
		_, _ = fmt.Fprintln(w)
		if len(item.Ports) > 0 {
			_, _ = fmt.Fprintf(w, "  Ports: %s\n", FormatPortMap(item.Ports))
		}
		if len(item.FailedDependencies) > 0 {
			_, _ = fmt.Fprintf(w, "  Failed dependencies: %s\n", strings.Join(item.FailedDependencies, ", "))
		}
	}
	return nil
}

func WriteBatchReport(w io.Writer, format cliout.Format, resp BatchResponse) error {
	if format == cliout.FormatJSON {
		data := map[string]any{
			"failed": resp.Failed,
		}
		if len(resp.Started) > 0 {
			data["started"] = resp.Started
		}
		if len(resp.Stopped) > 0 {
			data["stopped"] = resp.Stopped
		}
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"data":    data,
		})
	}

	if len(resp.Started) == 0 && len(resp.Stopped) == 0 && len(resp.Failed) == 0 {
		_, _ = fmt.Fprintln(w, "No running scenarios found")
		return nil
	}

	if len(resp.Started) > 0 {
		_, _ = fmt.Fprintf(w, "%s %d scenarios\n", resp.Verb, len(resp.Started))
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "%s scenarios:\n", resp.Verb)
		for _, item := range resp.Started {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", item.Name, item.Status)
		}
	}

	if len(resp.Stopped) > 0 {
		_, _ = fmt.Fprintf(w, "%s %d scenarios\n", resp.Verb, len(resp.Stopped))
	}

	if len(resp.Failed) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "Failed to %s:\n", strings.ToLower(resp.Verb))
		for _, item := range resp.Failed {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", item.Name, item.Error)
		}
	}
	return nil
}

func RenderListResponse(w io.Writer, format cliout.Format, resp ListResponse) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"summary": map[string]int{
				"total_scenarios": len(resp.Items),
				"running":         resp.RunningCount,
				"available":       len(resp.Items) - resp.RunningCount,
			},
			"scenarios": resp.Items,
		})
	}

	_, _ = fmt.Fprintln(w, "[INFO]    Available scenarios:")
	for _, item := range resp.Items {
		line := "  • " + item.Name
		if item.Description != "" {
			line += " - " + item.Description
		}
		if len(item.Ports) > 0 {
			portParts := make([]string, 0, len(item.Ports))
			for _, port := range item.Ports {
				portParts = append(portParts, fmt.Sprintf("%s=%d", port.Key, port.Port))
			}
			line += " (ports: " + strings.Join(portParts, ", ") + ")"
		}
		_, _ = fmt.Fprintln(w, line)
	}
	return nil
}

func RenderInfoResponse(w io.Writer, format cliout.Format, resp InfoOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	WriteInfoHuman(w, resp.Scenario, resp.Runtime)
	return nil
}

func RenderStatusResponse(w io.Writer, format cliout.Format, resp StatusResponse) error {
	if resp.Single == nil {
		if format == cliout.FormatJSON {
			runningCount := 0
			for _, item := range resp.List {
				if item.Status == "running" {
					runningCount++
				}
			}
			return cliout.WriteJSON(w, map[string]any{
				"success": true,
				"summary": map[string]int{
					"total_scenarios": len(resp.List),
					"running":         runningCount,
					"stopped":         len(resp.List) - runningCount,
				},
				"scenarios": resp.List,
			})
		}
		WriteStatusTable(w, resp.List)
		return nil
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp.Single)
	}
	WriteStatusHuman(w, *resp.Single)
	return nil
}

func RenderSetupResponse(w io.Writer, format cliout.Format, result lifecycle.PhaseResult) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"phase":   "setup",
			"status":  result.Status,
			"defined": result.Defined,
			"steps": map[string]int{
				"executed": result.ExecutedSteps,
				"skipped":  result.SkippedSteps,
			},
		})
	}

	switch result.Status {
	case lifecycle.PhaseExecutionCompleted:
		_, _ = fmt.Fprintf(w, "Completed setup for scenario (%d executed, %d skipped)\n", result.ExecutedSteps, result.SkippedSteps)
	case lifecycle.PhaseExecutionSkipped:
		_, _ = fmt.Fprintf(w, "Setup phase ran no steps (%d skipped)\n", result.SkippedSteps)
	default:
		_, _ = fmt.Fprintln(w, "Scenario does not define a setup phase")
	}
	return nil
}

func RenderPortResponse(w io.Writer, format cliout.Format, resp PortResponse) error {
	if resp.List != nil {
		if format == cliout.FormatJSON {
			return cliout.WriteJSON(w, resp.List)
		}
		if !resp.List.Success {
			return fmt.Errorf(resp.List.Error)
		}
		for _, port := range resp.List.Ports {
			_, _ = fmt.Fprintf(w, "%s=%d\n", port.Key, port.Port)
		}
		return nil
	}
	if resp.Single == nil {
		return nil
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp.Single)
	}
	if !resp.Single.Success {
		return fmt.Errorf(resp.Single.Error)
	}
	_, _ = fmt.Fprintf(w, "%d\n", resp.Single.Port)
	return nil
}

func RenderOpenResponse(w io.Writer, resp OpenOutput) error {
	if resp.URL == "" {
		return nil
	}
	_, _ = fmt.Fprintln(w, resp.URL)
	return nil
}

func WriteInfoHuman(w io.Writer, info InfoScenarioData, runtime InfoRuntimeData) {
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
		_, _ = fmt.Fprintf(w, "Ports: %s\n", FormatPortMap(runtime.Ports))
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

func WriteStatusTable(w io.Writer, items []StatusItemOutput) {
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
			FormatPortMap(item.Ports),
		})
	}
	_ = cliout.RenderTable(w, []string{"Name", "Status", "Health", "Processes", "Runtime", "Ports"}, rows)
}

func WriteStatusHuman(w io.Writer, output StatusSingleOutput) {
	info := output.Info
	runtime := output.Runtime
	status := output.Scenario

	_, _ = fmt.Fprintf(w, "Scenario: %s\n", info.Name)
	if info.DisplayName != "" {
		_, _ = fmt.Fprintf(w, "Display name: %s\n", info.DisplayName)
	}
	_, _ = fmt.Fprintf(w, "Status: %s\n", status.Status)
	if output.Scenario.Health != nil {
		_, _ = fmt.Fprintf(w, "Health: %v\n", output.Scenario.Health)
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
		_, _ = fmt.Fprintf(w, "Ports: %s\n", FormatPortMap(runtime.Ports))
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

func FormatPortMap(ports map[string]int) string {
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
