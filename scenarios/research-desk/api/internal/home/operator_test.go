package home

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"research-desk/internal/testutil"
)

// The needs-you and operator-file grammars are tested on the real
// ATTENTION.md and STATE.md copies, and on the contract's own examples
// (FILE-CONTRACT.md §4, §5) for fields no real file uses yet.

const contractItem = `## Needs you

### AV4 — yt-dlp target choice
One to three sentences: what is needed and why.
Recommended: keep — Keep yt-dlp uninstalled and continue AV5–AV7.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.
Packet: /abs/path/to/details.md

### AV9 — A secret
Secret: vrooli/sam-gov field api-key

## Closeout
`

const contractEntries = `# Operator — programme-five-20261112

### 2026-11-12T10:14:03Z · direction · via desk
Operator words, verbatim, any number of lines.

### 2026-11-12T11:02:47Z · answer AV4 · via hub
keep — Keep yt-dlp uninstalled and continue AV5–AV7.
Note: fine, but tell me if AV6 needs it

### 2026-11-12T11:03:00Z · delivered AV9 · via desk
hub notice n-1

### 2026-11-12T12:00:00Z · wind-down · via desk
Finish the current unit, write the closeout, then end.
### not an entry
`

// A hand-written heading may carry any RFC 3339 time, as the real
// reviews/local-models-20261111/VERDICTS.md does; a heading that still does
// not parse is reported with its line, never dropped.
func TestAnyRFC3339HeadingTimeIsAnEntry(t *testing.T) {
	text := "# Verdicts\n\n### 2026-11-12T00:42:22.604567+00:00 · verdict local-model-performance · via hand\nKeep.\n\n" +
		"### 2026-11-12T01:06:43-05:00 · verdict local-model-performance · via hand\nDrop.\n\n### 2026-13-40T99:00:00Z · verdict x · via hand\nlost?\n"
	entries, unparsed := ParseEntries(text)
	if len(entries) != 2 {
		t.Fatalf("both verdicts must be read, got %+v", entries)
	}
	if want := time.Date(2026, 11, 12, 0, 42, 22, 604567000, time.UTC); !entries[0].At.Equal(want) || entries[0].Body != "Keep." {
		t.Errorf("fractional UTC heading: %+v", entries[0])
	}
	if want := time.Date(2026, 11, 12, 6, 6, 43, 0, time.UTC); !entries[1].At.Equal(want) {
		t.Errorf("an offset is converted to UTC: %v", entries[1].At)
	}
	if len(unparsed) != 1 || unparsed[0].Line != 9 {
		t.Errorf("an impossible time is reported with its line: %+v", unparsed)
	}
}

func TestRealAttentionItemsAreGlobalNotices(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testutil.NewHome(t).Research, "ATTENTION.md"))
	if err != nil {
		t.Fatal(err)
	}
	items, legacy := ParseNeedsYou(string(data), AttentionNeedsSection, "ATTENTION.md")
	if len(items) != 7 || len(legacy) != 0 {
		t.Fatalf("want the 7 real items and no legacy text, got %d items, %d legacy", len(items), len(legacy))
	}
	if items[1].ID != "PROGRAMME-THREE-AV2-YTDLP-TARGET-20261111" || items[1].Title != "Choose how AV4 handles yt-dlp" {
		t.Errorf("second item: got %q %q", items[1].ID, items[1].Title)
	}
	for _, it := range items {
		if it.Recommended != nil || it.Commission != "" {
			t.Errorf("%s: a recommendation inside a bullet is not a `Recommended:` line, and no item names a commission; got %+v", it.ID, it)
		}
		f := FeedItem{Item: it, CommissionID: it.Commission}
		if d, _, _ := f.Delivery(); d != DeliverNotice {
			t.Errorf("%s: a global item is sent as a notice at most, got %s", it.ID, d)
		}
		if ok, _ := f.Answerable(); ok {
			t.Errorf("%s: no commission would read an answer to a global item", it.ID)
		}
		if it.Instruction() == "" {
			t.Errorf("%s: a global item carries what it says to do", it.ID)
		}
	}
	// The instruction is the item's own words; links resolve beside the file.
	if got := items[0].Instruction(); !strings.HasPrefix(got, "The approved HOST-DISK build has a candidate CLI") {
		t.Errorf("the Needs you line is the instruction: %q", got)
	}
	if got := items[6].Instruction(); !strings.HasPrefix(got, "none; three taps only you can make") {
		t.Errorf("a Decision line is the instruction: %q", got)
	}
	links := items[5].Links()
	if len(links) != 1 || links[0].Label != "public-data-fraud mission, \"PF1 prepared item\"" || links[0].Target != "topics/public-data-fraud/MISSION.md" {
		t.Errorf("links are listed with their labels, relative to ATTENTION.md's folder: %+v", links)
	}
	withSource := items[5]
	withSource.Source = "/research/ATTENTION.md"
	if l := withSource.Links(); l[0].Target != "/research/topics/public-data-fraud/MISSION.md" {
		t.Errorf("a relative link resolves beside the item's file: %+v", l)
	}

	// A `Commission:` line makes a global item answerable into that commission.
	named, _ := ParseNeedsYou("## Actionable now\n\n### X1 — Pick one\nCommission: local-models-w1-20261111\nOptions: a — A; b — B\n", AttentionNeedsSection, "ATTENTION.md")
	if len(named) != 1 || named[0].Commission != "local-models-w1-20261111" {
		t.Fatalf("Commission: line: %+v", named)
	}
	if ok, why := (FeedItem{Item: named[0], CommissionID: named[0].Commission, Commission: &Commission{ID: named[0].Commission}}).Answerable(); !ok {
		t.Errorf("a global item naming a commission is answerable into it: %s", why)
	}
}

func TestRealStateNumberedListIsLegacy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testutil.NewHome(t).Research, "cohorts/programme-three-20261111/STATE.md"))
	if err != nil {
		t.Fatal(err)
	}
	items, legacy := ParseNeedsYou(string(data), StateNeedsSection, "STATE.md")
	if len(items) != 0 || len(legacy) != 1 || !strings.HasPrefix(legacy[0].Text, "1. **HOST-DISK") {
		t.Fatalf("the numbered list is legacy text, never items: %d items, legacy %+v", len(items), legacy)
	}
}

func TestContractItemFields(t *testing.T) {
	items, legacy := ParseNeedsYou(contractItem, StateNeedsSection, "STATE.md")
	if len(items) != 2 || len(legacy) != 0 {
		t.Fatalf("want 2 items, got %d (legacy %v)", len(items), legacy)
	}
	av4 := items[0]
	if av4.Recommended == nil || av4.Recommended.Key != "keep" || len(av4.Options) != 2 || av4.Options[1].Key != "propose" || av4.Packet != "/abs/path/to/details.md" {
		t.Fatalf("AV4 fields: %+v", av4)
	}
	if _, ok := av4.Choice("hold"); ok {
		t.Error("an item with Options: offers only those options")
	}
	if items[1].Secret == "" || len(items[1].Choices()) != 2 || items[1].Choices()[0].Key != "accept" {
		t.Errorf("AV9: want the secret and the default options, got %+v", items[1])
	}
	if strings.Contains(items[1].Body, "Closeout") {
		t.Error("an item ends at the next `## ` heading")
	}
}

func TestContractEntries(t *testing.T) {
	entries, unparsed := ParseEntries(contractEntries)
	if len(entries) != 4 || len(unparsed) != 1 || unparsed[0] != (UnparsedHeading{Line: 15, Text: "### not an entry"}) {
		t.Fatalf("want 4 entries and the unparsed heading on line 15, got %d and %+v", len(entries), unparsed)
	}
	if a := entries[1]; a.Subject != "AV4" || a.Via != ViaHub || !strings.HasPrefix(a.Body, "keep — ") || !strings.HasSuffix(a.Body, "\nNote: fine, but tell me if AV6 needs it") {
		t.Errorf("answer entry: %+v", a)
	}
	if entries[2].Kind != EntryDelivered || entries[2].Subject != "AV9" {
		t.Errorf("delivered entry: %+v", entries[2])
	}
	if !strings.HasPrefix(entries[3].Body, "Finish the current unit") || !strings.Contains(entries[3].Body, "not an entry") {
		t.Errorf("an unparsed `### ` line stays in the body of the entry before it: %q", entries[3].Body)
	}
	for _, e := range entries {
		if !IsEntryHeading(e.Heading()) {
			t.Errorf("rendered heading %q does not parse", e.Heading())
		}
	}
	log := &OperatorLog{Entries: entries}
	if log.Newest(EntryAnswer, "AV4") != &log.Entries[1] || log.Newest(EntryAnswer, "AV9") != nil {
		t.Error("Newest finds the newest entry of a kind and subject")
	}
	if got := AnswerBody("other", "", " my words "); got != "other — my words" {
		t.Errorf("other answer body: %q", got)
	}
}

// writeRel writes one file of the fixture copy.
func (f *fixture) writeRel(t *testing.T, rel, text string, at time.Time) {
	t.Helper()
	path := filepath.Join(f.research, rel)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	f.touch(t, rel, at)
}

func TestFeedIdentityIsCommissionAndID(t *testing.T) {
	f := newFixture(t)
	// local-models is prepared (not ended); give its STATE.md a contract item
	// and its OPERATOR.md an answer, and point an ATTENTION.md copy at it.
	state, _ := os.ReadFile(filepath.Join(f.research, "cohorts/local-models-w1-20261111/STATE.md"))
	f.writeRel(t, "cohorts/local-models-w1-20261111/STATE.md", string(state)+"\n"+contractItem, time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC))
	f.writeRel(t, "cohorts/local-models-w1-20261111/OPERATOR.md", contractEntries, base)
	attention, _ := os.ReadFile(filepath.Join(f.research, AttentionFile))
	extra := strings.Replace(string(attention), "## Actionable now\n", "## Actionable now\n\n### AV4 — yt-dlp target choice\nCommission: local-models-w1-20261111\n\n### AV4 — a global item with the same ID\n", 1)
	f.writeRel(t, AttentionFile, extra, base)

	feed := f.snapshot(t, Options{}).NeedsYou()
	legacy := feed.Legacy
	var av4 []FeedItem
	for _, item := range feed.Items {
		if item.Item.ID == "AV4" {
			av4 = append(av4, item)
		}
	}
	if len(av4) != 2 {
		t.Fatalf("want the commission's AV4 (merged with its ATTENTION copy) and a separate global AV4, got %d", len(av4))
	}
	c := av4[0]
	if c.CommissionID != "local-models-w1-20261111" || len(c.Sources) != 2 || c.Answer == nil || c.Answer.Via != ViaHub {
		t.Errorf("commission item: %+v", c)
	}
	if !c.Read.Value || c.Read.Source == "" {
		t.Errorf("STATE.md was rewritten after the answer, so a unit has run since: %+v", c.Read)
	}
	if d, _, why := c.Delivery(); d != DeliverNone || !strings.Contains(why, "answered") {
		t.Errorf("an answered item is not sent: %s %s", d, why)
	}
	if av4[1].CommissionID != "" || av4[1].Commission != nil {
		t.Errorf("an ATTENTION.md item without Commission: is global: %+v", av4[1])
	}
	for _, item := range feed.Items {
		if item.Item.ID == "AV9" && item.Delivered == nil {
			t.Error("AV9's delivered entry is found")
		}
	}
	if len(legacy) == 0 {
		t.Error("local-models' numbered needs-you list is shown as legacy")
	}
}

// [REQ:RD-P0-013]
func TestChangesSinceLastLook(t *testing.T) {
	f := newFixture(t)
	id := "local-models-w1-20261111"
	first := f.snapshot(t, Options{})
	changes, err := first.Changes(nil, time.Time{}, LookCommission, id)
	look, lookErr := first.Look(LookCommission, id)
	if err != nil || lookErr != nil || !changes.Since.Equal(f.clock.Now().Add(-DefaultWindow)) || changes.BaselineKind != BaselineLastDay || !strings.HasPrefix(changes.Baseline, "no look recorded; the last 24 hours") || len(look.Ledgers) != 1 {
		t.Fatalf("first look: %+v %+v %v %v", changes, look, err, lookErr)
	}

	ledger := filepath.Join(f.research, "cohorts", id, LedgerFile)
	before, _ := os.ReadFile(ledger)
	added := "\n## Unit 4\nThe showcase rendered at 1080p.\n"
	f.writeRel(t, "cohorts/"+id+"/"+LedgerFile, string(before)+added, base.Add(3*time.Hour))
	f.clock.Advance(2 * time.Hour)

	changes, err = f.snapshot(t, Options{}).Changes(&look, time.Time{}, LookCommission, id)
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Since.Equal(look.At) || len(changes.Ledger) != 1 || changes.Ledger[0].Text != added {
		t.Fatalf("want exactly the appended ledger text since the look, got %+v", changes)
	}
	if len(changes.ShowcaseLines) != 1 || len(changes.Files) != 1 || changes.Files[0].Path != ledger {
		t.Errorf("showcase lines and written files: %+v %+v", changes.ShowcaseLines, changes.Files)
	}
	if _, err := first.Changes(nil, time.Time{}, LookTopic, "no-such-topic"); err == nil {
		t.Error("an unknown topic is not found")
	}

	// A time instead of a look lists the files written after it, shows the
	// ledger sections under headings dated after it, and says plainly that
	// workstream status changes need a recorded look.
	since := base.Add(150 * time.Minute)
	changes, err = f.snapshot(t, Options{}).Changes(nil, since, LookCommission, id)
	if err != nil || !changes.Since.Equal(since) || changes.BaselineKind != BaselineAsked || len(changes.Files) != 1 || !strings.Contains(changes.Baseline, "as asked") {
		t.Fatalf("since a time: %+v %v", changes, err)
	}
	if len(changes.NotShown) != 1 || !strings.Contains(changes.NotShown[0], "Workstream status changes need a recorded look") {
		t.Errorf("what cannot be shown is said: %q", changes.NotShown)
	}
	if len(changes.Ledger) != 1 || !changes.Ledger[0].FromDatedHeadings || !strings.HasPrefix(changes.Ledger[0].Text, "## 2026-11-11 — prepared") || strings.Contains(changes.Ledger[0].Text, "Unit 4") {
		t.Fatalf("the section under the heading dated after the time, up to the next heading at its level: %+v", changes.Ledger)
	}

	// A heading's own time and offset decide; sections dated before are left out.
	dated := "# Ledger\n\n## Entries\n\n### 2026-11-10T15:29:59.5+01:00 — before\nold\n\n### 2026-11-10T14:40Z — after\nnew\n#### detail\nkept\n"
	f.writeRel(t, "cohorts/"+id+"/"+LedgerFile, dated, base.Add(3*time.Hour))
	changes, _ = f.snapshot(t, Options{}).Changes(nil, since, LookCommission, id)
	if len(changes.Ledger) != 1 || changes.Ledger[0].Text != "### 2026-11-10T14:40Z — after\nnew\n#### detail\nkept\n" {
		t.Fatalf("only the later entry: %+v", changes.Ledger)
	}
	f.writeRel(t, "cohorts/"+id+"/"+LedgerFile, "# Ledger\n\n## Unit 4\nnew text\n", base.Add(3*time.Hour))
	changes, _ = f.snapshot(t, Options{}).Changes(nil, since, LookCommission, id)
	if len(changes.Ledger) != 0 || len(changes.NotShown) != 2 || !strings.Contains(changes.NotShown[1], ledger+" has no dated headings") {
		t.Errorf("a ledger with no dated headings is named, not guessed at: %+v %q", changes.Ledger, changes.NotShown)
	}
}

// An older item's `- **Delivered…**` bullet counts as delivered.
func TestAnOlderDeliveredBulletCountsAsDelivered(t *testing.T) {
	text := "## Actionable now\n\n### OLD-ITEM-20261101 — Older item\n\n- **Needs you:** choose.\n- **Delivered 2026-11-02:** Hub ask abc, pending.\n\n### NEW-ITEM — Newer item\n\n- **Delivery:** no Hub ask exists.\n"
	items, _ := ParseNeedsYou(text, AttentionNeedsSection, "ATTENTION.md")
	if len(items) != 2 || items[0].Delivered == "" || items[1].Delivered != "" {
		t.Fatalf("items = %+v", items)
	}
}

// A scheduler copies the desk's receipt into its `Delivered:` line; that is
// the desk's delivery, not the scheduler's.
func TestADeliveredLineCopiedFromTheDeskIsTheDesks(t *testing.T) {
	at := time.Date(2026, 11, 12, 6, 5, 16, 0, time.UTC)
	receipt := &Entry{At: at, Kind: EntryDelivered, Subject: "SCHEDULER-MODEL", Via: "desk", Body: "hub notice 7a723ac5-0bbe"}
	cases := []struct {
		name, line string
		receipt    *Entry
		reason     DeliveryReason
		want       string
	}{
		{"names the receipt's notice", "hub notice 7a723ac5-0bbe 2026-11-12T06:05:16Z", receipt, ReasonHubDelivered, "delivered by the desk at 2026-11-12T06:05:16Z (hub notice 7a723ac5-0bbe)"},
		{"desk route with a receipt", "research-desk hub notice 7a723ac5-0bbe 2026-11-12T06:05:16Z", receipt, ReasonHubDelivered, "delivered by the desk at 2026-11-12T06:05:16Z"},
		{"desk route, no receipt", "research-desk hub notice 99 2026-11-12T06:05:16Z", nil, ReasonHubDelivered, "its OPERATOR.md holds no receipt for it"},
		{"another route, a receipt for another notice", "hub 2026-11-12T14:00:00Z", receipt, ReasonSchedulerDelivered, "the scheduler delivered it itself (Delivered: hub 2026-11-12T14:00:00Z)"},
		{"another route, no receipt", "hub 2026-11-12T14:00:00Z", nil, ReasonSchedulerDelivered, "the scheduler delivered it itself"},
	}
	for _, tc := range cases {
		f := FeedItem{Item: NeedsYouItem{ID: "SCHEDULER-MODEL", Delivered: tc.line}, Delivered: tc.receipt}
		delivery, reason, why := f.Delivery()
		if delivery != DeliverNone || reason != tc.reason || !strings.Contains(why, tc.want) {
			t.Errorf("%s: %s %s %q", tc.name, delivery, reason, why)
		}
	}
}

// unreadable makes a fixture file exist but refuse reads for this test.
func (f *fixture) unreadable(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join(f.research, rel)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) })
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("this user reads a file with no permissions (root); cannot test an unreadable file")
	}
	return path
}

// [REQ:RD-P0-004]
// A STATE.md that exists but cannot be read is unknown, not "no items",
// and its owner's needs-you items are blocked.
func TestAnUnreadableStateIsUnknownNotNoItems(t *testing.T) {
	f := newFixture(t)
	const id = "local-models-w1-20261111"
	f.writeRel(t, "cohorts/"+id+"/STATE.md", "# State\n\n"+contractItem, base)
	path := f.unreadable(t, "cohorts/"+id+"/STATE.md")
	snap := f.snapshot(t, Options{})
	c := snap.Commission(id)
	if len(c.Unreadable) != 1 || c.Unreadable[0].Source != path || !strings.Contains(c.Unreadable[0].Reason, "cannot read") {
		t.Fatalf("STATE.md must be listed as unreadable, got %+v", c.Unreadable)
	}
	if c.Claim.Known || !strings.Contains(c.Claim.Reason, "cannot read") {
		t.Errorf("the claim is unknown because STATE.md cannot be read, not absent: %+v", c.Claim)
	}
	feed := snap.NeedsYou()
	if len(feed.Unreadable) != 1 || !strings.Contains(feed.Blocked(id), path) || feed.Blocked("programme-three-20261111") != "" {
		t.Errorf("only this commission is blocked, with the reason: %+v / %q", feed.Unreadable, feed.Blocked(id))
	}
	if len(snap.AllUnreadable()) != 1 {
		t.Errorf("the overview lists it: %+v", snap.AllUnreadable())
	}
}

// An OPERATOR.md that exists but cannot be read is unknown, not "no
// entries"; an unreadable ATTENTION.md blocks every owner.
func TestAnUnreadableOperatorFileIsUnknownNotNoEntries(t *testing.T) {
	f := newFixture(t)
	const id = "local-models-w1-20261111"
	f.writeRel(t, "cohorts/"+id+"/OPERATOR.md", contractEntries, base)
	path := f.unreadable(t, "cohorts/"+id+"/OPERATOR.md")
	snap := f.snapshot(t, Options{})
	c := snap.Commission(id)
	if c.Operator != nil || len(c.Unreadable) != 1 || c.Unreadable[0].Source != path {
		t.Fatalf("OPERATOR.md is unreadable, not empty: operator %+v, unreadable %+v", c.Operator, c.Unreadable)
	}
	if feed := snap.NeedsYou(); !strings.Contains(feed.Blocked(id), path) {
		t.Errorf("answers unknown block the commission: %q", feed.Blocked(id))
	}

	f.unreadable(t, AttentionFile)
	feed := f.snapshot(t, Options{}).NeedsYou()
	if feed.Blocked("") == "" || !strings.Contains(feed.Blocked("programme-three-20261111"), AttentionFile) {
		t.Errorf("an unreadable ATTENTION.md blocks global items and every commission: %q", feed.Blocked("programme-three-20261111"))
	}
}

// Each commission says when a new operator entry will be read, as a code
// a client matches on, and since when entries have been read.
func TestEachCommissionSaysWhenANewEntryIsRead(t *testing.T) {
	steering := Flag{Value: true, Evidence: Evidence{Known: true}}
	for _, tc := range []struct {
		state    State
		steering Flag
		want     ReadWhen
	}{
		{StateLive, steering, ReadNextUnit},
		{StatePaused, steering, ReadAfterResume},
		{StateLimited, steering, ReadAfterResume},
		{StateEnded, steering, ReadNever},
		{StatePrepared, steering, ReadAtStart},
		{StateStopped, steering, ReadAtStart},
		{StateUnknown, steering, ReadUnknown},
		{StateLive, Flag{Evidence: Evidence{Known: true}}, ReadNotUntilSteering},
		{StatePaused, Flag{Evidence: Unknown("charter unreadable")}, ReadUnknown},
		{StateEnded, Flag{Evidence: Evidence{Known: true}}, ReadNever},
	} {
		c := &Commission{State: tc.state, Steering: tc.steering}
		if got := c.EntryReading(); got.When != tc.want || got.Message == "" {
			t.Errorf("%s (steering %v, known %v): %+v, want %s", tc.state, tc.steering.Value, tc.steering.Known, got, tc.want)
		}
	}
	at := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	c := &Commission{State: StateLive, Steering: steering, Files: []FileFact{{Name: StateFile, Path: "/r/STATE.md", Modified: at}}}
	if got := c.EntryReading().ReadSince; !got.Known || !got.Value.Equal(at) || got.Source != "/r/STATE.md" {
		t.Errorf("read since is STATE.md's last rewrite: %+v", got)
	}
	if got := (&Commission{State: StateLive, Steering: steering}).EntryReading().ReadSince; got.Known {
		t.Errorf("without STATE.md no entry has been read: %+v", got)
	}
}
