# Deployment-manager progress

## 2026-09-04 — Outcome-linked learning

Implemented recommendations 1–3. The usage skill captures advice decisions and
verified outcomes through `vrooli-memory learning record`; the improvement board
adds failure recurrence, success effort, and advice-outcome measurements. Contexts,
unresolved tasks, missing evidence, and test provenance remain distinct. Targets
stay null until comparable operator baselines exist.

The shared implementation, scope provisioning receipt, capture/recall verification,
fixture evidence, and known Memory suite limitations are in the
[Vrooli Memory work record](../../vrooli-memory/docs/internal/PROGRESS.md). This scenario's five-phase validation passed
in Test Genie `20260904-213044-97c7f8cc`. Fresh runtime explanations and fixtures passed for both
setpoint programs. The skill divergence review resolves missing recall, evidence,
and baselines conservatively; no release gate or evidence floor was weakened.

## 2026-08-04 — Proto-first governance and evidence contract

The scenario now owns a per-domain SQLite governance ledger, generated
EvidenceService transport, exact-commit release gating, and a reference-only
shared evidence contract. scenario-to-desktop emits journey evidence with
ordered steps, screenshots, named degraded reasons, and producer-owned
recording links. The deployment-manager UI provides evidence review and
release views.

Validated locally:

- API and CLI health validators: passed.
- Structure validator: passed with non-blocking profile warnings.
- UI lint, type-check, build, and 85% coverage policy: passed.
- API and CLI unit suites: passed; aggregate Go coverage remains an open item
  recorded in `docs/internal/PROBLEMS.md`.

The remaining end-to-end proof requires provider-backed workflow, storage,
scenario suites, and baseline validation in an environment with the relevant
Test Genie and storage-manager providers.
