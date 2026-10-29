package orchestration

import (
	"bufio"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	codexrollout "github.com/vrooli/vrooli/packages/codex-rollout"
	_ "modernc.org/sqlite"
)

// TestLiveUsageComparison is an opt-in, read-only comparison of the usage
// account against the disk for named runs (SPEC A2 live evidence):
//
//	AM_USAGE_COMPARE_DB=scenarios/agent-manager/data/agent-manager.db \
//	AM_USAGE_COMPARE_RUNS=$HOME/.vrooli/state/vrooli/agent-manager/runs \
//	AM_USAGE_COMPARE_IDS=<run-id>,<run-id> go test ./internal/orchestration -run TestLiveUsageComparison -v
//
// Per run it reports the rollout's final counter, the read-model total, the
// account over the stored events, and the account over a fresh parse of the
// rollout. Stored events written before cumulative counters existed cannot be
// corrected by the account; after restart and re-capture they must agree.
func TestLiveUsageComparison(t *testing.T) {
	dbPath, root, ids := os.Getenv("AM_USAGE_COMPARE_DB"), os.Getenv("AM_USAGE_COMPARE_RUNS"), os.Getenv("AM_USAGE_COMPARE_IDS")
	if dbPath == "" || root == "" || ids == "" {
		t.Skip("set AM_USAGE_COMPARE_DB, AM_USAGE_COMPARE_RUNS and AM_USAGE_COMPARE_IDS to compare live runs")
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		var readModel sql.NullInt64
		_ = db.QueryRow(`SELECT total_tokens FROM invocation_read_model_runs WHERE run_id=?`, id).Scan(&readModel)
		rows, err := db.Query(`SELECT event_type, data FROM run_events WHERE run_id=? ORDER BY sequence`, id)
		if err != nil {
			t.Fatal(err)
		}
		var stored []*domain.RunEvent
		withCounters := 0
		for rows.Next() {
			var kind, data string
			if err := rows.Scan(&kind, &data); err != nil {
				t.Fatal(err)
			}
			payload, err := domain.DecodeEventPayload(domain.RunEventType(kind), []byte(data))
			if err != nil {
				continue
			}
			if usage, ok := payload.(*domain.UsageEventData); ok && usage.Cumulative != nil {
				withCounters++
			}
			stored = append(stored, &domain.RunEvent{EventType: domain.RunEventType(kind), Data: payload})
		}
		rows.Close()
		rollouts, _ := filepath.Glob(filepath.Join(root, id, "*", "sessions", "*", "*", "*", "rollout-*.jsonl"))
		nested, _ := filepath.Glob(filepath.Join(root, id, "*", "*", "sessions", "*", "*", "*", "rollout-*.jsonl"))
		rollouts = append(rollouts, nested...)
		sort.Strings(rollouts)
		var final, reparsed, retailed int64
		for _, path := range rollouts {
			events, retail, last := parseRollout(t, path)
			final += last
			reparsed += runsignal.AccountUsage(events).TotalTokens()
			retailed += runsignal.AccountUsage(retail).TotalTokens()
		}
		readModelText := "missing"
		if readModel.Valid {
			readModelText = strconv.FormatInt(readModel.Int64, 10)
		}
		t.Logf("run %s: disk final=%d read-model=%s stored-account=%d (events=%d, with counters=%d) fresh-parse-account=%d per-turn-retail+replay-account=%d rollouts=%d",
			id, final, readModelText, runsignal.AccountUsage(stored).TotalTokens(), len(stored), withCounters, reparsed, retailed, len(rollouts))
	}
}

// parseRollout returns the usage of a fresh parse, of a fresh parser per turn
// (each resuming the session at its turn) plus a replay of the last turn, and
// the rollout's final counter.
func parseRollout(t *testing.T, path string) ([]*domain.RunEvent, []*domain.RunEvent, int64) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	codec := codecs.NewCodexForTest()
	fresh, turn := codec.NewTranscriptParser(), codec.NewTranscriptParser()
	session := ""
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<20), codexrollout.MaxLineBytes)
	var events, retail []*domain.RunEvent
	var lastTurn []string
	var final int64
	usageOf := func(parsed []*domain.RunEvent) []*domain.RunEvent {
		var out []*domain.RunEvent
		for _, event := range parsed {
			if _, ok := event.Data.(*domain.UsageEventData); ok {
				out = append(out, event)
			}
		}
		return out
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `"type":"task_started"`) && session != "" {
			turn = codec.NewTranscriptParser()
			turn.(runner.TranscriptSessionSetter).SetTranscriptSession(session, false)
			lastTurn = nil
		}
		lastTurn = append(lastTurn, line)
		result := fresh.ParseTranscriptLine([16]byte{}, line)
		if result.SessionID != "" {
			session = result.SessionID
		}
		for _, event := range usageOf(result.Events) {
			events = append(events, event)
			if usage := event.Data.(*domain.UsageEventData); usage.Cumulative != nil && usage.Cumulative.Total() > final {
				final = usage.Cumulative.Total()
			}
		}
		retail = append(retail, usageOf(turn.ParseTranscriptLine([16]byte{}, line).Events)...)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	replay := codec.NewTranscriptParser()
	replay.(runner.TranscriptSessionSetter).SetTranscriptSession(session, false)
	for _, line := range lastTurn {
		retail = append(retail, usageOf(replay.ParseTranscriptLine([16]byte{}, line).Events)...)
	}
	return events, retail, final
}
