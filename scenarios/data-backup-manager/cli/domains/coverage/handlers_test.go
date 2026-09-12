package coverage

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"
	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage/coverage_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

// stubCoverageService records requests and returns canned responses.
type stubCoverageService struct {
	report    *coveragev1.CoverageReport
	gotAccept *coveragev1.AcceptDefaultTargetsRequest
	accept    *coveragev1.AcceptDefaultTargetsResponse
}

func (s *stubCoverageService) GetCoverageReport(_ context.Context, _ *connect.Request[coveragev1.GetCoverageReportRequest]) (*connect.Response[coveragev1.GetCoverageReportResponse], error) {
	return connect.NewResponse(&coveragev1.GetCoverageReportResponse{Report: s.report}), nil
}

func (s *stubCoverageService) AcceptDefaultTargets(_ context.Context, req *connect.Request[coveragev1.AcceptDefaultTargetsRequest]) (*connect.Response[coveragev1.AcceptDefaultTargetsResponse], error) {
	s.gotAccept = req.Msg
	return connect.NewResponse(s.accept), nil
}

func newStubApp(t *testing.T, stub *stubCoverageService) *cliapp.ScenarioApp {
	t.Helper()
	mux := http.NewServeMux()
	path, h := coverageconnect.NewCoverageServiceHandler(stub)
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

func TestReportCommand_ShowsCountsAndNextSteps(t *testing.T) {
	stub := &stubCoverageService{report: &coveragev1.CoverageReport{
		Summary: &coveragev1.CoverageSummary{
			RegisteredCount:         1,
			RecommendedCount:        2,
			SensitiveCount:          1,
			PlannedCount:            0,
			DefaultCoverageComplete: false,
			HasSensitiveUnreviewed:  true,
		},
		RecommendedTargets: []*coveragev1.SuggestedTarget{
			{Id: "s1", Owner: "vrooli", Name: "plans", Locator: "/p"},
			{Id: "s2", Owner: "vrooli", Name: "config", Locator: "/c"},
		},
		SensitiveTargets: []*coveragev1.SuggestedTarget{
			{Id: "s3", Owner: "codex", Name: "auth", Locator: "/a", Sensitive: true, Warning: "tokens"},
		},
		RegisteredTargets: []*coveragev1.RegisteredTarget{
			{Id: "t1", Owner: "swarm-manager", Name: "domain-data", Locator: "/d"},
		},
	}}
	hs := newHandlers(newStubApp(t, stub))
	schema := cliapp.ArgSchema{}
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, schema, cliapptest.TestRunContextOptions{})

	if err := hs.report(ctx); err != nil {
		t.Fatalf("report: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"1 registered", "2 recommended-unregistered", "INCOMPLETE", "coverage accept-defaults", "s1", "codex/auth"} {
		if !strings.Contains(out, want) {
			t.Errorf("report output missing %q\n%s", want, out)
		}
	}
}

func TestAcceptDefaultsCommand_PassesFlagsAndRendersResult(t *testing.T) {
	stub := &stubCoverageService{accept: &coveragev1.AcceptDefaultTargetsResponse{
		Accepted: []*coveragev1.AcceptedTarget{
			{TargetId: "tn", SuggestionId: "s1", Owner: "vrooli", Name: "plans", Locator: "/p"},
		},
		SkippedSensitive: []*coveragev1.SuggestedTarget{
			{Id: "s3", Owner: "codex", Name: "auth", Sensitive: true},
		},
		DryRun: false,
	}}
	hs := newHandlers(newStubApp(t, stub))
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "include-sensitive", Bool: true},
		{Name: "dry-run", Bool: true},
	}}
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, schema, cliapptest.TestRunContextOptions{
		BoolFlags: map[string]bool{"dry-run": false, "include-sensitive": false},
	})

	if err := hs.acceptDefaults(ctx); err != nil {
		t.Fatalf("acceptDefaults: %v", err)
	}
	if stub.gotAccept == nil {
		t.Fatal("server did not receive AcceptDefaultTargets")
	}
	if stub.gotAccept.IncludeSensitive || stub.gotAccept.DryRun {
		t.Fatalf("flags not defaulted false: %+v", stub.gotAccept)
	}
	out := stdout.String()
	for _, want := range []string{"Registered 1", "skipped 1 sensitive", "include-sensitive"} {
		if !strings.Contains(out, want) {
			t.Errorf("accept output missing %q\n%s", want, out)
		}
	}
}

func TestAcceptDefaultsCommand_DryRunFlag(t *testing.T) {
	stub := &stubCoverageService{accept: &coveragev1.AcceptDefaultTargetsResponse{
		Accepted: []*coveragev1.AcceptedTarget{{SuggestionId: "s1", Owner: "vrooli", Name: "plans"}},
		DryRun:   true,
	}}
	hs := newHandlers(newStubApp(t, stub))
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "include-sensitive", Bool: true},
		{Name: "dry-run", Bool: true},
	}}
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, schema, cliapptest.TestRunContextOptions{
		BoolFlags: map[string]bool{"dry-run": true},
	})

	if err := hs.acceptDefaults(ctx); err != nil {
		t.Fatalf("acceptDefaults: %v", err)
	}
	if !stub.gotAccept.DryRun {
		t.Fatal("dry-run flag not passed to server")
	}
	if !strings.Contains(stdout.String(), "Dry run") {
		t.Fatalf("dry-run not surfaced: %s", stdout.String())
	}
}

func TestRegisterCoverageLoadsFromManifest(t *testing.T) {
	manifest := readManifest(t)
	app := &cliapp.ScenarioApp{}
	group, err := Register(app, manifest)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	got := map[string]bool{}
	for _, c := range group.Subcommands {
		got[c.Name] = true
	}
	for _, want := range []string{"report", "accept-defaults"} {
		if !got[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}
