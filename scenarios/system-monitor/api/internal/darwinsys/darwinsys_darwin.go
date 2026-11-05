//go:build darwin

package darwinsys

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

// The runtime exports these for golang.org/x/sys; they perform a libc call
// on the system stack and return errno when the call reports -1.
//
//go:linkname syscall_syscall syscall.syscall
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname syscall_syscall6 syscall.syscall6
func syscall_syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

// Trampoline addresses are defined in the per-architecture assembly files.
var (
	libc_mach_host_self_trampoline_addr    uintptr
	libc_host_statistics64_trampoline_addr uintptr
	libc_proc_pid_rusage_trampoline_addr   uintptr
)

//go:cgo_import_dynamic libc_mach_host_self mach_host_self "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_host_statistics64 host_statistics64 "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_proc_pid_rusage proc_pid_rusage "/usr/lib/libSystem.B.dylib"

// mach_host_self allocates a send right on every call, so the port is
// resolved once and reused for the life of the process.
var (
	hostPortOnce sync.Once
	hostPort     uintptr
)

func machHostSelf() uintptr {
	hostPortOnce.Do(func() {
		port, _, _ := syscall_syscall(libc_mach_host_self_trampoline_addr, 0, 0, 0)
		hostPort = uintptr(uint32(port))
	})
	return hostPort
}

// HostStatistics64 returns the host-wide virtual-memory counters that
// vm_stat(1) prints.
func HostStatistics64() (VMStatistics64, error) {
	var stats VMStatistics64
	count := uint32(unsafe.Sizeof(stats) / 4)
	kr, _, _ := syscall_syscall6(libc_host_statistics64_trampoline_addr, machHostSelf(), HostVMInfo64,
		uintptr(unsafe.Pointer(&stats)), uintptr(unsafe.Pointer(&count)), 0, 0)
	if code := int32(kr); code != 0 {
		return VMStatistics64{}, fmt.Errorf("host_statistics64 returned kern_return %d", code)
	}
	return stats, nil
}

// ProcPIDRusage returns RUSAGE_INFO_V0 for pid. Other users' processes fail
// with EPERM unless the caller is privileged.
func ProcPIDRusage(pid int) (RusageInfoV0, error) {
	var info RusageInfoV0
	r, _, errno := syscall_syscall(libc_proc_pid_rusage_trampoline_addr, uintptr(pid), 0, uintptr(unsafe.Pointer(&info)))
	if int32(r) != 0 {
		if errno == 0 {
			errno = syscall.EINVAL
		}
		return RusageInfoV0{}, fmt.Errorf("proc_pid_rusage(%d): %w", pid, errno)
	}
	return info, nil
}
