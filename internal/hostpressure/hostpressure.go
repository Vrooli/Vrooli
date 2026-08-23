// Package hostpressure reads host pressure signals without depending on a
// scenario or on a shell.  A Reading is deliberately not a bare number:
// unsupported and temporarily unavailable sensors must not become zero.
package hostpressure

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type State string

const (
	Read   State = "read"
	Unread State = "unread"
)

type Reading struct {
	Value      float64 `json:"value,omitempty"`
	State      State   `json:"state"`
	Provenance string  `json:"provenance,omitempty"`
	Reason     string  `json:"reason,omitempty"`
}

func NewRead(value float64, provenance string) Reading {
	return Reading{Value: value, State: Read, Provenance: provenance}
}

func NewUnread(provenance, reason string) Reading {
	return Reading{State: Unread, Provenance: provenance, Reason: reason}
}

func (r Reading) Number() (float64, bool) { return r.Value, r.State == Read }

type Process struct {
	PID       int64   `json:"pid"`
	Name      string  `json:"name"`
	PPID      int64   `json:"ppid"`
	Resident  uint64  `json:"resident_bytes"`
	Swapped   uint64  `json:"swapped_bytes"`
	IdleSince float64 `json:"idle_since_unix,omitempty"`
}

type PressureSnapshot struct {
	CapturedAt   time.Time `json:"captured_at"`
	CPUPressure  Reading   `json:"cpu_pressure"`
	Load1        Reading   `json:"load_1m"`
	MemoryTotal  Reading   `json:"memory_total_bytes"`
	MemoryAvail  Reading   `json:"memory_available_bytes"`
	SwapTotal    Reading   `json:"swap_total_bytes"`
	SwapUsed     Reading   `json:"swap_used_bytes"`
	ProcessCount Reading   `json:"process_count"`
	ForkRate     Reading   `json:"fork_rate_per_second"`
	ForkCounter  Reading   `json:"fork_counter"`
	Processes    []Process `json:"processes,omitempty"`
}

type Options struct {
	// ProcRoot is primarily a test seam. Empty means /proc on Linux.
	ProcRoot string
	Previous *PressureSnapshot
	Now      func() time.Time
}

func Collect(ctx context.Context, opts Options) PressureSnapshot {
	return collect(ctx, opts)
}

func statCounter(content, key string) (uint64, bool) {
	for _, line := range strings.Split(content, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] == key {
			n, err := strconv.ParseUint(f[1], 10, 64)
			return n, err == nil
		}
	}
	return 0, false
}
