# Windows native source checkpoint

2026-11-11: Native owned SCM/runtime, exact root outcome/absence, restricted
payload account/Job authority, explicit quota-right preparation with durable
before-effect provenance, native system RuntimeSupervisor, immutable publishing,
peer query permissions and capsule transport source are implemented. No native
host, SCM, LSA, account or hardware effects were performed by this worker.

Latest addition: native_serving_image APIs retain exact serving executable
identity. Root/child observation duplicates the inherited birthlock from a pinned
kernel process after incarnation checks; child claims carry ImageHandle and
verify account/token/Job/image against owner-verified bytes. Current system
supervisor observation requires actual SYSTEM/current fixed running manager PID,
sealed descriptor/mapped approved checksum and no-write/no-delete read lock.
No generic path-only executable fallback is used.

Validation: platform module go test -count=1 ./... PASS (root package 4.336s);
Windows CGO0 amd64 and arm64 testcompile PASS; resource transport focused race
PASS 5.496s. Hermetic/native source checks do not qualify actual Windows SCM,
LSA, token ACL, inherited handle, Job kill-on-last-close, S4U, AF_UNIX, boot,
reconnect or device behavior. Parent owns integrated product proof and evidence.

Known deliberate unsupported contracts: Windows bounded hard thread/task jobs,
POSIX OperationGate, and legacy user-mode RuntimeSupervisor installation. Windows
owned persistent services require explicit native-process-limits, whose TasksMax
bounds processes (not threads). Whole vendor executable/plugin closure is sealed
by the artifact publisher; individual image locks do not cover arbitrary DLLs.
