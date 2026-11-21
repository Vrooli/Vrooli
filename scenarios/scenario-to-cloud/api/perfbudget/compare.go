package perfbudget

import (
	"fmt"
	"sort"
)

// Standing of one finding. Only StandingWithin is a satisfied budget;
// StandingInsufficientSamples is never a pass and never a fail.
type Standing string

// Standing vocabulary.
const (
	StandingWithin              Standing = "within"
	StandingExceeded            Standing = "exceeded"
	StandingInsufficientSamples Standing = "insufficient_samples"
)

// Finding is one (phase, metric) comparison.
type Finding struct {
	Phase    string   `json:"phase"`
	Metric   string   `json:"metric"`
	Standing Standing `json:"standing"`
	Observed float64  `json:"observed"`
	Budget   float64  `json:"budget"`
	Samples  int      `json:"samples,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

// Verdict is the comparison of one measurement against the budgets.
// Standing is exceeded if any finding is exceeded, else insufficient_samples
// if any finding lacks evidence, else within. A verdict is a report, not a
// certification receipt; the receipt owner decides what to do with it.
type Verdict struct {
	Profile  string    `json:"profile"`
	Standing Standing  `json:"standing"`
	Findings []Finding `json:"findings"`
}

// Compare evaluates a measurement against the budgets for its profile.
//
// Rules:
//   - latency percentiles and error rate over a phase with fewer than
//     min_samples_for_percentiles samples are insufficient_samples, whatever
//     the observed value: a small sample is reported, never judged;
//   - management-process ceilings apply to the baseline phase (idle) and to
//     the sustained and peak phases (active_operation);
//   - post-cleanup growth is the delta between post_cleanup and baseline
//     end snapshots, bounded by post_cleanup_growth;
//   - a process snapshot without /proc data yields insufficient_samples for
//     RSS and FDs, never within.
func Compare(m *Measurement, b *Budgets) (Verdict, error) {
	if m == nil || b == nil {
		return Verdict{}, fmt.Errorf("measurement and budgets are required")
	}
	profile, ok := b.Profile(m.Profile)
	if !ok {
		return Verdict{}, fmt.Errorf("profile %q has no frozen budget", m.Profile)
	}
	if missing := m.Complete(); len(missing) > 0 {
		return Verdict{}, fmt.Errorf("measurement incomplete: missing %v", missing)
	}
	v := Verdict{Profile: m.Profile}
	minSamples := b.Phase23.MinSamplesForPercentiles

	for _, name := range []string{PhaseSustained, PhasePeak} {
		ph := m.Phases[name]
		lat := ph.Latency
		samples := lat.Samples
		insufficient := samples < minSamples || lat.SmallSample
		v.Findings = append(v.Findings,
			latencyFinding(name, "latency_p50_ms", lat.P50Ms, profile.Latency.P50Max, samples, insufficient, minSamples),
			latencyFinding(name, "latency_p95_ms", lat.P95Ms, profile.Latency.P95Max, samples, insufficient, minSamples),
			latencyFinding(name, "latency_p99_ms", lat.P99Ms, profile.Latency.P99Max, samples, insufficient, minSamples),
			latencyFinding(name, "error_rate", lat.ErrorRate, profile.ErrorRateMax, samples, insufficient, minSamples),
		)
	}

	mp := b.Phase23.ManagementProcess
	v.Findings = append(v.Findings, processFindings(PhaseBaseline, m.Phases[PhaseBaseline].Process, mp.Idle)...)
	v.Findings = append(v.Findings, processFindings(PhaseSustained, m.Phases[PhaseSustained].Process, mp.ActiveOperation)...)
	v.Findings = append(v.Findings, processFindings(PhasePeak, m.Phases[PhasePeak].Process, mp.ActiveOperation)...)

	base := m.Phases[PhaseBaseline].Process
	post := m.Phases[PhasePostCleanup].Process
	v.Findings = append(v.Findings, growthFindings(base, post, mp.PostCleanupGrowth)...)

	v.Standing = StandingWithin
	for _, f := range v.Findings {
		if f.Standing == StandingExceeded {
			v.Standing = StandingExceeded
			break
		}
		if f.Standing == StandingInsufficientSamples {
			v.Standing = StandingInsufficientSamples
		}
	}
	sort.SliceStable(v.Findings, func(i, j int) bool {
		if v.Findings[i].Phase != v.Findings[j].Phase {
			return phaseOrder(v.Findings[i].Phase) < phaseOrder(v.Findings[j].Phase)
		}
		return v.Findings[i].Metric < v.Findings[j].Metric
	})
	return v, nil
}

func phaseOrder(name string) int {
	for i, p := range Phases() {
		if p == name {
			return i
		}
	}
	return len(Phases())
}

func latencyFinding(phase, metric string, observed, budget float64, samples int, insufficient bool, minSamples int) Finding {
	f := Finding{Phase: phase, Metric: metric, Observed: observed, Budget: budget, Samples: samples}
	switch {
	case insufficient:
		f.Standing = StandingInsufficientSamples
		f.Reason = fmt.Sprintf("%d samples below the %d needed for a percentile claim; observed value reported, not judged", samples, minSamples)
	case observed > budget:
		f.Standing = StandingExceeded
	default:
		f.Standing = StandingWithin
	}
	return f
}

func processFindings(phase string, snap ProcessSnapshot, ceiling ResourceCeiling) []Finding {
	out := []Finding{
		{Phase: phase, Metric: "goroutines", Observed: float64(snap.Goroutines), Budget: float64(ceiling.GoroutinesMax)},
		{Phase: phase, Metric: "rss_mib", Observed: snap.RSSMiB(), Budget: ceiling.RSSMiBMax},
		{Phase: phase, Metric: "open_fds", Observed: float64(snap.OpenFDs), Budget: float64(ceiling.OpenFDsMax)},
	}
	for i := range out {
		f := &out[i]
		if (f.Metric == "rss_mib" || f.Metric == "open_fds") && !snap.Available {
			f.Standing = StandingInsufficientSamples
			f.Reason = "/proc/self was not readable; resident set and fd count are unobserved"
			continue
		}
		if f.Budget <= 0 {
			f.Standing = StandingInsufficientSamples
			f.Reason = "no ceiling frozen for this metric"
			continue
		}
		if f.Observed > f.Budget {
			f.Standing = StandingExceeded
		} else {
			f.Standing = StandingWithin
		}
	}
	return out
}

func growthFindings(base, post ProcessSnapshot, g GrowthCeiling) []Finding {
	out := []Finding{
		{Phase: PhasePostCleanup, Metric: "growth_goroutines", Observed: float64(post.Goroutines - base.Goroutines), Budget: float64(g.GoroutinesMaxDelta)},
		{Phase: PhasePostCleanup, Metric: "growth_rss_mib", Observed: post.RSSMiB() - base.RSSMiB(), Budget: g.RSSMiBMaxDelta},
		{Phase: PhasePostCleanup, Metric: "growth_open_fds", Observed: float64(post.OpenFDs - base.OpenFDs), Budget: float64(g.OpenFDsMaxDelta)},
	}
	for i := range out {
		f := &out[i]
		if f.Metric != "growth_goroutines" && (!base.Available || !post.Available) {
			f.Standing = StandingInsufficientSamples
			f.Reason = "/proc/self was not readable in one of the compared phases"
			continue
		}
		if f.Observed > f.Budget {
			f.Standing = StandingExceeded
		} else {
			f.Standing = StandingWithin
		}
	}
	return out
}
