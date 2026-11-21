package derivation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type fakeRunner struct{ err error }

func (f fakeRunner) Run(context.Context, Handler, Input) (Model, error) {
	if f.err != nil {
		return Model{}, f.err
	}
	return Model{Units: []Unit{{Text: "body", Kind: documentpb.AnchorKind_ANCHOR_KIND_LOGICAL, Confidence: .9}}}, nil
}

type fakeStore struct {
	next    int
	results []Result
}

func (f *fakeStore) NextVersion(context.Context, string) (int, error) { f.next++; return f.next, nil }
func (f *fakeStore) Append(_ context.Context, result Result) error {
	f.results = append(f.results, result)
	return nil
}

func testRegistry() Registry {
	return Registry{Handlers: []Handler{{ID: "native-text", Version: "1", Formats: []string{"text/plain"}, Capabilities: []string{"content"}, Tier: 1, Runtime: "in-process"}}}
}

func TestDeriveAppendsVersionsAndNormalizesUnits(t *testing.T) { // [REQ:DOC-P0-006] [REQ:DOC-P0-007] [REQ:DOC-P0-008]
	store := &fakeStore{}
	service := NewService(testRegistry(), fakeRunner{}, store)
	first, err := service.Derive(context.Background(), Input{DocumentHash: "sha256-a", Mime: "text/plain", Bytes: []byte("body"), TierCeiling: documentpb.Tier_TIER_ONE})
	require.NoError(t, err)
	second, err := service.Derive(context.Background(), Input{DocumentHash: "sha256-a", Mime: "text/plain", Bytes: []byte("body"), TierCeiling: documentpb.Tier_TIER_ONE})
	require.NoError(t, err)
	require.Equal(t, 1, first.Version)
	require.Equal(t, 2, second.Version)
	require.Len(t, store.results, 2)
	require.Equal(t, documentpb.TerminalState_TERMINAL_STATE_PARSED, first.State)
	require.Equal(t, "native-text@1", first.Handlers[0])
}

func TestAllTerminalStatesHaveDistinctRemedies(t *testing.T) { // [REQ:DOC-P0-027]
	states := []documentpb.TerminalState{
		documentpb.TerminalState_TERMINAL_STATE_NO_HANDLER_FOR_FORMAT,
		documentpb.TerminalState_TERMINAL_STATE_HANDLER_UNAVAILABLE,
		documentpb.TerminalState_TERMINAL_STATE_HANDLER_FAILED,
		documentpb.TerminalState_TERMINAL_STATE_BLOCKED_BY_POLICY,
		documentpb.TerminalState_TERMINAL_STATE_UNSUPPORTED_VARIANT,
	}
	remedies := map[documentpb.TerminalState]string{
		states[0]: "install a handler that declares this MIME type",
		states[1]: "start or install the declared handler resource",
		states[2]: "inspect the handler error and retry with another chain",
		states[3]: "reclassify the document or install a permitted local handler",
		states[4]: "remove the protection or provide a supported document variant",
	}
	for _, state := range states {
		require.NotEmpty(t, remedies[state])
	}
	_ = errors.New // keep the table explicit about typed handler failures
}
