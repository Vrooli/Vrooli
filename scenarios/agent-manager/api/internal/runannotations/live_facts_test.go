package runannotations

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Opt-in, read-only: writes the facts the engine derives for the run ids of
// gold files, one row per run: the run's features plus every rule variable
// (RunFeatures.Tools: tool_failures, failed_commands, repeated_paths,
// tool_source and the rest), so a rule over those facts can be fitted and held
// out offline with exactly the code that will evaluate it. The file is 0600:
// last_tool_error and the message fields carry transcript text.
//
//	AGENT_MANAGER_ANNOTATIONS_LIVE_DB=<db> \
//	AGENT_MANAGER_ANNOTATIONS_GOLD=gold/a.jsonl,gold/b.jsonl \
//	AGENT_MANAGER_ANNOTATIONS_FACTS_OUT=facts.jsonl \
//	go test -run TestLiveGoldToolFacts ./internal/runannotations/
const (
	liveGoldEnv     = "AGENT_MANAGER_ANNOTATIONS_GOLD"
	liveFactsOutEnv = "AGENT_MANAGER_ANNOTATIONS_FACTS_OUT"
)

func TestLiveGoldToolFacts(t *testing.T) {
	db := openLiveCorpus(t, "derive tool facts for gold runs")
	files := strings.Split(os.Getenv(liveGoldEnv), ",")
	out := strings.TrimSpace(os.Getenv(liveFactsOutEnv))
	if strings.TrimSpace(files[0]) == "" || out == "" {
		t.Skipf("set %s (comma-separated gold files) and %s", liveGoldEnv, liveFactsOutEnv)
	}
	sink, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	source := NewSQLiteFeatureSource(db, codecParsers)
	seen := map[string]bool{}
	for _, file := range files {
		handle, err := os.Open(strings.TrimSpace(file))
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(handle)
		scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
		for scanner.Scan() {
			var row struct {
				RunID string `json:"run_id"`
			}
			if json.Unmarshal(scanner.Bytes(), &row) != nil || row.RunID == "" || seen[row.RunID] {
				continue
			}
			seen[row.RunID] = true
			facts, err := source.Facts(context.Background(), ScopeRun, row.RunID)
			if err != nil {
				t.Logf("%s: %v", row.RunID, err)
				continue
			}
			features := FeaturesFromFacts(facts)
			line, err := marshalFactsRow(features)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := sink.Write(append(line, '\n')); err != nil {
				t.Fatal(err)
			}
		}
		handle.Close()
	}
	t.Logf("wrote the facts of %d run(s) to %s", len(seen), out)
}

// marshalFactsRow merges the whole-run tool facts into the feature row. The
// rule variables win a name clash, as they do in the CEL activation.
func marshalFactsRow(features RunFeatures) ([]byte, error) {
	base, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}
	row := map[string]any{}
	if err := json.Unmarshal(base, &row); err != nil {
		return nil, err
	}
	for name, value := range features.Tools {
		row[name] = value
	}
	return json.Marshal(row)
}
