package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
)

type unusedAutomationFactory struct{}

func (unusedAutomationFactory) Resolve(context.Context, string) (autoengine.AutomationEngine, error) {
	return nil, errors.New("factory should not be resolved by the test executor")
}

type capturingAutomationExecutor struct {
	contexts []context.Context
	requests []autoexecutor.Request
}

func (e *capturingAutomationExecutor) Execute(ctx context.Context, req autoexecutor.Request) error {
	e.contexts = append(e.contexts, ctx)
	e.requests = append(e.requests, req)
	return ctx.Err()
}

func TestDefaultAutomationRunnerRetainsPerCallIdentityCancellationAndMemoryEvidence(t *testing.T) {
	executor := &capturingAutomationExecutor{}
	runner, err := NewDefaultAutomationRunner(logrus.New(),
		WithEngineFactory(unusedAutomationFactory{}),
		WithExecutor(executor),
	)
	require.NoError(t, err)

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("request"), "identity")
	_, _, err = runner.RunWithDeviceScale(ctx, 1024, 768, 2, []autocontracts.CompiledInstruction{{}})
	require.NoError(t, err)
	_, _, err = runner.Run(ctx, 1280, 720, nil)
	require.NoError(t, err)

	require.Len(t, executor.requests, 2)
	first, second := executor.requests[0], executor.requests[1]
	require.NotEqual(t, first.Plan.ExecutionID, second.Plan.ExecutionID)
	require.NotEqual(t, first.Plan.WorkflowID, second.Plan.WorkflowID)
	require.Same(t, ctx, executor.contexts[0])
	require.Same(t, ctx, executor.contexts[1])
	require.NotNil(t, first.Recorder)
	require.NotNil(t, first.EventSink)
	require.NotSame(t, first.Recorder, second.Recorder)
	require.NotSame(t, first.EventSink, second.EventSink)
	require.NotNil(t, first.BrowserProfile)
	require.NotNil(t, first.BrowserProfile.Fingerprint)
	require.Equal(t, 2.0, first.BrowserProfile.Fingerprint.DeviceScaleFactor)
	require.Nil(t, second.BrowserProfile)

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err = runner.Run(cancelled, 800, 600, nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Same(t, cancelled, executor.contexts[2])
	require.Len(t, executor.requests, 3)
}
