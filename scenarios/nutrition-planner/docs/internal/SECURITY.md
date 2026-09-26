# Security — Nutrition Planner

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

The working product name is **Nooch** (formerly Daily; decision D-042). Scenario id is `nutrition-planner`.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

**Status: designed, partly built, not verified.** The 2026-09-22 audit
([`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3) found that handlers do derive
ownership from a server-side principal, but the local runtime has **no
principal at all**, so every workspace RPC returns `401 unauthenticated`
(blocker B1), and no Connect handler has a test. Every mitigation below
describes the required behavior and the requirement that carries it; "by-design"
is not "tested" until a passing `AT-*`/`ACT-*` test, handler test, or schema test
proves it. The v2.0 redesign adds media, generation, offline, and calendar
surfaces whose rows are marked **(redesign)**.

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
| User food photos and original imported media (redesign) | **high** | media | Can show the home, people, and location metadata. Private to the workspace by default; originals kept for recrops (R17.4, R25.3). |
| Generated meal/scene images and their prompts (redesign) | medium | media / generation jobs | Prompts may be derived from private recipes; results are workspace-private unless they are curated catalog assets. |
| Curated scene, editorial, equipment, and ingredient assets (redesign) | low | bundled asset manifest | Public decorative assets; shared artwork never implies shared recipes (R25.3). |
| Equipment and kitchen inventory (redesign) | medium | kitchen | Reveals the household and what is in the home. |
| Cooking sessions, step completions, timers (redesign) | low | cooking | Routine timing; still workspace-scoped. |
| Offline outbox and cached shopping/recipe data (redesign) | **high** | UI outbox / service worker | Personal data at rest in the browser; must be partitioned per account and workspace and cleared on sign-out (R22). |
| personal-planner calendar links (redesign) | medium | calendar adapter | External event ids and times; non-authoritative references in exports (R24, R25.2). |

## Auth And Authorization

The single-user local tier runs without an external sign-in, but
**ownership is still a server-side property from day one** (`SYS-03`,
`OT-P0-001`).

**Local authentication profile (decision D-032).** The API only installs the
`authn` middleware when the lifecycle supplies authentication configuration
(`api/main.go`), and `.vrooli/service.json` currently declares **no
`authentication` block**, so no principal exists and every request is rejected
(blocker B1). The fix is to declare the platform profile — `hybrid` with the
default mode `personal_local`, as `scenarios/git-control-tower/.vrooli/service.json`
does — following repository `docs/concepts/IDENTITY-AND-AUTHENTICATION.md`.
`personal_local` is not an authorization bypass: the principal is the local OS
user plus the app-private runtime, and every rule below still applies. Never add a
handler-level "skip auth in dev" branch or a hard-coded owner subject.

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
| Image model credentials (redesign) | Owned by ai-gateway / resource scenarios | never held here | Nooch calls image-tools, which routes models through ai-gateway roles; this scenario stores only the image-tools base URL and never a provider key (D-029, R18.2). |
| personal-planner access (redesign) | Platform auth between scenarios | optional | Uses the scenario-to-scenario auth the platform provides; revoked access marks links "reconnection needed" and leaves nutrition tasks intact (R24.3). |

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
| SSRF through URL ingestion | Server-side requests to internal/private destinations | Restrict schemes; reject local, private, and link-local targets; validate resolved address and each redirect; bound size, redirects, and time; isolate outbound credentials (`AI-03`, `ACT-057`) | by-design |
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
| Malicious or oversized image upload (redesign) | Decompression bomb, polyglot file, or oversized media exhausts storage or executes in a viewer | Validate MIME by content (not extension), decoded dimensions, byte and decompression limits, and an allowed-format list before storing; re-encode derivatives; strip metadata from shared renditions (R25.3, AT-049) | planned |
| SVG or rich text carrying script (redesign) | Stored XSS through imported recipes, notes, or SVG media | Render imported rich text as plain text or a safe allowed subset; never inline user SVG — rasterize or sanitize it (R25.3, AT-049) | planned |
| Private media read from another workspace (redesign) | Photo or generated-image disclosure | Every media and job-result lookup checks workspace authorization; signed URLs are short-lived, never cached indefinitely, and never used as public image identities (R22, R25.3, AT-044) | planned |
| Offline cache shown to the next person (redesign) | Previous user's meals or list visible after sign-out or account switch | Service-worker and IndexedDB data partitioned by account and workspace; purged on sign-out per policy; an offline account switch shows nothing private (R22, AT-050) | planned |
| Unbounded generation spend (redesign) | Surprise cost from repeated clicks, retries, or concurrent jobs | Generation Off by default; quote before start; atomic reservation covering the maximum attempt cost; idempotency key; no generation on view, theme toggle, search, hover, or resize (R18, AT-040, AT-041) | planned |
| More personal data than an image task needs (redesign) | A generation request leaks targets, supplements, or history | Send only the reviewed visual summary, vessel, and appearance; never the target history or supplement schedule (R25.3) | planned |
| Demo content adopted as real data (redesign) | Invented foods, prices, or targets shown as the user's facts | Demo fixtures load only through the explicit demo seed into a separate demo workspace; the normal app never loads them (R27.1, AT-001) | planned |
| Calendar retry creates duplicate events (redesign) | Spam and conflicting schedules in personal-planner | Stable idempotent external key per task purpose; reconcile by key after a timeout before creating again (R24.3, AT-045) | planned |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| **The posture is designed, not verified.** Handlers derive ownership, but no Connect handler has a test and the local runtime cannot authenticate (B1). | high | The authentication profile is declared and every security-relevant `AT-*`/`ACT-*` test passes (`ACT-035`, `ACT-036`, `ACT-057`, `AT-044`, `AT-049`, `AT-050`), plus schema tests for ownership. |
| Workspace-scoped tables carry `workspace_id` without a foreign key to `workspaces`, and most multi-statement writes are not transactional | medium | Versioned migrations add the keys and transactions (D-034, blockers B12–B13). |
| Media, offline cache, and calendar surfaces do not exist yet, so their controls are unbuilt | high | Each redesign row above becomes "tested" only with its acceptance case. |
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
- [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) — current-state blockers B1–B14 and the redesign build order
- [`../reference/product-specification.md`](../reference/product-specification.md) — R18, R22, R24, R25.3, and Appendix A sections 16, 17.4, 18, and 21.2
