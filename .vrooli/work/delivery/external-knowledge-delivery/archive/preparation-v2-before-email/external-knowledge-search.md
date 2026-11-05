# External Knowledge Search

Status: owner-approved architecture direction; delivery prepared, not implemented or enabled.
Owner request: 2026-10-14. Finite team: `external-knowledge-delivery`.
Goal home: `docs/internal/external-knowledge-delivery/`.

Operator amendment FB-004 (2026-10-14): after future activation, prioritize a repeatable, commercial-compatible Integration Hub connection capability with professional CLI and UI. The team builds that capability before asking the operator to configure/consent to a live account; missing consent gates dependent live work only. No one-off pre-enablement bootstrap is required. Research sub-agents are authorized now; native delivery remains disabled.

## Purpose and stakes

Make selected external knowledge searchable through Vrooli's normal agent interfaces. Start with selected Google Drive folders, including Biosphere if the operator selects its exact folder ID. Biosphere is the entry point and operating knowledge library for the operator's always-on ChatGPT agent. Other project folders may be selected independently. Local agents without ChatGPT plugins and authorized remote agents should retrieve the same attributable evidence.

Drive is the first delivery priority. The same finite effort carries shared foundations and later Google Calendar and Gmail delivery. Email product and UX specifications remain pending a separate operator-led design conversation. Product implementation and background synchronization are not authorized by commissioning this disabled team.

## Source precedence and implementation posture

The operator's accepted direction in this document governs this cross-scenario extension. Each scenario's canonical PRD, requirements, domain map and experience records still own that scenario's product contract. Reconcile those through their governed authoring routes before admitting its implementation slice. This document does not silently rewrite an existing PRD or certify a runtime.

The dated observations and online references are in `docs/internal/external-knowledge-delivery/RESEARCH.md`. Recheck code and native state at admission. Record later operator decisions verbatim in the goal home's FEEDBACK.md. Candidates, unverified capabilities, selected designs and delivered evidence remain separate.

## Ownership

| Owner | Responsibility |
|---|---|
| integration-hub | Provider-neutral account connections, Google OAuth authorization/refresh/revocation, granted capabilities, credential-authority references, and scenario bindings. Own the typed connection API and repeatable CLI/UI onboarding, repair and management flows. Add a bounded authenticated-operation seam; consumers receive permitted operations rather than secret values. |
| Google connection drivers | Google-specific list/read/change/notification request shapes and errors. Advertise supported capabilities. Preserve service-specific cursor semantics. These are connection infrastructure, not a corpus owner. |
| document-manager | Selected Drive collections, source membership and revisions, ingestion jobs/checkpoints, format-aware derivations/chunks, sensitivity, enrichment, citations, indexes and authorized retrieval. Extend its contract with a remote-backed source mode. |
| personal-planner | Calendar selection, external event projections, recurrence/timezone semantics, availability and scheduling integration, calendar sync state and search. Coordinate with its existing delivery owner; do not create a competing schedule or modify that team's control state. |
| future email scenario | Mail accounts, messages, threads, labels, MIME parts, synchronization, classifications, search and the approved email experience. Name and broader UX remain proposals until selected. |
| packages/ai-go/search | Reusable retrieval/reconciliation mechanics. Keep provider-specific selection and domain policy out of the package. Preserve existing store adapters; Qdrant is not mandatory for document-manager. |
| search-hub | Generic authenticated federation, declarative provider registration, routing/ranking, typed provenance/metadata, status and direct/federated quality evidence. No Google client, credentials, authoritative content or vector store in the router. |
| meta-optimization-manager | Read owner-backed readiness and condition evidence. Do not centralize external data or count descriptor existence as delivered search. External personal-data readiness must not inflate the architectural Answer denominator without an explicit appropriate denominator extension. |

Shared schema/state remains per-domain and engine-independent behind repositories. Dependency changes use Scenario Dependency Analyzer. Extraction/OCR live behind resource or document-manager handler seams; inference uses existing AI Gateway roles/profiles with explicit residency policy.

## Integration Hub product and commercial boundary

The initial Hub pilot is not the desired mature product. Delivery includes verified principal integration, connection ownership and object policy, provider-neutral capability/status contracts, secure authorization sessions, refresh/reconnect/revoke behavior, deployment-aware client/callback profiles, and accessible responsive connection management. CLI and UI consume the same typed owner operations. Build and qualify their actual journeys, not only HTTP endpoints or a browser link.

Keep principal authentication, scenario capability, connection/object authorization and commercial entitlement separate under their existing owners. LPBS or the declared commerce owner supplies entitlements; Hub does not create a billing database or identity realm. Features may be commercially gated, but expiration does not turn another customer into the credential owner, silently revoke Google grants, or authorize agents to export secret values. Recovery, disconnect/revoke and required user data-access/deletion paths must have explicit policy rather than being accidentally hidden by a paid-feature toggle.

Support declared personal-local, local-multi-user and hosted/shared-provider topology through existing identity/deployment contracts. Initial rollout choice and operator-owned versus managed OAuth client are open business/deployment decisions. Design explicit compatible seams now; do not implement speculative enterprise administration or assume a developer-owned client secret can be safely shipped in a desktop bundle. A shared commercial Google client has project-wide provider quotas/revocation and verification requirements that differ from a personal setup.

Detailed current gaps, proposed typed operations, CLI/UI journeys, scopes/client/deployment choices, monetization integration and qualification gates are in `docs/internal/external-knowledge-delivery/INTEGRATION-HUB-GOOGLE-DESIGN.md`. Its recommendations are research inputs; business choices and governing visuals remain unselected until owner disposition.

## Connection and authorization model

ChatGPT plugin access is interactive session capability. It does not transfer refresh tokens, background jobs, change feeds or agent permissions to Vrooli. Use independent Vrooli Google authorization with offline refresh. Request scopes when each approved service is introduced; use incremental authorization for supported web clients and explicit renewed grants for desktop clients; initial Drive access is read-only. Do not request Gmail or Calendar write privileges for indexing.

A connection permission and a retrieval grant are separate: Google decides what the account connection can read; Vrooli decides which authenticated agent may retrieve which collection. Folder inclusion is not automatically a Google OAuth restriction. Validate narrow-scope behavior for both existing descendants and newly added descendants. If broader Google read access is needed, explain it and enforce Vrooli's folder allowlist before content reads.

The authenticated caller and delegated collection grants must reach the producing provider. A caller-supplied subject, `--scope`, static descriptor header or tuning control token is not authorization. Apply collection/workspace/privacy authorization before candidate scoring, snippet construction, reranking, context fetch and result emission. Enforce the same isolation in caches, metrics, logs and evals. Direct and federated paths must reject unauthorized access, including guessed IDs and cross-account lookups. No principal substitution after a real native refusal.

Withdraw removed, out-of-scope or access-revoked content from all serving indexes, caches and retained generations. An old index generation must not restore authorization. Maintain live visibility gates against the provider catalog. Disconnection/revocation has an explicit retention/purge policy and must revoke retrieval independently of eventual physical cleanup. Bound authorization freshness and disclose the offline behavior; do not claim instantaneous detection of remote revocation.

## Drive selection, identity and retention

Select root folder IDs explicitly. Configure descendant traversal, excluded subtrees, supported formats, maximum file/extraction sizes, attachment/media behavior and shortcut policy. Default to no traversal through shortcuts whose targets lie outside selected roots. Folder rename does not change identity. A moved folder can add/remove an entire subtree; reconcile descendants, not only the folder record.

Use account/connection plus remote file ID as source identity. Content hash identifies a version, not the remote object. Keep duplicate-content sources' membership and authorization separate even if extraction storage is deduplicated. Store source URL, parent membership, observed provider version/time, extraction version, content hash, policy version and indexed recipe/model version. Google remains authoritative for current source content. A citation binds the version actually read; it must not imply that mutable Drive content has permanent page or cell coordinates across edits.

Default remote-backed retention: metadata, normalized extracted text/structure, citable passages and derived lexical/semantic artifacts; no permanent original-file archive. Original bytes may be fetched transiently for parsing. An explicitly enabled archival mode may retain originals under document-manager's existing archival rules. Do not silently route remote-backed collections through permanent-byte intake.

Retaining only vectors is an optional, separately designed tradeoff requiring authorized live passage fetch and weaker offline/lexical behavior. Vector payload `body`, summaries, question views, FTS rows, caches and debug outputs are content copies too. Report them in retention policy. Unprocessable remote files keep a metadata-only visible state; do not retain raw bytes merely because the archival intake path does.

## Synchronization and freshness

1. Obtain a source checkpoint before initial enumeration. Traverse selected roots and store the normalized catalog in bounded pages.
2. Capture changes after that checkpoint to cover edits during enumeration. Enforce membership policy before reading content.
3. Commit source progress only after catalog mutations and pending work are durably recorded. Use an outbox/queue or equivalent transactional replay boundary.
4. Parse and index queued versions independently. Advance an indexing checkpoint only after its serving publication succeeds. Do not mark a source version searchable merely because it was downloaded.
5. Run incremental sync on a configurable schedule, initially targeting about five minutes as a proposed default. Add push wakeups when justified; keep periodic reconciliation as recovery.
6. On invalid cursors, build a scoped replacement snapshot, reconcile removals and publish after success. On transient failure, use bounded backoff/jitter and retain pending work.

Provider-specific facts: Drive uses its changes feed and folder-tree reconciliation; Gmail uses history IDs and mailbox message reads; Calendar uses per-calendar sync tokens. Tokens are opaque to consumers and not interchangeable. Notifications are wakeup hints; the change stream is the source of changed facts. Persist watch expiry and renewal state if push is enabled.

Expose last source sync, permission verification age, last indexed version, queue depth, source-to-lexical lag, source-to-semantic lag, failed/unsupported file count, and per-leg degradation. Embedding outages keep captured work durable and authorized lexical search available. Outages do not create free time in personal-planner. Distinguish source unavailable, reauthorization required, empty corpus, extraction failure and stale index.

## Document processing and chunking

Extraction precedes chunking. Google-native Docs/Sheets/Slides need tested export or structured-read adapters. Do not assume their URLs contain file bytes. Reuse document-manager's handler registry and resources for binary formats; format capability claims need actual fixtures and runtime evidence. Do not run spreadsheet macros or active embedded content.

| Source shape | Proposed index units and context | Required evidence/locator |
|---|---|---|
| Docs, Word, Markdown, flowing text | Heading-aware section chunks with bounded overlap, title and heading path. Reuse the shared Markdown chunker where normalized structure fits. Preserve lists and code blocks. | Source version plus logical section/passage anchors. |
| Sheets, Excel, CSV/TSV | Sheet/table overview plus bounded row groups with repeated column headers, units, labels and sheet context. Preserve typed values, formulas and cached displayed values separately. Do not flatten an entire workbook or split a row arbitrarily. | Workbook version, sheet identity and cell/range anchors. Exact numeric/aggregation answers use structured cells, not vector similarity or generated summaries. |
| Text-native PDFs | Layout-aware sections and positioned blocks; preserve page maps, reading order and tables. | PDF version and page/bounding-box anchors. |
| Scanned PDFs and images with text | OCR blocks with confidence and geometry. Index recovered text with source context. | Version, page/image region, OCR engine/version/confidence. |
| Images, charts and diagrams | Searchable known metadata/OCR; optional governed vision description when supported and policy permits. Semantic image vectors are a separate measured extension, not a claim made by text embedding. | Distinguish observed text, generated description and direct visual evidence. Mark visual interpretation unavailable honestly. |
| Slides | Slide-scoped units; separate speaker notes; bounded table/image units where supported. | Presentation version, slide identity and body/notes/region locator. |
| Email later | Message body units with bounded context, headers as facets/context, quoted-history deduplication and separate attachment units. | Account/message/thread/part identities and exact source-backed passages. |

The shared Markdown defaults are a starting point, not a universal strategy. Its token estimate and oversized atomic-block behavior require a real embedding-input bound for large tables/code blocks. Keep extraction, chunking and embedding-composition versions independent so one can change without re-downloading unchanged originals. Sensitivity resolves before any remote inference; local-first fallback alone does not prove local-only residency.

The dated fidelity investigation found unqualified production seams behind several declarations: current doc-ocr emits `image page`/`scanned page` labels rather than raster transcription; document-manager's runner and resource command/response contracts disagree; derivation is not wired into intake-to-search; mounted enrichment has no wired Gateway. Do not index these labels, printable binary fragments or an empty decoded units array as successfully extracted source evidence. Qualify owner adapters, real OCR on representative raster fixtures, optional vision and complete publication before claiming support. A format-registry declaration or native/WASI output equality is not independent evidence of correct text, cells or locators.

Use structured Google-native reads as the preferred fidelity lane: all Docs tabs and structural coordinates; Sheets entered/effective/displayed values with locale, timezone and range context; Slides body, notes and object identities. Export remains a tested fallback/visual derivative with its explicit losses. Retain typed numeric evidence alongside semantic hints; exact calculations use authorized coherent source values, never a generated question/summary or inferred chart pixels. Remote citations bind captured versions and retained evidence; old live coordinates are not permanent original geometry. Detailed limits, owner changes and qualification fixtures are in [the document fidelity investigation](../internal/external-knowledge-delivery/DOCUMENT-FIDELITY-DESIGN.md).

## Anticipated questions and retrieval views

The operator explicitly requires evaluation of anticipated-question enrichment. Keep three concepts separate: question families define expected demand; eval queries test retrieval; embedding views are derived index representations. Defining eval questions does not by itself create question vectors. Existing CLI Health command composition DOES embed authored measure intent/questions; Search Hub DOES index multiple provider answer-space/intent/example representations for routing. Reuse those concrete ideas without confusing provider routing with source-passage retrieval. A general generated-question-per-document-unit pipeline still needs extension and evidence.

For each stable source/version, start with raw citable units plus title/description/heading context. Add bounded document overview, section summaries and anticipated-question/paraphrase views only where they are grounded in that source. A synthetic question is a retrieval hint, never a cited answer or an independent source. Every view maps back to real units and carries recipe/model/prompt/source versions. All views inherit source authorization, sensitivity and withdrawal.

Suggested demand families: find a document; explain a procedure; locate a decision and rationale; identify prerequisites/owner; compare versions; find examples/evidence; locate a spreadsheet metric/range; identify a diagram or chart. Use deterministic metadata templates where enough; generated views are optional, budgeted enrichment. Retrieval must work without them.

Fuse exact lexical and semantic candidates, deduplicate views into source-backed passages, and preserve rank evidence. Measure raw/contextual/question-enriched recipes on held-out paraphrases, exact-match queries, hard negatives and gibberish. Hold evaluation queries apart from generated training/index questions; otherwise apparent improvement can be answer leakage. Promote a recipe only with measured quality/cost/freshness evidence. Question families and the number of views are bounded per content type, not unrestricted multiplicative re-embedding.

Recipe/version changes must invalidate source-level reconciliation skips, not only metadata written after composition. The current ordinary reconciler's source ContentHash/ChunkTotal shortcut does not by itself prove freshness after a new view/prompt recipe; streaming model/chunk-policy checks also need an explicit compatible view recipe. Start with deterministic contextual vectors and source-backed lexical units; generated views are off until measured. Experimental counts, model choice and thresholds in [the question-aware investigation](../internal/external-knowledge-delivery/QUESTION-AWARE-RETRIEVAL-DESIGN.md) remain proposed defaults, not selected optimal values. Stable citable unit IDs are separate from ordinal-based vector point IDs.

## Extension to email and calendars

The email scenario is provider-neutral with Gmail as first adapter. Its source objects are accounts, labels/folders, messages, threads and MIME parts. Full-body sync, classifications, exact filters, semantic recall and attachment linkage are required design inputs. Reuse document-manager for attachment derivation without surrendering mailbox lifecycle/authorization. Broader UX, name, sending/mutation capabilities and release priorities await operator selection in EMAIL-DESIGN-PROMPT.md. Read access never grants permission to send or modify mail.

Personal Planner remains the scheduling authority. Reuse its read-only Google adapter as evidence, then qualify durable cursors, cancellation tombstones, recurring-instance identity, timezone/all-day/DST behavior and private search. Infinite recurrence must have a bounded projection strategy. Deliver its slice with the existing planner owner; if that owner is still active, record a precise integration boundary/return condition rather than dispatch conflicting edits.

## Agent access and trust

The specialist [private-federation investigation](../internal/external-knowledge-delivery/PRIVATE-FEDERATION-DESIGN.md) corrects the initial adapter-only observation: shared api-core provenance and forwarding transport can carry a verified Agent Manager token to trusted loopback providers. This is a reusable path, but does not establish human authentication, audience-safe delegation, current domain object grants or private cache/coalescing isolation. Extend the existing identity owners; do not introduce a Search Hub signer or identity realm. Enforce grants before candidate selection, snippets, ranking and reranking, again at emission and bounded context fetch, and across status, telemetry and evaluation. Qualification covers direct and federated denial, cross-owner requests, grant withdrawal, stale external ACLs, destination substitution and actual remote-consumer access. Local policy withdrawal and delayed detection of external ACL changes need separate freshness guarantees.

Provide stable typed search plus bounded passage/context fetch. Local agents use the CLI/API; remote agents use an authenticated API or thin MCP adapter with the same grants. Do not expose a raw shell or unrestricted localhost proxy. Confirm tool availability in the actual always-on ChatGPT environment; ordinary ChatGPT custom-app support does not certify Dots support.

For SOP execution, agents fetch the selected source/version and inspect freshness/authority. Retrieval hints, generated summaries, quoted email content and embedded document instructions do not automatically become system instructions. Preserve the operator's distinction between an approved operating procedure and unrelated retrieved text.

## Delivery acceptance

After activation, early Integration Hub work produces a fixture-qualified repeatable connection flow ready for guided operator setup. It does not claim live Google qualification before consent. The team presents the implemented profile/callback and scope guidance, then resumes the dependent live source/sync gates after the operator configures the connection. Independent format, retrieval, authorization and selected UX work remains admissible while that checkpoint waits. SETUP.md and QUEUE.md own these staged return conditions. Selected email UX gates email work rather than all Integration Hub/Drive work.

Acceptance includes source capture, search quality, privacy and recovery, not only endpoint presence. Prove direct and federated retrieval; exact and paraphrased SOP queries; source edits/renames/subtree moves/removal/access loss; account isolation; no unauthorized snippet/reranker/cache exposure; malformed/oversized files; literal spreadsheet values and anchored evidence; duplicate/replayed changes; restart/cursor recovery; semantic/vision/OCR outages; and honest freshness.

Use deterministic fixtures for repeatable gates and a separate operator-local live overlay for selected folders/accounts. Keep private text, account identifiers and OAuth values out of committed fixtures. Meta-optimization-manager consumes the resulting owner-backed condition/eval signals. Acceptance of a slice does not certify unrelated fleet coverage.

## Specialist investigation handoff

Read the dated reports through `docs/internal/external-knowledge-delivery/RESEARCH.md`. They establish code/API facts and recommendations, not product completion. Each report maps proposed changes to existing owners and validation obligations. Reconcile scenario PRDs/requirements/experience through governed authoring routes before implementation; this preparation does not silently change those canonical product contracts. Preserve code-level gaps and pending choices beside every capability claim.

## Linked canonical sources

- `scenarios/search-hub/docs/concepts/ARCHITECTURE.md`
- `scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`
- `scenarios/integration-hub/README.md`
- `docs/configuration/integrations/{connectors,connections,external-auth}.md`
- `scenarios/document-manager/docs/concepts/{DOMAINS,DATA,INTEGRATIONS}.md`
- `scenarios/document-manager/docs/reference/{format-matrix,anchor-uri}.md`
- `scenarios/personal-planner/docs/concepts/INTEGRATIONS.md`
- `packages/ai-go/search/{README.md,contracts.go,markdown.go}`
- `docs/concepts/{IDENTITY-AND-AUTHENTICATION,AUTHORIZATION}.md`
