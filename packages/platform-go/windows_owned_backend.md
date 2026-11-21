# Windows owned workloads

This is the source implementation of actual operator workloads. Native Windows
runtime, permissions, device and logged-out boot qualification remain OPEN.
Cross-builds and fixture results do not qualify those behaviors.

## One owned lifecycle

`OwnedServiceSpec.NativeAccount` is the actual user's canonical `sid:<SID>`.
Resolution verifies a real user SID, its account roundtrip and equality with the
current operator. It creates no virtual account, SCM service or LSA privilege.
The ordinary maintained RuntimeSupervisor boot actor is separately owned by the
native Task Scheduler BootTrigger/S4U/LeastPrivilege integration. Per-role owned
workloads use the same platform Start/Lookup/Inspect/Stop/Release contract.

The ordinary caller and boot actor both use the installed immutable maintained
host as code authority. A sealed resource launcher copy may have a different
path/inode but must match those approved bytes, immutable ACL/ancestry and import
closure. There is no mutable configured launcher/global registration or direct
process fallback. Before any intent/native dispatch, the finite maintained
capsule entrypoint and canonical custom-root `launch/<attempt>/capsule.json`
shape are checked. Generic executable/argv services are refused.

Start publishes the complete frozen v2 intent/specification/account/boot in
`HKEY_USERS\<actualSID>\Software\Vrooli\OwnedJobs\<invocation>`, with a
protected user/System DACL and OWNER RIGHTS read-control. An unloaded SID hive
is a setup failure; S4U never falls back to HKCU/.DEFAULT. Start then registers
one UUID-specific on-demand Task Scheduler S4U/LeastPrivilege action with EXACT
frozen host, argv and working directory. The same user's task DACL permits only
that SID and System; the protected machine boot task remains a separate owner.
Registration is create-only, never update/adopt. No trigger/retry, time limit,
battery, idle or network policy can shorten or replay role lifetime. A durable
RunIssued marker precedes the sole native Run. Registration/Run/ACK failure
retains the reserved slot and cannot authorize a second Run. S4U/batch-logon
policy failures remain actual failures; this owner grants no privilege.

The task action IS the existing trusted wrapper, not a departing relay. After
its non-effecting typed capsule read, it calls
`PrepareWindowsOwnedServiceSubmission(ctx,intent)` before any witness, claim,
owner IPC, hardware or vendor code. Under the same admission mutex it checks
nonrevoked issued intent, exact frozen args/cwd, standard actual user/boot,
current mapped approved host and a stable singleton native scheduler GUID.
EnginePID is not used as the root PID: actual root birth and kernel parent
relation plus exact action fields bind it. Shared engine topology and actual
S4U/profile behavior remain native qualification obligations.

This trusted early bootstrap exclusively creates the named global Job, sets
recursive memory/process/CPU limits, joins itself, retains the no-write/delete
image lock, and flushes its exact self root/image/Job handles plus TaskInstance
GUID. The caller's scenario Job is never an ancestor of this manager-created
action; no caller breakaway exception is needed. The wrapper normalizes its
environment to the frozen specification before continued resource witnesses.
It writes pending custody, flushes exact bytes with QUERY_VALUE|SET_VALUE,
then publishes a hash-matched CustodyReady marker. Start returns only after
independent root/image/held-Job/task verification. The wrapper retains the only
owning Job handle until exit; vendor children inherit only their own image lock.
There is no Job-handle/environment bootstrap from the submitting CLI.

An empty protected invocation key created before Payload publication has not
admitted execution. Read-only lookup reports unpublished absence only after
zero values/subkeys and native Job namespace absence are independently proven.
Under the submission mutex, Start may remove/recreate that exact empty slot.
Value-bearing incomplete/corrupt records and retained Jobs never authorize this
cleanup or replay. Complete published intents retain ordinary no-replay rules.

Lost dispatch/ACK remains uncertain and is looked up from the existing intent;
it never authorizes duplicate launch. Inspect reports Running only with exact
live root/birth image/custody, actual held Job and native task incarnation. Each
query lease closes before returning. A rootless queued or partially prepared
submission is abort-only, never adopted as healthy.

Stop serializes admission and durably revokes the intent before disabling the
exact task. Disable alone is not pending cancellation: `IRegisteredTask.Stop`
stops that UUID-specific registered task before instance/quiescence inspection.
Every retry performs the same cancellation; native errors remain errors. Empty
GetInstances alongside queued/running state remains unknown. Native pending
Run/Stop semantics still require Windows execution proof. Exact Job limits/name
and boot are checked before recursive termination. A delayed early bootstrap
cannot claim a revoked/deleted intent or reach vendor/hardware work.

Release refuses a live root or retained Job before any retirement, then revokes
rootless queued admission, cancels the exact task, confirms no instances/Job,
deletes the task and confirms documented native missing-task evidence before
removing the ledger. After an independently proven boot change, old PID and task
instance identity never authorize signaling: exact Job absence permits retirement
of the still-verified no-trigger task, while any newly occupied same-name Job
is refused. The original old claim remains until retirement completes.
Credential/configuration/access/service HRESULT failures
are never classified as task absence. Unknown wrapper exit stays unknown;
process disappearance is not SDK success. Hard thread-count bounded operations
and Unix-style gates remain unsupported; services require explicit
`native-process-limits`, which is not a Linux task/thread quota.

## Actual operator and verified child

The approved model trusts the exact pinned OpenRGB image as on Linux. It runs
with the actual operator's standard non-elevated token. No dedicated resource
account, full restricting capability token, broad privilege grant, private
desktop prerequisite or arbitrary compromised-vendor sandbox is introduced.
The abandoned unoperated research candidate is retained under
`.vrooli/work/native-portability-history/platform/windows-restricted-authority-candidate-20261111`.

Start and trusted task bootstrap independently check actual SID and
non-elevation. A resource wrapper calls `PrepareOwnedImageChild` for its verified
open vendor image. The factory locks the original pathname against write/delete,
compares native file identity, and calls ordinary `CreateProcessW` suspended with
exact explicit argv/environment/cwd and no inherited console. The wrapper's Job
membership is inherited atomically at child birth. The explicit HANDLE_LIST has
only the vendor image lock: neither Job nor bootstrap/ledger/token handles leak.

The factory independently reads the child's actual SID and elevation before
returning and again before Resume. Persist `OwnedImageChild.Identity()` including
its lowercase security-facts hash before Resume; Abort if persistence fails.
The hash asserts only observed actual user/non-elevation, not sandboxing or least
privilege. It never compares token handle/TokenId equality or requests token
duplication. Independent live/crash child verification compares those facts,
exact incarnation, birth-established image handle, mapped image and exact Job.
`Wait` observes only that child, not successful light output or recursive
service completion. Generic Windows ReplaceProcessImage remains unsupported.

The resource/artifact owners retain the meaningful hazard boundaries: exact
pinned immutable executable/DLL closure, fixed six-detector PawnIO omission,
explicit config before construction, finite approved HID selectors and current
generation observations, private loopback SDK, durable capsule/once-claim and
managed Off/preset policy. Same-operator trusted native code is not represented
as an arbitrary compromised same-user sandbox. Standard `CreateProcessW` needs
no SeIncreaseQuota/SeAssignPrimaryToken requirement and introduces no account
pivot or alternate-logon fallback.

## Qualification

Source fixtures cover frozen invocation, broad/legacy identity refusal, reserved
bootstrap environment, custody binding, interrupted submission no-replay, and
actual-user security proof retention/refusal. Native targets must still qualify
loaded SID profile/S4U, sealed task action image replacement refusal, exact Job
custody/ACK and all submitter/wrapper crash boundaries, recursive stop/release/
absence and actual token/SDK observations. Parent/scheduler ancestor Jobs and
detached executor maintenance must be observed explicitly; nested Job compilation
proves neither breakaway nor survival. Logged-out boot, actual device output,
replug/resume and Windows ARM artifact availability are separate evidence. No
Windows ARM OpenRGB release artifact is implied by platform-go ARM64 source
compatibility.

Primary documentation: [Job objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects),
[process creation](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessw),
[extended startup attributes](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute).

The opt-in `windows_owned_task_live_test.go` native qualification harness requires
an explicitly prepared isolated real capsule and immutable installed host. It
places its submitter helper inside a kill-on-close outer Job, requires wrapper
plus SDK descendant, closes that outer Job and asserts role survival. Separate
fresh capsules exercise exact task stop and wrapper custody loss, requiring
recursive disappearance. It never runs by default and was NOT executed on the
Linux implementation host. Its compiled presence is qualification SOURCE only.

Task policy/identity and submission ordering fixtures run hermetically. The
lifetime model is labeled as a model; neither it nor Windows cross-compilation
is actual inherited Job or Scheduler evidence.

Further primary sources: [security contexts for tasks](https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks),
[nested jobs](https://learn.microsoft.com/en-us/windows/win32/procthread/nested-jobs),
[registered task stop](https://learn.microsoft.com/en-us/windows/win32/taskschd/registeredtask-stop),
[execution time limits](https://learn.microsoft.com/en-us/windows/win32/taskschd/tasksettings-executiontimelimit).
