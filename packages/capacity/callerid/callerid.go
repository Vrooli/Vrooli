// Package callerid names the local process on the other end of a loopback TCP
// connection by its Vrooli scope (scenario service, resource component, agent
// session) and executable, for example
// "scenario:agent-manager-start-api/agent-manager-api".
//
// A server that sees only a loopback peer cannot trust a caller's own claim of
// who it is, and most callers send none. The kernel knows: the client socket
// belongs to a process, and the process belongs to a systemd scope whose name
// carries the scenario and lifecycle step. The scope is what stays stable
// across restarts, so "who called" survives PID churn.
package callerid

import (
	"path"
	"strings"
)

// Unknown is the identity of a caller that could not be identified: a
// non-loopback peer, a platform without a peer-process lookup, or a sender
// that exited before it was resolved.
const Unknown = "unknown"

// Kinds of caller a label can name.
const (
	KindScenario = "scenario"
	KindResource = "resource"
	KindAgent    = "agent"
	KindHost     = "host"
)

// scopes maps a systemd unit prefix to the kind of caller it holds. Order
// matters: the resource prefix is a refinement of the service prefix.
var scopes = []struct{ prefix, kind string }{
	{prefix: "vrooli-service-resource-", kind: KindResource},
	{prefix: "vrooli-service-", kind: KindScenario},
	{prefix: "vrooli-agent-", kind: KindAgent},
}

// Label names a process by its cgroup unit and executable, for example
// "scenario:ai-gateway-start-api/ai-gateway-api" or "agent:codex/opencode".
// Unit names carry per-instance hex suffixes; they are dropped so a caller
// keeps one identity across restarts.
func Label(cgroupPath, executable string) string {
	exe := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(executable), " (deleted)"))
	exe = path.Base(exe)
	if exe == "." || exe == "/" || exe == "" {
		exe = Unknown
	}
	unit := path.Base(strings.TrimSpace(cgroupPath))
	for _, suffix := range []string{".scope", ".service", ".slice"} {
		unit = strings.TrimSuffix(unit, suffix)
	}
	kind := KindHost
	for _, scope := range scopes {
		if strings.HasPrefix(unit, scope.prefix) {
			kind, unit = scope.kind, strings.TrimPrefix(unit, scope.prefix)
			break
		}
	}
	unit = trimInstanceSuffix(unit)
	if unit == "" || unit == "." || unit == "/" {
		unit = Unknown
	}
	return kind + ":" + unit + "/" + exe
}

// Parse splits a label into its kind, scope unit and executable. ok is false
// for Unknown and anything else that is not a label.
func Parse(label string) (kind, unit, executable string, ok bool) {
	kind, rest, found := strings.Cut(strings.TrimSpace(label), ":")
	if !found || kind == "" {
		return "", "", "", false
	}
	unit, executable, found = strings.Cut(rest, "/")
	if !found || unit == "" || executable == "" {
		return "", "", "", false
	}
	return kind, unit, executable, true
}

// trimInstanceSuffix drops trailing dash-separated segments that are hex
// instance identifiers (scope digests, session IDs, UUID parts). A segment
// must contain a digit, so ordinary words made of hex letters stay.
func trimInstanceSuffix(name string) string {
	for {
		dash := strings.LastIndexByte(name, '-')
		if dash <= 0 || !isInstanceSegment(name[dash+1:]) {
			return name
		}
		name = name[:dash]
	}
}

func isInstanceSegment(segment string) bool {
	if segment == "" {
		return false
	}
	digit := false
	for _, r := range segment {
		switch {
		case r >= '0' && r <= '9':
			digit = true
		case r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return digit
}
