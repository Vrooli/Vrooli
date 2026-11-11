package graphreport

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	"google.golang.org/protobuf/types/known/emptypb"
)

func context(t *testing.T) (cliapp.RunContext, *bytes.Buffer) {
	t.Helper()
	core := cliapptest.NewTestApp(t, nil)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	return ctx, out
}

func TestRenderExtractionSortsAndCapsPackagesAndFormatsWarnings(t *testing.T) {
	ctx, out := context(t)
	packages := []string{"z", "a", "m", "b", "c", "d", "e", "f", "g", "h", "i"}
	if err := RenderExtraction(ctx, &emptypb.Empty{}, Extraction{
		Nodes: 11, Edges: 4, Warnings: 2, ElapsedMS: 42, GraphHash: "abc", Path: "/tmp/project", RewriteFlag: "project-path", RewriteTarget: "project", Packages: packages, ShowWarningsAlways: true,
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Extracted 11 node(s), 4 edge(s), 2 warning(s) in 42ms") {
		t.Fatalf("summary missing: %s", out)
	}
	if !strings.Contains(out.String(), "--project-path /tmp/project") || !strings.Contains(out.String(), "plan a rewrite for this project") {
		t.Fatalf("rewrite hint missing: %s", out)
	}
	if strings.Contains(out.String(), "\n  z\n") {
		t.Fatalf("package list was not capped: %s", out)
	}
	if packages[0] != "z" {
		t.Fatal("rendering mutated caller slice")
	}
}

func TestRenderExtractionKeepsOptionalWarningLine(t *testing.T) {
	ctx, out := context(t)
	if err := RenderExtraction(ctx, &emptypb.Empty{}, Extraction{Nodes: 1, Edges: 2, Warnings: 1, ElapsedMS: 3, GraphHash: "hash", RewriteFlag: "module-path"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Extracted 1 node(s), 2 edge(s) in 3ms") || !strings.Contains(out.String(), "1 warning(s) returned.") {
		t.Fatalf("Go-style optional warning summary changed: %s", out)
	}
}

func TestRenderFixturesAndFailedValidation(t *testing.T) {
	ctx, out := context(t)
	if err := RenderFixtures(ctx, &emptypb.Empty{}, []Fixture{{Name: "b", Path: "b", HasExpected: true}, {Name: "a", Path: "a"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "2 fixture(s) available") || !strings.Contains(out.String(), "expected graph present") {
		t.Fatalf("fixture response missing details: %s", out)
	}
	if strings.Index(out.String(), "a (a)") > strings.Index(out.String(), "b (b)") {
		t.Fatalf("fixtures are not sorted: %s", out)
	}

	ctx, out = context(t)
	err := RenderValidation(ctx, &emptypb.Empty{}, Validation{Name: "bad", Diff: "line mismatch", Passed: false, ExpectedBytes: 4, ActualBytes: 3})
	if err == nil || !strings.Contains(err.Error(), `fixture "bad" does not match its expected graph`) {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(out.String(), "FAIL — fixture \"bad\"") || !strings.Contains(out.String(), "line mismatch") {
		t.Fatalf("failure details missing: %s", out)
	}
}
