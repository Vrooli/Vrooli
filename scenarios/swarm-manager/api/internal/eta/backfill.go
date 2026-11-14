package eta

// CompletedItem is the minimal shape the backfill needs from a completed
// backlog item to derive a coarse calibration sample.
type CompletedItem struct {
	Ref       string // "<kind>/<name>"
	Kind      string
	Effort    string
	Milestone string
	Created   string // RFC3339
	Completed string // RFC3339 (the item's Updated timestamp is the completion proxy)
}

// BackfillSample is one derived calibration sample ready to be emitted as a
// backlog.duration_sample event.
type BackfillSample struct {
	Ref           string
	EffortClass   string
	DurationHours float64
	Kind          string
	Milestone     string
}

// BackfillReport summarizes a backfill pass.
type BackfillReport struct {
	Produced       int
	SkippedNoTime  int // created/completed missing, unparseable, or non-positive span
	SkippedAlready int // a duration sample already exists for this ref
}

// BuildBackfillSamples derives one backfill-origin sample per historical
// completed item, skipping items that already have a sample and items whose
// timestamps yield no positive lead time. It is pure: the caller emits the
// returned samples as events. Samples are returned in input order so a caller
// emitting them produces a deterministic log.
func BuildBackfillSamples(items []CompletedItem, alreadySampled map[string]struct{}) ([]BackfillSample, BackfillReport) {
	var samples []BackfillSample
	var rep BackfillReport
	for _, it := range items {
		if _, done := alreadySampled[it.Ref]; done {
			rep.SkippedAlready++
			continue
		}
		hours, ok := LeadTimeHours(it.Created, it.Completed)
		if !ok {
			rep.SkippedNoTime++
			continue
		}
		samples = append(samples, BackfillSample{
			Ref:           it.Ref,
			EffortClass:   NormalizeEffort(it.Effort),
			DurationHours: hours,
			Kind:          it.Kind,
			Milestone:     it.Milestone,
		})
		rep.Produced++
	}
	return samples, rep
}
