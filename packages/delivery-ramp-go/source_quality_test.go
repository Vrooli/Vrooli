package deliveryramp

import "testing"

func TestSourceQualityNeverImprovesByDroppingAdverseSignals(t *testing.T) {
	bad := EvaluateSourceQuality(SourceQualitySample{ClosureComplete: true, Reproducible: true, CleanBuild: GatePassed, PolicyEscapes: 1, ExportFailed: true})
	good := EvaluateSourceQuality(SourceQualitySample{ClosureComplete: true, Reproducible: true, CleanBuild: GatePassed})
	if bad.FloorPassed || good.Score <= bad.Score {
		t.Fatalf("bad=%+v good=%+v", bad, good)
	}
}
