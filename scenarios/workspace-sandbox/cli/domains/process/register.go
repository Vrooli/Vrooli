package process

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"

	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "process",
		Description: "Execute commands, inspect processes, and manage interactive sessions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "exec", Description: "Execute a command in a sandbox", Run: func(args []string) error { return runExec(deps, args) }},
			{Name: "run", Description: "Start a background process", Run: func(args []string) error { return runRun(deps, args) }},
			{Name: "list", Description: "List processes in a sandbox", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "kill", Description: "Kill one or all processes", Run: func(args []string) error { return runKill(deps, args) }},
			{Name: "logs", Description: "Show process logs", Run: func(args []string) error { return runLogs(deps, args) }},
			{Name: "shell", Description: "Open an interactive shell", Run: func(args []string) error { return runShell(deps, args) }},
			{Name: "attach", Description: "Run an interactive command", Run: func(args []string) error { return runAttach(deps, args) }},
		},
	}
}

func runExec(deps support.Dependencies, args []string) error {
	sandboxID, command, cmdArgs, opts, err := parseProcessCommandArgs(args, true)
	if err != nil {
		return err
	}

	resolvedID, err := support.ResolveSandboxID(deps.ScenarioApp(), sandboxID)
	if err != nil {
		return err
	}

	reqBody := map[string]any{"command": command}
	if len(cmdArgs) > 0 {
		reqBody["args"] = cmdArgs
	}
	applyExecOptions(reqBody, opts)

	body, err := deps.ScenarioApp().Request("POST", "/sandboxes/"+resolvedID+"/exec", nil, reqBody)
	if err != nil {
		return err
	}
	if opts.jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp support.ExecResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if resp.Stdout != "" {
		fmt.Print(resp.Stdout)
	}
	if resp.Stderr != "" {
		fmt.Fprint(os.Stderr, resp.Stderr)
	}
	if resp.TimedOut {
		fmt.Fprintf(os.Stderr, "\n[Process timed out with exit code %d]\n", resp.ExitCode)
	}
	if resp.ExitCode != 0 {
		return fmt.Errorf("exit code %d", resp.ExitCode)
	}
	return nil
}

func runRun(deps support.Dependencies, args []string) error {
	sandboxID, command, cmdArgs, opts, err := parseProcessCommandArgs(args, false)
	if err != nil {
		return err
	}

	resolvedID, err := support.ResolveSandboxID(deps.ScenarioApp(), sandboxID)
	if err != nil {
		return err
	}

	reqBody := map[string]any{"command": command}
	if len(cmdArgs) > 0 {
		reqBody["args"] = cmdArgs
	}
	if opts.name != "" {
		reqBody["name"] = opts.name
	}
	applyExecOptions(reqBody, opts)

	body, err := deps.ScenarioApp().Request("POST", "/sandboxes/"+resolvedID+"/start-process", nil, reqBody)
	if err != nil {
		return err
	}

	var resp support.RunResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Process started: PID %d", resp.PID),
			"Sandbox ID: " + resp.SandboxID,
		},
		Changes: []string{
			"Command: " + resp.Command,
		},
		NextCommand: []string{
			support.CLIName + " process logs " + resp.SandboxID + " --pid=" + fmt.Sprintf("%d", resp.PID),
			support.CLIName + " process list " + resp.SandboxID,
		},
	}
	if resp.Name != "" {
		report.Changes = append(report.Changes, "Name: "+resp.Name)
	}

	if opts.jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("process list", flag.ContinueOnError)
	runningOnly := fs.Bool("running", false, "Only show running processes")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: %s process list <sandbox-id> [--running] [--json]", support.CLIName)
	}

	sandboxID, err := support.ResolveSandboxID(deps.ScenarioApp(), fs.Arg(0))
	if err != nil {
		return err
	}

	query := url.Values{}
	if *runningOnly {
		query.Set("running", "true")
	}
	body, err := deps.ScenarioApp().Get("/sandboxes/"+sandboxID+"/processes", query)
	if err != nil {
		return err
	}

	var resp support.ProcessListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Sandbox ID: " + sandboxID,
			fmt.Sprintf("Processes: %d", resp.Total),
			fmt.Sprintf("Running: %d", resp.Running),
		},
		Results:        renderProcessRows(resp.Processes),
		RetrievalHints: []string{support.CLIName + " process logs " + sandboxID + " --pid=<pid>", support.CLIName + " process kill " + sandboxID + " --pid=<pid>"},
	}
	if *runningOnly {
		report.Summary = append(report.Summary, "Filter: running only")
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runKill(deps support.Dependencies, args []string) error {
	var sandboxID string
	var pid int
	var killAll, jsonOut bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--pid="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--pid="), "%d", &pid)
		case arg == "--all":
			killAll = true
		case arg == "--json":
			jsonOut = true
		case !strings.HasPrefix(arg, "-") && sandboxID == "":
			sandboxID = arg
		}
	}
	if sandboxID == "" {
		return fmt.Errorf("usage: %s process kill <sandbox-id> (--pid=<pid> | --all) [--json]", support.CLIName)
	}
	if pid == 0 && !killAll {
		return fmt.Errorf("must specify --pid=<pid> or --all")
	}

	resolvedID, err := support.ResolveSandboxID(deps.ScenarioApp(), sandboxID)
	if err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Process termination complete", "Sandbox ID: " + resolvedID},
		Changes:     []string{},
		NextCommand: []string{support.CLIName + " process list " + resolvedID},
	}

	if killAll {
		body, err := deps.ScenarioApp().Request("DELETE", "/sandboxes/"+resolvedID+"/processes", nil, nil)
		if err != nil {
			return err
		}
		var resp struct {
			Killed int      `json:"killed"`
			Errors []string `json:"errors,omitempty"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		report.Changes = append(report.Changes, fmt.Sprintf("Killed processes: %d", resp.Killed))
		for _, item := range resp.Errors {
			report.Changes = append(report.Changes, "Error: "+item)
		}
	} else {
		if _, err := deps.ScenarioApp().Request("DELETE", fmt.Sprintf("/sandboxes/%s/processes/%d", resolvedID, pid), nil, nil); err != nil {
			return err
		}
		report.Changes = append(report.Changes, fmt.Sprintf("Killed PID: %d", pid))
	}

	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runLogs(deps support.Dependencies, args []string) error {
	var sandboxID string
	var pid, tail int
	var follow, jsonOut bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--pid="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--pid="), "%d", &pid)
		case strings.HasPrefix(arg, "--tail="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--tail="), "%d", &tail)
		case arg == "--follow" || arg == "-f":
			follow = true
		case arg == "--json":
			jsonOut = true
		case !strings.HasPrefix(arg, "-") && sandboxID == "":
			sandboxID = arg
		}
	}
	if sandboxID == "" {
		return fmt.Errorf("usage: %s process logs <sandbox-id> [--pid=<pid>] [--follow] [--tail=N] [--json]", support.CLIName)
	}

	resolvedID, err := support.ResolveSandboxID(deps.ScenarioApp(), sandboxID)
	if err != nil {
		return err
	}

	if pid == 0 {
		body, err := deps.ScenarioApp().Get("/sandboxes/"+resolvedID+"/logs", nil)
		if err != nil {
			return err
		}
		var resp support.LogsListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		report := cliapp.ListReport{
			Summary:        []string{"Sandbox ID: " + resolvedID, fmt.Sprintf("Logs available: %d", resp.Total)},
			Results:        renderLogRows(resp.Logs),
			RetrievalHints: []string{support.CLIName + " process logs " + resolvedID + " --pid=<pid>", support.CLIName + " process logs " + resolvedID + " --pid=<pid> --follow"},
		}
		if jsonOut {
			return cliapp.PrintReportJSON(os.Stdout, report)
		}
		return cliapp.RenderListReport(os.Stdout, report)
	}

	if follow {
		return streamLogs(deps, resolvedID, pid)
	}

	query := url.Values{}
	if tail > 0 {
		query.Set("tail", fmt.Sprintf("%d", tail))
	}
	body, err := deps.ScenarioApp().Get(fmt.Sprintf("/sandboxes/%s/processes/%d/logs", resolvedID, pid), query)
	if err != nil {
		return err
	}
	if jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp support.LogResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	fmt.Print(resp.Content)
	return nil
}

func runShell(deps support.Dependencies, args []string) error {
	var sandboxID string
	var vrooliAware, network bool
	var memoryMB int

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--memory="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--memory="), "%d", &memoryMB)
		case arg == "--vrooli-aware":
			vrooliAware = true
		case arg == "--network":
			network = true
		case !strings.HasPrefix(arg, "-") && sandboxID == "":
			sandboxID = arg
		}
	}
	if sandboxID == "" {
		return fmt.Errorf("usage: %s process shell <sandbox-id> [--vrooli-aware] [--network] [--memory=MB]", support.CLIName)
	}

	resolvedID, err := support.ResolveSandboxID(deps.ScenarioApp(), sandboxID)
	if err != nil {
		return err
	}

	shellPath := "/bin/sh"
	if envShell := os.Getenv("SHELL"); envShell != "" {
		shellPath = envShell
	}
	return runInteractiveSession(deps, resolvedID, shellPath, nil, vrooliAware, network, memoryMB)
}

func runAttach(deps support.Dependencies, args []string) error {
	var sandboxID string
	var vrooliAware, network bool
	var memoryMB int

	cmdIdx := -1
	for i, arg := range args {
		if arg == "--" {
			cmdIdx = i
			break
		}
	}

	flagArgs := args
	if cmdIdx >= 0 {
		flagArgs = args[:cmdIdx]
	}
	for _, arg := range flagArgs {
		switch {
		case strings.HasPrefix(arg, "--memory="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--memory="), "%d", &memoryMB)
		case arg == "--vrooli-aware":
			vrooliAware = true
		case arg == "--network":
			network = true
		case !strings.HasPrefix(arg, "-") && sandboxID == "":
			sandboxID = arg
		}
	}

	var command string
	var cmdArgs []string
	if cmdIdx >= 0 && cmdIdx+1 < len(args) {
		command = args[cmdIdx+1]
		if cmdIdx+2 < len(args) {
			cmdArgs = args[cmdIdx+2:]
		}
	}
	if sandboxID == "" || command == "" {
		return fmt.Errorf("usage: %s process attach <sandbox-id> [--vrooli-aware] [--network] [--memory=MB] -- <command> [args...]", support.CLIName)
	}

	resolvedID, err := support.ResolveSandboxID(deps.ScenarioApp(), sandboxID)
	if err != nil {
		return err
	}
	return runInteractiveSession(deps, resolvedID, command, cmdArgs, vrooliAware, network, memoryMB)
}

type processCommandOptions struct {
	memoryMB    int
	cpuTime     int
	timeout     int
	maxProcs    int
	maxFiles    int
	vrooliAware bool
	network     bool
	jsonOut     bool
	workDir     string
	name        string
	envVars     map[string]string
}

func parseProcessCommandArgs(args []string, includeTimeout bool) (string, string, []string, processCommandOptions, error) {
	opts := processCommandOptions{envVars: map[string]string{}}
	var sandboxID string

	cmdIdx := -1
	for i, arg := range args {
		if arg == "--" {
			cmdIdx = i
			break
		}
	}

	flagArgs := args
	if cmdIdx >= 0 {
		flagArgs = args[:cmdIdx]
	}
	for _, arg := range flagArgs {
		switch {
		case strings.HasPrefix(arg, "--memory="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--memory="), "%d", &opts.memoryMB)
		case strings.HasPrefix(arg, "--cpu-time="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--cpu-time="), "%d", &opts.cpuTime)
		case includeTimeout && strings.HasPrefix(arg, "--timeout="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--timeout="), "%d", &opts.timeout)
		case strings.HasPrefix(arg, "--max-procs="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--max-procs="), "%d", &opts.maxProcs)
		case strings.HasPrefix(arg, "--max-files="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--max-files="), "%d", &opts.maxFiles)
		case arg == "--vrooli-aware":
			opts.vrooliAware = true
		case arg == "--network":
			opts.network = true
		case arg == "--json":
			opts.jsonOut = true
		case strings.HasPrefix(arg, "--workdir="):
			opts.workDir = strings.TrimPrefix(arg, "--workdir=")
		case strings.HasPrefix(arg, "--name="):
			opts.name = strings.TrimPrefix(arg, "--name=")
		case strings.HasPrefix(arg, "--env="):
			envPair := strings.TrimPrefix(arg, "--env=")
			if idx := strings.Index(envPair, "="); idx > 0 {
				opts.envVars[envPair[:idx]] = envPair[idx+1:]
			}
		case !strings.HasPrefix(arg, "-") && sandboxID == "":
			sandboxID = arg
		}
	}

	var command string
	var cmdArgs []string
	if cmdIdx >= 0 && cmdIdx+1 < len(args) {
		command = args[cmdIdx+1]
		if cmdIdx+2 < len(args) {
			cmdArgs = args[cmdIdx+2:]
		}
	}
	if sandboxID == "" || command == "" {
		if includeTimeout {
			return "", "", nil, opts, fmt.Errorf("usage: %s process exec <sandbox-id> [options] -- <command> [args...]", support.CLIName)
		}
		return "", "", nil, opts, fmt.Errorf("usage: %s process run <sandbox-id> [options] -- <command> [args...]", support.CLIName)
	}
	return sandboxID, command, cmdArgs, opts, nil
}

func applyExecOptions(reqBody map[string]any, opts processCommandOptions) {
	if opts.memoryMB > 0 {
		reqBody["memoryLimitMB"] = opts.memoryMB
	}
	if opts.cpuTime > 0 {
		reqBody["cpuTimeSec"] = opts.cpuTime
	}
	if opts.timeout > 0 {
		reqBody["timeoutSec"] = opts.timeout
	}
	if opts.maxProcs > 0 {
		reqBody["maxProcesses"] = opts.maxProcs
	}
	if opts.maxFiles > 0 {
		reqBody["maxOpenFiles"] = opts.maxFiles
	}
	if opts.network {
		reqBody["allowNetwork"] = true
	}
	if opts.vrooliAware {
		reqBody["isolationLevel"] = "vrooli-aware"
	}
	if opts.workDir != "" {
		reqBody["workingDir"] = opts.workDir
	}
	if len(opts.envVars) > 0 {
		reqBody["env"] = opts.envVars
	}
}

func renderProcessRows(processes []support.ProcessInfo) []string {
	if len(processes) == 0 {
		return nil
	}
	rows := make([]string, 0, len(processes))
	for _, process := range processes {
		status := "stopped"
		if process.Running {
			status = "running"
		}
		rows = append(rows, fmt.Sprintf(
			"PID=%d | %s | started=%s | command=%s",
			process.PID,
			status,
			process.StartedAt.Format("15:04:05"),
			support.TailTruncate(process.Command, 48),
		))
	}
	return rows
}

func renderLogRows(logs []support.LogResponse) []string {
	if len(logs) == 0 {
		return nil
	}
	rows := make([]string, 0, len(logs))
	for _, log := range logs {
		active := "inactive"
		if log.IsActive {
			active = "active"
		}
		rows = append(rows, fmt.Sprintf(
			"PID=%d | %s | size=%s | path=%s",
			log.PID,
			active,
			support.FormatBytes(log.SizeBytes),
			log.Path,
		))
	}
	return rows
}

func streamLogs(deps support.Dependencies, sandboxID string, pid int) error {
	fmt.Printf("Streaming logs for PID %d (press Ctrl+C to stop)...\n\n", pid)

	var lastOffset int64
	for {
		query := url.Values{}
		query.Set("offset", fmt.Sprintf("%d", lastOffset))

		body, err := deps.ScenarioApp().Get(fmt.Sprintf("/sandboxes/%s/processes/%d/logs", sandboxID, pid), query)
		if err != nil {
			return err
		}

		var resp support.LogResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if resp.Content != "" {
			fmt.Print(resp.Content)
			lastOffset = resp.SizeBytes
		}
		if !resp.IsActive {
			fmt.Println("\n[Process ended]")
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

type interactiveMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
	Code int    `json:"code,omitempty"`
}

type interactiveStartRequest struct {
	Command        string   `json:"command"`
	Args           []string `json:"args,omitempty"`
	IsolationLevel string   `json:"isolationLevel,omitempty"`
	AllowNetwork   bool     `json:"allowNetwork,omitempty"`
	MemoryLimitMB  int      `json:"memoryLimitMB,omitempty"`
	Cols           int      `json:"cols,omitempty"`
	Rows           int      `json:"rows,omitempty"`
}

func runInteractiveSession(deps support.Dependencies, sandboxID, command string, cmdArgs []string, vrooliAware, network bool, memoryMB int) error {
	baseURL := strings.TrimRight(strings.TrimSpace(deps.ScenarioApp().HTTPClient.BaseURL()), "/")
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/api/v1/sandboxes/%s/exec-interactive", wsURL, sandboxID)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to interactive session: %w", err)
	}
	defer conn.Close()

	cols, rows := getTerminalSize()
	startReq := interactiveStartRequest{
		Command: command,
		Args:    cmdArgs,
		Cols:    cols,
		Rows:    rows,
	}
	if vrooliAware {
		startReq.IsolationLevel = "vrooli-aware"
	}
	if network {
		startReq.AllowNetwork = true
	}
	if memoryMB > 0 {
		startReq.MemoryLimitMB = memoryMB
	}
	if err := conn.WriteJSON(startReq); err != nil {
		return fmt.Errorf("failed to send start request: %w", err)
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not set terminal to raw mode: %v\n", err)
	} else {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	done := make(chan struct{})
	var exitCode int

	go func() {
		defer close(done)
		for {
			var msg interactiveMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			switch msg.Type {
			case "stdout", "stderr":
				_, _ = os.Stdout.Write([]byte(msg.Data))
			case "exit":
				exitCode = msg.Code
				return
			case "error":
				fmt.Fprintf(os.Stderr, "\nError: %s\n", msg.Data)
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-done:
				return
			default:
			}

			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				if err := conn.WriteJSON(interactiveMessage{Type: "stdin", Data: string(buf[:n])}); err != nil {
					return
				}
			}
		}
	}()

	go func() {
		currentCols, currentRows := cols, rows
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				nextCols, nextRows := getTerminalSize()
				if nextCols != currentCols || nextRows != currentRows {
					currentCols, currentRows = nextCols, nextRows
					_ = conn.WriteJSON(interactiveMessage{Type: "resize", Cols: currentCols, Rows: currentRows})
				}
			}
		}
	}()

	<-done
	if exitCode != 0 {
		return fmt.Errorf("process exited with code %d", exitCode)
	}
	return nil
}

func getTerminalSize() (cols, rows int) {
	cols, rows = 80, 24
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err == nil {
		fmt.Sscanf(string(output), "%d %d", &rows, &cols)
	}
	return
}
