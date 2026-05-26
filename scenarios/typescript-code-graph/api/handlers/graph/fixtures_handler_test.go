package graph

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	intgraph "typescript-code-graph/internal/graph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

// newFixtureClient mounts the Connect handler with FixturesDir pointed at the
// scenario's bas/fixtures directory. The GraphService is constructed with a
// nil sidecar client because the exercised paths (ListFixtures and
// ValidateFixture's input-validation branches) never reach Extract — the real
// extract+compare path is covered by the determinism / integration tests,
// which require the live ts-morph sidecar.
func newFixtureClient(t *testing.T) graph_v1connect.TypeScriptCodeGraphServiceClient {
	t.Helper()
	fixturesDir, err := filepath.Abs("../../../bas/fixtures")
	if err != nil {
		t.Fatalf("resolve fixtures dir: %v", err)
	}
	svc := intgraph.NewService(nil, intgraph.NewPathMutex())
	_, h := graph_v1connect.NewTypeScriptCodeGraphServiceHandler(NewConnectHandler(Deps{
		GraphService: svc,
		Logger:       log.New(io.Discard, "", 0),
		FixturesDir:  fixturesDir,
	}))
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return graph_v1connect.NewTypeScriptCodeGraphServiceClient(server.Client(), server.URL)
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
	for _, want := range []string{"ts-junk-drawer", "ts-jsdoc-tags"} {
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
	if !strings.Contains(diff, "- c") || !strings.Contains(diff, "+ X") {
		t.Fatalf("diff did not isolate the changed line:\n%s", diff)
	}
	if strings.Contains(diff, "- a") || strings.Contains(diff, "+ a") {
		t.Fatalf("common prefix line leaked into diff body:\n%s", diff)
	}
}
