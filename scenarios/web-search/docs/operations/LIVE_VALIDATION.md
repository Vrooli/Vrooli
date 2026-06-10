# Live Validation Runbook

Some web-search requirements describe **live agent behavior** that genuinely cannot be
validated hermetically: a real L3 research run needs agent-manager, ollama, searxng, and
the open web. Wiring fake refs to those criteria would be a checkmark, not a validation —
so they are typed `manual` in the requirements registry and earn their evidence through
this attended runbook instead.

## What it covers

| Requirement | Criterion | How the runbook validates it |
|---|---|---|
| REQ-P1-002 (`10-l3-iterative-research-agent`) | L3 run completes via agent-manager, invokes L2 tools, emits a brief; agent re-searches on gaps | Kicks `web-search research l3`, polls to terminal, asserts ≥1 (re-search: ≥2) `RunL2`/`LiveSearchService` API-log entries in the run window and a non-empty brief |
| REQ-P1-004 (`12-finding-auto-capture-policy`) | Complete L3 cycle writes findings with no capture flag | Asserts ≥1 new `source=FINDING_SOURCE_L3` finding created during the run window (the `l3` verb exposes no capture flag at all) |
| REQ-P0-004 (`04-scope-aware-federated-blending`) | Default learnings-served query ≤500ms p95 with zero external calls | Times 5 warm `SearchFindings` calls (p95 = max), asserts ≤500ms and a zero delta of `LiveSearchService/Search` log lines |

## How to run

```bash
cd scenarios/web-search
make validate-live                       # or: scripts/live-validate.sh
scripts/live-validate.sh "custom query"  # override the rotating research query
```

Preflight requires: web-search healthy, searxng with ≥2 responsive engines
(`resource-searxng engine-health`), agent-manager up, ollama reachable.
browser-automation-studio is checked but only warns (L2 falls back to direct HTTP).

Tunables (env): `LIVE_VALIDATE_L3_BUDGET` (default 900s), `LIVE_VALIDATE_POLL_SECS`
(15s), `LIVE_VALIDATE_LEARNINGS_P95_MS` (500), `OLLAMA_URL`.

## Evidence and cadence

On **success** the script logs one entry per covered requirement via
`test-genie requirements manual-log --expires-in 30` (JSONL at
`coverage/manual-validations/log.jsonl`, run artifact under
`coverage/manual-validations/artifacts/`). Valid, non-expired entries convert to live
evidence in the requirements rollup on the next full `test-genie execute web-search`.

On **any failed assertion** it logs nothing and exits non-zero, printing the failed step.

**Cadence: monthly.** Evidence carries a 30-day TTL; when it lapses the covered
requirements regress honestly in the rollup until the runbook is re-run. That is the
designed behavior — stale "passed" claims about live agent behavior are exactly what
this scheme exists to prevent.

Caveats for the operator:
- Run on a quiet system: the log-delta assertions count API-wide traffic, so a
  concurrent L2/L3 run could inflate (or, for the latency check, fail) the counts.
- The rotating query set asks "latest release" questions so a stale cached answer
  can't satisfy the assertions; pass an explicit query if those are all unavailable.
- The runbook is **not** part of any test phase or required set — `make test` stays
  hermetic. Run it deliberately.
