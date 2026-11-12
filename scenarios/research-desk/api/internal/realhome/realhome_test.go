//go:build realhome

// Package realhome holds the one read-only test over the operator's real
// research home. Run it with `make real-home-check`. An absent home fails the
// check, because a check that passes without looking proves nothing; on a
// machine with no research home, REAL_HOME_CHECK_SKIP=1 turns that into a
// loud skip. It never writes the home: the courier's
// state and any write it might attempt go to temporary folders.
package realhome

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/courier"
	"research-desk/internal/home"
	"research-desk/internal/launch"
	"research-desk/internal/skillscripts"
	"research-desk/internal/testutil"
	"research-desk/internal/writer"
)

func realHome(t *testing.T) (home.Config, *home.Reader, *schedule.Fake) {
	t.Helper()
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	cfg := home.DefaultConfig(userHome)
	if info, err := os.Stat(filepath.Join(cfg.ResearchDir, "cohorts")); err != nil || !info.IsDir() {
		if os.Getenv("REAL_HOME_CHECK_SKIP") == "" {
			t.Fatalf("no research home at %s: this check needs the operator's real home (REAL_HOME_CHECK_SKIP=1 skips it on a machine without one)", cfg.ResearchDir)
		}
		t.Skipf("SKIPPED, NOT CHECKED: no research home at %s and REAL_HOME_CHECK_SKIP is set", cfg.ResearchDir)
	}
	paths := skillscripts.Paths{Skills: "../../../skills"}
	clock := schedule.NewFake(time.Now())
	reader := home.NewReader(cfg, home.Deps{
		Clock:    clock,
		Procs:    home.ProcTable{},
		Missions: skillscripts.MissionChecker{Paths: paths, Runner: skillscripts.NewPython()},
		Ceilings: paths.Ceilings,
	})
	return cfg, reader, clock
}

// desk lists the files the desk may write in the home (FILE-CONTRACT §1),
// with their size and time.
func desk(t *testing.T, research string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, pattern := range []string{"cohorts/*/*", "reviews/*/*"} {
		paths, _ := filepath.Glob(filepath.Join(research, pattern))
		for _, path := range paths {
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
				out[path] = fmt.Sprintf("%s/%d", info.ModTime(), info.Size())
			}
		}
	}
	return out
}

func known(t *testing.T, what string, e home.Evidence) {
	t.Helper()
	if e.Known && e.Source == "" {
		t.Errorf("%s: known without a source", what)
	}
	if !e.Known && e.Reason == "" {
		t.Errorf("%s: unknown without a reason", what)
	}
}

func TestRealHomeInvariants(t *testing.T) {
	cfg, reader, _ := realHome(t)
	before := desk(t, cfg.ResearchDir)
	snap, err := reader.Snapshot(context.Background(), home.Options{MissionCheck: true})
	if err != nil {
		t.Fatal(err)
	}

	cohorts, _ := os.ReadDir(filepath.Join(cfg.ResearchDir, "cohorts"))
	for _, entry := range cohorts {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") && snap.Commission(entry.Name()) == nil {
			t.Errorf("commission folder %s is not listed", entry.Name())
		}
	}
	missions, _ := filepath.Glob(filepath.Join(cfg.ResearchDir, "topics", "*", "MISSION.md"))
	for _, mission := range missions {
		if slug := filepath.Base(filepath.Dir(mission)); snap.Topic(slug) == nil {
			t.Errorf("topic %s has a MISSION.md but is not listed", slug)
		}
	}

	states := map[home.State]int{}
	enum := []home.State{home.StateEnded, home.StatePrepared, home.StateUnknown, home.StateStopped, home.StateLive, home.StatePaused, home.StateLimited}
	for _, c := range snap.Commissions {
		states[c.State]++
		if strings.HasPrefix(c.StateRule, "packet complete") {
			t.Logf("%s: %s (%s)", c.ID, c.State, c.StateRule)
		}
		if !slices.Contains(enum, c.State) || c.StateRule == "" {
			t.Errorf("%s: state %q, rule %q", c.ID, c.State, c.StateRule)
		}
		for name, e := range map[string]home.Evidence{
			"title": c.Title.Evidence, "claim": c.Claim.Evidence, "topics": c.TopicsEvidence,
			"started": c.Started.Evidence, "allowance end": c.AllowanceEnd.Evidence, "ended": c.Ended.Evidence,
			"last write": c.LastWrite.Evidence, "touched after end": c.TouchedAfterEnd.Evidence,
			"steering": c.Steering.Evidence, "decisions": c.Decisions.Evidence,
		} {
			known(t, c.ID+" "+name, e)
		}
		for i, e := range c.StateEvidence {
			known(t, fmt.Sprintf("%s state evidence %d", c.ID, i), e)
		}
		for _, run := range c.Runs {
			known(t, c.ID+" run "+run.Path, run.LastWrite.Evidence)
		}
		if c.TouchedAfterEnd.Known && !c.Ended.Known {
			t.Errorf("%s: touched after an end it does not have", c.ID)
		}
	}
	t.Logf("%d commissions by state: %v", len(snap.Commissions), states)
	for _, e := range snap.AllUnreadable() {
		t.Errorf("unreadable: %s", e.Reason)
	}
	for _, topic := range snap.Topics {
		known(t, topic.Slug+" title", topic.Title.Evidence)
		if w := topic.LastResearchWrite; !w.Known && w.Reason != "no file in any run folder" {
			t.Errorf("%s: last research write unknown: %s", topic.Slug, w.Reason)
		}
		known(t, topic.Slug+" mission check", topic.Check.Evidence)
	}
	t.Logf("%d topics", len(snap.Topics))

	// Overlapping windows on the real home.
	for _, o := range snap.Overlaps() {
		if slices.Contains(o.CommissionIDs, "programme-four-20261108") && slices.Contains(o.CommissionIDs, "programme-four-20261110") {
			for _, id := range o.CommissionIDs {
				c := snap.Commission(id)
				t.Logf("%s: %s (%s); started %s %s; ended %s %s; topics %+v", id, c.State, c.StateRule, c.Started.Value, c.Started.Reason, c.Ended.Value, c.Ended.Reason, c.Topics)
			}
			t.Errorf("programme four of 11-08 and of 11-10 reported overlapping on %s", o.Topic)
		}
	}
	// Research writes, mission edits and reviews are told apart.
	if medical := snap.Topic("medical-practice-it-modernization"); medical != nil {
		w := medical.LastResearchWrite
		t.Logf("medical-practice-it-modernization: research write %v %s (%s); mission %+v; review %+v", w.Known, w.Value.UTC().Format(time.RFC3339), w.Reason, medical.Mission, medical.LastReview)
	}
	// A delivery the desk made is never credited to the scheduler, even
	// where the scheduler copied the desk's receipt into its own line.
	for _, f := range snap.NeedsYou().Items {
		if _, reason, why := f.Delivery(); reason == home.ReasonSchedulerDelivered && f.Delivered != nil && strings.Contains(f.Item.Delivered, home.DeskRoute) {
			t.Errorf("%s: the desk delivered it, yet: %s", f.Key(), why)
		}
	}
	latest := snap.LatestResearchWrite()
	t.Logf("latest research write: %s %s by %s (%s) %q", latest.Topic, latest.Value.UTC().Format(time.RFC3339), latest.CommissionID, latest.State, latest.Note)

	after := desk(t, cfg.ResearchDir)
	for path, fact := range before {
		if strings.HasSuffix(path, home.OperatorFile) || strings.HasSuffix(path, home.LaunchFile) || strings.HasSuffix(path, "VERDICTS.md") || strings.HasSuffix(path, home.CharterFile) {
			if after[path] != fact {
				t.Errorf("%s changed during a read-only check", path)
			}
		}
	}
}

// Each packet waiting to launch (prepared, or unknown with nothing found that
// started it) reaches the Launch confirmation (preflight passes) on the
// operator's assertion where the desk classes it unknown, with the desk's
// recommended choice for any blocker; the launch defaults need no typing.
// Which packets wait changes as the operator launches them, so they are found,
// not named. Nothing is launched or written.
func TestTheRealPacketsReachTheLaunchConfirmation(t *testing.T) {
	cfg, reader, _ := realHome(t)
	before := desk(t, cfg.ResearchDir)
	svc := &launch.Service{
		Reader:  reader,
		Packets: skillscripts.Packets{Paths: skillscripts.Paths{Skills: "../../../skills"}, Runner: skillscripts.NewPython(), ResearchDir: cfg.ResearchDir},
	}
	ctx := context.Background()
	snap, err := reader.Snapshot(ctx, home.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var waiting []*home.Commission
	for _, c := range snap.Commissions {
		if c.State == home.StatePrepared || (c.State == home.StateUnknown && c.NeverStarted.Assertable) {
			waiting = append(waiting, c)
		}
	}
	t.Logf("%d packets wait to launch", len(waiting))
	for _, c := range waiting {
		id := c.ID
		t.Logf("%s: %s (%s); never started: %+v", id, c.State, c.StateRule, c.NeverStarted)
		first, err := svc.PreflightCommission(ctx, id, launch.Choices{})
		if err != nil {
			t.Fatal(err)
		}
		ch := launch.Choices{AssertNeverStarted: c.State == home.StateUnknown, Resolutions: map[string]launch.Resolution{}}
		for _, v := range first.Verdicts {
			t.Logf("%s: %s %s blocks=%v quiet=%v recommended=%q (%s): %s", id, v.Topic, v.CommissionID, v.Blocks, v.Quiet, v.Recommended, v.RecommendedReason, v.Message)
			if v.Blocks && v.Recommended != "" {
				ch.Resolutions[v.CommissionID] = v.Recommended
			}
		}
		p, err := svc.PreflightCommission(ctx, id, ch)
		if err != nil {
			t.Fatal(err)
		}
		if !p.Launchable() {
			t.Errorf("%s does not reach the Launch confirmation: %q", id, p.Blockers)
		}
		// What the operator reads names no command.
		for _, pf := range []launch.Preflight{first, p} {
			lines := slices.Clone(pf.Blockers)
			for _, v := range pf.Verdicts {
				lines = append(lines, v.Message, v.RecommendedReason)
			}
			for _, line := range lines {
				if strings.Contains(line, "`") || strings.Contains(line, "research-desk ") || strings.Contains(line, "vrooli ") {
					t.Errorf("%s: %q names a command", id, line)
				}
			}
		}
		var topics []string
		for _, ref := range c.Topics {
			topics = append(topics, ref.Topic)
		}
		sug, err := svc.Suggest(ctx, topics)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: %d suggested quotes (%s)", id, len(sug.Quotes), sug.QuotesReason)
		// Suggested quotes are the operator's words or nothing: only from a
		// packet the desk built or a commission that started.
		if len(sug.Quotes) == 0 && !strings.Contains(sug.QuotesReason, "type the stance quotes") {
			t.Errorf("%s: no quotes and no reason: %q", id, sug.QuotesReason)
		}
		for _, q := range sug.Quotes {
			src := snap.Commission(q.CommissionID)
			if src == nil || (src.Packet == nil && !src.Started.Known) {
				t.Errorf("%s: quote %q comes from %s, which the desk did not build and which never started", id, q.Text, q.CommissionID)
			}
		}
		for _, tw := range sug.Topics {
			titled := 0
			for _, w := range tw.Workstreams {
				if w.Title != "" {
					titled++
				}
			}
			t.Logf("%s: topic %s: %d workstreams, %d titled (%s)", id, tw.Topic, len(tw.Workstreams), titled, tw.Evidence.Reason)
		}
	}
	after := desk(t, cfg.ResearchDir)
	for path, fact := range before {
		if after[path] != fact {
			t.Errorf("%s changed during a read-only check", path)
		}
	}
}

// Turning every switch on now sends none of the items
// already there, at once or after the two-hour wait.
func TestTurningTheCourierOnSendsNoCurrentItem(t *testing.T) {
	_, reader, clock := realHome(t)
	scratch := t.TempDir()
	// Writes the courier might attempt land in an empty folder, never the home.
	w := writer.New(filepath.Join(scratch, "research"), filepath.Join(scratch, "state"), testutil.EntryChars)
	t.Cleanup(func() { w.Close() })
	hub := &recordingHub{}
	c := &courier.Courier{Reader: reader, Writer: w, Hub: hub, Clock: clock}
	ctx := context.Background()
	snap, err := reader.Snapshot(ctx, home.Options{})
	if err != nil {
		t.Fatal(err)
	}
	feed := snap.NeedsYou().Items
	if _, err := c.Set(ctx, "", true); err != nil {
		t.Fatal(err)
	}
	for _, cm := range snap.Commissions {
		if cm.State != home.StateEnded {
			if _, err := c.Set(ctx, cm.ID, true); err != nil {
				t.Fatal(err)
			}
		}
	}
	for range 2 {
		if _, err := c.Pass(ctx); err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Logf("pass: %v", err)
		}
		clock.Advance(3 * time.Hour)
	}
	global := 0
	for i := range feed {
		if feed[i].CommissionID == "" {
			global++
		}
	}
	t.Logf("%d needs-you items (%d global ATTENTION.md items) present when the switches were turned on", len(feed), global)
	if len(hub.asks)+len(hub.notices) != 0 {
		t.Fatalf("items already there were sent: %d asks %v, %d notices %v", len(hub.asks), hub.asks, len(hub.notices), hub.notices)
	}
}

type recordingHub struct {
	asks, notices []string
}

func (h *recordingHub) Sendable(context.Context) error { return nil }
func (h *recordingHub) Ask(_ context.Context, spec courier.AskSpec) error {
	h.asks = append(h.asks, spec.Key)
	return nil
}

func (h *recordingHub) Notify(_ context.Context, spec courier.NoticeSpec) error {
	h.notices = append(h.notices, spec.Key)
	return nil
}
func (h *recordingHub) Asks(context.Context, string) ([]courier.Ask, error) { return nil, nil }
func (h *recordingHub) Notices(context.Context) (map[string]courier.Notice, error) {
	return map[string]courier.Notice{}, nil
}
