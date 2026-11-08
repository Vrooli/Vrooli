package system

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

const kernelMemoryFixture = `{"source":"linux:/proc/slabinfo","allocated_bytes":12288,"cache_count":2,"caches":[{"name":"large-cache","allocated_bytes":8192,"reclaimable":false},{"name":"small-cache","allocated_bytes":4096}]}`

// [REQ:SYSTEM-MEMORY-001] [REQ:HEAL-ACTION-001]
func TestMemorySummaryUsesSetupManagedKernelDiagnostic(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.DefaultResponse.Error = errors.New("private host command forbidden")
	executor.Responses["vrooli host kernel-memory --json"] = testutil.MockResponse{Output: []byte(kernelMemoryFixture)}
	c := NewMemoryCheck(WithMemoryExecutor(executor), WithMemoryProcReader(&mockProcReader{
		memInfo: &checks.MemInfo{MemTotal: 65536, MemAvailable: 32768, SwapTotal: 4096, SwapFree: 2048},
	}))
	result := c.ExecuteAction(context.Background(), "memory-summary")
	if !result.Success || result.Error != "" {
		t.Fatalf("native diagnostics failed: %+v", result)
	}
	for _, want := range []string{"MemTotal: 65536 kB", "MemAvailable: 32768 kB", "large-cache", "reclaimable=false", "small-cache", "reclaimable=unknown"} {
		if !strings.Contains(result.Output, want) {
			t.Errorf("summary missing %q: %s", want, result.Output)
		}
	}
	if len(executor.Calls) != 1 || testutil.CommandKey(executor.Calls[0].Name, executor.Calls[0].Args) != "vrooli host kernel-memory --json" {
		t.Fatalf("diagnostic bypassed native owner: %+v", executor.Calls)
	}
}

// [REQ:SYSTEM-MEMORY-001] Missing or malformed diagnostics must not become success.
func TestMemorySummaryReportsIncompleteDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name       string
		kernel     testutil.MockResponse
		memory     *checks.MemInfo
		memoryErr  error
		wantError  string
		wantOutput string
	}{
		{"broker unavailable", testutil.MockResponse{Error: errors.New("needs_setup")}, &checks.MemInfo{MemTotal: 65536}, nil, "needs_setup", "MemTotal: 65536 kB"},
		{"malformed kernel report", testutil.MockResponse{Output: []byte("not json")}, &checks.MemInfo{MemTotal: 65536}, nil, "kernel", "MemTotal: 65536 kB"},
		{"empty kernel report", testutil.MockResponse{Output: []byte(`{}`)}, &checks.MemInfo{MemTotal: 65536}, nil, "kernel", "MemTotal: 65536 kB"},
		{"host inventory unavailable", testutil.MockResponse{Output: []byte(kernelMemoryFixture)}, nil, errors.New("inventory unavailable"), "inventory unavailable", "large-cache"},
		{"empty host inventory", testutil.MockResponse{Output: []byte(kernelMemoryFixture)}, &checks.MemInfo{}, nil, "memory", "large-cache"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			executor := testutil.NewMockExecutor()
			executor.Responses["vrooli host kernel-memory --json"] = tc.kernel
			c := NewMemoryCheck(WithMemoryExecutor(executor), WithMemoryProcReader(&mockProcReader{memInfo: tc.memory, memInfoErr: tc.memoryErr}))
			result := c.ExecuteAction(context.Background(), "memory-summary")
			if result.Success || !strings.Contains(result.Error, tc.wantError) || !strings.Contains(result.Output, tc.wantOutput) {
				t.Fatalf("incomplete diagnostic lost error or useful evidence: %+v", result)
			}
		})
	}
}
