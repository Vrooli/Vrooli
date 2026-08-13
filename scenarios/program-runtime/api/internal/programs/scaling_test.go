package programs

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

func TestKernelAgentBytesStayBoundedAcrossResultSizes(t *testing.T) { // [REQ:PRT-P0-003]
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve scaling test path")
	}
	engine := filepath.Join(filepath.Dir(file), "..", "..", "..", "kernel", "host", "engine.py")
	runner := NewSubprocessRunner(engine)
	defer runner.KillSession("scaling")

	counts := []int{10, 1000, 100000, 10000000}
	values := make([]int64, 0, len(counts))
	for _, count := range counts {
		source := "rows = range(" + formatInt(count) + ")\nprint([value for value in rows if value < 5])"
		result, err := runner.Execute(context.Background(), "scaling", source, false)
		if err != nil {
			t.Fatalf("count=%d: %v", count, err)
		}
		values = append(values, result.AgentBytes)
	}
	for _, value := range values {
		if value <= 0 || value > 4096 {
			t.Fatalf("bounded query escaped agent byte limit: values=%v", values)
		}
	}
	for _, value := range values[1:] {
		if value != values[0] {
			t.Fatalf("bounded query scaled with result size: values=%v", values)
		}
	}

	materialized, err := runner.Execute(context.Background(), "scaling", "rows = range(10000000)\nprint(list(rows))", true)
	if err != nil {
		t.Fatalf("materialized query: %v", err)
	}
	if materialized.AgentBytes <= values[len(values)-1] {
		t.Fatalf("materialized output did not exceed bounded band: bounded=%v materialized=%d", values, materialized.AgentBytes)
	}
	t.Logf("agent_bytes counts=%v values=%v materialized=%d", counts, values, materialized.AgentBytes)
}

func formatInt(value int) string {
	if value == 0 {
		return "0"
	}
	result := ""
	for value > 0 {
		result = string(rune('0'+value%10)) + result
		value /= 10
	}
	return result
}
