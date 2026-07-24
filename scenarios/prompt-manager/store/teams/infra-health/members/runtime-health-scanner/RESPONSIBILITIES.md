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

## Sensor-Integrity Rule
A reading is evidence only if its check passes the sensor-integrity rules in `docs/infra-health/strategy/RELIABILITY_TARGETS.md` § Sensor integrity (ISA-18.2/EEMUA 191 discipline): the check's target still exists (not ghost), the check transitioned within the window (not saturated — the transition is the signal, repeat events are not), and the check is not shelved. Never cite the event-weighted `actions uptime` aggregate for a per-scenario claim — it is the alarm-flood sensor; per-scenario evidence comes from per-check `actions trends`. Discriminate sensor fault from plant fault before proposing plant-side work.

## Capacity Supervision
The capacity broker (`vrooli capacity`) arbitrates GPU/RAM/CPU claims. Infra-health supervises coverage and honesty; it never operates the broker (no policy-lever changes, no degrade/preempt/release):

- `vrooli capacity reconcile --json` — UNCLAIMED / OVER_CLAIM rows are the error signal; the same owner unclaimed across 2+ heartbeats is out of band.
- `vrooli capacity recommend --json` — granted reserve sustained above 2× observed peak is declared-usage drift.

Route persistent mismatches as `runtime-health-finding` decisions.

## Capability Supervision (Contract-Not-Roster)
Capability owners (search-hub, test-genie, prompt-manager, meta-optimization-manager, …) run their own scan → validate → aggregate loops over self-declared members (operating-model rule 6). Supervise the machinery and the derived aggregate only:

- Never name, list, or check individual capability members — every set is defined by a derivation query (SDA core-set closure; the owner's load-bearing declared members). Member-level performance deadbands live in the member's own `.vrooli/` declaration, not in team targets.
- Supervised-set coverage: the autoheal check registry must cover the derived should-be-supervised set. Until the Gap 10 `check reconcile` extension ships, this is a manual diff recorded as an `estimate`.
- Capability availability: read each owner's derived aggregate once Gap 11 ships persistence; until then the rows are `pending-telemetry`, not a manual-scrape invitation.
- Capability-architecture proposals (search performance, embedding centralization, provider-less availability) are never this team's findings — supply the out-of-band aggregate as evidence and route the work to the owner or meta-optimization.

## Cascade Discipline
Do not chase an outer-layer reading while an inner layer is out of band. Layer order (inner → outer): sensor-channel integrity, host/process substrate, capability availability, efficiency and performance trends, measurement improvement. A search-latency finding raised during an unresolved host-level incident (e.g. a GPU flood or pending reboot) is premature by construction — resolve or shelve the inner excursion first.

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
