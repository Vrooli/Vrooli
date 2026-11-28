package flows

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"device-control/internal/execution"
	"github.com/stretchr/testify/require"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
)

type desktopFlowFixture struct {
	desktopStepFixture
	commands   []string
	unknown    bool
	cancel     context.CancelFunc
	assertions []*desktopv1.AssertTextAction
}

func (f *desktopFlowFixture) Act(_ context.Context, r *connect.Request[desktopv1.OwnerActRequest]) (*connect.Response[desktopv1.ActResponse], error) {
	f.commands = append(f.commands, r.Msg.CommandId)
	if assertion := r.Msg.Action.GetAssertText(); assertion != nil {
		f.assertions = append(f.assertions, assertion)
	}
	if f.cancel != nil {
		f.cancel()
	}
	outcome := "applied"
	if f.unknown {
		outcome = "outcome_unknown"
	}
	return connect.NewResponse(&desktopv1.ActResponse{Receipt: &desktopv1.Receipt{Outcome: outcome}}), nil
}

func TestDesktopFlowUsesExplicitTextAssertion(t *testing.T) {
	binding := &desktopv1.OwnerObserveRequest{Session: &commonv1.SessionRef{SessionId: "lease"}, ApplicationId: "app", ApplicationRevision: "catalog"}
	flow := execution.Flow{Transport: "desktop", Steps: []execution.Step{
		{ID: "verify", Kind: "desktop-text-assert", Target: "Text", Arguments: map[string]any{"window": "Editor", "text": ""}},
	}}
	f := &desktopFlowFixture{}
	result, err := ExecuteDesktopFlowWithID(context.Background(), f, binding, flow, "checked-run")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, f.assertions, 1)
	require.Empty(t, f.assertions[0].ExpectedText, "empty field is a valid expected outcome")
	require.Equal(t, "f", f.assertions[0].ElementId)
	require.Equal(t, "r", f.assertions[0].ObservationRevision)
	flow.Steps[0].Arguments["position"] = 0
	f = &desktopFlowFixture{}
	_, err = ExecuteDesktopFlow(context.Background(), f, binding, flow)
	require.Error(t, err, "assertion does not accept insertion offsets")
	require.Empty(t, f.commands)
}

func TestDesktopFlowPreflightsAllStepsAndHaltsUnknown(t *testing.T) {
	binding := &desktopv1.OwnerObserveRequest{Session: &commonv1.SessionRef{SessionId: "lease"}, ApplicationId: "app", ApplicationRevision: "catalog"}
	step := execution.Step{ID: "one", Kind: "desktop-text", Target: "Text", Arguments: map[string]any{"window": "Editor", "text": "private text"}}
	second := step
	second.ID = "two"
	flow := execution.Flow{Transport: "desktop", Steps: []execution.Step{step, second}}
	f := &desktopFlowFixture{}
	flow.Steps[1].Kind = "unsupported"
	_, err := ExecuteDesktopFlow(context.Background(), f, binding, flow)
	require.Error(t, err)
	require.Empty(t, f.commands)
	flow.Steps[1].Kind = "desktop-text"
	f.unknown = true
	result, err := ExecuteDesktopFlow(context.Background(), f, binding, flow)
	require.NoError(t, err)
	require.True(t, result.Incomplete)
	require.Len(t, f.commands, 1)
	require.Len(t, result.Chapters, 1)
	require.False(t, strings.Contains(result.Chapters[0].Message, "private text"))
	f = &desktopFlowFixture{}
	result, err = ExecuteDesktopFlowWithID(context.Background(), f, binding, flow, "claimed-run")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, result.Chapters, 2)
	require.Len(t, f.commands, 2)
	require.NotEqual(t, f.commands[0], f.commands[1])
	require.Equal(t, []string{"claimed-run:0", "claimed-run:1"}, f.commands)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f = &desktopFlowFixture{cancel: cancel}
	result, err = ExecuteDesktopFlow(ctx, f, binding, flow)
	require.NoError(t, err)
	require.True(t, result.Incomplete)
	require.Len(t, f.commands, 1)
}
