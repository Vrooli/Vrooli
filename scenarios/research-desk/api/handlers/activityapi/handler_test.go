package activityapi

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/home"
	"research-desk/internal/testutil"
	"research-desk/internal/writer"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Over the testutil archive of real research-home excerpts: local-models is a real prepared
// packet with a LEDGER.md.

const commission = "local-models-w1-20261111"

func newHandler(t *testing.T) (*handler, string, *schedule.Fake) {
	t.Helper()
	h := testutil.NewHome(t)
	research := h.Research
	clock := schedule.NewFake(time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))
	reader := home.NewReader(home.Config{ResearchDir: research, CodexDir: h.Codex}, home.Deps{Clock: clock, Procs: home.FixedProcs{}})
	w := writer.New(research, h.State, testutil.EntryChars)
	t.Cleanup(func() { w.Close() })
	return NewHandler(Deps{Reader: reader, Looks: w, EntryChars: testutil.EntryChars, Logger: log.New(io.Discard, "", 0)}), research, clock
}

// [REQ:RD-P0-013]
func TestChangesSinceTheRecordedLook(t *testing.T) {
	h, research, clock := newHandler(t)
	ctx := context.Background()
	req := &activityv1.GetChangesRequest{Subject: activityv1.ChangeSubject_CHANGE_SUBJECT_COMMISSION, Id: commission}
	first, err := h.GetChanges(ctx, connect.NewRequest(req))
	if err != nil || !first.Msg.Since.AsTime().Equal(clock.Now().Add(-24*time.Hour)) || first.Msg.BaselineKind != activityv1.Baseline_BASELINE_LAST_DAY || !strings.HasPrefix(first.Msg.Baseline, "no look recorded; the last 24 hours") {
		t.Fatalf("with no look recorded the changes are the last 24 hours' and say so: %+v %v", first, err)
	}
	// The look OperatorService.MarkSeen records.
	snap, err := h.deps.Reader.Snapshot(ctx, home.Options{MissionCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	look, err := snap.Look(home.LookCommission, commission)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.deps.Looks.(*writer.Writer).WriteState(home.LookName(home.LookCommission, commission), look); err != nil {
		t.Fatal(err)
	}

	ledger := filepath.Join(research, "cohorts", commission, home.LedgerFile)
	f, err := os.OpenFile(ledger, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\nUnit 5: nothing new.\n")
	f.Close()
	future := time.Now().Add(time.Hour)
	os.Chtimes(ledger, future, future)
	clock.Advance(time.Hour)

	second, err := h.GetChanges(ctx, connect.NewRequest(&activityv1.GetChangesRequest{Subject: req.Subject, Id: commission}))
	if err != nil {
		t.Fatal(err)
	}
	if second.Msg.Since == nil || len(second.Msg.Ledger) != 1 || second.Msg.Ledger[0].Text != "\nUnit 5: nothing new.\n" || !strings.HasPrefix(second.Msg.Baseline, "the look recorded at") {
		t.Fatalf("changes since the recorded look: %+v", second.Msg)
	}

	// --since compares with a time instead.
	since := &activityv1.GetChangesRequest{Subject: req.Subject, Id: commission, Since: timestamppb.New(time.Now().Add(30 * time.Minute))}
	third, err := h.GetChanges(ctx, connect.NewRequest(since))
	if err != nil || len(third.Msg.Files) != 1 || third.Msg.Files[0].Path != ledger || len(third.Msg.Ledger) != 0 || !third.Msg.Since.AsTime().Equal(since.Since.AsTime()) {
		t.Fatalf("changes since a time: %+v %v", third, err)
	}

	if _, err := h.GetChanges(ctx, connect.NewRequest(&activityv1.GetChangesRequest{Subject: activityv1.ChangeSubject_CHANGE_SUBJECT_TOPIC, Id: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("an unknown topic is not found: %v", err)
	}
}

func TestCommissionShowsSteeringAndEntries(t *testing.T) {
	h, research, _ := newHandler(t)
	op := "# Operator — x\n\n### 2026-11-12T10:14:03Z · direction · via hand\nModels first.\n### stray\n"
	if err := os.WriteFile(filepath.Join(research, "cohorts", commission, home.OperatorFile), []byte(op), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := h.GetCommission(context.Background(), connect.NewRequest(&activityv1.GetCommissionRequest{Id: commission}))
	if err != nil {
		t.Fatal(err)
	}
	m := resp.Msg
	if m.Steering.Value || !m.Steering.Evidence.Known {
		t.Errorf("the real local-models charter has no Steering sentence, and that is known from the charter: %+v", m.Steering)
	}
	if len(m.OperatorEntries) != 1 || m.OperatorEntries[0].Via != "hand" || len(m.OperatorUnparsedHeadings) != 1 || m.OperatorUnparsedHeadings[0].Line == 0 || m.OperatorUnparsedHeadings[0].Text == "" {
		t.Errorf("entries written by hand are read, and a heading that is not one is reported with its line: %+v unparsed=%+v", m.OperatorEntries, m.OperatorUnparsedHeadings)
	}
	if max, _ := testutil.EntryChars(); max == 0 || m.EntryCharsMax != int32(max) {
		t.Errorf("the entry limit comes from limits.json: %d, want %d", m.EntryCharsMax, max)
	}
}

// The overview shows the desk's own routes beside the lenses it read.
func TestOverviewShowsTheDesksRoutes(t *testing.T) {
	h, _, _ := newHandler(t)
	h.deps.Routes = func(context.Context) []home.Lens {
		return []home.Lens{{Name: "web-console access", Available: true, Reason: "Web Console access: local availability policy (no desk credential)"}}
	}
	resp, err := h.GetOverview(context.Background(), connect.NewRequest(&activityv1.GetOverviewRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	last := resp.Msg.Lenses[len(resp.Msg.Lenses)-1]
	if last.Name != "web-console access" || !strings.Contains(last.Reason, "local availability policy") {
		t.Fatalf("lenses = %v", resp.Msg.Lenses)
	}
}

// An unknown commission says whether the operator
// may assert it never started; every commission says when an entry is read;
// topics carry their last research write apart from mission edits.
func TestTheWireCarriesAssertionReadingAndResearchWrites(t *testing.T) {
	h, research, clock := newHandler(t)
	ctx := context.Background()
	// STATE.md rewritten after the build: unknown, with nothing found for it.
	state := filepath.Join(research, "cohorts", commission, home.StateFile)
	later := testutil.Base.Add(time.Hour)
	if err := os.Chtimes(state, later, later); err != nil {
		t.Fatal(err)
	}
	list, err := h.ListCommissions(ctx, connect.NewRequest(&activityv1.ListCommissionsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list.Msg.Commissions {
		if c.EntryReading.GetWhen() == sharedv1.ReadWhen_READ_WHEN_UNSPECIFIED || c.EntryReading.GetMessage() == "" {
			t.Errorf("%s: no entry reading", c.Id)
		}
		if c.Id == commission && (c.State != activityv1.CommissionState_COMMISSION_STATE_UNKNOWN || !c.NeverStarted.GetAssertable() || !strings.HasPrefix(c.NeverStarted.GetReason(), "no session or goal names this charter")) {
			t.Errorf("%s: %s %+v", c.Id, c.State, c.NeverStarted)
		}
		if c.State != activityv1.CommissionState_COMMISSION_STATE_UNKNOWN && c.NeverStarted != nil {
			t.Errorf("%s: only an unknown commission carries never_started", c.Id)
		}
	}

	topics, err := h.ListTopics(ctx, connect.NewRequest(&activityv1.ListTopicsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if observed := topics.Msg.GetObservedAt(); observed == nil || !observed.AsTime().Equal(clock.Now()) {
		t.Errorf("topic list carries the time its snapshot was read: got %v, want %s", observed, clock.Now())
	}
	first := topics.Msg.Topics[0]
	if w := first.LastResearchWrite; !w.GetAt().GetEvidence().GetKnown() || !strings.Contains(w.GetAt().GetEvidence().GetSource(), "/runs/") || w.GetTopic() != first.Topic {
		t.Errorf("the first topic's last research write is in a run folder: %+v", w)
	}
	overview, err := h.GetOverview(ctx, connect.NewRequest(&activityv1.GetOverviewRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if overview.Msg.LatestResearchWrite.GetTopic() != first.Topic {
		t.Errorf("the overview's latest research write is the first topic's: %+v", overview.Msg.LatestResearchWrite)
	}
}

func TestHistoricalSettlementDoesNotChangeObservedUnknownState(t *testing.T) {
	h, research, _ := newHandler(t)
	statePath := filepath.Join(research, "cohorts", commission, home.StateFile)
	later := testutil.Base.Add(time.Hour)
	if err := os.Chtimes(statePath, later, later); err != nil {
		t.Fatal(err)
	}
	activityStart, activityEnd := testutil.Base.Add(-2*time.Hour), testutil.Base.Add(-time.Hour)
	record := home.HistoricalSettlement{
		Schema: home.SettlementSchema, CommissionID: commission, Status: string(home.HistoricalClosed),
		AsOf: testutil.Base.Add(-time.Minute), ActivityStart: &activityStart, ActivityEnd: &activityEnd, RecordedAt: testutil.Base, RecordedBy: "run-42",
		Reason: "The closeout records the historical disposition.", Evidence: []string{"cohorts/" + commission + "/README.md"},
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(research, "cohorts", commission, home.SettlementFile)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := h.ListCommissions(context.Background(), connect.NewRequest(&activityv1.ListCommissionsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range resp.Msg.Commissions {
		if got.Id != commission {
			continue
		}
		if got.State != activityv1.CommissionState_COMMISSION_STATE_UNKNOWN {
			t.Fatalf("historical disposition changed current observed state to %s", got.State)
		}
		if s := got.HistoricalSettlement; s == nil || s.Status != string(home.HistoricalClosed) || len(s.EvidencePaths) != 1 || s.RecordedBy != "run-42" {
			t.Fatalf("historical disposition and citation are surfaced separately: %+v", s)
		}
		return
	}
	t.Fatalf("commission %s missing from list", commission)
}

// Topics carry their research skills and programs by prefix id, and the
// list carries the research home's own, each with what stops it resolving.
func TestListTopicsCarriesResearchSkillsAndPrograms(t *testing.T) {
	h, research, _ := newHandler(t)
	for rel, content := range map[string]string{
		"topics/local-model-performance/skills/present/SKILL.md": "---\nname: present\ndescription: Present.\n---\n",
		"topics/local-model-performance/programs/pulse.json":     `{"name":"research/pulse"}`,
		"topics/local-model-performance/programs/pulse.py":       "program.report()\n",
		"skills/research-review/SKILL.md":                        "---\nname: research-review\ndescription: Review.\n---\n",
	} {
		path := filepath.Join(research, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	topics, err := h.ListTopics(context.Background(), connect.NewRequest(&activityv1.ListTopicsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var files *activityv1.TopicWorkspaceFiles
	for _, topic := range topics.Msg.Topics {
		if topic.Topic == "local-model-performance" {
			files = topic.WorkspaceFiles
		}
	}
	if !files.GetCapabilitiesEvidence().GetKnown() || len(files.GetCapabilities()) != 2 {
		t.Fatalf("topic capabilities = %+v", files.GetCapabilities())
	}
	skill, program := files.Capabilities[0], files.Capabilities[1]
	if skill.Id != "research/local-model-performance/present" || skill.Kind != activityv1.CapabilityKind_CAPABILITY_KIND_SKILL || len(skill.Problems) != 0 || !strings.HasSuffix(skill.Path, "SKILL.md") {
		t.Errorf("skill = %+v", skill)
	}
	if program.Id != "research/local-model-performance/pulse" || program.Kind != activityv1.CapabilityKind_CAPABILITY_KIND_PROGRAM || len(program.Problems) != 1 || !strings.Contains(program.Problems[0], "research/pulse") {
		t.Errorf("a contract named for another scope is a problem: %+v", program)
	}
	supervisor := topics.Msg.HomeCapabilities
	if !topics.Msg.GetHomeCapabilitiesEvidence().GetKnown() || len(supervisor) != 1 || supervisor[0].Id != "research/research-review" {
		t.Errorf("home capabilities = %+v", supervisor)
	}
}
