package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

type signalCorpus struct {
	Groups []signalCorpusGroup `json:"groups"`
}

type signalCorpusGroup struct {
	ID        string   `json:"id"`
	Scenario  string   `json:"scenario"`
	Label     string   `json:"label"`
	Rationale string   `json:"rationale"`
	Excerpt   string   `json:"excerpt"`
	Lines     int      `json:"lines"`
	Role      string   `json:"role"`
	Locations []string `json:"locations"`
}

func TestSignalCorpusClassifierQuality(t *testing.T) {
	data, err := os.ReadFile("testdata/signal-corpus/groups.json")
	if err != nil {
		t.Fatalf("read signal corpus: %v", err)
	}
	var corpus signalCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("parse signal corpus: %v", err)
	}
	if len(corpus.Groups) < 50 {
		t.Fatalf("signal corpus has %d groups, want at least 50", len(corpus.Groups))
	}
	scenarios := map[string]struct{}{}
	confusion := map[string]map[DuplicationClass]int{}
	structuralOrIncidental, debtBearingMistakes := 0, 0
	for _, entry := range corpus.Groups {
		if entry.ID == "" || entry.Scenario == "" || entry.Rationale == "" || entry.Excerpt == "" || len(entry.Locations) < 2 || entry.Lines < DuplicateMinimumLines {
			t.Fatalf("incomplete corpus entry: %#v", entry)
		}
		scenarios[entry.Scenario] = struct{}{}
		block := DuplicateBlock{Lines: entry.Lines, Files: []DuplicateLocation{{Path: entry.Locations[0]}, {Path: entry.Locations[1]}}}
		class := classifyDuplicateBlockSignals(block, strings.Split(entry.Excerpt, "\n"), []FileRole{parseFileRole(entry.Role), parseFileRole(entry.Role)})
		if confusion[entry.Label] == nil {
			confusion[entry.Label] = map[DuplicationClass]int{}
		}
		confusion[entry.Label][class]++
		switch entry.Label {
		case "structural", "incidental":
			structuralOrIncidental++
			if class == DuplicationClassOpportunity || class == DuplicationClassHighLeverage {
				debtBearingMistakes++
			}
		case "opportunity":
			if class != DuplicationClassOpportunity && class != DuplicationClassHighLeverage {
				t.Fatalf("missed opportunity %s classified %q; confusion: %s", entry.ID, class, formatCorpusConfusion(confusion))
			}
		case "uncertain":
			if class != DuplicationClassIncidental {
				t.Fatalf("uncertain entry %s classified %q, want conservative incidental; confusion: %s", entry.ID, class, formatCorpusConfusion(confusion))
			}
		default:
			t.Fatalf("unknown corpus label %q for %s", entry.Label, entry.ID)
		}
	}
	if len(scenarios) < 3 {
		t.Fatalf("signal corpus covers %d scenarios, want at least 3", len(scenarios))
	}
	if debtBearingMistakes*10 > structuralOrIncidental {
		t.Fatalf("signal corpus debt-bearing disagreement %d/%d exceeds 10%%; confusion: %s", debtBearingMistakes, structuralOrIncidental, formatCorpusConfusion(confusion))
	}
}

func formatCorpusConfusion(confusion map[string]map[DuplicationClass]int) string {
	return fmt.Sprintf("%v", confusion)
}
