# Marketing full UX 0.3 — initial contract challenge

Reviewed 2026-10-04, approximately 04:12 UTC. This is a read-only design/source challenge. No scenario, account, runtime, canonical Drive document, or publishing action was changed. The only write is this review.

## Result

Nine material gaps remain between the proposed connected experience and a sufficiently determinate domain contract. They do not invalidate the integrated Content Desk direction. They require specific semantic decisions and non-operating traces before the new controls can be considered designed. Simply labeling all APIs “proposed” does not resolve what their successful result means.

The most consequential are C2 (editorial acceptance versus native operator approval), C4 (operation replay with changed payload), C5 (partial multi-item releases), and C8 (concurrent cross-identity asset use). C1 and C6 are prerequisites for the proposed family and planning identities. C3, C7 and C9 prevent credible-looking review, cost and learning surfaces from losing their actual basis.

## Evidence and limits

Inspected inputs:

- `i16-design-v02/source/dossier.md`, SHA-256 `d79afc305d6c6741afd37fdc43c3543b36a65500a5d460c3989543d73f480d0b`.
- `i16-design-v03/source/redesign-inserts.json`, SHA-256 `1e3e7fee490b5d74f0d54603185a4c7eba42f36ca1ab35b090617c01c2ed4a7e`. Locations below name its chapter key and subsection.
- `i16-design-v02/qa/source-review.md`, SHA-256 `c4c4412056fc481ffb596609e393e5ef01a97d3e7f4c076719437709095108a8`. Its final bounded closure of S1–S6 is retained. These findings test additional 0.3 semantics; they do not silently reopen that older review.
- Exact relevant source files under `i16-design-v02/source/local-files/`. All source-path locators below are relative to that directory.

The supplied 71-file snapshot was captured at 00:36:46 UTC on October 4. It establishes captured declarations and code, not current runtime behavior. Asset Studio ingress, Content Desk handler invalidation and the optional Prose execution route retain the earlier report's evidence status; I did not freshly inspect missing Asset Studio source, Content Desk handlers, or a Prose proto. Current-source statement below means “present in the captured source,” not “qualified in production.”

Verified foundations include native draft IDs; body-focused effective revision/review authority; human-operator-only approval in the Content Desk PRD; explicit campaign artifact slots; Channel Manager account/release ownership; the immediate-time typed release seam; Planner-owned accepted allocations; optional read-only agent suggestions; and Prose's append-only candidate/cost lineage. New family/project relations, whole-package approvals, typed timed release, complete source adapters and the expanded UI states remain proposed.

## C1 — Define family membership and scope before it becomes a second implicit identity

**Location:** inserts 2, “Relationships that stay intelligible as work grows”; original Chapter 2 project seam and A8.

**Verified:** `artifacts/artifacts.proto:24–52` identifies a draft, campaign, channel and format, with no family/project field. `campaigns/campaigns.proto:28–38` declares slots by channel and format. Content Desk PRD OT-P0-002 requires a declared campaign budget; OT-P2-002 treats per-channel derivation as optional breadth.

**Counterexample:** The same article supports a founder reflection and two app campaigns. One LinkedIn draft is deliberately in two family collections, but is one native draft and one budget-consuming artifact. A user then moves that draft to another project or severs one derivation. The new text does not decide whether family membership is exclusive, whether families may cross access scopes, whether campaign membership follows the family, or whether moving/severing affects the source article and siblings. The phrase “optional campaign or family” can also suggest mutually exclusive relationships that the subsequent prose does not require.

**Required design:** Choose relationship cardinalities and ownership, distinguishing a collection relation from a causal derivation edge. State which identity owns the working title, which reference consumes a campaign slot, whether one draft can participate in multiple campaigns, and whether cross-project derivation is allowed as a permitted reference. Define detach/archive/move behavior without cascaded mutation of descendants. These choices can remain proposed; storage tables need not be designed here.

**Retest:** Draw one source, two families, two campaigns and three variants with native IDs. Move one variant while a sibling has an active release; enumerate preserved links, blocked links, slot counts and unchanged historical scope. A one-off must still need neither family nor campaign.

## C2 — Map “Accept wording” to native authority instead of relying on the label

**Location:** inserts 2 and 8; original Chapter 8, “A package, not a green global status.”

**Verified:** `review/review.proto:7–34` records craft/policy verdicts against a draft and explicitly does not own truth checks. `artifacts/artifacts.proto:166–196` has `ApproveDraft(id, identity_id, lane)` and applicable current-body authority. Content Desk PRD OT-P0-006–009 requires verified claims, human operator approval and active post type. `claims/claims.proto:25–33` expressly separates evidence qualification from verification lifecycle.

**Counterexample:** A qualified reviewer accepts destination-unbound wording r2. Later a publisher chooses an account and authorizes release. Is the first decision an ordinary review verdict, a new editorial approval, or native `ApproveDraft`? Can that reviewer be an agent? If “supported” evidence is displayed, does it pass the native verification gate? A publisher cannot infer that a reviewer action satisfied the human operator gate merely because the UI shows “editorially accepted.” The original package definition includes account/audience, whereas the insert deliberately allows an unbound editorial package; their normative relationship must be explicit.

**Required design:** Add a small command/authority mapping for Adopt edit, Record craft verdict, Accept wording, Approve native draft and Authorize release. Identify exact actor class, stored decision, required gate, and state changed for each. Preserve human operator approval and claim verification unless a separately reviewable native change is proposed. State explicitly whether scoped editorial acceptance is a new record and that it cannot impersonate existing native approval. Define which older package wording the split supersedes.

**Retest:** Agent review passes; human reviewer accepts wording without publisher capability; account is then selected; one claim is supported but not verification-ready. Show the exact remaining human/operator decision and blocked command. Repeat with the same human holding all roles without requiring redundant approvals.

## C3 — A package name does not yet define what is immutable

**Location:** inserts 2, 4, 5 and 8; original Chapters 3–5 and 8.

**Verified:** `artifacts/artifacts.proto:58–84` writes an unversioned body request and attachment metadata (asset ID, role, aspect ratio, alt text and position). Its current-revision contract is body-focused. `claims/claims.proto:69–75` anchors citations by integer spans and body; it does not expose a draft revision or anchor unit. The missing handlers cannot be used to prove wider invalidation. The capture does not establish a package/crop/media-version record.

**Counterexample:** Wording r2 is unchanged, but image 2 finishes admission, a crop hides a disclosure, or an emoji before a supported sentence shifts offsets. A method adapter resolves a newer guide snapshot. The UI still has r2 and an old “accepted” decision, yet its newly rendered package is different. “Content-addressed or version-bound” permits materially different behavior unless the identity covers a canonical manifest, its referenced revisions and the exact rendering inputs.

**Required design:** Supply one populated package manifest defining body/structured-thread representation, source/claim anchors, immutable media bytes or asset version, placement/crop, alt text, method/adapter pin and decision scope. Define what is frozen versus revalidated. Choose one anchor convention and conservative stale-anchor behavior. Resolving pending media must create a new package and must not retroactively fill a reviewed empty slot. A byte version and a current permission/release check are separate requirements.

**Retest:** Hold body r2 constant while changing each consequential non-body input; show the new package ID and precisely which decisions survive. Add a Unicode edit before a claim and demonstrate that unsupported text cannot retain an old green span. If exact original media is unavailable, the manifest remains blocked rather than fabricating a digest.

## C4 — Scope idempotency to an immutable request, not merely a string key

**Location:** inserts 3, 6 and 10; original operation envelope and intake specimen IA-01.

**Verified:** `channel-manager/api/internal/channelmanager/service.go:1164–1170` returns the prior release found by key before comparing identity, draft, lane or asset inputs. Its `Enqueue` at 694–749 accepts a key but does not itself deduplicate it. Content Desk body update, approval and suggestion-adoption DTOs lack the proposed revision/operation fields. The generic envelope is correctly proposed, but existing behavior is not yet that envelope.

**Counterexample:** A release times out; the user changes the selected identity or corrects r2 to r3; recovery reuses the original key. Current release code can return r2/account-A's old receipt as if it answered r3/account-B. Conversely, always deriving import identity from source/revision/target/action makes an intentional second copy indistinguishable from a retry.

**Required design:** Bind each operation to a canonical request fingerprint covering action, owning scope, target/base revision, package, destination and timing. Same key/same request returns the original result; same key/different request returns a typed conflict and never success. Define parent-plan identity versus child-command identity, key retention/lookup, and an explicit new-intent/copy nonce for an intentional second effect. Lookup must remain access checked. No new key may be issued solely to escape an unknown outcome.

**Retest:** Lost receipt, stale edit, different identity, two-device retry and intentional duplicate import. Show one original effect, one conflict where required, and an explicit separate effect only for the new approved intent.

## C5 — The proposed partial-thread state needs an itemized release contract and timing rule

**Location:** inserts 5 and 8; original Chapters 6 and 8, partial/uncertain outcomes.

**Verified:** `channel_manager.proto:72–79` exposes one platform-post ID, URL and first-comment status. `service.go:968–1001` sets `partial` when the first comment fails; `Complete()` at 365–366 treats partial as complete, and a later `CompleteRelease` returns it unchanged. `artifacts/artifacts.proto:142–151` receives one post identity in its published/partial callback. These contracts do not establish ordered thread items or recovery of an unposted suffix.

**Counterexample:** A three-post thread publishes items 1 and 2. Item 3 has an unknown outcome, the latest authorized instant passes, then a late receipt confirms item 3. A continuation or new authorization may already be in progress. One root URL and a generic partial flag cannot distinguish a failed first comment, two-item prefix, unknown suffix, later completion and a separately authorized correction. The text does not say whether the allowed window covers start of dispatch or every external item.

**Required design:** Define stable item IDs/order/dependencies and an append-oriented per-item attempt/receipt ledger, plus a derived aggregate state. Explicitly map or extend native partial semantics. Define when timeout remains unknown, how late receipts reconcile, how repair actions reference original items, and whether every external step must remain inside the accepted timing/consent window. Preserve published evidence even when the action is cancelled or repaired.

**Retest:** Prefix success, unknown last item, consent or time expiry, and an out-of-order late receipt. No duplicate prefix, no automatic suffix after authority expiry, no erasure of the earlier partial observation, and no false claim that all items were published at one timestamp.

## C6 — Give shared preparation work a source identity and a lifecycle

**Location:** insert 6, “Work” and recoverable plan admissions; original Chapter 6.

**Verified:** Planner `source-implementation-plan.md:752–859` requires registered sources, source-object and constraint revisions, stable source-to-work mappings, tombstones and source-owned completion. It is an illustrative future contract. The captured `work/work.proto:19–40` exposes a work ID and display `source_label`, not source-object identity. `calendar/calendar.proto:119–173` supplies native preview/apply proposal IDs and revision/idempotency fields; `integrations/integrations.proto:9–16` currently concerns calendar connections, not the proposed source adapter.

**Counterexample:** One proof-check task serves three variants. A partial plan save is retried, then the user removes two variants and changes the remaining release time. What proves that this is the same work item, which source revision carries its changed prerequisites, and whether its already accepted Planner allocation is still useful? A label such as “Marketing” cannot prevent duplicate tasks or define completion. Finishing a Planner session must not mark the source proof check complete.

**Required design:** Choose the Content Desk-owned work object and canonical source key, including scope and occurrence where relevant. Map source work → Planner projection → allocation and specify update, shared dependency, removal, cancellation, completion and tombstone behavior. State which constraint changes invalidate a proposal and how an accepted reservation becomes conflicted rather than silently moved. Keep Calendar preview/apply and source projection admission as distinct results.

**Retest:** Three variants share one task; duplicate projection is delivered; two variants disappear; remaining effort changes after a preview; the source is unavailable during completion. One source task persists, allocations retain their IDs, source completion remains pending, and no unrelated Planner titles leak.

## C7 — Assistance permission and cost must attach to the actual job and all attempts

**Location:** insert 5, “Visible optional assistance and cost”; insert 12; original Route B and IA-01.

**Verified:** `artifacts/artifacts.proto:95–118` commissions with only draft ID/action, returns run identifiers/status, retrieves status/body and adopts by commission/body. It does not expose selected span, base revision, method/context pin, provider choice, per-request ceiling, cancellation or usage accounting. Prose `DATA.md:38–64,88–92` establishes file authority, immutable candidate history, per-session budgets and cost across the full candidate set. This does not establish the richer agent-job UI contract or a fresh runtime price.

**Counterexample:** Two selected-paragraph requests run concurrently against one draft; each is shown the same remaining allowance. One is cancelled but continues billable provider work. A retry returns after the draft's method was deliberately changed. A completed-candidate view showing only the adopted suggestion's cost omits rejected candidates and failed attempts. Neither a session ceiling nor “remaining allowance” resolves which authorized work was actually reserved and paid for.

**Required design:** Provide a capability map per Agent Manager/Prose route and a job manifest with immutable input revision/span, permitted context, adopted method/adapter, runner/profile, budget scope, job/attempt identity and allowed cancellation semantics. Name the budget owner, reservation/accounting behavior under concurrent work and unknown charges, and which charges are estimates versus measured. Aggregate all related attempts/candidates without charging a replay twice. A missing control is unsupported, not a simulated enabled Cancel action.

**Retest:** Concurrent jobs near the ceiling, cancellation after provider acceptance, lost response, partial candidate and changed method pin. Account for every incurred attempt, block overspend when enforcement is unavailable, preserve only received text, and require a revision comparison before adoption.

## C8 — Preserve the asset reuse rule under concurrent admission, not just after publication

**Location:** original Chapter 7 cross-identity restriction; inserts 6 and 8 batch/release behavior.

**Verified:** `service.go:1179–1187` rejects an asset associated with another identity in `AssetPublications`; lines 1004–1005 populate that association only when release completion is recorded. The inspected typed handler checks released-asset existence before enqueue (`module.go:203–219`). This source pass does not reproduce any runtime race, and does not establish an execution-time cross-identity lock.

**Counterexample:** The same released asset is admitted to personal and brand identity queues before either publishes. Both initial checks see no publication. If the product promises to retain the rule, which action loses when one completes first? What if both start externally before either completion is observed? A final recheck after completion alone cannot undo an already dispatched second action.

**Required design:** State whether admission reserves cross-identity use, or whether the native dispatch owner performs an atomic conflict decision before any external attempt. Define unknown-attempt reservations and cancellation/release behavior. Surface the conflicting identity/action without exposing unauthorized content. Legitimate reuse remains a separate policy decision; reimporting identical bytes remains disallowed as a bypass.

**Retest:** Two simultaneous identities, lost completion receipt and cancellation during execution. One applicable policy decision governs every attempt; an unknown first outcome cannot free the asset for a conflicting second attempt. This is a new concurrency case beyond the older review's sequential reuse test.

## C9 — Make observations specific enough for multiple destinations and corrected imports

**Location:** insert 9 and the one-variant/many-intentions model in insert 2; original Chapter 9.

**Verified:** Channel Manager `MetricSample` (`service.go:352–363`) records release/draft/metric/value/observation time. Content Desk `ledger/ledger.proto:55–86` ingests that shape and returns latest readings at draft level. It has no exposed metric window, denominator, measurement-definition version or correction lineage. Native PublishRecord's read DTO contains draft/post/URL, not exact package/version.

**Counterexample:** One variant is used on two accounts. A reported seven-day click count is later superseded by a provider-verified lifetime count, and the same imported row is reissued with a corrected value. Draft-level “latest clicks” can now compare different definitions, overwrite rather than reconcile evidence, or misattribute one account's result to the other. “Deduplicate by source identity/window” does not define corrections versus a new measurement.

**Required design:** Distinguish immutable observation identity, external release/item and exact package, metric definition/window/denominator, collection/evidence basis, and supersedes/corrects relation. Native draft aggregate views should be projections with disclosed inclusion rules, not the authoritative observation identity. Keep owner-reported evidence available when verified evidence arrives; show its reconciliation rather than silently rewriting history. Qualitative evidence needs its own permitted attribution scope.

**Retest:** Two releases of one variant, repeated import, corrected provider sample, missing denominator and conflicting reported/verified value. No inflated total, no cross-account substitution, and a learning decision can be reopened on the exact observation versions it used.

## Closure standard

For each C1–C9, record a disposition and point to the changed normative contract, populated specimen or explicit justified deferral. A deferral must disable or narrow the affected UI promise rather than leave an enabled control whose effect is undefined. Recheck the concrete counterexample against the final assembled 0.3 dossier, because appended inserts can leave old definitions simultaneously normative. None of these repairs requires runtime execution, account access, canonical source edits or a new accepted architecture in this design commission.

## Bounded repair recheck at 04:20 UTC

Inspected the assembled `i16-design-v03/source/dossier.md`, SHA-256 `78cf05ecd5b0b7266f6585950c1ec1d91c3d33a12cfffdd86903f51f5fce3283`, before and after this pass. Its added contract basis is `source/contract-resolutions.json`, SHA-256 `56bc52ab0b399bff8e34ed759e491bd547a5a0361c378cb462e351b26d3c67f8`. The source snapshot was not rerun. Recheck scope was the nine original counterexamples and overlapping old/new normative text, not a new exhaustive product audit.

**Disposition: C1, C2, C3, C5, C6, C8 and C9 close at design-contract level. C4 and C7 have their substantive repairs but retain two direct contradictory instructions in the assembled text.** Resolve the two sentences below before recording all nine closed. This does not pass runtime, source application, owner acceptance, pixels or native/PDF delivery.

- **C1 closed.** Chapter 2 now chooses one owning project, at most one campaign and one primary same-project family; collections are projections, multiple permitted causal references remain possible, and detach/archive/move behavior and slot accounting are explicit. The D1/D2/D3 trace answers the prior membership/move counterexample without changing siblings or their release history.
- **C2 closed.** Chapter 8 explicitly distinguishes author adoption, craft review, a new scoped human wording decision, native human-only approval and separately bounded release authority. Supported-but-unverified claims cannot pass the native gate. The explicit supersession paragraph resolves the older all-fields review-package definition for wording-only assessment; that old passage is not evidence that account-free native approval exists. Same-human combined presentation retains separate obligations and receipts.
- **C3 closed.** P2/P3 establishes immutable manifest inputs, media version/byte identity, revised package identity for resolved media, method/adapter pins, and a proposed revision-bound UTF-16 anchor with quoted-span/context verification. Current permission and readiness are revalidated rather than frozen indefinitely. These are sufficiently concrete design semantics; actual canonical serialization/hashes and adopted native anchoring remain later implementation obligations.
- **C4 substantially repaired; residual R1 open.** Canonical fingerprint, same-key conflict, explicit separate-copy identity and parent/child command identities resolve the original changed-destination and deliberate-copy counterexamples. The Chapter 3 media-retry sentence still conflicts with that rule, as detailed below.
- **C5 closed.** The design now maps native first-comment partial separately, requires itemized attempts/receipts and derives the aggregate. It chooses still-valid authority at each new external-item dispatch, retains in-flight outcomes after expiry, and supplies the T1/T2/T3 late-receipt trace. Read this with the retained provider-owned/manual control limits: it does not newly establish an ability to stop an already provider-controlled item.
- **C6 closed.** Proposed MarketingWorkItem ownership, shared dependencies, source-to-projection-to-allocation mapping, constraint changes, tombstones and source-owned completion resolve the three-variant retry/removal/unavailable-source case. Native owner scope must still be part of the authenticated source-registration namespace when implementing the shorthand work key; a display label remains expressly insufficient.
- **C7 substantially repaired; residual R2 open.** The route capability/job manifest, concurrent budget enforcement, whole-attempt accounting, unknown-charge handling and stale-result/method treatment resolve the original substantive gaps. The earlier running-state sentence still promises cancellation unconditionally, as detailed below.
- **C8 closed.** Atomic Channel Manager reservations, an explicit prohibition on simultaneous conflicting dispatch, retained unknown reservations and confirmed-publication associations resolve the original simultaneous same-asset example. Preserve the existing asset-ID policy when implementing the identity/version key: version metadata must not make two versions of the same native asset ID independently eligible across identities. The rule's scope is retained, not broadened by this review.
- **C9 closed.** Observations now bind release/item/package and definition/window/denominator; correction/reconciliation retains original evidence and exact learning bases. Two accounts cannot collapse through draft-level latest readings. Conflicting same-provider-ID values become reconciliation work rather than overwriting history.

### R1 — Media retry still changes a fingerprint while reusing its operation key

At assembled line 175, Chapter 3 says: “Retry media targets the original operation/media key and current destination revision.” At line 115, Chapter 2 says the operation key binds the expected base revision and rejects a changed fingerprint. After an admitted draft advances from r2 to r3, those instructions disagree. A changed/repaired rejected media request can have the same issue even if the body is unchanged.

**Narrow repair:** Distinguish stable intake/media *business identity* from a command attempt's key. An unchanged request against the original base reuses its command key. First reconcile an unknown old command. After a definite rejection, or an explicitly reviewed revision/payload change, use a new guarded child-command key linked to the same media slot and intake intent. This does not authorize a duplicate draft or asset.

**Retest:** Save r2, lose image-admission response, edit to r3, then reopen Resolve image 2. The system first recovers the r2 attempt; any required r3 mutation is a new linked command with a reviewed expected base. No changed fingerprint is accepted under the old command key.

### R2 — The generic running-state sentence still promises unsupported cancellation

At assembled line 319, Chapter 5 requires a running indicator to include “a useful cancel route.” At line 329, the repaired contract correctly says “Cancel is offered only when the route supports it.” A qualified authoring route may produce results and disclose charges while lacking an abort capability, so these are simultaneously normative conflicting instructions.

**Narrow repair:** The running state always shows the actual stage; it offers Cancel only when supported, and otherwise gives the supported observation/leave-editor route plus the provider's actual stop/control limit. Qualify the A2 cancellation exercise as using a cancellable route, and include the truthful unsupported-control case when exercising another route.

**Retest:** Two route records, one cancellable and one not. Only the former renders an enabled Cancel control; both retain editor availability, received partial output and honest incurred/unknown usage.

### Additional normative-conflict check

Chapter 11 line 711 expressly identifies the captured native DESIGN token contract as binding and makes Modernist Press contingent on a scoped native amendment under CQ12. CQ12 retains owner review and exact specimen/token acceptance. This resolves the potential visual-authority conflict at design level; no token/source amendment has occurred. The proposed permanent-inspector and accessibility obligations remain retained.

The reviewed assembly does not claim that a receipt, role, package, adapter or mockup proves runtime qualification. Recheck only R1/R2 and any dependent changed text after repair; no second broad source discovery is needed for this bounded closure.
