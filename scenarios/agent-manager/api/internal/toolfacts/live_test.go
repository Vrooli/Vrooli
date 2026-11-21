package toolfacts_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/toolfacts"
)

// Opt-in, read-only live-corpus tests. Point them at the Agent Manager
// database; they open it with mode=ro and read transcript files the runs name.
//
//	AGENT_MANAGER_TOOLFACTS_LIVE_DB=<db> \
//	AGENT_MANAGER_TOOLFACTS_RUNS=<run id prefix>,<run id prefix> \
//	go test -run TestLiveRunFacts -v ./internal/toolfacts/
const (
	liveDBEnv   = "AGENT_MANAGER_TOOLFACTS_LIVE_DB"
	liveRunsEnv = "AGENT_MANAGER_TOOLFACTS_RUNS"
)

func openLive(t *testing.T) *sqlx.DB {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(liveDBEnv))
	if path == "" {
		t.Skipf("set %s to a database path", liveDBEnv)
	}
	db, err := sqlx.Connect("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(30000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func liveReader(t *testing.T) *toolfacts.Reader {
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	return &toolfacts.Reader{DB: openLive(t), Parsers: codecParsers, Rules: rules}
}

// liveRunIDs expands id prefixes into full run ids.
func liveRunIDs(t *testing.T, db *sqlx.DB) []string {
	t.Helper()
	var ids []string
	for _, prefix := range strings.Split(os.Getenv(liveRunsEnv), ",") {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		var found []string
		if err := db.Select(&found, `SELECT id FROM runs WHERE id LIKE ?`, prefix+"%"); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, found...)
	}
	if len(ids) == 0 {
		t.Skipf("set %s to run id prefixes", liveRunsEnv)
	}
	return ids
}

func TestLiveRunFacts(t *testing.T) {
	reader := liveReader(t)
	for _, id := range liveRunIDs(t, reader.DB.(*sqlx.DB)) {
		start := time.Now()
		facts, err := reader.Facts(context.Background(), id)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		encoded, _ := json.Marshal(facts)
		t.Logf("%s %dms %s", id[:8], time.Since(start).Milliseconds(), encoded)
	}
}

// measure runs fn and reports its wall time and the peak live heap it reached,
// sampled every few milliseconds after a collection at the start.
func measure(fn func()) (time.Duration, uint64) {
	runtime.GC()
	var stats runtime.MemStats
	var peak uint64
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for {
			runtime.ReadMemStats(&stats)
			if stats.HeapAlloc > peak {
				peak = stats.HeapAlloc
			}
			select {
			case <-done:
				return
			case <-time.After(5 * time.Millisecond):
			}
		}
	}()
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	close(done)
	<-finished
	return elapsed, peak
}

// TestLiveTranscriptFacts reads each run's transcript file directly, whether
// or not the run also has events, and prints the facts beside the time and
// peak heap the read cost.
func TestLiveTranscriptFacts(t *testing.T) {
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	db := openLive(t)
	for _, id := range liveRunIDs(t, db) {
		var path, harness string
		if err := db.QueryRow(`SELECT transcript_path, COALESCE(NULLIF(import_source_harness, ''), CASE WHEN json_valid(resolved_config) THEN json_extract(resolved_config, '$.runnerType') END) FROM runs WHERE id = ?`, id).Scan(&path, &harness); err != nil || path == "" {
			t.Logf("%s: no transcript", id[:8])
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Logf("%s: transcript missing", id[:8])
			continue
		}
		parser, err := codecParsers(harness)
		if err != nil {
			t.Fatal(err)
		}
		var facts toolfacts.Facts
		elapsed, peak := measure(func() {
			facts, err = toolfacts.FromTranscript(context.Background(), rules, path, parser)
		})
		if err != nil {
			t.Errorf("%s: %v", id[:8], err)
			continue
		}
		encoded, _ := json.Marshal(facts)
		t.Logf("%s %s %.1fMB %dms peak %.0fMB %s", id[:8], harness, float64(info.Size())/1e6, elapsed.Milliseconds(), float64(peak)/1e6, encoded)
	}
}

// TestLiveGoldFacts derives the facts of the runs in gold files (jsonl rows
// with run_id and positive) so a rule over them can be fitted and held out
// offline with exactly the code that will evaluate it.
//
//	AGENT_MANAGER_TOOLFACTS_GOLD=a.jsonl,b.jsonl AGENT_MANAGER_TOOLFACTS_GOLD_OUT=facts.jsonl
const (
	liveGoldEnv    = "AGENT_MANAGER_TOOLFACTS_GOLD"
	liveGoldOutEnv = "AGENT_MANAGER_TOOLFACTS_GOLD_OUT"
)

func TestLiveGoldFacts(t *testing.T) {
	files, out := strings.Split(os.Getenv(liveGoldEnv), ","), strings.TrimSpace(os.Getenv(liveGoldOutEnv))
	if strings.TrimSpace(files[0]) == "" || out == "" {
		t.Skipf("set %s and %s", liveGoldEnv, liveGoldOutEnv)
	}
	reader := liveReader(t)
	sink, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	seen := map[string]bool{}
	for _, file := range files {
		raw, err := os.ReadFile(strings.TrimSpace(file))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			var row struct {
				RunID    string `json:"run_id"`
				Label    string `json:"label"`
				Positive bool   `json:"positive"`
			}
			if json.Unmarshal([]byte(line), &row) != nil || row.RunID == "" || seen[row.Label+row.RunID] {
				continue
			}
			seen[row.Label+row.RunID] = true
			facts, err := reader.Facts(context.Background(), row.RunID)
			if err != nil {
				t.Logf("%s: %v", row.RunID, err)
				continue
			}
			encoded, _ := json.Marshal(map[string]any{"run_id": row.RunID, "file": file, "positive": row.Positive, "facts": facts})
			sink.Write(append(encoded, '\n'))
		}
	}
}

// TestLiveNormalizationEvidence shows what each normalize rule changes on real
// runs: unchanged_repeats and longest_unchanged_run with every rule, with none,
// and with each rule alone removed. A rule that moves no run is not needed.
func TestLiveNormalizationEvidence(t *testing.T) {
	reader := liveReader(t)
	all := reader.Rules
	variants := []struct {
		name  string
		rules toolfacts.Rules
	}{{"all", all}}
	none := all
	none.Normalize = nil
	variants = append(variants, struct {
		name  string
		rules toolfacts.Rules
	}{"none", none})
	for i, rule := range all.Normalize {
		without := all
		without.Normalize = append(append([]toolfacts.NormalizeRule(nil), all.Normalize[:i]...), all.Normalize[i+1:]...)
		variants = append(variants, struct {
			name  string
			rules toolfacts.Rules
		}{"without " + rule.ID, without})
	}
	for _, id := range liveRunIDs(t, reader.DB.(*sqlx.DB)) {
		var line []string
		for _, variant := range variants {
			reader.Rules = variant.rules
			facts, err := reader.Facts(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			line = append(line, fmt.Sprintf("%s=%d/%d", variant.name, facts.UnchangedRepeats, facts.LongestUnchangedRun))
		}
		t.Logf("%s calls=%s", id[:8], strings.Join(line, "  "))
	}
}

// TestLiveChunkEquivalence checks that releasing events in chunks changes no
// command outcome: failed_commands from the extractor equals failed_commands
// from one runsignal.DeriveInvocationFacts pass over the whole run, which is
// what a chunk boundary could break (a nested command result arriving after
// its call's chunk was released).
func TestLiveChunkEquivalence(t *testing.T) {
	db := openLive(t)
	reader := liveReader(t)
	for _, id := range liveRunIDs(t, db) {
		rows, err := db.Queryx(`SELECT id, event_type, data FROM run_events WHERE run_id = ? AND event_type IN ('tool_call', 'tool_result') ORDER BY sequence`, id)
		if err != nil {
			t.Fatal(err)
		}
		var events []*domain.RunEvent
		for rows.Next() {
			var eid, eventType, data string
			if err := rows.Scan(&eid, &eventType, &data); err != nil {
				t.Fatal(err)
			}
			payload, err := domain.DecodeEventPayload(domain.RunEventType(eventType), []byte(data))
			if err != nil {
				continue
			}
			parsed, _ := uuid.Parse(eid)
			events = append(events, &domain.RunEvent{ID: parsed, EventType: domain.RunEventType(eventType), Data: payload})
		}
		rows.Close()
		calls, distinct := 0, map[string]bool{}
		for _, event := range events {
			if call, ok := event.Data.(*domain.ToolCallEventData); ok {
				calls++
				distinct[call.ToolCallID] = true
			}
		}
		whole := 0
		for _, fact := range runsignal.DeriveInvocationFacts(events) {
			if fact.Outcome == runsignal.OutcomeFailure && fact.ExitCode != nil {
				whole++
			}
		}
		facts, err := reader.Facts(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s events=%d whole-run failed_commands=%d chunked=%d", id[:8], len(events), whole, facts.FailedCommands)
		if len(distinct) < calls {
			// Codex exec reuses ids across turns; one whole-run pass keys calls
			// by id and collapses them, so it is not the reference there.
			t.Logf("%s reuses call ids (%d calls, %d ids): whole-run pass undercounts, chunked is the reference", id[:8], calls, len(distinct))
			continue
		}
		if whole != facts.FailedCommands {
			t.Errorf("%s: chunked failed_commands %d differs from whole-run %d", id[:8], facts.FailedCommands, whole)
		}
	}
}

// TestLivePopulation reads, read-only, every run that has no tool events in
// run_events but names a transcript file, writes one JSON line per run (facts,
// harness, size, read time) and reports how many runs gain tool facts, by
// harness and for runs of at least 20 minutes.
//
//	AGENT_MANAGER_TOOLFACTS_POPULATION_OUT=population.jsonl \
//	AGENT_MANAGER_TOOLFACTS_WORKERS=3 go test -timeout 3h -run TestLivePopulation ./internal/toolfacts/
const (
	livePopulationOutEnv = "AGENT_MANAGER_TOOLFACTS_POPULATION_OUT"
	liveWorkersEnv       = "AGENT_MANAGER_TOOLFACTS_WORKERS"
)

func TestLivePopulation(t *testing.T) {
	out := strings.TrimSpace(os.Getenv(livePopulationOutEnv))
	if out == "" {
		t.Skipf("set %s", livePopulationOutEnv)
	}
	workers := 3
	if value, err := strconv.Atoi(os.Getenv(liveWorkersEnv)); err == nil && value > 0 {
		workers = value
	}
	db := openLive(t)
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	type run struct {
		ID, Path, Harness string
		Minutes           sql.NullFloat64
	}
	var runs []run
	rows, err := db.Queryx(`
SELECT r.id, COALESCE(r.transcript_path, ''),
       COALESCE(NULLIF(r.import_source_harness, ''), CASE WHEN json_valid(r.resolved_config) THEN json_extract(r.resolved_config, '$.runnerType') END, ''),
       (julianday(r.ended_at) - julianday(r.started_at)) * 1440.0
FROM runs r
WHERE r.transcript_path <> ''
  AND NOT EXISTS (SELECT 1 FROM run_events e WHERE e.run_id = r.id AND e.event_type = 'tool_call')`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var r run
		if err := rows.Scan(&r.ID, &r.Path, &r.Harness, &r.Minutes); err != nil {
			t.Fatal(err)
		}
		runs = append(runs, r)
	}
	rows.Close()

	sink, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	type result struct {
		RunID, Harness string
		Minutes        *float64
		Bytes          int64
		Millis         int64
		Missing        bool
		Facts          toolfacts.Facts
	}
	jobs := make(chan run)
	results := make(chan result)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for r := range jobs {
				res := result{RunID: r.ID, Harness: r.Harness}
				if r.Minutes.Valid {
					res.Minutes = &r.Minutes.Float64
				}
				info, err := os.Stat(r.Path)
				if err != nil {
					res.Missing = true
					results <- res
					continue
				}
				res.Bytes = info.Size()
				parser, err := codecParsers(r.Harness)
				if err != nil {
					res.Facts = toolfacts.Facts{Source: toolfacts.SourceNone}
					results <- res
					continue
				}
				start := time.Now()
				facts, err := toolfacts.FromTranscript(context.Background(), rules, r.Path, parser)
				res.Millis = time.Since(start).Milliseconds()
				if err != nil {
					res.Facts = toolfacts.Facts{Source: toolfacts.SourceNone}
				} else {
					res.Facts = facts
				}
				results <- res
			}
		}()
	}
	go func() {
		for _, r := range runs {
			jobs <- r
		}
		close(jobs)
		wait.Wait()
		close(results)
	}()

	type tally struct{ runs, missing, gained, long, longGained int }
	byHarness := map[string]*tally{}
	var done int
	var peakHeap uint64
	started := time.Now()
	for res := range results {
		done++
		line, _ := json.Marshal(res)
		sink.Write(append(line, '\n'))
		entry := byHarness[res.Harness]
		if entry == nil {
			entry = &tally{}
			byHarness[res.Harness] = entry
		}
		long := res.Minutes != nil && *res.Minutes >= 20
		entry.runs++
		if long {
			entry.long++
		}
		switch {
		case res.Missing:
			entry.missing++
		case res.Facts.ToolCalls > 0:
			entry.gained++
			if long {
				entry.longGained++
			}
		}
		if done%250 == 0 {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			if stats.HeapAlloc > peakHeap {
				peakHeap = stats.HeapAlloc
			}
			fmt.Fprintf(os.Stderr, "population %d/%d after %s\n", done, len(runs), time.Since(started).Round(time.Second))
		}
	}
	for harness, entry := range byHarness {
		t.Logf("harness=%-12s runs-without-events=%d transcript-missing=%d gain-tool-facts=%d | >=20min: runs=%d gain=%d",
			harness, entry.runs, entry.missing, entry.gained, entry.long, entry.longGained)
	}
	t.Logf("read %d runs in %s with %d workers", done, time.Since(started).Round(time.Second), workers)
}
