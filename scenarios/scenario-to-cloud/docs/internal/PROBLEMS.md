# Problems / Risks

## Work ladder — VPS delivery certification execution, 2026-09-09

- Rung: W1 (obligations declared, evidence attaching per phase).
- Evidence: PRD operational targets OT-P0-007..017 and requirement modules 10–18 declare the launch contract; `business-health validate scenario scenario-to-cloud` passes. Package-lane certification receipts exist for RUN, REACH, RELEASE (via P10 tests), DATA, SECRET, EDGE and GOV families under `certification/evidence/`; `api/certification` reports `not_ready` (158 missing, 55 stale, 53 unavailable of 266 required cells; `certification-report-2026-09-09-final.json`) because every QEMU, real-VPS, relay, BAS-UI, independent-review and soak lane is pending external inputs.
- Blocker: no disposable VPS, delegated DNS zone, QEMU host tools, independent reviewer or soak window is available (plan-artifacts `ledgers/external-inputs.md` EXT-01..11). The only enrolled cloud target is the production LPBS host and is never used as a fixture.
- Measured: 2026-09-09.

## Open Issues

- **Remaining SSH-era surfaces**: the executor runs every effect as typed argv through `api/reach`; the preflight disk routes and their CLI/UI consumers are retired, while the remaining transport, repair, and error-classifier work stays tracked in deletion-ledger rows DL-02/DL-03/DL-08.
- **Pre-enrollment egress check**: preflight cannot observe outbound reachability through the read-only observation set before a host is enrolled; the check reports `warn` and `host.prepare` surfaces a blocked egress at apply time.
- **Legacy data bindings**: the production deployment's mutable directories are only inventoried (`cloud-target data inventory`); mapping them to declared `persistent_data` bindings has not been executed.
- **Operator console**: the console, plan review and durable operation view shipped (phase 19); the independent operator walkthrough (EXT-08) and a real BAS run are still pending.
- **Governance**: cloud → Deployment Manager calls need a service bearer for the `scenario-to-cloud` principal; Deployment Manager now independently verifies Ed25519 receipt signatures, while publication remains fail-closed until a qualification lane drives the ramp Driver.
- **Production TLS renewal**: the live edge observation on vrooli.com reports `renewal_failed` (tls-alpn challenge) while the certificate is valid until 2026-12-07.
- **Scope governance**: `cli/manifest.json` now declares the scenario scopes; destructive verbs (`deployment apply|rollback`, `recovery-points restore`) are run-eligible only with confirmation under `scenario-to-cloud:destructive`, so a governed program applying against a fixture needs a session grant.
- **Browser-automation dependency**: UI smoke and BAS journeys require browser-automation-studio; only observer API cases are registered today.
- **CLI reference catalog**: the manifest and generated command reference contain the recovery, edge, credential and inspection leaf commands, but the current CLI Health catalog does not expose all of them. Its validator reports `unknown_command` for those current paths, and a scenario-scoped reindex is denied because the search control plane is unavailable. The generated examples now quote required placeholders; rerun the docs phase after the owner restores reindex authority.

## Resolved in current execution

- **Health call cost (2026-09-11)**: the health path now compares the declared
  scenario version and defers the full local bundle fingerprint to the explicit
  release-freshness workflow; the version-only regression proves it does not
  hash the local bundle on interactive health calls.

## Deferred (explicitly out of scope for the certification plan)

- Delta uploads, managed-service swaps, bastion hosts, Kubernetes/multi-region, billing and mobile packaging.

## Preserved visual sources

Historical HTML and editable canvas sources live beneath the protected
control-plane runtime home (`~/.vrooli` for the invoking user).
These are dated evidence or design supplements; this relocation does not
change plan status or establish release readiness.

- `plan-artifacts/docs-html-progress-20260908/scenarios/scenario-to-cloud/docs/internal/cloud-readiness-2026-09-07.html`

The originating installation must retain these artifacts with its durable backups.
Exact originals and SHA-256 hashes are in `backups/docs-html-progress-20260908/manifest.json`.
