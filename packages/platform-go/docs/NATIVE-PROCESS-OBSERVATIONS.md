# Native process observations

`ProcessArguments`, environment, working directory, image and socket observations
are evidence, not authorization. Callers must check the process incarnation before
and after collecting them. Environment and arguments can contain secrets; compare
in memory and omit them from receipts. Mutable process metadata does not establish
what a process originally launched with.

`ProcessTCPListeners(pid)` includes every observed IPv4/IPv6 listener of one PID.
`TCPListeners(port)` includes every observed PID listening at that port, including
wildcard addresses. Requiring loopback means explicitly accepting the expected
literal address and excluding other returned endpoints. Permission, truncation or
unrecognized-table errors must not be interpreted as an empty listener set. These
are snapshots, not atomic exclusion against a subsequent competing bind.

Windows reads process metadata through native process memory with bounded sizes
and rejects cross-architecture PEB layouts. Image paths use the native kernel image
name query. Token observations return SID and UAC elevation; a non-elevated token
alone does not prove least privilege. Socket ownership comes from both native TCP
owner tables. `OpenProcessExecutable` remains unsupported: opening an executable
pathname alone cannot establish the mapped file identity. Verified suspended-child
launch belongs to the owned-image/Job owner.

Darwin uses kernel sysctls for arguments, environment, children and effective UID.
The cgo backend uses libproc for cwd, descriptor socket ownership and an executable
file whose device/inode match its executable mapping. It does not substitute
pathname equality for mapped-file identity. Whole-host listener enumeration instead uses the read-only global
`net.inet.tcp.pcblist_n` snapshot, available independently of foreign-process
FD permissions. Its packed-4 record prefixes and 8-byte alignment follow Apple's
maintained XNU headers. Unknown/incomplete groups, missing trailers, invalid
owners and changed generations fail closed with bounded retries. Global snapshot
PIDs are kernel recorded last/effective ownership; the per-PID descriptor witness
independently proves the expected process actually holds the socket. A global
snapshot does not enumerate every holder of a duplicated descriptor. Because this
route has no PID enumeration, process-list self/PID1 canaries do not apply; kernel
header/trailer counts and generations guard completeness. Counts must match all
emitted groups, conservatively refusing even kernel-skipped dead PCB entries.
The no-cgo backend supports global PCB observation and explicitly reports
unsupported for libproc operations. Neither backend shells out to ps/lsof for these facts.

The new security and socket APIs return unsupported on other platforms; existing
Linux process and resource-specific listener mechanisms are unchanged.

## Evidence boundary

2026-11-11: Linux package tests and six-target no-cgo source builds passed during
implementation. Parser fixtures cover IPv4/IPv6, foreign owners, wildcards, port
byte order, truncated tables and unknown listener states. Darwin cgo SDK compilation
and native runtime behavior, Windows native runtime behavior, and hardware/headless
qualification remain open. No service, device, target dispatch or host security
configuration was changed to obtain this evidence.

Native ABI sources: Apple's maintained
[proc_info.h](https://github.com/apple/darwin-xnu/blob/main/bsd/sys/proc_info.h)
and Microsoft's
[GetExtendedTcpTable](https://learn.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getextendedtcptable)
and [owner row structures](https://learn.microsoft.com/en-us/windows/win32/api/tcpmib/ns-tcpmib-mib_tcp6row_owner_pid).

## Native qualification command

On each Darwin target, run
`VROOLI_PLATFORM_NATIVE_LIVE=1 go test -run TestDarwinNativeProcessObservationLive .`
with cgo enabled and the target SDK. The fixture checks mapped image file identity,
cwd, effective UID, an actual self-held loopback listener, and its presence in the
global PCB table under the same account. Run it under the provisioned narrow
service account as well as the target operator. It does not qualify lighting
hardware or the service lifecycle. No native live test has been executed by this
Linux source worker.

Source audit 2026-11-11 verified public libproc declarations and mapping members
against Apple's maintained
[libproc.h](https://github.com/apple-oss-distributions/xnu/blob/main/libsyscall/wrappers/libproc/libproc.h)
and [proc_info.h](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/proc_info.h).
The global route follows
[in_pcblist.c](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/netinet/in_pcblist.c),
[in_pcb.h](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/netinet/in_pcb.h),
[socketvar.h](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/socketvar.h)
and [tcp_var.h](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/netinet/tcp_var.h).
The kernel handler iterates all PCBs without caller-UID filtering. Runtime ABI and
permission qualification remain distinct from that primary-source finding.
