//go:build linux

package network

import (
	"bufio"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

func ListenersForPort(port int) ([]PortListener, error) {
	inspection, err := inspectPortListeners(port)
	if err != nil {
		return nil, err
	}
	return inspection.Listeners, nil
}

func listenerInspectionStatus() ListenerInspection {
	if path, err := exec.LookPath("lsof"); err == nil {
		return ListenerInspection{
			Available: true,
			Tool:      path,
		}
	}
	if path, err := exec.LookPath("ss"); err == nil {
		return ListenerInspection{
			Available: true,
			Tool:      path,
		}
	}
	return ListenerInspection{
		Available: false,
		Tool:      "lsof,ss",
		Reason:    "neither lsof nor ss is installed",
	}
}

func inspectPortListeners(port int) (PortInspection, error) {
	pids, lsofPath, err := listListeningPIDs(port)
	if err != nil {
		return PortInspection{}, err
	}
	if len(pids) > 0 {
		listeners := make([]PortListener, 0, len(pids))
		for _, pid := range pids {
			state, command, readErr := readProcessState(pid)
			if readErr != nil {
				command = ""
			}
			listeners = append(listeners, PortListener{
				PID:     pid,
				Command: command,
				Zombie:  state == "Z",
			})
		}
		return PortInspection{
			Listeners: listeners,
			Inspection: ListenerInspection{
				Available: true,
				Tool:      lsofPath,
			},
		}, nil
	}

	listening, ssPath, err := hasListeningSocketWithSS(port)
	if err != nil {
		return PortInspection{}, err
	}
	if ssPath == "" && lsofPath == "" {
		return PortInspection{
			Inspection: ListenerInspection{
				Available: false,
				Tool:      "lsof,ss",
				Reason:    "neither lsof nor ss is installed",
			},
		}, nil
	}
	tool := lsofPath
	if listening {
		if tool != "" {
			tool += "+"
		}
		tool += ssPath
		return PortInspection{
			Listeners: []PortListener{{
				Command: "listener detected by ss",
			}},
			Inspection: ListenerInspection{
				Available: true,
				Tool:      tool,
			},
		}, nil
	}
	return PortInspection{
		Inspection: ListenerInspection{
			Available: true,
			Tool:      tool,
		},
	}, nil
}

func listListeningPIDs(port int) ([]int, string, error) {
	path, err := exec.LookPath("lsof")
	if err != nil {
		return nil, "", nil
	}
	cmd := exec.Command(path, "-tiTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-P", "-n")
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, path, nil
		}
		return nil, path, err
	}

	pids := make([]int, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids, path, scanner.Err()
}

func hasListeningSocketWithSS(port int) (bool, string, error) {
	path, err := exec.LookPath("ss")
	if err != nil {
		return false, "", nil
	}
	cmd := exec.Command(path, "-ltnH", "sport", "=", ":"+strconv.Itoa(port))
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, path, nil
		}
		return false, path, err
	}
	return ssOutputHasListenerForPort(output, port), path, nil
}

func ssOutputHasListenerForPort(output []byte, port int) bool {
	want := ":" + strconv.Itoa(port)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		for _, field := range fields {
			if strings.HasSuffix(field, want) || strings.Contains(field, want+" ") {
				return true
			}
		}
	}
	return false
}

func readProcessState(pid int) (string, string, error) {
	cmd := exec.Command("ps", "-o", "state=,command=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	line := strings.TrimSpace(string(output))
	if line == "" {
		return "", "", nil
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", "", nil
	}
	state := fields[0]
	command := ""
	if len(fields) > 1 {
		command = strings.Join(fields[1:], " ")
	}
	return state, command, nil
}
