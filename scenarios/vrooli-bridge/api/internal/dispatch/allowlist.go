package dispatch

import "strings"

// DefaultManifest is the control-plane allowlist of recognised verb namespaces —
// the "scenario-CLI manifest" half of the dispatch check. A job whose verb is
// not one of these is rejected as not-in-manifest before the per-node scope
// check even runs. It is deliberately limited to the safe operational/test
// verbs; privileged or destructive namespaces (`scenario deploy`, `secrets …`)
// are absent and therefore never dispatchable through bridge, regardless of any
// node scope. Override per deployment if a node legitimately needs more.
var DefaultManifest = []string{
	"scenario test",
	"scenario build",
	"scenario start",
	"scenario stop",
	"scenario status",
	"scenario logs",
}

// shellMetachars are characters that would have meaning to a shell. A typed job
// never reaches a shell (the node-agent execs an argv, never `sh -c`), so any
// token containing one is rejected as defence in depth — it can only be an
// attempt to smuggle a shell construct through a typed field.
const shellMetachars = "|&;<>()$`\\\"'\n\r\t*?[]{}!#~"

// Allow validates the job against the manifest allowlist and the node's granted
// scopes. It is pure (no I/O) so the highest-stakes decision in the scenario is
// exhaustively table-testable. Returns nil when the verb is allowlisted and
// in-scope; otherwise a typed sentinel naming the precise reason.
//
// Order of checks (each a distinct, testable rejection reason):
//  1. verb non-empty (ErrInvalidJob)
//  2. no shell metacharacter in any token (ErrUnsafeToken)
//  3. verb is a recognised manifest verb (ErrVerbNotInManifest)
//  4. verb matches at least one node scope (ErrVerbOutOfScope)
func Allow(job Job, scopes, manifest []string) error {
	j := job.trimmed()
	if j.Verb == "" {
		return ErrInvalidJob{Field: "verb", Reason: "required"}
	}

	for _, tok := range append([]string{j.Verb, j.Scenario}, j.Args...) {
		if i := strings.IndexAny(tok, shellMetachars); i >= 0 {
			return ErrUnsafeToken{Token: tok, Reason: "contains a shell metacharacter"}
		}
	}

	if !inManifest(j.Verb, manifest) {
		return ErrVerbNotInManifest{Verb: j.Verb}
	}
	if !anyScopeMatches(scopes, j.Verb) {
		return ErrVerbOutOfScope{Verb: j.Verb}
	}
	return nil
}

// inManifest reports whether verb is exactly one of the manifest entries.
func inManifest(verb string, manifest []string) bool {
	for _, m := range manifest {
		if strings.TrimSpace(m) == verb {
			return true
		}
	}
	return false
}

// anyScopeMatches reports whether any granted scope covers verb.
func anyScopeMatches(scopes []string, verb string) bool {
	for _, s := range scopes {
		if scopeMatches(strings.TrimSpace(s), verb) {
			return true
		}
	}
	return false
}

// scopeMatches implements the scope glob grammar: an exact match, a trailing-`*`
// prefix match (e.g. "scenario test*" covers "scenario test" and "scenario
// testall"), or the universal "*". The wildcard is only honoured as a trailing
// character — an interior `*` is treated literally, so a scope can never be
// tricked into matching across the namespace boundary.
func scopeMatches(scope, verb string) bool {
	switch {
	case scope == "":
		return false
	case scope == "*":
		return true
	case strings.HasSuffix(scope, "*"):
		return strings.HasPrefix(verb, strings.TrimSuffix(scope, "*"))
	default:
		return scope == verb
	}
}
