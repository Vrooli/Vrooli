package audit

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	"quality-health/internal/testsyntax"
	"testing"
)

type syntaxStub struct {
	calls int
	err   error
}

func (s *syntaxStub) Observe(context.Context, string, []string) ([]testsyntax.Observation, error) {
	s.calls++
	return []testsyntax.Observation{{Schema: "vitest-lint/v1", File: "a.test.ts", Digest: "digest", PluginVersion: "1.6.9", Checks: []testsyntax.Check{{Rule: "malformed-expectation", Status: "violation", Reason: "none"}}, Diagnostics: []testsyntax.Diagnostic{{RuleID: "vitest/valid-expect", CanonicalRule: "malformed-expectation", Line: 3, Column: 2, Message: "Expect must have a corresponding matcher call"}}}}, s.err
}
func TestSyntaxRPCPreservesNativeEvidenceAndCallsOwnerOnce(t *testing.T) {
	owner := &syntaxStub{}
	handler := NewHandlerWithDeps(Deps{SyntaxObserver: owner})
	resp, err := handler.ObserveTestSyntax(context.Background(), connect.NewRequest(&auditv1.ObserveTestSyntaxRequest{RootPath: t.TempDir(), Files: []string{"a.test.ts"}}))
	require.NoError(t, err)
	require.Equal(t, 1, owner.calls)
	require.Equal(t, "digest", resp.Msg.Observations[0].SourceDigest)
	require.Equal(t, int32(3), resp.Msg.Observations[0].Diagnostics[0].Line)
	require.Equal(t, "violation", resp.Msg.Observations[0].Checks[0].Status)
	owner.err = errors.New("owner unavailable")
	resp, err = handler.ObserveTestSyntax(context.Background(), connect.NewRequest(&auditv1.ObserveTestSyntaxRequest{RootPath: t.TempDir(), Files: []string{"a.test.ts"}}))
	require.NoError(t, err)
	require.Equal(t, "owner-unavailable", resp.Msg.UnavailableReason)
	require.Empty(t, resp.Msg.Observations)
}
