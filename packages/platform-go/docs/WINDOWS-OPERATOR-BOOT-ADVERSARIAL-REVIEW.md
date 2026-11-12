# Windows operator boot adversarial maturation

2026-11-11. Independent task/COM/lifetime pass in the isolated native-lighting source tree. Material findings were repaired and affected checks rerun. This is a source review and build result; it is not a clean consecutive acceptance pass or Windows runtime qualification.

Changed:

- Ordinary reserved RuntimeSupervisor start/stop/restart now use the selected-operator lifecycle mutex and protected descriptor. Restart holds the mutex across both operations. Ordinary stop proves and retains the actual action process, stops the recorded scheduler GUID, and waits for both manager absence and that exact process termination. Setup/removal additionally disable the task. Ordinary lifecycle retains read/execute task authority and never requests task-definition writes; managed-job opt-out remains durable runtime policy.
- Reserved RuntimeSupervisor status no longer adopts a legacy same-name SCM service. A running result requires the protected task and actual action identity; unproved topology returns an error/unknown result. Generic task/service status remains generic.
- Boot-definition proof now includes boot delay, no extra time trigger, unlimited execution, IgnoreNew, no battery/network/idle/RemoteApp/volatile gates, start-on-demand, missed-start handling, and expected restart policy. Registration compares observed boot/time-trigger and relevant settings fields as well as principal/action. Optional documented false defaults normalize before comparison. Foreign attribute namespaces are refused.
- COM EXCEPINFO deferred fill-in runs while the apartment is alive, before SCODE interpretation and BSTR cleanup. Native ABI and deferred missing-task HRESULT fixtures compile for both supported Windows architectures.
- A live held-handle check closes the PID-reuse window between reopening an action handle and confirming the exact action mapping.

Verified:

- Focused `go test -v -run 'TestWindows(Task|OperatorBoot)' .` passed, including new continuous-operation counterexamples, namespace rejection, action-parent/birth identity and replacement fixtures.
- Full platform module `go test -json ./...` passed in 4.461s: 453 pass events, zero fail events, eight skips. Complete local diagnostic output: `/tmp/vrooli-windows-boot-review-platform-tests.jsonl`. This executed the current common/Linux-selected suite; it did not execute Windows-only tests.
- Windows amd64 and arm64 `CGO_ENABLED=0 go test -c` passed after the final Windows status change. Windows amd64 and arm64 `go vet ./...` passed. Test executables were compilation artifacts only and removed after checking.
- Initial concurrent common migration failure (`native_owned_persistent.go` referenced removed `darwinAccountReservation`) resolved at its owner; no shim or generated-file edit was introduced here.

Remaining and unverified:

- Execute native COM fixtures and actual register/query/security-descriptor normalization on Windows. S4U boot/logout, existing Batch-logon policy, selected profile initialization, exact engine/action topology, detached Job survival, selected HID access, crash recovery and physical lighting remain open. No account, logon-right, task/service registration, device effect or target dispatch was performed.
- Native scheduler methods are synchronous COM calls. Context bounds govern retry/wait loops and are checked around identity work; they do not currently cancel an in-flight COM RPC. A native unavailable-service/hung-call test must characterize that limitation before claiming hard lifecycle deadline enforcement. No cancellation goroutine or leaked worker thread was added.
- No-instance manager status is manager evidence, not independently proven absence of all orphan processes. Exact ordinary stop has a retained action handle only when the native instance exists. Engine/action topology and scheduler crash behavior require native qualification, without adopting processes by name/PID.
- This material-finding pass resets any consecutive clean-source-pass count. Full CHARTER/SPEC, actual native and hardware acceptance, and independent subsequent passes remain the parent's destination.

Primary references: [Task security contexts and read/write/execute permissions](https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks), [Task Scheduler schema and defaults](https://learn.microsoft.com/en-us/windows/win32/taskschd/task-scheduler-schema), [EXCEPINFO ABI and deferred callback](https://learn.microsoft.com/en-us/windows/win32/api/oaidl/ns-oaidl-excepinfo). Native COM ABI uses the SDK definition; remote wire EXCEPINFO fields are a different contract.

No service, resource, model, account or native target process was started. No Git, dependency acquisition or main-checkout source mutation occurred. Shared common/Windows Job/Darwin files remain their owning siblings' responsibility.
