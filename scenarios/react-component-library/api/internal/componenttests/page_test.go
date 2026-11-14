package componenttests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/components"
)

func TestPageUsesSharedVerdictsAndNeverReusesMutableState(t *testing.T) {
	contract := components.StoryContract{SchemaVersion: 5, Kind: components.StoryKindPage, Route: "/coverage", Stories: []components.StoryDefinition{{ID: "ready", Name: "Ready", Role: "anatomy", Args: json.RawMessage(`{}`), Expect: []components.StoryExpectation{{Kind: "text", Value: "Coverage"}}}}}
	runner := Runner{Pages: func(context.Context, string) (components.PageStorySubject, error) {
		return components.PageStorySubject{Contract: &contract, Revision: "source-revision"}, nil
	}, Executor: executor{}}
	request := Request{ComponentID: "page:CoveragePage", Version: "workspace"}
	passed, err := runner.Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, VerdictPassed, passed.Verdict)
	require.NotEmpty(t, passed.Artifacts)
	_, reusable := runner.ExpectedReport(context.Background(), request)
	require.False(t, reusable)
	runner.Executor = executor{results: map[string]StoryExecution{"page:CoveragePage@workspace:ready": {Passed: false, Failures: []string{"Coverage is missing"}}}}
	failed, err := runner.Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, failed.Verdict)
	require.NotEqual(t, passed.ID, failed.ID)
	request.Version = "1.0.0"
	_, err = runner.Run(context.Background(), request)
	require.ErrorContains(t, err, "workspace")
}
