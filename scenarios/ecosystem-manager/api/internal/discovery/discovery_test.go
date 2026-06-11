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
			{output: []byte(`{"resources":[{"name":"redis","path":"/r/redis","exists":true,"registered":true,"enabled":true}]}`)},
		},
	}

	resources, err := discoverResources(runner)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if len(resources) != 1 || resources[0].Name != "redis" || resources[0].Path != "/r/redis" || !resources[0].Healthy || resources[0].Status != "enabled" {
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

func TestDiscoverResourcesParsesWrappedObjectAndDerivesStatus(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"resources":[
				{"name":"redis","exists":true,"registered":true,"enabled":true},
				{"name":"vault","exists":true,"registered":true,"enabled":false},
				{"name":"ollama","exists":true,"registered":false,"enabled":false},
				{"name":"gone","exists":false,"registered":false,"enabled":false}
			]}`)},
		},
	}

	resources, err := discoverResources(runner)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(resources) != 4 {
		t.Fatalf("expected 4 resources, got %d: %+v", len(resources), resources)
	}
	// Sorted alphabetically: gone, ollama, redis, vault.
	want := map[string]string{
		"redis":  "enabled",
		"vault":  "disabled",
		"ollama": "[UNREGISTERED]",
		"gone":   "[MISSING]",
	}
	for _, r := range resources {
		if got := r.Status; got != want[r.Name] {
			t.Errorf("resource %q: status = %q, want %q", r.Name, got, want[r.Name])
		}
	}
}

func TestUnmarshalResourceListAcceptsBareArray(t *testing.T) {
	got, err := unmarshalResourceList([]byte(`[{"name":"redis"}]`))
	if err != nil {
		t.Fatalf("bare array should parse: %v", err)
	}
	if len(got) != 1 || got[0]["name"] != "redis" {
		t.Fatalf("unexpected parse: %+v", got)
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
