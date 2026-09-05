---
name: "ai-gateway-improve"
description: "Regulate AI Gateway against its setpoint: cost per caller, local share, fallback, failure, breaker-open and capacity-rejection counts, and p95 latency over route evidence. Routes each out-of-band row to a curation move, a work-ladder rung, or an owner, and names the missing cost measure that blocks adaptive routing."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["ai-gateway", "improve", "self-improvement", "control-loop", "setpoint", "routing", "measures", "cost", "breaker", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 3
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["ai-gateway", "program-runtime", "prompt-manager", "vrooli-memory", "agent-manager"]
    commands: ["ai-gateway measures total", "ai-gateway measures cost", "ai-gateway measures tokens", "ai-gateway measures local-share", "ai-gateway measures success-rate", "ai-gateway measures fallback-rate", "ai-gateway measures failure-rate", "ai-gateway measures breaker-open", "ai-gateway measures capacity-rejections", "ai-gateway measures latency-p95", "ai-gateway routing evidence-list", "ai-gateway routing health", "ai-gateway inventory roles", "ai-gateway inventory smoke", "ai-gateway conformance scan", "program-runtime bindings condition", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: AI Gateway Improve

Regulate AI Gateway, the routing instrument, against the setpoint below. The plant is the route: role and profile resolution, provider health, capacity admission, and the `route_events` evidence every call leaves. This skill is read by an agent whose task is AI Gateway itself (`goal-loop`, or a heartbeat). It never edits a resource's model policy or another scenario's calls; it files.

Required reading:
- `prompt-manager skill read ai-gateway` — the usage skill; every command named here is documented there or in the CLI.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming, cited by section below.
- `prompt-manager skill read scenario-work-ladder` — where code routes go.
- `prompt-manager skill read measures-adoption` — the route for a missing measure.
- `path:scenarios/ai-gateway/PRD.md` — `OT-P1-007` (route measures) and `OT-P2-001` (adaptive routing, blocked on them).

### 1. Focus and scope

**In scope:** the setpoint rows below; reading route evidence; filing ladder rungs against ai-gateway; filing `report-bug` against a resource whose role policy is wrong or a scenario that bypasses the gateway.

**Out of scope:** editing `resources/*/model-policy.json` (resource-owned; file instead); adding a provider; the usage skill's content; tuning a caller's schema or prompt (the caller's usage skill).

### 2. Setpoint

Bands are targets. Readings are dated observations; re-read them every cycle with `run ai-gateway.setpoint-read`.

| Row | Sensor | Band | Today (2026-09-02) |
|---|---|---|---|
| cost-per-caller | `ai-gateway measures cost --window last_7d` → priced `route_events.cost_estimate`; the backing aggregate also groups by `(scenario, operation)` | cost_usd per (scenario, operation) readable for `last_7d` | newly readable total; per-caller CLI projection remains a follow-up |
| local-share | `ai-gateway measures local-share --window last_7d` → successful rows with `selected_locality == local` | ≥ 0.80 | re-read from the windowed measure |
| fallback-rate | `ai-gateway measures fallback-rate --window last_7d` → `rate` | ≤ 0.02 | 0.0083 over 5,267 routes |
| failure-rate | `ai-gateway measures failure-rate --window last_7d` → `rate` | ≤ 0.02 | 0.0237 over 5,267 routes |
| breaker-open | `ai-gateway measures breaker-open --window last_7d` → `count` | 0 | 89 of 5,267 routes |
| latency-p95 | `ai-gateway measures latency-p95 --window last_7d` → `latency_ms` | ≤ 4,000 ms | 6,223 ms (in-program read; `ai-gateway/measures/latency-p95` resolves in `bindings list` on 2026-09-02. A registry snapshot taken before an API restart can omit it, and `setpoint-read` then reports the row `unavailable`: re-read next cycle, do not substitute a hand reading) |
| capacity-rejections | `ai-gateway measures capacity-rejections --window last_7d` → `count` | 0 | 0 (field omitted by protojson; `total` is 5,267 so the zero is real) |
| success-rate | `ai-gateway measures success-rate --window last_7d` → `rate` | ≥ 0.97 | 0.9594 |
| route-volume | `ai-gateway measures total --window last_7d` → `count` | pending-baseline: rising with adoption | 5,267 |
| external-friction | `run agent-manager.friction-digest` with inputs `scenario=ai-gateway`, `window_days=7` → `recurring_count` | 0 recurring fingerprints | 0 recurring, 0 episodes for this scenario across the last 40 runs (all created 2026-09-02; the program reads `run_limit=40` runs) |

### 3. Sensors

Read all rows through `run ai-gateway.setpoint-read` (contract: `.vrooli/program-runtime/setpoint-read.json`). The ten measures are governed bindings; in-kernel they take `window={"token": "TIME_WINDOW_TOKEN_LAST_7D"}` and the CLI takes `--window last_7d`. Cost is unavailable when the window has no priced rows; do not substitute a hand sum or a model price.

Fleet sensors every scenario has: `program-runtime bindings condition` for ai-gateway's own bindings, and Agent Manager friction for `ai-gateway` commands through `run agent-manager.friction-digest` with `scenario=ai-gateway`.

External sensors outrank self-reported ones: `ai-gateway conformance scan --scenario <slug>` findings and a caller's `report-bug` outrank the gateway's own success rate.

### 4. Golden corpora

AI Gateway ships no `evals/` corpus with a floor. The closest fixed instrument is the typed-inference schema subset (`docs/reference/typed-inference-schema-subset.md`) exercised by the API package tests (`GOWORK=off go test ./...` from `scenarios/ai-gateway/api`). Until a routing corpus exists the floor is the test suite passing; a routing corpus with a floor is a `measures-adoption` item (§5), and this section is re-derived when it lands.

### 5. Actuators and ladder routing

`Filing` rows hand off: a `measures-adoption` item, a work-ladder rung against ai-gateway, or `report-bug` against a resource or a calling scenario. No row is a curation move: the gateway exposes no data lever this skill may turn without a diff, which is itself recorded in §1. One route per row per cycle. When two rows' predicates hold for the same reading, take the first matching row from the top; re-read next cycle before taking the next.

| Kind | Row out of band | Route | Sensor that should move |
|---|---|---|---|
| Filing | cost-per-caller has no caller projection | `measures-adoption` item against ai-gateway: expose the existing per-caller cost aggregate as a descriptor-backed table measure grouped by (scenario, operation) | cost-per-caller |
| Filing | local-share below band, `policy_reasons` name a breaker | Read `routing health`; an `open` local breaker with `last_failure_class=timeout` is W3 against ai-gateway when the timeout is the gateway's default, and `report-bug` against `resource-ollama` when the model itself is slow (`inventory smoke --provider ollama`) | local-share, breaker-open |
| Filing | local-share below band, `policy_reasons` name capacity | Read `capacity_verdict` on the rejected rows; `advisory_reclaim_unavailable` is a host capacity-broker finding: `report-bug` against the control plane's capacity owner with the verdicts | local-share, capacity-rejections |
| Filing | local-share below band, callers request `remote-only` or `quality-first` for internal data | `report-bug` against the calling scenario with the evidence event ids; the route is policy-correct | local-share |
| Filing | fallback-rate above band | Group the fallback rows by role (`evidence-list`, then `group_by("role")` in a program); a role with no `available` local candidate (`inventory roles --provider ollama`) is a role-catalog gap: file against the owning resource to add the role to `model-policy.json` | fallback-rate |
| Filing | failure-rate above band, one provider dominates | `inventory smoke --provider <p>`; a failing smoke is `report-bug` against that resource | failure-rate |
| Filing | failure-rate above band, one role dominates with `VALIDATION_FAILED` | The role's local candidate does not hold the schema: W3 against ai-gateway to reorder candidates for that role in `config/inference-role-catalog.json`, recorded with the failure sample | failure-rate |
| Filing | breaker-open above 0 | Read `routing health` cooldowns; a breaker open for more than one cooldown with a healthy smoke is W3 against ai-gateway (half-open probe not firing); a breaker that reopens on every probe is `report-bug` against the resource | breaker-open |
| Filing | latency-p95 above band | Group the slow rows by `selected_model`; one local model dominating is `report-bug` against `resource-ollama` with the p95 and model; the gateway adding latency (routing preview slow with no provider call) is W3 here | latency-p95 |
| None, or Filing | capacity-rejections above 0 with `local-only` or `privacy-sensitive` callers refused | Descriptive by design; journal the count and the callers; `report-bug` against the capacity owner only when `unknown_capacity` appears with a declared footprint | capacity-rejections |
| Filing | success-rate below band with failure-rate in band | Routes are ending neither succeeded nor failed: W2 against ai-gateway (outcome codes incomplete on some path) | success-rate |
| Filing | a scenario bypasses the gateway (`conformance scan`) | `report-bug` against that scenario with the finding; never edit its code from here | route-volume |
| Filing | external-friction recurring fingerprint | Read the fingerprint's episode; if the command is ai-gateway's, W3 here; if the fix is skill prose, `skill-improvement-suggestions` on the usage skill | external-friction |

### 6. Anti-gaming

`improvement-do-and-dont` §1 and its three DON'T subheadings (tagged test, known-issue ledger, suppression) and §2 (the skeptic test) apply verbatim. AI Gateway's own gaming moves, each worth zero credit and a review flag:

- Raising a breaker threshold or lengthening a cooldown to lower `breaker-open` without a smoke that proves the provider healthy.
- Reordering role candidates toward remote providers to lower `failure-rate` while `local-share` falls, without recording both readings.
- Changing a request's default profile or privacy class in the gateway to move `local-share`.
- Counting `routing preview` calls in `route-volume`.
- Estimating `cost-per-caller` from model list prices and reporting it as a reading.
- Marking a conformance finding as an approved exception without an operator record.

### 7. Evidence

One `vrooli-memory journal note --kind work-record` per cycle:

```
--trigger  "<goal> cycle <n>: <row> <reading> vs <band>"
--approach "<route row text>"
--evidence "<before> -> <after> on <sensor command>; event ids sampled"
--outcome  "<in band | filed <ref> | unavailable: <reason>>"
```

A sensor unavailable for three cycles is a `docs/internal/PROBLEMS.md` entry with the three dated readings. Filings against resources use `report-bug` with the role, provider, and evidence event ids as the observation.

### 8. Stop rules

| Condition | Action |
|---|---|
| The API package tests fail | Only the test route runs this cycle; no routing change lands on a red suite |
| A row reads `unavailable` | Journal; do not estimate; after three cycles, PROBLEMS.md and W2 |
| `route-volume` is below 100 for the window | Rates are not evaluated this cycle; journal the count |
| A route needs a grant (`refused_no_grant`) | Stop and request the grant through the session path |
| Every readable row in band for two consecutive cycles | Propose close-out to the operator; stop |
| The session's inference or delegation ceiling is reached | Stop; journal; do not open a new session to continue |

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `setpoint-read` reports every row unavailable | program-runtime restarted and the CLI resolved a stale port | `vrooli scenario status program-runtime`; `PROGRAM_RUNTIME_API_PORT` | Set the port for this cycle; file W3 for auto-detect |
| A measure row fails with `proto: syntax error ... unexpected token` | `window` passed as a string in-kernel | the program's `window=` argument | Pass `{"token": "TIME_WINDOW_TOKEN_LAST_7D"}` |
| `capacity-rejections` returns `{}` | protojson omits a zero count | `measures total` for the same window | Read as 0 when total is above 0; unavailable when total is 0 |
| `local-share` reads 1.00 while fallback-rate is above 0 | The evidence sample is the most recent 50 rows, not the window | the reading's `basis` | Read the two rows on their own bases; do not average them |
| `routing health` shows `cooldown_until` weeks in the past with `state=open` | Stale breaker rows never closed by a probe | `inventory smoke --provider <p>` | W3 against ai-gateway: half-open recovery not applied to stale rows |
| `conformance scan` reports a scenario that only reads `ai-gateway` docs | The scanner matched a doc path | the finding's file | `report-bug` against ai-gateway's scanner, not the scenario |
