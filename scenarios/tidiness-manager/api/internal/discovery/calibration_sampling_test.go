package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestIncompleteProbabilitySampleKeepsGlobalMetricsUnknown(t *testing.T) {
	corpus := readCorpus()
	if corpus.Sampling.FrameID == "" || corpus.Sampling.Selection == "" || corpus.Sampling.Representative {
		t.Fatalf("sampling contract must identify an incomplete probability sample without claiming fleet representativeness: %+v", corpus.Sampling)
	}
	if corpus.Sampling.PopulationID == "" || corpus.Sampling.RandomSeed == "" || corpus.Sampling.AdjudicationComplete {
		t.Fatalf("probability frame identity must be retained while incomplete adjudication keeps global rates unknown: %+v", corpus.Sampling)
	}

	for _, detector := range New().Describe() {
		calibration := detector.Calibration
		if calibration.HistoricalCases == 0 {
			continue
		}
		if calibration.SamplingFrame != corpus.Sampling.FrameID || calibration.SamplingSelection != corpus.Sampling.Selection ||
			calibration.SamplingUnit != corpus.Sampling.SamplingUnit || calibration.SamplingMethod != corpus.Sampling.SelectionMethod ||
			calibration.AdjudicationProtocol != corpus.Sampling.AdjudicationProtocol || calibration.AdjudicationComplete {
			t.Errorf("%s lost sampling provenance: %+v", detector.ID, calibration)
		}
		if calibration.Precision != nil || calibration.Recall != nil {
			t.Errorf("%s published rates from selected cases: %+v", detector.ID, calibration)
		}
		if got := calibration.TruePositive + calibration.FalsePositive + calibration.FalseNegative + calibration.TrueNegative; got != calibration.HistoricalCases {
			t.Errorf("%s confusion denominator %d != historical replay count %d", detector.ID, got, calibration.HistoricalCases)
		}
	}
}

func TestRegistryCarriesHistoricalRunAndHuntAuditWithoutInventingLabels(t *testing.T) {
	const evidenceRef = "programme-four-20261110/dr1-historical-detector-evidence-v2.json"
	const evidenceSHA256 = "15b312062b5eb9f68702eb95c68cb9b68bb116c6573e8e8e176d88ef3ac14b4b"
	for _, detector := range New().Describe() {
		audit := detector.Calibration.HistoricalAudit
		if audit == nil || audit.EvidenceRef != evidenceRef || audit.EvidenceSHA256 != evidenceSHA256 {
			t.Errorf("%s lost the retained historical audit identity: %+v", detector.ID, audit)
			continue
		}
		if audit.RunsReviewed != 48 || audit.HuntEntriesReviewed != 110 || audit.DetectorsAudited != 5 || audit.ReductionDetectorsAudited != 4 || audit.ArchivedPredictionsJoinable != 0 {
			t.Errorf("%s historical audit coverage changed: %+v", detector.ID, audit)
		}
		if detector.ID != "standards.scenario-auditor" && (detector.Calibration.Precision != nil || detector.Calibration.Recall != nil) {
			t.Errorf("%s inferred detector metrics from an archive with no joinable predictions: %+v", detector.ID, detector.Calibration)
		}
		wantSourceLabels := map[string]int{
			"duplication.identical-function-body": 10,
			"migration.unbound-cli-handler":       8,
		}[detector.ID]
		if audit.SourceLabelCount != wantSourceLabels {
			t.Errorf("%s source-identifiable labels=%d, want %d: %+v", detector.ID, audit.SourceLabelCount, wantSourceLabels, audit)
		}
		if wantSourceLabels > 0 && (len(audit.DetectorContractSHA256) != 64 || len(audit.SourceManifestSHA256) != 64) {
			t.Errorf("%s omitted its source/config identity manifest: %+v", detector.ID, audit)
		}
	}
}

func TestHistoricalSourceReplaysHaveExplicitLanguageAndBoundaryProvenance(t *testing.T) {
	detectors := map[string]Detector{}
	for _, detector := range New().Describe() {
		detectors[detector.ID] = detector
	}
	wantCases := map[string]int{
		"duplication.identical-function-body": 9,
		"migration.unbound-cli-handler":       8,
	}
	gotCases := map[string]int{}
	corpus := readCorpus()
	for detectorID, contract := range corpus.HistoricalDetectorChecks {
		digest := sha256.Sum256([]byte(contract.Contract))
		if hex.EncodeToString(digest[:]) != contract.SHA256 {
			t.Errorf("%s detector contract hash does not match its declared contract", detectorID)
		}
	}
	for _, c := range corpusCases() {
		if c.Origin == "repository-beforeimage" || c.Origin == "source-replay-boundary" {
			contract, ok := corpus.HistoricalDetectorChecks[c.DetectorID]
			if !ok || c.SourceSnapshotSHA256 == "" || len(c.SourceSnapshotSHA256) != 64 || c.DetectorContractSHA256 != contract.SHA256 || len(contract.SHA256) != 64 || c.Revision == "" {
				t.Errorf("historical source replay %s lacks exact source/revision/detector-contract identity: %+v", c.ID, c)
			}
		}
		if c.Origin != "repository-beforeimage" {
			continue
		}
		language := c.Language
		if language == "" {
			language = "go"
		}
		if language != "go" {
			t.Errorf("historical source replay %s has language %q; expected explicit Go input", c.ID, language)
		}
		gotCases[c.DetectorID]++
		detector, ok := detectors[c.DetectorID]
		if !ok || detector.detect == nil {
			t.Errorf("historical replay %s refers to an unavailable detector %q", c.ID, c.DetectorID)
			continue
		}
		predicted := len(detector.detect(c.input())) > 0
		if predicted != c.Expected {
			t.Errorf("historical replay %s predicted %v, expected source label %v", c.ID, predicted, c.Expected)
		}
		if c.EvidenceRef == "" || c.OriginalPrediction != "unknown" {
			t.Errorf("historical replay %s must link retained evidence and keep its original prediction unknown: evidence_ref=%q original_prediction=%q", c.ID, c.EvidenceRef, c.OriginalPrediction)
		}
	}
	for detectorID, want := range wantCases {
		if gotCases[detectorID] != want {
			t.Errorf("%s historical source replay count=%d, want %d", detectorID, gotCases[detectorID], want)
		}
		calibration := detectors[detectorID].Calibration
		if calibration.HistoricalCases != want || calibration.TruePositive != want || calibration.FalsePositive != 0 || calibration.FalseNegative != 0 || calibration.TrueNegative != 0 {
			t.Errorf("%s selected historical replay counts disagree with its source labels: %+v", detectorID, calibration)
		}
		if calibration.Precision != nil || calibration.Recall != nil {
			t.Errorf("%s published rates from selected historical cases: %+v", detectorID, calibration)
		}
	}
	if gotCases["duplication.identical-function-body"]+gotCases["migration.unbound-cli-handler"] != 17 {
		t.Errorf("historical source replay total=%d, want 17", gotCases["duplication.identical-function-body"]+gotCases["migration.unbound-cli-handler"])
	}

	var boundaryFound bool
	for _, c := range corpusCases() {
		if c.ID != "agent-manager-json-types-dupl-blocks-are-not-identical-function-bodies" {
			continue
		}
		boundaryFound = true
		if c.DetectorID != "duplication.identical-function-body" || c.Language != "go" || c.Expected ||
			c.Origin != "source-replay-boundary" || c.EvidenceRef != "dr1-historical-detector-evidence-v2.json#/scope_boundary_labels/labels/0" || c.OriginalPrediction != "unknown" {
			t.Errorf("historical clone-block boundary lost its scoped negative or evidence provenance: %+v", c)
			continue
		}
		if len(detectors[c.DetectorID].detect(c.input())) != 0 {
			t.Error("full-function detector predicted a duplicate from distinct enclosing functions in the dupl boundary case")
		}
	}
	if !boundaryFound {
		t.Fatal("missing source-replayed historical clone-block scope-boundary negative")
	}
}

func TestTypeScriptCensusRatesRemainBoundToSavedSourceAndConfig(t *testing.T) {
	const (
		frameID    = "dr1-go-stratified-sample-browser-automation-studio-typescript-census-v1"
		sourceHash = "c22d16347c2e89c32ef75b4ad39278b78e276e7fdbab27f041897b094ae1c5b2"
		configHash = "59363f08aabb9623a62055514ab7f3640ed07d433bc224e0efbd8292eab1e230"
	)
	var calibration Calibration
	for _, detector := range New().Describe() {
		if detector.ID == "duplication.identical-function-body.typescript" {
			calibration = detector.Calibration
			break
		}
	}
	if calibration.Precision != nil || calibration.Recall != nil {
		t.Fatalf("one exact TypeScript target must not become a fleet estimate: %+v", calibration)
	}
	var frame *CalibrationFrame
	for i := range calibration.EvidenceFrames {
		if calibration.EvidenceFrames[i].FrameID == frameID {
			frame = &calibration.EvidenceFrames[i]
			break
		}
	}
	if frame == nil || frame.Cases != 19 || frame.TruePositive != 17 || frame.FalsePositive != 0 || frame.FalseNegative != 2 || frame.TrueNegative != 0 {
		t.Fatalf("unexpected TypeScript target confusion matrix: %+v", frame)
	}
	if frame.Precision == nil || *frame.Precision != 1 || frame.Recall == nil || *frame.Recall != 17.0/19.0 || len(frame.Targets) != 1 || frame.Targets[0].SourceHash != sourceHash || frame.Targets[0].ConfigHash != configHash {
		t.Fatalf("TypeScript metrics lost their exact saved source/config identity: %+v", frame)
	}
	if got := calibrationForTarget(calibration, "browser-automation-studio", sourceHash, configHash); got.ScopeFrameID != frameID || got.Precision == nil || *got.Precision != 1 || got.Recall == nil || *got.Recall != 17.0/19.0 {
		t.Fatalf("exact saved target must receive scoped rates: %+v", got)
	}
	if got := calibrationForTarget(calibration, "browser-automation-studio", sourceHash, "changed-config"); got.ScopeFrameID != "" || got.Precision != nil || got.Recall != nil {
		t.Fatalf("stale config must not receive the old TypeScript rates: %+v", got)
	}
}

func TestRepresentativeFrameRatesUseConfusionDenominators(t *testing.T) {
	calibration := Calibration{
		RepresentativeSample: true,
		SamplingFrame:        "frame-v1",
		SamplingSelection:    "equal-probability-simple-random-sample",
		SamplingPopulation:   "sha256:population-digest",
		SamplingUnit:         "scenario-target",
		SamplingMethod:       "simple-random-sample-without-replacement",
		SamplingSeed:         "seed-20261110",
		AdjudicationProtocol: "two-reviewer-source-replay-v1",
		AdjudicationComplete: true,
		TruePositive:         3,
		FalsePositive:        1,
		FalseNegative:        2,
		TrueNegative:         4,
	}
	calibration.setRates()
	if calibration.Precision == nil || *calibration.Precision != .75 {
		t.Fatalf("precision = %v, want 3/(3+1)=0.75", calibration.Precision)
	}
	if calibration.Recall == nil || *calibration.Recall != .6 {
		t.Fatalf("recall = %v, want 3/(3+2)=0.6", calibration.Recall)
	}
}

func TestRepresentativeFlagWithoutSamplingAndAdjudicationProvenanceKeepsMetricsUnknown(t *testing.T) {
	calibration := Calibration{RepresentativeSample: true, TruePositive: 3, FalsePositive: 1, FalseNegative: 2}
	calibration.setRates()
	if calibration.Precision != nil || calibration.Recall != nil {
		t.Fatalf("representative flag alone must not authorize rates: %+v", calibration)
	}
}

func TestLiveSourceCensusRatesRemainScopedToTheirDeclaredPopulation(t *testing.T) {
	corpus := readCorpus()
	var bodyDetector, migrationDetector Detector
	for _, detector := range New().Describe() {
		switch detector.ID {
		case "duplication.identical-function-body":
			bodyDetector = detector
		case "migration.unbound-cli-handler":
			migrationDetector = detector
		}
	}
	if bodyDetector.ID == "" {
		t.Fatal("missing Go duplicate-body detector")
	}
	calibration := bodyDetector.Calibration
	if calibration.Precision != nil || calibration.Recall != nil || calibration.RepresentativeSample {
		t.Fatalf("bounded source censuses must not become fleet-level estimates: %+v", calibration)
	}
	var boundedFrameCount, probabilityFrameCount, completeProbabilityTargets, probabilityGroupCases int
	var sampleAggregate *CalibrationFrame
	for _, frame := range calibration.EvidenceFrames {
		if strings.HasPrefix(frame.FrameID, "dr1-live-") {
			boundedFrameCount++
		}
		if strings.HasPrefix(frame.FrameID, "dr1-probability-go-target-") {
			probabilityFrameCount++
			if frame.ResponseComplete {
				completeProbabilityTargets++
			}
			probabilityGroupCases += frame.Cases
			if frame.AdjudicationComplete && frame.Cases > 0 && (frame.Precision == nil || *frame.Precision != 1 || frame.Recall == nil || *frame.Recall != 1) {
				t.Errorf("complete probability-target frame has unexpected measured rate: %+v", frame)
			}
			if !frame.ResponseComplete && (frame.Precision != nil || frame.Recall != nil) {
				t.Errorf("unavailable served response published a rate: %+v", frame)
			}
		}
		if frame.FrameID == corpus.Sampling.FrameID {
			copy := frame
			sampleAggregate = &copy
		}
	}
	if boundedFrameCount != 4 || probabilityFrameCount != 32 || completeProbabilityTargets != 31 || probabilityGroupCases != 177 {
		t.Fatalf("bounded/sample evidence frames: old=%d sampled=%d complete=%d labeled-groups=%d; want 4/32/31/177", boundedFrameCount, probabilityFrameCount, completeProbabilityTargets, probabilityGroupCases)
	}
	if sampleAggregate == nil || sampleAggregate.Cases != 177 || sampleAggregate.TruePositive != 177 || sampleAggregate.FalsePositive != 0 || sampleAggregate.FalseNegative != 0 || sampleAggregate.Precision != nil || sampleAggregate.Recall != nil || sampleAggregate.CompleteCensus || sampleAggregate.AdjudicationComplete {
		t.Fatalf("sample responder counts must be retained without publishing rates from an incomplete frame: %+v", sampleAggregate)
	}
	wantFrames := []string{"dr1-live-architecture-cartographer-go-census-v1", "dr1-live-three-target-go-census-v1", "dr1-live-two-target-go-census-v1", "dr1-live-two-target-go-census-v2"}
	var boundedFrames []CalibrationFrame
	for _, frame := range calibration.EvidenceFrames {
		if strings.HasPrefix(frame.FrameID, "dr1-live-") {
			boundedFrames = append(boundedFrames, frame)
		}
	}
	for i, frame := range boundedFrames {
		wantCases, wantTargetCount := 36, 2
		switch frame.FrameID {
		case "dr1-live-architecture-cartographer-go-census-v1":
			wantCases, wantTargetCount = 9, 1
		case "dr1-live-three-target-go-census-v1":
			wantTargetCount = 3
		}
		if !frame.CompleteCensus || !frame.AdjudicationComplete || frame.FrameID != wantFrames[i] || len(frame.Targets) != wantTargetCount {
			t.Fatalf("bounded frame provenance is incomplete: %+v", frame)
		}
		if frame.Cases != wantCases || frame.TruePositive != wantCases || frame.FalsePositive != 0 || frame.FalseNegative != 0 || frame.TrueNegative != 0 {
			t.Fatalf("unexpected bounded group confusion matrix: %+v", frame)
		}
		if frame.Precision == nil || *frame.Precision != 1 || frame.Recall == nil || *frame.Recall != 1 {
			t.Fatalf("bounded complete-census rates = precision %v, recall %v; want 1.0 each", frame.Precision, frame.Recall)
		}
		if frame.FrameID != "dr1-live-two-target-go-census-v1" {
			for _, target := range frame.Targets {
				if target.SourceFileCount == 0 || target.SourceManifestSHA256 == "" {
					t.Fatalf("latest frame lacks per-file source manifest provenance: %+v", target)
				}
			}
		}
		if frame.FrameID == "dr1-live-three-target-go-census-v1" {
			var zeroGroupTarget bool
			for _, target := range frame.Targets {
				if target.Scenario == "agent-metareasoning-manager" {
					zeroGroupTarget = target.EligibleFunctions == 96 && target.SourceGroups == 0 && target.ServedGroups == 0 && target.SourceFileCount == 14 && target.SourceManifestSHA256 != "" && target.Applicability != ""
				}
			}
			if !zeroGroupTarget {
				t.Fatalf("three-target frame must retain an applicable, fully enumerated zero-group target: %+v", frame.Targets)
			}
		}
	}
	if calibration.TargetNegativeCases != 1 || calibration.RegressionFailures != 0 {
		t.Fatalf("target applicability negative must replay separately and pass: %+v", calibration)
	}
	if calibration.Precision != nil || calibration.Recall != nil || calibration.RepresentativeSample {
		t.Fatalf("one unresolved sampled target must keep fleet rates and ranking precision unknown: %+v", calibration)
	}
	negativeBoundary := false
	for _, c := range corpusCases() {
		if c.ID == "agent-manager-json-types-dupl-blocks-are-not-identical-function-bodies" {
			negativeBoundary = c.DetectorID == "duplication.identical-function-body" && !c.Expected && c.Origin == "source-replay-boundary"
		}
	}
	if !negativeBoundary {
		t.Fatal("source-backed dupl scope-boundary negative must remain labeled outside live census metrics")
	}
	var targetNegative bool
	for _, c := range corpusCases() {
		if c.ID == "live-v3-agent-metareasoning-manager-zero-go-body-groups" {
			targetNegative = !c.Expected && c.Origin == "live-source-target-negative" && len(c.Symbols) == 96 && c.SamplingFrame == "dr1-live-three-target-go-census-v1"
		}
	}
	if !targetNegative {
		t.Fatal("complete zero-group target must be retained as an applicability negative outside group rates")
	}
	if migrationDetector.ID == "" || len(migrationDetector.Calibration.EvidenceFrames) != 0 {
		t.Fatalf("Go-only census must not be presented as migration evidence: %+v", migrationDetector.Calibration.EvidenceFrames)
	}
}

func TestBoundedFrameWithoutFullProvenanceKeepsRatesUnknown(t *testing.T) {
	frame := CalibrationFrame{
		FrameID: "frame-v1", PopulationID: "sha256:population", Population: "finite frame",
		CompleteCensus: true, AdjudicationComplete: true,
		TruePositive: 2,
	}
	frame.setRates()
	if frame.Precision != nil || frame.Recall != nil {
		t.Fatalf("incomplete census provenance must not publish scoped rates: %+v", frame)
	}
}
