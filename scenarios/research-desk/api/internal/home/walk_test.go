package home

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// [REQ:RD-P0-004]
// The run folder names and the 1108/1110 sequence of the real home's
// freelance-workflow-readiness topic. Programme four of 11-08 ended at its
// allowance; its delivery-kits run folder (2.8 million files on the real
// home) was written again during programme four of 11-10.
func TestRunFoldersBelongToTheirCommissionAndALaterEditIsNotActivity(t *testing.T) {
	root := t.TempDir()
	f := &fixture{research: filepath.Join(root, "research"), codex: filepath.Join(root, "codex"), procs: &fakeProcs{known: true}}
	start := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	f.clock = schedule.NewFake(start.Add(60 * time.Hour))
	const (
		topic = "freelance-workflow-readiness"
		old   = "programme-four-20261108"
		next  = "programme-four-20261110"
	)
	at := func(rel string, h int) {
		path := filepath.Join(f.research, rel)
		writeFile(t, path, rel)
		if err := os.Chtimes(path, start.Add(time.Duration(h)*time.Hour), start.Add(time.Duration(h)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{old, next} {
		for _, name := range []string{CharterFile, StanceFile, KernelFile} {
			at("cohorts/"+id+"/"+name, 0)
		}
		writeFile(t, filepath.Join(f.research, "cohorts", id, PacketFile), `{"schema":"research-desk.packet/v1","commission_id":"`+id+`","spec":{"topics":["`+topic+`"],"allowance_hours":24}}`)
		_ = os.Chtimes(filepath.Join(f.research, "cohorts", id, PacketFile), start, start)
	}
	at("topics/"+topic+"/MISSION.md", 0)
	at("topics/"+topic+"/runs/"+old+"-delivery-kits/kit/early.md", 2)
	at("topics/"+topic+"/runs/"+old+"-delivery-kits/kit/deep/later.md", 50)
	at("topics/"+topic+"/runs/"+next+"/CHECKPOINT.md", 49)

	f.codexHome(t, "s-old", []goalRow{{"t-old", "Work until every card in " + f.charter(old) + " lands.", "active", start}}, nil)
	f.procs.window(f.codexHome(t, "s-next", []goalRow{{"t-next", "Work until every card in " + f.charter(next) + " lands.", "active", start.Add(48 * time.Hour)}}, nil))
	snap := f.snapshot(t, Options{})

	o := snap.Commission(old)
	if len(o.Runs) != 1 || filepath.Base(o.Runs[0].Path) != old+"-delivery-kits" {
		t.Fatalf("the <id>-<suffix> run folder belongs to %s: %+v", old, o.Runs)
	}
	if o.State != StateEnded || !o.Ended.Known || !o.Ended.Value.Equal(start.Add(24*time.Hour)) || o.Ended.Reason != "allowance ended" {
		t.Fatalf("%s ended at its allowance: %s %+v", old, o.State, o.Ended)
	}
	if !o.LastWrite.Value.Equal(start.Add(2*time.Hour)) || !o.TouchedAfterEnd.Known || !o.TouchedAfterEnd.Value.Equal(start.Add(50*time.Hour)) {
		t.Fatalf("activity stops at the end and the later edit is touched after end: last %+v, after %+v", o.LastWrite, o.TouchedAfterEnd)
	}
	if got := o.Runs[0].LastWrite; !strings.HasSuffix(got.Source, "deep/later.md") {
		t.Fatalf("the run folder's last write is its newest file anywhere inside: %+v", got)
	}
	if overlaps := snap.Overlaps(); len(overlaps) != 0 {
		t.Fatalf("a write after the end never creates an overlap: %+v", overlaps)
	}
	if n := snap.Commission(next); n.State != StateLive || n.Ended.Known || len(n.Runs) != 1 {
		t.Fatalf("%s is live and has not ended: %s %+v", next, n.State, n.Ended)
	}
	got := snap.Topic(topic).LastResearchWrite
	if !got.Value.Equal(start.Add(50*time.Hour)) || got.CommissionID != old || got.State != StateEnded {
		t.Fatalf("the topic's last research write is the newest file in any of its run folders, with its owner: %+v", got)
	}
	if got.Note != "written after the commission ended: an owner may still be running" {
		t.Fatalf("a write after its commission ended says so: %q", got.Note)
	}
}

func TestTheLongestCommissionIDOwnsARunFolder(t *testing.T) {
	ids := []string{"programme-four-20261110", "programme-four-20261110-w2", "programme-four"}
	for name, want := range map[string]string{
		"programme-four-20261110":            "programme-four-20261110",
		"programme-four-20261110-kits":       "programme-four-20261110",
		"programme-four-20261110-w2-kits":    "programme-four-20261110-w2",
		"programme-four-20261110x":           "programme-four",
		"programme-five-20261110":            "",
		"programme-four-20261110-w2":         "programme-four-20261110-w2",
		"local-models-w1-20261111-follow-up": "",
	} {
		if got := runOwner(name, ids); got != want {
			t.Errorf("%s: owner %q, want %q", name, got, want)
		}
	}
}

func TestACutWalkSaysAtLeast(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 11, 11, 19, 8, 0, 0, time.UTC)
	for i := range 5 {
		path := filepath.Join(dir, "d"+strconv.Itoa(i), "f.md")
		writeFile(t, path, "x")
		_ = os.Chtimes(path, at.Add(time.Duration(i)*time.Minute), at.Add(time.Duration(i)*time.Minute))
	}
	whole := walkNewest(dir, "", time.Time{}, 100, time.Now().Add(time.Minute))
	if !whole.All.Value.Equal(at.Add(4*time.Minute)) || whole.All.Reason != "" {
		t.Fatalf("a whole walk: %+v", whole.All)
	}
	cut := walkNewest(dir, "", time.Time{}, 4, time.Now().Add(time.Minute))
	if !cut.All.Known || !cut.All.AtLeast || !strings.HasPrefix(cut.All.Reason, "at least") || !strings.HasSuffix(cut.All.Source, "(walk cut short)") {
		t.Fatalf("a cut walk reports at least what it saw and says so: %+v", cut.All)
	}
}

// [REQ:RD-P0-002]
// Overlaps are pairs: a commission that overlaps two others does not make
// those two overlap.
func TestOverlapsArePairs(t *testing.T) {
	at := func(h int) time.Time { return base.Add(time.Duration(h) * time.Hour) }
	c := func(id string, start, end int) *Commission {
		return &Commission{ID: id, Topics: []TopicRef{{Topic: "t"}}, Started: Time{Value: at(start), Evidence: Evidence{Known: true}}, Ended: Time{Value: at(end), Evidence: Evidence{Known: true}}}
	}
	snap := &Snapshot{Now: at(100), Commissions: []*Commission{c("a", 0, 10), c("b", 20, 30), c("c", 5, 25)}}
	var got []string
	for _, o := range snap.Overlaps() {
		got = append(got, strings.Join(o.CommissionIDs, "+"))
	}
	if strings.Join(got, " ") != "a+c b+c" {
		t.Fatalf("overlaps = %v", got)
	}
}
