package resources

import (
	"context"
	"io"
	"strings"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// mockStatusFetcher implements ResourceStatusFetcher for testing.
type mockStatusFetcher struct {
	resp *cliv1.ResourceStatusesResponse
	err  error
}

func (m *mockStatusFetcher) ResourceStatuses(ctx context.Context) (*cliv1.ResourceStatusesResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

// status builds one ResourceStatus row. healthy is tri-state: pass a *bool for
// an evaluated probe, or nil for "not probed".
func status(name string, running bool, healthy *bool) *cliv1.ResourceStatus {
	rs := &cliv1.ResourceStatus{
		Resource: &cliv1.Resource{Name: name},
		Running:  running,
	}
	if healthy != nil {
		rs.Healthy = structpb.NewBoolValue(*healthy)
	} else {
		rs.Healthy = structpb.NewNullValue()
	}
	return rs
}

func boolPtr(b bool) *bool { return &b }

func statusesResp(rows ...*cliv1.ResourceStatus) *cliv1.ResourceStatusesResponse {
	return &cliv1.ResourceStatusesResponse{Success: true, Resources: rows}
}

func TestCheckerNoRequiredResourcesSkips(t *testing.T) {
	fetcher := &mockStatusFetcher{resp: statusesResp()}
	result := NewChecker(nil, fetcher, io.Discard).Check(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("expected 1 skip observation, got %d", len(result.Observations))
	}
}

func TestCheckerAllRequiredHealthy(t *testing.T) {
	fetcher := &mockStatusFetcher{resp: statusesResp(
		status("postgres", true, boolPtr(true)),
		status("redis", true, boolPtr(true)),
		status("optional-extra", false, boolPtr(false)), // not required → ignored
	)}

	result := NewChecker([]string{"postgres", "redis"}, fetcher, io.Discard).Check(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) != 2 {
		t.Fatalf("expected 2 success observations, got %d", len(result.Observations))
	}
}

func TestCheckerRequiredNotRunningFails(t *testing.T) {
	fetcher := &mockStatusFetcher{resp: statusesResp(status("postgres", false, boolPtr(false)))}

	result := NewChecker([]string{"postgres"}, fetcher, io.Discard).Check(context.Background())
	if result.Success {
		t.Fatalf("expected failure for a stopped required resource")
	}
	if !strings.Contains(result.Error.Error(), "running=false") {
		t.Errorf("error should mention running=false, got: %v", result.Error)
	}
}

func TestCheckerRequiredRunningButUnhealthyFails(t *testing.T) {
	fetcher := &mockStatusFetcher{resp: statusesResp(status("postgres", true, boolPtr(false)))}

	result := NewChecker([]string{"postgres"}, fetcher, io.Discard).Check(context.Background())
	if result.Success {
		t.Fatalf("expected failure for an unhealthy required resource")
	}
	if !strings.Contains(result.Error.Error(), "healthy=false") {
		t.Errorf("error should mention healthy=false, got: %v", result.Error)
	}
}

func TestCheckerRequiredMissingFails(t *testing.T) {
	fetcher := &mockStatusFetcher{resp: statusesResp(status("redis", true, boolPtr(true)))}

	result := NewChecker([]string{"postgres"}, fetcher, io.Discard).Check(context.Background())
	if result.Success {
		t.Fatalf("expected failure when a required resource is absent from status")
	}
	if !strings.Contains(result.Error.Error(), "postgres (not found)") {
		t.Errorf("error should mention the missing resource, got: %v", result.Error)
	}
}

func TestCheckerRunningWithUnprobedHealthPasses(t *testing.T) {
	// Health not evaluated (null) but running → pass, with a note rather than a
	// false failure on absent probe data.
	fetcher := &mockStatusFetcher{resp: statusesResp(status("postgres", true, nil))}

	result := NewChecker([]string{"postgres"}, fetcher, io.Discard).Check(context.Background())
	if !result.Success {
		t.Fatalf("expected success for running resource with unprobed health, got: %v", result.Error)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("expected an unprobed-health observation, got %+v", result.Observations)
	}
}

func TestCheckerMultipleFailuresListed(t *testing.T) {
	fetcher := &mockStatusFetcher{resp: statusesResp(
		status("postgres", false, boolPtr(false)),
		status("redis", true, boolPtr(false)),
	)}

	result := NewChecker([]string{"postgres", "redis"}, fetcher, io.Discard).Check(context.Background())
	if result.Success {
		t.Fatalf("expected failure")
	}
	for _, want := range []string{"postgres", "redis"} {
		if !strings.Contains(result.Error.Error(), want) {
			t.Errorf("error should mention %s, got: %v", want, result.Error)
		}
	}
}

func TestCheckerFetchErrorIsBlocking(t *testing.T) {
	fetcher := &mockStatusFetcher{err: &mockFetchError{"connection refused"}}

	var log strings.Builder
	result := NewChecker([]string{"postgres"}, fetcher, &log).Check(context.Background())
	if result.Success {
		t.Fatalf("expected blocking failure when resource status cannot be read")
	}
	if result.Error == nil || !strings.Contains(result.Error.Error(), "resource status") {
		t.Errorf("expected resource-status read error, got: %v", result.Error)
	}
	if !strings.Contains(log.String(), "WARNING") {
		t.Errorf("expected a warning log, got: %q", log.String())
	}
}

func TestCheckerFetchErrorWithNilLogWriterDoesNotPanic(t *testing.T) {
	fetcher := &mockStatusFetcher{err: &mockFetchError{"boom"}}
	result := NewChecker([]string{"postgres"}, fetcher, nil).Check(context.Background())
	if result.Success {
		t.Fatalf("expected failure")
	}
}

// Ensure mock satisfies the interface at compile time.
var _ ResourceStatusFetcher = (*mockStatusFetcher)(nil)

type mockFetchError struct {
	msg string
}

func (e *mockFetchError) Error() string {
	return e.msg
}
