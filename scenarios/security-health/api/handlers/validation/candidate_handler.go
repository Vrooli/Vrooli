package validation

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	internalvalidation "security-health/internal/validation"

	securityhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1"
	securityhealthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/securityhealthv1connect"
)

type candidateScanner interface {
	ScanCandidate(context.Context, map[string][]byte) ([]internalvalidation.Finding, error)
}

type candidateConnectHandler struct {
	scanner candidateScanner
}

func newCandidateConnectHandler(scanner candidateScanner) *candidateConnectHandler {
	return &candidateConnectHandler{scanner: scanner}
}

func (h *candidateConnectHandler) ScanCandidate(ctx context.Context, req *connect.Request[securityhealthv1.ScanCandidateRequest]) (*connect.Response[securityhealthv1.ScanCandidateResponse], error) {
	if h.scanner == nil {
		return connect.NewResponse(&securityhealthv1.ScanCandidateResponse{Complete: false, FailureCode: "scanner_unavailable"}), nil
	}
	if len(req.Msg.GetFiles()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	findings, err := h.scanner.ScanCandidate(ctx, req.Msg.GetFiles())
	if err != nil {
		// Scanner diagnostics can include candidate content. Return only a stable
		// failure category; callers must treat this as incomplete coverage.
		failureCode := "scanner_failed"
		if errors.Is(err, internalvalidation.ErrCandidateScannerUnavailable) {
			failureCode = "scanner_unavailable"
		}
		return connect.NewResponse(&securityhealthv1.ScanCandidateResponse{Complete: false, FailureCode: failureCode}), nil
	}
	resp := &securityhealthv1.ScanCandidateResponse{Complete: true}
	for _, finding := range findings {
		resp.Findings = append(resp.Findings, &securityhealthv1.CandidateFinding{
			RuleId: finding.RuleID, Severity: finding.Severity.String(), Title: finding.Title,
			Description: finding.Description, Remediation: finding.Remediation,
			FilePath: finding.FilePath, Scanner: finding.Scanner,
		})
	}
	return connect.NewResponse(resp), nil
}

var _ securityhealthconnect.CandidateScannerServiceHandler = (*candidateConnectHandler)(nil)
