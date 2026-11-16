package coverage

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	review    *coveragev1.RequestSensitiveReviewResponse
	gotReview bool
	policy    *coveragev1.SensitivePolicy
	gotRevoke *coveragev1.RevokeSensitivePolicyRequest
	gotExcl   *coveragev1.ExcludeSensitiveTargetRequest
	gotAdopt  *coveragev1.AdoptSensitiveConsentRequest
}

func (s *stubCoverageService) GetSensitivePolicy(context.Context, *connect.Request[coveragev1.GetSensitivePolicyRequest]) (*connect.Response[coveragev1.GetSensitivePolicyResponse], error) {
	return connect.NewResponse(&coveragev1.GetSensitivePolicyResponse{Policy: s.policy}), nil
}

func (s *stubCoverageService) RevokeSensitivePolicy(_ context.Context, req *connect.Request[coveragev1.RevokeSensitivePolicyRequest]) (*connect.Response[coveragev1.RevokeSensitivePolicyResponse], error) {
	s.gotRevoke = req.Msg
	return connect.NewResponse(&coveragev1.RevokeSensitivePolicyResponse{Policy: s.policy}), nil
}

func (s *stubCoverageService) AdoptSensitiveConsent(_ context.Context, req *connect.Request[coveragev1.AdoptSensitiveConsentRequest]) (*connect.Response[coveragev1.AdoptSensitiveConsentResponse], error) {
	s.gotAdopt = req.Msg
	return connect.NewResponse(&coveragev1.AdoptSensitiveConsentResponse{Policy: s.policy, RegisteredCount: 44}), nil
}

func (s *stubCoverageService) ExcludeSensitiveTarget(_ context.Context, req *connect.Request[coveragev1.ExcludeSensitiveTargetRequest]) (*connect.Response[coveragev1.ExcludeSensitiveTargetResponse], error) {
	s.gotExcl = req.Msg
	return connect.NewResponse(&coveragev1.ExcludeSensitiveTargetResponse{Policy: s.policy}), nil
}

func (s *stubCoverageService) GetCoverageReport(_ context.Context, _ *connect.Request[coveragev1.GetCoverageReportRequest]) (*connect.Response[coveragev1.GetCoverageReportResponse], error) {
	return connect.NewResponse(&coveragev1.GetCoverageReportResponse{Report: s.report}), nil
}

func (s *stubCoverageService) AcceptDefaultTargets(_ context.Context, req *connect.Request[coveragev1.AcceptDefaultTargetsRequest]) (*connect.Response[coveragev1.AcceptDefaultTargetsResponse], error) {
	s.gotAccept = req.Msg
	return connect.NewResponse(s.accept), nil
}

func (s *stubCoverageService) RequestSensitiveReview(context.Context, *connect.Request[coveragev1.RequestSensitiveReviewRequest]) (*connect.Response[coveragev1.RequestSensitiveReviewResponse], error) {
	s.gotReview = true
	return connect.NewResponse(s.review), nil
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

func TestReportCommandMakesReacquisitionProofAndBoundsExplicit(t *testing.T) {
	stub := &stubCoverageService{report: &coveragev1.CoverageReport{
		Summary: &coveragev1.CoverageSummary{RegisteredCount: 1},
		ReacquirableTargets: []*coveragev1.SuggestedTarget{
			{Id: "model-cache", Owner: "ollama", Name: "models", Locator: "/models", Reacquirable: true, BudgetMaxBytes: "90GiB"},
			{Id: "pinned-model", Owner: "ace-step", Name: "models", Locator: "/pinned", Reacquirable: true, BudgetMaxAge: "30d", ReacquisitionCommand: "vrooli resource acquisition model-cache ace-step", ReacquisitionEvidence: "resource.json data_artifacts", ReacquisitionNeedsOperator: true, ReacquisitionAuthReason: "license"},
		},
		RegisteredTargets: []*coveragev1.RegisteredTarget{{Id: "t1", Owner: "ace-step", Name: "models", Reacquirable: true}},
	}}
	hs := newHandlers(newStubApp(t, stub))
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	if err := hs.report(ctx); err != nil {
		t.Fatalf("report: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{
		"Reacquisition evidence: 1/2 suggestions have both an owner command and evidence; 0 lack both a byte and age bound.",
		"ollama/models", "reacquisition-proof=INCOMPLETE", "ace-step/models",
		"reacquire=vrooli resource acquisition model-cache ace-step", "evidence=resource.json data_artifacts", "authorization-required=license",
		"model-cache", "1 registered",
	} {
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
		{Name: "dry-run", Bool: true},
	}}
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, schema, cliapptest.TestRunContextOptions{
		BoolFlags: map[string]bool{"dry-run": false},
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
	for _, want := range []string{"Registered 1", "skipped 1 sensitive", "review-sensitive"} {
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
	for _, want := range []string{"report", "accept-defaults", "review-sensitive", "sensitive-policy", "revoke-sensitive-policy", "exclude-sensitive", "adopt-sensitive-consent"} {
		if !got[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestReviewSensitiveCommand_RaisesOneDecisionOnly(t *testing.T) {
	stub := &stubCoverageService{review: &coveragev1.RequestSensitiveReviewResponse{AskId: "ask-1", CandidateCount: 44, Pending: true}}
	hs := newHandlers(newStubApp(t, stub))
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	if err := hs.reviewSensitive(ctx); err != nil {
		t.Fatalf("reviewSensitive: %v", err)
	}
	if !stub.gotReview {
		t.Fatal("review request was not sent")
	}
	for _, want := range []string{"ask-1", "44", "No target was registered"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output missing %q: %s", want, stdout.String())
		}
	}
}

func TestReviewSensitiveCommand_WithTheStandingRuleSaysNothingWasAsked(t *testing.T) {
	stub := &stubCoverageService{review: &coveragev1.RequestSensitiveReviewResponse{CandidateCount: 2, Pending: false}}
	hs := newHandlers(newStubApp(t, stub))
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	if err := hs.reviewSensitive(ctx); err != nil {
		t.Fatalf("reviewSensitive: %v", err)
	}
	for _, want := range []string{"standing rule is on", "nothing was asked", "2 place"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output missing %q: %s", want, stdout.String())
		}
	}
}

func TestSensitivePolicyCommand_ShowsWhoConsentedWhenAndWhatItAdded(t *testing.T) {
	granted := timestamppb.New(time.Date(2026, 11, 15, 9, 30, 0, 0, time.UTC))
	stub := &stubCoverageService{policy: &coveragev1.SensitivePolicy{
		Active: true, GrantedBy: "operator", GrantedByIdentityKind: "verified-human", GrantedByIdentitySource: "notification-hub", GrantedAt: granted, GrantAskId: "ask-9", AddedTotal: 3,
		Excluded:   []*coveragev1.SensitivePolicyExclusion{{Owner: "grok", Name: "home", Reason: "Too large", ExcludedBy: "operator"}},
		Unreported: []*coveragev1.SensitivePolicyAddition{{Owner: "codex", Name: "home", Group: "AI tools", ApproxBytes: 1 << 20}},
		History:    []*coveragev1.SensitivePolicyEvent{{At: granted, Kind: "granted", By: "operator", Detail: "Standing rule turned on"}},
	}}
	hs := newHandlers(newStubApp(t, stub))
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	if err := hs.sensitivePolicy(ctx); err != nil {
		t.Fatalf("sensitivePolicy: %v", err)
	}
	for _, want := range []string{"Standing rule ON", "operator", "verified-human", "2026-11-15T09:30:00Z", "ask-9", "grok/home", "Too large", "codex/home", "revoke-sensitive-policy"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output missing %q: %s", want, stdout.String())
		}
	}
}

func TestRevokeAndExcludeCommandsSendTheirRequests(t *testing.T) {
	stub := &stubCoverageService{policy: &coveragev1.SensitivePolicy{Active: false, RevokedBy: "operator via CLI", RevokedAt: timestamppb.New(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC))}}
	hs := newHandlers(newStubApp(t, stub))
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	if err := hs.revokeSensitivePolicy(ctx); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if stub.gotRevoke == nil || !strings.Contains(stdout.String(), "Standing rule OFF") || !strings.Contains(stdout.String(), "stay backed up") {
		t.Fatalf("revoke request=%v output=%s", stub.gotRevoke, stdout.String())
	}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "owner"}, {Name: "name"}, {Name: "reason"}}}
	ctx, stdout = cliapptest.NewCapturedRunContext(hs.core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"owner": "grok", "name": "home", "reason": "Too large"}})
	if err := hs.excludeSensitive(ctx); err != nil {
		t.Fatalf("exclude: %v", err)
	}
	if stub.gotExcl == nil || stub.gotExcl.Owner != "grok" || stub.gotExcl.Name != "home" || stub.gotExcl.Reason != "Too large" {
		t.Fatalf("exclude request = %+v", stub.gotExcl)
	}
	if !strings.Contains(stdout.String(), "grok/home is kept out") {
		t.Errorf("output = %s", stdout.String())
	}
}

func TestAdoptSensitiveConsentCommandSendsTheAskAndShowsTheVerbatimWords(t *testing.T) {
	stub := &stubCoverageService{policy: &coveragev1.SensitivePolicy{Active: true, GrantBasis: "free_text", GrantText: "back up anything that declares storage", GrantedBy: "operator", GrantedByIdentityKind: "verified-human", GrantedByIdentitySource: "notification-hub", GrantAskId: "hub-ask-1"}}
	hs := newHandlers(newStubApp(t, stub))
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "ask"}}}
	ctx, stdout := cliapptest.NewCapturedRunContext(hs.core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"ask": "hub-ask-1"}})
	if err := hs.adoptSensitiveConsent(ctx); err != nil {
		t.Fatal(err)
	}
	if stub.gotAdopt == nil || stub.gotAdopt.HubAskId != "hub-ask-1" {
		t.Fatalf("request = %+v", stub.gotAdopt)
	}
	for _, want := range []string{"given in their own words", "back up anything that declares storage", "Registered 44", "hub-ask-1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output missing %q: %s", want, stdout.String())
		}
	}
}

func TestDriveLossLineSaysWhatASameDriveCopyDoesNotProtect(t *testing.T) {
	line := driveLossLine(&coveragev1.CoverageSummary{RegisteredCount: 244, PhysicallyIndependentTargetCount: 0})
	for _, want := range []string{"same drive as its source", "deletion and corruption", "not against losing that drive", "0 of 244"} {
		if !strings.Contains(line, want) {
			t.Errorf("drive-loss line missing %q: %s", want, line)
		}
	}
}
