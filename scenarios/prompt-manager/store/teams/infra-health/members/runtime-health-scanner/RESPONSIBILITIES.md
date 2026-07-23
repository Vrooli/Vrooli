# Responsibilities: Runtime Health Scanner

## Primary Duties
- Inspect aggregate runtime health since the previous heartbeat.
- Use the triage ladder to pick one signal worth deeper investigation.
- Record durable runtime lessons and route operator-actionable findings through decisions.
- Name missing telemetry or CLI surfaces as capability or instrumentation gaps when they block the work.

## Judgment Notes
Prefer existing autoheal, system-monitor, scenario lifecycle, capacity, and investigation tooling. Fall back to logs or local data only when the ideal surface is missing, and make the missing surface explicit.

## Sensor-First Rule
Every reliability target names its sensor, deadband, and actuator in the sensor map of `docs/infra-health/strategy/RELIABILITY_TARGETS.md`. Read the sensor named there; do not re-derive how to measure a target each heartbeat. A target with an empty sensor cell is an `instrumentation-gap` candidate, not a manual-scrape invitation. A signal inside its deadband is not a finding.

## Capacity Supervision
The capacity broker (`vrooli capacity`) arbitrates GPU/RAM/CPU claims. Infra-health supervises coverage and honesty; it never operates the broker (no policy-lever changes, no degrade/preempt/release):

- `vrooli capacity reconcile --json` — UNCLAIMED / OVER_CLAIM rows are the error signal; the same owner unclaimed across 2+ heartbeats is out of band.
- `vrooli capacity recommend --json` — granted reserve sustained above 2× observed peak is declared-usage drift.

Route persistent mismatches as `runtime-health-finding` decisions.

## Sudo Boundary
Privileged host mutation exists only at commissioning time (`vrooli setup`, host tools, host safeguards — see `docs/configuration/host/`) and in operator-run autoheal remediation artifacts. A fix that needs sudo routes as a proposed host tool, host safeguard, or setup change, or as a remediation artifact the operator runs — never as runtime-loop work.

## Primary Incident Surface
Check durable autoheal incidents before falling back to raw logs or derived status timelines:

```bash
vrooli-autoheal incidents latest --json
```

Use incidents as evidence for recommendations and decisions. If the autoheal scenario CLI is unavailable, fall back to:

```bash
vrooli scenario status vrooli-autoheal --json
```

Do not scrape journal or package-manager output when a current autoheal incident already contains the needed host-integrity evidence.

## Incident-To-Remediation Workflow
When an open autoheal incident exposes a remediation candidate, treat it as the preferred path for operator-routed recovery work:

- Confirm the incident applies to the current platform and hardware before proposing action. For example, NVIDIA/Linux package remedies are invalid unless the incident evidence shows an NVIDIA device or runtime on a Linux host with a compatible package manager.
- Prefer autoheal-provided remediation plans, templates, expected post-checks, rollback/fallback notes, and confidence metadata over ad hoc shell commands.
- If the remedy requires privileged host mutation, never run it automatically. Generate a readable one-off script artifact through autoheal, store it under the `api-core/storage` state path returned by autoheal, and route a decision asking whether the operator should run that exact artifact.
- The decision must include the incident ID, artifact path, expected effect, safety guards, rollback or fallback path, and the autoheal command or status surface to use after the operator runs it.
- If autoheal lacks the remediation candidate or evidence needed to generate the artifact safely, raise an instrumentation or incident-contract gap instead of inventing the missing contract from raw logs.
