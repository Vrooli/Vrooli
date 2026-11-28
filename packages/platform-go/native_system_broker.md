# Native operator supervisor installation

`ServiceInstallOptions{User:false, SystemBroker:...}` explicitly installs the
existing RuntimeSupervisor for boot on Darwin/Windows. The historical broker
API and hidden entrypoint names do not authorize a privileged runtime loop.
Elevated setup publishes immutable maintained control-plane bytes and native
boot metadata; the manager runs the supervisor as the actual non-elevated
operator. Resource opt-out and device permission decisions retain their owners.

Setup verifies the selected native operator and account home, validates the
checksum-approved native executable and loader closure, and publishes it with a
strictly decoded immutable descriptor in a machine-owned version directory.
Descriptor identity includes the typed options. Existing different bytes are
never replaced by publication. The fixed entrypoint is
`__vrooli-runtime-system-broker <descriptor-path>`; its early dispatcher calls
`ReadNativeSystemServiceBrokerDescriptor`, then `RunNativeSystemSupervisor`.
These prove the exact native manager, operator, descriptor and mapped image
before entering the ordinary runtime. Operator HOME and working directory are
frozen in the native definition; source-root and arbitrary log overrides are
refused. Runtime state belongs to that operator, not to a root-owned worker.

Darwin installs the existing RuntimeSupervisor label as a system LaunchDaemon
with explicit UserName/GroupName. Native root/mode/extended-ACL checks protect
its executable and descriptor. Role workloads separately use exact user-domain
launchd jobs and the closed maintained capsule admission contract.

Windows installs the existing supervisor as a BootTrigger Task Scheduler task
with S4U and LeastPrivilege. The native COM adapter verifies exact XML,
registration security, selected SID and actual action process authority; task
EnginePID alone never proves the running action. S4U supplies local-only
credentials and does not imply network credentials, an unlocked keychain,
interactive login or account-policy readiness. No fallback to LocalSystem,
HighestAvailable or an interactive user's token is permitted. Ordinary task
start/stop preserves the protected registration; administrator replacement
first disables the exact task and stops the verified action/instance.

The optional frozen GlobalResourceStorageRoot binds the operator's existing
configured resource namespace, including an explicitly enrolled default.
`EnrolledNativeOperatorResourceStorageRoot` reads only the protected fixed boot
definition and descriptor, checks the requested operator home, and distinguishes
no enrollment from enrolled default. A missing descriptor for an existing task,
permission failure or changing definition is an error, never absence. The native
adapter applies the frozen environment only inside its verified operator
callback. Resource lifecycle owns refusal of silent root replacement.

`OpenInstalledNativeSupervisorImage` opens approved installed code authority
without requiring the caller to be the running manager. The current-supervisor
variant additionally proves its actual manager slot and mapped image.
`OpenOwnedServiceRootImage` and Windows `OpenOwnedServiceChildImage` retain exact
birth-established executable identity. Windows child claims include the
inherited read-only ImageHandle; image observation also verifies the actual
same-operator non-elevated token, exact invocation Job and retained wrapper.
The trusted wrapper holds Job custody; vendor children receive no Job handle.
The executable lock does not replace immutable dependency-closure validation.

The unoperated split-root/dedicated-account/restricting-token candidate designs
are retained as noncompiled research under .vrooli/work/native-portability-history/platform. They are not fallback
execution modes or historical deployed compatibility requirements. Linux's
existing contracts and omitted-Native identity encoding remain preserved.

Hermetic fixtures and crossbuilds establish source behavior only. Actual Apple
SDK/CGO APIs, user-domain readiness before login, Task Scheduler/S4U account
policy and action topology, Job handle inheritance, crash/logoff/cold-boot
recovery, sandbox transport/fork denial, real HID permissions and physical
headless lighting remain target qualification. No native installation or device
effect was performed by this source lane.
