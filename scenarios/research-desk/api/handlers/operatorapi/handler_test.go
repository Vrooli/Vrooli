package operatorapi

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/courier"
	"research-desk/internal/home"
	"research-desk/internal/testutil"
	"research-desk/internal/writer"

	operatorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

// Handlers run over the testutil archive of real research-home excerpts. local-models is a real
// prepared packet; its STATE.md gets the contract's items (FILE-CONTRACT §5).

const items = `
## Needs you

### AV4 — yt-dlp target choice
Recommended: keep — Keep yt-dlp uninstalled and continue AV5–AV7.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.

### AV9 — SAM.gov key
Secret: vrooli/sam-gov field api-key
`

const commission = "local-models-w1-20261111"

func newHandler(t *testing.T) (*handler, string) {
	t.Helper()
	h := testutil.NewHome(t)
	research := h.Research
	state := filepath.Join(research, "cohorts", commission, home.StateFile)
	data, _ := os.ReadFile(state)
	if err := os.WriteFile(state, append(data, items...), 0o644); err != nil {
		t.Fatal(err)
	}
	clock := schedule.NewFake(time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))
	reader := home.NewReader(home.Config{ResearchDir: research, CodexDir: h.Codex}, home.Deps{Clock: clock, Procs: home.FixedProcs{}})
	w := writer.New(research, h.State, testutil.EntryChars)
	t.Cleanup(func() { w.Close() })
	c := &courier.Courier{Reader: reader, Writer: w, Clock: clock, Logger: log.New(io.Discard, "", 0)}
	return NewHandler(Deps{ResearchDir: research, Reader: reader, Writer: w, Courier: c, Clock: clock}), research
}

func code(err error) connect.Code { return connect.CodeOf(err) }

// [REQ:RD-P0-006]
func TestAnswer(t *testing.T) {
	h, research := newHandler(t)
	ctx := context.Background()
	answer := func(item, key, note string) (*connect.Response[operatorv1.AnswerResponse], error) {
		return h.Answer(ctx, connect.NewRequest(&operatorv1.AnswerRequest{CommissionId: commission, ItemId: item, OptionKey: key, Note: note}))
	}
	if _, err := answer("AV4", "hold", ""); code(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "`keep`, `propose`") {
		t.Errorf("an option the item does not offer is refused with the choices: %v", err)
	}
	if _, err := answer("AV4", "other", ""); code(err) != connect.CodeInvalidArgument {
		t.Errorf("other needs words: %v", err)
	}
	if _, err := answer("AV9", "accept", ""); code(err) != connect.CodeFailedPrecondition {
		t.Errorf("a secret is never answered into a file: %v", err)
	}
	if _, err := answer("AV7", "keep", ""); code(err) != connect.CodeNotFound {
		t.Errorf("an unknown item is not found: %v", err)
	}
	resp, err := answer("AV4", "keep", "tell me if AV6 needs it")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Warning == "" {
		t.Error("the real local-models charter predates steering, so the answer warns it will not be read")
	}
	data, _ := os.ReadFile(filepath.Join(research, "cohorts", commission, home.OperatorFile))
	if !strings.Contains(string(data), "· answer AV4 · via desk\nkeep — Keep it uninstalled\nNote: tell me if AV6 needs it\n") {
		t.Errorf("OPERATOR.md:\n%s", data)
	}

	feed, err := h.ListNeedsYou(ctx, connect.NewRequest(&operatorv1.ListNeedsYouRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range feed.Msg.Items {
		if it.Id == "AV4" && (it.Answer == nil || it.Delivery != operatorv1.Delivery_DELIVERY_NONE) {
			t.Errorf("AV4 shows its answer and is no longer sent: %+v", it)
		}
	}
	if max, _ := testutil.EntryChars(); max == 0 || feed.Msg.EntryCharsMax != int32(max) {
		t.Errorf("the entry limit comes from limits.json: %d, want %d", feed.Msg.EntryCharsMax, max)
	}
	if feed.Msg.Courier.HubRead || !strings.Contains(feed.Msg.Courier.HubReason, "no courier switch is on") {
		t.Errorf("no switch is on, so the Hub is not read: %+v", feed.Msg.Courier)
	}
}

// [REQ:RD-P0-005]
func TestAddEntryAndSteering(t *testing.T) {
	h, research := newHandler(t)
	ctx := context.Background()
	if _, err := h.AddEntry(ctx, connect.NewRequest(&operatorv1.AddEntryRequest{CommissionId: commission, Kind: operatorv1.EntryKind_ENTRY_KIND_DIRECTION})); code(err) != connect.CodeInvalidArgument {
		t.Errorf("direction needs words: %v", err)
	}
	resp, err := h.AddEntry(ctx, connect.NewRequest(&operatorv1.AddEntryRequest{CommissionId: commission, Kind: operatorv1.EntryKind_ENTRY_KIND_WIND_DOWN}))
	if err != nil || resp.Msg.Entry.Body != WindDownText {
		t.Fatalf("a wind-down without words uses the standard sentence: %+v %v", resp, err)
	}
	if _, err := h.EnableSteering(ctx, connect.NewRequest(&operatorv1.EnableSteeringRequest{CommissionId: commission})); err != nil {
		t.Fatal(err)
	}
	resp, err = h.AddEntry(ctx, connect.NewRequest(&operatorv1.AddEntryRequest{CommissionId: commission, Kind: operatorv1.EntryKind_ENTRY_KIND_DIRECTION, Text: "Models first."}))
	if err != nil || resp.Msg.Warning != "" {
		t.Fatalf("after enable-steering there is no warning: %+v %v", resp.Msg, err)
	}
	charter, _ := os.ReadFile(filepath.Join(research, "cohorts", commission, home.CharterFile))
	if !strings.Contains(string(charter), home.SteeringSentence) {
		t.Error("the charter carries the Steering sentence")
	}
}

func TestSetCourierNeedsOneTarget(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()
	if _, err := h.SetCourier(ctx, connect.NewRequest(&operatorv1.SetCourierRequest{CommissionId: commission, Global: true, On: true})); code(err) != connect.CodeInvalidArgument {
		t.Errorf("a commission and global together are refused: %v", err)
	}
	if _, err := h.SetCourier(ctx, connect.NewRequest(&operatorv1.SetCourierRequest{CommissionId: "nope", On: true})); code(err) != connect.CodeNotFound {
		t.Errorf("an unknown commission is not found: %v", err)
	}
	resp, err := h.SetCourier(ctx, connect.NewRequest(&operatorv1.SetCourierRequest{CommissionId: commission, On: true}))
	if err != nil || !resp.Msg.On {
		t.Fatalf("on: %+v %v", resp, err)
	}
}

func TestFileRoute(t *testing.T) {
	_, research := newHandler(t)
	outside := filepath.Join(filepath.Dir(research), "secret.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(research, "topics", "link.txt")); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(fileHandler(research))
	defer srv.Close()
	cases := map[string]int{
		FileRoute + "topics/niche-book-production/MISSION.md": http.StatusOK,
		FileRoute + "topics/link.txt":                         http.StatusNotFound,
		FileRoute + "../secret.txt":                           http.StatusNotFound,
		FileRoute + ".research-desk/x":                        http.StatusNotFound,
		FileRoute:                                             http.StatusBadRequest,
	}
	for path, want := range cases {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("%s: want %d, got %d", path, want, resp.StatusCode)
		}
		if want == http.StatusOK && !strings.Contains(resp.Header.Get("Content-Security-Policy"), "sandbox") {
			t.Errorf("%s: served files are sandboxed", path)
		}
	}
}

// [REQ:RD-P0-008]
// A review lists the topics REVIEW.md has a section for and its
// proposals; a verdict is appended verbatim to the review's VERDICTS.md and
// read back; the desk never writes the topic's MISSION.md.
func TestAReviewListsItsTopicsAndRecordsAVerdictVerbatim(t *testing.T) {
	h, research := newHandler(t)
	ctx := context.Background()
	dir := filepath.Join(research, home.ReviewsDir, "books-20261112")
	if err := os.MkdirAll(filepath.Join(dir, home.ProposalsDir), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		home.ReviewFile: "# Books review\n\n## niche-book-production\n\nThree titles shipped.\n\n## not-a-topic\n",
		filepath.Join(home.ProposalsDir, "tidy.md"): "Archive run 1.\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mission := filepath.Join(research, "topics", "niche-book-production", "MISSION.md")
	before, _ := os.ReadFile(mission)

	words := "Films first.\nKeep the books line running."
	added, err := h.AddVerdict(ctx, connect.NewRequest(&operatorv1.AddVerdictRequest{ReviewId: "books-20261112", Topic: "niche-book-production", Text: words}))
	if err != nil {
		t.Fatal(err)
	}
	if added.Msg.Path != filepath.Join(dir, home.VerdictsFile) || added.Msg.Entry.Body != words {
		t.Fatalf("the verdict goes verbatim to VERDICTS.md: %+v", added.Msg)
	}
	if _, err := h.AddVerdict(ctx, connect.NewRequest(&operatorv1.AddVerdictRequest{ReviewId: "absent", Topic: "niche-book-production", Text: "x"})); code(err) != connect.CodeNotFound {
		t.Errorf("an unknown review is not found: %v", err)
	}

	list, err := h.ListReviews(ctx, connect.NewRequest(&operatorv1.ListReviewsRequest{}))
	if err != nil || len(list.Msg.Reviews) != 1 {
		t.Fatalf("one review: %+v %v", list, err)
	}
	r := list.Msg.Reviews[0]
	if r.Title.Value != "Books review" || len(r.Topics) != 1 || r.Topics[0] != "niche-book-production" || len(r.Proposals) != 1 || !r.TopicsEvidence.GetKnown() {
		t.Errorf("title, existing topics only and proposals: %+v", r)
	}
	if max, _ := testutil.EntryChars(); list.Msg.EntryCharsMax != int32(max) {
		t.Errorf("the verdict limit comes from limits.json: %d, want %d", list.Msg.EntryCharsMax, max)
	}
	if len(r.Verdicts) != 1 || r.Verdicts[0].Body != words || r.Verdicts[0].Subject != "niche-book-production" {
		t.Errorf("the verdict reads back verbatim: %+v", r.Verdicts)
	}
	if after, _ := os.ReadFile(mission); string(after) != string(before) {
		t.Error("a verdict changed the topic's MISSION.md")
	}
}

// A dry run returns the exact text a write would append and writes
// nothing; every answer says when the supervisor will read it.
func TestADryRunShowsTheExactTextAndWhenItIsRead(t *testing.T) {
	h, research := newHandler(t)
	ctx := context.Background()
	operator := filepath.Join(research, "cohorts", commission, home.OperatorFile)
	dry, err := h.Answer(ctx, connect.NewRequest(&operatorv1.AnswerRequest{CommissionId: commission, ItemId: "AV4", OptionKey: "keep", DryRun: true}))
	if err != nil {
		t.Fatal(err)
	}
	if dry.Msg.Written || !strings.Contains(dry.Msg.Text, "· answer AV4 · via desk\nkeep — Keep it uninstalled\n") || dry.Msg.Path != operator {
		t.Fatalf("dry run: %+v", dry.Msg)
	}
	if _, err := os.Stat(operator); !os.IsNotExist(err) {
		t.Fatalf("a dry run wrote OPERATOR.md: %v", err)
	}
	if r := dry.Msg.Reading; r.GetWhen() != sharedv1.ReadWhen_READ_WHEN_NOT_UNTIL_STEERING || r.GetMessage() == "" || r.GetReadSince() == nil {
		t.Fatalf("the pre-steering charter is not read until steering is enabled: %+v", r)
	}
	real, err := h.Answer(ctx, connect.NewRequest(&operatorv1.AnswerRequest{CommissionId: commission, ItemId: "AV4", OptionKey: "keep"}))
	if err != nil || !real.Msg.Written {
		t.Fatalf("write: %+v %v", real, err)
	}
	if data, _ := os.ReadFile(operator); string(data) != dry.Msg.Text {
		t.Fatalf("the write appended %q, the dry run showed %q", data, dry.Msg.Text)
	}

	if _, err := h.EnableSteering(ctx, connect.NewRequest(&operatorv1.EnableSteeringRequest{CommissionId: commission})); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(operator)
	entry, err := h.AddEntry(ctx, connect.NewRequest(&operatorv1.AddEntryRequest{CommissionId: commission, Kind: operatorv1.EntryKind_ENTRY_KIND_DIRECTION, Text: "Models first.", DryRun: true}))
	if err != nil || entry.Msg.Written || !strings.HasSuffix(entry.Msg.Text, "· direction · via desk\nModels first.\n") || !strings.HasPrefix(entry.Msg.Text, "\n") {
		t.Fatalf("direction dry run: %+v %v", entry, err)
	}
	if after, _ := os.ReadFile(operator); string(after) != string(before) {
		t.Fatal("a dry run changed OPERATOR.md")
	}
	// The items appended to STATE.md after the build leave the commission
	// unknown, so when the entry is read is unknown too, and says so.
	if r := entry.Msg.Reading; r.GetWhen() != sharedv1.ReadWhen_READ_WHEN_UNKNOWN || !strings.Contains(r.GetMessage(), "unknown") {
		t.Fatalf("with steering, an unknown commission's reading is unknown: %+v", r)
	}

	dir := filepath.Join(research, home.ReviewsDir, "books-20261112")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	verdict, err := h.AddVerdict(ctx, connect.NewRequest(&operatorv1.AddVerdictRequest{ReviewId: "books-20261112", Topic: "niche-book-production", Text: "Films first.", DryRun: true}))
	if err != nil || verdict.Msg.Written || !strings.HasPrefix(verdict.Msg.Text, "# Verdicts — books-20261112\n\n") {
		t.Fatalf("verdict dry run: %+v %v", verdict, err)
	}
	if _, err := os.Stat(filepath.Join(dir, home.VerdictsFile)); !os.IsNotExist(err) {
		t.Fatalf("a dry run wrote VERDICTS.md: %v", err)
	}
}

// Delivery reasons are codes, and a global item says what it
// asks and where, instead of being answerable.
func TestAGlobalItemCarriesItsInstructionAndEveryReasonIsACode(t *testing.T) {
	h, _ := newHandler(t)
	feed, err := h.ListNeedsYou(context.Background(), connect.NewRequest(&operatorv1.ListNeedsYouRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var global int
	for _, it := range feed.Msg.Items {
		if it.DeliveryReasonCode == operatorv1.DeliveryReason_DELIVERY_REASON_UNSPECIFIED || it.Courier.GetReason() == operatorv1.DeliveryReason_DELIVERY_REASON_UNSPECIFIED {
			t.Errorf("%s: no reason code (%q, %q)", it.Id, it.DeliveryReason, it.Courier.GetWhy())
		}
		if it.CommissionId != "" {
			if it.Answerable && it.Instruction != "" {
				t.Errorf("%s: an answerable item needs no instruction", it.Id)
			}
			continue
		}
		global++
		if it.Answerable || it.Instruction == "" {
			t.Errorf("%s: a global item is never answerable and says what to do: %+v", it.Id, it)
		}
		if it.DeliveryReasonCode != operatorv1.DeliveryReason_DELIVERY_REASON_NO_COMMISSION && it.DeliveryReasonCode != operatorv1.DeliveryReason_DELIVERY_REASON_NOTICE {
			t.Errorf("%s: global item reason %s", it.Id, it.DeliveryReasonCode)
		}
		if it.Id == "FRAUD-FREE-KEYS-20261107" && (len(it.Links) != 1 || !strings.HasSuffix(it.Links[0].Target, "topics/public-data-fraud/MISSION.md") || !filepath.IsAbs(it.Links[0].Target)) {
			t.Errorf("links resolve beside ATTENTION.md: %+v", it.Links)
		}
	}
	if global != 7 {
		t.Errorf("want the 7 real global items, got %d", global)
	}
}

// One call records a look at everything.
func TestCaughtUpMarksEverythingSeen(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()
	if _, err := h.MarkSeen(ctx, connect.NewRequest(&operatorv1.MarkSeenRequest{All: true, Id: "x"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("all with an id: err = %v, want invalid_argument", err)
	}
	resp, err := h.MarkSeen(ctx, connect.NewRequest(&operatorv1.MarkSeenRequest{All: true}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Commissions != 4 || resp.Msg.Topics != 5 || resp.Msg.At == nil || resp.Msg.Path != "" {
		t.Fatalf("every commission and topic: %+v", resp.Msg)
	}
	var look home.Look
	if found, err := h.deps.Writer.ReadState(home.LookName(home.LookTopic, "niche-book-production"), &look); err != nil || !found || !look.At.Equal(resp.Msg.At.AsTime()) {
		t.Fatalf("the topic's look: %v %v %+v", found, err, look)
	}
}

// An older numbered needs-you list is one row of the feed, readable in place,
// never sent and never answerable. The item count is the items still waiting:
// an answered item is listed but not counted.
func TestOlderListsAreFeedRows(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()
	if _, err := h.Answer(ctx, connect.NewRequest(&operatorv1.AnswerRequest{CommissionId: commission, ItemId: "AV4", OptionKey: "keep"})); err != nil {
		t.Fatal(err)
	}
	resp, err := h.ListNeedsYou(ctx, connect.NewRequest(&operatorv1.ListNeedsYouRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var items, waiting, older int
	answered := false
	var hostDisk *operatorv1.NeedsYouItem
	for _, it := range resp.Msg.Items {
		switch it.Kind {
		case operatorv1.NeedsYouKind_NEEDS_YOU_KIND_ITEM:
			if older > 0 {
				t.Errorf("items come before older lists: %s after an older list", it.Id)
			}
			items++
			if it.Answer == nil {
				waiting++
			} else if it.Id == "AV4" {
				answered = true
			}
		case operatorv1.NeedsYouKind_NEEDS_YOU_KIND_OLDER_LIST:
			older++
			if it.Answerable || it.DeliveryReasonCode != operatorv1.DeliveryReason_DELIVERY_REASON_OLDER_LIST || it.Courier.GetAction() != "none" || it.Body == "" || len(it.Sources) != 1 || it.Instruction == "" {
				t.Errorf("an older list is shown in place, never sent or answered, and says where to act: %+v", it)
			}
			if strings.HasPrefix(it.Body, "1. **HOST-DISK") {
				hostDisk = it
			}
		default:
			t.Errorf("every row has a kind: %+v", it)
		}
	}
	if !answered || waiting == 0 || waiting >= items || int(resp.Msg.ItemCount) != waiting || int(resp.Msg.OlderListCount) != older {
		t.Errorf("counts: %d items (%d waiting, AV4 answered %v) and %d older lists, response says %d and %d", items, waiting, answered, older, resp.Msg.ItemCount, resp.Msg.OlderListCount)
	}
	if hostDisk == nil || hostDisk.CommissionId != "programme-three-20261111" || !strings.Contains(hostDisk.Instruction, "programme-three-20261111") {
		t.Errorf("programme-three's real numbered list is a row naming its commission: %+v", hostDisk)
	}
}

// Enabling steering has the same dry run as the other writes: the exact text
// and the charter it would be appended to, with nothing written.
func TestEnableSteeringDryRunShowsTheExactText(t *testing.T) {
	h, research := newHandler(t)
	ctx := context.Background()
	charter := filepath.Join(research, "cohorts", commission, home.CharterFile)
	before, _ := os.ReadFile(charter)
	dry, err := h.EnableSteering(ctx, connect.NewRequest(&operatorv1.EnableSteeringRequest{CommissionId: commission, DryRun: true}))
	if err != nil || dry.Msg.Written || dry.Msg.Path != charter || !strings.Contains(dry.Msg.Text, "## Steering\n\n"+home.SteeringSentence) {
		t.Fatalf("dry run: %+v %v", dry, err)
	}
	if now, _ := os.ReadFile(charter); string(now) != string(before) {
		t.Fatal("a dry run writes nothing")
	}
	real, err := h.EnableSteering(ctx, connect.NewRequest(&operatorv1.EnableSteeringRequest{CommissionId: commission}))
	if err != nil || !real.Msg.Written || real.Msg.Text != dry.Msg.Text {
		t.Fatalf("the write appends what the dry run showed: %+v %v", real, err)
	}
	if after, _ := os.ReadFile(charter); string(after) != string(before)+dry.Msg.Text {
		t.Error("the charter gains exactly the shown text")
	}
}

// Verdicts written by hand with any RFC 3339 time are listed (the real
// local-models review has two such); a heading that does not parse is
// reported with its line.
func TestHandWrittenVerdictsAreListed(t *testing.T) {
	h, research := newHandler(t)
	dir := filepath.Join(research, home.ReviewsDir, "local-models-20261111")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	verdicts := "# Verdicts — local-models-20261111\n\n" +
		"### 2026-11-12T00:42:22.604567+00:00 · verdict local-model-performance · via hand\nKeep the bench.\n\n" +
		"### 2026-11-12T01:06:43.347461+00:00 · verdict local-model-performance · via hand\nThen tidy.\n\n" +
		"### yesterday · verdict local-model-performance · via hand\nLost?\n"
	if err := os.WriteFile(filepath.Join(dir, home.VerdictsFile), []byte(verdicts), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := h.ListReviews(context.Background(), connect.NewRequest(&operatorv1.ListReviewsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range resp.Msg.Reviews {
		if r.Id != "local-models-20261111" {
			continue
		}
		if len(r.Verdicts) != 2 || r.Verdicts[1].Body != "Then tidy." {
			t.Errorf("both verdicts are listed: %+v", r.Verdicts)
		}
		if len(r.UnparsedHeadings) != 1 || r.UnparsedHeadings[0].Line != 9 {
			t.Errorf("the unreadable heading is reported with its line: %+v", r.UnparsedHeadings)
		}
		return
	}
	t.Fatal("the review is listed")
}
