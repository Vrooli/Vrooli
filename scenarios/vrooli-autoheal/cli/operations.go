package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func (a *App) runLoop(args []string) error {
	loopBinary, err := support.ResolveLoopBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(loopBinary, translateLoopArgs(args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func translateLoopArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--interval-seconds="):
			out = append(out, "--interval="+strings.TrimPrefix(arg, "--interval-seconds="))
		case arg == "--interval-seconds" && i+1 < len(args):
			i++
			out = append(out, "--interval", args[i])
		default:
			out = append(out, arg)
		}
	}
	return out
}

func (a *App) diagnosePort(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: vrooli-autoheal diagnose-port <port> [scenario]")
	}
	port := strings.TrimSpace(args[0])
	scenario := ""
	if len(args) > 1 {
		scenario = strings.TrimSpace(args[1])
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("port must be an integer")
	}

	lines := []string{
		fmt.Sprintf("Port: %s", port),
	}
	if scenario != "" {
		lines = append(lines, fmt.Sprintf("Scenario filter: %s", scenario))
	}

	triage := []string{}
	if pids, details := listenersOnPort(port); len(pids) > 0 {
		triage = append(triage, fmt.Sprintf("Processes currently listening on %s: %s", port, strings.Join(pids, ", ")))
		triage = append(triage, details...)
	} else {
		triage = append(triage, fmt.Sprintf("No listeners currently bound to %s.", port))
	}

	lockPath := filepath.Join(os.Getenv("HOME"), ".vrooli", "state", "scenarios", ".port_"+port+".lock")
	if data, err := os.ReadFile(lockPath); err == nil {
		lockValue := strings.TrimSpace(string(data))
		triage = append(triage, fmt.Sprintf("Lock file present: %s", lockPath))
		triage = append(triage, fmt.Sprintf("Lock contents: %s", lockValue))
	} else {
		triage = append(triage, fmt.Sprintf("No lock file found at %s", lockPath))
	}

	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: lines,
		Triage: []cliapp.TriageGroup{
			{Heading: "Diagnostics", Items: triage},
		},
		NextSteps: []string{
			fmt.Sprintf("vrooli-autoheal locks list"),
			fmt.Sprintf("vrooli-autoheal orphans list"),
			fmt.Sprintf("vrooli scenario logs %s", scenarioOrDefault(scenario)),
		},
	})
}

func listenersOnPort(port string) ([]string, []string) {
	output, err := exec.Command("lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN").CombinedOutput()
	if err != nil {
		return nil, []string{fmt.Sprintf("Unable to inspect listeners with lsof: %v", err)}
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return nil, nil
	}
	pids := make([]string, 0, len(lines)-1)
	details := make([]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pids = append(pids, fields[1])
		details = append(details, line)
	}
	return pids, details
}

func scenarioOrDefault(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "vrooli-autoheal"
	}
	return value
}
