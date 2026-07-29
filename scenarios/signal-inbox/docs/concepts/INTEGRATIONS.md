# Integrations — Signal Inbox

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, all persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| ai-gateway | scenario | yes (degraded without) | enrichment, categories, retrieval | Embedding and completion requests | Capture still succeeds; classification falls back to `uncategorized` with a recorded reason and the signal is queued for re-classification. |
| search-hub | scenario | no | retrieval | `.vrooli/search.json` provider descriptor | Local search keeps working; only federated reach is lost. |
| image-tools | scenario | no | enrichment | Image-to-text request | Image signals are marked `needs-attention` for manual text entry rather than stored blank. |
| video-downloader | scenario | no | enrichment | Transcript-only request | Video signals are marked `needs-attention`. **Contract does not exist yet** — see Blocked Dependencies. |
| browser-automation-studio | scenario | no | sources | Authenticated session capture (tier 2 only) | Tier-2 adapters are unavailable; every other capture path is unaffected. |
| prompt-manager | scenario | no | triage | Team knowledge-entry write for intake routing | Routing is unavailable; disposition and annotation still work, and the signal stays `triaged`. |
| vrooli-events | scenario | no | all | Receipts published automatically by api-core | No integration code; nothing to fail. Correlation is simply absent. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (in-process) | required | Journal, categories, triage state, and FTS5 index. Not declared as a resource — it is embedded. | Revisit only if the corpus outgrows single-file storage (D-020). |
| ollama | required, via ai-gateway | Embedding and classification inference. Reached through ai-gateway rather than directly. | Revisit if ai-gateway changes its provider contract. |

No other resource is declared. In particular there is **no Postgres dependency**:
the predecessor scenario declared one and shipped `schema.sql` plus `seed.sql`,
and that was dropped deliberately (D-020).

## Scenario Dependencies

### Enrichment delegation

Every medium this scenario captures is owned by another scenario, and none of
their capabilities are reimplemented here (D-018). This scenario links **no OCR
library, no media downloader, and no browser driver**.

| Medium | Delegated To | Why not in-process |
|---|---|---|
| Image → text | `image-tools` | It owns image-to-text/OCR (`OT-P0-004`, shipped). `text-tools` owns document→text; the boundary is already drawn and non-overlapping. |
| Video → transcript | `video-downloader` | Media handling, format sprawl, and platform drift are its problem domain, not this scenario's. |
| Authenticated browser session | `browser-automation-studio` | Session handling is the highest-risk operation in the product; concentrating it in the scenario that specializes in it is both safer and less duplicated. |

The one extraction this scenario owns is plain HTML-and-text resolution, plus the
conversation share-link decoder recorded in
[`../reference/conversation-extraction.md`](../reference/conversation-extraction.md),
because no other scenario owns that shape.

### Measured delegation surface (2026-07-27)

- Image OCR is invoked as `image-tools analyze ocr <input> [--json]`. The
  input is a local image path; the signals BlobStore seam must materialize a
  temporary read-only file for that request and remove only the temporary copy
  afterwards, never the retained signal media.
- `ai-gateway inventory roles --json` exposed `embedding.default` (embedding)
  and `classify.routing` (classify/generate). Classification therefore uses the
  gateway classification route rather than a nearest-neighbour substitute.
- No `video-downloader` CLI is installed in this workspace, so a
  transcript-only operation cannot be measured. This leaves the existing P1
  dependency blocked; it does not authorize a local media-download workaround.

### Source-format measurement status

The browser bookmark format was measured on 2026-07-27 from an operator-supplied
Chrome export. It is Netscape Bookmark HTML, UTF-8 encoded, with nested
`DL`/`DT` lists: folder entries are `H3`, and bookmark entries are `A`.

| Source | URL field | Timestamp field | Stable identity | Other parsed structure | Ignored fields |
|---|---|---|---|---|---|
| Chrome bookmarks HTML | `A@HREF` | `A@ADD_DATE` (Unix seconds) | Normalized `HREF` | Folder path from enclosing `H3` hierarchy; title from anchor text | `ICON`, folder `LAST_MODIFIED`, and toolbar flags |
| X archive (2026-07-28) | `like.expandedUrl`; authored `tweet.id_str` is sufficient to construct its status URL | Authored `tweet.created_at`; the like payload has no saved-at timestamp | `like.tweetId`; authored `tweet.id_str` | `data/like.js` is a JavaScript assignment to an array of `{like:{tweetId,fullText,expandedUrl}}`; `data/tweets.js` is the analogous authored-post array | DMs, contacts, ads, profile/security/account data, inferred interests, and media bytes are not intake streams |

The Reddit parser is measured and intentionally saved-only. The X archive is
now measured: it contains likes and authored posts, but no bookmarks. The next
adapter implementation must validate this measured layout before writing and
must not infer an X bookmark stream from a missing file.

### Source sync configuration model (planned)

A **source** is an account or export family (for example, X or Reddit). A
source can expose several independently configured **streams**. A stream is one
specific kind of operator activity, such as X authored posts, X bookmarks, X
likes, Reddit saved posts, or Chrome bookmarks. The stream—not the source—is
the unit that is enabled, scheduled, checkpointed, and risk-controlled.

Each stream configuration retains a stable source/stream ID, intake method,
enabled state, credential reference, priority, local ai-gateway processing
profile, and successful-import checkpoint evidence. Credentials remain in the
owning secret system, never in an import record. Priority affects review and
ambient ordering, never retention or search inclusion.

The configuration is deliberately two-level: source-wide settings describe the
account or export family, while stream settings express a separately consented
collection action. Changing the X likes stream must not also enable the X
bookmarks API, for example. A stream may have more than one **configured intake
method** only when the methods produce the same declared activity and use the
same identity/deduplication rule; exactly one method is active at a time. This
permits a tier-0 archive import now and a future tier-1 official API sync
without merging their consent or risk controls.

| Source | Stream | Supported intake methods | Default | Priority | Deliberate exclusion |
|---|---|---|---|---|---|
| Chrome | bookmarks | Operator-supplied Netscape HTML export | Enabled when an export is selected | primary | Browser history, passwords, extensions, and sync metadata |
| Reddit | saved posts and comments | Operator-supplied GDPR ZIP; future date-bounded export request after terms review | Archive import available; network request disabled | candidate | Votes, chats, profile/account data, ads, subscriptions, and inferred interests |
| X | authored posts, reposts, quote-posts | Operator-supplied archive | Archive import available | primary | DMs, contacts, ads, profile/security/account data, and media bytes |
| X | bookmarks | Official authenticated API; future archive format if X adds one | Disabled until explicit OAuth authorization | primary | No inferred bookmark stream from an archive that lacks bookmark records |
| X | likes | Operator-supplied archive | Disabled until operator enables this candidate stream | candidate | A like is not treated as an endorsement, category, or ambient-worthy item |

For every enabled stream, the operator can configure: method, schedule or
manual-only mode, enabled state, priority, local or explicitly approved hosted
inference profile, credential reference (when required), and checkpoint policy.
The scenario records the selected method and successful import evidence with a
run, but never stores credential values in its database or UI.

For X, authored posts, reposts, and quote-posts are `primary`; bookmarks are
`primary` and take precedence where both streams carry the same post; likes are
`candidate`. A candidate is durable capture evidence, not a claim of relevance,
quality, or safe-for-ambient status.

### Blocked dependencies

| Dependency | Blocked Requirement | Status |
|---|---|---|
| `video-downloader` transcript-only request | `SIG-P1-005` | **The capability does not exist today.** This scenario assumes it will, and must never work around its absence by downloading media and transcribing locally — that would violate D-018 and duplicate a whole problem domain. If the contract does not materialize, `SIG-P1-005` stays blocked rather than being satisfied another way. |

### Federated retrieval registration

Registration with `search-hub` is declarative. The scenario ships
`.vrooli/search.json`; the router holds no corpus content and no vectors, and
adding this provider is a registry row rather than a router change (D-023).

The descriptor must declare, per the shape every existing provider uses:

| Field | Value for this scenario | Notes |
|---|---|---|
| `provider_id` | `signal-inbox.signals` | Scenario-qualified corpus name. |
| `provider_group` | `signal-inbox` | — |
| `bucket` | `BUCKET_KNOW` | Buckets are named for what the *consumer* does, not for the owning scenario. Saved external material is reference knowledge. A bucket named `bookmarks` would be a scenario-shaped bucket and is explicitly disallowed by the registry contract. |
| `type` | `signal` | — |
| `scope` | `SCOPE_PROJECT` | — |
| `class` | `local_index` | Corpus and vectors live here. |
| `endpoint` / `status_endpoint` | This scenario's `SearchService/Search` and `/Status` | Connect-RPC paths under `vrooli.signal_inbox.v1.search`. |
| `result_mapping` | id, title, score, snippet, path | `score_scale` must match the engine actually used, or the router misranks results against other providers. |
| `tuning` | `engine: dense`, `embed_model: nomic-embed-text`, `embed_task_prefix: true` | The task prefix is fleet-wide; indexing uses the document prefix and querying uses the query prefix. |
| `scoring` | `recall_at`, `recall_target`, `mrr_at`, `deep_k` | — |
| `tests` | Golden corpus | Certification requires at least one **reviewed** positive and at least one junk negative (`expect_no_strong_hit`). A generated-only corpus cannot certify. |

Two consequences worth stating because they surprise people:

- Owning a descriptor makes this scenario **search-applicable** to fleet scan and
  to the test-genie `search` phase. It will be validated, not skipped.
- The descriptor should be authored when the search endpoint exists. Declaring
  one earlier means inventing tuning constants and eval cases against a corpus
  that does not exist, which produces a suite that measures fixtures.

### Consumption by other scenarios

This scenario is a dependency *of* others as much as it depends on them. The
structured query contract (`SIG-P0-009`) is a product surface, and category names
are part of it. A consuming scenario reads its signals by category and disposition
and resolves content through that contract; it never reads this scenario's tables
directly and never receives a copied signal body in an event (D-025).

| Consumer | Reads | Status |
|---|---|---|
| `vision-walk-prep` (director-swarm member) | Ambient view, alpha categories | First consumer; the reason this scenario is built now. |
| Any future meal-planning scenario | `meals` category | Illustrative of the general case — the substrate privileges no category. |
| `search-hub` federated query | Whole corpus | Via the provider descriptor. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Reddit API | planned (P1) | First tier-1 adapter; saved-posts endpoint is documented and permitted. | OAuth under operator-supplied credentials for a registered application. Ships disabled. |
| X bookmarks API | planned (tier 1) | The measured archive omits bookmarks, while X documents an official authenticated bookmarks endpoint and folder endpoints. | OAuth 2.0 authorization-code-with-PKCE with `tweet.read`, `users.read`, and `bookmark.read`; explicit operator consent; disabled by default. Current pricing/scopes/rate limits must be rechecked immediately before enablement. |
| X archive streams | planned (tier 0) | The operator-supplied archive is measured locally and includes authored posts and likes. | Shape-validated ZIP parser; authored posts/reposts/quote-posts are `primary`, likes are `candidate`; no platform request. |
| Other platforms | not-planned as an API | Access terms, capability, and costs vary. | Operator-supplied archive import (tier 0) and, only if necessary, tier-2 session replay (`SIG-P2-001`). |

**Platform terms are not this scenario's to assume.** Any claim about a
platform's API scopes, pricing tier, or rate limits must be verified against
current documentation before an adapter is enabled — these change independently
of this repository, and a stale assumption encoded in an adapter is exactly how
an account gets flagged.

### Risk tiers

Every adapter declares a tier; the runner enforces it (D-015, D-016).

| Tier | Mechanism | Account Risk | Default | Enforced Behavior |
|---|---|---|---|---|
| 0 | Manual entry; operator-supplied export | None | **Enabled** | No network request to any platform. |
| 1 | Official API, operator credentials | Low | Disabled | Rate envelope from the descriptor; auto-disable on anomaly. |
| 2 | Authenticated session replay via BAS | **Real and unrecoverable** | Disabled, per-adapter explicit enable | Human-paced envelope; auto-disable on anomaly; operator sign-off before first enable. |

Tier is enforced per stream. Enabling X bookmarks through the official API does
not enable X likes, an archive import, or any session-replay stream.

Auto-disable on anomaly is the mechanism that actually protects the operator:
any rate-limit, forbidden, or challenge response disables the adapter and raises
an alert rather than retrying, and the disabled state survives restart. This is
deliberately more conservative than a backoff policy — stalled capture is an
acceptable cost, a lost account is not.

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| ai-gateway | Request error or timeout | Capture succeeds; classification records `uncategorized` with a reason and queues re-classification. Never blocks a write. | `SIG-P0-005` integration |
| image-tools | Unavailable or extraction error | Image signal marked `needs-attention`; never stored with an empty body. | `SIG-P0-003` |
| video-downloader | Unavailable, or contract absent | Video signal marked `needs-attention`. | `SIG-P1-005` |
| search-hub | Registration or query failure | Local search unaffected; federated reach degrades. | `SIG-P0-011` integration |
| prompt-manager | Knowledge-write failure | Routing fails; signal stays `triaged` and no outcome link is recorded, so the routing is retryable and not silently half-done. | `SIG-P1-002` |
| Any tier-1/2 adapter | 429, 403, or challenge response | Adapter disables itself and alerts. **No retry.** | `SIG-P0-014` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`../reference/conversation-extraction.md`](../reference/conversation-extraction.md) — the one extraction technique owned here
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-015 to D-018 and D-023 record the reasoning above
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
