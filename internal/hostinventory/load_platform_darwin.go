//go:build darwin

package hostinventory

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"golang.org/x/sys/unix"
)

func collectPlatformLoad(snap *Snapshot, observedAt time.Time) bool {
	raw, err := unix.SysctlRaw("vm.loadavg")
	if err != nil {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("read vm.loadavg: %v", err))
		snap.ProbeStatuses["load"] = "failed"
		return true
	}
	if len(raw) < 24 {
		snap.ProbeStatuses["load"] = "failed"
		snap.Warnings = append(snap.Warnings, "vm.loadavg returned fewer than three values")
		return true
	}
	load := Load{Load1: mathFloat64(raw[0:8]), Load5: mathFloat64(raw[8:16]), Load15: mathFloat64(raw[16:24])}
	if snap.CPU.Cores > 0 {
		load.NormalizedLoad1 = load.Load1 / float64(snap.CPU.Cores)
		load.NormalizedLoad5 = load.Load5 / float64(snap.CPU.Cores)
	}
	snap.Load = load
	snap.ProbeStatuses["load"] = "ok"
	snap.FieldProvenance["load"] = Provenance{SourceKind: SourceKindFile, Source: "darwin sysctl vm.loadavg", ObservedAt: observedAt, Confidence: "high", File: "vm.loadavg"}
	return true
}

func mathFloat64(b []byte) float64 { return math.Float64frombits(binary.LittleEndian.Uint64(b)) }
