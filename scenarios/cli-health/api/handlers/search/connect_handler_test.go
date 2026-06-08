package search

import (
	"context"
	"io"
	"log"
	"testing"

	"connectrpc.com/connect"

	"cli-health/internal/aisearch"
	pkg "github.com/vrooli/aisearch-go"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search"
)

// recordingSearcher records how many SearchOptions the handler threaded through
// — the observable proxy for "were the request's overrides honored". It cannot
// inspect the opaque option closures, but their PRESENCE is exactly what the
// gate decides, so counting them tests the gate end to end.
type recordingSearcher struct {
	lastOptCount int
}

func (r *recordingSearcher) Search(_ context.Context, _ string, _ int, _ aisearch.SearchMode, opts ...pkg.SearchOption) (*aisearch.SearchResponse, error) {
	r.lastOptCount = len(opts)
	return &aisearch.SearchResponse{Method: "ai"}, nil
}

func (r *recordingSearcher) Status(context.Context) aisearch.StatusReport {
	return aisearch.StatusReport{}
}

func quietHandler(t *testing.T, gate *OverrideGate) (*connectHandler, *recordingSearcher) {
	t.Helper()
	rs := &recordingSearcher{}
	h := NewConnectHandler(Deps{
		Logger:    log.New(io.Discard, "", 0),
		Searcher:  rs,
		Overrides: gate,
	})
	return h, rs
}

// searchWith issues a Search through the handler with the given override +
// token headers (empty string = header omitted) and returns the opt count the
// searcher saw.
func searchWith(t *testing.T, h *connectHandler, rs *recordingSearcher, overridesHdr, tokenHdr string) int {
	t.Helper()
	req := connect.NewRequest(&searchv1.SearchRequest{Query: "restart a scenario", Limit: 10})
	if overridesHdr != "" {
		req.Header().Set(pkg.OverridesHeader, overridesHdr)
	}
	if tokenHdr != "" {
		req.Header().Set(pkg.ControlTokenHeader, tokenHdr)
	}
	if _, err := h.Search(context.Background(), req); err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	return rs.lastOptCount
}

const goodOverrides = `{"rerank_enabled":true,"rerank_shortlist":40}`

func gateEnabled(token string) *OverrideGate {
	return &OverrideGate{Token: func() string { return token }}
}

func TestOverrideIgnoredWhenNoHeader(t *testing.T) {
	h, rs := quietHandler(t, gateEnabled("tok"))
	if n := searchWith(t, h, rs, "", "tok"); n != 0 {
		t.Fatalf("a request with no override header must thread 0 options, got %d", n)
	}
}

func TestOverrideIgnoredWhenGateNil(t *testing.T) {
	h, rs := quietHandler(t, nil)
	if n := searchWith(t, h, rs, goodOverrides, "tok"); n != 0 {
		t.Fatalf("nil gate must ignore overrides, got %d options", n)
	}
}

func TestOverrideIgnoredWhenGateHasNilTokenFunc(t *testing.T) {
	// A gate with no token source can never match — the channel is closed until
	// self-registration populates the token. (The old per-env "disabled channel"
	// flag is gone; token presence is the only gate.)
	h, rs := quietHandler(t, &OverrideGate{Token: nil})
	if n := searchWith(t, h, rs, goodOverrides, "tok"); n != 0 {
		t.Fatalf("gate with nil token func must ignore overrides, got %d options", n)
	}
}

func TestOverrideIgnoredWhenTokenMissing(t *testing.T) {
	h, rs := quietHandler(t, gateEnabled("tok"))
	if n := searchWith(t, h, rs, goodOverrides, ""); n != 0 {
		t.Fatalf("missing token header must ignore overrides, got %d options", n)
	}
}

func TestOverrideIgnoredWhenTokenWrong(t *testing.T) {
	h, rs := quietHandler(t, gateEnabled("tok"))
	if n := searchWith(t, h, rs, goodOverrides, "not-the-token"); n != 0 {
		t.Fatalf("wrong token must ignore overrides, got %d options", n)
	}
}

func TestOverrideIgnoredWhenCachedTokenEmpty(t *testing.T) {
	// Provider not yet registered → cached token "" → no override ever matches,
	// even if the caller presents an empty token too.
	h, rs := quietHandler(t, gateEnabled(""))
	if n := searchWith(t, h, rs, goodOverrides, ""); n != 0 {
		t.Fatalf("empty cached token must close the channel, got %d options", n)
	}
}

func TestOverrideHonoredWhenTokenMatches(t *testing.T) {
	h, rs := quietHandler(t, gateEnabled("tok"))
	if n := searchWith(t, h, rs, goodOverrides, "tok"); n != 1 {
		t.Fatalf("matching token must thread the override option, got %d", n)
	}
}

func TestOverrideIgnoredWhenMalformedJSON(t *testing.T) {
	h, rs := quietHandler(t, gateEnabled("tok"))
	if n := searchWith(t, h, rs, "{not json", "tok"); n != 0 {
		t.Fatalf("malformed override header must be ignored (not error), got %d options", n)
	}
}

func TestOverrideIgnoredWhenEmptyObject(t *testing.T) {
	// A well-formed but factor-less override object resolves to zero and is a
	// no-op (no option threaded), even past the gate.
	h, rs := quietHandler(t, gateEnabled("tok"))
	if n := searchWith(t, h, rs, `{}`, "tok"); n != 0 {
		t.Fatalf("empty override object must thread 0 options, got %d", n)
	}
}

// TestTokenMatchesConstantTime is a direct check of the token comparison helper
// (the gate's security primitive) across the empty/nil edge cases.
func TestTokenMatches(t *testing.T) {
	cases := []struct {
		name      string
		cached    string
		presented string
		want      bool
	}{
		{"match", "tok", "tok", true},
		{"mismatch", "tok", "nope", false},
		{"empty cached", "", "tok", false},
		{"empty presented", "tok", "", false},
		{"both empty", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := &OverrideGate{Token: func() string { return c.cached }}
			if got := g.tokenMatches(c.presented); got != c.want {
				t.Fatalf("tokenMatches(%q,%q) = %v, want %v", c.cached, c.presented, got, c.want)
			}
		})
	}
	// nil Token func never matches.
	g := &OverrideGate{Token: nil}
	if g.tokenMatches("anything") {
		t.Fatal("nil Token func must never match")
	}
}
