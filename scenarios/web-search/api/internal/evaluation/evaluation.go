// Package evaluation owns controlled research-method comparisons. It consumes
// frozen observations and never treats test provenance as operator usage.
package evaluation

import (
	"fmt"
	"sort"
)

type Observation struct {
	TaskID           string
	MethodHash       string
	CorpusHash       string
	SourcePopulation string
	Stratum          string
	Effort           float64
	Support          bool
	Coverage         bool
	Provenance       string
}

type Pair struct {
	TaskID       string
	Baseline     Observation
	Candidate    Observation
	EffortDelta  float64
	QualityEqual bool
}

type Report struct {
	Pairs               []Pair
	Accepted            bool
	Reason              string
	BaselineMethod      string
	CandidateMethod     string
	Population          int
	MeanBaselineEffort  float64
	MeanCandidateEffort float64
	Strata              map[string]StratumReport
}

type StratumReport struct {
	Population          int
	MeanBaselineEffort  float64
	MeanCandidateEffort float64
	QualityPreserved    bool
	EffortReduced       bool
}

// Compare pairs observations by task and stratum. A candidate is accepted
// only when every pair preserves both hard quality assertions and its mean
// effort is lower. Missing or duplicate observations fail closed.
func Compare(baseline, candidate []Observation) (Report, error) {
	if len(baseline) == 0 || len(candidate) == 0 {
		return Report{Reason: "insufficient_population"}, nil
	}
	base := map[string]Observation{}
	corpusHash := ""
	sourcePopulation := ""
	for _, o := range baseline {
		if o.Provenance != "test" {
			return Report{}, fmt.Errorf("baseline observation %q is not test provenance", o.TaskID)
		}
		if o.TaskID == "" || o.MethodHash == "" || o.Stratum == "" {
			return Report{}, fmt.Errorf("observation identity is required")
		}
		if corpusHash == "" {
			corpusHash = o.CorpusHash
			sourcePopulation = o.SourcePopulation
		} else if o.CorpusHash != corpusHash || o.SourcePopulation != sourcePopulation {
			return Report{}, fmt.Errorf("baseline population identity changed")
		}
		key := o.TaskID + "\x00" + o.Stratum
		if _, exists := base[key]; exists {
			return Report{}, fmt.Errorf("duplicate baseline observation %q", o.TaskID)
		}
		base[key] = o
	}
	seen := map[string]bool{}
	report := Report{BaselineMethod: baseline[0].MethodHash, CandidateMethod: candidate[0].MethodHash, Strata: map[string]StratumReport{}}
	for _, o := range candidate {
		if o.Provenance != "test" {
			return Report{}, fmt.Errorf("candidate observation %q is not test provenance", o.TaskID)
		}
		key := o.TaskID + "\x00" + o.Stratum
		if o.CorpusHash != corpusHash {
			return Report{}, fmt.Errorf("corpus hash changed within comparison")
		}
		if o.SourcePopulation != sourcePopulation {
			return Report{}, fmt.Errorf("source population changed within comparison")
		}
		if seen[key] {
			return Report{}, fmt.Errorf("duplicate candidate observation %q", o.TaskID)
		}
		seen[key] = true
		b, ok := base[key]
		if !ok {
			continue
		}
		if report.BaselineMethod != b.MethodHash || report.CandidateMethod != o.MethodHash {
			return Report{}, fmt.Errorf("method hash changed within population")
		}
		quality := b.Support == o.Support && b.Coverage == o.Coverage
		report.Pairs = append(report.Pairs, Pair{TaskID: o.TaskID, Baseline: b, Candidate: o, EffortDelta: o.Effort - b.Effort, QualityEqual: quality})
		report.MeanBaselineEffort += b.Effort
		report.MeanCandidateEffort += o.Effort
	}
	sort.Slice(report.Pairs, func(i, j int) bool { return report.Pairs[i].TaskID < report.Pairs[j].TaskID })
	report.Population = len(report.Pairs)
	if report.Population == 0 {
		report.Reason = "no_paired_population"
		return report, nil
	}
	if report.Population != len(base) || report.Population != len(candidate) {
		report.Reason = "incomplete_pairs"
		return report, nil
	}
	report.MeanBaselineEffort /= float64(report.Population)
	report.MeanCandidateEffort /= float64(report.Population)
	for _, pair := range report.Pairs {
		s := report.Strata[pair.Baseline.Stratum]
		s.Population++
		s.MeanBaselineEffort += pair.Baseline.Effort
		s.MeanCandidateEffort += pair.Candidate.Effort
		if s.Population == 1 {
			s.QualityPreserved = pair.QualityEqual
		} else {
			s.QualityPreserved = s.QualityPreserved && pair.QualityEqual
		}
		report.Strata[pair.Baseline.Stratum] = s
	}
	for stratum, s := range report.Strata {
		s.MeanBaselineEffort /= float64(s.Population)
		s.MeanCandidateEffort /= float64(s.Population)
		s.EffortReduced = s.MeanCandidateEffort < s.MeanBaselineEffort
		report.Strata[stratum] = s
	}
	for _, pair := range report.Pairs {
		if !pair.QualityEqual {
			report.Reason = "quality_regression"
			return report, nil
		}
	}
	if report.MeanCandidateEffort >= report.MeanBaselineEffort {
		report.Reason = "no_effort_improvement"
		return report, nil
	}
	report.Accepted = true
	report.Reason = "quality_preserved_effort_reduced"
	return report, nil
}
