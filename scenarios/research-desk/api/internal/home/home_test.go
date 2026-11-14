package home

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/testutil"
)

// The layouts come from internal/testutil's archive of real-home excerpts:
// programme-three (hand-launched, dated amendments, numbered needs list),
// local-models (prepared), standing-teams (prepared, one MISSION.md link) and
// research-night (an older layout with STATUS.md). Codex records are written
// per test with the real thread_goals schema, because their objectives must
// name the expanded copy's charter path.

var base = testutil.Base

type fixture struct {
	research string
	codex    string
	clock    *schedule.Fake
	procs    *fakeProcs
	missions MissionChecker
	ceilings func() (*Ceilings, error)
	// others are the Codex homes outside the Web Console session folder.
	others []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	h := testutil.NewHome(t)
	return &fixture{research: h.Research, codex: h.Codex, clock: schedule.NewFake(base.Add(2 * time.Hour)), procs: &fakeProcs{known: true}}
}

func (f *fixture) snapshot(t *testing.T, opts Options) *Snapshot {
	t.Helper()
	reader := NewReader(Config{ResearchDir: f.research, CodexDir: f.codex, OtherCodexHomes: f.others}, Deps{Clock: f.clock, Procs: f.procs, Missions: f.missions, Ceilings: f.ceilings})
	snap, err := reader.Snapshot(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func (f *fixture) charter(id string) string {
	return filepath.Join(f.research, "cohorts", id, CharterFile)
}

// touch sets one fixture file's modification time.
func (f *fixture) touch(t *testing.T, rel string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(filepath.Join(f.research, rel), at, at); err != nil {
		t.Fatal(err)
	}
}

type goalRow struct {
	thread, objective, status string
	created                   time.Time
}

type historyRow struct {
	thread, text string
	at           time.Time
}

// codexHome writes one Codex home with the real goals_1.sqlite schema.
func (f *fixture) codexHome(t *testing.T, session string, goals []goalRow, history []historyRow) string {
	t.Helper()
	return writeCodexHome(t, filepath.Join(f.codex, session), goals, history)
}

// writeCodexHome writes a Codex home at dir.
func writeCodexHome(t *testing.T, dir string, goals []goalRow, history []historyRow) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "goals_1.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE thread_goals (
    thread_id TEXT PRIMARY KEY NOT NULL,
    goal_id TEXT NOT NULL,
    objective TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('active','paused','blocked','usage_limited','budget_limited','complete')),
    token_budget INTEGER,
    tokens_used INTEGER NOT NULL DEFAULT 0,
    time_used_seconds INTEGER NOT NULL DEFAULT 0,
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for _, g := range goals {
		if _, err := db.Exec(`INSERT INTO thread_goals (thread_id, goal_id, objective, status, created_at_ms, updated_at_ms) VALUES (?,?,?,?,?,?)`,
			g.thread, "goal-"+g.thread, g.objective, g.status, g.created.UnixMilli(), g.created.UnixMilli()); err != nil {
			t.Fatal(err)
		}
	}
	var lines []string
	for _, h := range history {
		line, _ := json.Marshal(map[string]any{"session_id": h.thread, "ts": h.at.Unix(), "text": h.text})
		lines = append(lines, string(line))
	}
	if err := os.WriteFile(filepath.Join(dir, "history.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// programmeThreeGoal is the real goal objective of programme-three, with the
// charter path pointing at the fixture copy.
func (f *fixture) programmeThreeGoal() string {
	return "Work until every card in " + f.charter("programme-three-20261111") +
		" lands its listed showcases in order, then the next open ones, or a fresh challenger (LONG_EFFORT_DESIGN.md §2a) returns NONE; 24h allowance."
}

type fakeProcs struct {
	processes []Process
	known     bool
	reason    string
}

func (p *fakeProcs) Codex() ([]Process, bool, string) { return p.processes, p.known, p.reason }

func (p *fakeProcs) window(home string) {
	p.processes = append(p.processes, Process{PID: 100 + len(p.processes), CodexHome: home})
}

func TestSnapshotRealFolders(t *testing.T) {
	f := newFixture(t)
	started := base.Add(-time.Hour)
	home := f.codexHome(t, "b7117dc2", []goalRow{{"thread-p3", f.programmeThreeGoal(), "paused", started}},
		[]historyRow{{"thread-p3", "Research commission, third window. Charter: " + f.charter("programme-three-20261111"), started.Add(-time.Minute)}})
	f.procs.window(home)
	f.procs.processes = append(f.procs.processes, Process{PID: 9, CodexHome: filepath.Join(f.codex, "other"), AppServer: true})

	snap := f.snapshot(t, Options{})
	if len(snap.Commissions) != 4 {
		t.Fatalf("want 4 commissions, got %d", len(snap.Commissions))
	}

	p3 := snap.Commission("programme-three-20261111")
	if p3.State != StatePaused {
		t.Fatalf("programme-three: want paused, got %s (%s)", p3.State, p3.StateRule)
	}
	if p3.Session == nil || p3.Session.Basis != "goal-objective" || p3.Session.Source != filepath.Join(home, "goals_1.sqlite") {
		t.Fatalf("programme-three: want exact goal-objective match from goals_1.sqlite, got %+v", p3.Session)
	}
	if len(p3.Matches) != 1 {
		t.Errorf("the stance prompt in the same thread must not add a related match: %+v", p3.Matches)
	}
	wantTopics := []string{"medical-practice-it-modernization", "niche-book-production", "ai-video-production-and-funnels"}
	if got := topicNames(p3.Topics); strings.Join(got, ",") != strings.Join(wantTopics, ",") {
		t.Errorf("programme-three topics: want %v from cards, got %v", wantTopics, got)
	}
	for _, ref := range p3.Topics {
		if ref.Basis != "card" || ref.Source != f.charter("programme-three-20261111") {
			t.Errorf("topic %s: want card basis from the charter, got %s %s", ref.Topic, ref.Basis, ref.Source)
		}
	}
	kernel := filepath.Join(f.research, "cohorts/programme-three-20261111/KERNEL.txt")
	if p3.Allowance != 24*time.Hour || p3.AllowanceEv.Source != kernel {
		t.Errorf("allowance: want 24h from KERNEL.txt, got %s from %q", p3.Allowance, p3.AllowanceEv.Source)
	}
	if !p3.AllowanceEnd.Value.Equal(started.Add(24*time.Hour)) || p3.AllowanceEnd.Source != kernel {
		t.Errorf("allowance end: want goal start + 24h naming KERNEL.txt, got %s %q", p3.AllowanceEnd.Value, p3.AllowanceEnd.Source)
	}
	if !p3.Started.Value.Equal(started.Truncate(time.Millisecond)) || !p3.Started.Known {
		t.Errorf("started: want %s, got %+v", started, p3.Started)
	}
	if p3.Title.Source != f.charter("programme-three-20261111") || p3.Title.Value == "" {
		t.Errorf("title must come from the charter heading, got %+v", p3.Title)
	}
	if p3.Claim.Source != filepath.Join(f.research, "cohorts/programme-three-20261111/STATE.md") || p3.Claim.Value == "" {
		t.Errorf("claim must come from STATE.md, got %+v", p3.Claim)
	}
	if len(p3.Runs) != 1 || p3.Runs[0].Checkpoint == nil {
		t.Errorf("want the ai-video run folder with its checkpoint, got %+v", p3.Runs)
	}

	if c := snap.Commission("local-models-w1-20261111"); c.State != StatePrepared {
		t.Errorf("local-models: want prepared, got %s (%s)", c.State, c.StateRule)
	} else if c.AllowanceEv.Known {
		t.Errorf("local-models kernel names no allowance; want unknown, got %+v", c.AllowanceEv)
	}

	st := snap.Commission("standing-teams-w1-20261110")
	if st.State != StatePrepared {
		t.Errorf("standing-teams: want prepared, got %s", st.State)
	}
	if len(st.Topics) != 1 || st.Topics[0].Topic != "standing-teams-work-routes" || st.Topics[0].Basis != "charter-mission-link" {
		t.Errorf("standing-teams: want its one topic from the charter's MISSION.md link, got %+v", st.Topics)
	}

	night := snap.Commission("research-night-20261031")
	if night.State != StateUnknown || night.Session != nil {
		t.Errorf("research-night: an older folder no session names must be unknown, got %s", night.State)
	}
	if night.Claim.Source != filepath.Join(f.research, "cohorts/research-night-20261031/STATUS.md") {
		t.Errorf("research-night claim must fall back to STATUS.md, got %+v", night.Claim)
	}
	if night.TopicsEvidence.Known || night.Started.Known || night.AllowanceEnd.Known {
		t.Errorf("research-night: topics, start and allowance end cannot be derived and must be unknown")
	}

	if got := snap.Topic("ai-video-production-and-funnels"); got == nil || strings.Join(got.Paused, ",") != "programme-three-20261111" || len(got.Live)+len(got.Limited) != 0 {
		t.Errorf("ai-video must list programme-three as paused, not live, got %+v", got)
	}
	for _, lens := range snap.Lenses {
		if lens.Name == "codex sessions" && !lens.Available {
			t.Errorf("codex lens should be available: %+v", lens)
		}
	}
}

// [REQ:RD-P0-004]
func TestEveryDerivedFieldNamesItsSourceOrIsUnknown(t *testing.T) {
	f := newFixture(t)
	home := f.codexHome(t, "s1", []goalRow{{"t1", f.programmeThreeGoal(), "active", base}}, nil)
	f.procs.window(home)
	snap := f.snapshot(t, Options{})
	for _, c := range snap.Commissions {
		for name, ev := range map[string]Evidence{
			"title": c.Title.Evidence, "claim": c.Claim.Evidence, "topics": c.TopicsEvidence,
			"allowance": c.AllowanceEv, "started": c.Started.Evidence, "allowance end": c.AllowanceEnd.Evidence,
			"last write": c.LastWrite.Evidence,
		} {
			if ev.Known && ev.Source == "" {
				t.Errorf("%s %s: known without a source", c.ID, name)
			}
			if !ev.Known && ev.Reason == "" {
				t.Errorf("%s %s: unknown without a reason", c.ID, name)
			}
		}
		if c.StateRule == "" {
			t.Errorf("%s: state %s without its rule", c.ID, c.State)
		}
	}
}

// [REQ:RD-P0-004]
func TestStateFromSessionRecords(t *testing.T) {
	const p3 = "programme-three-20261111"
	cases := []struct {
		name    string
		setup   func(t *testing.T, f *fixture)
		want    State
		wantWhy string
	}{
		{"window open and goal active", func(t *testing.T, f *fixture) {
			f.procs.window(f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "active", base}}, nil))
		}, StateLive, "goal active"},
		{"window open and usage limited", func(t *testing.T, f *fixture) {
			f.procs.window(f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "usage_limited", base}}, nil))
		}, StateLimited, "usage_limited"},
		{"no window, allowance left", func(t *testing.T, f *fixture) {
			f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "paused", base}}, nil)
		}, StateStopped, "no supervisor window"},
		{"no window, allowance passed", func(t *testing.T, f *fixture) {
			f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "paused", base}}, nil)
			f.clock.Advance(25 * time.Hour)
		}, StateEnded, "allowance ended"},
		{"window open, allowance passed: still paused", func(t *testing.T, f *fixture) {
			f.procs.window(f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "paused", base}}, nil))
			f.clock.Advance(25 * time.Hour)
		}, StatePaused, "goal paused"},
		{"goal complete", func(t *testing.T, f *fixture) {
			f.procs.window(f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "complete", base}}, nil))
		}, StateEnded, "complete"},
		{"process table unreadable", func(t *testing.T, f *fixture) {
			f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "active", base}}, nil)
			f.procs.known, f.procs.reason = false, "process table unreadable: test"
		}, StateUnknown, "process table unreadable"},
		{"discarded in LAUNCH.json", func(t *testing.T, f *fixture) {
			f.procs.window(f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "active", base}}, nil))
			writeFile(t, filepath.Join(f.research, "cohorts", p3, LaunchFile), `{"schema":"research-desk.launch/v1","commission_id":"`+p3+`","launched_by":"hand","steps":[],"discarded":{"at":"2026-11-10T13:00:00Z","reason":"bad start"},"controls":[]}`)
		}, StateEnded, "discarded"},
		{"/goal prompt names the charter, goal row elsewhere", func(t *testing.T, f *fixture) {
			f.procs.window(f.codexHome(t, "s", []goalRow{{"t", "Read the Codex goal objective file at /elsewhere/goal-objective.md before continuing.", "active", base}},
				[]historyRow{{"t", "/goal Work until every card in " + f.charter(p3) + " lands", base}}))
		}, StateLive, "goal active"},
		{"only a preparation chat names the folder", func(t *testing.T, f *fixture) {
			f.codexHome(t, "s", nil, []historyRow{{"t", "prepare cohorts/" + p3 + " please", base}})
		}, StateUnknown, "only preparation prompts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.setup(t, f)
			c := f.snapshot(t, Options{}).Commission(p3)
			if c.State != tc.want || !strings.Contains(c.StateRule, tc.wantWhy) {
				t.Fatalf("want %s (%q), got %s (%q)", tc.want, tc.wantWhy, c.State, c.StateRule)
			}
		})
	}
}

func TestObjectiveAttachmentIsRead(t *testing.T) {
	f := newFixture(t)
	home := filepath.Join(f.codex, "s")
	attachment := filepath.Join(home, "attachments", "a1", "goal-objective.md")
	writeFile(t, attachment, f.programmeThreeGoal())
	f.codexHome(t, "s", []goalRow{{"t", "Read the Codex goal objective file at " + attachment + " before continuing.", "paused", base}}, nil)
	f.procs.window(home)
	c := f.snapshot(t, Options{}).Commission("programme-three-20261111")
	if c.Session == nil || c.Session.Kind != MatchExact || c.Session.Basis != "goal-objective-file" {
		t.Fatalf("want an exact goal-objective-file match, got %+v", c.Matches)
	}
}

func TestRecognitionWindowIgnoresOlderGoals(t *testing.T) {
	f := newFixture(t)
	// The window starts at local midnight one day before the folder's
	// earliest file time.
	f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "active", base.Add(-72 * time.Hour)}}, nil)
	c := f.snapshot(t, Options{}).Commission("programme-three-20261111")
	if len(c.Matches) != 0 {
		t.Fatalf("a goal three days older than the folder must be outside the window, got %+v", c.Matches)
	}
}

func TestUnreadableRecordsStillDisplay(t *testing.T) {
	f := newFixture(t)
	writeFile(t, filepath.Join(f.research, "cohorts/programme-three-20261111", LaunchFile), "{not json")
	writeFile(t, filepath.Join(f.research, "cohorts/programme-three-20261111", PacketFile), `{"schema":"other/v9"}`)
	c := f.snapshot(t, Options{}).Commission("programme-three-20261111")
	if len(c.RecordErrors) != 2 {
		t.Fatalf("want both record errors reported, got %v", c.RecordErrors)
	}
	if c.Title.Value == "" {
		t.Fatalf("the folder must still display")
	}
}

// [REQ:RD-P0-010]
func TestDeskRecordsAreRead(t *testing.T) {
	f := newFixture(t)
	const id = "local-models-w1-20261111"
	writeFile(t, filepath.Join(f.research, "cohorts", id, PacketFile), `{"schema":"research-desk.packet/v1","commission_id":"`+id+`","built_at":"2026-11-11T16:26:00Z","template_version":"1","spec":{"topics":["local-model-performance","niche-book-production"],"allowance_hours":12}}`)
	writeFile(t, filepath.Join(f.research, "cohorts", id, LaunchFile), `{"schema":"research-desk.launch/v1","commission_id":"`+id+`","launched_by":"desk","web_console_session_id":"s","thread_id":"t","steps":[{"step":"created","at":"2026-11-10T12:00:00Z"}],"failed":null,"discarded":null,"controls":[]}`)
	home := f.codexHome(t, "s", []goalRow{{"t", "unrelated words", "active", base}}, nil)
	f.procs.window(home)
	c := f.snapshot(t, Options{}).Commission(id)
	if c.LaunchedBy() != "desk" || c.Session == nil || c.Session.Basis != "launch-record" {
		t.Fatalf("want the launch record to name the session, got %+v", c.Session)
	}
	if c.State != StateLive {
		t.Errorf("want live from the launch record's thread goal, got %s (%s)", c.State, c.StateRule)
	}
	if got := topicNames(c.Topics); strings.Join(got, ",") != "local-model-performance,niche-book-production" || c.Topics[0].Basis != "packet" {
		t.Errorf("want packet topics first, got %+v", c.Topics)
	}
	if c.Allowance != 12*time.Hour || c.AllowanceEv.Source != filepath.Join(f.research, "cohorts", id, PacketFile) {
		t.Errorf("want the packet allowance, got %s %+v", c.Allowance, c.AllowanceEv)
	}
}

func TestTopics(t *testing.T) {
	f := newFixture(t)
	if err := os.Symlink("niche-book-production", filepath.Join(f.research, "topics", "alias")); err != nil {
		t.Fatal(err)
	}
	f.touch(t, "topics/ai-video-production-and-funnels/runs/programme-three-20261111/CHECKPOINT.md", base.Add(time.Hour))
	snap := f.snapshot(t, Options{})
	if alias := snap.Topic("alias"); alias == nil || alias.Slug != "niche-book-production" || !slices.Equal(alias.Aliases, []string{"alias"}) {
		t.Errorf("a link to a sibling topic is that topic's alias, listed once: %+v", alias)
	}
	if len(snap.Topics) != 5 {
		t.Errorf("want 5 topics, got %d", len(snap.Topics))
	}
	first := snap.Topics[0]
	if first.Slug != "ai-video-production-and-funnels" || !strings.HasSuffix(first.LastResearchWrite.Source, "CHECKPOINT.md") || first.LastResearchWrite.CommissionID != "programme-three-20261111" {
		t.Errorf("want ai-video first, written through its run folder's checkpoint, got %s %+v", first.Slug, first.LastResearchWrite)
	}
	niche := snap.Topic("niche-book-production")
	if niche.Mission == nil || niche.Title.Source != niche.Mission.Path {
		t.Errorf("niche-book title must come from MISSION.md, got %+v", niche.Title)
	}
	if medical := snap.Topic("medical-practice-it-modernization"); medical.Mission != nil || !strings.HasSuffix(medical.Title.Source, "README.md") {
		t.Errorf("without MISSION.md the title comes from README.md, got %+v", medical.Title)
	}
	if niche.Check.Known {
		t.Errorf("mission check not requested: must be unknown")
	}
}

type fakeMissions struct {
	out []byte
	err error
}

func (m fakeMissions) MissionCheck(context.Context, string) ([]byte, string, error) {
	return m.out, "/skills/mission_check.py", m.err
}

func TestMissionCheck(t *testing.T) {
	f := newFixture(t)
	f.missions = fakeMissions{out: []byte(`[{"topic":"niche-book-production","mode":"build","workstreams":{"open":3,"active":1,"waiting-operator":1,"landed":2},"workstream_rows":[{"id":"NB1","status":"active","showcase":"a book"}],"errors":[],"warnings":["w"]}]`)}
	snap := f.snapshot(t, Options{MissionCheck: true})
	niche := snap.Topic("niche-book-production").Check
	if !niche.Known || niche.Open != 3 || niche.WaitingOperator != 1 || len(niche.Workstreams) != 1 || niche.Source != "/skills/mission_check.py" {
		t.Errorf("want the script's summary, got %+v", niche)
	}
	if other := snap.Topic("ai-video-production-and-funnels").Check; other.Known || !strings.Contains(other.Reason, "did not report") {
		t.Errorf("a topic the script did not report must be unknown, got %+v", other)
	}

	f.missions = fakeMissions{err: errors.New("python3 missing")}
	snap = f.snapshot(t, Options{MissionCheck: true})
	if check := snap.Topic("niche-book-production").Check; check.Known || !strings.Contains(check.Reason, "python3 missing") {
		t.Errorf("a failing script leaves every check unknown with the reason, got %+v", check)
	}
}

// [REQ:RD-P0-007]
// The read sets are limits.json's lists, with the commission's own files
// in place of its commission items and the measured skill text for the rest.
func TestReadSetsFollowTheLimitsLists(t *testing.T) {
	f := newFixture(t)
	dir := filepath.Join(f.research, "cohorts", "programme-three-20261111")
	size := func(name string) int64 {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return info.Size()
	}
	charter, state := size(CharterFile), size(StateFile)
	gate := int(charter) - 1
	brief := FileFact{Name: "owner-brief.md", Path: "skills/owner-brief.md", Bytes: 3000}
	f.ceilings = func() (*Ceilings, error) {
		return &Ceilings{
			CharterBytes: gate,
			Roles: []RoleReads{
				{Key: "scheduler_start", Items: []string{"charter", "STATE.md", "brief", "gone.md-skill/x.md"}, CeilingBytes: 1 << 20},
				{Key: "scheduler_unit", Items: []string{"charter", "STATE.md"}, CeilingBytes: charter + state - 1},
				{Key: OwnerStartKey, Items: []string{"brief"}, CeilingBytes: 1 << 20},
			},
			Fixed:        map[string]FileFact{"brief": brief},
			Unmeasured:   map[string]string{"gone.md-skill/x.md": "not in the skills folder"},
			OwnerTopic:   []string{MissionFile},
			MissionBytes: 100,
			Source:       "limits.json",
		}, nil
	}
	snap := f.snapshot(t, Options{})
	c := snap.Commission("programme-three-20261111")
	sets := snap.ReadSets(c)
	start, unit := sets[0], sets[1]
	if start.Role != "scheduler-start" || start.Bytes != charter+state+brief.Bytes || start.EstimatedTokens != start.Bytes/4 {
		t.Fatalf("want charter + STATE.md + the brief, got %+v", start)
	}
	flags := strings.Join(start.Flags, "\n")
	for _, want := range []string{fmt.Sprintf("charter %d bytes is over the %d-byte gate", charter, gate), "dated amendment headings", "gone.md-skill/x.md not counted: not in the skills folder"} {
		if !strings.Contains(flags, want) {
			t.Errorf("want %q among the start flags, got %q", want, flags)
		}
	}
	if unit.Role != "scheduler-unit" || len(unit.Flags) != 1 || !strings.Contains(unit.Flags[0], fmt.Sprintf("%d bytes is over the %d-byte scheduler_unit ceiling", charter+state, charter+state-1)) {
		t.Errorf("the unit set is flagged over its ceiling and only that, got %+v", unit)
	}
	if len(sets) != 2+len(c.Topics) || sets[2].Role != "owner-start:"+c.Topics[0].Topic {
		t.Fatalf("want the owner set once per topic, got %+v", sets)
	}
	// Each owner also reads its own topic's mission: added to its total,
	// never to the skill-text ceiling, and flagged with the tidy when large.
	withMission := 0
	for i, ref := range c.Topics {
		owner := sets[2+i]
		mission, err := os.Stat(filepath.Join(f.research, "topics", ref.Topic, MissionFile))
		if err != nil {
			if len(owner.Files) != 1 || !strings.Contains(strings.Join(owner.Flags, " "), MissionFile+" not counted") {
				t.Errorf("%s: a missing mission is said, not counted as zero: %+v", owner.Role, owner)
			}
			continue
		}
		withMission++
		if len(owner.Files) != 2 || owner.Files[1].Bytes != mission.Size() || owner.Bytes != brief.Bytes+mission.Size() {
			t.Errorf("%s reads the brief and its own mission, got %+v", owner.Role, owner)
		}
		flags := strings.Join(owner.Flags, "\n")
		if strings.Contains(flags, "owner_start ceiling") || !strings.Contains(flags, "the next architect pass tidies the mission") || !strings.Contains(flags, ref.Topic+" mission is") {
			t.Errorf("%s: the mission is flagged with the tidy and never held to the skill ceiling, got %q", owner.Role, flags)
		}
	}
	if withMission == 0 {
		t.Fatal("the fixture needs a topic with a mission")
	}
	reading := snap.OwnerReading([]string{c.Topics[0].Topic, "no-such-topic"})
	if len(reading) != 2 || reading[0].Bytes != sets[2].Bytes || !strings.Contains(strings.Join(reading[1].Flags, " "), "no topic no-such-topic") {
		t.Errorf("a spec's topics get the same owner reading, an unknown topic says so: %+v", reading)
	}

	f.ceilings = func() (*Ceilings, error) { return nil, errors.New("limits.json missing") }
	snap = f.snapshot(t, Options{})
	if sets := snap.ReadSets(snap.Commission("programme-three-20261111")); sets != nil {
		t.Errorf("without limits.json there is no list to measure, got %+v", sets)
	}
}

func TestOverlaps(t *testing.T) {
	f := newFixture(t)
	const id = "local-models-w1-20261111"
	writeFile(t, filepath.Join(f.research, "cohorts", id, PacketFile), `{"schema":"research-desk.packet/v1","commission_id":"`+id+`","spec":{"topics":["niche-book-production"],"allowance_hours":12}}`)
	other := "Work until every card in " + f.charter(id) + " lands."
	f.procs.window(f.codexHome(t, "a", []goalRow{{"t", f.programmeThreeGoal(), "active", base}}, nil))
	f.procs.window(f.codexHome(t, "b", []goalRow{{"t", other, "active", base.Add(time.Hour)}}, nil))
	snap := f.snapshot(t, Options{})
	overlaps := snap.Overlaps()
	if len(overlaps) != 1 || overlaps[0].Topic != "niche-book-production" || len(overlaps[0].CommissionIDs) != 2 {
		t.Fatalf("want one overlap on niche-book-production, got %+v", overlaps)
	}
}

// A commission whose end is not known ends at its last write, never now, so
// old stopped commissions do not overlap everything after them; "open now"
// keeps only pairs that both have a window open.
func TestAnUnknownEndIsTheLastWriteNotNow(t *testing.T) {
	known := func(at time.Time) Time { return Time{Value: at, Evidence: Evidence{Known: true}} }
	topic := []TopicRef{{Topic: "t"}}
	commission := func(id string, state State, start, last time.Time) *Commission {
		return &Commission{ID: id, State: state, Topics: topic, Started: known(start), LastWrite: known(last)}
	}
	now := base.Add(10 * 24 * time.Hour)
	snap := &Snapshot{Now: now, Commissions: []*Commission{
		commission("old", StateStopped, base, base.Add(5*time.Hour)),
		commission("later", StateStopped, base.Add(24*time.Hour), base.Add(30*time.Hour)),
		commission("live-a", StateLive, base.Add(9*24*time.Hour), now),
		commission("live-b", StatePaused, base.Add(9*24*time.Hour+time.Hour), now),
	}}
	all := snap.Overlaps()
	if len(all) != 1 || strings.Join(all[0].CommissionIDs, " ") != "live-a live-b" {
		t.Errorf("stopped commissions end at their last write, so only the two open ones overlap: %+v", all)
	}
	open := snap.OpenOverlaps()
	if len(open) != 1 || strings.Join(open[0].CommissionIDs, " ") != "live-a live-b" {
		t.Errorf("open now: %+v", open)
	}
	snap.Commissions[1].Started = known(base.Add(4 * time.Hour))
	if got := snap.Overlaps(); len(got) != 2 || len(snap.OpenOverlaps()) != 1 {
		t.Errorf("a real overlap of two stopped commissions is history, not open now: %+v", got)
	}
}

func TestMissingResearchHome(t *testing.T) {
	reader := NewReader(Config{ResearchDir: filepath.Join(t.TempDir(), "absent")}, Deps{Clock: schedule.NewFake(base), Procs: &fakeProcs{known: true}})
	if _, err := reader.Snapshot(context.Background(), Options{}); !errors.Is(err, ErrNoResearchHome) {
		t.Fatalf("want ErrNoResearchHome, got %v", err)
	}
}

func TestCodexCacheSeesWrites(t *testing.T) {
	f := newFixture(t)
	home := f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), "active", base}}, nil)
	f.procs.window(home)
	reader := NewReader(Config{ResearchDir: f.research, CodexDir: f.codex}, Deps{Clock: f.clock, Procs: f.procs})
	snap, _ := reader.Snapshot(context.Background(), Options{})
	if got := snap.Commission("programme-three-20261111").State; got != StateLive {
		t.Fatalf("want live, got %s", got)
	}
	db, err := sql.Open("sqlite", filepath.Join(home, "goals_1.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE thread_goals SET status='paused'`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	later := time.Now().Add(time.Minute)
	_ = os.Chtimes(filepath.Join(home, "goals_1.sqlite"), later, later)
	snap, _ = reader.Snapshot(context.Background(), Options{})
	if got := snap.Commission("programme-three-20261111").State; got != StatePaused {
		t.Fatalf("the cache must reread a changed goals database: want paused, got %s", got)
	}
}

func topicNames(refs []TopicRef) []string {
	var out []string
	for _, ref := range refs {
		out = append(out, ref.Topic)
	}
	return out
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// [REQ:RD-P0-002]
// A complete packet is `prepared` only on evidence that nothing started
// it; otherwise `unknown` with the reason.
func TestPreparedNeedsEvidenceOfAbsence(t *testing.T) {
	const id = "local-models-w1-20261111"
	state := func(f *fixture) *Commission {
		t.Helper()
		return f.snapshot(t, Options{}).Commission(id)
	}

	f := newFixture(t)
	agentRuns := filepath.Join(filepath.Dir(f.codex), "agent-manager", "runs")
	f.others = []string{filepath.Join(filepath.Dir(f.codex), "dot-codex"), filepath.Join(agentRuns, "*", "codex")}
	writeCodexHome(t, filepath.Join(agentRuns, "run-1", "codex"), []goalRow{{"t1", "Unrelated work", "active", base}}, nil)
	c := state(f)
	if c.State != StatePrepared || !strings.Contains(c.StateRule, "no goal names this charter") {
		t.Fatalf("no goal anywhere, files unchanged: %s (%s)", c.State, c.StateRule)
	}
	var proof bool
	for _, e := range c.StateEvidence {
		proof = proof || (e.Known && strings.Contains(e.Source, "1 other Codex homes"))
	}
	if !proof {
		t.Errorf("the evidence names the homes read: %+v", c.StateEvidence)
	}

	// A goal in an Agent Manager home names the charter.
	f = newFixture(t)
	agentRuns = filepath.Join(filepath.Dir(f.codex), "agent-manager", "runs")
	f.others = []string{filepath.Join(agentRuns, "*", "codex")}
	writeCodexHome(t, filepath.Join(agentRuns, "run-2", "codex"), []goalRow{{"t2", "Work through " + f.charter(id), "active", base}}, nil)
	if c := state(f); c.State != StateUnknown || !strings.Contains(c.StateRule, "run-2/codex/goals_1.sqlite (thread t2, active) names this charter") {
		t.Fatalf("a goal elsewhere: %s (%s)", c.State, c.StateRule)
	} else if c.NeverStarted.Assertable || !strings.Contains(c.NeverStarted.Reason, "names this charter") {
		t.Fatalf("a goal found anywhere refuses the never-started assertion: %+v", c.NeverStarted)
	}

	// A goal database that cannot be read proves nothing.
	f = newFixture(t)
	dotCodex := filepath.Join(filepath.Dir(f.codex), "dot-codex")
	f.others = []string{dotCodex}
	if err := os.MkdirAll(dotCodex, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dotCodex, "goals_1.sqlite"), []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := state(f); c.State != StateUnknown || !strings.Contains(c.StateRule, "other Codex homes unreadable") {
		t.Fatalf("an unreadable home: %s (%s)", c.State, c.StateRule)
	} else if !c.NeverStarted.Assertable || !strings.Contains(c.NeverStarted.Reason, "not read: 1 other Codex homes") {
		t.Fatalf("what the desk cannot read, the operator may assert, and the unread homes are named: %+v", c.NeverStarted)
	}

	// LEDGER.md written after the build.
	f = newFixture(t)
	f.touch(t, "cohorts/"+id+"/"+LedgerFile, base.Add(time.Minute))
	if c := state(f); c.State != StateUnknown || !strings.Contains(c.StateRule, "LEDGER.md was written at") {
		t.Fatalf("a ledger written after the build: %s (%s)", c.State, c.StateRule)
	} else if !c.NeverStarted.Assertable || !strings.HasPrefix(c.NeverStarted.Reason, "no session or goal names this charter in ") {
		t.Fatalf("with no session or goal found, the operator may assert it never started: %+v", c.NeverStarted)
	}
	// A recorded launch attempt is never asserted away.
	writeFile(t, filepath.Join(f.research, "cohorts", id, LaunchFile), `{"schema":"`+LaunchSchema+`","commission_id":"`+id+`","launched_by":"desk"}`)
	if c := state(f); c.State == StatePrepared || c.NeverStarted.Assertable {
		t.Fatalf("a LAUNCH.json refuses the assertion: %s %+v", c.State, c.NeverStarted)
	}
}

// One name rule, equal to packet.py's NAME, read from packet.py itself.
func TestTheNameRuleIsPacketPys(t *testing.T) {
	data, err := os.ReadFile("../../../skills/research-coordination/scripts/packet.py")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^NAME = re\.compile\(r"([^"]+)"\)`).FindSubmatch(data)
	if m == nil || string(m[1]) != NamePattern {
		t.Fatalf("packet.py NAME = %q, Go NamePattern = %q", m, NamePattern)
	}
	f := newFixture(t)
	for _, name := range []string{"Fraud.PPP_2", "x"} {
		if err := os.MkdirAll(filepath.Join(f.research, "topics", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	snap := f.snapshot(t, Options{})
	if snap.Topic("Fraud.PPP_2") == nil || snap.Topic("x") == nil {
		t.Fatal("every topic folder packet.py accepts is listed")
	}
	if !IsEntryHeading("### 2026-11-12T09:00:00Z · verdict Fraud.PPP_2 · via desk") {
		t.Fatal("a verdict heading takes every topic name")
	}
}

// An older charter that links a topic's README.md or its folder names
// the topic; a deeper path does not.
func TestOlderCharterLinksNameTheTopic(t *testing.T) {
	f := newFixture(t)
	id := "research-night-20261031"
	charter := f.charter(id)
	data, err := os.ReadFile(charter)
	if err != nil {
		t.Fatal(err)
	}
	extra := "\nSee [the topic](" + f.research + "/topics/niche-book-production/README.md) and `" + f.research + "/topics/local-model-performance/`.\n" +
		"Background: " + f.research + "/topics/ai-video-production-and-funnels/runs/old/CHECKPOINT.md\n"
	if err := os.WriteFile(charter, append(data, extra...), 0o644); err != nil {
		t.Fatal(err)
	}
	c := f.snapshot(t, Options{}).Commission(id)
	got := map[string]string{}
	for _, ref := range c.Topics {
		got[ref.Topic] = ref.Basis
	}
	if got["niche-book-production"] != "charter-topic-link" || got["local-model-performance"] != "charter-topic-link" {
		t.Errorf("README.md and folder links name the topic: %+v", c.Topics)
	}
	if _, ok := got["ai-video-production-and-funnels"]; ok {
		t.Errorf("a run-folder path inside a topic is not a topic link: %+v", c.Topics)
	}
}

// PACKET.json supersedes: packets built before lists hold one id or null;
// packet.py now records a list. All three read, and anything else is refused.
func TestASupersedesListAndAnOlderSingleIDBothRead(t *testing.T) {
	dir := t.TempDir()
	for name, tc := range map[string]struct {
		spec string
		want []string
	}{
		"older single id": {`"supersedes":"programme-three-20261111"`, []string{"programme-three-20261111"}},
		"older null":      {`"supersedes":null`, nil},
		"list of two":     {`"supersedes":["programme-three-20261111","programme-four-20261110"]`, []string{"programme-three-20261111", "programme-four-20261110"}},
		"empty list":      {`"supersedes":[]`, []string{}},
	} {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".json")
		record := `{"schema":"` + PacketSchema + `","commission_id":"c","spec":{"topics":["t"],` + tc.spec + `}}`
		if err := os.WriteFile(path, []byte(record), 0o644); err != nil {
			t.Fatal(err)
		}
		_, spec, err := ReadPacket(path)
		if err != nil || len(spec.Supersedes) != len(tc.want) || (len(tc.want) > 0 && !slices.Equal([]string(spec.Supersedes), tc.want)) {
			t.Errorf("%s: supersedes = %#v, %v; want %v", name, spec, err, tc.want)
		}
	}
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{"schema":"`+PacketSchema+`","commission_id":"c","spec":{"supersedes":7}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadPacket(path); err == nil || !strings.Contains(err.Error(), "neither a commission id nor a list") {
		t.Fatalf("a number: err = %v", err)
	}
}

// A review's topics come from its per-topic headings, else its links into
// topics/; a review that names none either way has unknown topics, never
// an empty list that reads as none.
func TestAReviewsTopicsAreKnownOrSaidUnknown(t *testing.T) {
	topicOf := func(name string) (string, bool) {
		return name, name == "local-model-performance" || name == "debt-reduction"
	}
	for name, tc := range map[string]struct {
		text  string
		want  []string
		known bool
	}{
		"headings":     {"# R\n\n## debt-reduction\n\nSee topics/local-model-performance/STATE.md.\n", []string{"debt-reduction"}, true},
		"links only":   {"# R\n\n## Open\n\nSee topics/local-model-performance/STATE.md and topics/local-model-performance/MISSION.md.\n", []string{"local-model-performance"}, true},
		"neither":      {"# R\n\n## Open\n\nAll fifteen workstreams remain.\n", nil, false},
		"unknown link": {"# R\n\nSee topics/not-a-topic/.\n", nil, false},
	} {
		got, ev := reviewTopics("REVIEW.md", []byte(tc.text), topicOf)
		if !slices.Equal(got, tc.want) || ev.Known != tc.known || (!ev.Known && ev.Reason == "") {
			t.Errorf("%s: topics %v, evidence %+v", name, got, ev)
		}
	}
}
