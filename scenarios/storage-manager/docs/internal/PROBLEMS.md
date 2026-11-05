# Problems — Storage Manager

## Setup-owned temporary observation — 2026-10-08

The reviewed standalone observer installer conflicted with the project's
setup-managed privilege boundary. The existing broker also failed to honor
setup's interactive elevation policy. Repair belongs in the control plane:
setup installs/upgrades the existing broker, which observes temporary activity
for its authenticated Unix peer without accepting a path, UID, or executable.
Storage Manager consumes that typed observation and never invokes sudo.
Ordinary unelevated setup must remain successful with honest capability status;
setup with elevation consent must provision the capability. Validation covers
closed request shapes, peer identity, incomplete observations, cancellation,
response bounds and setup elevation routing.

Implemented through the existing broker and shared transport; focused race
tests and setup's live dry run pass. The installed host broker predates this
capability and setup status reports it as outdated. The operator's next ordinary
setup with elevation consent upgrades it; no separate grant command remains.
Managed log rotation also now forwards its closed log subject instead of
dropping it, with a negative-before-positive transport regression.

Post-operator setup verification: the upgraded root broker advertises the
capability, but live observation returned `temporary_activity_incomplete`.
The root cause remains unconfirmed. Follow-up handling recognizes procfs
`ESRCH` as process exit and returns a redacted process/read-stage diagnostic
for remaining failures. Root and affected storage race tests plus Test Genie
`20261008-180952-7911457d` contracts/unit pass. Deploying the follow-up root
service still requires interactive authentication in normal project setup;
the agent's attempt stopped at that boundary. Automatic temporary cleanup
remains unarmed until live observation is qualified and policy is granted.

Verified after the operator's next project setup: the live broker returned
`completed`, UID 1000, `complete=true`, both `/tmp` and the Go work root,
and 1,001 protected paths at 2026-10-08T18:21:12Z. Permission provisioning
and live activity observation are now qualified on this host. The original
failure's exact process was not captured. Recurring deletion remains a
separate operator policy grant. Receipt: `observer-after-second-setup.json`
in the investigation evidence directory.

Automatic temporary retention is now enabled at the operator's clarified
direction. The first scheduled cycle completed and reported 2,462,419,240 bytes
reclaimed. The guarded temporary provider now defaults to automatic approval
in the balanced profile; conservative mode stays disabled, and persisted
operator choices are preserved. This resolves the remaining automatic-policy
gap for this host.

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-07-08 — Named regression baseline missing

**Symptom:** `git-control-tower baseline snapshot status --scenario <target> --name storage-manager-scenario-implementation-baseline --wait --json` reports `status: "missing"` for storage-manager, system-monitor, vrooli-autoheal, and workspace-sandbox.

**Root cause:** The plan names a regression anchor that has not been registered as a baseline manifest or snapshot intent in git-control-tower.

**Workaround:** Use the plan's fallback anchor: HEAD SHA `7b01bc3140c1baf470021ae2c6cb536eb34be1dc` plus allowlist diff inspection until a named baseline is registered.

**Real fix:** Register the named baseline through the owning Plan Manager or git-control-tower workflow before relying on scenario baseline diffs as the pass/fail oracle.

**Owner:** unassigned.

**Refs:** `~/.vrooli/plans/storage-manager-scenario-implementation.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## Work ladder

- Scoped repair, 2026-10-08: W3 implementation. The operator authorized fixing
  the observed cleanup failures and reducing reliability debt. Existing
  OT-P0-001/004 and OT-P1-001/003 establish safety, apply receipts, temporary
  providers, and monitoring handoff. The prior investigation retained the
  ten-second temporary scan failures, rejected owner approval modes, and missing
  supervision under `~/.vrooli/plan-artifacts/disk-investigation-20261008/`.
  Acceptance is recorded in the operations runbook. This scoped repair does not
  claim a full W0-W2 maturity assessment or certification.

- Rung: W0
- Evidence: `swarm-manager goals list --json` returned no goal whose name, title, or description mentions `storage-manager`; the active user goal is external to swarm-manager. The scenario PRD/requirements therefore cannot be compared against an owning scenario goal through the deterministic W0 gate. Rechecked 2026-09-02.
- Blocker: contract truth is unverifiable until a named swarm-manager goal exists; implementation work may continue against the explicit user plan, but W1-W3 scenario-ladder gates must not be treated as contract proof.
- Measured: 2026-09-02


## Owner retention enforcement — 2026-09-05

Generic retention now defers storage entries declaring `reclaim.pruner: custom`, and does not overwrite their owner-wide enforcement receipt after a partial generic pass. This preserves live-work checks for BAS evidence and desktop staging. Focused `TestEnforce*` regressions pass. Test-genie unit run 20260905-050716-b2e745dc was submitted and remains server-owned while queued validation completes. Runtime activation was not performed as part of this code change.


Activation verified 2026-09-05 05:41 UTC: service restarted and reports healthy/ready. A missing generated-proto checksum was repaired through `scenario-dependency-analyzer deps install`, followed by passing dependency governance validation. Owner cleanup completed and the normal schedules were restored; generic custom-entry deference remains active.

### 2026-09-12 — Shared Go cache recovery deleted build inputs

**Symptom:** Go compile/link/vet reported missing shared-cache artifacts during
otherwise passing source builds. Recovery receipts `recovery-962049-1789170239922466140-75`
and `-76` reclaimed 1,128,742,621 and 969,941,935 bytes at 16:11:59 and 16:14:01 UTC.
Their `go-build-cache` apply audit events are `audit-1baa9bbf9cfcf523` and
`audit-1d061b76aab7543f`. The receipts identify the deleting provider and root, not
each missing artifact hash.

**Root cause:** A false regenerable/no-lease declaration let RATE recovery
override policy/age protection and delete whole cache shards. Go permits
concurrent builds on a local cache; external deletion violated its output-use
lifetime. The triggering pressure warning was about 74.5% disk use, not disk full.

**Workaround:** No ordinary policy toggle or perpetual new GOCACHE directory is
a safety repair. Coordinate the canonical Storage Manager lifecycle rollout;
source-only changes do not contain the running reclaimer. No restart or cache
deletion was performed by this repair.

**Real fix:** The declaration now requires owner evidence and the canonical
provider refuses preview/apply, including old-plan replay and recovery overrides.
Future external reclamation needs a tool-owner build-use/quiescence protocol;
mtime and open-handle snapshots are insufficient. Runtime adoption remains a
separate operator-owned step.

**Owner:** Storage Manager provider owner; control-plane owner for lifecycle adoption.

**Refs:** [Safety invariant](../reference/recovery-ledger.md#shared-go-build-cache-safety),
`api/internal/providers/file_provider.go`, `spec_provider_test.go`,
`.vrooli/repo-contract.json`; prior memory record
`source-ledger.agent-memory/4ac20388-16fd-4867-89e9-de3b681bb995` correctly identified
churn but its external mtime-cap recommendation lacked active-build safety.


## Work ladder — 2026-10-13T23:21:28.406231+00:00 — checked closure and split bootstrap

- Rung: W3, shared static-analyzer defects blocking Nooch's existing storage consumer.
- Hypotheses: real cursor/schema omissions versus unrecognized valid Go/API forms. Nooch's populated bootstrap/reopen regression and the actual checked Close call sites contradicted an absent schema/cleanup explanation. Minimal analyzer fixtures reproduced missed assigned/if-initializer Close and qualified ApplySchemas followed by ReconcileDeclaredColumns or VerifyDeclaredColumns.
- Repair: recognize unconditional assignment/if-initializer cursor cleanup; share a narrow qualified split-bootstrap recognizer between schema application and routed seam analyzers. Require ordered calls with the same context, pool and providers in one function. Incomplete, reversed, conditional, different-pool/provider, unrelated-import and different-cursor cases remain refused. No Nooch query, schema declaration, gate severity or threshold was changed for this repair.
- Validation: new positive regressions failed before implementation, then passed. Full internal/validation package passes. Storage-manager unit run `20261013-231238-5ce819c4` PASS. Fixed the combined file-registration regression that had been accidentally embedded in another fixture; it now executes and passes.
- Serving and consumer: supported storage-manager restart `startop-d2b2b082895e4b916217903111ca9c7d` reached healthy. Live Nooch validation has zero errors and two SQL_DB_HANDLE_CAPTURE warnings at startup-only function parameters. Native Nooch storage run `20261013-231714-b3c867ad` PASS. This closes the observed scanner failures at the original storage consumer; it does not accept Nooch's product, image, authentication or restore outcomes.
- Tracking: original QA report `knw-1791916274834471865`; QA closure remains the investigator's operation. Current native team continuation consent is pending and separate.

Final adversarial tightening: return-before-reconciliation and pool rebinding regressions also pass; final storage-manager unit run `20261013-232826-4ca9559f` PASS, final deployed detector build `c878fd39…` healthy and live Nooch validation still has zero errors/two warnings. Summary: `docs/internal/delivery-operations/2026-10-12/sources/execution-evidence/2026-10-13/storage-repair-final-summary.json`.


## Work ladder — 2026-10-16 — zero-byte temporary recovery starvation

- Rung: W3, localized implementation defect encountered during operator-authorized disk cleanup.
- Prior art: reused the 2026-10-08 bounded-recovery investigation and temporary observation qualification above.
- Evidence: the normal census offered 258.52 GiB of temporary candidates, while manual recovery reclaimed 28,618 temporary bytes and then reported no eligible bytes. Three read-only bounded observations repeatedly returned `/tmp/deep-night-sky-types.log` with zero bytes. The setup broker reported complete activity evidence and no held handles beneath a known expired cache; the strict filesystem reader measured that cache successfully. A canonical service refresh did not resolve the discrepancy.
- Cause: the resumable provider cached the first fully measured empty file as its ready unit. The controller requires positive estimated bytes before apply, so it never consumed that unit or reached the positive-size directories behind it.
- Repair: advance past complete zero-byte units before publishing a recovery candidate. Keep age, ownership, identity, activity, containment, special-file and lease checks intact.
- Validation: `TestNativeTemporaryRecoveryAdvancesPastEmptyFiles` failed before the repair and passed afterward. Focused native temporary recovery and provider safety regressions pass. Scoped Test Genie contracts/unit passed as `20261017-030247-90d28b56`. The exact-item selection extension passed contracts/unit as `20261017-030800-d792a6b0`; its CLI transport regression also passes with the production protobuf wire format. Live cleanup receipts are retained in the investigation evidence directory.
- Evidence directory: `~/.vrooli/plan-artifacts/disk-investigation-20261016/`.


### Exact selection for the approved cleanup pass

The operator approved reclaiming temporary artifacts and eligible Trash. The
whole-plan apply endpoint could not isolate those items from the model and
history providers included in the same census. Added optional exact item IDs
to the existing apply contract and CLI. Unknown or ambiguous IDs reject the
request before any provider runs; selection retains the original preview and
all provider safety gates. Idempotency keys bind both plan and normalized item
selection. Empty selection preserves whole-plan behavior. This permits selecting
large temporary units directly without waiting behind thousands of tiny files
or granting model/history deletion. No retention profile or standing authority
was changed.
