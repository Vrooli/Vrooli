# Native operator supervisor composition

Darwin/Windows explicit machine installation hashes the maintained CLI, resolves
its original operator from native account metadata, and publishes an immutable
boot image. The native manager executes the existing RuntimeSupervisor directly
as that actual non-elevated operator. The historical SystemBroker API name does
not authorize a privileged runtime loop. Linux and explicit user installation
retain their existing paths.

Darwin uses a system LaunchDaemon with the selected UserName/GroupName. Windows
uses the native Task Scheduler BootTrigger with S4U and LeastPrivilege. S4U is a
local-only account context: network credentials, an unlocked keychain, an
interactive session, and arbitrary account-policy readiness are not assumed.
Native logon/account-policy failures remain errors, without another identity or
interactive-token fallback.

The fixed installed entrypoint verifies native manager/account/image authority,
pins compiled runtime-home authority, and checks activation using the verified
image digest before entering the ordinary supervisor. The same operator owns
SQLite and its sidecars, lease renewal, scenario/source execution, demand cleanup,
lighting reconciliation, and update recovery. There is one roster and one loop;
no root process interprets operator-writable PID records or executes source.
The maintained bounded source-root pointer supplies checkout location independently
of the manager working directory. Alias, FIFO/nonregular, replaced inode,
relative/noncanonical path, and over-4096-byte pointers are refused.

`vrooli runtime lighting prepare --json` is explicit elevated machine publication.
It holds the disabled-intent fence and validates target support before effects.
It publishes the sealed boot CLI, configures the exact maintained backend host,
prepares native device ownership, reopens and verifies the pinned operator cache,
and publishes immutable software/launcher artifacts. It does not enable lighting
or launch the backend. Launcher publication uses the exact installed-manager
image authority, with digest equality to the trusted installation result.

Ordinary Setup acquires software through the existing operator-owned Scenario
Dependency Analyzer gateway. Elevated setup never executes that user-owned CLI.
When immutable publication is required, the supported elevated prepare operation
runs after acquisition; ordinary Setup can then reuse the exact sealed artifacts.
Setup/enable/disable refuse elevated native lifecycle execution before reaching
operator-owned scenario scripts. Elevated device Apply/rollback use their existing
public exact-plan operations and hold the disabled-intent fence.

Targets without an official governed asset retain truthful degraded status and
an explicit launch blocker, without hardware probing or a fabricated selection.
Setup and Enable refuse those targets before machine effects.

The retired split root-parent/private-worker/RPC research and its prior source
fixtures are preserved under docs/research/20261111-split-native-supervisor as
noncompiled text. They describe an abandoned design, not the product contract.

Source tests and crossbuilds establish only their stated fixture/build behavior.
Native launchd/Task Scheduler logged-out boot, S4U policy readiness, real process
lineage, device permissions, recovery, physical Off/static and suspend/resume
remain target qualification. This source lane performs no native account, token,
service, device or hardware effects.

The existing protected boot descriptor also freezes the optional whole-resource
`VROOLI_RESOURCE_STORAGE_ROOT`. Absence of enrollment and an enrolled default root
are distinct. Startup applies the frozen value only after native actual-operator
identity validation. Public lighting actions, supervisor installation/removal and
resource lifecycle validate the current root before intent writes or quiescence.
Changing the ambient root cannot acknowledge a disabled backend in a different,
missing namespace; restore the enrolled value (or unset the override for an
enrolled default) before normal cleanup. No automatic root migration is implied.
Readonly enrollment observation accepts only the fixed protected native definition,
its exact immutable descriptor/image and selected operator/home association. Only
absence of that fixed unit/task means unenrolled; descriptor/image/query failures
remain errors. Native definition replacement during observation is refused.
