package graph

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	intgraph "go-code-graph/internal/graph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// newFixtureClient mounts the Connect handler with a REAL packages loader (the
// fixture validation flow needs genuine extraction) and FixturesDir pointed at
// the scenario's bas/fixtures directory.
func newFixtureClient(t *testing.T) graph_v1connect.GoCodeGraphServiceClient {
	t.Helper()
	fixturesDir, err := filepath.Abs("../../../bas/fixtures")
	if err != nil {
		t.Fatalf("resolve fixtures dir: %v", err)
	}
	svc := intgraph.NewService(intgraph.NewPackagesLoader(), intgraph.NewPathMutex())
	_, h := graph_v1connect.NewGoCodeGraphServiceHandler(NewConnectHandler(Deps{
		GraphService: svc,
		Logger:       log.New(io.Discard, "", 0),
		FixturesDir:  fixturesDir,
	}))
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return graph_v1connect.NewGoCodeGraphServiceClient(server.Client(), server.URL)
}

func TestListFixtures(t *testing.T) {
	client := newFixtureClient(t)
	resp, err := client.ListFixtures(context.Background(), connect.NewRequest(&graphv1.ListFixturesRequest{}))
	if err != nil {
		t.Fatalf("ListFixtures: %v", err)
	}
	byName := make(map[string]*graphv1.FixtureInfo)
	for _, f := range resp.Msg.GetFixtures() {
		byName[f.GetName()] = f
	}
	for _, want := range []string{"go-cycles", "go-mislocated", "go-usage-facts"} {
		f, ok := byName[want]
		if !ok {
			t.Fatalf("expected fixture %q in list, got %v", want, byName)
		}
		if !f.GetHasExpected() {
			t.Errorf("fixture %q: want has_expected=true", want)
		}
	}
	// Stable name-sorted order.
	fixtures := resp.Msg.GetFixtures()
	for i := 1; i < len(fixtures); i++ {
		if fixtures[i-1].GetName() > fixtures[i].GetName() {
			t.Fatalf("fixtures not sorted: %q before %q", fixtures[i-1].GetName(), fixtures[i].GetName())
		}
	}
}

func TestValidateFixturePasses(t *testing.T) {
	client := newFixtureClient(t)
	for _, name := range []string{"go-cycles", "go-mislocated", "go-usage-facts"} {
		name := name
		t.Run(name, func(t *testing.T) {
			resp, err := client.ValidateFixture(context.Background(), connect.NewRequest(&graphv1.ValidateFixtureRequest{Name: name}))
			if err != nil {
				t.Fatalf("ValidateFixture(%q): %v", name, err)
			}
			if !resp.Msg.GetPassed() {
				t.Fatalf("fixture %q expected to pass; diff:\n%s", name, resp.Msg.GetDiff())
			}
			if resp.Msg.GetGraphHash() == "" {
				t.Errorf("expected non-empty graph hash")
			}
			if resp.Msg.GetExpectedBytes() != resp.Msg.GetActualBytes() {
				t.Errorf("byte counts differ on a pass: expected=%d actual=%d",
					resp.Msg.GetExpectedBytes(), resp.Msg.GetActualBytes())
			}
		})
	}
}

func TestValidateFixtureInvalidName(t *testing.T) {
	client := newFixtureClient(t)
	for _, name := range []string{"", "../secrets", "a/b"} {
		_, err := client.ValidateFixture(context.Background(), connect.NewRequest(&graphv1.ValidateFixtureRequest{Name: name}))
		if err == nil {
			t.Fatalf("name %q: expected error", name)
		}
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("name %q: want InvalidArgument, got %v", name, connect.CodeOf(err))
		}
	}
}

func TestValidateFixtureNotFound(t *testing.T) {
	client := newFixtureClient(t)
	_, err := client.ValidateFixture(context.Background(), connect.NewRequest(&graphv1.ValidateFixtureRequest{Name: "does-not-exist"}))
	if err == nil {
		t.Fatal("expected error for missing fixture")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("want NotFound, got %v", connect.CodeOf(err))
	}
}

func TestLineDiffFocusesOnChange(t *testing.T) {
	expected := "a\nb\nc\nd\ne"
	actual := "a\nb\nX\nd\ne"
	diff := lineDiff(expected, actual)
	if !contains(diff, "- c") || !contains(diff, "+ X") {
		t.Fatalf("diff did not isolate the changed line:\n%s", diff)
	}
	if contains(diff, "- a") || contains(diff, "+ a") {
		t.Fatalf("common prefix line leaked into diff body:\n%s", diff)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
