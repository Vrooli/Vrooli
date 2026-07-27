# Instrumentation Roadmap

A living list of **stats the infra-health team needs** and the **capabilities that would make them `measured` instead of qualitative**. Modeled on [docs/monetization/evidence/TELEMETRY_ROADMAP.md](../../monetization/evidence/TELEMETRY_ROADMAP.md). Purpose:

1. Make the qualitative-vs-quantitative gap visible.
2. Point at which existing scenario most likely hosts each capability.
3. Provide a grep-able migration list (`REPLACES-MANUAL`) so when a capability ships, the affected prompts and docs can be found and updated.

This file is **not** a build plan. It identifies gaps; swarm-manager and director-swarm decide when to close them.

## Core principles

1. **Extend before creating.** Most needs can be served by adding a verb to an existing scenario CLI. New scenarios are a last resort.
2. **Emit first, query later.** When a scenario ships with a relevant event, it should emit even before a consumer exists. This seeds data without requiring query and emit to ship together.
3. **Honesty flags travel with the stat.** When a stat moves from `pending-telemetry` to `measured`, every consumer (this doc, `RELIABILITY_TARGETS.md`, agent prompts) gets updated in the same decision.
4. **No instrumentation without a finding it would have unblocked.** infra-contrarian will challenge instrumentation-gap entries that don't name a concrete consumer.

## REPLACES-MANUAL migration list

Agent prompts and team docs carry `REPLACES-MANUAL` markers wherever qualitative reasoning would be replaced by a structured query if the capability existed. When a capability ships, search the repo for the marker and update affected prompts/docs.

```bash
grep -rn "REPLACES-MANUAL" docs/infra-health/ scenarios/prompt-manager/store/teams/infra-health/
```

## Capability gaps

Each gap lists: stats unblocked, most likely host scenario, shape proposed, downstream effects, and the priority signal that should activate the build.

### Gap 1: Autoheal history CLI — SHIPPED (substantially)

- **Status:** Substantially shipped; verified live 2026-07-23 (operator session, teams paused). The shipped verb shape differs from the one proposed here:
  - `vrooli-autoheal check history` — per-check history (proposed as `history`)
  - `vrooli-autoheal actions history` — recovery action history (proposed as `heal-attempts`)
  - `vrooli-autoheal actions uptime | trends | transitions | timeline` — aggregates the proposal didn't anticipate
- **Unblocked:** Repeat-failure tier-1 detection, heal-loop tier-2 detection, and trend views are now single CLI calls. Uptime and restart-frequency rows in `../strategy/RELIABILITY_TARGETS.md` moved to `pending-baseline`.
- **Remaining sliver:** resolved 2026-07-23 (round 2) — per-scenario uptime is served by `actions trends` per-check `uptimePercent` (checks map 1:1 to scenarios/resources); the event-weighted `actions uptime` aggregate was re-purposed as the alarm-channel flood sensor in `../strategy/RELIABILITY_TARGETS.md`.
- **REPLACES-MANUAL:** resolved — the previously referenced fallback sections (`runtime-health-scanner/RESPONSIBILITIES.md` "Current Gaps & Fallbacks", `TOOLS.md`) no longer exist in the member contracts; the scanner's contract now points at the sensor map instead of raw SQLite.

### Gap 2: Scenario lifecycle stats

- **Unblocks:** Cold-start latency targets (`../strategy/RELIABILITY_TARGETS.md`), slow-restart trend tier-3 detection, setup-step duration tracking, build-time trends. (Per-scenario uptime, previously folded in here, is served by `actions trends` as of 2026-07-23 — this gap narrows to latency/build/runtime stats.)
- **Most likely host:** Extend the `vrooli` core CLI (the lifecycle owner).
- **Shape required:**
  - `vrooli scenario stats --since=<duration> --json` — per-scenario aggregated restart count, total runtime, longest-uptime streak, mean cold-start latency, last-N-restart timestamps
  - `vrooli scenario stats --scenario=<name> --since=<duration> --json` — single-scenario detail
- **Downstream effect:** Most platform-component targets in `../strategy/RELIABILITY_TARGETS.md` move from `pending-telemetry` to `measured`. Slow-restart-trend tier in runtime-health-scanner becomes mechanical.
- **Priority signal:** activates with Gap 1 since the data sources overlap.

### Gap 3: System-monitor incident export

- **Unblocks:** Investigation-cluster tier-4 detection (same investigation type ≥3× in 7d).
- **Most likely host:** Extend `system-monitor` CLI.
- **Shape required:**
  - `system-monitor incidents list --since=<duration> --json` — list of triggered investigations with type, root signature, outcome, duration
  - `system-monitor investigations stats --since=<duration> --json` — clustering view: investigation type → count + sample IDs
- **Downstream effect:** runtime-health-scanner stops walking `path:investigations/active/` and `path:investigations/results/` directories.
- **Priority signal:** activates when investigation-cluster tier produces ≥1 actionable finding via the directory-walk fallback.

### Gap 4: Internal-package coverage report

- **Unblocks:** Test-coverage dimension grading in platform-code-auditor.
- **Most likely host:** Extend the platform's existing test harness (or add a make target like `make coverage-internal`).
- **Shape required:** `make coverage-internal` produces a JSON or HTML report scoped to internal packages (not scenarios).
- **Downstream effect:** Test-coverage grades become `measured` instead of `estimate`.
- **Priority signal:** activates when platform-code-auditor cannot grade test-coverage with confidence on three consecutive heartbeats.

### Gap 5: Cross-platform CI matrix

- **Unblocks:** Cross-platform readiness dimension grading. CROSS_PLATFORM_LEDGER entries become measurable instead of inspection-only.
- **Most likely host:** Repo-level CI infrastructure (a future scenario or external service).
- **Shape required:** A CI matrix that runs the platform CLI smoke tests on Linux + macOS + Windows on every PR.
- **Downstream effect:** Cross-platform-debt entries gain "broken on which platform: <list>" data automatically.
- **Priority signal:** explicitly tied to deployment-tier-2 (desktop) entering the active roadmap. Until tier 2 is live, ledger-only is fine.

### Gap 6: Setup-flow timing

- **Unblocks:** "vrooli setup end-to-end" target in `../strategy/RELIABILITY_TARGETS.md`.
- **Most likely host:** Extend `vrooli setup` itself to emit per-step durations.
- **Shape required:**
  - `vrooli setup --json` (or a `--report` flag) emits per-step timing and aggregated total
  - Stored historically so trend detection works across setup runs
- **Downstream effect:** Setup-time target moves to `measured`. Regressions in setup time are detectable.
- **Priority signal:** activates when a clean-machine-install workflow is part of the deployment story (tier-2 onboarding).

### Gap 7: Heal-success-rate stat — SHIPPED (confirm on resume)

- **Status:** Surface shipped with Gap 1 — `vrooli-autoheal actions history --json` lists recovery actions. The resume-protocol scan must confirm the field set includes outcome (success / failure / no-action-needed) before flipping the target row to `measured`; until then it is `pending-baseline`.
- **Unblocked:** "Heal success rate" platform target in `../strategy/RELIABILITY_TARGETS.md`.

### Gap 8: Host update-risk forecasting

- **Unblocks:** Early warning before kernel, driver, package-manager, or reboot-required updates create host capability drift.
- **Most likely host:** Extend `vrooli-autoheal` after host inventory and durable incidents have operated long enough to prove the base signal quality.
- **Shape required:**
  - `vrooli-autoheal host updates --json` — safe read-only view of pending host updates, reboot-required signals, kernel/module package pairings, held/broken packages, and known compatibility advisories when available.
  - `vrooli-autoheal host update-risk --json` — policy-oriented assessment: `safe`, `caution`, or `defer`, with evidence and recommendations.
- **Downstream effect:** runtime-health-scanner and infra-health can flag risky host-update windows before an incident rather than only after a crash.
- **Priority signal:** activates after at least one approved infra-health decision names a real incident that would have been easier to prevent with update-risk forecasting.
- **Implementation note:** This is future work only. Do not implement package mutation, driver installation, kernel pinning, or automatic update decisions as part of the host-integrity/incident workflow.

### Gap 9: Remediation artifact provenance and outcome reporting

- **Status:** Partially shipped on 2026-05-08 via the autoheal incident remediation workflow.
- **Unblocks:** Operator-approved recovery for serious incidents without requiring infra-health agents to invent privileged shell commands from raw logs.
- **Most likely host:** `vrooli-autoheal` durable incidents and remediation API/CLI.
- **Shape shipped:**
  - `vrooli-autoheal incidents remediations <incident-id> --json` — list structured remediation candidates with applicability, risk, preflight, simulation, fallback, post-checks, and decision prompt.
  - `vrooli-autoheal incidents remediation generate <incident-id> <remediation-id> --json` — generate a user-state artifact only when the candidate is applicable.
  - `vrooli-autoheal incidents remediation outcome <incident-id> <remediation-id> --status <status> --note ...` — record operator-reported outcome.
- **Artifact location rule:** Generated one-off scripts are incident artifacts under the `api-core/storage` state class for `vrooli-autoheal`, beneath `incidents/<incident-id>/remediation/<remediation-id>/`. The exact root is profile- and environment-dependent; agents should use the path returned by autoheal. Artifacts are host-specific and must not be checked into git. Reusable remediation templates, if extracted from generator code later, may live in scenario source only as machine-agnostic tested templates.
- **Downstream effect:** runtime-health-scanner can generate a visible operator decision from incident evidence instead of scraping logs or package-manager output.
- **Remaining gap:** Outcome quality is operator-reported today. Future work can correlate reported outcomes with post-check evidence and incident resolution automatically.

### Gap 10: Check shelving and registry reconciliation

- **Unblocks:** Honest uptime measurement during deliberate stops; the alarm-channel flood target; mechanical enforcement of the sensor-integrity rules (ghost / saturated / shelved) in `../strategy/RELIABILITY_TARGETS.md`.
- **Most likely host:** `vrooli-autoheal` (owns the check registry).
- **Shape required:**
  - `vrooli-autoheal check shelve <check-id> --until <duration|timestamp> --reason <text>` / `check unshelve <check-id>` / `check shelved --json` — shelved checks keep recording but are excluded from uptime and flood aggregates and flagged in `actions trends`. Expiry is mandatory (no permanent suppression); an expired shelf reverts to live alarming.
  - `vrooli-autoheal check reconcile --json` — diff the check registry against the expected set, **in both directions**: ghost checks (registry entry whose target no longer exists) and unsupervised plant (should-be-supervised member with no check) are the two error-row classes, mirroring `vrooli capacity reconcile`. The expected set is derived at reconcile time — core-set closure via scenario-dependency-analyzer ∪ load-bearing declared capability members (per operating-model rule 6) — never a hardcoded list inside autoheal.
- **Downstream effect:** The alarm-flood target becomes controllable; the supervised-set coverage target in `../strategy/RELIABILITY_TARGETS.md` moves from manual-diff `estimate` to `measured`; the scanner stops hand-tracking shelves in the runtime lessons artifact; resume-protocol re-baselining starts from a clean channel.
- **Priority signal:** already fired, twice — the 2026-07-23 flood decomposition (4,624 criticals/24h, ~92% from ghost + saturated checks) fired the ghost direction; the 2026-07-24 supervised-set sweep (~55 scenarios running, ~25 supervised; search-hub, plan-manager, and the `*-health` phase-provider fleet all unchecked) fired the missing-check direction. Top of the sensor queue.

### Gap 11: Capability-availability persistence

- **Unblocks:** The capability availability targets in `../strategy/RELIABILITY_TARGETS.md` (all `pending-telemetry`). Today, capability degradation is handled gracefully at query/run time but leaves no queryable history: search-hub's availability state is an in-memory circuit breaker plus an on-demand status probe; meta-optimization-manager marks projections UNAVAILABLE honestly but records nothing; test-genie's per-phase provider-readiness gate (the model contract — `required_when_applicable` vs `best_effort`, emitting run / run_degraded / skip) exists per run but has no aggregate/history surface. An outage is indistinguishable from missing data after the fact.
- **Most likely host:** each capability owner itself (extend-before-create; the owner already computes the signal, it just doesn't persist it). No central capability-health scenario — that would recreate the roster this PoR forbids.
- **Shape required:** each owner exposes a queryable availability/coverage aggregate with history, derived over its currently-declared members — e.g. search-hub provider-coverage % and degraded-member count over time; test-genie phase-runnability rates over the declared catalog; meta-optimization projection-availability history. Exact verbs are the owners' design choice.
- **Contract note (owner-side, recorded here as a pointer):** `test-genie.json` declarations carry a load-bearing distinction (`policy.providerReadiness`); `search.json` declarations carry per-provider performance targets but no equivalent required/best-effort notion. That distinction is the membership filter the Gap 10 reconcile derivation needs from search-hub — schema work for search-hub to close in its own contract, not for infra-health to compensate for with a list.
- **Downstream effect:** Capability availability rows move `pending-telemetry` → `pending-baseline` → `measured`; repeated unabsorbed capability degradation becomes escalatable evidence under the operating-model escalation rule.
- **Priority signal:** partially fired — the 2026-07-24 supervised-set sweep showed the entire capability substrate outside supervision, and no owner can currently answer "how available was this capability last week."

### Gap 12: Resource attribution and growth trends

- **Unblocks:** Burst attribution and storage growth-slope rows in the sensor map. Two related blind spots: (a) when host CPU/RAM saturates for a sustained window (e.g. a comprehensive test suite pinning CPU at 100%), no query names the owning scenario/run — triage is manual; (b) no per-scenario disk/DB growth trend exists, so a database growing anomalously (an optimization finding for its owner) surfaces only when system-level disk pressure alerts.
- **Most likely host:** system-monitor for burst attribution (it already samples per-process metrics and runs investigations); storage-health and/or cleanup-manager for growth trends. The capacity broker is explicitly **not** the host today — it enforces VRAM only, with CPU/RAM named as its V2 vision (`internal/capacity/types.go`); this gap supervises toward that V2 rather than duplicating the broker.
- **Shape required:** (a) a query mapping a sustained-saturation window to top owning scenarios/runs with evidence; (b) a per-scenario storage trend query (data dir + DB sizes over time, slope flagged against history). Findings route to the **owning scenario** — growth is the owner's optimization opportunity; cleanup-manager and storage-health are remediation surfaces, not the responsible parties.
- **Downstream effect:** Burst-attribution and growth-slope sensor cells fill; capacity-broker V2 (CPU/RAM claims) gains the observed-usage evidence it needs for honest admission.
- **Priority signal:** activates on the first sustained-saturation or disk-growth incident the scanner cannot attribute within one heartbeat using existing surfaces.

## Update protocol

Entries change when:
1. A capability ships → update the affected gap entry with "shipped on YYYY-MM-DD via <decision-id>" and search REPLACES-MANUAL markers to update consumers.
2. An `instrumentation-gap` decision is approved at the morning vision walk → append a new entry citing the decision id.
3. infra-contrarian challenges an entry as instrumentation-sprawl → if accepted, retire the entry with the rejection decision id.

## Change log

- `2026-07-24` (round 3) — Cascade completion (operator session, teams paused): Gap 10's `check reconcile` extended to both diff directions (ghost checks + unsupervised plant, expected set derived per operating-model rule 6) with the missing-check priority signal fired by the 2026-07-24 supervised-set sweep; Gap 11 added (capability-availability persistence, hosted by each owner — no central capability-health scenario — plus the `search.json` load-bearing contract note); Gap 12 added (burst attribution via system-monitor + storage growth trends via storage-health/cleanup-manager; capacity broker explicitly not the host until its CPU/RAM V2).
- `2026-07-23` (round 2) — Sensor-integrity hardening (operator session, teams paused): Gap 1's event-weighted sliver resolved via `actions trends` per-check uptime (Gap 2 narrowed to latency/build stats); Gap 10 added (check shelving with mandatory expiry + `check reconcile` registry diff) with its priority signal already fired by the 2026-07-23 alarm-flood decomposition.
- `2026-07-23` — Operator-curated re-baseline (operator session, teams paused): Gap 1 marked substantially shipped (`check history`, `actions history|uptime|trends|transitions|timeline`), its event-weighted-vs-per-scenario sliver folded into Gap 2, Gap 7 marked shipped pending outcome-field confirmation, dangling REPLACES-MANUAL pointers resolved. Prioritization rule adopted: build sensors for the targets the team intends to control, in loop-importance order — the sensor map in `../strategy/RELIABILITY_TARGETS.md` (empty sensor cells) is the queue.
- `2026-04-28` — File created with initial seven gaps grounded in runtime-health-scanner and platform-code-auditor's known fallbacks.
