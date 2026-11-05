package coverage

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/sources"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
)

// fakeService records calls and returns canned domain values so the contract
// test asserts pure request→domain and domain→response translation.
type fakeService struct {
	report       coverage.Report
	reportErr    error
	accept       coverage.AcceptResult
	acceptOpts   []coverage.AcceptOptions
	unregistered []coverage.Suggestion
}

func (f *fakeService) Report(context.Context) (coverage.Report, error) {
	return f.report, f.reportErr
}

func (f *fakeService) AcceptDefaults(_ context.Context, opts coverage.AcceptOptions) (coverage.AcceptResult, error) {
	f.acceptOpts = append(f.acceptOpts, opts)
	return f.accept, nil
}

func (f *fakeService) UnregisteredDefaultTargets(context.Context) ([]coverage.Suggestion, error) {
	return f.unregistered, nil
}

func TestCoverageService_Contract(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1700000000, 0).UTC()

	t.Run("GetCoverageReport maps summary, rows and suggestions", func(t *testing.T) {
		svc := &fakeService{report: coverage.Report{
			Summary: coverage.Summary{
				RegisteredCount:         1,
				RecommendedCount:        2,
				SensitiveCount:          1,
				PlannedCount:            1,
				BackedUpCount:           1,
				VerifiedCount:           1,
				DefaultCoverageComplete: false,
				HasSensitiveUnreviewed:  true,
			},
			Registered: []coverage.RegisteredTarget{{
				CatalogTarget:  coverage.CatalogTarget{ID: "t1", Owner: "vrooli", Name: "secrets", SourceKind: sources.KindFilesystem, Locator: "/d"},
				Planned:        true,
				LastSuccessAt:  now,
				LastVerifiedAt: now,
			}},
			Recommended: []coverage.Suggestion{{ID: "s1", Owner: "vrooli", Name: "plans", SourceKind: sources.KindFilesystem, Locator: "/p"}},
			Sensitive:   []coverage.Suggestion{{ID: "s2", Owner: "codex", Name: "auth", SourceKind: sources.KindFilesystem, Locator: "/a", Sensitive: true, Warning: "careful"}},
		}}
		h := NewConnectHandler(Deps{Service: svc})

		resp, err := h.GetCoverageReport(ctx, connect.NewRequest(&coveragev1.GetCoverageReportRequest{}))
		if err != nil {
			t.Fatalf("GetCoverageReport: %v", err)
		}
		rep := resp.Msg.Report
		if rep.Summary.RecommendedCount != 2 || !rep.Summary.HasSensitiveUnreviewed {
			t.Fatalf("summary not mapped: %+v", rep.Summary)
		}
		if len(rep.RegisteredTargets) != 1 || rep.RegisteredTargets[0].Id != "t1" || !rep.RegisteredTargets[0].Planned {
			t.Fatalf("registered not mapped: %+v", rep.RegisteredTargets)
		}
		if rep.RegisteredTargets[0].LastSuccessAt == nil || rep.RegisteredTargets[0].LastVerifiedAt == nil {
			t.Fatal("timestamps missing")
		}
		if rep.RegisteredTargets[0].SourceKind != sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM {
			t.Fatalf("source kind not mapped: %v", rep.RegisteredTargets[0].SourceKind)
		}
		if len(rep.RecommendedTargets) != 1 || len(rep.SensitiveTargets) != 1 {
			t.Fatalf("suggestions not mapped: rec=%d sens=%d", len(rep.RecommendedTargets), len(rep.SensitiveTargets))
		}
		if rep.SensitiveTargets[0].Warning != "careful" || !rep.SensitiveTargets[0].Sensitive {
			t.Fatalf("sensitive suggestion not mapped: %+v", rep.SensitiveTargets[0])
		}
	})

	t.Run("GetCoverageReport surfaces internal error", func(t *testing.T) {
		svc := &fakeService{reportErr: context.DeadlineExceeded}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.GetCoverageReport(ctx, connect.NewRequest(&coveragev1.GetCoverageReportRequest{}))
		if connect.CodeOf(err) != connect.CodeInternal {
			t.Fatalf("code = %v, want internal", connect.CodeOf(err))
		}
	})

	t.Run("AcceptDefaultTargets passes options and maps results", func(t *testing.T) {
		svc := &fakeService{accept: coverage.AcceptResult{
			Accepted:         []coverage.AcceptedTarget{{TargetID: "tn", SuggestionID: "s1", Owner: "vrooli", Name: "plans", SourceKind: sources.KindFilesystem, Locator: "/p"}},
			SkippedSensitive: []coverage.Suggestion{{ID: "s2", Owner: "codex", Name: "auth", Sensitive: true}},
			Failed:           []coverage.AcceptFailure{{SuggestionID: "s3", Owner: "x", Name: "y", Message: "boom"}},
			DryRun:           true,
		}}
		h := NewConnectHandler(Deps{Service: svc})

		resp, err := h.AcceptDefaultTargets(ctx, connect.NewRequest(&coveragev1.AcceptDefaultTargetsRequest{
			IncludeSensitive: true,
			DryRun:           true,
		}))
		if err != nil {
			t.Fatalf("AcceptDefaultTargets: %v", err)
		}
		if len(svc.acceptOpts) != 1 || !svc.acceptOpts[0].IncludeSensitive || !svc.acceptOpts[0].DryRun {
			t.Fatalf("options not passed: %+v", svc.acceptOpts)
		}
		msg := resp.Msg
		if len(msg.Accepted) != 1 || msg.Accepted[0].TargetId != "tn" {
			t.Fatalf("accepted not mapped: %+v", msg.Accepted)
		}
		if len(msg.SkippedSensitive) != 1 || len(msg.Failed) != 1 || !msg.DryRun {
			t.Fatalf("accept result not mapped: %+v", msg)
		}
	})
}
