package discovery

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type stubResponse struct {
	output []byte
	err    error
	delay  time.Duration
}

type stubRunner struct {
	responses []stubResponse
	calls     []struct {
		name string
		args []string
	}
	idx int
}

func (s *stubRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	s.calls = append(s.calls, struct {
		name string
		args []string
	}{name: name, args: append([]string(nil), args...)})

	if s.idx >= len(s.responses) {
		return nil, errors.New("no stub response")
	}

	resp := s.responses[s.idx]
	s.idx++

	if resp.delay > 0 {
		select {
		case <-time.After(resp.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if resp.err != nil {
		return nil, resp.err
	}

	return resp.output, nil
}

func TestDiscoverResourcesFallsBackOnVerboseFailure(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{err: errors.New("boom")},
			{output: []byte(`[{"Name":"postgres","Status":"running","Description":"db","Running":true,"Port":5432,"Version":"14"}]`)},
		},
	}

	resources, err := discoverResources(runner)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if len(resources) != 1 || resources[0].Name != "postgres" || resources[0].Port != 5432 || !resources[0].Healthy {
		t.Fatalf("unexpected resources parsed: %+v", resources)
	}

	wantCalls := [][]string{
		{"vrooli", "resource", "list", "--json", "--verbose"},
		{"vrooli", "resource", "list", "--json"},
	}

	if len(runner.calls) != len(wantCalls) {
		t.Fatalf("expected %d runner calls, got %d", len(wantCalls), len(runner.calls))
	}

	for i, call := range runner.calls {
		if call.name != "vrooli" || !reflect.DeepEqual(call.args, wantCalls[i][1:]) {
			t.Fatalf("call %d mismatch: got %s %v, want %v", i, call.name, call.args, wantCalls[i])
		}
	}
}

func TestDiscoverScenariosHonorsCommandTimeout(t *testing.T) {
	originalTimeout := commandTimeout
	commandTimeout = 10 * time.Millisecond
	defer func() { commandTimeout = originalTimeout }()

	runner := &stubRunner{
		responses: []stubResponse{
			{delay: 50 * time.Millisecond},
		},
	}

	_, err := discoverScenarios(runner)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}
