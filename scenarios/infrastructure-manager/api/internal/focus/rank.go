// Package focus contains the deterministic ranking and efficacy rules for the
// read-only next-finding surface. It has no actuation seam by construction.
package focus

import (
	"context"
	"sort"
	"time"
)

type Stage int

const (
	StageIntegrity Stage = iota
	StageSubstrate
	StageAvailability
	StageEfficiency
	StageMeasurement
)

type Finding struct {
	ID             string
	Source         string
	CellRef        string
	SensorRef      string
	Title          string
	Message        string
	Stage          Stage
	Severity       int
	TrustValid     bool
	ExpectedReturn string
}

type RankedFinding struct {
	Finding
	RankExplanation string
	Rank            int
}

type GapSource struct {
	ID           string
	Label        string
	Available    bool
	Reason       string
	FindingCount int
}

type Source interface {
	Read(context.Context) ([]Finding, []GapSource, error)
}

type EfficacyRecord struct {
	FindingID      string
	SensorRef      string
	ExpectedReturn string
	ObservedReturn string
	Verdict        EfficacyVerdict
	ObservedAt     time.Time
}

type Repository interface {
	SaveFindings(context.Context, []RankedFinding) error
	Efficacy(context.Context, string) ([]EfficacyRecord, error)
}

type Service struct {
	Source     Source
	Repository Repository
	Now        func() time.Time
}

func (s *Service) Next(ctx context.Context, limit int) ([]RankedFinding, []GapSource, bool, bool) {
	if s.Source == nil {
		return nil, []GapSource{{ID: "focus", Label: "focus sources", Available: false, Reason: "focus sources are not configured"}}, false, true
	}
	findings, sources, err := s.Source.Read(ctx)
	if err != nil {
		return nil, []GapSource{{ID: "focus", Label: "focus sources", Available: false, Reason: err.Error()}}, false, true
	}
	ranked := Rank(findings)
	if s.Repository != nil && len(ranked) > 0 {
		if err := s.Repository.SaveFindings(ctx, ranked); err != nil {
			sources = append(sources, GapSource{ID: "focus-persistence", Label: "focus persistence", Available: false, Reason: err.Error()})
		}
	}
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}
	allUnavailable := len(sources) > 0
	for _, source := range sources {
		if source.Available {
			allUnavailable = false
			break
		}
	}
	return ranked, sources, len(ranked) == 0, allUnavailable
}

func (s *Service) Efficacy(ctx context.Context, findingID string) ([]EfficacyRecord, error) {
	if s.Repository == nil {
		return []EfficacyRecord{{FindingID: findingID, Verdict: EfficacyUnmeasurable, ObservedReturn: "finding efficacy history is not persisted", ObservedAt: s.now()}}, nil
	}
	records, err := s.Repository.Efficacy(ctx, findingID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []EfficacyRecord{{FindingID: findingID, Verdict: EfficacyUnmeasurable, ObservedReturn: "no completed work join exists", ObservedAt: s.now()}}, nil
	}
	return records, nil
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func Rank(findings []Finding) []RankedFinding {
	ordered := append([]Finding(nil), findings...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Stage != ordered[j].Stage {
			return ordered[i].Stage < ordered[j].Stage
		}
		if ordered[i].Severity != ordered[j].Severity {
			return ordered[i].Severity > ordered[j].Severity
		}
		if ordered[i].Source != ordered[j].Source {
			return ordered[i].Source < ordered[j].Source
		}
		return ordered[i].ID < ordered[j].ID
	})
	out := make([]RankedFinding, 0, len(ordered))
	for i, finding := range ordered {
		out = append(out, RankedFinding{Finding: finding, Rank: i + 1, RankExplanation: StageExplanation(finding.Stage)})
	}
	return out
}

func StageExplanation(stage Stage) string {
	switch stage {
	case StageIntegrity:
		return "sensor-channel integrity outranks plant condition"
	case StageSubstrate:
		return "host/process substrate precedes capability work"
	case StageAvailability:
		return "capability availability follows substrate integrity"
	case StageEfficiency:
		return "efficiency and performance follow availability"
	default:
		return "measurement improvement follows operational findings"
	}
}

type EfficacyVerdict string

const (
	EfficacyMoved        EfficacyVerdict = "MOVED"
	EfficacyDidNotMove   EfficacyVerdict = "DID_NOT_MOVE"
	EfficacyAwaitingWork EfficacyVerdict = "AWAITING_WORK"
	EfficacyUnmeasurable EfficacyVerdict = "UNMEASURABLE"
)

func EvaluateEfficacy(expected, observed string, measurable bool, workComplete bool) EfficacyVerdict {
	if !workComplete {
		return EfficacyAwaitingWork
	}
	if !measurable {
		return EfficacyUnmeasurable
	}
	if expected != "" && expected == observed {
		return EfficacyMoved
	}
	return EfficacyDidNotMove
}
