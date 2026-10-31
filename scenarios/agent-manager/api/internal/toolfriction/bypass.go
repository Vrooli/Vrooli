package toolfriction

import (
	"regexp"
	"strconv"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
)

// Bypass methods: how a command reached a scenario's internals around its CLI.
const (
	BypassDatabase = "sqlite3 on its database"
	BypassHTTP     = "HTTP to its port"
	BypassState    = "reads of its state directory"
)

var (
	// scenarioData is a scenario's own data directory or its per-host state.
	scenarioData = regexp.MustCompile(`(?:^|/)scenarios/([a-z0-9][a-z0-9-]*)/data/|\.vrooli/(?:state|data)/vrooli/([a-z0-9][a-z0-9-]*)`)
	// scenarioDB is a scenario's database named for it under a data
	// directory (`data/agent-manager.db`), as seen from the scenario's root.
	scenarioDB = regexp.MustCompile(`(?:^|[/:])data/([a-z0-9][a-z0-9-]*)\.(?:db|sqlite3?)\b`)
	localPort  = regexp.MustCompile(`(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1\]):(\d{2,5})(/[^\s'"?#]*)?`)
	// healthRoute is a liveness probe, which every caller may make.
	healthRoute = regexp.MustCompile(`^/(?:api/)?(?:v\d+/)?(?:health|healthz|ready|readyz|livez)/?$`)
	// stateReaders read files or directories without changing them.
	stateReaders = map[string]bool{"cat": true, "head": true, "tail": true, "less": true, "ls": true, "find": true, "rg": true, "grep": true, "jq": true, "sed": true, "du": true, "stat": true, "wc": true, "tree": true}
	httpClients  = map[string]bool{"curl": true, "wget": true, "http": true, "xh": true}
)

// bypassOf names the scenario whose internals a command reaches directly, and
// how: sqlite3 on its data, an HTTP client to a non-health route of the API
// port it serves (UI ports and health probes are smoke checks), or a read
// under its data or state directory (a read tool's own path included).
func bypassOf(f Fact, call *domain.ToolCallEventData, p runsignal.Pipeline, found bool, ports PortOwners) (owner, method string) {
	if !found {
		if f.Capability == runsignal.KindFileRead && call != nil {
			for _, key := range []string{"file_path", "path", "filePath"} {
				if path, ok := call.Input[key].(string); ok {
					if scenario := dataOwner(path); scenario != "" {
						return scenario, BypassState
					}
				}
			}
		}
		return "", ""
	}
	program := p.Words[0]
	for _, word := range p.Words[1:] {
		switch {
		case program == "sqlite3" || program == "sqlite":
			if scenario := dataOwner(word); scenario != "" {
				return scenario, BypassDatabase
			}
			if m := scenarioDB.FindStringSubmatch(word); m != nil && namedOwner(m[1]) == m[1] {
				return m[1], BypassDatabase
			}
		case httpClients[program] && ports != nil:
			if m := localPort.FindStringSubmatch(word); m != nil {
				if port, err := strconv.Atoi(m[1]); err == nil {
					if scenario := ports(port); scenario != "" && !healthRoute.MatchString(m[2]) {
						return scenario, BypassHTTP
					}
				}
			}
		case stateReaders[program]:
			if scenario := dataOwner(word); scenario != "" {
				return scenario, BypassState
			}
		}
	}
	return "", ""
}

// dataOwner is the scenario whose data or state a path lies under; a variant
// directory (web-console_presentation) matches up to its scenario's name.
func dataOwner(path string) string {
	if m := scenarioData.FindStringSubmatch(path); m != nil {
		return m[1] + m[2]
	}
	return ""
}
