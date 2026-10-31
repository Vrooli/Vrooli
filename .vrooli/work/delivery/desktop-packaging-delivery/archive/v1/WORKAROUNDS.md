# Workarounds

Broken-infrastructure log for the supervisor. Each entry records what failed,
the fallback, and when to retry. Open entries go at the top.

## Open

### DPF-WA-007 — 2026-10-07 — supervisor heartbeat trigger requires verified human caller

After recording F1's Needs operator wait in QUEUE, the documented
`prompt-manager team heartbeat-trigger effort-supervision effort-supervisor`
operation failed with `unauthenticated: verified human heartbeat caller
required`. This orchestrator cannot satisfy that caller requirement, so no
supervisor heartbeat was triggered.

Fallback: the F1 owner waits, reopening conditions and recommendation are in
QUEUE.md. Stop retrying this trigger; operator or supervisor feedback can wake
the orchestrator through the documented FEEDBACK.md path.

### DPF-WA-001 — 2026-10-06 — scenario-to-desktop CLI cannot reach its API

`vrooli scenario status scenario-to-desktop` reported `running`, but
`scenario-to-desktop targets list` failed with
`dial tcp 127.0.0.1:19925: connect: connection refused`. No running API port
was detected, and the saved config `api_base` is `http://localhost:19925`.

Fallback: the first E1 work unit runs
`vrooli scenario restart scenario-to-desktop` (after the TARGETS.md
in-flight check), or `scenario-to-desktop --auto-start ...`. It records which
one worked.

Retry: at E1 start.

### DPF-WA-002 — 2026-10-06 — live BAS is `start-failed`

This is known and owned by the BAS delivery goal (a shadow engagement makes
live BAS build from a broken restore point). Journeys that need BAS use the
TARGETS.md fallback.

Retry: when the BAS goal QUEUE shows the live BAS repaired.


- DPF-WA-010 — Resolved: the authorized D8 retry sequence completed; full dispatch history retained in the dated archive. Full entry: `archive/workarounds-2026-10-12.md`.

- DPF-WA-011 — Resolved for F1: the unavailable Staticcheck binary was not an acceptance gate; focused owner checks and mechanical gates passed. Full entry: `archive/workarounds-2026-10-12.md`.

- DPF-WA-012 — Resolved for F1 under DPF-FB-015: proposal creation is not an acceptance gate; do not retry late anchors. Future epochs anchor at admission. Full entry: `archive/workarounds-2026-10-12.md`.
## Resolved

- DPF-WA-015 — Resolved 2026-10-18 for API reachability: the 17Z supervisor receipt's original `agent-manager run get` succeeded and established the current NEEDS_REVIEW/crash state. The earlier 06:21Z outage no longer blocks reads. The missing Web Console session and bounded goal-evidence limit are recorded as the current E3 recovery limitation in QUEUE.md; they are not resolved by this API recovery.
- DPF-WA-014 — Resolved 2026-10-14; full launch history and receipt preserved below.
- DPF-WA-013 — Resolved 2026-10-14 for E3 after DPF-FB-018 corrected capacity counting and authorized fresh admission. The historical E2 `30/10` refusal and expired continuation remain as described in the earlier handoff; fresh E3 task `6041722d-aa98-4cd7-b179-b61310c2bf7f` accepted child `2d2177df-973c-4a36-abdb-849a1ede5fc1`.
- DPF-WA-009 — Resolved 2026-10-08: terminal FAIL JSON retained and treated as a failed gate; no rerun or artifact-index retry. Full entry: `archive/workarounds-2026-10-08.md`.
- DPF-WA-008 — Resolved 2026-10-08: later G7 repair worker delivered, repaired the cause and passed the single replacement unit receipt. Full entry: `archive/workarounds-2026-10-08.md`.
- DPF-WA-003 — Resolved 2026-10-07: SDA-governed wheel-only docopt repair built Kokoro and verified its four-artifact manifest. Full entry: `archive/workarounds-2026-10-08.md`.
- DPF-WA-004 — Resolved 2026-10-08: F1 scoped repairs passed and all four required G7 Test Genie unit receipts passed after the memory-pressure condition cleared. Preserve original failed receipts; do not rerun them. See QUEUE.md G7 receipts and DPF-FB-012.

- DPF-WA-006 — Resolved 2026-10-07: SDA now registers the canonical vanilla template surface; governed Electron 42.11.11 install and SDA validation passed. See F1 slice log 21:05 and 21:33.
- DPF-WA-005 — Resolved 2026-10-07: current-source Git Control Tower bundle export, preflight, generation, and Linux build pass through the supported owner route. See F1 slice log 21:35.
- DPF-WA-003 — Resolved 2026-10-07: SDA-governed wheel-only docopt 0.6.2 lock repair built Kokoro linux-amd64 and verified its four-artifact manifest. See F1 slice log 21:33.


### DPF-WA-014 — Resolved 2026-10-14 — Agent Manager interactive launch rejected by Web Console identity adapter

Task `6041722d-aa98-4cd7-b179-b61310c2bf7f` produced two terminal interactive
launch failures, runs `2d2177df-973c-4a36-abdb-849a1ede5fc1` and
`3e7fbe55-7f7f-4ef5-a81e-7ab6a4fe8a02`, both before prompt delivery with
`web-console stable-create identity unsupported; no Create submitted`. The
second used the same task/profile/parent after SUP-CAPACITY-QUALIFICATION;
eligible occupancy was 2/10, so this was not a capacity refusal. Neither run
had worker turns/tools or product changes. Profile: delivery-epoch-worker,
interactive, gpt-6-luna medium. Bug report: `knw-1791945836169396257`.

Resolution: DPF-FB-020 records the operator's passing live authenticated Create
and typed-input proof after the owner repair. The existing task/profile/parent
admitted run `f4677392-d27c-4fc4-8651-5b17251c3e9b` (which completed
goal-blocked) and, after DPF-FB-021, successor
`41a8e793-3817-4f66-a8a7-c85ff50891a0` at eligible occupancy 1/10. The
successor is pending with native interactive admission; worker start remains
unverified. The two failed attempts and bug report remain preserved above.
