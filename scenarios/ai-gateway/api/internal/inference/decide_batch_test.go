package inference

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"ai-gateway/internal/providers"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

// sentBatch is what one decide-batch command carried.
type sentBatch struct {
	role      string
	questions json.RawMessage
	states    []string
}

func (w *bakeoff) batches(t *testing.T) []sentBatch {
	t.Helper()
	var out []sentBatch
	for _, command := range w.runner.Commands {
		text := command.String()
		if !strings.Contains(text, "gateway decide-batch --role ") {
			continue
		}
		var body struct {
			Questions json.RawMessage   `json:"questions"`
			States    []json.RawMessage `json:"states"`
		}
		require.NoError(t, json.Unmarshal([]byte(command.Stdin), &body))
		sent := sentBatch{role: strings.Fields(strings.SplitN(text, "--role ", 2)[1])[0], questions: body.Questions}
		for _, state := range body.States {
			var text string
			require.NoError(t, json.Unmarshal(state, &text))
			sent.states = append(sent.states, text)
		}
		out = append(out, sent)
	}
	return out
}

// refusesState is a decision resource that answers every state of a batch for
// role except refused, which it reports as that state's own typed failure.
func refusesState(role, refused, code string) func(providers.Command) (providers.Result, bool) {
	answer := answersEveryState(answeredOutput)
	return func(command providers.Command) (providers.Result, bool) {
		if !strings.Contains(command.String(), "gateway decide-batch --role "+role+" ") {
			return answer(command)
		}
		var body struct {
			States []string `json:"states"`
		}
		if err := json.Unmarshal([]byte(command.Stdin), &body); err != nil {
			return providers.Result{}, false
		}
		items := make([]string, len(body.States))
		for index, state := range body.States {
			if state == refused {
				items[index] = `{"error":{"code":"` + code + `","message":"state does not fit"}}`
				continue
			}
			items[index] = `{"result":` + answeredOutput + `}`
		}
		return providers.Result{Stdout: `{"results":[` + strings.Join(items, ",") + `]}`}, true
	}
}

func batchTemplate() DecisionRequest {
	template := pinned("")
	template.State = ""
	return template
}

func batchStates(states ...string) []DecisionRequest {
	out := make([]DecisionRequest, len(states))
	for index, state := range states {
		out[index] = DecisionRequest{State: state}
	}
	return out
}

// The reported burst: a 30-state DecideBatch became 30 resource processes, one
// per state. A batch now reaches each candidate once with every state still
// unanswered, the question set sent once; a state the candidate refuses falls
// through to the next candidate on its own.
func TestDecideBatchSendsEachCandidateOneCommandForItsPendingStates(t *testing.T) {
	world := newBakeoff(t, "decide.local", "decide.local.kev")
	world.runner.Respond = refusesState("decide.local", "too long", providers.CodeContextOverflow)
	states := []string{"s0", "s1", "too long", "s3", "s4"}

	response := world.service.DecideBatch(context.Background(), batchTemplate(), batchStates(states...))

	sent := world.batches(t)
	require.Len(t, sent, 2, "one command per candidate tried, never one per state")
	require.Equal(t, "decide.local", sent[0].role)
	require.Equal(t, states, sent[0].states)
	require.JSONEq(t, systemOneQuestions(batchTemplate().Questions), string(sent[0].questions))
	require.Equal(t, "decide.local.kev", sent[1].role)
	require.Equal(t, []string{"too long"}, sent[1].states, "only the refused state walks on")

	require.Len(t, response.GetResults(), len(states))
	for index, result := range response.GetResults() {
		require.Nil(t, result.GetError(), "%d: %+v", index, result.GetError())
		require.Len(t, result.GetAnswers(), 2)
		if states[index] == "too long" {
			require.Equal(t, []string{"FAILED:context_overflow", "ANSWERED", "NOT_ATTEMPTED"}, statuses(result))
			require.Equal(t, "decide.local.kev", result.GetResourceRole())
			continue
		}
		require.Equal(t, []string{"ANSWERED", "NOT_ATTEMPTED", "NOT_ATTEMPTED"}, statuses(result))
		require.Equal(t, "decide.local", result.GetResourceRole())
		require.EqualValues(t, 6, result.GetLatencyMs(), "a state's latency is its own round trip, not the batch's")
	}
	require.EqualValues(t, 103*len(states), response.GetUsage().GetInputTokens())
	require.Equal(t, []string{"decide.local", "decide.local", "decide.local.kev", "decide.local.kev"}, world.resolved(),
		"a candidate is qualified (resolved, then re-resolved for the rights recheck) once per batch, not once per state")
}

// A refusal of the whole command (the model cannot be placed) fails the
// candidate for every pending state, and the next candidate takes them all in
// one command.
func TestDecideBatchCarriesAWholeBatchRefusalToTheNextCandidate(t *testing.T) {
	world := newBakeoff(t, "decide.local", "decide.local.kev")
	world.runner.ErrorMatch = func(command providers.Command) error {
		if strings.Contains(command.String(), "gateway decide-batch --role decide.local ") {
			return &providers.CommandError{Code: providers.CodeInsufficientVRAM, Command: command.String(), ExitCode: 1}
		}
		return nil
	}

	response := world.service.DecideBatch(context.Background(), batchTemplate(), batchStates("a", "b", "c"))

	sent := world.batches(t)
	require.Len(t, sent, 2)
	require.Equal(t, []string{"a", "b", "c"}, sent[1].states)
	for _, result := range response.GetResults() {
		require.Nil(t, result.GetError(), "%+v", result.GetError())
		require.Equal(t, []string{"FAILED:insufficient_vram", "ANSWERED", "NOT_ATTEMPTED"}, statuses(result))
	}
}

// Each state keeps its own route event, attributed the candidate that answered
// it, its share of the one command, and its own round trip as provider time.
func TestDecideBatchRecordsOneRouteEventPerState(t *testing.T) {
	world := newBakeoff(t, "decide.local")
	world.runner.Respond = answersEveryState(answeredOutput)
	log, option := recordingOptions()
	service := NewService(world.repository, option)

	response := service.DecideBatch(context.Background(), batchTemplate(), batchStates("a", "b", "c"))
	require.Len(t, response.GetResults(), 3)

	for range 3 {
		ev := log.next(t)
		require.Equal(t, OperationDecide, ev.GetOperation())
		require.Equal(t, "succeeded", ev.GetStatus())
		require.Equal(t, "ollaya", ev.GetSelectedProvider())
		require.Equal(t, "decide.local", ev.GetSelectedResourceRole())
		require.EqualValues(t, 6*time.Millisecond, ev.GetProviderRuntimeDurationNs(), "the state's own round trip")
		require.LessOrEqual(t, ev.GetProviderExecutionDurationNs(), ev.GetTotalDurationNs())
	}
	log.none(t)
}

// A state the walk must refuse (an empty state) is answered alone and never
// reaches a resource; the rest of the batch is unaffected.
func TestDecideBatchRefusesABadStateWithoutDispatchingIt(t *testing.T) {
	world := newBakeoff(t, "decide.local")
	world.runner.Respond = answersEveryState(answeredOutput)

	response := world.service.DecideBatch(context.Background(), batchTemplate(), batchStates("a", "", "c"))

	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.GetResults()[1].GetError().GetCode())
	require.Nil(t, response.GetResults()[0].GetError())
	require.Nil(t, response.GetResults()[2].GetError())
	sent := world.batches(t)
	require.Len(t, sent, 1)
	require.Equal(t, []string{"a", "c"}, sent[0].states)
}
