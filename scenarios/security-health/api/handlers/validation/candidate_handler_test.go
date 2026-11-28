package validation

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"connectrpc.com/connect"

	securityhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1"
	internalvalidation "security-health/internal/validation"
)

type candidateScannerStub struct {
	files    map[string][]byte
	findings []internalvalidation.Finding
	err      error
}

func (s *candidateScannerStub) ScanCandidate(_ context.Context, files map[string][]byte) ([]internalvalidation.Finding, error) {
	s.files = files
	return s.findings, s.err
}

func TestCandidateScanReturnsNormalizedFindingAndExactCandidate(t *testing.T) {
	files := map[string][]byte{"dir/file.env": []byte("opaque-secret-value")}
	scanner := &candidateScannerStub{findings: []internalvalidation.Finding{{
		RuleID: "gitleaks.test", Severity: internalvalidation.SeverityError,
		Title: "Potential secret detected", Description: "matched value is redacted",
		FilePath: "dir/file.env:1", Scanner: "gitleaks",
	}}}
	h := newCandidateConnectHandler(scanner)
	resp, err := h.ScanCandidate(context.Background(), connect.NewRequest(&securityhealthv1.ScanCandidateRequest{Files: files}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scanner.files, files) {
		t.Fatalf("scanner input = %#v, want exact request files %#v", scanner.files, files)
	}
	if !resp.Msg.GetComplete() || len(resp.Msg.GetFindings()) != 1 {
		t.Fatalf("response = %+v", resp.Msg)
	}
	got := resp.Msg.GetFindings()[0]
	if got.GetRuleId() != "gitleaks.test" || got.GetSeverity() != "error" || got.GetFilePath() != "dir/file.env:1" {
		t.Fatalf("finding = %+v", got)
	}
	if got.GetDescription() == "opaque-secret-value" {
		t.Fatal("raw candidate content leaked into finding")
	}
}

func TestCandidateScanFailureIsIncompleteAndRedacted(t *testing.T) {
	h := newCandidateConnectHandler(&candidateScannerStub{err: errors.New("scanner output included opaque-secret-value")})
	resp, err := h.ScanCandidate(context.Background(), connect.NewRequest(&securityhealthv1.ScanCandidateRequest{Files: map[string][]byte{"file.env": []byte("opaque-secret-value")}}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetComplete() || resp.Msg.GetFailureCode() != "scanner_failed" || len(resp.Msg.GetFindings()) != 0 {
		t.Fatalf("failure response = %+v", resp.Msg)
	}
}

func TestCandidateScanReportsMissingScannerAsUnavailable(t *testing.T) {
	h := newCandidateConnectHandler(&candidateScannerStub{err: internalvalidation.ErrCandidateScannerUnavailable})
	resp, err := h.ScanCandidate(context.Background(), connect.NewRequest(&securityhealthv1.ScanCandidateRequest{Files: map[string][]byte{"file.env": []byte("candidate")}}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetComplete() || resp.Msg.GetFailureCode() != "scanner_unavailable" {
		t.Fatalf("response = %+v", resp.Msg)
	}
}

func TestCandidateScanRejectsEmptyRequest(t *testing.T) {
	h := newCandidateConnectHandler(&candidateScannerStub{})
	_, err := h.ScanCandidate(context.Background(), connect.NewRequest(&securityhealthv1.ScanCandidateRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error = %v, want invalid argument", err)
	}
}
