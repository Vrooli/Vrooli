# Actual-operator high-level migration checkpoint — 2026-11-11

Changed: native installed entry calls the ordinary RuntimeSupervisor directly;
removed the high-level split parent/worker/resource-RPC composition and fixed
quiesce/acquire private commands from compiled product. Prior source/tests are
retained here as `.txt`. The maintained bounded source pointer remains product
code. RuntimeSupervisor no longer accepts NativeSystemBroker run options.

Ordinary native supervisor startup and Setup/Enable/Disable reject elevated
execution before DB or scenario/source effects. Explicit Prepare remains a
separate elevated disabled-intent operation. Its pipeline is sealed install,
trusted host configuration, device preparation, exact cached software staging,
then launcher publication. Ordinary Setup invokes SDA as the operator, and an
elevated acquisition attempt fails before resolving/executing the user CLI.
Missing native publication authority maps only the typed elevation sentinel to
public `runtime lighting prepare --json`; corrupt artifacts retain their errors.
Launcher publication compares immutable installed/staged content digests and
uses the resource owner's installed-manager-image authority (no nominated image).
Windows device Apply/rollback now uses ProcessSecurity.Elevated for its disabled
fence; Unix-only geteuid does not stand in for Windows elevation.

Verified after migration: focused race suite in app/runtime, cmd/vrooli and
runtimehandlers PASS (1.496s/1.039s/1.098s) in
/tmp/native-simple-final-race.log. Tests cover identity refusal before any owner
effect, ordinary Setup/SDA without privileged preparation, hostile user SDA not
executed as root, prepare ordering/failure/retry, disabled fencing, immutable
launcher digest binding, source pointer bounds/alias/FIFO, native verified image
activation guard and fixed hidden entry parsing. Darwin amd64 and arm64 CGO0
root CLI crossbuilds PASS (/tmp/native-simple-darwin-*-build.log). Commands set
both source environment variables to the isolated checkout and use offline Go
modules plus -buildvcs=false. No generated files or dependencies were edited.

Windows amd64 CGO0 root CLI crossbuild also PASS after the device adapter
landed (/tmp/native-simple-windows-build.log).

Remaining: Native platform backend/capsule ownership is still
being migrated by sibling owners. Parent owns fresh integration review and
crossbuild reruns after those coherent changes land, plus broad owner tests and
persistence-stamp qualification. Retired platform operator APIs/docs are the
platform owner's archive/removal responsibility; no compiled high-level caller
remains for them.

Unverified: actual native Task Scheduler/launchd execution, boot while logged
out, S4U logon policy, native process lineage, backend containment, device access,
physical lights, suspend/resume and recovery on target hardware. This lane made
no service/account/token/logon/host/device effects. Source/build proof is not
native runtime qualification.
