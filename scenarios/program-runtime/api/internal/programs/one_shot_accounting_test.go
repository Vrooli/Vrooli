package programs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

type sampledOneShotRunner struct{ executed, killed bool }

func (r *sampledOneShotRunner) Execute(context.Context, string, string, bool) (Result, error) {
	r.executed = true
	return Result{Stdout: "ok"}, nil
}

func (r *sampledOneShotRunner) CPUTime(string) (time.Duration, bool) {
	if r.killed {
		return 0, false
	}
	if r.executed {
		return 12 * time.Millisecond, true
	}
	return 0, true
}

func (r *sampledOneShotRunner) MemoryBytes(string) (int64, bool) {
	if r.killed {
		return 0, false
	}
	return 1024, true
}
func (r *sampledOneShotRunner) KillSession(string) { r.killed = true }

func TestOneShotSamplesUsageBeforeKernelReclamation(t *testing.T) {
	runner := &sampledOneShotRunner{}
	var memory int64
	service := NewService(Options{
		Runner:           runner,
		AuthorizeSession: func(context.Context, string) (bool, error) { return true, nil },
		RecordMemory:     func(_ string, value int64) { memory = value },
	})
	result, err := service.Submit(context.Background(), "one-shot", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false)
	require.NoError(t, err)
	require.EqualValues(t, 12, result.CpuTimeMillis)
	require.EqualValues(t, 1024, memory)
	require.True(t, runner.killed)
}
