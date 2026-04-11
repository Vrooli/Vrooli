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
	pids, err := listListeningPIDs(port)
	if err != nil {
		return nil, err
	}
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
	return listeners, nil
}

func listListeningPIDs(port int) ([]int, error) {
	path, err := exec.LookPath("lsof")
	if err != nil {
		return nil, nil
	}
	cmd := exec.Command(path, "-tiTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-P", "-n")
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil
		}
		return nil, err
	}

	pids := make([]int, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids, scanner.Err()
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
