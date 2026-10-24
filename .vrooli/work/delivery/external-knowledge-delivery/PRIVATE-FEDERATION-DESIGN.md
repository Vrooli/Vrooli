# Private source federation investigation

Observed/researched **2026-10-14**. Preparation only; no product certification. Governing direction: `docs/architecture/external-knowledge-search.md` and this goal home's `GOAL.md`, `RESEARCH.md`, `QUEUE.md`. Read with `DOCUMENT-FIDELITY-DESIGN.md`. The native team remains disabled. Integration-hub's repeatable Google CLI/UI connection is an early outcome after later activation; staged operator consent gates dependent live work, not independent fixture work.

## Conclusion and current path

Private federation needs verified request identity **and** current domain-owned object grants at every search/context boundary. Reuse api-core identity/authn/provenance, scenario-authenticator and Agent Manager; extend their delegation seams rather than make Search Hub a signer, private identity realm, billing owner, Google client or corpus store.

**Current verified-agent forwarding exists.** Standard `api-core/server` installs passive provenance middleware. It verifies `X-Agent-Identity-Token` through Agent Manager, and shared `ForwardingTransport` copies only a context-held verified token to localhost/loopback destinations. Search Hub's default HTTP client uses this transport. Its `context.WithoutCancel` coalesced call retains context values. Consequently, the explicit fan-out code's descriptor-only headers do **not** prove that every caller identity is absent downstream. [L01–L03] This corrects the broad interpretation of RESEARCH R02; historical document-manager federation restrictions remain scoped product-contract evidence.

The forwarding is an internal run-attribution mechanism, not a complete private federation contract. It does not exchange a human/MCP bearer for the destination audience, restrict the run token to the selected provider, establish collection grants, or make provider handlers enforce them. Current gaps:

| Surface | Concrete observation and implication |
|---|---|
| Hub ingress | `main.go` passes no `Authentication` configuration; routing handler has no identity/capability check. Shared provenance remains passive on invalid/unavailable identity. A private handler must fail closed explicitly. [L01–L03] |
| Fan-out | `callProviderDirect` renders query/limit/type/scope and applies descriptor headers. No dedicated audience-bound delegated credential seam exists. Current `scope` is a corpus facet, not a grant. [L03–L04] |
| Coalescing | Key is provider ID + query + limit + scope. It excludes caller, workspace, grants and policy generation. One waiter's verified context can supply the call another waiter joins. This is unsafe for caller-dependent results. [L03] |
| Registry/status | Descriptor scope is PROJECT/EXTERNAL routing/cost policy, not privacy. Headers are arbitrary descriptor data; validation checks endpoint shape, not protected identity headers. Status caches one response globally for five seconds and exposes provider counts/generation/source facts. [L04–L05] |
| Direct document search | Handler accepts `CallerMaxPrivacy`, collection ID and `Federated` from request; repository filters those values before scoring but has no verified account/workspace/object-grant lookup. Corpus `CanRead` checks federation flag/class only. This is a pre-scoring filter, not caller authorization. [L09] |
| Registration/searchability | Document-manager descriptor is experimental, maps anchor URI as snippet, and registration function does not submit an HTTP registration. Descriptor existence is not private serving acceptance. [L09] |
| Ranking/eval | Hub fuses snippets/titles into cross-provider reranking. Eval provider client repeats descriptor-driven transport; suites/runs persist protojson, including queries and scored-hit titles/IDs. Background recovery/eval contexts are scheduling markers, not grants. [L06–L08] |
| Observability | Query telemetry stores unsalted SHA-256 and per-provider counts/latency; hash guessing and cross-user correlation remain possible. Routing internal-error logs include raw query; scenario middleware logs RequestURI. Shared server access log uses URL.Path. [L07] |

Current SearchHit is defined in `search-hub/v1/routing/routing.proto`, not a common identity contract. It carries metadata Struct, rank evidence, locators and confidence; common proto carriers add evidence/confidence vocabulary. Provider-declared metadata transported by ResultMapping is not a verified grant or signed identity. [L04]

## Existing identity owners and limits

`identity.Principal` separates actor Subject from agent Owner, exposes verified source/audience/expiry, and leaves authorization to domains. `authn.RequireCapability` resolves coarse scopes; `RequireOperation` handles finite reviewed operation grants. Owner bindings require exact verified provider identities; email, UID, arbitrary headers and equal subject strings cannot create a mapping. Authentication middleware is deliberately passive, so its presence alone does not protect private reads. [L10]

Shared JWT verification checks RS256/kid, signature, configured issuer/audience, expiry and time claims through cached JWKS. This local verification does not consult authenticator session blacklist/auth-version state. Authenticator `Validate` does; refresh/logout update relevant state. Its integration contract favors local JWKS verification at hosted scale, so define revocation freshness through shared lifecycle/revocation mechanisms rather than adding a remote Validate call to every consumer ad hoc. [L11–L12]

Scenario-authenticator issues resource-specific audiences from operator-owned registration, preserves audience on refresh, and has configured service-principal exchange. Service profiles fix resource/scopes; credentials remain in credential authority, and exchanged tokens identify the service. A service token is not the user it represents. No inspected account RPC provides general downstream subject+actor token exchange. The create-run-only accepted-audience adapter is a deliberately narrow existing exception, not permission for document-manager to accept Search Hub or default-realm audiences indefinitely. [L11–L12]

Agent Manager HMAC run claims contain verified owner/workspace/scopes, run/task identity and expiry, without an audience field. Attenuation preserves owner/workspace, forbids scope/expiry widening and refuses inheritance of interactive operation grants. Online verification requires the active token hash/run association and rejects revoked tokens; generic signature verification alone is insufficient. [L13] Reuse these controls when exchanging an agent identity; do not copy its full capability set to every provider. Run identity and account identity remain distinct.

## Proposed per-hop authorization contract

All names below are interface recommendations, not existing wire APIs.

1. **Ingress:** shared verification creates an immutable `VerifiedCaller` in server context: actor kind/ID, verified owner where applicable, issuer/source, deployment-owned workspace, audience, expiry, run/session/delegation reference. HTTP query fields may request filters but cannot set any verified field. Resolve coarse `search`/`context-read` capability, then eligible private providers/collections. Explicit `--all`, `--group`, guessed IDs and `--scope` remain within that boundary.
2. **Delegation:** an existing identity authority exchanges a verified inbound credential plus authenticated calling service identity for a short-lived credential targeted to one destination. Scenario-authenticator owns interoperable resource/audience issuance; Agent Manager owns live agent attenuation and run validity. The exchange validates both and intersects caller authority, service delegation policy, requested operation and domain grant references. Search Hub must not sign its own caller assertion.
3. **Provider:** verify destination audience, issuer/key/type/time, current calling service and delegated actor, then resolve current owner/account/workspace/collection grants in the provider repository. Returned rights are an intersection, never a union of service and user scopes. The router's service principal alone cannot read personal collections. Missing grant or unavailable required verification denies private work.
4. **Retrieval:** derive `AuthorizedSearchScope` server-side: account/connection identity, workspace, collection/source IDs, permitted operations, policy/grant epoch and privacy/residency stance. Apply it before lexical/vector candidate selection, snippets, score calculation and reranking. Recheck visibility at final emission and each bounded context fetch.

Suggested shared client seam: `DelegationClient.ForDestination(ctx, verifiedCaller, destination, operation, grantRefs)` returns an opaque credential, effective scope digest and validity; `AuthorizedProviderClient.Search/Fetch/Status` consumes it. Domain seam: `CollectionAuthorizer.Resolve(ctx, verifiedCaller, selector)` returns immutable authorized scope plus freshness/epoch. These references may be opaque server-resolved grants; bearer tokens and raw Google access/refresh tokens never enter descriptor metadata, SearchHit, logs or agent-facing responses.

Preserve both delegated user/agent identity and service actor. RFC 8693 describes subject/actor exchange and `act`; it does not make revocation propagate automatically. Its historical nested actors are audit information, not additional rights. [W03] Select one audience per hop and explicit token type. RFC 8707 resource targeting, RFC 9700 audience restriction and RFC 8725 audience/algorithm validation support this design. [W02, W04–W05] If OAuth JWT access tokens are selected, follow their token-type/claim-validation profile rather than assume current generic JWTs already meet it. [W06]

For local-only agent calls, extending the existing Agent Manager identity contract with destination-bound delegated reads is another implementation option. It must retain active-run verification and use a governed issuer/verifier seam. Select one primary wire approach with existing owners; do not maintain parallel unqualified identity protocols. Protect destination resolution, paths and redirects so a descriptor cannot send proof to another loopback service or an external URL. Strip protected static headers; loopback reachability alone does not prove service identity.

## Provider visibility, passages and races

Document-manager must add object grants and source membership to its own corpus/source domain, with account/connection and workspace isolation. Caller privacy class is a requested output constraint at most; effective permission comes from verified policy. A collection-level federated flag is opt-in exposure, not permission for every principal. Shared search QueryFilter can apply metadata constraints to all vector legs, but filter construction and policy completeness remain owner responsibilities.

Keep current visibility independent of retained index generations. Authoritative withdrawal first makes removed/out-of-root/access-revoked sources unservable; cleanup follows. Publication and generation rollback must recheck the current policy epoch. When a deletion races ranking, exclude it before snippet/reranker access where possible and before final emission; no stale generation resurrects rights. Remote ACL detection has bounded latency, so show verification age and defined offline policy rather than claim instantaneous Google revocation detection.

Agent Manager conversation search offers existing visibility/catalog/tombstone mechanics (`VisibleDocument`, visible lexical/semantic projections), but these are projection controls, not evidence that external collection/account grants already exist. [L14] Reuse the pattern without transferring domain authority.

Context API should accept opaque source/unit/version locators and bounded adjacent-unit requests. Resolve grant/membership again; a citation or prior hit is not an access capability. Return source version, capture/index/ACL freshness, actual passage origin and coverage from document-manager. Cross-account equal hashes, guessed anchors, title search, exact cell fetch and historical-version fetch receive the same checks. Bind cursors to caller/scope digest, query/filter recipe and serving generation; validate authenticity/expiry and reject reuse under a different grant. Passages retain captured-version identity even when live source links change.

In-flight/result caches must key actor/owner/workspace, effective object-scope digest, policy epoch, query, provider and recipe/generation; disable private coalescing initially until proven. At join/emission, recheck grant validity. Never inherit one caller's decision merely because queries match. Distinguish transport outage, denied grant, authentication required, source stale and semantic unavailable without exposing unauthorized source names or existence.

## Privacy across router surfaces

Authorize **eligible routing metadata** before provider selection or model invocation. Descriptions/examples must not contain private folder titles, account identifiers, source excerpts or personal query examples. A private query cannot automatically escalate to web/external providers or remote inference because project results are empty. `--all` is provider selection, not egress consent. Provider and Hub query/rerank residency must use compatible permitted roles; current local TEI/Ollama seams do not qualify every configured fallback as local-only.

A proposed additive descriptor policy separates `public-project` and `private-authorized` serving class from PROJECT/EXTERNAL routing scope, with destination audience/delegation support, bounded search/context/status capability, query/rerank residency and result metadata schema. Reject protected credentials/impersonation headers in registration. Authenticated provider ownership governs updates; registration/control-token possession does not grant data access. Existing token re-registration permits empty-token reacquisition, so qualify registry governance before using it to identify private trusted destinations. [L04]

Expose non-content service liveness separately from authorized corpus status. Filter provider-list/status/explain/coverage and snapshot fields by visibility; private source counts, generation names, index times and failure reasons can reveal account activity. Partition status caches or use public health only. Authentication denials must not poison global provider circuits/demotion or alter public query quality. Private zero-results are not fleet health evidence.

For private queries, default to no stored raw query, no persistent unkeyed query hash, no title/snippet in generic eval records, and no private query text in free-form errors. If correlation is selected, use a rotating tenant-scoped keyed digest with limited retention and access; it remains sensitive metadata. Apply the same policy to traces, CLI history/debug, provider logs, errors, metrics exports and generated questions. Redact only at persistence is too late for remote model egress.

Run deterministic synthetic privacy suites through real direct and federation boundaries. Live personal-data overlays remain operator-local, bounded and separately retained; export only sanitized aggregate receipts. Scheduled eval/recovery needs a selected service grant and synthetic collection, never a saved owner bearer or scheduler context treated as authorization. Current suite/run protojson persistence must be reviewed before private examples are admitted. [L06–L08]

Public project providers remain compatible: unauthenticated calls may still query explicitly public corpora; private selectors require verified authority. Anonymous omission of private providers avoids disclosure. Presented invalid private proof does not trigger an owner/local-machine fallback. Existing explicit personal-local exchange requires operator intent and is not automatic recovery from a denial. [L10]

## Bounds and remote consumer

Hub currently enforces nonempty query and default result limit, per-provider/query timeouts, concurrency and an 8 MiB response reader; it does not show a hard positive upper limit/query-byte bound. Read cap does not explicitly detect overflow using cap+1. Document-manager materializes candidates; vector scan reads the vectors table before filtering candidate IDs. Those mechanics need bounded authorized queries, not a mandatory Qdrant migration. [L03, L09]

Proposed admission defaults: 8 KiB query bytes, 100 results/provider, 20 authorized providers/request, 8 MiB checked response cap, 64 KiB text context/fetch and model-specific token limits after composition. Use typed errors and opaque continuations; never silently truncate evidence. Bound metadata depth/strings/locations, lexical expression/regex cost, structured cell ranges, vector dimension/nonfinite values and rerank candidate/text budgets. Rate/cost admission attaches to verified actor/tenant while preserving existing billing ownership.

A thin remote API/MCP adapter exposes only search, bounded context and authorized status. It uses the same producing provider grants; no shell, arbitrary localhost proxy, unscoped URL fetch or raw source token. If its public canonical resource differs from Search Hub, use a separate audience and downstream exchange; if it is the same protected resource, document that deployment explicitly. Current MCP authorization (latest resolved to **2026-07-28**) requires protected-resource metadata, authorization-server discovery, resource indicators and intended-audience validation; OAuth 2.1 remains a referenced draft. Do not assume an accounts Login/JWKS API is already an interoperable OAuth authorization server. [W01, W07]

MCP forbids wrong-audience token passthrough and identifies confused-deputy/SSRF risks. [W08] Reuse existing identity authorities for missing protocol discovery/issuance support. Select advertised registration flow for the actual client; qualify PKCE, issuer/redirect validation and transport/session behavior against the chosen protocol version.

Official ChatGPT documentation establishes supported custom MCP-app configurations, not the operator's actual always-on Dots capability. [W09] Return condition: in that real environment, discover the selected tool/schema, authenticate with the intended account/audience, search a synthetic permitted source, fetch its versioned passage, prove a denied cross-collection request, then revoke and observe refusal. This investigation performs no account connection or live call and cannot certify that availability.

## Owner changes and acceptance

| Owner | Concrete required outcome |
|---|---|
| api-core/authn/identity/provenance + CLI | Verified caller and audience-safe delegated client interfaces; explicit protected-read requirements; expiry/revocation state; no owner substitution after refusal. |
| scenario-authenticator + Agent Manager | Governed resource-specific exchange/service actor and live run attenuation; actor/owner/workspace preservation; documented token/parent revocation behavior. No Search Hub signer. |
| document-manager | Source/object grants, live membership epochs, scoped candidate stores, context/citation authorization, retention withdrawal and privacy-qualified inference. |
| search-hub | Eligible private-provider routing, protected registration metadata, delegated fan-out/eval/status, authorization-partitioned caches, bounded inputs and private telemetry/errors. |
| shared search / proto owners | Generic typed provenance/coverage/context carriers and constrained retrieval seams; additive schema changes and owner regeneration. Domain grants stay domain-owned. |
| remote API/MCP adapter | Protocol-compatible protected ingress and bounded tools; same grants/evidence; actual consumer verification. Integration-hub remains Google connection lifecycle owner. |

Every negative gate runs direct **and** federated, plus context/status where applicable. Assertions include no snippet, title, metadata, scorer/reranker input, metric/eval/log disclosure—not merely an empty final hit list.

| Given / action | Required result |
|---|---|
| Anonymous, expired, forged, wrong issuer/algorithm/audience/type; conflicting proofs | Private calls refuse; public compatibility follows explicit public policy; no privilege fallback. |
| Account A guesses B's collection/file/hash/anchor/cursor; wrong workspace | No candidate/model input or existence disclosure; same boundary on context and historical evidence. |
| Actor/owner header forgery; service token without user grant; child requests scope/expiry widening | Trusted fields remain unchanged; exchange/provider refuses. |
| Identical simultaneous query across unequal grants | No coalesced result/context cross-talk; late joining cannot borrow first caller's proof. |
| Parent/run/session/grant revoked; proof/cursor replay; token expires in-flight | Current epoch/liveness gate prevents emission and refresh; replay policy and revocation bounds are evidenced. |
| Delete/move/ACL withdrawal races scoring, publication or generation rollback | Withdrawn content stays unservable across every view/generation/cache; queued work cannot restore it. |
| Identity/grant owner unavailable; stale remote ACL; JWKS rotate/outage | Required private verification fails closed; status states uncertainty; public and authorized available lexical lanes remain honest. |
| Semantic/rerank outage; malformed/oversized output; huge input/vector/range | Bounded work and accurate partial errors; available authorized evidence survives without widening grants. |
| Private selector/query plus unauthorized status/list/explain/eval/metrics read | No source counts/names/query hash/title/activity leak; public circuits and quality do not change due to denied private traffic. |
| Private query with empty results or external-provider selection | No automatic external/model egress without selected policy; residency evidence covers every leg. |
| Descriptor protected headers, bad path/destination/redirect; forged eval marker/control token | No impersonation, secret forwarding, SSRF or private eval bypass. |
| Real permitted exact/paraphrased query, followed by bounded fetch and revoke | Same attributable versioned evidence direct/federated/actual consumer; revocation refuses within selected freshness bound. |

Proposed defaults: five-minute maximum delegated credential lifetime, clipped to parent/grant expiry; current provider grant check each request, no positive cache initially; private coalescing disabled; deny when required current grant verification fails. Select bounded remote ACL age/offline policy separately—five-minute sync is not a revocation guarantee. Open decisions: primary exchange wire approach, authority coordination, sender-constraining/replay requirements, hosted revocation feed, private telemetry retention and numeric quality/latency thresholds. Use scoped Test Genie phases under `docs/TESTING.md` in future delivery. No tests ran here.

## Evidence index

Local paths are relative to `/home/matthalloran8/Vrooli`; all inspected 2026-10-14. No credential files or source accounts were read.

| ID | Exact paths |
|---|---|
| L01 | `packages/api-core/server/server.go`; `packages/api-core/provenance/provenance.go`; `packages/cli-core/cliutil/httpclient.go` (`TrustedIdentityDestination`) |
| L02 | `scenarios/search-hub/api/main.go`; `scenarios/search-hub/api/internal/httpc/doer.go`; `scenarios/search-hub/api/handlers/routing/connect_handler.go` |
| L03 | `scenarios/search-hub/api/internal/routing/router.go` (`Query`, `callProvider`, `callProviderDirect`, `maybeRerank`) |
| L04 | `packages/proto/schemas/search-hub/v1/{routing/routing,registry/registry}.proto`; `scenarios/search-hub/api/internal/providers/{call,adapter}.go`; `scenarios/search-hub/api/internal/registry/{validate,store,types}.go`; `api/handlers/registry/connect_handler.go` beneath search-hub |
| L05 | `scenarios/search-hub/api/internal/routing/status.go` |
| L06 | `scenarios/search-hub/api/handlers/eval/provider_client.go`; `api/internal/eval/{runner,store}.go`, `api/internal/eval/schema.sql` and `packages/proto/schemas/search-hub/v1/eval/eval.proto` |
| L07 | `scenarios/search-hub/api/internal/routing/telemetry.go`; `api/internal/telemetry/schema.sql`; `api/internal/middleware/logging.go`; routing handler in L02; shared accessLog in L01 |
| L08 | `scenarios/search-hub/api/internal/routing/{background,reranker,reranker_ollama,provider_breaker}.go` |
| L09 | `scenarios/document-manager/api/handlers/retrieval/connect_handler.go`; `api/internal/retrieval/{types,service,sqlite,registration}.go`; `api/internal/corpus/sqlite.go`; `.vrooli/search.json` beneath document-manager |
| L10 | `packages/api-core/identity/identity.go`; `packages/api-core/authn/{authn,operation,request_operation,local_exchange,caller_audience}.go` |
| L11 | `packages/api-core/authn/jwt.go`; `scenarios/scenario-authenticator/api/internal/accounts/{service,refresh,service_principal}.go`; `api/internal/authcrypto/jwt.go` beneath scenario-authenticator |
| L12 | `packages/proto/schemas/scenario-authenticator/v1/accounts/accounts.proto`; `scenarios/scenario-authenticator/docs/concepts/INTEGRATIONS.md`; `docs/concepts/{IDENTITY-AND-AUTHENTICATION,AUTHORIZATION}.md` |
| L13 | `scenarios/agent-manager/api/internal/identity/{claims,token,attenuation}.go`; `api/internal/orchestration/run_execution_reads.go` (`VerifyIdentityToken`) beneath agent-manager |
| L14 | `scenarios/agent-manager/api/internal/conversationsearch/{sqlite,repository,semantic_source}.go`; `api/internal/handlers/conversation_search_connect.go`; `.vrooli/search.json` beneath agent-manager |

Official primary sources opened **2026-10-14**; API/spec capability does not certify Vrooli implementation:

| ID | Source |
|---|---|
| W01 | [MCP authorization, 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization) |
| W02 | [RFC 9700 OAuth security BCP](https://www.rfc-editor.org/rfc/rfc9700.html) |
| W03 | [RFC 8693 token exchange](https://www.rfc-editor.org/rfc/rfc8693.html) |
| W04 | [RFC 8707 resource indicators](https://www.rfc-editor.org/rfc/rfc8707.html) |
| W05 | [RFC 8725 JWT BCP](https://www.rfc-editor.org/rfc/rfc8725.html) |
| W06 | [RFC 9068 JWT access-token profile](https://www.rfc-editor.org/rfc/rfc9068.html) |
| W07 | [RFC 9728 protected-resource metadata](https://www.rfc-editor.org/rfc/rfc9728.html) |
| W08 | [MCP security guidance, 2026-07-28](https://modelcontextprotocol.io/docs/2026-07-28/tutorials/security/security_best_practices) |
| W09 | [OpenAI developer mode and MCP apps](https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt) |
