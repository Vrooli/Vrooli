# Phase C2 — Native evidence permanence

Date: 2026-08-25

## Decision

Native qualification is now represented as durable, typed evidence rather than
an implicit property of a runner name. The portability grid exposes every
record through `native_evidence`. Broad CI evidence remains useful for the
runner surface that produced it; hardware-persistence evidence may name an
explicit host and a capability subset.

For a capability and host/architecture cell, the newest applicable native
record is authoritative. A failed newest record removes a stale qualified
rung and retains the implementation at `build_verified`. An unrelated
capability-scoped record cannot qualify or decay another capability. A broad
record with no capability list applies to all capabilities, matching the
legacy Unit Health evidence format.

The vocabulary contains a narrow dated `hardware-persistence` policy ceiling
of `build_verified` when no dedicated native machine exists. This is metadata
about hardware-persistence evidence, not a platform-wide ceiling. It is
reviewable by 2027-09-01, before the expected Fall 2027 Intel macOS runner
retirement.

## Implemented surfaces

- `NativeEvidence` is added to the infrastructure-manager portability proto,
  API projection, JSON ledger, and generated platform-support document.
- `tools/portability-evidence` writes schema-versioned records atomically and
  requires an explicit pass/fail verdict.
- `.github/workflows/portability-hardware-validation.yml` performs scheduled,
  explicit-node Bridge lifecycle validation for `minimouse`, writes both pass
  and fail records, uploads its transcript, and attempts to persist the record
  through the repository workflow.
- The existing `cross-platform-lifecycle` and Unit Health jobs retain the
  required `macos-latest` and `windows-latest` lifecycle surfaces. Unit Health
  emits parser-compatible native runner evidence for the CI matrix, including
  Darwin and Windows target architectures.
- The platform-support generator renders the current native evidence list and
  the checked-in document records the narrow hardware-persistence policy.

## Evidence

The following local validations passed:

```text
make -C packages/proto generate SCENARIO=infrastructure-manager
go test ./tools/portability-evidence ./internal/deployability
go test ./internal/portability ./handlers/portability
go test ./...                         (scenarios/infrastructure-manager/api)
go test ./...                         (scenarios/infrastructure-manager/cli)
yq e '.' .github/workflows/portability-hardware-validation.yml
bash -n docs/reference/generate-platform-support.sh
```

Focused grid tests prove failed-record decay and capability scoping. The
tool smoke test produced a valid failed `hardware-persistence` record with a
sorted capability list. Schema JSON parsing and the platform-policy tests
also passed.

## Remaining external verification

This artifact does not claim that the scheduled workflow has already run or
that repository branch protection accepts its evidence commit. Those require
GitHub workflow/secret and branch-policy access.

The generated ledger was initially blocked after the proto/API changes because
the UI's frozen pnpm install rejected a package.json/lockfile mismatch for
`@vrooli/react-component-library`. The mismatch was repaired through the
governed dependency analyzer's approved local-file install path. A governed
restart then completed successfully and left `infrastructure-manager` healthy
on API port 19432 and UI port 23792. Startup still reports optional dependency
degradation for `kopia` (manual host installation required) and
`vrooli-autoheal`; the infrastructure-manager capability owner itself is
healthy and `vrooli capability fleet --json` reports `available: true`.

The separate C1 hardware ladder remains pending. During this investigation,
`minimouse` briefly went offline while the local Bridge API was being
restarted; a typed remote relay attempt ended with `unexpected EOF`. The node
was not unregistered or revoked. Its registry record and Ed25519 heartbeat
identity remained present, and the latest check shows it online,
heartbeat-fresh, channel-held, protocol-compatible, and dispatchable.
`nodes doctor` passes. A read-only remote status request also reached the Mac
successfully and reported `system-monitor` stopped; that is a scenario state,
not node offline evidence.

The authoritative current node ID is
`25c7e426-c76c-421a-8351-aaf964589802`; an earlier investigation note
mistyped the fourth UUID group as `4217`, which caused misleading targeted
`get`/`doctor` not-found responses. The Machine lineage and registry row agree
on `421a`.

The installed node reports revision
`872271a917a1117d17dd1897037ef21030e86548+dirty`, which predates the current
agent changes. The agent's startup encryption-key registration and credential-
grant synchronization previously treated temporary control-plane transport
errors as terminal. The agent now has bounded retry logic for those preflights,
with tests passing locally; that fix has not yet been installed on minimouse.
This is a resilience mitigation, not C1 hardware qualification evidence.

Bridge cleanup still correctly refuses to create an operation until the
node's X25519 encryption credential is provisioned. The heartbeat credential
and durable node identity remain healthy. No cleanup, revoke, re-pair, reboot,
or destructive reset was performed during the investigation.

The C1 producer-owned validation ticket was started and its baseline oracle
re-anchored against the current source collection. Its first dispatch was
not-ready because child processes resolved `/usr/local/bin/vrooli` to a shim
whose per-user root binary had disappeared, causing Test Genie fast-skips for
five captured members. The root CLI and required scenario CLIs were restored
through governed setup/install paths; this result is retained as invalid
validation evidence rather than being treated as a pass. The repeated loss of
installed sibling CLIs during setup activity is a local lifecycle/tooling
reliability issue that must be resolved or guarded before a final full-plan
validation run.

The current local toolchain itself reports the `kyutai-stt` managed-service
manifest as valid when invoked through the freshly installed root CLI; an older
checkout-local `./vrooli` binary produced the stale `compose step kind "local"`
diagnostic. Future validation must use the governed installed root CLI, not a
stale repository binary.

The canonical plan baseline was then repaired through the governed Plan
Manager update path (`--baseline-mode current --anchor-strategy
change_boundary`). It now contains all 11 affected scenario targets and 43
boundary paths. The prior producer ticket remains retained as an invalid
not-ready attempt; it was not replayed. A new ticket was created after the
toolchain repair, and its producer collection diff reached Test Genie
admission saturation with `structure-health` unavailable. The ticket remains
queued until that producer capacity/provider condition is resolved.

## Root CLI disappearance investigation — 2026-08-25

The repeated disappearance of `~/.vrooli/bin/vrooli` was investigated as a
separate local tooling incident because it invalidated the first C1 producer
attempt. Prior-art recall found no matching record. The searches were:

```text
search-hub query "vrooli root executable ~/.vrooli/bin/vrooli repeatedly disappears or is unlinked during scenario lifecycle and validation" --type record,doc
swarm-manager scenarios fixes --name vrooli-bridge --all --search "vrooli bin unlinked missing root executable install rebuild race"
prompt-manager discover "trace root CLI unlink/rename during lifecycle" "coordinate stale rebuild with project install" --type all
```

The first two returned no matching prior fix, and the search-hub providers were
partly degraded (source-ledger and swarm-manager returned HTTP 500 while the
template-manager provider was unreachable). The swarm-manager CLI also does not
provide the `ai-search` command described by the current skill text.

Three hypotheses were tested:

1. A stale-check rebuild and `make install` could replace the same root binary
   concurrently.
2. Scenario cleanup could mistake `vrooli` for a scenario CLI and remove it.
3. An unrelated lifecycle/toolchain process could remove or replace the install
   directory.

The second hypothesis is rejected by the source guard in
`internal/cliinstall`: root names are explicitly refused by
`RemoveScenarioCLIReport`. Controlled `strace` runs of `make install`, scenario
setup, and governed scenario start showed no root unlink. The third hypothesis
was not observed in the controlled traces and remains the fallback investigation
path if the problem recurs outside these operations.

The first hypothesis exposed a real coordination defect. Stale self-rebuilds
already used `<binary>.lock`, but `AtomicInstall` did not use that lock. Both
paths performed atomic replacement independently. That can produce the observed
old-process view `/home/.../.vrooli/bin/vrooli (deleted)` after its inode is
replaced, and it can cause repeated replacement/rebuild churn. It does not by
itself prove that an explicit `unlink` caused every earlier missing-path
observation, so the exact historical deletion event is recorded as unconfirmed.

The fix centralizes the binary install lock in `internal/buildinfo` and makes
both `RebuildAndReexec` and `cliinstall.AtomicInstall` acquire it. A regression
test first failed because `AtomicInstall` returned while the shared lock was
held; it now passes after the fix. A second regression test proves that a stale
sibling process recognizes a fresh installed fingerprint sidecar and skips a
duplicate rebuild.

Focused tests pass:

```text
go test ./internal/buildinfo ./internal/cliinstall -count=1
go test ./internal/buildinfo ./internal/cliinstall ./cmd/vrooli-atomic-install ./cmd/vrooli
```

An integration stress run launched a forced stale self-rebuild and `make
install` concurrently. The self-rebuild command exited because the local API
was unavailable, while `make install` succeeded and the root executable
remained present and executable. The installed root binary, shared lock, and
fingerprint sidecar were then verified on disk. Future recurrence triage should
capture a filesystem event trace around the exact timestamp; the shared lock
now ensures any legitimate replacement is serialized and leaves a complete
binary at the canonical path.

## Follow-up incident and bridge presence correction — 2026-08-25

The root-path failure recurred immediately after the server-owned
`vrooli-bridge` test run. At that point `~/.vrooli/bin/vrooli`,
`plan-manager`, `git-control-tower`, and `scenario-dependency-analyzer` were
absent, while `~/.vrooli/libexec/vrooli.previous` remained present. No
artifact-removal receipt was written for the missing paths. Retention receipts
were created in the same minute, so an older retention/cleanup process remains
the leading historical actor hypothesis, but the exact unlink syscall was not
captured because this user cannot attach `strace` or install an audit probe.

The live processes explained why source fixes had not protected the running
host: Storage Manager had started before the cleanup-root exclusion was
installed, and the systemd-user runtime supervisor was executing a deleted old
`vrooli` inode even though its unit pointed at the canonical path. Storage
Manager was restarted through the lifecycle and its live provider catalog now
contains no `runtime-home-bin` provider. The exact supervisor PID was then
restarted by its existing systemd `Restart=always` policy; its executable now
maps to the current canonical inode. The two interactive agent processes that
still hold old deleted inodes were intentionally left untouched.

The missing helper CLIs were restored with governed setup. The root executable
and fallback remain present after restoration, and focused Storage Manager
retention/cleanup/provider tests pass.

Separately, minimouse was not offline during the sustained period reported by
the operator. The Bridge SSE handler acknowledged the stream before registering
`Hub.Connect`, creating a short observation race in which an already-live TCP
node could appear offline. Presence registration now precedes HTTP
acknowledgement, with a focused regression test; bridge channel and presence
tests pass. The current node readout is online, heartbeat-fresh,
channel-held, protocol-compatible, and dispatchable. A read-only remote
`system-monitor` status request also reaches the Mac and reports the scenario
stopped, which is distinct from node absence.

## Follow-up recurrence and controlled attribution attempt — 2026-08-25 18:00–18:06

The recurrence was confirmed again after a successful supported setup: the
canonical root and selected CLIs were present at different points, but the
set changed concurrently. `plan-manager` disappeared completely, while
`git-control-tower` and `scenario-dependency-analyzer` were later restored;
scenario APIs remained running throughout. This is real path absence, not
only an `lsof +L1` view of an old inode.

A traced setup run (`strace -ff -e trace=unlink,unlinkat,rename,renameat,
renameat2`) completed successfully and showed only expected temporary-file
cleanup and atomic replacement. It did not unlink the root or selected CLI
targets. An event-driven inotify watch during concurrent validation likewise
observed temporary cleanup and atomic renames, but no target deletion. Storage
Manager logs contained no cleanup Apply or reclaim operation, and the removal
receipt directory remained empty. The external actor is therefore still not
identified; several concurrent agents were active, including deliberate
validation/reproduction work.

One additional control-plane gap was fixed: native resource-CLI uninstall
used direct `os.Remove` calls outside the artifact ledger. It now rejects
protected control-plane names and removes the binary/manifest/metadata triple
through the shared ledger lock and predicate, then clears its lease. Focused
`internal/resources`, `internal/artifactledger`, `internal/cliinstall`, and
`internal/buildinfo` tests pass. Older installed CLIs without lease sidecars
remain legacy artifacts; newly installed plan-manager and test-genie records
have leases.

## Definitive retention attribution and live mitigation — 2026-08-25 18:37–18:59

The previously unresolved unlink actor was identified by correlating the
filesystem watcher with Storage Manager access logs. The mass deletion began at
18:37:08 while a `GET /api/v1/retention/owners` request was in flight; that
request started at approximately 18:36:56 and completed at 18:38:41 after
1m44s. The watcher recorded 68 confirmed removals from `~/.vrooli/bin`,
including `plan-manager` and its sidecars. This was not the traced CLI
installer: its trace contained only temporary-file cleanup and atomic renames.

The root defect was that a read-only retention-owner endpoint synchronously
called the destructive storage-entry enforcer, while the older running
Storage Manager had a broad runtime-home/bin retention declaration. A second
defense is now present in the control plane: retention enforcement refuses the
shared executable install root and every overlapping ancestor/descendant path,
even if stale inventory data reaches it. The retention-owner and budget-health
GET handlers are now read-only and no longer invoke enforcement.

Focused tests pass:

```text
go test ./internal/retention ./handlers/storage
go test ./retention -run 'Test(DirectoryPrunerRefusesProtectedRootAndBroadAncestor|FilePrunerRefusesProtectedExecutable|DirectoryPrunerRequiresAnAbsolutePath)' -count=1
```

Storage Manager was restarted through `make restart` and became healthy. A live
`GET /api/v1/retention/owners` returned HTTP 200 in 1m14s with the install-root
entry count and sorted-name hash unchanged (`81 → 81`, identical SHA-256); no
new watcher deletion event occurred. `plan-manager` and `vrooli-bridge` were
then restored through their supported scenario `make start` paths. This closes
the observed Vrooli-owned deletion path; arbitrary external deletion remains
outside application control.

## Durable lifecycle and presence correction — 2026-08-25 19:27 EDT

The Bridge node remained online while the remote lifecycle investigation was
being repeated. The authoritative node record reported a fresh heartbeat,
held channel, compatible protocol, and `dispatchable: true`. The earlier
offline report was a Bridge presence/read-model race, compounded by treating a
stopped remote scenario as node absence. Presence registration now precedes
the SSE handshake acknowledgement, with a focused regression test.

The same investigation found a separate timeout-propagation defect. A project
CLI lifecycle request with a 600-second scenario timeout was still using the
Bridge relay client's 120-second default transport deadline. The project CLI
now mirrors the forwarded timeout to the outer relay request, and the relay
client honors its explicit timeout when constructing the HTTP transport.
Focused tests cover both seams. A durable dispatch then reached minimouse and
returned a concrete remote build error: the Mac checkout's `system-monitor`
UI imports `features/logs/pages/LogsPage`, but that tracked source file is
missing from the remote checkout. This is stale/incomplete source state, not
offline hardware evidence.

The permanence workflow is therefore structurally present but not yet
qualified: `.github/workflows/test.yml` contains required macOS and Windows
control-plane lifecycle jobs; the scheduled Bridge workflow records pass/fail
hardware evidence and uploads its transcript; the capability vocabulary
contains the dated `hardware-persistence` ceiling reviewed by 2027-09-01; and
the platform-support generator passes `--check`. The ledger still has zero
native qualification records until C1 supplies a successful clean-host,
reboot, credential-store, and lifecycle evidence record. The safe options are
an authorized clean reset/onboarding, or a governed source synchronization
followed by lifecycle validation (the latter repairs service capability but
does not by itself satisfy the clean-host claim).

## Independent gate recheck — 2026-08-25 19:37 EDT

The post-repair standing gates were re-run locally:

```text
vrooli capability conformance       checked_targets=598 findings=0
go test ./internal/deployability/... -count=1  PASS
vrooli hygiene --fail-on error       PASS (0 blocking issues)
vrooli capability fleet --json       dockerBlocked=[] peerless=[]
docs/reference/generate-platform-support.sh --check  PASS
```

The fleet output still contains expected dependency-ineligible rows for
capabilities that are deliberately unavailable on particular platforms; it
contains no self-blocking system-monitor or vrooli-bridge row. The platform
skip budget remains 155 for Linux, macOS, and Windows, with over-budget runs
explicitly ineligible as evidence. The ledger still records zero qualified
cells and zero native evidence records because the C1 Mac ladder has not been
completed. Plan Manager's durable C1 validation operation
`e78a4bbc-c67e-4081-9092-924df7fa0831` could not attach its producer diff:
the required Git Control Tower wait returned `baseline not found` even though
the captured collection itself reports 14/14 required members ready. That
validation result is retained as infrastructure friction, not treated as a
passing plan gate.

## Ignore-rule source closure repair — 2026-08-25 19:49 EDT

The concrete remote `system-monitor` build failure was reproduced and fixed.
The source file `ui/src/features/logs/pages/LogsPage.tsx` existed in the local
working tree but was omitted from the working-tree onboarding transfer because
two nested `.gitignore` files used the broad `logs` pattern. The pattern was
changed to anchored runtime-directory rules, and the existing source file is
now included in the repository source closure.

Evidence:

```text
pnpm --dir scenarios/system-monitor/ui build                         PASS
working-tree re-onboarding bootstrap                                 ONLINE
vrooli-bridge dispatch scenario start system-monitor                 PASS
vrooli-bridge dispatch scenario status system-monitor                PASS
vrooli-bridge dispatch scenario stop system-monitor                  PASS
```

The governed `provision sync` attempt was intentionally not used as the fix:
the node is in documented working-tree mode and has no `.git` repository, so
the privileged helper correctly rejected `git fetch` with `not a git
repository`. The supported repair is re-shipping the working tree through
onboarding. The onboarding operation's terminal handoff returned HTTP 404
after bootstrap completed; that is retained as a separate Bridge handoff
defect. C1 remains unqualified until the clean-host, reboot, credential-store,
and complete lifecycle ladder evidence is collected.

## Onboarding contract and headless setup hardening — 2026-08-25 20:00 EDT

Bridge discovery returns the onboarding scenario base URL while the API route is
`POST /api/v2/handoff`. Bridge previously posted to the base URL and received
HTTP 404. `onboarding.HandoffEndpoint` now joins the discovered base URL to the
stable route, with a focused regression test; the fix is covered by the active
Bridge boundary.

Two governed onboarding retries separated setup-mode failures from presence:

```text
bootstrap/pair/service/autostart/online with --skip-setup       PASS
final selection with --skip-setup                               FAIL 127 (CLI not installed)
setup minimal/none/none                                          FAIL 5 (headless launchd safeguard)
```

The shared `emergency_watchdog` safeguard now detects the absence of the
invoking user's GUI launchd domain and returns manual/degraded instead of
aborting the whole setup. Linux unit tests pass and a Darwin test binary
cross-compiles. The next real onboarding retry is the needed evidence that the
headless setup path now proceeds to the onboarding CLI and handoff.

C2 remains structurally present but unqualified: required macOS/Windows
lifecycle workflow definitions and the recurring Bridge persistence workflow
exist, while the ledger still has no successful native or CI evidence record.

## Final onboarding contract convergence — 2026-08-25 20:24 EDT

The complete governed retry reached the remote selection/readiness boundary.
The contract fixes now in place are:

```text
base URL -> POST /api/v2/handoff                         fixed
ignored system-monitor source closure                    fixed
headless emergency/autoheal LaunchAgent handling         fixed
darwin/macOS host token normalization                     fixed
runtime-home CLI and PATH resolution                      fixed
wizard apply -> wizard commit                             fixed
onboarding CLI auto-start                                fixed
```

The final operation passed setup, native CLI build, pairing, service install,
LaunchDaemon autostart, online confirmation, and remote API auto-start. It then
reported six configuration blockers from the real readiness authority. The
node remains online and dispatchable; this result is not an offline diagnosis.

The independent system-monitor lifecycle start run
`40901282-6028-46a9-82b7-f4b439130981` passed, status returned `running` with
`health_status=degraded`, and stop run
`7150d748-4923-4288-a571-e8e72d45abf3` passed. The current remote scenario is
stopped. C2 still cannot be marked qualified because the clean-host/reboot
ladder and durable qualified evidence record remain outstanding.

The device-graph check has a separately reproducible authority result:
`system-monitor metrics devices` is in the derived Bridge vocabulary, while
the current minimouse node lacks `system-monitor:read`. The next qualification
run must either carry that read grant through the authorized onboarding
selection or record why the device graph is intentionally excluded. The
refusal is a scope decision, not a liveness failure.

## Presence false-negative and retention unlink investigation — 2026-08-26

Authoritative Bridge presence for minimouse is online, heartbeat-fresh,
channel-held, protocol-compatible, and dispatchable. The reported offline
state was a Bridge SSE/presence read-model ordering defect; registration now
occurs before the initial flush. Remote scenario state is intentionally
separate from node presence.

The repeated unlinking incident was traced to Storage Manager's read-only
retention-owner endpoint synchronously invoking the destructive retention
enforcer over a broad runtime-home/bin declaration. The endpoint is now
read-only, control-plane roots and the exact artifact triple are protected,
and a post-repair watcher run recorded no new deletions. The remaining risk is
arbitrary deletion outside the application control plane, which requires host
watcher evidence rather than more retention changes.

The workspace-sandbox contract now declares its secondary Go launcher output.
Lifecycle build, staged publish, and freshness consume the same multi-output
contract; the remote workspace-sandbox and agent-manager artifacts are fresh,
and workspace-sandbox reached healthy runtime status.

C2 remains unqualified because external branch-protected CI/native
verification and qualified hardware evidence are still absent.
