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

### 2026-06-24 — AdGuard rollout affordance suite hit known embedded validator EOFs

- Signal: `vrooli scenario test network-manager` completed 15/17 after the AdGuard rollout checklist slice. Embedded `phase-contracts` failed with `cli-health validation RPC failed: unavailable: unexpected EOF`, and embedded `phase-storage` failed with `storage-manager validation RPC failed: unavailable: unexpected EOF`.
- Impact: Direct API/CLI/UI/resource tests and builds, proto lint, resource validation, direct `cli-health`, `ui-health`, SDA, Structure Health, Security Health, requirements validation, Unit Health execution, live AdGuard resource health, live `network-manager resolver rollout`, and DNS smoke all passed. The full-suite failure matches the existing embedded validator EOF class rather than the rollout implementation.
- Next action: Track under existing embedded cli-health EOF (`knw-1782239328731253209`) and storage-manager EOF/deadline (`knw-1782239066919260424`, `knw-1782240251178806531`) reports; rerun the comprehensive suite after those validators are responsive.

### 2026-06-24 — AdGuard UI parity suite hit known embedded validator EOFs

- Signal: `vrooli scenario test network-manager` completed 15/17 after the AdGuard UI parity and live-validation slice. Embedded `phase-contracts` failed with `cli-health validation RPC failed: unavailable: unexpected EOF`, and embedded `phase-storage` failed with `storage-manager validation RPC failed: unavailable: unexpected EOF`.
- Impact: Direct `cli-health validate scenario network-manager`, `ui-health validate scenario network-manager`, SDA health, requirements validation, resource validation, resource schema validation, API/CLI/resource tests and builds, UI tests/lint/build, Unit Health execution, live AdGuard resolver health, live inventory refresh, inert global policy apply/rollback, and post-rollback snapshot validation all passed. The comprehensive suite failure matches the existing embedded validator EOF class.
- Next action: Track under existing embedded cli-health EOF (`knw-1782239328731253209`) and storage-manager EOF/deadline (`knw-1782239066919260424`, `knw-1782240251178806531`) reports; rerun the comprehensive suite after those validators are responsive.

### 2026-06-24 — AdGuard optimization slice suite hit known embedded validator EOFs

- Signal: `vrooli scenario test network-manager` run `20260624-190327-12273dd9` completed 15/17 after the AdGuard optimization slice. Embedded `phase-contracts` failed with `cli-health validation RPC failed: unavailable: unexpected EOF`, and embedded `phase-storage` failed with `storage-manager validation RPC failed: unavailable: unexpected EOF`.
- Impact: Direct `cli-health validate scenario network-manager`, SDA health, requirements validation, resource validation, API tests/build, CLI tests/build, Unit Health execution, live AdGuard resolver health, and live optimization apply/rollback all passed. The full-suite failure matches the existing embedded validator EOF class rather than the AdGuard optimization code path.
- Next action: Track under existing embedded cli-health EOF (`knw-1782239328731253209`) and storage-manager EOF/deadline (`knw-1782239066919260424`, `knw-1782240251178806531`) reports; rerun the comprehensive suite after those validators are responsive.

### 2026-06-24 — storage-manager deadline during AdGuard inventory slice

- Signal: After AdGuard client inventory import, direct `storage-manager validate scenario network-manager` failed with `deadline_exceeded` while awaiting `ScenarioValidationService/ValidateScenario` response headers.
- Impact: API/CLI tests and builds, `cli-health`, SDA health, requirements validation, Unit Health execution, resource validation, live resolver health, and live device refresh passed. Static storage validation remains blocked by the previously filed external validator timeout class.
- Next action: Track under existing storage-manager EOF/deadline reports (`knw-1782239066919260424`, `knw-1782240251178806531`) and rerun after the validator is responsive.

### 2026-06-24 — AdGuard Home is healthy but household enforcement is still gated

- Signal: The `adguard-home` resource is running with DNS published at `192.168.1.173:53/tcp` and `192.168.1.173:53/udp`, first-run setup completed through `resource-adguard-home bootstrap`, credentials stored at `secret/resources/adguard-home/admin`, resource health `healthy`, protection enabled, and query logging disabled. Network Manager resolver health now reports `healthy` through the resource-backed client, global policy mutations are rollback-backed, and device inventory can import AdGuard client evidence without query-level DNS logs. Resolver status also exposes `enforcement_status` and `enforcement_evidence` from AdGuard client metadata.
- Impact: Network Manager can truthfully report AdGuard resolver protection as active for the resource, can apply global AdGuard user-rule/protection changes through approval and rollback ledgers, can refresh inventory from configured/auto AdGuard clients, and can apply/rollback the supported global DNS filtering optimization candidate through the same AdGuard policy adapter. It can now distinguish no client evidence from observed AdGuard client evidence, but it still must not claim router-wide enforcement until router DHCP/RDNSS evidence verifies clients are being assigned `192.168.1.173`; client/group-specific policies remain gated.
- Next action: Reserve or confirm `192.168.1.173` in router DHCP, point router DNS/RDNSS at that address when ready, then add client/group-specific policy mapping after identity semantics are explicit.

### 2026-06-24 — Continuous monitoring is operator-triggered

- Signal: Network Manager now persists monitoring schedules, runs snapshot comparisons against a baseline, and records regression alerts through API/CLI/UI, but it does not run a background scheduler loop.
- Impact: Operators and agents can create schedules and trigger checks safely without adding long-running timer complexity to the API process. Automatic recurring execution still needs a lifecycle-aware scheduler that avoids duplicate runs and respects scenario shutdown.
- Next action: Connect a Vrooli-managed scheduler or worker after schedule ownership, locking, and missed-run behavior are defined.

### 2026-06-24 — storage-manager deadline during monitoring slice

- Signal: `storage-manager validate scenario network-manager` failed with `deadline_exceeded` while awaiting `ScenarioValidationService/ValidateScenario` response headers after the monitoring slice.
- Impact: Direct API/CLI/UI/proto/SDA/requirements/unit-health checks passed or were in progress, and the new monitoring domain has domain-owned schema/repository tests. Static storage validation remains blocked by the previously filed external validator timeout class.
- Next action: Track under existing storage-manager EOF/deadline reports (`knw-1782239066919260424`, `knw-1782240251178806531`) and rerun after the validator is responsive.

### 2026-06-24 — Monitoring slice suite hit embedded validator EOFs

- Signal: `vrooli scenario test network-manager` run `20260624-145028-9d371586` completed 15/17; embedded contracts failed with `cli-health validation RPC failed: unavailable: unexpected EOF`, and embedded storage failed with `storage-manager validation RPC failed: unavailable: unexpected EOF`.
- Impact: Direct `cli-health validate scenario network-manager` passed immediately after, runtime `network-manager status --json` was healthy, UI Health rendered the running UI successfully, and direct storage-manager had already reproduced the known deadline/EOF class.
- Next action: Track under existing embedded cli-health EOF (`knw-1782239328731253209`) and storage-manager EOF/deadline (`knw-1782239066919260424`, `knw-1782240251178806531`) reports; rerun the full suite after those validators are responsive.

### 2026-06-23 — Encrypted DNS guidance is advisory

- Signal: Network Manager now generates IPv6 resolver, DoT/DoQ, DoH, and endpoint/browser guidance through PolicyService, CLI, and UI, but it does not mutate router/firewall or endpoint policy.
- Impact: Operators can identify bypass risk areas and get manual or adapter-preview instructions without unsafe enforcement claims, TLS interception, packet capture, or hidden monitoring. Actual enforcement still depends on future governed router/firewall/endpoint adapters with rollback support.
- Next action: Promote guidance items to adapter-backed previews only after the relevant adapter advertises safe mutation and rollback capability.

### 2026-06-23 — Encrypted DNS guidance suite hit embedded validator EOFs

- Signal: `vrooli scenario test network-manager` for the encrypted-DNS guidance slice completed 15/17; embedded contracts failed with `cli-health validation RPC failed: unavailable: unexpected EOF`, and embedded storage failed with `storage-manager validation RPC failed: unavailable: unexpected EOF`. Direct `cli-health validate scenario network-manager` passed, while direct `storage-manager validate scenario network-manager` returned `deadline_exceeded` awaiting response headers.
- Impact: API/CLI/UI/proto/SDA/requirements/unit-health/ui-health checks passed directly, and live CLI smoke verified the new guidance RPCs. The comprehensive suite remains blocked by external validator instability, not by localized guidance behavior.
- Next action: Track under existing embedded cli-health EOF (`knw-1782239328731253209`) and storage-manager EOF/deadline (`knw-1782239066919260424`, `knw-1782240251178806531`) reports; rerun the full suite after those validators are responsive.

### 2026-06-23 — Household policy schedules are advisory

- Signal: Phase 11 now persists household policy profiles and evaluates schedules for targets, but schedule evaluation returns manual-required intent instead of applying live DNS/router changes.
- Impact: Operators can model household profiles and see active/inactive windows through API/CLI/UI without unsafe automation. Actual scheduled enforcement still depends on governed AdGuard/router adapters with rollback support.
- Next action: Wire profile enforcement only after a resolver/router adapter advertises safe mutation and rollback capability.

### 2026-06-23 — Phase 11 suite hit embedded validator EOFs

- Signal: `vrooli scenario test network-manager` run `20260623-202256-d05ab551` completed 15/17; embedded contracts failed with `cli-health validation RPC failed: unavailable: unexpected EOF`, and embedded storage failed with `storage-manager validation RPC failed: unavailable: unexpected EOF`.
- Impact: Direct API/CLI/UI/proto/SDA/requirements/unit-health/ui-health checks passed, direct `cli-health validate scenario network-manager` passed before and after the suite, and direct `storage-manager validate scenario network-manager` reproduced the known EOF. The comprehensive suite remains blocked by external validator RPC instability, not by a localized Phase 11 profile/schedule failure.
- Next action: Track under existing embedded cli-health EOF (`knw-1782239328731253209`) and storage-manager EOF/deadline (`knw-1782239066919260424`, `knw-1782240251178806531`) reports; rerun the full suite after those validators are responsive.

### 2026-06-23 — Product UI is live but still conservative

- Signal: The operator UI now has dashboard, snapshot, resolver/policy, devices, optimization, and settings/privacy pages backed by generated Connect clients. Mutating surfaces remain preview/approval-oriented and reflect the same conservative backend behavior as CLI/API.
- Impact: Operators can inspect and exercise P0 workflows from the UI, but live DNS/router mutation still depends on governed AdGuard/router adapters. The UI should not imply filtering or optimization has been applied unless backend capabilities confirm it.
- Next action: When a live AdGuard policy adapter lands, extend UI confirmation tests around real supported apply/rollback states.

### 2026-06-23 — Privacy retention is storage-backed but query logs are absent

- Signal: Privacy now persists retention/visibility settings and sweep audit records. Sweeps prune expired non-baseline snapshots, but DNS query-level logs do not exist yet and optimization-specific retention pruning is not wired.
- Impact: P0 privacy defaults are fail-closed and storage-backed before broad UI exposure. Query-log retention is recorded as disabled/no-op until a governed query-log source exists.
- Next action: Extend retention sweeps to optimization ledgers if experiments begin accumulating long-lived before/after evidence.

### 2026-06-23 — Optimization apply is conservative outside AdGuard global DNS filtering

- Signal: Optimization now persists runs/candidates/approval/rollback ledgers, requires a baseline snapshot, captures read-only candidate and after snapshots, scores reliability-first evidence, and can apply/rollback the supported AdGuard global DNS filtering candidate through the policy adapter. Other candidates return `manual_required` until a resolver/router adapter can prove safe apply and rollback.
- Impact: Operators can compare candidates without unsafe automation and can exercise AdGuard-backed rollback for the supported DNS filtering candidate. Network Manager still does not claim router-wide or client-specific optimization until router/client adapters are proven.
- Next action: Add router/client-specific optimization only after those adapters report safe mutation, identity mapping, and rollback support.

### 2026-06-23 — storage-manager validator returns unexpected EOF

- Signal: `storage-manager validate scenario network-manager` exited twice with `Error: validate scenario "network-manager": unavailable: unexpected EOF` during Phase 7 validation.
- Impact: API/CLI tests, builds, `cli-health`, SDA health, requirements validation, and `unit-health --execution` passed, but static storage validation could not produce a current verdict for this slice.
- Next action: Filed `bug-inbox/unexpected-error/storage-manager-network-manager-eof` as `knw-1782239066919260424`; rerun storage-manager after the external validator is healthy.

### 2026-06-23 — storage-manager validator deadline during Phase 8

- Signal: `storage-manager validate scenario network-manager` failed with `deadline_exceeded` while awaiting `ScenarioValidationService/ValidateScenario` response headers.
- Impact: API/CLI tests, builds, `cli-health`, SDA health, requirements validation, and `unit-health --execution` passed, but standalone static storage validation again could not produce a current verdict for this slice.
- Next action: Filed `bug-inbox/unexpected-error/storage-manager-network-manager-deadline` as `knw-1782240251178806531`; rerun storage-manager after the external validator is responsive.

### 2026-06-23 — storage-manager validator EOF during Phase 9

- Signal: `storage-manager validate scenario network-manager` exited with `Error: validate scenario "network-manager": unavailable: unexpected EOF` during the operator UI slice.
- Impact: The UI/API/CLI/unit/requirements/SDA/ui-health gates passed, but standalone storage validation could not produce a current verdict. This matches the previously filed storage-manager EOF class.
- Next action: Track under the existing storage-manager EOF/deadline reports (`knw-1782239066919260424`, `knw-1782240251178806531`) and rerun after the external validator is responsive.

### 2026-06-23 — Orientation-triggered suite hit embedded ui-health once

- Signal: `vrooli scenario orient network-manager` spawned comprehensive run `20260623-193623-e4394270`, which completed 15/17 with embedded `phase-ui-health` reporting one error/blocker and `phase-storage` reporting the known EOF. Direct `ui-health validate scenario network-manager --json` passed before and after this run.
- Impact: The UI product surface still validates directly at L5, but orientation/scaffold-health remains blocked by the comprehensive suite result until embedded validator instability and storage-manager EOF are gone.
- Next action: Do not change UI code based on the embedded-only failure unless it reproduces through direct `ui-health`; track storage under the existing storage-manager reports and embedded validator behavior under prior suite-instability notes.

### 2026-06-23 — scenario contracts phase returned cli-health EOF

- Signal: `vrooli scenario test network-manager` completed 16/17 with only `phase-contracts` failing: `cli-health validation RPC failed: unavailable: unexpected EOF`. Direct `cli-health validate scenario network-manager` passed before and after the suite.
- Impact: The scenario-local CLI contract is clean, but the full suite verdict was failed due to an embedded validator/RPC EOF.
- Next action: Filed `bug-inbox/unexpected-error/test-genie-contracts-cli-health-eof` as `knw-1782239328731253209`; rerun the suite once after confirming direct cli-health passes.

### 2026-06-23 — Phase 8 scenario suite hit embedded validator failures

- Signal: `vrooli scenario test network-manager` after the Home Automation slice completed 13/17. Direct `cli-health validate scenario network-manager` passed, direct `ui-health validate scenario network-manager` passed after rebuilding the UI Health CLI, standalone `storage-manager` timed out, and the embedded suite reported contracts EOF, storage EOF, standards timeout, and UI handshake failure.
- Impact: After increasing the standards phase timeout from 120s to 300s and rerunning, the suite improved to 16/17: contracts, UI Health, standards, and all code/product phases passed; only embedded storage-manager EOF remains.
- Next action: Rerun the full suite after storage-manager is responsive; current storage-manager timeout/EOF evidence is filed under `knw-1782239066919260424` and `knw-1782240251178806531`.

### 2026-06-23 — Inventory discovery source is conservative

- Signal: Device inventory has persisted device/group records and identity reconciliation. As of 2026-06-24, production discovery can import AdGuard Home configured and auto clients when the governed resolver backend and secret reference are configured.
- Impact: Operators can refresh device inventory from AdGuard evidence without fake LAN scan data or query-level DNS logs. IP-only auto clients remain low-confidence observations, and absent/unreachable AdGuard still returns persisted inventory with explicit findings.
- Next action: Add router/client usage evidence and client/group policy mapping only after identity confidence and rollback semantics are explicit.

### 2026-06-23 — Snapshot throughput and gateway probes are capability gaps

- Signal: P0 snapshot now runs standard-library read-only probes, but gateway reachability and throughput are marked unsupported/unavailable unless a platform adapter or privacy-reviewed measurement backend is added.
- Impact: Operators get honest baseline evidence without fake throughput or privileged ICMP/router assumptions.
- Next action: Implement host/router/manual adapter capabilities before treating gateway or throughput as measured values.

### 2026-06-23 — AdGuard Home resource still needs implementation decision

- Signal: PRD selects AdGuard Home as the first resolver backend. Network Manager can now store governed backend config by `token_ref`, run dry-run setup, and expose conservative unverified health, but no resource-backed AdGuard HTTP client is connected.
- Impact: P0 can record backend and policy intent safely, but cannot claim filtering is active or apply persistent upstream/filter changes until a governed resource-backed client confirms them.
- Next action: Decide whether AdGuard Home is a first-class Vrooli resource or resource-backed adapter, then connect the client through secret/resource governance.

### 2026-06-23 — Client-specific policy adapter support is still gated

- Signal: Policy preview, approval, apply, pause/resume, and rollback persist change plans and audit records. The production AdGuard adapter now supports global user-defined rules and global protection pause/resume with rollback handles, but returns `unsupported` for client/group targets until AdGuard client identity mapping is implemented.
- Impact: Operators can safely test global inert rules such as `example.invalid` through rollback-backed policy flows, but household/group policy enforcement should remain advisory until devices are mapped to AdGuard clients.
- Next action: Implement AdGuard client inventory import, then translate client/group profile intents only after identity confidence and rollback behavior are explicit.

### 2026-06-23 — First router adapter not selected

- Signal: P0 intentionally uses manual router guidance; P1 needs one explicit router adapter.
- Impact: Router-enforced DNS rules and Wi-Fi/router changes remain manual until a platform is chosen.
- Next action: Select based on first real deployment environment.

## Architecture Drift

No drift yet. The template example domain has been removed; the main risk is leaving scaffold responses in place without requirement-tagged domain tests.

## Cross-references

- [`DECISIONS.md`](DECISIONS.md)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
