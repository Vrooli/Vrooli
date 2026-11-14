package runsignal

import (
	"reflect"
	"testing"

	"agent-manager/internal/domain"
)

// A call's pipelines are its commands as the shell runs them, with the
// commands their output is piped through and whether a loop re-runs them.
func TestCallPipelinesReadEveryCommandShape(t *testing.T) {
	words := func(ps []Pipeline) [][]string {
		var out [][]string
		for _, p := range ps {
			out = append(out, p.Words)
		}
		return out
	}
	bash := CallPipelines(&domain.ToolCallEventData{ToolName: "Bash", Input: map[string]any{"command": `cd scenarios/x && FOO=1 timeout 30 vrooli locks --json | jq '.claims[]' | head -5; sleep 2`}})
	if got := words(bash); !reflect.DeepEqual(got, [][]string{{"cd", "scenarios/x"}, {"vrooli", "locks", "--json"}, {"sleep", "2"}}) {
		t.Errorf("bash pipelines = %v", got)
	}
	if got := bash[1].Filters; !reflect.DeepEqual(got, [][]string{{"jq", ".claims[]"}, {"head", "-5"}}) {
		t.Errorf("filters = %v", got)
	}
	wrapped := CallPipelines(&domain.ToolCallEventData{ToolName: "exec_command", Input: map[string]any{"cmd": `bash -lc 'until device-control session list --json | grep -q ready; do sleep 5; done'`}})
	if len(wrapped) == 0 || wrapped[0].Words[0] != "device-control" || !wrapped[0].Loop {
		t.Errorf("until loop = %+v", wrapped)
	}
	script := CallPipelines(&domain.ToolCallEventData{ToolName: "exec", Input: map[string]any{"input": `const r = await tools.exec_command({cmd: "test-genie runs wait --json a 1"}); text(r.output)`}})
	if got := words(script); !reflect.DeepEqual(got, [][]string{{"test-genie", "runs", "wait", "--json", "a", "1"}}) {
		t.Errorf("exec script pipelines = %v", got)
	}
	argv := ArgvPipelines([]string{"/bin/bash", "-lc", "agent-manager run list"})
	if got := words(argv); !reflect.DeepEqual(got, [][]string{{"agent-manager", "run", "list"}}) {
		t.Errorf("argv pipelines = %v", got)
	}
}

// A fact's pipeline is the n-th whose words carry its command path in order.
func TestPipelineForPicksTheFactsOwnOccurrence(t *testing.T) {
	ps := CallPipelines(&domain.ToolCallEventData{ToolName: "Bash", Input: map[string]any{"command": "vrooli scenario status a; vrooli scenario logs a --tail 5; vrooli scenario status b"}})
	second, ok := PipelineFor(ps, "vrooli scenario status", 1)
	if !ok || second.Words[3] != "b" {
		t.Fatalf("second status = %+v, %v", second, ok)
	}
	if _, ok := PipelineFor(ps, "vrooli scenario status", 2); ok {
		t.Error("there is no third status call")
	}
}
