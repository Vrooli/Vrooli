package portparse

import (
	"regexp"
	"scenario-to-cloud/domain"
	"strconv"
	"strings"
)

// ParseSSOutput parses the output of `ss -tlnp`.
//
// Format: State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
// Example: LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1234,fd=3))
func ParseSSOutput(output string) []domain.PortBinding {
	var ports []domain.PortBinding
	lines := strings.Split(output, "\n")

	portRegex := regexp.MustCompile(`:(\d+)$`)
	processRegex := regexp.MustCompile(`\(\("([^"]+)",pid=(\d+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "State") || strings.HasPrefix(line, "Netid") {
			continue
		}
		if !strings.HasPrefix(line, "LISTEN") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		localAddr := fields[3]
		portMatch := portRegex.FindStringSubmatch(localAddr)
		if len(portMatch) < 2 {
			continue
		}

		port, err := strconv.Atoi(portMatch[1])
		if err != nil {
			continue
		}

		var processName string
		var pid int
		if len(fields) >= 6 {
			processInfo := fields[5]
			if len(fields) > 6 {
				processInfo = strings.Join(fields[5:], " ")
			}
			procMatch := processRegex.FindStringSubmatch(processInfo)
			if len(procMatch) >= 3 {
				processName = procMatch[1]
				pid, _ = strconv.Atoi(procMatch[2])
			}
		}

		binding := domain.PortBinding{Port: port, Process: processName}
		if pid > 0 {
			binding.PID = &pid
		}
		ports = append(ports, binding)
	}

	return ports
}

// ExtractPIDsFromSS parses PIDs from ss -ltnp output.
func ExtractPIDsFromSS(output string) []string {
	var pids []string
	seen := make(map[string]bool)
	pidRegex := regexp.MustCompile(`pid=(\d+)`)

	for _, line := range strings.Split(output, "\n") {
		matches := pidRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) <= 1 {
				continue
			}
			pid := match[1]
			if seen[pid] {
				continue
			}
			seen[pid] = true
			pids = append(pids, pid)
		}
	}

	return pids
}
