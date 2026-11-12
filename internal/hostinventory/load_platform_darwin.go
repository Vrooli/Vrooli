//go:build darwin

package hostinventory

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

func collectPlatformLoad(snap *Snapshot, observedAt time.Time) bool {
	markDarwinLoadProcessFields(snap)
	raw, err := unix.SysctlRaw("vm.loadavg")
	if err != nil {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("read vm.loadavg: %v", err))
		snap.ProbeStatuses["load"] = "failed"
		return true
	}
	load, err := decodeDarwinLoad(raw)
	if err != nil {
		snap.ProbeStatuses["load"] = "failed"
		snap.Warnings = append(snap.Warnings, err.Error())
		return true
	}
	if snap.CPU.Cores > 0 {
		load.NormalizedLoad1 = load.Load1 / float64(snap.CPU.Cores)
		load.NormalizedLoad5 = load.Load5 / float64(snap.CPU.Cores)
	}
	snap.Load = load
	snap.ProbeStatuses["load"] = "ok"
	snap.FieldProvenance["load"] = Provenance{SourceKind: SourceKindFile, Source: "darwin sysctl vm.loadavg", ObservedAt: observedAt, Confidence: "high", File: "vm.loadavg"}
	return true
}
