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
	if frame.Precision != nil || frame.Recall != nil || frame.ResponseComplete || len(frame.Targets) != 1 || frame.Targets[0].SourceHash != sourceHash || frame.Targets[0].ConfigHash != configHash || frame.Targets[0].ResponseComplete {
		t.Fatalf("TypeScript source labels without a retained matching served response must remain rate-free: %+v", frame)
	}
	if got := calibrationForTarget(calibration, "browser-automation-studio", sourceHash, configHash); got.ScopeFrameID != "" || got.Precision != nil || got.Recall != nil {
		t.Fatalf("source/config identity without a retained matching served response must not receive scoped rates: %+v", got)
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
	if boundedFrameCount != 12 || probabilityFrameCount != 32 || completeProbabilityTargets != 31 || probabilityGroupCases != 177 {
		t.Fatalf("bounded/sample evidence frames: old=%d sampled=%d complete=%d labeled-groups=%d; want 12/32/31/177", boundedFrameCount, probabilityFrameCount, completeProbabilityTargets, probabilityGroupCases)
	}
	if sampleAggregate == nil || sampleAggregate.Cases != 177 || sampleAggregate.TruePositive != 177 || sampleAggregate.FalsePositive != 0 || sampleAggregate.FalseNegative != 0 || sampleAggregate.Precision != nil || sampleAggregate.Recall != nil || sampleAggregate.CompleteCensus || sampleAggregate.AdjudicationComplete {
		t.Fatalf("sample responder counts must be retained without publishing rates from an incomplete frame: %+v", sampleAggregate)
	}
	wantFrames := []string{"dr1-live-architecture-cartographer-census-20261116-v1", "dr1-live-architecture-cartographer-go-census-v1", "dr1-live-go-code-graph-census-20261116-v1", "dr1-live-quality-health-go-census-20261116-v1", "dr1-live-quality-health-go-post-count-autofixable-20261116-v1", "dr1-live-quality-health-go-post-shared-counts-20261116-v1", "dr1-live-quality-health-go-post-shared-parser-20261116-v1", "dr1-live-scenario-auditor-census-20261116-v1", "dr1-live-three-target-go-census-v1", "dr1-live-two-target-go-census-v1", "dr1-live-two-target-go-census-v2", "dr1-live-typescript-code-graph-census-20261116-v1"}
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
		case "dr1-live-architecture-cartographer-census-20261116-v1", "dr1-live-scenario-auditor-census-20261116-v1":
			wantCases, wantTargetCount = 1, 1
		case "dr1-live-go-code-graph-census-20261116-v1", "dr1-live-typescript-code-graph-census-20261116-v1":
			wantCases, wantTargetCount = 1, 1
		case "dr1-live-quality-health-go-census-20261116-v1":
			wantCases, wantTargetCount = 4, 1
		case "dr1-live-quality-health-go-post-shared-parser-20261116-v1":
			wantCases, wantTargetCount = 3, 1
		case "dr1-live-quality-health-go-post-count-autofixable-20261116-v1":
			wantCases, wantTargetCount = 1, 1
		case "dr1-live-quality-health-go-post-shared-counts-20261116-v1":
			wantCases, wantTargetCount = 2, 1
		case "dr1-live-three-target-go-census-v1":
			wantTargetCount = 3
		}
		if !frame.CompleteCensus || !frame.AdjudicationComplete || frame.FrameID != wantFrames[i] || len(frame.Targets) != wantTargetCount {
			t.Fatalf("bounded frame provenance is incomplete: %+v", frame)
		}
		if frame.Cases != wantCases || frame.TruePositive != wantCases || frame.FalsePositive != 0 || frame.FalseNegative != 0 || frame.TrueNegative != 0 {
			t.Fatalf("unexpected bounded group confusion matrix: %+v", frame)
		}
		if frame.ResponseComplete {
			if frame.Precision == nil || *frame.Precision != 1 || frame.Recall == nil || *frame.Recall != 1 {
				t.Fatalf("complete served-census rates for %s = precision %v, recall %v; want 1.0 each", frame.FrameID, frame.Precision, frame.Recall)
			}
		} else if frame.Precision != nil || frame.Recall != nil {
			t.Fatalf("unverified served response for %s must not publish rates: %+v", frame.FrameID, frame)
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

func TestQualityHealthCensusPrecisionIsScopedToExactServedSnapshot(t *testing.T) {
	const (
		frameID    = "dr1-live-quality-health-go-census-20261116-v1"
		sourceHash = "011323a19f8c1f0abc4e2fe29ea75ccec9d029ac031ceb81ee5e9fec28654d76"
		configHash = "1745e28e4f744c73b94cd2aa87b6b73a073d911fe5eb23ad9d40df88077eb684"
	)
	var calibration Calibration
	var bodyDetector Detector
	for _, detector := range New().Describe() {
		if detector.ID == "duplication.identical-function-body" {
			calibration = detector.Calibration
			bodyDetector = detector
			break
		}
	}
	if calibration.Precision != nil || calibration.Recall != nil || calibration.RepresentativeSample {
		t.Fatalf("one exact target census must not become a fleet estimate: %+v", calibration)
	}
	var frame *CalibrationFrame
	for i := range calibration.EvidenceFrames {
		if calibration.EvidenceFrames[i].FrameID == frameID {
			frame = &calibration.EvidenceFrames[i]
			break
		}
	}
	if frame == nil || frame.Cases != 4 || frame.TruePositive != 4 || frame.FalsePositive != 0 || frame.FalseNegative != 0 || frame.TrueNegative != 0 {
		t.Fatalf("unexpected quality-health exact-source confusion matrix: %+v", frame)
	}
	if frame.Precision == nil || *frame.Precision != 1 || frame.Recall == nil || *frame.Recall != 1 || len(frame.Targets) != 1 || frame.Targets[0].SourceHash != sourceHash || frame.Targets[0].ConfigHash != configHash || frame.Targets[0].SourceManifestSHA256 != "8c80537fc3368e0600f02e9eef3e927b1228a7b8ea293bef883304bf0a00e19a" {
		t.Fatalf("quality-health rates lost exact target identity: %+v", frame)
	}
	scoped := calibrationForTarget(calibration, "quality-health", sourceHash, configHash)
	if scoped.ScopeFrameID != frameID || scoped.Precision == nil || *scoped.Precision != 1 || scoped.Recall == nil || *scoped.Recall != 1 {
		t.Fatalf("exact target snapshot must receive scoped rates: %+v", scoped)
	}
	stale := calibrationForTarget(calibration, "quality-health", sourceHash, "changed-config")
	if stale.ScopeFrameID != "" || stale.Precision != nil || stale.Recall != nil {
		t.Fatalf("changed config must not receive the saved target rates: %+v", stale)
	}
	for _, c := range corpusCases() {
		if c.SamplingFrame != frameID {
			continue
		}
		if hit := len(bodyDetector.detect(c.input())) > 0; hit != c.Expected {
			t.Errorf("live case %s prediction=%v, expected=%v", c.ID, hit, c.Expected)
		}
	}
}

func TestPostMigrationCensusesAreScopedToExactServedSnapshots(t *testing.T) {
	cases := []struct {
		name, scenario, frameID, sourceHash, configHash, manifest string
		groups                                                    int
	}{
		{
			name: "shared parser", scenario: "quality-health", configHash: "1745e28e4f744c73b94cd2aa87b6b73a073d911fe5eb23ad9d40df88077eb684", frameID: "dr1-live-quality-health-go-post-shared-parser-20261116-v1",
			sourceHash: "1ed167fc84b18f3438f07bf1045a1f6025b052591e7a290fe3242c54f29c615a",
			manifest:   "62c374ca1aa72fb37ed91907bff63b88c7aebea5b6f5833a410f74f8a4c66986",
			groups:     3,
		},
		{
			name: "shared finding counter", scenario: "quality-health", configHash: "1745e28e4f744c73b94cd2aa87b6b73a073d911fe5eb23ad9d40df88077eb684", frameID: "dr1-live-quality-health-go-post-shared-counts-20261116-v1",
			sourceHash: "67d0f22b6dec6be68d4e59e03fb9909d57fd77b601fbef1c37979358397fda54",
			manifest:   "003cb8b73192e406f10f3e211a6e947fe0b5d8ce605d153893f8ba04327dbf1b",
			groups:     2,
		},
		{
			name: "autofixable counter", scenario: "quality-health", configHash: "1745e28e4f744c73b94cd2aa87b6b73a073d911fe5eb23ad9d40df88077eb684", frameID: "dr1-live-quality-health-go-post-count-autofixable-20261116-v1",
			sourceHash: "74613fd4bb489cd25fd95d632b1fbd1fa9fb338d4cd5b8dff7b65d3fa1bff78e",
			manifest:   "a43b8d4eaeca73ac5959b49bc5c6776fd47130f382592db8db10c90c618656cd",
			groups:     1,
		},
		{
			name: "architecture cartographer", scenario: "architecture-cartographer", configHash: "d5512b67cd451ca0ca639a3ac2579132ea92daa37e9d94c1e6aa4a07216f8310",
			frameID: "dr1-live-architecture-cartographer-census-20261116-v1", sourceHash: "a8025ee84bfd244a4184cec4830536a592c334e24e2698f5b2eae6d2b7815ce5",
			manifest: "d8ef0b960f40d45c4945dd25a96d09e7dda67f240ae5b83c6fd7a5ad52f7c6d5", groups: 1,
		},
		{
			name: "scenario auditor", scenario: "scenario-auditor", configHash: "e45579ee264c4b427b3bbd94c936c4e312e3ad79250fc5cd9080abe53f17c312",
			frameID: "dr1-live-scenario-auditor-census-20261116-v1", sourceHash: "e6d39e8ee39705dd25b043ab1252316474ab501cf5a118e0110f59cec7393bff",
			manifest: "a34542e01b1e4ad04f4340fb853074e85dfc70b5d76542cd3f15cf6e19fe681e", groups: 1,
		},
		{
			name: "go code graph", scenario: "go-code-graph", configHash: "569fa7fcb6cf86b77b05432eb383a9a3e16c6590acebdb1994a93bc05f679864",
			frameID: "dr1-live-go-code-graph-census-20261116-v1", sourceHash: "d944c55ef0a15c0ca1c455e086d40810b6dea49246f243e44abb6099b0d8ef91",
			manifest: "dcd7b1aa9c35c6b418c5911179ce780165b4a355401e53c73f03249c9b14d196", groups: 1,
		},
		{
			name: "typescript code graph Go CLI", scenario: "typescript-code-graph", configHash: "5d88a5dad6499e5f21004f058e02bc27cf0c36a69ed246914e5097da06f665ef",
			frameID: "dr1-live-typescript-code-graph-census-20261116-v1", sourceHash: "76b2e9f9bfe45f4fdf4fc6ef8f8c4f9eea11e6a23d7db54784f9165a7857b72d",
			manifest: "5924ae62e47435d4bc325fca6ba4538b56e79d223c08cfb037c5f681005e1f48", groups: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calibration Calibration
			var detector Detector
			for _, candidate := range New().Describe() {
				if candidate.ID == "duplication.identical-function-body" {
					calibration = candidate.Calibration
					detector = candidate
					break
				}
			}
			if calibration.Precision != nil || calibration.Recall != nil || calibration.RepresentativeSample {
				t.Fatalf("exact target censuses must not become fleet estimates: %+v", calibration)
			}
			var frame *CalibrationFrame
			for i := range calibration.EvidenceFrames {
				if calibration.EvidenceFrames[i].FrameID == tc.frameID {
					frame = &calibration.EvidenceFrames[i]
					break
				}
			}
			if frame == nil || frame.Cases != tc.groups || frame.TruePositive != tc.groups || frame.FalsePositive != 0 || frame.FalseNegative != 0 || frame.TrueNegative != 0 {
				t.Fatalf("unexpected exact-source confusion matrix: %+v", frame)
			}
			if frame.Precision == nil || *frame.Precision != 1 || frame.Recall == nil || *frame.Recall != 1 || len(frame.Targets) != 1 || frame.Targets[0].SourceHash != tc.sourceHash || frame.Targets[0].ConfigHash != tc.configHash || frame.Targets[0].SourceManifestSHA256 != tc.manifest {
				t.Fatalf("rates lost exact target identity: %+v", frame)
			}
			scoped := calibrationForTarget(calibration, tc.scenario, tc.sourceHash, tc.configHash)
			if scoped.ScopeFrameID != tc.frameID || scoped.Precision == nil || *scoped.Precision != 1 || scoped.Recall == nil || *scoped.Recall != 1 {
				t.Fatalf("exact target must receive scoped rates: %+v", scoped)
			}
			stale := calibrationForTarget(calibration, tc.scenario, tc.sourceHash, "changed-config")
			if stale.ScopeFrameID != "" || stale.Precision != nil || stale.Recall != nil {
				t.Fatalf("changed config must not receive saved rates: %+v", stale)
			}
			for _, c := range corpusCases() {
				if c.SamplingFrame == tc.frameID {
					if hit := len(detector.detect(c.input())) > 0; hit != c.Expected {
						t.Errorf("live case %s prediction=%v, expected=%v", c.ID, hit, c.Expected)
					}
				}
			}
		})
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
