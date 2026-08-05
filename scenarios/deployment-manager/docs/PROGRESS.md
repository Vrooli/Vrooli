# Deployment-manager progress

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
