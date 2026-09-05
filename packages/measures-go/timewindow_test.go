package measures

import (
	"testing"
	"time"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// anchor is a fixed, timezone-explicit "now" for deterministic window tests:
// 2026-06-10T12:00:00Z is a Wednesday, so the week's Monday is 2026-06-08.
var anchor = time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

func TestResolveToken(t *testing.T) {
	loc := time.UTC
	cases := []struct {
		token    TimeWindowToken
		wantFrom time.Time
		wantTo   time.Time
	}{
		{TokenThisWeek, time.Date(2026, 6, 8, 0, 0, 0, 0, loc), anchor},
		{TokenLast7d, anchor.AddDate(0, 0, -7), anchor},
		{TokenLast30d, anchor.AddDate(0, 0, -30), anchor},
		{TokenThisMonth, time.Date(2026, 6, 1, 0, 0, 0, 0, loc), anchor},
		{TokenLastMonth, time.Date(2026, 5, 1, 0, 0, 0, 0, loc), time.Date(2026, 6, 1, 0, 0, 0, 0, loc)},
		{TokenThisQuarter, time.Date(2026, 4, 1, 0, 0, 0, 0, loc), anchor},
	}
	for _, c := range cases {
		got, err := ResolveToken(c.token, anchor, loc)
		if err != nil {
			t.Fatalf("ResolveToken(%s): %v", c.token, err)
		}
		if !got.From.Equal(c.wantFrom) {
			t.Errorf("%s From = %s, want %s", c.token, got.From, c.wantFrom)
		}
		if !got.To.Equal(c.wantTo) {
			t.Errorf("%s To = %s, want %s", c.token, got.To, c.wantTo)
		}
		if !got.To.After(got.From) {
			t.Errorf("%s: To %s not after From %s", c.token, got.To, got.From)
		}
	}
}

func TestResolveTokenTimezone(t *testing.T) {
	// In a +14:00 zone, 2026-06-10T12:00:00Z is already 2026-06-11T02:00 local,
	// so "this month" still starts at the local 1st — exercising loc handling.
	loc := time.FixedZone("plus14", 14*3600)
	got, err := ResolveToken(TokenThisMonth, anchor, loc)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	if !got.From.Equal(wantFrom) {
		t.Errorf("this_month From = %s, want %s", got.From, wantFrom)
	}
}

func TestResolveTokenUnknown(t *testing.T) {
	if _, err := ResolveToken("nope", anchor, time.UTC); err == nil {
		t.Fatal("expected error for unknown token")
	}
}

func TestResolveTimeWindowProto(t *testing.T) {
	// Relative token via proto.
	tw := &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{
		Token: measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D,
	}}
	got, err := ResolveTimeWindow(tw, anchor, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if !got.From.Equal(anchor.AddDate(0, 0, -7)) {
		t.Errorf("last_7d From = %s", got.From)
	}

	// Custom absolute range.
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	custom := &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Custom{
		Custom: &measuresv1.CustomRange{From: timestamppb.New(from), To: timestamppb.New(to)},
	}}
	gotc, err := ResolveTimeWindow(custom, anchor, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if !gotc.From.Equal(from) || !gotc.To.Equal(to) {
		t.Errorf("custom range = [%s,%s), want [%s,%s)", gotc.From, gotc.To, from, to)
	}

	// Inverted custom range rejected.
	bad := &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Custom{
		Custom: &measuresv1.CustomRange{From: timestamppb.New(to), To: timestamppb.New(from)},
	}}
	if _, err := ResolveTimeWindow(bad, anchor, time.UTC); err == nil {
		t.Error("expected error for inverted custom range")
	}

	// Unspecified token rejected.
	unset := &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{
		Token: measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED,
	}}
	if _, err := ResolveTimeWindow(unset, anchor, time.UTC); err == nil {
		t.Error("expected error for unspecified token")
	}
}

func TestMatchTimeWindowToken(t *testing.T) {
	cases := map[string]TimeWindowToken{
		"how many backlog items did we complete this week": TokenThisWeek,
		"backlog items closed last month":                  TokenLastMonth,
		"throughput over the last 7 days":                  TokenLast7d,
		"completed work in the last 30 days":               TokenLast30d,
		"items this month":                                 TokenThisMonth,
		"velocity this quarter":                            TokenThisQuarter,
	}
	for q, want := range cases {
		got, ok := MatchTimeWindowToken(q)
		if !ok || got != want {
			t.Errorf("MatchTimeWindowToken(%q) = (%q,%v), want %q", q, got, ok, want)
		}
	}
	if _, ok := MatchTimeWindowToken("how many items in total"); ok {
		t.Error("expected no match for a question with no time phrase")
	}
}

func TestTokenProtoRoundTrip(t *testing.T) {
	for _, tok := range []TimeWindowToken{TokenThisWeek, TokenLast7d, TokenLast30d, TokenThisMonth, TokenLastMonth, TokenThisQuarter} {
		back, ok := TokenFromProto(tok.Proto())
		if !ok || back != tok {
			t.Errorf("round trip %q -> %v -> (%q,%v)", tok, tok.Proto(), back, ok)
		}
	}
}
