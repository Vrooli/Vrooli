package calibration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentbrief "github.com/vrooli/agentbrief-go"
)

type caseRecord struct {
	ID         string `json:"id"`
	Prompt     string `json:"prompt"`
	Answerable bool   `json:"answerable"`
	Provider   string `json:"expected_provider"`
	Title      string `json:"expected_title"`
}

func TestMeasuredDefaultThresholdIsPinned(t *testing.T) {
	if agentbrief.DefaultThresholds.MinRerankScore != 0.01 {
		t.Fatalf("MinRerankScore = %.3f, want measured 0.010", agentbrief.DefaultThresholds.MinRerankScore)
	}
	if agentbrief.DefaultThresholds.MinItemRerankScore != 0.20 {
		t.Fatalf("MinItemRerankScore = %.3f, want 0.200", agentbrief.DefaultThresholds.MinItemRerankScore)
	}
}

func TestCorpusHasMinimumLabelledCoverage(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var records []caseRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	answers, noAnswers := 0, 0
	seen := map[string]bool{}
	for _, record := range records {
		if record.ID == "" || record.Prompt == "" || seen[record.ID] {
			t.Fatalf("invalid or duplicate corpus record: %+v", record)
		}
		if strings.Contains(record.Provider, "/home/") || strings.Contains(record.Title, "/home/") || strings.Contains(record.Provider, "2026-") || strings.Contains(record.Title, "2026-") {
			t.Fatalf("portable corpus contains operator/live identifier: %+v", record)
		}
		seen[record.ID] = true
		if record.Answerable {
			if record.Provider == "" || record.Title == "" {
				t.Fatalf("answerable record lacks expected provider/title: %+v", record)
			}
			answers++
		} else {
			if record.Provider != "" || record.Title != "" {
				t.Fatalf("no-answer record has an expected result: %+v", record)
			}
			noAnswers++
		}
	}
	if answers < 30 || noAnswers < 8 {
		t.Fatalf("coverage answers=%d no_answers=%d; want at least 30 and 8", answers, noAnswers)
	}
}
