package derivation_test

import (
	"context"
	"errors"
	"testing"

	"document-manager/internal/derivation"
	"document-manager/internal/derivation/mocks"
	"github.com/stretchr/testify/require"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

func testRegistry() derivation.Registry {
	return derivation.Registry{Handlers: []derivation.Handler{{ID: "native-text", Version: "1", Formats: []string{"text/plain"}, Capabilities: []string{"content"}, Tier: 1, Runtime: "in-process"}}}
}

func TestDeriveAppendsVersionsAndNormalizesUnits(t *testing.T) { // [REQ:DOC-P0-006] [REQ:DOC-P0-007] [REQ:DOC-P0-008]
	store := &mocks.Store{}
	runner := &mocks.Runner{Model: derivation.Model{Units: []derivation.Unit{{Text: "body", Kind: documentpb.AnchorKind_ANCHOR_KIND_LOGICAL, Confidence: .9}}}}
	service := derivation.NewService(testRegistry(), runner, store)
	first, err := service.Derive(context.Background(), derivation.Input{DocumentHash: "sha256-a", Mime: "text/plain", Bytes: []byte("body"), TierCeiling: documentpb.Tier_TIER_ONE})
	require.NoError(t, err)
	second, err := service.Derive(context.Background(), derivation.Input{DocumentHash: "sha256-a", Mime: "text/plain", Bytes: []byte("body"), TierCeiling: documentpb.Tier_TIER_ONE})
	require.NoError(t, err)
	require.Equal(t, 1, first.Version)
	require.Equal(t, 2, second.Version)
	require.Len(t, store.Results, 2)
	require.Equal(t, documentpb.TerminalState_TERMINAL_STATE_PARSED, first.State)
	require.Equal(t, "native-text@1", first.Handlers[0])
}

func TestDeriveWritesDistinctTerminalStatesAndRemedies(t *testing.T) { // [REQ:DOC-P0-027]
	tests := []struct {
		name            string
		registry        derivation.Registry
		runnerErr       error
		ceiling         documentpb.Tier
		wantState       documentpb.TerminalState
		wantReason      string
		wantRemedy      string
		wantRunnerCalls int
	}{
		{
			name: "no handler for format", registry: derivation.Registry{}, ceiling: documentpb.Tier_TIER_ONE,
			wantState:  documentpb.TerminalState_TERMINAL_STATE_NO_HANDLER_FOR_FORMAT,
			wantReason: "no handler for format: application/x-unknown",
			wantRemedy: "install a handler that declares this MIME type",
		},
		{
			name: "blocked by policy", registry: derivation.Registry{Handlers: []derivation.Handler{{ID: "tier-three", Formats: []string{"text/plain"}, Tier: 3}}}, ceiling: documentpb.Tier_TIER_ONE,
			wantState:  documentpb.TerminalState_TERMINAL_STATE_BLOCKED_BY_POLICY,
			wantReason: "handler blocked by policy: text/plain",
			wantRemedy: "reclassify the document or install a permitted local handler",
		},
		{
			name: "handler unavailable", registry: testRegistry(), runnerErr: &derivation.HandlerError{Unavailable: true, Err: errors.New("resource unavailable")}, ceiling: documentpb.Tier_TIER_ONE,
			wantState:  documentpb.TerminalState_TERMINAL_STATE_HANDLER_UNAVAILABLE,
			wantReason: "resource unavailable", wantRemedy: "start or install the declared handler resource", wantRunnerCalls: 1,
		},
		{
			name: "handler failed", registry: testRegistry(), runnerErr: errors.New("malformed handler response"), ceiling: documentpb.Tier_TIER_ONE,
			wantState:  documentpb.TerminalState_TERMINAL_STATE_HANDLER_FAILED,
			wantReason: "malformed handler response", wantRemedy: "inspect the handler error and retry with another chain", wantRunnerCalls: 1,
		},
		{
			name: "unsupported variant", registry: testRegistry(), runnerErr: &derivation.HandlerError{Variant: true, Err: errors.New("unsupported encrypted variant")}, ceiling: documentpb.Tier_TIER_ONE,
			wantState:  documentpb.TerminalState_TERMINAL_STATE_UNSUPPORTED_VARIANT,
			wantReason: "unsupported encrypted variant", wantRemedy: "remove the protection or provide a supported document variant", wantRunnerCalls: 1,
		},
	}
	seenStates := map[documentpb.TerminalState]bool{}
	seenRemedies := map[string]bool{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &mocks.Store{}
			runner := &mocks.Runner{Err: tc.runnerErr}
			service := derivation.NewService(tc.registry, runner, store)
			mime := "text/plain"
			if tc.wantState == documentpb.TerminalState_TERMINAL_STATE_NO_HANDLER_FOR_FORMAT {
				mime = "application/x-unknown"
			}
			result, err := service.Derive(context.Background(), derivation.Input{DocumentHash: "sha256-terminal", Mime: mime, Bytes: []byte("retained source"), TierCeiling: tc.ceiling})
			require.NoError(t, err)
			require.Equal(t, tc.wantState, result.State)
			require.Equal(t, tc.wantReason, result.Reason)
			require.Equal(t, tc.wantRemedy, result.Remedy)
			require.Equal(t, tc.wantRunnerCalls, runner.Calls)
			require.Len(t, store.Results, 1, "every unsupported outcome is appended to custody")
			require.Equal(t, result, store.Results[0])
			require.Equal(t, 1, result.Version)
			seenStates[result.State] = true
			require.False(t, seenRemedies[result.Remedy], "each terminal state needs a distinct remedy")
			seenRemedies[result.Remedy] = true
		})
	}
	require.Len(t, seenStates, len(tests))
}
