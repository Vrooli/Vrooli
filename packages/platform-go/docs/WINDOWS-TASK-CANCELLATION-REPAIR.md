# G1 Windows Task cancellation and extinction repair

2026-11-11; isolated source only. The whole-lifecycle G review is finding-bearing.
This repair and its aftercheck do not become a fresh clean whole-product review
or native Task Scheduler qualification.

## Cause and documented proof

Per-role stop accepted unknown and out-of-range task states when GetInstances was
empty; its final delete could also erase queued/running evidence. The ordinary
boot supervisor accepted any empty instance list as quiet, stopped enumerated
instances only and could Run again during restart despite queued work.

Microsoft defines a finite [TASK_STATE enum](https://learn.microsoft.com/en-us/windows/win32/api/taskschd/ne-taskschd-task_state).
Only DISABLED1 and READY3 establish that no action is queued or running. UNKNOWN0
cannot prove extinction; values outside0..4 are invalid. [GetInstances](https://learn.microsoft.com/en-us/windows/win32/api/taskschd/nf-taskschd-iregisteredtask-getinstances)
reports currently running instances under the caller's security context, so its
empty result is insufficient by itself.

[IRegisteredTask.Stop](https://learn.microsoft.com/en-us/windows/win32/api/taskschd/nf-taskschd-iregisteredtask-stop)
uses flags0 and acknowledges cancellation with S_OK. S_FALSE remains failure.
The documented [SCHED_E_TASK_NOT_RUNNING](https://learn.microsoft.com/en-us/windows/win32/taskschd/task-scheduler-error-and-success-constants)
means no running instance; it does not exclude queued work. Only that exact error
can use a fresh positive READY/DISABLED, zero-instance observation for idempotent
already-inactive Stop. Missing, access, manager, account and other errors are not
reclassified as this case.

## Repair in existing owners

- Shared finite-state/quiescence, Run and deletion policies are interpreted by
  the existing role and boot owners. Native task and instance readers reject
  invalid enum values; UNKNOWN remains observable but never proves quiet.
- Per-role stop still revokes, disables and cancels its exact registered task on
  each attempt. It requires positive quiet state before the existing independent
  Job/root extinction proof. Final deletion separately revalidates disabled,
  known-quiet exact definition and verifies native absence afterward.
- The ordinary boot owner invokes that same native registered-task Stop even
  when GetInstances exposes no running instance. It then requires known quiet
  installed-task evidence and held exact action-process exit where applicable.
  Ordinary Stop leaves the BootTrigger enabled; authorized setup/removal retains
  its existing disable branch and independently confirms disabled quiescence.
- Restart freshly revalidates the installed owner after stop instead of clearing
  its local instance list. Start/Restart issues Run only for an enabled READY task;
  QUEUED/RUNNING waits for the already-submitted exact action. UNKNOWN/invalid or
  disabled state cannot authorize another Run.
- Protected boot removal revalidates its exact descriptor/account/image and
  disabled known-quiet snapshot immediately before Delete, then proves actual
  absence. Status reports Stopped only from positive quiet state with zero
  instances; contradictory observations remain Unknown.

The existing COM registered-stop implementation moved to the Task Scheduler
owner and is shared. No new account, privilege, broker, roster, native persisted
format or execution fallback was introduced. Legacy Linux and other OS behavior
were not changed. Existing source-owner patterns already cover the new common
`windows_task_contract_state.go` policy.

## Focused proof and limits

`windows_task_contract_state_test.go` exercises the actual shared normal
Start/Stop/Restart/Delete orchestration: already-quiet repeat Stop, preserved boot
enablement, queued/no-instance cancellation, fresh post-stop UNKNOWN/invalid or
queued/running state, no duplicate Run, bounded unresolved cancellation, exact
finite deletion states and post-delete absence. Native cancellation/process
edges remain modeled; the original G OS facet owns actual Windows-body extraction
and independent HRESULT/caller sequencing aftercheck.

All checks set isolated VROOLI_ROOT and VROOLI_SOURCE_ROOT and private
VROOLI_PROTO_ARTIFACT_ROOT. Focused task/operator race PASS1.037s; complete ordinary
platform module PASS5.538s (all subpackages pass); scoped statecontract checks
PASS0.836s and maintained supervisor service/native fixtures PASS2.211s. All six
CGO0 **complete CLI builds** pass: Linux amd64/arm64, Darwin amd64/arm64, Windows
amd64/arm64. Exact commands/results are retained under
`.vrooli/work/native-g1/builds/command-build-receipts.json`.

The repair source/dependency manifest is
`.vrooli/work/native-g1/source-manifest.json`, SHA256
`d9328c121065f8e0eedad7b260ad4e65afb4a3874edea99a6b5ea9b048756bba`.
Independent original G OS aftercheck passed: retained original assertions plus
actual native-body HRESULT/cancellation/boot-control probes, 9 top-level tests /
20 entries race PASS1.197s; current Task/operator selection, 20 top-level tests /
47 entries race PASS1.042s. The independent 266-input manifest
`6c2f2576e42380af2192deccd5866dfd52415a2e89aae934feb2c23382977f5b`
has zero source drift. Exact aftercheck report/log receipts are under
`.vrooli/work/reviews/native-lifecycle-g-os/repair-aftercheck`. No remaining
material G1 source finding was established. No Windows Task, Job, token, process, service, account or device was
operated. Actual scheduler cancellation/HRESULT/state reachability, S4U/profile,
action lifetime, native HID/SDK/physical output and boot qualification remain
open. No MAIN transfer or additional review round is part of this repair.
