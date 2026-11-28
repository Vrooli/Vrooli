package providers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const batchKey = "resource-ollaya gateway decide-batch --role decide.local --input-stdin --json --timeout 30s --concurrency 4"

func batchRequest() ExecutionRequest {
	return ExecutionRequest{Kind: sharedv1.RequestKind_REQUEST_KIND_DECISION, Role: "decide.local", Timeout: 30 * time.Second}
}

// One command carries the question set once and every state in order, and its
// deadline covers the first (loading) state plus the rest in waves.
func TestExecuteDecisionBatchSendsOneCommandForEveryState(t *testing.T) {
	runner := &fakeRunner{results: map[string]Result{batchKey: {Stdout: `{"results":[
		{"result":{"model":"laya:en","latency_ms":7,"answers":{"q":{"type":"noul","noul":0.9}}}},
		{"error":{"code":"context_overflow","message":"state does not fit laya's context"}},
		{"error":{"code":"timeout","message":"context deadline exceeded"}},
		{"error":{"code":"something_new","message":"?"}},
		{"result":{"model":"laya:en","latency_ms":5,"answers":{"q":{"type":"noul","noul":0.1}}}}
	]}`}}}
	adapter := Adapter{Provider: ProviderOllaya, CommandName: "resource-ollaya", Locality: "local", Runner: runner}
	states := []json.RawMessage{json.RawMessage(`"one"`), json.RawMessage(`{"two":2}`), json.RawMessage(`"three"`), json.RawMessage(`"four"`), json.RawMessage(`"five"`)}

	items, err := adapter.ExecuteDecisionBatch(context.Background(), batchRequest(), `{"q":{"type":"noul"}}`, states)
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.Commands) != 1 {
		t.Fatalf("commands = %d, want one for the whole batch", len(runner.Commands))
	}
	command := runner.Commands[0]
	if command.Stdin != `{"questions":{"q":{"type":"noul"}},"states":["one",{"two":2},"three","four","five"]}` {
		t.Fatalf("stdin = %s", command.Stdin)
	}
	// 5 states: the first alone, then 4 in one wave of DecisionBatchConcurrency.
	if want := 2*30*time.Second + resourceCommandGrace; command.Timeout != want {
		t.Fatalf("runner timeout = %s, want %s", command.Timeout, want)
	}
	if items[0].Err != nil || items[0].LatencyMS != 7 || !strings.Contains(items[0].OutputText, `"noul":0.9`) {
		t.Fatalf("item 0 = %+v", items[0])
	}
	wantCodes := map[int]string{1: CodeContextOverflow, 2: "timeout", 3: CodeProviderFailed}
	for index, code := range wantCodes {
		var cmdErr *CommandError
		if !errors.As(items[index].Err, &cmdErr) || cmdErr.Code != code {
			t.Fatalf("item %d err = %v, want code %s", index, items[index].Err, code)
		}
	}
	if items[4].Err != nil || items[4].LatencyMS != 5 {
		t.Fatalf("item 4 = %+v", items[4])
	}
}

func TestExecuteDecisionBatchRefusesAnAnswerThatDoesNotCoverEveryState(t *testing.T) {
	runner := &fakeRunner{results: map[string]Result{batchKey: {Stdout: `{"results":[{"result":{"answers":{}}}]}`}}}
	adapter := Adapter{Provider: ProviderOllaya, CommandName: "resource-ollaya", Locality: "local", Runner: runner}
	_, err := adapter.ExecuteDecisionBatch(context.Background(), batchRequest(), `{"q":{"type":"noul"}}`, []json.RawMessage{json.RawMessage(`"a"`), json.RawMessage(`"b"`)})
	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) || cmdErr.Code != CodeProviderFailed || !strings.Contains(err.Error(), "1 of 2") {
		t.Fatalf("err = %v, want a provider failure naming the missing states", err)
	}
}

func TestExecuteDecisionBatchCarriesAWholeBatchRefusal(t *testing.T) {
	runner := &fakeRunner{errors: map[string]error{batchKey: &CommandError{Code: CodeInsufficientVRAM, ExitCode: 1}}}
	adapter := Adapter{Provider: ProviderOllaya, CommandName: "resource-ollaya", Locality: "local", Runner: runner}
	_, err := adapter.ExecuteDecisionBatch(context.Background(), batchRequest(), `{"q":{"type":"noul"}}`, []json.RawMessage{json.RawMessage(`"a"`)})
	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) || cmdErr.Code != CodeInsufficientVRAM {
		t.Fatalf("err = %v, want the resource's typed refusal", err)
	}
}

func TestExecuteDecisionBatchIsOnlyServedByADecisionResource(t *testing.T) {
	runner := &fakeRunner{}
	adapter := Adapter{Provider: ProviderOllama, CommandName: "resource-ollama", Locality: "local", Runner: runner}
	_, err := adapter.ExecuteDecisionBatch(context.Background(), batchRequest(), `{}`, []json.RawMessage{json.RawMessage(`"a"`)})
	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) || cmdErr.Code != CodeUnconfigured || len(runner.Commands) != 0 {
		t.Fatalf("err = %v commands = %d, want an unconfigured refusal before any command", err, len(runner.Commands))
	}
}
