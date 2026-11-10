package discovery

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
)

//go:embed corpus.json
var labelledCorpus []byte

type corpusCase struct {
	ID         string `json:"id"`
	DetectorID string `json:"detector_id"`
	Expected   bool   `json:"expected"`
	Origin     string `json:"origin"`
	Source     string `json:"source"`
	Reason     string `json:"reason"`
	Symbols    []struct {
		Name, Path, Package, Receiver, Signature, BodyHash, TypeExpression string
		Start, End, BodyLines, Occurrences                                 int
		Exported                                                           bool
	} `json:"symbols"`
}

func corpusCases() []corpusCase {
	var corpus struct {
		Cases []corpusCase `json:"cases"`
	}
	if err := json.Unmarshal(labelledCorpus, &corpus); err != nil {
		panic(err)
	}
	return corpus.Cases
}

func (c corpusCase) input() []symbol {
	symbols := make([]symbol, 0, len(c.Symbols))
	for _, s := range c.Symbols {
		symbols = append(symbols, symbol{name: s.Name, path: s.Path, pkg: s.Package, receiver: s.Receiver, signature: s.Signature, bodyHash: s.BodyHash, typeExpression: s.TypeExpression, start: s.Start, end: s.End, bodyLines: s.BodyLines, nameOccurrences: s.Occurrences, exported: s.Exported})
	}
	return symbols
}

func (r *Registry) calibrate() {
	digest := sha256.Sum256(labelledCorpus)
	for i := range r.detectors {
		detector := &r.detectors[i]
		calibration := Calibration{Basis: "uncalibrated", CorpusVersion: "sha256:" + hex.EncodeToString(digest[:])}
		for _, c := range corpusCases() {
			if c.DetectorID != detector.ID {
				continue
			}
			hit := len(detector.detect(c.input())) > 0
			if c.Origin != "repository-beforeimage" {
				calibration.RegressionCases++
				if hit != c.Expected {
					calibration.RegressionFailures++
				}
				continue
			}
			calibration.HistoricalCases++
			calibration.Basis = "labelled-historical-symbol-replay"
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
		if count := calibration.TruePositive + calibration.FalsePositive; count > 0 {
			value := float64(calibration.TruePositive) / float64(count)
			calibration.Precision = &value
		}
		if count := calibration.TruePositive + calibration.FalseNegative; count > 0 {
			value := float64(calibration.TruePositive) / float64(count)
			calibration.Recall = &value
		}
		detector.Calibration = calibration
	}
}
