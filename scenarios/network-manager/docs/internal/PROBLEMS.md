# Problems — Network Manager

## What belongs here

Scenario-specific known issues, deferred decisions, architecture drift, and implementation blockers.

## What does NOT belong here

General Vrooli platform issues, unrelated bugs, or product ideas that belong in backlog.

## Entry template

```markdown
### YYYY-MM-DD — Title

- Signal:
- Impact:
- Next action:
```

## Entries

### 2026-06-23 — Product UI is live but still conservative

- Signal: The operator UI now has dashboard, snapshot, resolver/policy, devices, optimization, and settings/privacy pages backed by generated Connect clients. Mutating surfaces remain preview/approval-oriented and reflect the same conservative backend behavior as CLI/API.
- Impact: Operators can inspect and exercise P0 workflows from the UI, but live DNS/router mutation still depends on governed AdGuard/router adapters. The UI should not imply filtering or optimization has been applied unless backend capabilities confirm it.
- Next action: When a live AdGuard policy adapter lands, extend UI confirmation tests around real supported apply/rollback states.

### 2026-06-23 — Privacy retention is storage-backed but query logs are absent

- Signal: Privacy now persists retention/visibility settings and sweep audit records. Sweeps prune expired non-baseline snapshots, but DNS query-level logs do not exist yet and optimization-specific retention pruning is not wired.
- Impact: P0 privacy defaults are fail-closed and storage-backed before broad UI exposure. Query-log retention is recorded as disabled/no-op until a governed query-log source exists.
- Next action: Extend retention sweeps to optimization ledgers if experiments begin accumulating long-lived before/after evidence.

### 2026-06-23 — Optimization apply is conservative by default

- Signal: Optimization now persists runs/candidates/approval/rollback ledgers, requires a baseline snapshot, captures read-only candidate and after snapshots, and scores reliability-first evidence. The production applier returns `manual_required` until a real resolver/router adapter can prove safe apply and rollback.
- Impact: Operators can compare candidates without unsafe automation, and tests prove successful/failed apply and rollback behavior through a deterministic fake seam. Network Manager still does not claim live DNS/router optimization was applied in production.
- Next action: Connect a governed AdGuard/router optimization adapter after adapter capability and rollback semantics are proven.

### 2026-06-23 — storage-health validator returns unexpected EOF

- Signal: `storage-health validate scenario network-manager` exited twice with `Error: validate scenario "network-manager": unavailable: unexpected EOF` during Phase 7 validation.
- Impact: API/CLI tests, builds, `cli-health`, SDA health, requirements validation, and `unit-health --execution` passed, but static storage validation could not produce a current verdict for this slice.
- Next action: Filed `bug-inbox/unexpected-error/storage-health-network-manager-eof` as `knw-1782239066919260424`; rerun storage-health after the external validator is healthy.

### 2026-06-23 — storage-health validator deadline during Phase 8

- Signal: `storage-health validate scenario network-manager` failed with `deadline_exceeded` while awaiting `ScenarioValidationService/ValidateScenario` response headers.
- Impact: API/CLI tests, builds, `cli-health`, SDA health, requirements validation, and `unit-health --execution` passed, but standalone static storage validation again could not produce a current verdict for this slice.
- Next action: Filed `bug-inbox/unexpected-error/storage-health-network-manager-deadline` as `knw-1782240251178806531`; rerun storage-health after the external validator is responsive.

### 2026-06-23 — storage-health validator EOF during Phase 9

- Signal: `storage-health validate scenario network-manager` exited with `Error: validate scenario "network-manager": unavailable: unexpected EOF` during the operator UI slice.
- Impact: The UI/API/CLI/unit/requirements/SDA/ui-health gates passed, but standalone storage validation could not produce a current verdict. This matches the previously filed storage-health EOF class.
- Next action: Track under the existing storage-health EOF/deadline reports (`knw-1782239066919260424`, `knw-1782240251178806531`) and rerun after the external validator is responsive.

### 2026-06-23 — Orientation-triggered suite hit embedded ui-health once

- Signal: `vrooli scenario orient network-manager` spawned comprehensive run `20260623-193623-e4394270`, which completed 15/17 with embedded `phase-ui-health` reporting one error/blocker and `phase-storage` reporting the known EOF. Direct `ui-health validate scenario network-manager --json` passed before and after this run.
- Impact: The UI product surface still validates directly at L5, but orientation/scaffold-health remains blocked by the comprehensive suite result until embedded validator instability and storage-health EOF are gone.
- Next action: Do not change UI code based on the embedded-only failure unless it reproduces through direct `ui-health`; track storage under the existing storage-health reports and embedded validator behavior under prior suite-instability notes.

### 2026-06-23 — scenario contracts phase returned cli-health EOF

- Signal: `vrooli scenario test network-manager` completed 16/17 with only `phase-contracts` failing: `cli-health validation RPC failed: unavailable: unexpected EOF`. Direct `cli-health validate scenario network-manager` passed before and after the suite.
- Impact: The scenario-local CLI contract is clean, but the full suite verdict was failed due to an embedded validator/RPC EOF.
- Next action: Filed `bug-inbox/unexpected-error/test-genie-contracts-cli-health-eof` as `knw-1782239328731253209`; rerun the suite once after confirming direct cli-health passes.

### 2026-06-23 — Phase 8 scenario suite hit embedded validator failures

- Signal: `vrooli scenario test network-manager` after the Home Automation slice completed 13/17. Direct `cli-health validate scenario network-manager` passed, direct `ui-health validate scenario network-manager` passed after rebuilding the UI Health CLI, standalone `storage-health` timed out, and the embedded suite reported contracts EOF, storage EOF, standards timeout, and UI handshake failure.
- Impact: After increasing the standards phase timeout from 120s to 300s and rerunning, the suite improved to 16/17: contracts, UI Health, standards, and all code/product phases passed; only embedded storage-health EOF remains.
- Next action: Rerun the full suite after storage-health is responsive; current storage-health timeout/EOF evidence is filed under `knw-1782239066919260424` and `knw-1782240251178806531`.

### 2026-06-23 — Inventory discovery source is conservative

- Signal: Device inventory now has persisted device/group records and identity reconciliation, but production discovery returns an explicit unsupported finding until resolver-derived client evidence is available.
- Impact: Operators can safely store and explain device records produced by future resolver/router/host sources without fake LAN scan data. Live refresh will not invent devices.
- Next action: Connect AdGuard Home or another governed resolver client source that can provide client IDs, hostnames, IPs, MAC/stable IDs, and last-seen timestamps.

### 2026-06-23 — Snapshot throughput and gateway probes are capability gaps

- Signal: P0 snapshot now runs standard-library read-only probes, but gateway reachability and throughput are marked unsupported/unavailable unless a platform adapter or privacy-reviewed measurement backend is added.
- Impact: Operators get honest baseline evidence without fake throughput or privileged ICMP/router assumptions.
- Next action: Implement host/router/manual adapter capabilities before treating gateway or throughput as measured values.

### 2026-06-23 — AdGuard Home resource still needs implementation decision

- Signal: PRD selects AdGuard Home as the first resolver backend. Network Manager can now store governed backend config by `token_ref`, run dry-run setup, and expose conservative unverified health, but no resource-backed AdGuard HTTP client is connected.
- Impact: P0 can record backend and policy intent safely, but cannot claim filtering is active or apply persistent upstream/filter changes until a governed resource-backed client confirms them.
- Next action: Decide whether AdGuard Home is a first-class Vrooli resource or resource-backed adapter, then connect the client through secret/resource governance.

### 2026-06-23 — Policy writes are ledger-backed but live adapter support is conservative

- Signal: Policy preview, approval, apply, pause/resume, and rollback now persist change plans and audit records, but the production policy adapter returns `unsupported` for live resolver writes.
- Impact: Operators can inspect and approve intended DNS filtering changes without fake success claims. Actual AdGuard allowlist/denylist/blocklist mutation still needs a governed adapter client.
- Next action: Implement the AdGuard Home policy adapter after the resource/secret governance decision is resolved.

### 2026-06-23 — First router adapter not selected

- Signal: P0 intentionally uses manual router guidance; P1 needs one explicit router adapter.
- Impact: Router-enforced DNS rules and Wi-Fi/router changes remain manual until a platform is chosen.
- Next action: Select based on first real deployment environment.

## Architecture Drift

No drift yet. The template example domain has been removed; the main risk is leaving scaffold responses in place without requirement-tagged domain tests.

## Cross-references

- [`DECISIONS.md`](DECISIONS.md)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
