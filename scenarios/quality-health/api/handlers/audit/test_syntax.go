package audit

import (
	"context"
	"errors"
	"path/filepath"

	"connectrpc.com/connect"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	"quality-health/internal/testsyntax"
)

type SyntaxObserver interface {
	Observe(context.Context, string, []string) ([]testsyntax.Observation, error)
}

func (h *Handler) ObserveTestSyntax(ctx context.Context, req *connect.Request[auditv1.ObserveTestSyntaxRequest]) (*connect.Response[auditv1.ObserveTestSyntaxResponse], error) {
	if !filepath.IsAbs(req.Msg.GetRootPath()) || len(req.Msg.GetFiles()) == 0 || len(req.Msg.GetFiles()) > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("absolute workspace root and 1..100 test files required"))
	}
	result := &auditv1.ObserveTestSyntaxResponse{SchemaVersion: "vitest-lint/v1"}
	if h.syntax == nil {
		result.UnavailableReason = "owner-unavailable"
		return connect.NewResponse(result), nil
	}
	rows, err := h.syntax.Observe(ctx, req.Msg.GetRootPath(), req.Msg.GetFiles())
	if err != nil {
		h.logger.Printf("test syntax observation unavailable: %v", err)
		result.UnavailableReason = "owner-unavailable"
		return connect.NewResponse(result), nil
	}
	for _, row := range rows {
		out := &auditv1.TestSyntaxObservation{SchemaVersion: row.Schema, File: row.File, SourceDigest: row.Digest, Profile: row.Profile, PluginVersion: row.PluginVersion, EslintVersion: row.ESLintVersion, ParserVersion: row.ParserVersion, Status: row.Status, Reason: row.Reason, Limitations: append([]string(nil), row.Limitations...)}
		for _, check := range row.Checks {
			out.Checks = append(out.Checks, &auditv1.TestSyntaxCheck{Rule: check.Rule, Status: check.Status, Reason: check.Reason})
		}
		out.ConfigDigest = row.ConfigDigest
		for _, d := range row.Diagnostics {
			out.Diagnostics = append(out.Diagnostics, &auditv1.TestSyntaxDiagnostic{NativeRuleId: d.RuleID, CanonicalRule: d.CanonicalRule, MessageId: d.MessageID, Message: d.Message, Line: int32(d.Line), Column: int32(d.Column), EndLine: int32(d.EndLine), EndColumn: int32(d.EndColumn), NativeSeverity: int32(d.Severity)})
		}
		result.Observations = append(result.Observations, out)
	}
	return connect.NewResponse(result), nil
}
