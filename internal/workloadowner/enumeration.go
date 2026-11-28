package workloadowner

import "strings"

// ParseServiceUnits parses the stable whitespace columns emitted by
// `systemctl list-units --all --no-legend --no-pager`. It accepts the fixture
// form and ignores headers, avoiding a platform-specific shell parser.
func ParseServiceUnits(data []byte) []Workload {
	var out []Workload
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasSuffix(fields[0], ".service") {
			continue
		}
		out = append(out, Workload{Kind: "service-unit", Name: fields[0], Running: fields[3] == "running", Evidence: []string{"systemd unit listing"}})
	}
	return out
}
