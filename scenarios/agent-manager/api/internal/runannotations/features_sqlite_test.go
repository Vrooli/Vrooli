package runannotations

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func sqliteFixture(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "facts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE runs (id TEXT PRIMARY KEY, workload_kind TEXT, workload_key TEXT, harness_kind TEXT, run_mode TEXT, status TEXT, terminal_class TEXT, stop_reason TEXT, started_at TEXT, ended_at TEXT, exit_code INTEGER)`,
		`CREATE TABLE run_events (id TEXT PRIMARY KEY, run_id TEXT NOT NULL, sequence INTEGER NOT NULL, event_type TEXT NOT NULL, timestamp TEXT, schema_version INTEGER NOT NULL DEFAULT 1, data TEXT NOT NULL)`,
		`CREATE TABLE operator_signals (event_id TEXT, signal_version TEXT, run_id TEXT, occurred_at TEXT, status TEXT, label TEXT, probability REAL, recorded_at TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestSQLiteFeatureSourceReadsBoundedFacts(t *testing.T) {
	db := sqliteFixture(t)
	started := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	ended := started.Add(45 * time.Minute)
	if _, err := db.Exec(`INSERT INTO runs VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"run-1", "workflow_node", "classify", "codex", "codex_pipe", "completed", "goal_complete", "",
		started.Format(time.RFC3339Nano), ended.Format(time.RFC3339Nano), 0); err != nil {
		t.Fatal(err)
	}
	events := []struct {
		id, kind string
		at       time.Time
		data     string
	}{
		{"e1", "message", started, `{"role":"user","content":"please fix the parser"}`},
		{"e2", "tool_call", started.Add(2 * time.Minute), `{"toolName":"bash","input":{"file_path":"api/run.go"}}`},
		{"e3", "tool_result", started.Add(3 * time.Minute), `{"toolName":"bash","success":false,"error":"exit 1","exitCode":1}`},
		{"e4", "tool_call", started.Add(4 * time.Minute), `{"toolName":"bash","input":{"file_path":"api/run.go"}}`},
		{"e5", "message", started.Add(5 * time.Minute), `{"role":"assistant","content":"I changed the parser"}`},
		{"e6", "compaction", started.Add(6 * time.Minute), `{}`},
	}
	for i, event := range events {
		if _, err := db.Exec(`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data) VALUES (?,?,?,?,?,?)`,
			event.id, "run-1", i+1, event.kind, event.at.Format(time.RFC3339Nano), event.data); err != nil {
			t.Fatal(err)
		}
	}
	source := &SQLiteFeatureSource{DB: db, Now: func() time.Time { return ended }}
	facts, err := source.Facts(context.Background(), ScopeRun, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if facts.Workload.Kind != "workflow_node" || facts.Workload.Key != "classify" {
		t.Fatalf("workload = %+v", facts.Workload)
	}
	if len(facts.OperatorMessages) != 1 || len(facts.AgentText) != 1 {
		t.Fatalf("messages = %d/%d", len(facts.OperatorMessages), len(facts.AgentText))
	}
	if facts.ToolSummary.Calls != 2 || facts.ToolSummary.Failures != 1 || facts.ToolSummary.FailedCommands != 1 {
		t.Fatalf("tool summary = %+v", facts.ToolSummary)
	}
	if facts.ToolSummary.RepeatedPaths != 1 {
		t.Fatalf("repeated paths = %d", facts.ToolSummary.RepeatedPaths)
	}
	if facts.DeterministicFlags.CompactionCount != 1 {
		t.Fatalf("compactions = %d", facts.DeterministicFlags.CompactionCount)
	}
	if facts.Timings.DurationMinutes != 45 {
		t.Fatalf("duration = %v", facts.Timings.DurationMinutes)
	}
	if facts.Timings.TimeToFirstToolMin == nil || *facts.Timings.TimeToFirstToolMin != 2 {
		t.Fatalf("time to first tool = %v", facts.Timings.TimeToFirstToolMin)
	}

	if _, err := db.Exec(`INSERT INTO operator_signals VALUES (?,?,?,?,?,?,?,?)`,
		"s1", "signal-v1", "run-1", started.Format(time.RFC3339Nano), "classified", "frustration", 0.8, started.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	signals, err := NewSQLiteSignalReader(db).RunSignals(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if fact := signals["frustration"]; fact.Probability == nil || *fact.Probability != 0.8 || fact.LatestEventID != "s1" {
		t.Fatalf("signal = %+v", fact)
	}

	selected, err := NewSQLiteRunSelector(db).CandidateRunsWindow(context.Background(), started.Add(-time.Hour), ended.Add(time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0] != "run-1" {
		t.Fatalf("selected = %v", selected)
	}
	empty, err := NewSQLiteRunSelector(db).CandidateRunsWindow(context.Background(), ended.Add(time.Hour), ended.Add(2*time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("future window = %v", empty)
	}
}

func TestSQLiteFeatureSourceUnknownRunIsNotFound(t *testing.T) {
	db := sqliteFixture(t)
	_, err := NewSQLiteFeatureSource(db).Facts(context.Background(), ScopeRun, "missing")
	if err != ErrScopeNotFound {
		t.Fatalf("err = %v", err)
	}
}

// A run can end with hundreds of assistant messages after the operator's last
// turn. Reading only a fixed tail made operator_message_count zero and skipped
// every operator label; the reader must page back to the operator's own words.
func TestSQLiteFeatureSourceFindsOperatorMessagesBeyondTheRecentTail(t *testing.T) {
	db := sqliteFixture(t)
	started := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	ended := started.Add(30 * time.Minute)
	if _, err := db.Exec(`INSERT INTO runs VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"run-1", "interactive", "fix", "claude-code", "interactive", "completed", "goal_complete", "",
		started.Format(time.RFC3339Nano), ended.Format(time.RFC3339Nano), 0); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	insert := func(sequence int, role, content string, at time.Time) {
		data := `{"role":"` + role + `","content":` + strconv.Quote(content) + `}`
		if _, err := tx.Exec(`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data) VALUES (?,?,?,?,?,?)`,
			"e"+strconv.Itoa(sequence), "run-1", sequence, "message", at.Format(time.RFC3339Nano), data); err != nil {
			t.Fatal(err)
		}
	}
	// Operator turns early in the run.
	insert(1, "user", "please stop rewriting the same helper", started)
	insert(2, "user", "this is getting worse, revert it", started.Add(time.Minute))
	// A long assistant-only tail larger than the reader's page size.
	for sequence := 3; sequence <= 403; sequence++ {
		insert(sequence, "assistant", "working on it", started.Add(time.Duration(sequence)*time.Second))
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	source := &SQLiteFeatureSource{DB: db, Now: func() time.Time { return ended }}
	facts, err := source.Facts(context.Background(), ScopeRun, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.OperatorMessages) != 2 {
		t.Fatalf("operator messages = %d, want 2: %+v", len(facts.OperatorMessages), facts.OperatorMessages)
	}
	if facts.OperatorMessages[1].Text != "this is getting worse, revert it" {
		t.Fatalf("last operator message = %q", facts.OperatorMessages[1].Text)
	}
	features := FeaturesFromFacts(facts)
	if features.OperatorMessageCount != 2 {
		t.Fatalf("operator_message_count = %d", features.OperatorMessageCount)
	}
}

func TestHarnessOnlySkillBlockIsNotOperatorText(t *testing.T) {
	db := sqliteFixture(t)
	started := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	ended := started.Add(10 * time.Minute)
	if _, err := db.Exec(`INSERT INTO runs VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"run-skills", "workflow_node", "draft", "codex", "sandboxed", "completed", "goal_complete", "",
		started.Format(time.RFC3339Nano), ended.Format(time.RFC3339Nano), 0); err != nil {
		t.Fatal(err)
	}
	// A stored "operator" source predating the <skills> tag recognition must
	// not make a harness-inlined skill body look like the operator speaking.
	events := []struct {
		id       string
		sequence int
		data     string
	}{
		{"s1", 1, `{"role":"user","source":"operator","content":"<skills count=\"1\"><skill id=\"a\"># body</skill></skills>"}`},
		{"s2", 2, `{"role":"assistant","content":"The inventory is complete."}`},
	}
	for _, event := range events {
		if _, err := db.Exec(`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data) VALUES (?,?,?,?,?,?)`,
			event.id, "run-skills", event.sequence, "message", started.Format(time.RFC3339Nano), event.data); err != nil {
			t.Fatal(err)
		}
	}
	source := &SQLiteFeatureSource{DB: db, Now: func() time.Time { return ended }}
	facts, err := source.Facts(context.Background(), ScopeRun, "run-skills")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.OperatorMessages) != 0 {
		t.Fatalf("operator messages = %d, want the harness block demoted", len(facts.OperatorMessages))
	}
	if len(facts.AgentText) != 1 {
		t.Fatalf("agent text = %d, want the assistant message kept", len(facts.AgentText))
	}
}

func TestToolSummarySeesRepeatedShellCommands(t *testing.T) {
	db := sqliteFixture(t)
	started := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	ended := started.Add(20 * time.Minute)
	if _, err := db.Exec(`INSERT INTO runs VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"run-shell", "adhoc", "fix", "codex", "in_place", "completed", "goal_complete", "",
		started.Format(time.RFC3339Nano), ended.Format(time.RFC3339Nano), 0); err != nil {
		t.Fatal(err)
	}
	commands := []string{
		"/bin/bash -lc 'cat /repo/api/run.go'",
		"/bin/bash -lc 'go test ./internal/api'",
		"/bin/bash -lc 'cat /repo/api/run.go'",
		"/bin/bash -lc 'go test ./internal/api'",
	}
	for index, command := range commands {
		data, _ := json.Marshal(map[string]any{"toolName": "bash", "input": map[string]any{"command": command}})
		if _, err := db.Exec(`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data) VALUES (?,?,?,?,?,?)`,
			"t"+strconv.Itoa(index), "run-shell", index+1, "tool_call", started.Format(time.RFC3339Nano), string(data)); err != nil {
			t.Fatal(err)
		}
	}
	source := &SQLiteFeatureSource{DB: db, Now: func() time.Time { return ended }}
	facts, err := source.Facts(context.Background(), ScopeRun, "run-shell")
	if err != nil {
		t.Fatal(err)
	}
	if facts.ToolSummary.RepeatedCommands != 2 {
		t.Fatalf("repeated commands = %d, want 2", facts.ToolSummary.RepeatedCommands)
	}
	// Two repeated targets: the cat file and the tested directory. The program
	// token itself must never count, or every command would look repeated.
	if facts.ToolSummary.RepeatedPaths != 2 {
		t.Fatalf("repeated paths = %d, want the two repeated targets", facts.ToolSummary.RepeatedPaths)
	}
}
