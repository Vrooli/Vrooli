package cleanup

import "strings"

// Redact strips filesystem paths out of a message destined for an audit record
// or a warning.
//
// Cleanup errors quote the path that failed, and those paths are not
// necessarily safe to persist: a temp directory name can carry a ticket ID, a
// branch name, a customer identifier, or a token that some other tool wrote
// into a filename. The audit log outlives the incident, so anything durable
// gets the path removed and keeps the surrounding words, which is what makes an
// error like "permission denied" still diagnosable.
//
// This lives on the shared types package rather than in the orchestrator
// because both the orchestrator (audit events) and the providers (apply
// warnings) must redact identically. Two implementations would drift, and the
// one that drifted would leak.
func Redact(s string) string {
	parts := strings.Fields(s)
	for i, part := range parts {
		if strings.Contains(part, "/") || strings.Contains(part, "\\") {
			parts[i] = "[path]"
		}
	}
	return strings.Join(parts, " ")
}
