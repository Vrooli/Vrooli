package discovery

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"sort"
)

//go:embed corpus.json
var labelledCorpus []byte

type corpusCase struct {
	ID                     string `json:"id"`
	DetectorID             string `json:"detector_id"`
	Language               string `json:"language"`
	Expected               bool   `json:"expected"`
	Origin                 string `json:"origin"`
	Source                 string `json:"source"`
	EvidenceRef            string `json:"evidence_ref,omitempty"`
	OriginalPrediction     string `json:"original_prediction,omitempty"`
	Reason                 string `json:"reason"`
	SamplingFrame          string `json:"sampling_frame,omitempty"`
	SamplingPopulationID   string `json:"sampling_population_id,omitempty"`
	SourceHash             string `json:"source_hash,omitempty"`
	ConfigHash             string `json:"config_hash,omitempty"`
	SourceManifestSHA256   string `json:"source_manifest_sha256,omitempty"`
	Predicted              *bool  `json:"predicted,omitempty"`
	AggregateSamplingFrame string `json:"aggregate_sampling_frame,omitempty"`
	Symbols                []struct {
		Name, Path, Package, Receiver, Signature, BodyHash, TypeExpression string
		Start, End, BodyLines, Occurrences                                 int
		Exported                                                           bool
	} `json:"symbols"`
}

type corpusFrameTarget = CalibrationTarget

type corpusFrame struct {
	DetectorID           string              `json:"detector_id,omitempty"`
	PopulationID         string              `json:"population_id"`
	Population           string              `json:"population"`
	CompleteCensus       bool                `json:"complete_census"`
	SamplingUnit         string              `json:"sampling_unit"`
	SelectionMethod      string              `json:"selection_method"`
	RandomSeed           string              `json:"random_seed"`
	AdjudicationProtocol string              `json:"adjudication_protocol"`
	AdjudicationComplete bool                `json:"adjudication_complete"`
	MeasurementMethod    string              `json:"measurement_method,omitempty"`
	ResponseComplete     bool                `json:"response_complete,omitempty"`
	GeneralizationScope  string              `json:"generalization_scope"`
	Targets              []corpusFrameTarget `json:"targets"`
}

type corpusDocumentSampling struct {
	FrameID              string `json:"frame_id"`
	Selection            string `json:"selection"`
	Representative       bool   `json:"representative"`
	PopulationID         string `json:"population_id"`
	SamplingUnit         string `json:"sampling_unit"`
	SelectionMethod      string `json:"selection_method"`
	RandomSeed           string `json:"random_seed"`
	AdjudicationProtocol string `json:"adjudication_protocol"`
	AdjudicationComplete bool   `json:"adjudication_complete"`
	Description          string `json:"description"`
}

type corpusDocument struct {
	Sampling corpusDocumentSampling `json:"sampling"`
	Frames   map[string]corpusFrame `json:"frames,omitempty"`
	Cases    []corpusCase           `json:"cases"`
}

func readCorpus() corpusDocument {
	var corpus corpusDocument
	if err := json.Unmarshal(labelledCorpus, &corpus); err != nil {
		panic(err)
	}
	return corpus
}

func corpusCases() []corpusCase {
	return readCorpus().Cases
}

func (c corpusCase) input() []symbol {
	symbols := make([]symbol, 0, len(c.Symbols))
	language := c.Language
	if language == "" {
		language = "go" // Existing historical corpus cases predate language tags.
	}
	for _, s := range c.Symbols {
		symbols = append(symbols, symbol{language: language, name: s.Name, path: s.Path, pkg: s.Package, receiver: s.Receiver, signature: s.Signature, bodyHash: s.BodyHash, typeExpression: s.TypeExpression, start: s.Start, end: s.End, bodyLines: s.BodyLines, nameOccurrences: s.Occurrences, exported: s.Exported})
	}
	return symbols
}

func (r *Registry) calibrate() {
	digest := sha256.Sum256(labelledCorpus)
	corpus := readCorpus()
	for i := range r.detectors {
		detector := &r.detectors[i]
		calibration := Calibration{Basis: "uncalibrated", CorpusVersion: "sha256:" + hex.EncodeToString(digest[:])}
		if detector.detect == nil {
			// External policy evidence is not a prediction of removable code.
			// Keep its calibration explicitly not-applicable instead of treating
			// absent reduction labels as zero precision.
			detector.Calibration = Calibration{Basis: "not-applicable: policy findings are not reduction predictions", CorpusVersion: calibration.CorpusVersion}
			continue
		}
		frameCounts := map[string]*CalibrationFrame{}
		for frameID, frame := range corpus.Frames {
			if frame.DetectorID != detector.ID {
				continue
			}
			frameCounts[frameID] = &CalibrationFrame{
				FrameID: frameID, PopulationID: frame.PopulationID, Population: frame.Population,
				CompleteCensus: frame.CompleteCensus, SamplingUnit: frame.SamplingUnit,
				SelectionMethod: frame.SelectionMethod, RandomSeed: frame.RandomSeed,
				AdjudicationProtocol: frame.AdjudicationProtocol, AdjudicationComplete: frame.AdjudicationComplete,
				MeasurementMethod: frame.MeasurementMethod, ResponseComplete: frame.ResponseComplete,
				GeneralizationScope: frame.GeneralizationScope, Targets: frame.Targets,
			}
		}
		for _, c := range corpus.Cases {
			if c.DetectorID != detector.ID {
				continue
			}
			hit := len(detector.detect(c.input())) > 0
			if c.Origin == "live-source-census" {
				frame, exists := corpus.Frames[c.SamplingFrame]
				if !exists || c.SamplingPopulationID != frame.PopulationID {
					calibration.RegressionCases++
					calibration.RegressionFailures++
					continue
				}
				result := frameCounts[c.SamplingFrame]
				if result == nil {
					result = &CalibrationFrame{
						FrameID: c.SamplingFrame, PopulationID: frame.PopulationID, Population: frame.Population,
						CompleteCensus: frame.CompleteCensus, SamplingUnit: frame.SamplingUnit,
						SelectionMethod: frame.SelectionMethod, RandomSeed: frame.RandomSeed,
						AdjudicationProtocol: frame.AdjudicationProtocol, AdjudicationComplete: frame.AdjudicationComplete,
						GeneralizationScope: frame.GeneralizationScope, Targets: frame.Targets,
					}
					frameCounts[c.SamplingFrame] = result
				}
				result.Cases++
				predicted := hit
				if c.Predicted != nil {
					predicted = *c.Predicted
				}
				countConfusion(&result.TruePositive, &result.FalsePositive, &result.FalseNegative, &result.TrueNegative, predicted, c.Expected)
				if c.AggregateSamplingFrame != "" && c.AggregateSamplingFrame != c.SamplingFrame {
					aggregate, exists := frameCounts[c.AggregateSamplingFrame]
					if !exists {
						calibration.RegressionCases++
						calibration.RegressionFailures++
						continue
					}
					aggregate.Cases++
					countConfusion(&aggregate.TruePositive, &aggregate.FalsePositive, &aggregate.FalseNegative, &aggregate.TrueNegative, predicted, c.Expected)
				}
				continue
			}
			if c.Origin == "live-source-target-negative" {
				calibration.TargetNegativeCases++
				if hit != c.Expected {
					calibration.RegressionFailures++
				}
				continue
			}
			if c.Origin != "repository-beforeimage" {
				calibration.RegressionCases++
				if hit != c.Expected {
					calibration.RegressionFailures++
				}
				continue
			}
			calibration.HistoricalCases++
			calibration.Basis = "repository-beforeimage-selected-case-replay"
			switch {
			case hit && c.Expected:
				calibration.TruePositive++
			case hit:
				calibration.FalsePositive++
			case c.Expected:
				calibration.FalseNegative++
			default:
				calibration.TrueNegative++
			}
		}
		frameIDs := make([]string, 0, len(frameCounts))
		for frameID := range frameCounts {
			frameIDs = append(frameIDs, frameID)
		}
		sort.Strings(frameIDs)
		for _, frameID := range frameIDs {
			frame := frameCounts[frameID]
			frame.setRates()
			calibration.EvidenceFrames = append(calibration.EvidenceFrames, *frame)
		}
		calibration.SamplingFrame = corpus.Sampling.FrameID
		calibration.SamplingSelection = corpus.Sampling.Selection
		calibration.SamplingPopulation = corpus.Sampling.PopulationID
		calibration.SamplingUnit = corpus.Sampling.SamplingUnit
		calibration.SamplingMethod = corpus.Sampling.SelectionMethod
		calibration.SamplingSeed = corpus.Sampling.RandomSeed
		calibration.AdjudicationProtocol = corpus.Sampling.AdjudicationProtocol
		calibration.AdjudicationComplete = corpus.Sampling.AdjudicationComplete
		calibration.RepresentativeSample = corpus.Sampling.hasRateProvenance()
		calibration.setRates()
		detector.Calibration = calibration
	}
}

func countConfusion(tp, fp, fn, tn *int, hit, expected bool) {
	switch {
	case hit && expected:
		*tp++
	case hit:
		*fp++
	case expected:
		*fn++
	default:
		*tn++
	}
}

func (c *CalibrationFrame) setRates() {
	if c.FrameID == "" || c.PopulationID == "" || c.Population == "" || !c.CompleteCensus ||
		c.SamplingUnit == "" || c.SelectionMethod == "" || c.RandomSeed == "" ||
		c.AdjudicationProtocol == "" || !c.AdjudicationComplete {
		c.Precision, c.Recall = nil, nil
		return
	}
	if denominator := c.TruePositive + c.FalsePositive; denominator > 0 {
		value := float64(c.TruePositive) / float64(denominator)
		c.Precision = &value
	}
	if denominator := c.TruePositive + c.FalseNegative; denominator > 0 {
		value := float64(c.TruePositive) / float64(denominator)
		c.Recall = &value
	}
}

func (s corpusDocumentSampling) hasRateProvenance() bool {
	return s.Representative && s.FrameID != "" && s.Selection != "" && s.PopulationID != "" &&
		s.SamplingUnit != "" && s.SelectionMethod != "" && s.RandomSeed != "" &&
		s.AdjudicationProtocol != "" && s.AdjudicationComplete
}

func (c *Calibration) setRates() {
	// A selected case set can validate detector behavior on included cases,
	// but cannot estimate prevalence or recall. Publish rates only when the
	// corpus has a fully specified representative frame and completed labels.
	if !c.hasRateProvenance() {
		c.Precision, c.Recall = nil, nil
		return
	}
	if count := c.TruePositive + c.FalsePositive; count > 0 {
		value := float64(c.TruePositive) / float64(count)
		c.Precision = &value
	}
	if count := c.TruePositive + c.FalseNegative; count > 0 {
		value := float64(c.TruePositive) / float64(count)
		c.Recall = &value
	}
}

func (c Calibration) hasRateProvenance() bool {
	return c.RepresentativeSample && c.SamplingFrame != "" && c.SamplingSelection != "" &&
		c.SamplingPopulation != "" && c.SamplingUnit != "" && c.SamplingMethod != "" &&
		c.SamplingSeed != "" && c.AdjudicationProtocol != "" && c.AdjudicationComplete
}
