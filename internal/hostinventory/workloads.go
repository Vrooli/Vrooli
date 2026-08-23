package hostinventory

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/workloadowner"
)

// CollectWorkloads serves ordinary callers from the shared cross-process
// facts reader. The emergency watchdog deliberately keeps its own longer
// cadence and cache so it never depends on stale pressure data.
func CollectWorkloads(ctx context.Context) (WorkloadSnapshot, error) {
	var snapshot WorkloadSnapshot
	if raw, err := sharedFactsReader().Read(ctx, "workloads"); err == nil && json.Unmarshal(raw, &snapshot) == nil {
		return snapshot, nil
	}
	return SystemCollector().CollectWorkloads(ctx)
}

// CollectWorkloads performs the deliberately opt-in workload census. It uses
// at most one command for each registry and records unavailable registries as
// unread rather than silently treating them as empty.
func (c Collector) CollectWorkloads(ctx context.Context) (WorkloadSnapshot, error) {
	c = c.withDefaults()
	out := WorkloadSnapshot{}
	if c.GOOS == "" {
		c.GOOS = "unknown"
	}
	if path, err := c.Commands.LookPath("docker"); err == nil {
		data, runErr := c.Commands.Run(ctx, path, "ps", "-a", "--format", "{{json .}}")
		if runErr != nil {
			out.Unread = append(out.Unread, "containers: docker query failed: "+runErr.Error())
		} else if parsed, parseErr := workloadowner.ParseDockerPS(data); parseErr != nil {
			return WorkloadSnapshot{}, parseErr
		} else {
			out.Containers = parsed
			out.Evidence = append(out.Evidence, "docker ps --format json")
			if len(parsed) > 0 {
				args := []string{"inspect", "--format", "{{.Name}}\t{{.RestartCount}}"}
				for _, container := range parsed {
					args = append(args, container.Name)
				}
				if inspect, inspectErr := c.Commands.Run(ctx, path, args...); inspectErr == nil {
					counts := workloadowner.ParseDockerInspectRestartCounts(inspect)
					for i := range out.Containers {
						out.Containers[i].RestartCount = counts[out.Containers[i].Name]
						out.Containers[i].Evidence = append(out.Containers[i].Evidence, "docker inspect restart count")
					}
					out.Evidence = append(out.Evidence, "docker inspect --format restart count")
				} else {
					out.Unread = append(out.Unread, "container restart counts: docker inspect failed: "+inspectErr.Error())
				}
			}
		}
	} else {
		out.Unread = append(out.Unread, "containers: docker is unavailable")
	}

	switch c.GOOS {
	case "linux":
		if path, err := c.Commands.LookPath("systemctl"); err == nil {
			// User units contain the Vrooli lifecycle and watchdog declarations;
			// system units are retained for whole-host posture reporting. They are
			// separate scopes because a system listing cannot answer whether a
			// user service is alive.
			for _, args := range [][]string{{"--user", "list-units", "--all", "--no-legend", "--no-pager"}, {"list-units", "--all", "--no-legend", "--no-pager"}} {
				data, runErr := c.Commands.Run(ctx, path, args...)
				if runErr != nil {
					out.Unread = append(out.Unread, "service units: systemctl scope query failed: "+runErr.Error())
					continue
				}
				out.ServiceUnits = append(out.ServiceUnits, workloadowner.ParseServiceUnits(data)...)
				out.Evidence = append(out.Evidence, "systemctl "+strings.Join(args, " "))
			}
		} else {
			out.Unread = append(out.Unread, "service units: systemctl is unavailable")
		}
		if path, err := c.Commands.LookPath("ss"); err == nil {
			data, runErr := c.Commands.Run(ctx, path, "-ltnup")
			if runErr != nil {
				out.Unread = append(out.Unread, "listeners: ss query failed: "+runErr.Error())
			} else {
				out.Listening = parseSS(data)
				out.Evidence = append(out.Evidence, "ss -ltnup")
			}
		} else {
			out.Unread = append(out.Unread, "listeners: ss is unavailable")
		}
	case "darwin":
		if path, err := c.Commands.LookPath("launchctl"); err == nil {
			data, runErr := c.Commands.Run(ctx, path, "list")
			if runErr != nil {
				out.Unread = append(out.Unread, "service units: launchctl query failed: "+runErr.Error())
			} else {
				out.ServiceUnits = append(out.ServiceUnits, parseLaunchctl(data)...)
				out.Evidence = append(out.Evidence, "launchctl list")
			}
		} else {
			out.Unread = append(out.Unread, "service units: launchctl is unavailable")
		}
		if path, err := c.Commands.LookPath("netstat"); err == nil {
			data, runErr := c.Commands.Run(ctx, path, "-anv", "-p", "tcp")
			if runErr != nil {
				out.Unread = append(out.Unread, "listeners: netstat query failed: "+runErr.Error())
			} else {
				out.Listening = parseNetstat(data, "darwin")
				out.Evidence = append(out.Evidence, "netstat -anv -p tcp")
			}
		} else {
			out.Unread = append(out.Unread, "listeners: netstat is unavailable")
		}
	case "windows":
		if path, err := c.Commands.LookPath("sc.exe"); err == nil {
			data, runErr := c.Commands.Run(ctx, path, "query", "type=", "service", "state=", "all")
			if runErr != nil {
				out.Unread = append(out.Unread, "service units: sc.exe query failed: "+runErr.Error())
			} else {
				out.ServiceUnits = append(out.ServiceUnits, parseSCQuery(data)...)
				out.Evidence = append(out.Evidence, "sc.exe query type= service state= all")
			}
		} else {
			out.Unread = append(out.Unread, "service units: sc.exe is unavailable")
		}
		if path, err := c.Commands.LookPath("schtasks.exe"); err == nil {
			data, runErr := c.Commands.Run(ctx, path, "/Query", "/FO", "CSV", "/NH")
			if runErr != nil {
				out.Unread = append(out.Unread, "scheduled tasks: schtasks query failed: "+runErr.Error())
			} else if tasks, parseErr := ParseWindowsTaskCSV(data); parseErr != nil {
				return WorkloadSnapshot{}, parseErr
			} else {
				out.ScheduledTasks = tasks
			}
		} else {
			out.Unread = append(out.Unread, "scheduled tasks: schtasks.exe is unavailable")
		}
		if path, err := c.Commands.LookPath("netstat.exe"); err == nil {
			data, runErr := c.Commands.Run(ctx, path, "-ano", "-p", "tcp")
			if runErr != nil {
				out.Unread = append(out.Unread, "listeners: netstat.exe query failed: "+runErr.Error())
			} else {
				out.Listening = parseNetstat(data, "windows")
				out.Evidence = append(out.Evidence, "netstat.exe -ano -p tcp")
			}
		} else {
			out.Unread = append(out.Unread, "listeners: netstat.exe is unavailable")
		}
	default:
		out.Unread = append(out.Unread, fmt.Sprintf("service units and listeners: unsupported platform %q", c.GOOS))
	}
	return out, nil
}

func parseLaunchctl(data []byte) []workloadowner.Workload {
	var out []workloadowner.Workload
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] == "PID" {
			continue
		}
		name := strings.TrimSpace(strings.Join(fields[2:], " "))
		if name == "" {
			continue
		}
		pid, _ := strconv.ParseInt(fields[0], 10, 64)
		out = append(out, workloadowner.Workload{Kind: "service-unit", Name: name, Running: pid > 0, Evidence: []string{"launchctl list"}})
	}
	return out
}

func parseSCQuery(data []byte) []workloadowner.Workload {
	var out []workloadowner.Workload
	var current *workloadowner.Workload
	flush := func() {
		if current != nil && current.Name != "" {
			out = append(out, *current)
		}
		current = nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SERVICE_NAME:") {
			flush()
			current = &workloadowner.Workload{Kind: "service-unit", Name: strings.TrimSpace(strings.TrimPrefix(trimmed, "SERVICE_NAME:")), Evidence: []string{"sc.exe query type= service state= all"}}
			continue
		}
		if current == nil || !strings.HasPrefix(trimmed, "STATE") {
			continue
		}
		parts := strings.Fields(trimmed)
		for _, part := range parts {
			if strings.EqualFold(part, "RUNNING") {
				current.Running = true
				break
			}
		}
	}
	flush()
	return out
}

func parseNetstat(data []byte, platform string) []ListeningPort {
	var out []ListeningPort
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		stateIndex := -1
		for i, field := range fields {
			if strings.EqualFold(field, "LISTEN") || strings.EqualFold(field, "LISTENING") {
				stateIndex = i
				break
			}
		}
		if stateIndex < 1 {
			continue
		}
		endpointIndex := stateIndex - 1
		if platform == "darwin" {
			endpointIndex = 3
		} else if platform == "windows" {
			endpointIndex = 1
		}
		if endpointIndex < 0 || endpointIndex >= len(fields) {
			continue
		}
		endpoint := fields[endpointIndex]
		address, port, ok := splitEndpoint(endpoint)
		if !ok && platform == "darwin" {
			idx := strings.LastIndex(endpoint, ".")
			if idx >= 0 {
				parsedPort, err := strconv.Atoi(endpoint[idx+1:])
				if err == nil && parsedPort >= 0 && parsedPort <= 65535 {
					address, port, ok = endpoint[:idx], parsedPort, true
				}
			}
		}
		if !ok {
			continue
		}
		process := ""
		if platform == "windows" && len(fields) > stateIndex+1 {
			process = fields[stateIndex+1]
		}
		out = append(out, ListeningPort{Protocol: "tcp", Address: address, Port: port, Process: process, Evidence: "native netstat"})
	}
	return out
}

func parseSS(data []byte) []ListeningPort {
	var out []ListeningPort
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || (fields[0] != "LISTEN" && fields[0] != "UNCONN") {
			continue
		}
		address, port, ok := splitEndpoint(fields[3])
		if !ok {
			continue
		}
		process := ""
		if len(fields) > 5 {
			process = strings.Join(fields[5:], " ")
		}
		out = append(out, ListeningPort{Protocol: "tcp", Address: address, Port: port, Process: process, Evidence: "ss -ltnup"})
	}
	return out
}

func splitEndpoint(value string) (string, int, bool) {
	value = strings.TrimSpace(value)
	idx := strings.LastIndex(value, ":")
	if idx < 0 {
		return "", 0, false
	}
	port, err := strconv.Atoi(strings.Trim(value[idx+1:], "[]"))
	if err != nil || port < 0 || port > 65535 {
		return "", 0, false
	}
	return strings.Trim(value[:idx], "[]"), port, true
}

// ParseWindowsTaskCSV is intentionally small but strict about the stable CSV
// surface emitted by `schtasks /Query /FO CSV /NH`.
func ParseWindowsTaskCSV(data []byte) ([]workloadowner.Workload, error) {
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		return nil, err
	}
	var out []workloadowner.Workload
	for _, row := range rows {
		if len(row) < 3 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		out = append(out, workloadowner.Workload{Kind: "scheduled-task", Name: strings.TrimSpace(row[0]), Running: strings.EqualFold(strings.TrimSpace(row[2]), "running"), Evidence: []string{"schtasks /Query /FO CSV /NH"}})
	}
	return out, nil
}
