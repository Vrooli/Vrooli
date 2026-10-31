// Package darwinsys binds the two macOS memory APIs system-monitor needs and
// golang.org/x/sys does not expose: the Mach host_statistics64 call
// (HOST_VM_INFO64) and libproc's proc_pid_rusage.
//
// It is cgo-free. The darwin build uses the same libSystem trampoline
// technique golang.org/x/sys/unix uses for every darwin syscall: a
// //go:cgo_import_dynamic directive names the libSystem symbol, a one-line
// assembly trampoline jumps to it, and the runtime's syscall.syscall helper
// performs the libc call on the system stack. Verified on macOS 15.7 (Intel)
// against vm_stat output; Apple Silicon is compile-verified only.
//
// The types below are declared for every platform so collectors can convert
// recorded fixtures in tests on any host.
package darwinsys

import "errors"

// ErrUnsupported is returned on platforms other than darwin.
var ErrUnsupported = errors.New("darwinsys: macOS native memory APIs are unavailable on this platform")

// HostVMInfo64 is the host_statistics64 flavor that fills VMStatistics64.
const HostVMInfo64 = 4

// VMStatistics64 mirrors struct vm_statistics64 from <mach/vm_statistics.h>.
// natural_t fields are uint32; counters are uint64. Layout: 152 bytes, which
// is HOST_VM_INFO64_COUNT (38) natural_t words.
type VMStatistics64 struct {
	FreeCount                          uint32
	ActiveCount                        uint32
	InactiveCount                      uint32
	WireCount                          uint32
	ZeroFillCount                      uint64
	Reactivations                      uint64
	Pageins                            uint64
	Pageouts                           uint64
	Faults                             uint64
	CowFaults                          uint64
	Lookups                            uint64
	Hits                               uint64
	Purges                             uint64
	PurgeableCount                     uint32
	SpeculativeCount                   uint32
	Decompressions                     uint64
	Compressions                       uint64
	Swapins                            uint64
	Swapouts                           uint64
	CompressorPageCount                uint32
	ThrottledCount                     uint32
	ExternalPageCount                  uint32
	InternalPageCount                  uint32
	TotalUncompressedPagesInCompressor uint64
}

// RusageInfoV0 mirrors struct rusage_info_v0 from <sys/resource.h>
// (RUSAGE_INFO_V0). PageIns counts pages read from disk for the process,
// which is the closest macOS analogue of Linux major faults.
type RusageInfoV0 struct {
	UUID             [16]byte
	UserTime         uint64
	SystemTime       uint64
	PkgIdleWkups     uint64
	InterruptWkups   uint64
	PageIns          uint64
	WiredSize        uint64
	ResidentSize     uint64
	PhysFootprint    uint64
	ProcStartAbstime uint64
	ProcExitAbstime  uint64
}
