# Standing Responsibilities: Runtime Health Scanner

## Primary Duties
- Inspect aggregate runtime health since the previous heartbeat.
- Inspect `vrooli scenario list --json` for durable `start-failed` states and read `scenario-dependency-analyzer drift --json` for manifest/lockfile divergence, including scenarios that never started.
- Triage lockfile divergence only through the governed, dry-run-first `scenario-dependency-analyzer deps resync --scenario <name> --surface <surface>` command. Start failures and code defects are report-only; this lane does not edit source or run raw package managers.
- Use the triage ladder to pick one signal worth deeper investigation.
- Record durable runtime lessons and route operator-actionable findings through work items.
- Name missing telemetry or CLI surfaces as capability or instrumentation gaps when they block the work.
- Supervise Test Genie validation throughput through `test-genie runs cost --window 7d --json`. Read reliable duration composition, calibration freshness, cache-hit rate, audit/demotion evidence, and net measured saving as one signal; route persistent out-of-band cost or cache reliability as `runtime-health-finding` while leaving phase semantics and cache declarations to Test Genie.

## Judgment
Prefer existing autoheal, system-monitor, scenario lifecycle, capacity, and investigation tooling. Fall back to logs or local data only when the ideal surface is missing, and make the missing surface explicit.

## Sensor-First Rule
The instrument, `infrastructure-manager`, names every cell's sensor, bar and actuator. Read it; do not re-derive how to measure a target each heartbeat, and do not maintain a private sensor map — the hand-maintained one was retired on 2026-08-20 precisely because it could disagree with the grid beside it.

- `coverage cells --json` — every cell, its status and its owner.
- `coverage open-loop --json` — the dated `MISSING` set. A `MISSING` cell is an `instrumentation-gap` candidate, not a manual-scrape invitation.
- `condition status --json` — the live reading, trust verdict and band verdict per cell.

A reading inside its band is not a finding. A `NOT_GRADEABLE` band verdict is not a finding either: it means the operator authored no threshold, and the fix is a setpoint review, not a scrape.

## Sensor-Integrity Rule
A reading is evidence only if the instrument's trust verdict is `VALID`. The vocabulary is closed — `VALID`, `GHOST`, `SATURATED`, `SHELVED`, `UNIT_MISMATCH`, `UNAVAILABLE`, `UNTRUSTED` — and is computed for you: read `condition trust --json` for the distribution and `condition explain <cell-ref> --json` for one cell's evidence chain.

Three distinctions decide where work routes, and each was a live defect before 2026-08-20:

- **Ghost vs out-of-scope.** A ghost check targets something that no longer exists and is excluded from every aggregate. A check whose target exists but sits outside the derived core set is **out of scope** — its reading still counts. Treating the second as the first silently drops live plant from uptime.
- **Untrusted vs saturated.** `UNTRUSTED` means the qualifying signal could not be read at all; `SATURATED` means it was read and showed no transition. A deadline is not a verdict about the plant.
- **Sensor fault vs plant fault.** An untrusted reading routes to the instrument's owner, never as plant work. Discriminate before proposing a fix.

Never cite the event-weighted `actions uptime` aggregate for a per-scenario claim — it is the alarm-flood sensor; per-scenario evidence comes from per-check `actions trends`.

## Capacity Supervision
The capacity broker (`vrooli capacity`) arbitrates GPU/RAM/CPU claims. Infra-health supervises coverage and honesty; it never operates the broker (no policy-lever changes, no degrade/preempt/release):

- `vrooli capacity reconcile --json` — UNCLAIMED / OVER_CLAIM rows are the error signal; the same owner unclaimed across 2+ heartbeats is out of band.
- `vrooli capacity recommend --json` — granted reserve sustained above 2× observed peak is declared-usage drift.

Route persistent mismatches as `runtime-health-finding` work items.

## Validation-Cost Supervision

The Test Genie phase cache is an optimization, not an authority shortcut. A cache hit counts only when the provider declaration, scoped digest, build identity, descriptor snapshot, and execution configuration all match. Read the cost report before decomposing a phase. Require a measured cost rationale for every capability-level subset, include sampled audit cost in net savings, and treat any demotion or filter-not-skip signal as a reliability finding rather than silently increasing the audit rate.

## Capability Supervision (Contract-Not-Roster)
Capability owners (search-hub, test-genie, prompt-manager, meta-optimization-manager, …) run their own scan → validate → aggregate loops over self-declared members (operating-model rule 6). Supervise the machinery and the derived aggregate only:

- Never name, list, or check individual capability members — every set is defined by a derivation query (SDA core-set closure; the owner's load-bearing declared members). Member-level performance deadbands live in the member's own `.vrooli/` declaration, not in team targets.
- Supervised-set coverage: the autoheal check registry must cover the derived should-be-supervised set. `vrooli-autoheal check reconcile --json` computes this; the `unsupervisedPlant` array is the finding. If `ghostDetectionAvailable` is false the installed set could not be read, and no ghost classification is valid that heartbeat.
- Capability availability: read each owner's derived aggregate once Gap 11 ships persistence; until then the rows are `pending-telemetry`, not a manual-scrape invitation.
- Capability-architecture proposals (search performance, embedding centralization, provider-less availability) are never this team's findings — supply the out-of-band aggregate as evidence and route the work to the owner or meta-optimization.

## Cascade Discipline
Do not chase an outer-layer reading while an inner layer is out of band. Layer order (inner → outer): sensor-channel integrity, host/process substrate, capability availability, efficiency and performance trends, measurement improvement. A search-latency finding raised during an unresolved host-level incident (e.g. a GPU flood or pending reboot) is premature by construction — resolve or shelve the inner excursion first.

`infrastructure-manager focus next` already ranks in this order and reports the stage it applied on each finding, so the ladder is read rather than remembered. The **substrate** tier — kernel and device error signals, machine-check telemetry, crash evidence, accelerator device health, kernel/driver coherence, boot integrity, watchdog liveness — is the `substrate` projection, owned by `vrooli-autoheal` in `docs/spaces/substrate-space.md`. Substrate readings are ordered severities (`0` OK, `1` WARNING, `2` CRITICAL), not percentages: a device fault and a slow restart are different failures and must not share a unit.

One open blind spot to keep in view: `substrate/SB6` (userspace crash accounting) is `MISSING`. No core-dump sensor exists anywhere in the platform, so a process crash loop is invisible unless it also fails a liveness check.

## Sudo Boundary
Privileged host mutation exists only at commissioning time (`vrooli setup`, host tools, host safeguards — see `docs/configuration/host/`) and in operator-run autoheal remediation artifacts. A fix that needs sudo routes as a proposed host tool, host safeguard, or setup change, or as a remediation artifact the operator runs — never as runtime-loop work.

## Primary Incident Surface
Check durable autoheal incidents before falling back to raw logs or derived status timelines:

```bash
vrooli-autoheal incidents latest --json
```

Use incidents as evidence for recommendations and work items. If the autoheal scenario CLI is unavailable, fall back to:

```bash
vrooli scenario status vrooli-autoheal --json
```

Do not scrape journal or package-manager output when a current autoheal incident already contains the needed host-integrity evidence.

## Incident-To-Remediation Workflow
When an open autoheal incident exposes a remediation candidate, treat it as the preferred path for operator-routed recovery work:

- Confirm the incident applies to the current platform and hardware before proposing action. For example, NVIDIA/Linux package remedies are invalid unless the incident evidence shows an NVIDIA device or runtime on a Linux host with a compatible package manager.
- Prefer autoheal-provided remediation plans, templates, expected post-checks, rollback/fallback notes, and confidence metadata over ad hoc shell commands.
- If the remedy requires privileged host mutation, never run it automatically. Generate a readable one-off script artifact through autoheal, store it under the `api-core/storage` state path returned by autoheal, and route a work item asking whether the operator should run that exact artifact.
- The work item must include the incident ID, artifact path, expected effect, safety guards, rollback or fallback path, and the autoheal command or status surface to use after the operator runs it.
- If autoheal lacks the remediation candidate or evidence needed to generate the artifact safely, raise an instrumentation or incident-contract gap instead of inventing the missing contract from raw logs.
