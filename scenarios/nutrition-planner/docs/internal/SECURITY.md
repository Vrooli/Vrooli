# Security — Nutrition Planner

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

The working product name is **Daily**. Scenario id is `nutrition-planner`.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

**Status: designed, not verified.** This is the intended posture for a
product whose implementation has not started (see
[`PROBLEMS.md`](PROBLEMS.md)). Every mitigation below describes the
required behavior and the requirement that carries it; none is evidence
that the behavior exists yet. Do not read "by-design" as "tested" until a
passing `ACT-*` test, handler test, or schema test proves it.

**The one-paragraph summary.** This scenario holds private personal data: what a person eats, the dietary
rules and allergies they must not violate, body-related targets, supplement
schedules, receipts, and prices. The dataset is **non-regenerable** — it
cannot be recomputed from any other source. Three properties carry the
posture: the runtime is self-hosted so data never leaves the user's box by
default, every user-owned aggregate is scoped to a **server-derived**
workspace so one account can never read another's data, and external
content (URLs, labels, model output) is treated as untrusted input that
must pass the same domain validation as manual edits. **Integrity of the
user's rules matters as much as confidentiality**: a nutrition suggestion
that silently violates an exclusion is a safety-relevant failure, not just
a bug.

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Food/intake log — what was eaten, when, how much | **high** | intake/feedback | Discloses diet, health context, and daily routine. Non-regenerable. |
| Dietary restrictions, allergies, exclusions | **high** | profile/rules | Safety-relevant. An unauthorized change could produce an unsafe suggestion. |
| Nutrition targets, supplement schedules | **high** | profile/rules | Body/health-adjacent, personal. |
| Recipes and revisions (including user source notes) | medium | recipe library | Reveals preferences, restrictions, household, and possibly location. Non-regenerable. |
| Receipts, prices, purchases, spend | **high** | inventory/costs | Financial behavior and shopping location. |
| Inventory and prepared-batch state | medium | inventory/costs | Reveals what is in the home and when. |
| Label/product images and their OCR text | **high** | food catalog | Can contain personal surroundings; provider text is untrusted. |
| Provider/AI credentials | **critical** | adapters | Never stored by this scenario; references into the platform secret store. |
| AI/provider proposals and source artifacts | medium | provider/job adapters | Untrusted until validated; may contain injected instructions. |
| Export bundles (`daily.*`) | **high** | transfer | A complete copy of owned data; must exclude secrets and billing identifiers. |
| Job records and operation IDs | low | jobs | Timing and counts; not personal content, but must not carry it. |

## Auth And Authorization

The single-user local tier may run without an external auth provider, but
**ownership is still a server-side property from day one** (`SYS-03`,
`P0-001`):

- **Derive actor and workspace scope from the authenticated server
  session.** A client-supplied `workspaceId` or owner ID is a *requested
  scope, never proof of authorization* (spec §17.5). Never trust a
  client-supplied owner ID.
- **Validate ownership of every referenced aggregate**, including nested
  references inside imports and AI proposals: recipes, revisions, files,
  plans, batches, prices, stock events, sources, and jobs. A foreign nested
  reference must deny the whole operation, not silently drop the item.
- **No cross-workspace reads, writes, or downloads.** This includes
  background job result URLs, generated PDFs, and export downloads, which
  are as much an access surface as an API read (`SYS-03`).
- **Authorization belongs at the API/service layer.** The CLI and UI are
  translation layers over the same services and must enforce nothing
  locally; the CLI is equally reachable.
- **Do not expose another workspace's existence** through detailed
  authorization errors (spec §17.5 error guidance). Return `UNAUTHORIZED`
  or `NOT_FOUND` without leaking which.
- **Restore/import cannot grant authority.** An import or restore into the
  authorized workspace must not import original account membership,
  credentials, entitlements, or access tokens (`SYS-05`, `DOM-07` #11).

**Public/community recipe sharing is deferred** (`OT-P2-004`). Until it is
deliberately designed, saved recipes and source attachments are private to
their workspace. A future sharing feature must handle attribution,
permission, redaction, and copied-versus-linked revisions before any public
catalog exists. Do not add a public read path as a convenience.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| AI/model provider credential | Platform secret store | optional (R2) | Read at use and never copied into scenario state or the database. |
| Nutrition/price provider credentials | Platform secret store | optional (R2) | Same handling; respect provider limits. |
| External HTTP fetch credentials | Platform secret store | optional (R2) | Isolated per outbound adapter; never sent to user-supplied destinations. |
| Auth/session material | Platform auth | per deployment | Never persisted by this scenario; the API verifies, it does not mint. |
| Billing identifiers (R3) | Billing provider | deferred | Excluded from portable meal backups and exports (`OT-P2-002`). |
| Database encryption key | Platform secret store | deferred | See Security Gaps. |

Rules that hold for every secret:

1. Secrets are server-side only. They never appear in the client bundle, an
   export, a log, an error message, a generated PDF, or a URL returned to a
   client (`SYS-03`, `OBSERVABILITY.md`).
2. A credential is read at use time and never copied into scenario state.
3. An adapter that cannot reach its credential reports **unavailable with a
   reason**; it must never fall back to an unauthenticated path or return
   an empty result as if it succeeded.
4. Missing configuration for an optional provider **disables that
   provider** and leaves the manual path working — it does not disable the
   product.

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Cross-workspace access via a forged client owner ID | Private diet/health/financial data disclosed or corrupted | Scope is derived server-side from the session; client IDs are requested scope only (`SYS-03`, `ACT-036`) | by-design |
| Foreign reference smuggled in through import or AI proposal | Cross-workspace read/write through a nested reference | Ownership validated on every referenced aggregate, including nested references (`SYS-03`, `ACT-036`) | by-design |
| Prompt injection via a fetched recipe page or imported document | The page instructs the app/model to change rules, buy, or exfiltrate | Fetched content and document text are untrusted data; scripts/instructions have no authority; model output is a proposal that passes domain validation (`AI-02`, `AI-03`, `ACT-057`) | by-design |
| SSRF through URL ingestion | Server-side requests to internal/private destinations | Restrict schemes; reject local/private/link-local targets; validate resolved address and each redirect; bound size, redirects, and time; isolate outbound credentials (`AI-03`, `ACT-057`) | by-design |
| CSV injection in a shopping/export file | A spreadsheet opens a formula cell and executes | Export cells are sanitized/escaped under the documented policy for formula-like cells (`UX-DAT-04`, `ACT-033`) | by-design |
| A model receives more personal data than the task needs | Unnecessary disclosure to a provider | Send only the task-relevant subset; use opaque local references instead of account identifiers; disclose provider behavior in settings (`SYS-04`) | by-design |
| Log leakage of recipes, receipts, labels, or tokens | Personal disclosure through an operational surface | Logs carry operation IDs, timings, counts, codes, and safe revision refs; personal content is prohibited below bounded debug capture (`SYS-04`, `OBSERVABILITY.md`) | by-design |
| Stale write silently overwriting newer data | Loss of user work | Expected-revision checks with all-or-nothing transactions; conflict is distinct from network error (`SYS-01`, `SYS-02`, `ACT-035`) | by-design |
| Retry double-applies a purchase, consumption, import, or replan | Duplicated stock, spend, or plan effects | Idempotency keys scoped to workspace and operation type; durable external transaction identities (`SYS-02`, `DOM-07` #12) | by-design |
| Historical revision rewritten by an edit | Past intake/plans change silently | Immutable content revisions plus current projection; corrections are new records with a link (`DOM-02`, `DOM-07` #5) | by-design |
| Local database exfiltration | Full personal dataset disclosed | Self-hosted with no network exposure by default; at-rest encryption deferred (see Gaps) | partial |
| Backup handling of a non-regenerable dataset | Permanent loss | Declared non-regenerable in DATA and the Runbook; restore uses a checkpoint (`SYS-05`) | by-design |
| Unsafe file/label upload | Malicious or oversized upload affects storage | Bounded MIME/size limits; bytes isolated behind the BlobStore seam; no macro execution (`AI-03`) | planned |
| Public sharing leaks private recipes | Accidental disclosure | Public sharing is deferred; attachments private by default (`OT-P2-004`) | by-design |
| Provider card/money actions | Financial harm | No checkout or order placement exists; purchasing is a shopping plan and export only (`BIZ-02`, spec §3.2) | out-of-scope by design |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| **The entire posture is designed, not verified.** No implementation or security tests exist yet. | high | Every P0/P1 security-relevant `ACT-*` test passes (`ACT-035`, `ACT-036`, `ACT-057`), plus schema tests for ownership. |
| No at-rest encryption of the SQLite database | medium | Any deployment where the host is not solely the user's own machine, or a shared device. |
| No auth model for a networked/multi-user deployment | conditional | Before any hosted or multi-user tier. The single-user local tier is the only supported target today. |
| Export path not yet threat-modelled as a bulk disclosure surface | medium | Before import/export ships; sanitization and secret-exclusion need their own tests. |
| SSRF controls specified but unbuilt and untested | high | With the R2 URL-ingestion adapter; test against loopback, link-local, and redirect cases. |
| CSV-injection policy specified but unbuilt | medium | With CSV export (`ACT-033`); test formula-like cells, quotes, and non-ASCII. |
| Prompt-injection defenses are architectural intent, not proven | high | With R2 AI adapters; adversarial fixtures must fail to mutate settings or trigger actions (`ACT-057`). |
| No retention/deletion policy for a commercial deployment | medium | Before R3; define account deletion, backup retention, and deletion timing from the actual hosting policy. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — what may and may not be emitted
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — backup, restore, and recovery
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
- [`DECISIONS.md`](DECISIONS.md) — why workspace scope is server-derived and providers are optional
- [`../reference/product-specification.md`](../reference/product-specification.md) — sections 16, 17.4, 18, and 21.2
