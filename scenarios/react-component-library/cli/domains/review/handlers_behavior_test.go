package review

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review/review_v1connect"
)

func TestReviewFindingCommandsForwardFiltersAndRenderResults(t *testing.T) {
	var listed *reviewv1.ListFindingsRequest
	var fetched *reviewv1.GetFindingRequest
	mux := http.NewServeMux()
	mux.Handle(reviewconnect.ReviewServiceListFindingsProcedure, connect.NewUnaryHandlerSimple(reviewconnect.ReviewServiceListFindingsProcedure,
		func(_ context.Context, req *reviewv1.ListFindingsRequest) (*reviewv1.ListFindingsResponse, error) {
			listed = req
			return &reviewv1.ListFindingsResponse{Total: 4, Findings: []*reviewv1.Finding{{Id: "f-1", Severity: reviewv1.FindingSeverity_FINDING_SEVERITY_BLOCKER, Status: reviewv1.FindingStatus_FINDING_STATUS_OPEN, SubjectKind: "component", SubjectId: "button", Message: "contrast", Gate: "visual", Owner: "design", Remediation: "raise contrast"}}}, nil
		}))
	mux.Handle(reviewconnect.ReviewServiceGetFindingProcedure, connect.NewUnaryHandlerSimple(reviewconnect.ReviewServiceGetFindingProcedure,
		func(_ context.Context, req *reviewv1.GetFindingRequest) (*reviewv1.GetFindingResponse, error) {
			fetched = req
			return &reviewv1.GetFindingResponse{Finding: &reviewv1.Finding{Id: req.GetId(), Severity: reviewv1.FindingSeverity_FINDING_SEVERITY_LOW, Status: reviewv1.FindingStatus_FINDING_STATUS_RETIRED, Dispositions: []*reviewv1.FindingDisposition{{Id: "d-1", Decision: reviewv1.DispositionDecision_DISPOSITION_DECISION_RETIRE, Reason: "obsolete"}}}}, nil
		}))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	out, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "findings"), core, "--status", "finding_status-open", "--severity", "BLOCKER", "--subject-kind", "component", "--query", "button", "--limit", "5", "--offset", "2")
	require.NoError(t, err)
	require.Equal(t, reviewv1.FindingStatus_FINDING_STATUS_OPEN, listed.GetStatus())
	require.Equal(t, reviewv1.FindingSeverity_FINDING_SEVERITY_BLOCKER, listed.GetSeverity())
	require.Equal(t, "component", listed.GetSubjectKind())
	require.Equal(t, "button", listed.GetQuery())
	require.Equal(t, int32(5), listed.GetLimit())
	require.Equal(t, int32(2), listed.GetOffset())
	require.Contains(t, out, "Found 1 finding(s) of 4")
	require.Contains(t, out, "f-1 [blocker/open] component/button — contrast gate=visual owner=design | remediation: raise contrast")

	out, err = cliapptest.RunCommand(t, cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "finding"), core, "f-1")
	require.NoError(t, err)
	require.Equal(t, "f-1", fetched.GetId())
	require.Contains(t, out, "disposition d-1 decision=retire actor=(unattributed) at=(unknown) reason=obsolete")
	_, err = cliapptest.RunCommand(t, cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "findings"), core, "--status", "bogus")
	require.ErrorContains(t, err, "--status must be one of")
	_, err = cliapptest.RunCommand(t, cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "findings"), core, "--severity", "bogus")
	require.ErrorContains(t, err, "--severity must be one of")
	_, err = cliapptest.RunCommand(t, cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "findings"), core, "--offset", "-1")
	require.ErrorContains(t, err, "--offset must be a non-negative integer")
}

func TestReviewAssignmentSummaryAndMissingOwner(t *testing.T) {
	var assigned *reviewv1.AssignFindingRequest
	mux := http.NewServeMux()
	mux.Handle(reviewconnect.ReviewServiceAssignFindingProcedure, connect.NewUnaryHandlerSimple(reviewconnect.ReviewServiceAssignFindingProcedure, func(_ context.Context, req *reviewv1.AssignFindingRequest) (*reviewv1.AssignFindingResponse, error) {
		assigned = req
		return &reviewv1.AssignFindingResponse{Finding: &reviewv1.Finding{Id: req.GetId(), Owner: req.GetOwner()}}, nil
	}))
	mux.Handle(reviewconnect.ReviewServiceGetReviewSummaryProcedure, connect.NewUnaryHandlerSimple(reviewconnect.ReviewServiceGetReviewSummaryProcedure, func(context.Context, *reviewv1.GetReviewSummaryRequest) (*reviewv1.GetReviewSummaryResponse, error) {
		return &reviewv1.GetReviewSummaryResponse{Summary: &reviewv1.ReviewSummary{Open: 2, Blockers: 1, High: 1, Acknowledged: 3, Accepted: 4, Fixed: 5}}, nil
	}))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	assign := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "assign")
	out, err := cliapptest.RunCommand(t, assign, core, "f-2", "--owner", "  team-a  ")
	require.NoError(t, err)
	require.Equal(t, "f-2", assigned.GetId())
	require.Equal(t, "team-a", assigned.GetOwner())
	require.Contains(t, out, "Assigned finding f-2 to team-a.")
	_, err = cliapptest.RunCommand(t, assign, core, "f-2", "--owner", " ")
	require.ErrorContains(t, err, "--owner is required")
	out, err = cliapptest.RunCommand(t, cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "summary"), core)
	require.NoError(t, err)
	require.Contains(t, out, "Review queue: 2 open, 1 blocker(s), 1 high, 3 acknowledged, 4 accepted, 5 fixed.")
}

func TestReviewFormatParsersAndInt32EdgeCases(t *testing.T) {
	require.Equal(t, "(nil finding)", formatFinding(nil))
	require.Contains(t, formatFinding(&reviewv1.Finding{}), "subject — (no message)")
	for _, raw := range []string{"", "ANY", "finding_status-fixed", "retired"} {
		_, err := parseFindingStatus(raw)
		require.NoError(t, err, raw)
	}
	for _, raw := range []string{"", "finding_severity-info", "high"} {
		_, err := parseFindingSeverity(raw)
		require.NoError(t, err, raw)
	}
	for _, raw := range []string{"", "accept", "disposition_decision-fix", "retire"} {
		_, err := parseDispositionDecision(raw)
		require.NoError(t, err, raw)
	}
	for _, raw := range []string{"bad", "-1", "2147483648"} {
		_, err := parseNonNegativeInt32(raw, "--limit")
		require.Error(t, err, raw)
	}
	require.Equal(t, "unspecified", severityLabel(0))
	require.Equal(t, "unspecified", statusLabel(0))
	require.Equal(t, "unspecified", decisionLabel(0))
}
