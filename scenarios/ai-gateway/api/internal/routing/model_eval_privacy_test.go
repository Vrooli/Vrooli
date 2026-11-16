package routing_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/providers/mocks"
	internalrouting "ai-gateway/internal/routing"
	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type evalOutputRunner struct {
	inner  *mocks.FakeRunner
	output string
}

func (r evalOutputRunner) Run(ctx context.Context, command providers.Command) (providers.Result, error) {
	result, err := r.inner.Run(ctx, command)
	if err == nil && strings.Contains(command.String(), " gateway generate ") {
		result.Stdout = `{"response":"` + r.output + `","eval_count":1}`
	}
	return result, err
}

func TestSuccessfulLocalModelEvalRoutePersistsNoPromptOrOutput(t *testing.T) {
	const promptSentinel = "LM2_PROMPT_PERSISTENCE_SENTINEL"
	const outputSentinel = "LM2_OUTPUT_PERSISTENCE_SENTINEL"
	runner := roleRunner()
	db := newSchemaDB(t)
	service := newService(testAdapters(evalOutputRunner{inner: runner, output: outputSentinel}), internalrouting.NewSQLRepository(db))
	response, err := service.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), promptSentinel)
	require.NoError(t, err)
	require.True(t, response.GetValid())
	require.Contains(t, response.GetOutputText(), outputSentinel, "commands=%+v", runner.Commands)

	rows, err := db.Query(`SELECT * FROM route_events`)
	require.NoError(t, err)
	defer rows.Close()
	columns, err := rows.Columns()
	require.NoError(t, err)
	require.True(t, rows.Next())
	values := make([]any, len(columns))
	dest := make([]any, len(columns))
	for i := range values {
		dest[i] = &values[i]
	}
	require.NoError(t, rows.Scan(dest...))
	require.NoError(t, rows.Close())
	persisted := make(map[string]any, len(columns))
	for i, column := range columns {
		persisted[column] = values[i]
	}
	encoded, err := json.Marshal(persisted)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), promptSentinel)
	require.NotContains(t, string(encoded), outputSentinel)

	listed, err := service.ListEvidence(context.Background(), internalrouting.EvidenceFilter{Limit: 10})
	require.NoError(t, err)
	listedJSON, err := json.Marshal(listed)
	require.NoError(t, err)
	require.NotContains(t, string(listedJSON), promptSentinel)
	require.NotContains(t, string(listedJSON), outputSentinel)
	require.False(t, strings.Contains(runner.Commands[len(runner.Commands)-1].String(), promptSentinel), "prompt must not enter command arguments")
}
