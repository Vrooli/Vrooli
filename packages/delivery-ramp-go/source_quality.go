package deliveryramp

// SourceQualitySample is an auditable, comparable source-export measurement.
// Attempts and operator adoption are intentionally separate denominators.
type SourceQualitySample struct {
	ClosureComplete bool
	Reproducible    bool
	PolicyEscapes   int
	CleanBuild      GateDisposition
	StaleEvidence   bool
	ExportFailed    bool
	OperatorAttempt bool
}

type SourceQualityVerdict struct {
	FloorPassed bool
	Score       int
	Failures    []string
}

func EvaluateSourceQuality(s SourceQualitySample) SourceQualityVerdict {
	v := SourceQualityVerdict{FloorPassed: true, Score: 100}
	if !s.ClosureComplete {
		v.FloorPassed = false
		v.Score -= 30
		v.Failures = append(v.Failures, "closure incomplete")
	}
	if !s.Reproducible {
		v.FloorPassed = false
		v.Score -= 25
		v.Failures = append(v.Failures, "reproducibility failed")
	}
	if s.PolicyEscapes > 0 {
		v.FloorPassed = false
		v.Score -= 25
		v.Failures = append(v.Failures, "policy escapes detected")
	}
	if s.CleanBuild != GatePassed {
		v.FloorPassed = false
		v.Score -= 15
		v.Failures = append(v.Failures, "clean build not passed")
	}
	if s.StaleEvidence {
		v.Score -= 5
		v.Failures = append(v.Failures, "evidence stale")
	}
	if s.ExportFailed {
		v.Score -= 5
		v.Failures = append(v.Failures, "export failed")
	}
	if v.Score < 0 {
		v.Score = 0
	}
	return v
}
