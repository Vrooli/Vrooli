package runsignal

import (
	"strings"
	"time"
)

// These detectors intentionally use only typed invocation facts and event
// timing. Phrase spans may corroborate a signal but never create one of these
// structural episodes on their own.

func detectRetryAfterFailure(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i := 1; i < len(ctx.Facts); i++ {
		before, after := ctx.Facts[i-1], ctx.Facts[i]
		if before.Outcome != "failure" || after.Outcome == "unknown" || before.Executable == "" || before.Executable != after.Executable {
			continue
		}
		e := newEpisode("retry-after-failure", before, after, ctx.EventsByID, ctx.Events)
		e.EvidenceEventIDs = []string{before.CallEventID, before.ResultEventID, after.CallEventID, after.ResultEventID}
		e.FailedJoinedCalls = 1
		out = append(out, e)
	}
	return out
}

func detectFlagHunting(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i := 0; i < len(ctx.Facts); i++ {
		first := ctx.Facts[i]
		if first.Executable == "" || first.Outcome != "failure" {
			continue
		}
		seenShapes := map[string]bool{first.IntentClass: true}
		last := first
		for j := i + 1; j < len(ctx.Facts) && j <= i+detectorWindow; j++ {
			current := ctx.Facts[j]
			if current.Executable != first.Executable {
				continue
			}
			seenShapes[current.IntentClass] = true
			last = current
			if len(seenShapes) >= 3 {
				e := newEpisode("flag-hunting", first, last, ctx.EventsByID, ctx.Events)
				e.CycleCount = len(seenShapes)
				e.EvidenceEventIDs = []string{first.CallEventID, last.CallEventID}
				out = append(out, e)
				break
			}
		}
	}
	return out
}

func detectAbandonedCommandFamily(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i, failed := range ctx.Facts {
		if failed.Executable == "" || failed.Outcome != "failure" {
			continue
		}
		last := failed
		resolved := false
		for j := i + 1; j < len(ctx.Facts) && j <= i+detectorWindow; j++ {
			candidate := ctx.Facts[j]
			if candidate.Executable != failed.Executable {
				continue
			}
			last = candidate
			if candidate.Outcome == "success" {
				resolved = true
				break
			}
		}
		if !resolved && last.CallEventID != failed.CallEventID {
			e := newEpisode("abandoned-command-family", failed, last, ctx.EventsByID, ctx.Events)
			e.FailedJoinedCalls = 1
			out = append(out, e)
		}
	}
	return out
}

func detectReadThenReread(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i, first := range ctx.Facts {
		if first.Capability != "file-read" || first.Fingerprint == "" {
			continue
		}
		for j := i + 1; j < len(ctx.Facts); j++ {
			second := ctx.Facts[j]
			if second.Capability == "file-read" && second.Fingerprint == first.Fingerprint {
				e := newEpisode("read-then-reread", first, second, ctx.EventsByID, ctx.Events)
				e.CycleCount = 2
				out = append(out, e)
				break
			}
		}
	}
	return out
}

func detectTimeToFirstSuccess(ctx EpisodeDetectorContext) []FrictionEpisode {
	if len(ctx.Events) == 0 || len(ctx.Facts) == 0 {
		return nil
	}
	firstEvent := ctx.Events[0]
	if firstEvent == nil {
		return nil
	}
	var success *InvocationFact
	for i := range ctx.Facts {
		if ctx.Facts[i].Outcome == "success" {
			candidate := ctx.Facts[i]
			success = &candidate
			break
		}
	}
	if success == nil {
		return nil
	}
	successEvent := ctx.EventsByID[success.ResultEventID]
	if successEvent == nil {
		successEvent = ctx.EventsByID[success.CallEventID]
	}
	if successEvent == nil || successEvent.Timestamp.Sub(firstEvent.Timestamp) < 30*time.Second {
		return nil
	}
	first := InvocationFact{CallEventID: firstEvent.ID.String()}
	e := newEpisode("time-to-first-success", first, *success, ctx.EventsByID, ctx.Events)
	e.WallClockMS = successEvent.Timestamp.Sub(firstEvent.Timestamp).Milliseconds()
	e.Turns = len(ctx.Facts)
	if strings.TrimSpace(e.OwnerConfidence) == "" {
		e.OwnerConfidence = "unknown"
	}
	return []FrictionEpisode{e}
}
