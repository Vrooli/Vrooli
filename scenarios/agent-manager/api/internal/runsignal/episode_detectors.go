package runsignal

import "agent-manager/internal/domain"

// EpisodeDetector is a pure, registered classification rule. Its declaration
// is intentionally data-bearing so labels, projections, and later metrics can
// identify the exact detector that made an episode without inspecting code.
type EpisodeDetector interface {
	Identifier() string
	ClassifierVersion() string
	CauseScope() string
	Detect(EpisodeDetectorContext) []FrictionEpisode
}

type EpisodeDetectorContext struct {
	Facts      []InvocationFact
	Events     []*domain.RunEvent
	EventsByID map[string]*domain.RunEvent
}

type episodeDetector struct {
	id, version, cause string
	detect             func(EpisodeDetectorContext) []FrictionEpisode
}

func (d episodeDetector) Identifier() string                                  { return d.id }
func (d episodeDetector) ClassifierVersion() string                           { return d.version }
func (d episodeDetector) CauseScope() string                                  { return d.cause }
func (d episodeDetector) Detect(ctx EpisodeDetectorContext) []FrictionEpisode { return d.detect(ctx) }

// EpisodeDetectors is the complete deterministic registry. Adding a detector
// requires a named declaration and labelled coverage; DeriveEpisodes itself
// remains an unchanged generic dispatcher.
func EpisodeDetectors() []EpisodeDetector {
	return []EpisodeDetector{
		episodeDetector{"repeated-work", EpisodeClassifierVersion, "recurring-workaround", adjacentEpisode("repeated-work", func(a, b InvocationFact) bool { return b.Fingerprint != "" && b.Fingerprint == a.Fingerprint })},
		episodeDetector{"command-failure", EpisodeClassifierVersion, "toolchain", adjacentEpisode("command-failure", func(a, b InvocationFact) bool {
			return a.Outcome == "failure" && b.Outcome == "failure" && a.Executable != "" && a.Executable == b.Executable
		})},
		episodeDetector{"help-recovery", EpisodeClassifierVersion, "run-execution", adjacentEpisode("help-recovery", func(a, b InvocationFact) bool { return a.Outcome == "failure" && b.HelpRecovery })},
		episodeDetector{"stall", EpisodeClassifierVersion, "run-execution", detectStalls},
		episodeDetector{"poll-loop", EpisodeClassifierVersion, "recurring-workaround", detectPollLoops},
		episodeDetector{"oscillation", EpisodeClassifierVersion, "recurring-workaround", detectOscillations},
		episodeDetector{"edit-revert", EpisodeClassifierVersion, "file-editing", detectEditReverts},
	}
}

// detectorWindow is constant, which bounds each scan and keeps both detectors
// linear in the size of even very large imported transcripts.
const detectorWindow = 32

func episodeCauseScope(id string) string {
	for _, detector := range EpisodeDetectors() {
		if detector.Identifier() == id {
			return detector.CauseScope()
		}
	}
	return "unknown"
}

func adjacentEpisode(pattern string, matches func(InvocationFact, InvocationFact) bool) func(EpisodeDetectorContext) []FrictionEpisode {
	return func(ctx EpisodeDetectorContext) []FrictionEpisode {
		out := []FrictionEpisode{}
		for i := 1; i < len(ctx.Facts); i++ {
			if matches(ctx.Facts[i-1], ctx.Facts[i]) {
				out = append(out, newEpisode(pattern, ctx.Facts[i-1], ctx.Facts[i], ctx.EventsByID, ctx.Events))
			}
		}
		return out
	}
}

func detectStalls(ctx EpisodeDetectorContext) []FrictionEpisode {
	out := []FrictionEpisode{}
	for i := 1; i < len(ctx.Events); i++ {
		if ctx.Events[i-1] == nil || ctx.Events[i] == nil || ctx.Events[i].Timestamp.Sub(ctx.Events[i-1].Timestamp) <= idleEpisodeThreshold {
			continue
		}
		fact := InvocationFact{CallEventID: ctx.Events[i-1].ID.String(), ResultEventID: ctx.Events[i].ID.String(), Ownership: "unknown"}
		e := newEpisode("stall", fact, fact, ctx.EventsByID, ctx.Events)
		e.EndEventID, e.EvidenceEventIDs = ctx.Events[i].ID.String(), []string{e.StartEventID, ctx.Events[i].ID.String()}
		e.WallClockMS, e.Tokens = ctx.Events[i].Timestamp.Sub(ctx.Events[i-1].Timestamp).Milliseconds(), episodeTokens(e.StartEventID, e.EndEventID, ctx.EventsByID, ctx.Events)
		e.EpisodeID, e.Fingerprint = fingerprint(e.Pattern, e.StartEventID, e.EndEventID), fingerprint(e.Pattern, e.SuspectedOwnerCommand, e.CauseScope)
		out = append(out, e)
	}
	return out
}
