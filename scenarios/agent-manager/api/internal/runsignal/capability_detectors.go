package runsignal

import (
	"strings"

	"agent-manager/internal/domain"
)

// genericCapabilities say nothing about the job a call does.
var genericCapabilities = map[string]bool{"": true, "shell": true, "other": true, "code": true}

// directAccessPrograms read a service's data without its CLI.
var directAccessPrograms = map[string]bool{"sqlite3": true, "psql": true, "redis-cli": true}

// reachesServiceDirectly is an external call that talks to a local service or
// its database itself: a database client, or an HTTP client or script aimed at
// a loopback address.
func reachesServiceDirectly(fact InvocationFact, byID map[string]*domain.RunEvent) bool {
	if fact.Ownership != OwnershipExternal {
		return false
	}
	tokens := factTokens(fact, byID)
	if len(tokens) == 0 {
		return false
	}
	if directAccessPrograms[tokens[0]] {
		return true
	}
	text := strings.Join(tokens, " ")
	return strings.Contains(text, "127.0.0.1:") || strings.Contains(text, "localhost:")
}

// serviceCommandWindow is how many calls after a working project command a
// direct service access still reads as its fallback.
const serviceCommandWindow = 6

// detectFallbackAfterCapability finds a Vrooli command that worked, followed by
// a different call, outside Vrooli, doing the same job: the same capability
// (search-hub query, then rg), or the service's own API or database reached
// directly (a project command, then curl or sqlite3 against it). Two distinct
// calls are required; one call never falls back from itself.
func detectFallbackAfterCapability(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i, first := range ctx.Facts {
		if first.Ownership != OwnershipResolved || first.Outcome == OutcomeFailure || isHelpProbe(first, ctx.EventsByID) {
			continue
		}
		limit := min(i+detectorWindow, len(ctx.Facts))
		for k, next := range ctx.Facts[i+1 : limit] {
			sameJob := !genericCapabilities[first.Capability] && next.Ownership == OwnershipExternal && next.Capability == first.Capability
			direct := k < serviceCommandWindow && reachesServiceDirectly(next, ctx.EventsByID)
			if next.CallEventID == first.CallEventID || !(sameJob || direct) {
				continue
			}
			e := newEpisode("fallback-after-capability", first, next, ctx.EventsByID, ctx.Events)
			e.SuspectedOwnerScenario = first.Executable
			e.SuspectedOwnerCommand = first.CommandPath
			e.OwnerConfidence = "manifest-derived"
			e.EvidenceEventIDs = []string{first.CallEventID, next.CallEventID}
			out = append(out, e)
			break
		}
	}
	return out
}

// detectCapabilityAbandoned finds a Vrooli command that failed and was never
// tried again in its family. Only a failure with an error signature counts: an
// exit code alone may be a result. A failed usage probe is not a failure of the
// capability, and neither a later attempt nor a later success of the family
// counts as abandonment.
func detectCapabilityAbandoned(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i, fail := range ctx.Facts {
		if fail.Ownership != OwnershipResolved || !failed(fail) || fail.FailureSignature == "" || fail.CommandPath == "" || isHelpProbe(fail, ctx.EventsByID) {
			continue
		}
		limit := min(i+detectorWindow, len(ctx.Facts))
		retried := false
		for _, next := range ctx.Facts[i+1 : limit] {
			if sameFamily(fail, next) && !isHelpProbe(next, ctx.EventsByID) {
				retried = true
				break
			}
		}
		if retried {
			continue
		}
		e := newEpisode("capability-abandoned", fail, fail, ctx.EventsByID, ctx.Events)
		e.SuspectedOwnerScenario = fail.Executable
		e.SuspectedOwnerCommand = fail.CommandPath
		e.OwnerConfidence = "manifest-derived"
		e.EvidenceEventIDs = []string{fail.CallEventID}
		out = append(out, e)
	}
	return out
}
