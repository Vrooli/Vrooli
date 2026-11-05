package scenariocli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	scenariocliRestarted = "restarted"
)

// isQuietOutput consults VROOLI_OUTPUT (set once by the root CLI runner
// based on --quiet / --verbose / --json) so render helpers that can't see
// the GlobalOptions struct can still collapse their output. Keeping the
// env var as the single source of truth avoids a scenariocli → rootcli
// import cycle.
// lifecycleItemsExitCode returns the worst verdict exit code across the
// items: degraded → 2, everything else (healthy/running/empty) → 0. Failures
// never reach the renderer (they surface as command errors → exit 1).
func lifecycleItemsExitCode(items []LifecycleItemOutput) int {
	code := 0
	for _, item := range items {
		if item.Verdict == scenarioruntime.HealthStatusDegraded && code < 2 {
			code = 2
		}
	}
	return code
}

func isQuietOutput() bool {
	return strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_OUTPUT"))) == "quiet"
}

type (
	ListPortOutput      = scenarioapp.ListPortOutput
	ListItemOutput      = scenarioapp.ListItemOutput
	StatusItemOutput    = scenarioapp.StatusItemOutput
	InfoOutput          = scenarioapp.InfoOutput
	InfoScenarioData    = scenarioapp.InfoScenarioData
	InfoRuntimeData     = scenarioapp.InfoRuntimeData
	StatusSingleOutput  = scenarioapp.StatusSingleOutput
	LifecycleItemOutput = scenarioapp.LifecycleItemOutput
	EndpointOutput      = scenarioapp.EndpointOutput
	BatchFailure        = scenarioapp.BatchFailure
	BatchResponse       = scenarioapp.BatchResponse
	ListResponse        = scenarioapp.ListResponse
	StatusResponse      = scenarioapp.StatusResponse
	PortSingleOutput    = scenarioapp.PortSingleOutput
	PortListOutput      = scenarioapp.PortListOutput
	PortResponse        = scenarioapp.PortResponse
	OpenOutput          = scenarioapp.OpenOutput
)

func WriteLifecycleItems(w io.Writer, format cliout.Format, items []LifecycleItemOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		if err := cliout.WriteProtoJSON(w, ScenarioLifecycleResponse(items)); err != nil {
			return err
		}
		if code := lifecycleItemsExitCode(items); code != 0 {
			return VerdictExitError{Code: code}
		}
		return nil
	}, func(w io.Writer) error {
		if isQuietOutput() {
			return writeLifecycleItemsCompact(w, items)
		}

		for _, item := range items {
			// Leading blank line visually separates the summary block from
			// the preceding progress pings and slog stream, which otherwise
			// run flush against the first "Started …"/"Restarted …" line.
			_, _ = fmt.Fprintln(w)
			switch item.Status {
			case "already_running":
				_, _ = fmt.Fprintf(w, "Scenario '%s' is already running", item.Name)
			case scenariocliRestarted:
				_, _ = fmt.Fprintf(w, "Restarted scenario '%s'", item.Name)
			case scenarioruntime.StatusStopped:
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
			if len(item.Endpoints) > 0 {
				_, _ = fmt.Fprintln(w, "  URLs:")
				for _, endpoint := range item.Endpoints {
					_, _ = fmt.Fprintf(w, "    %s: %s\n", endpoint.Key, endpoint.URL)
				}
			}
			if len(item.FailedDependencies) > 0 {
				_, _ = fmt.Fprintf(w, "  Failed dependencies: %s\n", strings.Join(item.FailedDependencies, ", "))
			}
			if len(item.FailedResources) > 0 {
				_, _ = fmt.Fprintf(w, "  Failed resources: %s\n", strings.Join(item.FailedResources, ", "))
			}
			// Log pointer: give the user a ready-to-run command for the
			// scenario-wide lifecycle log. Stopped scenarios get it too —
			// the log still exists and often explains why something was
			// stopped unexpectedly.
			if strings.TrimSpace(item.Name) != "" {
				_, _ = fmt.Fprintf(w, "  Logs: vrooli scenario logs %s\n", item.Name)
			}
		}
		return nil
	})
}

// writeLifecycleItemsCompact renders one line per scenario, inlining ports
// and any failure lists so that `vrooli scenario restart` at quiet mode
// stays under a handful of total lines. Failures still surface — they're
// appended to the same line with an explicit marker rather than multi-line
// blocks. JSON output is handled before this function is called.
func writeLifecycleItemsCompact(w io.Writer, items []LifecycleItemOutput) error {
	for _, item := range items {
		var verb string
		switch item.Status {
		case "already_running":
			verb = "already running"
		case scenariocliRestarted:
			verb = scenariocliRestarted
		case scenarioruntime.StatusStopped:
			verb = scenarioruntime.StatusStopped
		default:
			verb = "started"
		}
		health := ""
		if item.Health != "" {
			health = ", " + item.Health
		}
		_, _ = fmt.Fprintf(w, "%s %s (%s%s)", statusGlyph(item), item.Name, verb, health)
		if len(item.Ports) > 0 {
			_, _ = fmt.Fprintf(w, " | %s", FormatPortMap(item.Ports))
		}
		if len(item.FailedDependencies) > 0 {
			_, _ = fmt.Fprintf(w, " | failed deps: %s", strings.Join(item.FailedDependencies, ","))
		}
		if len(item.FailedResources) > 0 {
			_, _ = fmt.Fprintf(w, " | failed resources: %s", strings.Join(item.FailedResources, ","))
		}
		_, _ = fmt.Fprintln(w)
	}
	return nil
}

// statusGlyph picks a short ASCII marker for the compact form. The full
// unicode checkmark would be nicer but would break under NO_COLOR / older
// terminals and is not worth the branching cost for a one-line summary.
func statusGlyph(item LifecycleItemOutput) string {
	switch {
	case len(item.FailedDependencies) > 0 || len(item.FailedResources) > 0:
		return "!!"
	case item.Health == "" || item.Health == scenarioruntime.HealthStatusHealthy:
		return "OK"
	default:
		return "??"
	}
}

func WriteBatchReport(w io.Writer, format cliout.Format, resp BatchResponse) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ScenarioBatchResponse(resp)) }, func(w io.Writer) error {
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
	})
}

func RenderListResponse(w io.Writer, format cliout.Format, resp ListResponse) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		return cliout.WriteProtoJSON(w, ScenarioListResponse(resp.Items, resp.RunningCount, resp.Failures))
	}, func(w io.Writer) error {
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
		if len(resp.Failures) > 0 {
			_, _ = fmt.Fprintf(w, "\n[WARN]    Skipped %d scenarios with discovery errors\n", len(resp.Failures))
			for _, failure := range resp.Failures {
				_, _ = fmt.Fprintf(w, "  - %s: %s\n", failure.Name, failure.Error)
			}
		}
		return nil
	})
}

func RenderInfoResponse(w io.Writer, format cliout.Format, resp InfoOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ScenarioInfoResponse(resp)) }, func(w io.Writer) error { WriteInfoHuman(w, resp.Scenario, resp.Runtime); return nil })
}

func RenderStatusResponse(w io.Writer, format cliout.Format, resp StatusResponse) error {
	if len(resp.Raw) > 0 {
		_, err := w.Write(resp.Raw)
		return err
	}
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		if resp.Single == nil {
			return cliout.WriteProtoJSON(w, ScenarioStatusListResponse(resp.List, resp.Failures))
		}
		return cliout.WriteProtoJSON(w, ScenarioStatusSingleResponse(*resp.Single))
	}, func(w io.Writer) error {
		if resp.Single == nil {
			WriteStatusTable(w, resp.List)
			if len(resp.Failures) > 0 {
				_, _ = fmt.Fprintf(w, "\nSkipped %d scenarios with discovery errors:\n", len(resp.Failures))
				for _, failure := range resp.Failures {
					_, _ = fmt.Fprintf(w, "  %s: %s\n", failure.Name, failure.Error)
				}
			}
			return nil
		}
		WriteStatusHuman(w, *resp.Single)
		return nil
	})
}

func RenderSetupResponse(w io.Writer, format cliout.Format, result lifecycle.PhaseResult) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ScenarioSetupResponse(result)) }, func(w io.Writer) error {
		switch result.Status {
		case lifecycle.PhaseExecutionCompleted:
			_, _ = fmt.Fprintf(w, "Completed setup for scenario (%d executed, %d skipped)\n", result.ExecutedSteps, result.SkippedSteps)
		case lifecycle.PhaseExecutionSkipped:
			_, _ = fmt.Fprintf(w, "Setup phase ran no steps (%d skipped)\n", result.SkippedSteps)
		default:
			_, _ = fmt.Fprintln(w, "Scenario does not define a setup phase")
		}
		return nil
	})
}

func RenderPortResponse(w io.Writer, format cliout.Format, resp PortResponse) error {
	if resp.List != nil {
		return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ScenarioPortListResponse(*resp.List)) }, func(w io.Writer) error {
			if !resp.List.Success {
				return fmt.Errorf("%s", resp.List.Error)
			}
			for _, port := range resp.List.Ports {
				_, _ = fmt.Fprintf(w, "%s=%d\n", port.Key, port.Port)
			}
			return nil
		})
	}
	if resp.Single == nil {
		return nil
	}
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ScenarioPortSingleResponse(*resp.Single)) }, func(w io.Writer) error {
		if !resp.Single.Success {
			return fmt.Errorf("%s", resp.Single.Error)
		}
		_, _ = fmt.Fprintf(w, "%d\n", resp.Single.Port)
		return nil
	})
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
	if info.Generation != nil && strings.TrimSpace(info.Generation.Template.ID) != "" {
		line := fmt.Sprintf("Template: %s", info.Generation.Template.ID)
		if v := strings.TrimSpace(info.Generation.Template.Version); v != "" {
			line += fmt.Sprintf(" (%s)", v)
		}
		_, _ = fmt.Fprintln(w, line)
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
	// An in-flight lifecycle operation is reported next to the runtime status
	// it contradicts: mid-start a scenario reads scenariocliStopped, and without this
	// line the operator's next move is a restart the scenario lock refuses.
	if op := status.StartOperation; op != nil {
		if summary := op.InFlightSummary(); summary != "" {
			_, _ = fmt.Fprintf(w, "Lifecycle: %s\n", summary)
		}
	}
	if output.Scenario.Health != nil {
		_, _ = fmt.Fprintf(w, "Health: %v\n", output.Scenario.Health)
	}
	if output.Scenario.HealthError != "" {
		_, _ = fmt.Fprintf(w, "Health error: %s\n", output.Scenario.HealthError)
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
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, ports[key]))
	}
	return strings.Join(parts, ", ")
}
