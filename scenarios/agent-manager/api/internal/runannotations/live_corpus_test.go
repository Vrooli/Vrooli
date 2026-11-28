package runannotations

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// liveCorpusEnv names a copy of (or read-only handle to) the real Agent Manager
// database. The test measures the provenance rule on the corpus the engine
// actually annotates; it is skipped everywhere else because a fixture cannot say
// how many of a real corpus's runs carry a person's words.
const liveCorpusEnv = "AGENT_MANAGER_ANNOTATIONS_LIVE_DB"

// openLiveCorpus opens the database named by liveCorpusEnv read-only, or skips.
func openLiveCorpus(t *testing.T, purpose string) *sqlx.DB {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(liveCorpusEnv))
	if path == "" {
		t.Skipf("set %s to a database path to %s", liveCorpusEnv, purpose)
	}
	db, err := sqlx.Connect("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestLiveCorpusProvenance(t *testing.T) {
	db := openLiveCorpus(t, "measure provenance on the real corpus")
	var runIDs []string
	if err := db.Select(&runIDs, `SELECT DISTINCT scope_id FROM run_annotations WHERE scope_kind = 'run' ORDER BY scope_id`); err != nil {
		t.Fatal(err)
	}
	source := NewSQLiteFeatureSource(db, nil)
	var defaults Pack
	defaults.normalize()
	policy := defaults.Runtime.Provenance
	type tally struct{ runs, withHuman, briefOnly, none, human, brief int }
	byKind := map[string]*tally{}
	total := &tally{}
	for _, runID := range runIDs {
		facts, err := source.Facts(context.Background(), ScopeRun, runID)
		if err != nil {
			continue
		}
		policy.Apply(&facts)
		kind := facts.Workload.Kind
		if kind == "" {
			kind = "(imported)"
		}
		row := byKind[kind]
		if row == nil {
			row = &tally{}
			byKind[kind] = row
		}
		for _, entry := range []*tally{row, total} {
			entry.runs++
			entry.human += len(facts.HumanMessages)
			entry.brief += facts.BriefMessages
			switch {
			case len(facts.HumanMessages) > 0:
				entry.withHuman++
			case facts.BriefMessages > 0:
				entry.briefOnly++
			default:
				entry.none++
			}
		}
	}
	kinds := make([]string, 0, len(byKind))
	for kind := range byKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		row := byKind[kind]
		t.Logf("%-14s runs=%-5d human-text=%-5d brief-only=%-5d no-operator-text=%-5d human-messages=%-5d brief-messages=%d",
			kind, row.runs, row.withHuman, row.briefOnly, row.none, row.human, row.brief)
	}
	t.Logf("%-14s runs=%-5d human-text=%-5d brief-only=%-5d no-operator-text=%-5d human-messages=%-5d brief-messages=%d",
		"TOTAL", total.runs, total.withHuman, total.briefOnly, total.none, total.human, total.brief)
	if total.runs == 0 {
		t.Fatal("the corpus has no annotated runs")
	}
}

// liveCensusEnv names a JSONL census of operator-source messages with a person's
// reading of who wrote each one ({"event_id","run_id","author":"person|agent"}).
// The test measures the authorship rule against it.
const liveCensusEnv = "AGENT_MANAGER_ANNOTATIONS_CENSUS"

func TestLiveCorpusAuthorshipAgainstACensus(t *testing.T) {
	db := openLiveCorpus(t, "measure authorship against a census")
	path := strings.TrimSpace(os.Getenv(liveCensusEnv))
	if path == "" {
		t.Skipf("set %s to a census JSONL", liveCensusEnv)
	}
	handle, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	type censusRow struct {
		EventID  string `json:"event_id"`
		RunID    string `json:"run_id"`
		Author   string `json:"author"`
		Workload string `json:"workload"`
	}
	var census []censusRow
	scanner := bufio.NewScanner(handle)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for scanner.Scan() {
		var row censusRow
		if json.Unmarshal(scanner.Bytes(), &row) == nil && row.EventID != "" {
			census = append(census, row)
		}
	}
	var defaults Pack
	defaults.normalize()
	policy := defaults.Runtime.Provenance
	source := NewSQLiteFeatureSource(db, nil)
	human := map[string]map[string]bool{}
	for _, row := range census {
		if _, done := human[row.RunID]; done {
			continue
		}
		facts, err := source.Facts(context.Background(), ScopeRun, row.RunID)
		if err != nil {
			t.Fatalf("run %s: %v", row.RunID, err)
		}
		policy.Apply(&facts)
		ids := map[string]bool{}
		for _, message := range facts.HumanMessages {
			ids[message.EventID] = true
		}
		human[row.RunID] = ids
	}
	type cell struct{ personKept, personDropped, agentKept, agentDropped int }
	byWorkload := map[string]*cell{}
	total := &cell{}
	for _, row := range census {
		kind := row.Workload
		if kind == "" {
			kind = "(imported)"
		}
		entry := byWorkload[kind]
		if entry == nil {
			entry = &cell{}
			byWorkload[kind] = entry
		}
		kept := human[row.RunID][row.EventID]
		for _, target := range []*cell{entry, total} {
			switch {
			case row.Author == "person" && kept:
				target.personKept++
			case row.Author == "person":
				target.personDropped++
			case kept:
				target.agentKept++
			default:
				target.agentDropped++
			}
		}
	}
	for kind, entry := range byWorkload {
		t.Logf("%-14s person kept %-4d person dropped %-3d agent kept %-4d agent dropped %d", kind, entry.personKept, entry.personDropped, entry.agentKept, entry.agentDropped)
	}
	t.Logf("%-14s person kept %-4d person dropped %-3d agent kept %-4d agent dropped %d", "TOTAL", total.personKept, total.personDropped, total.agentKept, total.agentDropped)
	if total.personDropped > 0 {
		t.Errorf("the rule dropped %d message(s) a person wrote", total.personDropped)
	}
}
