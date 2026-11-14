package androidadb

import (
	"context"
	"strings"
	"sync"
)

type scriptedRunner struct {
	mu        sync.Mutex
	responses map[string][]byte
	errors    map[string]error
	calls     []string
}

type scriptedProcess struct{ signaled bool }

func (p *scriptedProcess) Start() error { return nil }

func (p *scriptedProcess) Interrupt() error { p.signaled = true; return nil }

func (p *scriptedProcess) Kill() error { p.signaled = true; return nil }

func (p *scriptedProcess) Wait() error { return nil }

type processRunner struct {
	*scriptedRunner
	process *scriptedProcess
}

func (r *processRunner) Start(string, ...string) (Process, error) { return r.process, nil }

func (r *scriptedRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	call := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, call)
	if err := r.errors[call]; err != nil {
		return r.responses[call], err
	}
	return r.responses[call], nil
}

func boolPtr(value bool) *bool { return &value }

func countCalls(runner *scriptedRunner, command string) int {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	n := 0
	for _, call := range runner.calls {
		if call == command {
			n++
		}
	}
	return n
}

func indexOfCall(calls []string, fragment string) int {
	for i, call := range calls {
		if strings.Contains(call, fragment) {
			return i
		}
	}
	return len(calls)
}
