//go:build darwin

package procsampler

import (
	"context"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// darwinSampler uses the kernel's KERN_PROC table. It does not shell out and
// retains parent/child identity, which is the part of the process contract
// investigations need for genealogy and zombie analysis.
type darwinSampler struct {
	delta *cpuDeltaTracker
	now   func() time.Time
}

func NewSampler() Sampler {
	return &darwinSampler{delta: newCPUDeltaTracker(), now: time.Now}
}

func (s *darwinSampler) Sample(ctx context.Context) ([]ProcessSample, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	samples := make([]ProcessSample, 0, len(procs))
	for _, proc := range procs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		comm := strings.TrimRight(string(proc.Proc.P_comm[:]), "\x00")
		samples = append(samples, ProcessSample{
			PID:     int(proc.Proc.P_pid),
			PPID:    int(proc.Eproc.Ppid),
			Comm:    comm,
			State:   string([]byte{byte(proc.Proc.P_stat)}),
			Threads: 1,
			RSSKB:   int64(proc.Eproc.Xrssize) * int64(unix.Getpagesize()/1024),
		})
	}
	s.sortAndDelta(samples)
	return samples, nil
}

func (s *darwinSampler) sortAndDelta(samples []ProcessSample) {
	s.delta.apply(samples, s.now())
	sortByCPUDesc(samples)
}
