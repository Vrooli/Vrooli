package toolfriction

import (
	"regexp"
	"strconv"
	"strings"

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
	localPort  = regexp.MustCompile(`^(?:https?://)?(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1\]):(\d{2,5})(/[^\s'"?#]*)?(?:[?#][^\s]*)?$`)
	// healthRoute is a liveness probe, which every caller may make.
	healthRoute = regexp.MustCompile(`^/(?:api/)?(?:v\d+/)?(?:health|healthz|ready|readyz|livez)/?$`)
	// stateReaders read files or directories without changing them.
	stateReaders = map[string]bool{"cat": true, "head": true, "tail": true, "less": true, "ls": true, "find": true, "rg": true, "grep": true, "jq": true, "sed": true, "du": true, "stat": true, "wc": true, "tree": true}
	httpClients  = map[string]bool{"curl": true, "wget": true, "http": true, "xh": true}
	// These curl options take values that can contain URL-looking text but are
	// not request destinations. Keep this list limited to common payload,
	// header, identity and proxy options so the endpoint scan stays auditable.
	curlValueFlags = map[string]bool{
		"-d": true, "--data": true, "--data-ascii": true, "--data-binary": true,
		"--data-raw": true, "--data-urlencode": true, "-F": true, "--form": true,
		"--form-string": true, "-H": true, "--header": true, "-X": true,
		"--request": true, "-u": true, "--user": true, "-A": true,
		"--user-agent": true, "-e": true, "--referer": true, "-b": true,
		"--cookie": true, "-x": true, "--proxy": true, "--preproxy": true,
	}
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
		case stateReaders[program]:
			if scenario := dataOwner(word); scenario != "" {
				return scenario, BypassState
			}
		}
	}
	if httpClients[program] && ports != nil {
		for _, endpoint := range httpEndpoints(program, p.Words) {
			port, path, ok := endpointPort(endpoint)
			if ok {
				if scenario := ports(port); scenario != "" && !healthRoute.MatchString(path) {
					return scenario, BypassHTTP
				}
			}
		}
	}
	return "", ""
}

// httpEndpoints returns the arguments that can name an HTTP destination. Curl
// accepts arbitrary data and header values, so those option values must not be
// mistaken for the request URL. Other supported clients retain their prior
// argument scan until their own invocation grammars are modeled.
func httpEndpoints(program string, words []string) []string {
	if program != "curl" {
		return words[1:]
	}
	var endpoints []string
	for i := 1; i < len(words); i++ {
		word := words[i]
		if word == "--" {
			return append(endpoints, words[i+1:]...)
		}
		if word == "--url" {
			if i+1 < len(words) {
				endpoints = append(endpoints, words[i+1])
				i++
			}
			continue
		}
		if flag, value, hasValue := strings.Cut(word, "="); hasValue && strings.HasPrefix(flag, "-") {
			if flag == "--url" {
				endpoints = append(endpoints, value)
			}
			continue
		}
		if curlValueFlags[word] {
			i++
			continue
		}
		if strings.HasPrefix(word, "-") {
			continue
		}
		endpoints = append(endpoints, word)
	}
	return endpoints
}

// endpointPort resolves the host and port from a request URL. Searching an
// entire argument for a local host misattributes remote URLs whose query string
// happens to mention localhost.
func endpointPort(endpoint string) (port int, path string, ok bool) {
	m := localPort.FindStringSubmatch(endpoint)
	if m == nil {
		return 0, "", false
	}
	port, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, "", false
	}
	return port, m[2], true
}

// dataOwner is the scenario whose data or state a path lies under; a variant
// directory (web-console_presentation) matches up to its scenario's name.
func dataOwner(path string) string {
	if m := scenarioData.FindStringSubmatch(path); m != nil {
		return m[1] + m[2]
	}
	return ""
}
