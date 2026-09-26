package interactivefeedbackqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const EvidenceGlob = ".vrooli/runtime/rehabilitation-evidence/interactive-feedback-*.json"

const (
	contractPath = "docs/internal/REFRACTOR_CONTRACT.json"
	// RequiredSourceFile is the producer identity that every cohort must bind.
	RequiredSourceFile = "playwright-driver/tests/integration/input-feedback.test.ts"
)

type receipt struct {
	EvidenceKind         string            `json:"evidence_kind"`
	OutcomeID            string            `json:"outcome_id"`
	Status               string            `json:"status"`
	ManagedBuildIdentity string            `json:"managed_build_identity"`
	SourceSHA256         map[string]string `json:"source_sha256"`
	Cohort               string            `json:"cohort"`
	Measurement          struct {
		SampleCountSnake               int     `json:"sample_count"`
		SampleCountCamel               int     `json:"sampleCount"`
		CorrelatedReceiptCountSnake    int     `json:"correlated_receipt_count"`
		CorrelatedReceiptCountCamel    int     `json:"correlatedReceiptCount"`
		CorrelatedCanvasCountSnake     int     `json:"correlated_canvas_paint_count"`
		CorrelatedCanvasCountCamel     int     `json:"correlatedCanvasPaintCount"`
		CorrelationCompleteSnake       bool    `json:"correlation_complete"`
		CorrelationCompleteCamel       bool    `json:"correlationComplete"`
		ReceiptSequencesMonotonicSnake bool    `json:"receipt_sequences_monotonic"`
		ReceiptSequencesMonotonicCamel bool    `json:"receiptSequencesMonotonic"`
		P50MSSnake                     float64 `json:"p50_ms"`
		P50MSCamel                     float64 `json:"p50Ms"`
		P95MSSnake                     float64 `json:"p95_ms"`
		P95MSCamel                     float64 `json:"p95Ms"`
		P99MSSnake                     float64 `json:"p99_ms"`
		P99MSCamel                     float64 `json:"p99Ms"`
	} `json:"measurement"`
}

// Validate accepts only complete local and remote cohorts for the same source,
// contract, and managed build. A diagnostic or one-cohort receipt remains
// unavailable instead of being treated as partial credit.
func Validate(root, build string) error {
	paths, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(EvidenceGlob)))
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	contractSHA, err := fileSHA(filepath.Join(root, filepath.FromSlash(contractPath)))
	if err != nil {
		return err
	}
	testSHA, err := fileSHA(filepath.Join(root, filepath.FromSlash(RequiredSourceFile)))
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var candidate receipt
		if json.Unmarshal(raw, &candidate) != nil || !valid(candidate, build, contractSHA, testSHA) {
			continue
		}
		seen[candidate.Cohort] = true
	}
	if !seen["local"] || !seen["remote"] {
		return fmt.Errorf("no complete current-build interactive-feedback local and remote cohorts")
	}
	return nil
}

func valid(r receipt, build, contractSHA, testSHA string) bool {
	m := r.Measurement
	sampleCount := maxInt(m.SampleCountSnake, m.SampleCountCamel)
	correlatedReceiptCount := maxInt(m.CorrelatedReceiptCountSnake, m.CorrelatedReceiptCountCamel)
	correlatedCanvasCount := maxInt(m.CorrelatedCanvasCountSnake, m.CorrelatedCanvasCountCamel)
	correlationComplete := m.CorrelationCompleteSnake || m.CorrelationCompleteCamel
	receiptSequencesMonotonic := m.ReceiptSequencesMonotonicSnake || m.ReceiptSequencesMonotonicCamel
	p50 := maxFloat(m.P50MSSnake, m.P50MSCamel)
	p95 := maxFloat(m.P95MSSnake, m.P95MSCamel)
	p99 := maxFloat(m.P99MSSnake, m.P99MSCamel)
	return r.EvidenceKind == "interactive_feedback_cohort" &&
		r.OutcomeID == "interactive-feedback" && r.Status == "passed" &&
		r.ManagedBuildIdentity == build && (r.Cohort == "local" || r.Cohort == "remote") &&
		r.SourceSHA256[contractPath] == contractSHA && r.SourceSHA256[RequiredSourceFile] == testSHA &&
		sampleCount == 1000 && correlatedReceiptCount == 1000 &&
		correlatedCanvasCount == 1000 && correlationComplete &&
		receiptSequencesMonotonic && p50 <= 50 && p95 <= 100 &&
		p99 <= 200 && (r.Cohort != "remote" || p95 <= 200)
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func fileSHA(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
