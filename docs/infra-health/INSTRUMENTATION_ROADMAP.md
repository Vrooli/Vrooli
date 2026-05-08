# Instrumentation Roadmap

A living list of **stats the infra-health team needs** and the **capabilities that would make them `measured` instead of qualitative**. Modeled on [docs/monetization/TELEMETRY_ROADMAP.md](../monetization/TELEMETRY_ROADMAP.md). Purpose:

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

### Gap 1: Autoheal history CLI

- **Unblocks:** Repeat-failure tier-1 detection without raw SQLite reads. Heal-loop tier-2 detection. Trend graphs of check pass/fail counts over time.
- **Most likely host:** Extend `vrooli-autoheal` CLI.
- **Shape required:**
  - `vrooli-autoheal history --since=<duration> --json` — time-series of check pass/fail counts and per-check status changes
  - `vrooli-autoheal heal-attempts --since=<duration> --json` — list of heal attempts with scenario / resource, trigger, outcome, duration
- **Downstream effect:** runtime-health-scanner stops reading SQLite directly; finding cycle time drops; tier-1 / tier-2 triage becomes a single CLI call.
- **Priority signal:** activates when runtime-health-scanner starts producing recurring `capability-gap` decisions naming this verb shape.
- **REPLACES-MANUAL:** SQLite fallback path in `runtime-health-scanner/RESPONSIBILITIES.md` ("Current Gaps & Fallbacks") and `TOOLS.md` ("Primary Surfaces").

### Gap 2: Scenario lifecycle stats

- **Unblocks:** Restart-frequency targets (`RELIABILITY_TARGETS.md`), slow-restart trend tier-3 detection, setup-step duration tracking, build-time trends.
- **Most likely host:** Extend the `vrooli` core CLI (the lifecycle owner).
- **Shape required:**
  - `vrooli scenario stats --since=<duration> --json` — per-scenario aggregated restart count, total runtime, longest-uptime streak, mean cold-start latency, last-N-restart timestamps
  - `vrooli scenario stats --scenario=<name> --since=<duration> --json` — single-scenario detail
- **Downstream effect:** Most platform-component targets in `RELIABILITY_TARGETS.md` move from `pending-telemetry` to `measured`. Slow-restart-trend tier in runtime-health-scanner becomes mechanical.
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

- **Unblocks:** "vrooli setup end-to-end" target in `RELIABILITY_TARGETS.md`.
- **Most likely host:** Extend `vrooli setup` itself to emit per-step durations.
- **Shape required:**
  - `vrooli setup --json` (or a `--report` flag) emits per-step timing and aggregated total
  - Stored historically so trend detection works across setup runs
- **Downstream effect:** Setup-time target moves to `measured`. Regressions in setup time are detectable.
- **Priority signal:** activates when a clean-machine-install workflow is part of the deployment story (tier-2 onboarding).

### Gap 7: Heal-success-rate stat

- **Unblocks:** "Heal success rate" platform target in `RELIABILITY_TARGETS.md`.
- **Most likely host:** Probably already in autoheal's SQLite (`heal_trackers`); needs surfaced via the CLI from Gap 1.
- **Shape required:** Implicit in `vrooli-autoheal heal-attempts --json` if the field set includes outcome (success / failure / no-action-needed).
- **Downstream effect:** Heal success rate moves to `measured`.
- **Priority signal:** activates with Gap 1.

### Gap 8: Host update-risk forecasting

- **Unblocks:** Early warning before kernel, driver, package-manager, or reboot-required updates create host capability drift.
- **Most likely host:** Extend `vrooli-autoheal` after host inventory and durable incidents have operated long enough to prove the base signal quality.
- **Shape required:**
  - `vrooli-autoheal host updates --json` — safe read-only view of pending host updates, reboot-required signals, kernel/module package pairings, held/broken packages, and known compatibility advisories when available.
  - `vrooli-autoheal host update-risk --json` — policy-oriented assessment: `safe`, `caution`, or `defer`, with evidence and recommendations.
- **Downstream effect:** runtime-health-scanner and infra-health can flag risky host-update windows before an incident rather than only after a crash.
- **Priority signal:** activates after at least one approved infra-health decision names a real incident that would have been easier to prevent with update-risk forecasting.
- **Implementation note:** This is future work only. Do not implement package mutation, driver installation, kernel pinning, or automatic update decisions as part of the host-integrity/incident workflow.

## Update protocol

Entries change when:
1. A capability ships → update the affected gap entry with "shipped on YYYY-MM-DD via <decision-id>" and search REPLACES-MANUAL markers to update consumers.
2. An `instrumentation-gap` decision is approved at the morning vision walk → append a new entry citing the decision id.
3. infra-contrarian challenges an entry as instrumentation-sprawl → if accepted, retire the entry with the rejection decision id.

## Change log

- `2026-04-28` — File created with initial seven gaps grounded in runtime-health-scanner and platform-code-auditor's known fallbacks.
