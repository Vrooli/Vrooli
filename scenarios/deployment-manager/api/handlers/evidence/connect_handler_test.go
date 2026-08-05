package evidencehandler

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	internalEvidence "deployment-manager/internal/evidence"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
)

type fakeEvidenceRepository struct {
	verdicts []*commonv1.TargetVerdict
	err      error
}

func (f *fakeEvidenceRepository) Save(_ context.Context, _ string, _ string, verdict *commonv1.TargetVerdict) error {
	if f.err != nil {
		return f.err
	}
	f.verdicts = append(f.verdicts, verdict)
	return nil
}

func (f *fakeEvidenceRepository) List(_ context.Context, _ string, _ string, _ int) ([]*commonv1.TargetVerdict, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.verdicts, nil
}

var _ internalEvidence.Repository = (*fakeEvidenceRepository)(nil)

func validRequest() *connect.Request[evidencev1.ReportTargetVerdictRequest] {
	return connect.NewRequest(&evidencev1.ReportTargetVerdictRequest{
		ProfileId: "profile-1", GitCommitHash: "commit-1",
		Verdict: &commonv1.TargetVerdict{
			Target: &commonv1.EvidenceTarget{
				Ramp: "ramp", Platform: "desktop", Os: "linux",
				DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST,
			},
			Disposition: commonv1.Disposition_DISPOSITION_PASSED, RunId: "run-1",
		},
	})
}

func TestReportTargetVerdictValidatesAndPersists(t *testing.T) {
	repo := &fakeEvidenceRepository{}
	h := NewConnectHandler(repo)
	resp, err := h.ReportTargetVerdict(context.Background(), validRequest())
	if err != nil || resp == nil || len(repo.verdicts) != 1 {
		t.Fatalf("report failed: resp=%v err=%v saved=%d", resp, err, len(repo.verdicts))
	}

	bad := validRequest()
	bad.Msg.Verdict.Target.Ramp = ""
	if _, err := h.ReportTargetVerdict(context.Background(), bad); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
	if _, err := h.ReportTargetVerdict(context.Background(), nil); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected nil request rejection, got %v", err)
	}
	failed := &fakeEvidenceRepository{err: errors.New("store down")}
	if _, err := NewConnectHandler(failed).ReportTargetVerdict(context.Background(), validRequest()); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("expected internal repository error, got %v", err)
	}
}

func TestListAndReview(t *testing.T) {
	passed := validRequest().Msg.Verdict
	repo := &fakeEvidenceRepository{verdicts: []*commonv1.TargetVerdict{passed}}
	h := NewConnectHandler(repo)
	list, err := h.ListTargetVerdicts(context.Background(), connect.NewRequest(&evidencev1.ListTargetVerdictsRequest{ProfileId: "p", GitCommitHash: "c"}))
	if err != nil || len(list.Msg.Verdicts) != 1 {
		t.Fatalf("list failed: %v", err)
	}
	review, err := h.GetEvidenceReview(context.Background(), connect.NewRequest(&evidencev1.GetEvidenceReviewRequest{ProfileId: "p", GitCommitHash: "c"}))
	if err != nil || !review.Msg.Ready {
		t.Fatalf("expected ready review: %v", err)
	}
	passed.Disposition = commonv1.Disposition_DISPOSITION_FAILED
	review, err = h.GetEvidenceReview(context.Background(), connect.NewRequest(&evidencev1.GetEvidenceReviewRequest{ProfileId: "p", GitCommitHash: "c"}))
	if err != nil || review.Msg.Ready || review.Msg.Reason != "one_or_more_targets_not_passed" {
		t.Fatalf("expected failed review: %v", err)
	}
	empty := NewConnectHandler(&fakeEvidenceRepository{})
	review, err = empty.GetEvidenceReview(context.Background(), connect.NewRequest(&evidencev1.GetEvidenceReviewRequest{ProfileId: "p", GitCommitHash: "c"}))
	if err != nil || review.Msg.Reason != "no_target_evidence" {
		t.Fatalf("expected empty review reason: %v", err)
	}
	for _, call := range []func() error{
		func() error { _, err := h.ListTargetVerdicts(context.Background(), nil); return err },
		func() error { _, err := h.GetEvidenceReview(context.Background(), nil); return err },
		func() error {
			_, err := h.ListTargetVerdicts(context.Background(), connect.NewRequest(&evidencev1.ListTargetVerdictsRequest{}))
			return err
		},
	} {
		if connect.CodeOf(call()) != connect.CodeInvalidArgument {
			t.Fatal("expected invalid argument for incomplete request")
		}
	}
}
