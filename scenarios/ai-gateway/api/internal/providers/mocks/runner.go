package mocks

import (
	"context"
	"sync"

	"ai-gateway/internal/providers"
)

type FakeRunner struct {
	mu        sync.Mutex
	Results   map[string]providers.Result
	Errors    map[string]error
	Commands  []providers.Command
	Default   providers.Result
	DefaultOK bool
}

var _ providers.CommandRunner = (*FakeRunner)(nil)

func (f *FakeRunner) Run(_ context.Context, command providers.Command) (providers.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Commands = append(f.Commands, command)
	key := command.String()
	if err, ok := f.Errors[key]; ok {
		return f.Results[key], err
	}
	if result, ok := f.Results[key]; ok {
		return result, nil
	}
	if f.DefaultOK {
		return f.Default, nil
	}
	return providers.Result{}, &providers.CommandError{Code: "missing_fixture", Command: key, ExitCode: -1}
}
