package hostinventory

import (
	"encoding/binary"
	"fmt"
)

// decodeDarwinLoad decodes XNU user64_loadavg: three uint32 fixpt_t values,
// four bytes of alignment padding, then a signed 64-bit long fscale.
// Both supported Darwin Go architectures (amd64 and arm64) use this ABI.
func decodeDarwinLoad(raw []byte) (Load, error) {
	if len(raw) != 24 {
		return Load{}, fmt.Errorf("vm.loadavg returned %d bytes; expected user64_loadavg (24)", len(raw))
	}
	scale := int64(binary.LittleEndian.Uint64(raw[16:24]))
	if scale <= 0 {
		return Load{}, fmt.Errorf("vm.loadavg returned invalid fscale %d", scale)
	}
	return Load{Load1: float64(binary.LittleEndian.Uint32(raw[0:4])) / float64(scale), Load5: float64(binary.LittleEndian.Uint32(raw[4:8])) / float64(scale), Load15: float64(binary.LittleEndian.Uint32(raw[8:12])) / float64(scale)}, nil
}

// Process fields are absent from vm.loadavg even when all load averages are
// measured. Keep absence in the status map carried through scalar proto views.
func markDarwinLoadProcessFields(snap *Snapshot) {
	if snap.ProbeStatuses == nil {
		snap.ProbeStatuses = map[string]string{}
	}
	for _, field := range []string{"running_procs", "total_procs", "last_pid"} {
		snap.ProbeStatuses["load."+field] = "unsupported"
	}
}
