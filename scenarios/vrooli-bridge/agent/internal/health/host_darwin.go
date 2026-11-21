//go:build darwin

package health

import (
	"encoding/binary"
	"syscall"
)

// readHostMetrics reads memory and swap through sysctl, without cgo or a
// subprocess: hw.memsize, kern.memorystatus_vm_pressure_level (the level
// Activity Monitor's pressure graph shows) and vm.swapusage. macOS grows swap
// on demand, so swap is also judged against installed memory.
func readHostMetrics() hostMetrics {
	m := hostMetrics{PSISomeAvg60: -1, SwapGrowsOnDemand: true}
	if total, ok := sysctlUint64("hw.memsize"); ok {
		m.MemTotal = total
	}
	if level, err := syscall.SysctlUint32("kern.memorystatus_vm_pressure_level"); err == nil {
		m.PressureLevel = int(level)
	}
	if m.MemTotal > 0 || m.PressureLevel != pressureUnknown {
		m.MemoryKnown = true
	} else {
		m.ReadError = "sysctl reported neither installed memory nor memory pressure"
	}
	// struct xsw_usage { u_int64_t total, avail, used; u_int32_t pagesize; boolean_t encrypted; }
	if raw, ok := sysctlRaw("vm.swapusage", 24); ok {
		m.SwapKnown = true
		m.SwapTotal = binary.LittleEndian.Uint64(raw[0:8])
		m.SwapUsed = binary.LittleEndian.Uint64(raw[16:24])
	} else {
		m.SwapReadError = "sysctl vm.swapusage could not be read"
	}
	return m
}

// sysctlRaw returns at least minLen bytes of a binary sysctl value. The stdlib
// Sysctl drops one trailing NUL byte, which truncates little-endian integers
// whose top byte is zero; restoring it with zero padding is exact.
func sysctlRaw(name string, minLen int) ([]byte, bool) {
	value, err := syscall.Sysctl(name)
	if err != nil || len(value) == 0 || len(value) < minLen-1 {
		return nil, false
	}
	raw := []byte(value)
	for len(raw) < minLen {
		raw = append(raw, 0)
	}
	return raw, true
}

func sysctlUint64(name string) (uint64, bool) {
	raw, ok := sysctlRaw(name, 8)
	if !ok {
		return 0, false
	}
	return binary.LittleEndian.Uint64(raw[0:8]), true
}
