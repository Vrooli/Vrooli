package runsignal

func detectFallbackAfterCapability(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i, first := range ctx.Facts {
		if first.Ownership != OwnershipResolved || first.Outcome != "success" || first.IntentClass == "" {
			continue
		}
		limit := i + detectorWindow
		if limit > len(ctx.Facts) {
			limit = len(ctx.Facts)
		}
		for _, next := range ctx.Facts[i+1 : limit] {
			if next.Ownership != OwnershipExternal || next.IntentClass != first.IntentClass {
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

func detectCapabilityAbandoned(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i, failed := range ctx.Facts {
		if failed.Ownership != OwnershipResolved || failed.Outcome != "failure" || failed.IntentClass == "" {
			continue
		}
		limit := i + detectorWindow
		if limit > len(ctx.Facts) {
			limit = len(ctx.Facts)
		}
		retried := false
		for _, next := range ctx.Facts[i+1 : limit] {
			if next.IntentClass == failed.IntentClass {
				retried = true
				break
			}
		}
		if retried {
			continue
		}
		e := newEpisode("capability-abandoned", failed, failed, ctx.EventsByID, ctx.Events)
		e.SuspectedOwnerScenario = failed.Executable
		e.SuspectedOwnerCommand = failed.CommandPath
		e.OwnerConfidence = "manifest-derived"
		e.EvidenceEventIDs = []string{failed.CallEventID}
		out = append(out, e)
	}
	return out
}
