The four specialist reports add the constraints below. All are dated **2026-10-14** and remain **research/proposed contracts, without email implementation or product qualification**.

A material handoff update appeared during this read: [PLATFORM-AMENDMENT.md](/home/matthalloran8/Vrooli/.vrooli/work/delivery/external-knowledge-delivery/PLATFORM-AMENDMENT.md), under **“Successful native approval and application — 2026-10-14,”** records successful preparation-v2 application at **19:27:58.683Z**. The retained [native receipt](/home/matthalloran8/Vrooli/.vrooli/work/delivery/external-knowledge-delivery/native-amendment-receipt.json) and [leader binding](/home/matthalloran8/Vrooli/.vrooli/work/delivery/external-knowledge-delivery/leader-binding.json) agree: team/heartbeat disabled, unscheduled, idle, and undispatched. This supersedes the earlier pending checkpoint. `sources.json` still contains the **19:14Z** pending description and v1 native revision; that manifest is stale relative to the later receipt.

**Additional connection and account UI constraints**, from [INTEGRATION-HUB-GOOGLE-DESIGN.md](/home/matthalloran8/Vrooli/.vrooli/work/delivery/external-knowledge-delivery/INTEGRATION-HUB-GOOGLE-DESIGN.md):

- **“Records and ownership” / “Proposed repeatable browser and CLI experience” — proposed contract/UX:** Connection state and authorization-attempt state are separate. A failed or cancelled renewal can coexist with a usable prior grant. Do not turn every reconnect failure into “account disconnected.”
- **“Native Google authorization, refresh, and revocation” — recommended lifecycle:** Same-account renewal preserves bindings; choosing a different Google account requires an explicit replace/new-account flow and binding review. Distinguish permission problems, disabled APIs, organizational restrictions, client setup failure, provider outage and credential-authority unavailability. The latter should not ask users to recreate Google consent.
- **“Native Google authorization, refresh, and revocation” — dated provider fact plus recommended UI:** Pause/unbind, local deletion and Google revocation have different effects. Revocation can affect known Drive/Calendar/Gmail siblings in the same Google project and possibly other installations. Show impact before confirmation; distinguish remote revocation **confirmed, pending and unknown**.
- **“Stable callback and client strategies” — topology candidates:** External-browser authorization is required; no embedded consent screen or manual code/token paste. Desktop service expansion differs from web incremental authorization. Phone email UX does not establish a qualified mobile OAuth strategy.
- **“Tenant, account, and commercial isolation” — recommended policy:** Keep provider health, mailbox freshness, identity availability and managed-service entitlement separate. Paid status cannot grant mailbox access. BYO account inspection, disconnect, deletion/export and recovery must not accidentally disappear behind subscription gates.
- **“Proposed repeatable browser and CLI experience” — proposed UX:** A successful callback does not itself start background collection. Return to verified account/capability confirmation and explicit mailbox selection/binding.

**Additional attachment and evidence UI constraints**, from [DOCUMENT-FIDELITY-DESIGN.md](/home/matthalloran8/Vrooli/.vrooli/work/delivery/external-knowledge-delivery/DOCUMENT-FIDELITY-DESIGN.md):

- **“Extraction-to-unit contract and publication” / “Limits, streaming and honest degradation” — proposed evidence/state contract:** Track coverage and origin per attachment, page or range. “Partial” alone is insufficient: distinguish too large, policy blocked, unsupported variant, handler failure, OCR unavailable/low confidence and visual interpretation unavailable. Empty content differs from an adapter that omitted content.
- **“Type-specific chunks and locators” / “Mutable remote versions and retention” — proposed citation contract:** Historical normalized passages may remain available while their original layout/image is unavailable. Show those separately. A current remote link is navigation, not proof that the captured historical citation still aligns.
- **“Exact retrieval and semantic hints” — recommended numeric behavior:** Spreadsheet previews/answers must distinguish entered formula, effective value, formatted display and uploaded cached value. Preserve locale, units, blank/error states and capture time; never manufacture missing results or infer exact numbers from chart pixels.
- **“Google-native reads, exports and scopes” — dated API observations, adapters unqualified:** Native Docs tabs, Sheets ranges and Slides notes need explicit coverage. Linked documents/charts do not expand permission. Avoid promising full attachment comprehension from a filename or format declaration.
- **“Limits, streaming and honest degradation” — proposed budgets:** Large attachments need visible covered pages/ranges and resumable continuation, rather than silent truncation. The report’s 32 MiB, 60-second and 20-page/region values are hypotheses, not selected email limits.
- **“Mutable remote versions and retention” — open policy:** Current capture plus one prior capture for 30 days is a **Drive proposal**, not an email retention decision. Do not inherit it silently for mail or downloaded attachments.

**Additional search and assistance constraints**, from [QUESTION-AWARE-RETRIEVAL-DESIGN.md](/home/matthalloran8/Vrooli/.vrooli/work/delivery/external-knowledge-delivery/QUESTION-AWARE-RETRIEVAL-DESIGN.md):

- **“Retrieval, deduplication, fusion and returned context” — proposed retrieval contract:** Several question/summary vectors must collapse into one canonical evidence result; they must not inflate result counts or monopolize the first page. Show real message passages, with bounded adjacent/thread context, rather than whole-thread expansion by default.
- **“Generation and model-input bounds” — recommended assistance behavior:** Summaries must preserve negation, uncertainty, exceptions and commitment status. Source instructions cannot trigger tools or account actions. A second model’s agreement is not evidence.
- **“Chunking interaction and service extensions” — email-specific proposal:** Thread overviews use the intersection of supporting messages’ grants and identify precise messages. A mailed proposal is not an accepted task. Do not create cross-provider synthetic summaries by default.
- **“Evidence units and view families” / “Held-out evaluation and recipe promotion” — experiment candidates:** Generated questions are off until measured. Two questions per eligible unit is an experimental ceiling, not a user-facing quota or required feature. Normal extraction/enrichment need not create a per-item human approval queue.
- **“Cost, residency and operational policy” — recommended policy surface:** Query embedding, source embedding, generation, reranking and OCR/vision can have different permitted processing locations. A single “AI is local” switch cannot describe them accurately.

**Additional permission and privacy constraints**, from [PRIVATE-FEDERATION-DESIGN.md](/home/matthalloran8/Vrooli/.vrooli/work/delivery/external-knowledge-delivery/PRIVATE-FEDERATION-DESIGN.md):

- **“Provider visibility, passages and races” — proposed authorization contract:** Opening a citation or historical message rechecks current access; a prior hit is not a capability. Expired pagination should offer a safe rerun. Denied states must not reveal hidden account labels, source titles or existence.
- **“Privacy across router surfaces” — recommended privacy policy:** Empty private results must not automatically fall back to web search or remote models. Provider lists, counts, explain/debug views and freshness status also require visibility filtering.
- **“Privacy across router surfaces” — proposed telemetry defaults:** No persistent raw private query or unkeyed query hash by default. Debug/export/evaluation surfaces must not quietly retain mailbox titles, excerpts or queries.
- **“Bounds and remote consumer” — proposed integration gate:** Agent access is only search, bounded context and authorized status. The real consumer must prove permitted retrieval, denied cross-collection access and refusal after revocation before the dossier calls that integration available.

The documented handoff supports this **reviewable import sequence**, without activation:

1. Deliver the complete specification, atlas and governing originals at exact accessible versions. Include asset identity, role, format, bytes/SHA-256, dimensions, screen/state/theme/viewport mapping and requirement/acceptance links. Mark governing, illustrative and rejected assets distinctly.
2. Import through the existing goal home; append actual owner statements verbatim to FEEDBACK, while recording delegated designer decisions separately. Link the specification/assets from the governing architecture and C05 in QUEUE. Reconcile the latest full-client/Communications direction, rather than leaving the obsolete scope question as a blocker.
3. Refresh `sources.json` and selection/version mappings; preserve prior preparation snapshots and receipts. Do not rewrite scenario PRDs by hand: the delivery owner later reconciles canonical product/requirements/experience records through governed routes.
4. Prepare a **new** guarded amendment with the current expected native revision:
   `preparation-v2:sha256:ad31c53a71676fae6e4af743f3f1bb29d5a5c7f5e5d070b895ecf19c9eeb264d`.
   Preserve effort, leader, profile, schedule and liveness fields. Add the new target/source references and a retained evidence reference.
5. The documented owner operation is:
   `prompt-manager team heartbeat-amend-effort external-knowledge-delivery external-knowledge-orchestrator --request-file <new-request.json> --json`.
   This is a future owner-authorized action. **Do not replay `guarded-amendment-request.json`: it expects v1 and was already applied.**
6. Under current verified native authority, retain amendment/readback evidence, source integrity, contract validation, goal-home lint and a fresh-reader receipt. Confirm disabled, unscheduled and idle state. None of those operations constitutes activation.

Sources: **“Required design handoff”** in EMAIL-DESIGN-PROMPT; **“Later explicit activation and source amendments”** in SETUP; **“Supported draft amendment now implemented; native proof pending”** plus its supersession in WORKAROUNDS; and PLATFORM-AMENDMENT’s guarded-operation contract.

No one-command Drive-spec/asset importer is documented in the inspected sources. Proposed destination filenames and import mechanics must therefore be distinguished from the existing documented route. If local import has not happened, return the exact Drive links/versions and say local reconciliation remains pending.

No API calls, mutations or amendments were performed in this pass. The filesystem inspection completed without an access blocker.
