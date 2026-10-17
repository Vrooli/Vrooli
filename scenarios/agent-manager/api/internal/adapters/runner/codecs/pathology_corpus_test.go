package codecs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// TestPathologyCorpusIsCommitted keeps the named investigation pathologies
// durable and offline. Higher-level report tests may replay these captures
// through fake-agent without spending a live-run budget.
func TestPathologyCorpusIsCommitted(t *testing.T) {
	for _, name := range []string{
		"ambiguous-final-output", "unavailable-final-output", "invalid-structured-result",
		"abstained-structured-result", "tool-failures", "model-fallback", "heartbeat-gap",
		"oversized-diff", "zero-events",
	} {
		path := filepath.Join("testdata", "corpus", "pathology", name+".jsonl")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing pathology corpus %s: %v", name, err)
		}
	}
}

func TestPathologyFinalOutputFixturesExerciseSelectionStates(t *testing.T) {
	for _, tc := range []struct {
		name string
		want domain.FinalOutputSelectionStatus
	}{
		{"ambiguous-final-output", domain.FinalOutputSelectionAmbiguous},
		{"unavailable-final-output", domain.FinalOutputSelectionUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "corpus", "pathology", tc.name+".jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			codec := NewCodexForTest()
			state, runID := codec.NewState(), uuid.New()
			var events []*domain.RunEvent
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				decoded, err := codec.DecodeStreamLine(state, runID, line)
				if err != nil {
					t.Fatalf("decode %q: %v", line, err)
				}
				events = append(events, decoded...)
			}
			result := domain.ResolveRunResult(events, true, 0, "completed")
			if result.Selection.Status != tc.want {
				t.Fatalf("selection=%q want=%q", result.Selection.Status, tc.want)
			}
		})
	}
}
