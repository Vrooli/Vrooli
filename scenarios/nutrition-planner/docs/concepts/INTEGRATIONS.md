# Integrations — Nutrition Planner

This document is the canonical dependency contract for resources, other scenarios, and
third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on, at runtime and during development?
- Which dependencies are required versus optional?
- Which domain uses each dependency, and through which contract?
- What degrades when a dependency is absent, and how is that reported?
- Where is the dependency declared or configured?

The core loop — setup, capture, eligibility, planning, groceries, inventory, cooking, intake,
portability — runs on local data with bundled curated artwork. Every external integration is
optional at runtime, sits behind an adapter, and degrades to a working fallback. **Today no
adapter below is wired into a production path** (2026-09-22 audit); the USDA and URL-fetch
providers exist only as test-reached libraries.

**Capability-state rule (R21.3).** The API reports what each adapter can actually do right
now — configured, unconfigured, unavailable, or access revoked — through its capability and
diagnostics endpoints. The UI reads that state and shows an honest disabled explanation or
omits the action; an unconfigured adapter never produces a dead button, a fake success, or a
hardcoded provider response presented as real.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | When Absent |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | every persistence-backed domain | `api-core/database`, per-domain schemas and migrations | API reports unhealthy; UI shows a load error, never a fake empty workspace. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile, `vrooli scenario start` | Direct binary starts bypass ports and health checks; unsupported. |
| Platform authentication profile | platform | yes | application-services, workspace | `authentication` block in `.vrooli/service.json` (hybrid, `personal_local` default) — D-032 | Every workspace RPC returns 401 — the state today (B1). |
| `image-tools` | scenario | development: yes (curated artwork); runtime: no (P1) | media-and-scenes, generation | CLI and Connect services (`ai`, `ops`, `jobs`) | Bundled approved assets, editorial photos, and the minimal treatment keep every surface usable; generation shows a disabled capability state. |
| `ai-gateway` | scenario (behind image-tools; direct for assisted import) | no (P1) | generation (via image-tools), provider-and-job-adapters | Policy roles routed to `resource-openrouter` / `resource-ollama` | Generation and extraction unavailable, not failed; manual paths continue. |
| `personal-planner` | scenario | no (P1) | calendar-link | Verified adapter over its Connect API | Prep and cooking tasks stay untimed inside nutrition-planner. |
| `notification-hub` | scenario | no (P1) | provider-and-job-adapters, cooking-sessions | Opt-in delivery | In-app badge and in-app timer alerts only. |
| `document-manager` | scenario | no (P1) | transfer-and-documents | Optional attachment custody | Local blob storage. |
| `@vrooli/react-component-library` | shared package | yes (UI build) | ui | Linked package imports; contribution through its draft workflow | n/a — build dependency. |
| USDA FoodData Central | third-party API | no (P1/R2) | food-and-product-catalog | Search/detail adapter | Manual nutrient entry; unknown stays unknown. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None required at runtime | not-applicable | One private workspace on in-process SQLite with bundled artwork. | A second concurrent writer, outgrowing local storage, or a multi-tenant deployment. |
| `resource-openrouter`, `resource-ollama` | indirect, optional | Model execution for image generation and assisted import. nutrition-planner never calls them; ai-gateway routes to them, and image-tools calls ai-gateway. | Only if routing ownership changes. |

Provider credentials live in the platform secret store and are resolved by the owning scenario;
they never appear in nutrition-planner config, its database, exports, logs, or client bundles
(R25.3).

## Scenario Dependencies

### image-tools (with ai-gateway behind it)

- **Two uses.** (1) **Development-time asset production** of the curated kit —
  scene templates, composed Today scenes, editorial recipe photos, the equipment layer kit,
  ingredient icons — authorized by the operator on 2026-09-22 (D-029; plan in
  [`../internal/REDESIGN_PLAN.md` §7](../internal/REDESIGN_PLAN.md#7-production-artwork)).
  (2) **In-app opt-in generation** (OT-P1-007) through a nutrition-planner generation adapter.
- **Commands** for production work: `image-tools ai generate` (text to image), `ai edit`
  (identity-preserving: place a recipe photo into a scene template, relight day to evening),
  `ai inpaint` and `ai object-removal` (empty counters, cabinet infill), `ai bg-removal`
  (cutouts, ingredient icons), `ai upscale`; `image-tools ops resize|crop|convert|compress`
  (renditions, WebP/AVIF); `image-tools jobs get|wait|list|cancel|download` (durable async
  jobs — block once with `jobs wait`, never poll). Confirm flags with `image-tools ai <op>
  --help`.
- **Runtime contract.** The adapter calls image-tools' Connect services (proto under
  `packages/proto/schemas/image-tools/v1/`, including `ai` and `jobs`) and maps their job
  states onto nutrition-planner's GenerationJob; quotes, atomic budget reservation, dedup
  keys, review, and activation stay in nutrition-planner (R18). The adapter never calls a
  model provider directly and never holds provider credentials.
- **Rules.** Generation policy defaults Off; no spend from views, theme changes, search, hover,
  or resize; only task-relevant data is sent (a reviewed visual summary and references, never
  targets or supplement schedules); a style reference is cropped to the photograph region so
  the model does not copy UI; every result carries provenance and reported cost into the asset
  manifest and passes review before activation.
- **Absent or unconfigured.** Curated bundled assets and the editorial/minimal treatments cover
  every surface; Settings › Generation & usage shows the capability as unavailable; AT-043.

### ai-gateway (direct)

Optional model routing for the assisted-import pipeline (capture → extract → validate →
compare → review → apply, AI-02) and explanations. Proposals are data, never domain writes;
model output never authorizes a rule, price, target, stock change, or recipe edit (D-004).
Absent: paste-text and structured-file imports still work without a model (R10.3).

### personal-planner

- **Boundary.** nutrition-planner owns meals, recipes, servings, requirements, and prep and
  cooking tasks; personal-planner owns calendar timing and semantics (R24.2; see
  [`DOMAINS.md`](DOMAINS.md#ownership-boundary-with-personal-planner)).
- **Adapter capabilities** (R24.3): create, read, update, and cancel an event; observe changes
  (subscribe or poll); deep-link to the event; optional actual-time feedback. personal-planner
  exposes calendar and work services under `packages/proto/schemas/personal-planner/v1/`
  (for example `calendar/` with allocation preview/apply RPCs); the exact endpoints are
  **discovered and verified locally before use** — nothing here asserts their semantics.
- **Identity.** One stable idempotent external key per source task and purpose; after a timeout
  reconcile by key or stored operation before creating another event (AT-045). Link records
  hold source task and revision, external event id and revision, sync state, and operation id.
- **Absent, unconfigured, or revoked.** Prep tasks stay untimed and fully usable; Settings ›
  Integrations shows the state and a reconnect action; nutrition-planner never builds a
  competing calendar.

### notification-hub

Opt-in reminders (timer expiry where the platform supports it, scheduled drafts, prep). No
messages are sent during development without authorization (R24.4). Absent: in-app badges and
in-app timer alerts; the timer settings state when background alerts are not reliable (R22).

### document-manager

Optional custody for attachment bytes and full media backups. Absent: local blob storage behind
the seam in [`../internal/SEAMS.md`](../internal/SEAMS.md).

### react-component-library

The UI builds on `@vrooli/react-component-library` primitives (the shell is `AppShell/2` today)
when they fit the Nooch design language. A missing reusable primitive is contributed through
the library's draft workflow (`react-component-library components draft-begin <asset>`), never
by editing a release directory; nutrition-specific components stay in this scenario. Gaps are
recorded in [`../reference/component-library-gaps.md`](../reference/component-library-gaps.md).

## Third-Party Services

No third-party service is required at any release. Entries are **interfaces, not promised
vendors**; an adapter is built only after its access, terms, coverage, and maintenance cost are
evaluated.

| Service | Status | Contract |
|---|---|---|
| USDA FoodData Central | optional (P1/R2) | Search/detail adapter returning pinned local source revisions with source identity, data type, version, observed date, nutrient mapping, caching, and error classification; credentials server-side. A library exists (`api/internal/providers/usda.go`) but no production path calls it. |
| Price sources | optional (P2) | Structured observations with package, retailer, conditions, currency, date, source. Manual package prices always work. |
| Receipt / order sources | optional (P2) | Opt-in, source-specific scope; staged proposals deduplicated by source transaction identity plus normalized line; old receipts never imply current stock. |
| Recipe URLs | optional (P1) | Controlled server-side fetch (below); the URL is stored even when fetching is unavailable. A library exists (`api/internal/providers/urlfetch.go`). |
| Named retailers | potential only | No checkout or ordering in any release. |

### Controlled URL And Attachment Ingestion

User-supplied URLs are fetched through a controlled server adapter: restricted schemes; no
local, private, or link-local destinations; resolved addresses and every redirect validated;
bounded size, redirects, and time; isolated outbound credentials (OWASP SSRF guidance, R25.3).
Uploads are validated by content-sniffed MIME, decoded dimensions, byte and decompression
limits, and allowed formats; imported rich text and SVG render through a safe subset. Fetched
content and imported text are untrusted data: "ignore previous instructions" has no authority,
and nothing in an imported page becomes a domain action.

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | unreachable or ping error | `/health` unhealthy; UI load error, never a fake empty account. | health handler tests; boot E2E |
| Authentication profile | missing principal | Today: every RPC 401 (B1). Intended: `personal_local` resolves the local principal; foreign-workspace access stays denied (AT-044). | handler auth tests; cross-workspace tests |
| image-tools / ai-gateway | unavailable, over budget, malformed or incompatible output, canceled after dispatch | No activation; reservation settled or held pending honestly; ordinary photo and minimal treatment remain; capability state shown. | adapter tests with recorded responses labelled simulated; AT-040–AT-043 with real configuration when available |
| personal-planner | timeout after remote commit, cross-date move, deletion, revoked access | Reconcile by stable key; review implications; unschedule only; reconnect state; untimed tasks keep working. | adapter tests; AT-045–AT-046 against a configured instance or a recorded blocked state |
| notification-hub | unavailable or rejected | Draft or timer state still persists; no duplicate reminders; in-app alert remains. | faked notification seam |
| document-manager | unavailable | Named failure; local blob fallback. | faked attachment seam |
| USDA FoodData Central | HTTP, auth, timeout, ambiguous match | Report unavailable; queue for manual resolution; never assign a vaguely similar food. | recorded contract tests; manual fallback tests |
| Price / receipt source | unavailable or malformed | Missing price stays unpriced; duplicate receipts cannot double-apply. | cost and receipt dedup tests |
| Controlled URL fetch | blocked destination, oversize, failure | URL preserved; fetch status separate from draft save; manual entry offered. | SSRF and limit tests (AT-049) |

Integration tests against simulated adapters are labelled as such; AT-040–AT-043 and
AT-045–AT-046 are not reported passed from hardcoded provider responses (R27.4).

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — module boundaries and the local authentication profile
- [`DOMAINS.md`](DOMAINS.md) — which domain owns each adapter
- [`DATA.md`](DATA.md) — asset manifest, privacy, and export of media metadata
- [`FLOWS.md`](FLOWS.md) — generation job and calendar link state machines
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — adapter seams
- [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) — artwork production plan (§7)
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-029 (artwork authorization), D-032 (authentication profile)
- [`../reference/product-specification.md`](../reference/product-specification.md) — R18, R19, R21.3, R24, R25.3, Appendix A §16
