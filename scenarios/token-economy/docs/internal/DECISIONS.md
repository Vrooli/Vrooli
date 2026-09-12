# Decisions — Token Economy

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-18 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-18 | **A grant is a mandate.** The grant contract is authored congruent with the `treasury` mandate shape — subject, amount, scope, expiry, provenance — from the first commit. | The scenario was scoped in the same workshop as `treasury` and `persona`. A token grant carrying spend rules *is* structurally the object that authorizes real spend. | The two scenarios can share one policy model. The eventual real-value rail becomes a new adapter rather than a rewrite. Cost: a parity test (`TKE-P0-002`) and the discipline of recording every divergence in this table. | Divergence pressure that cannot be recorded as an intentional exception, or `TKE-P2-004` extracting the shared contract. |
| 2026-08-18 | **`treasury` is a contract sibling, not a dependency.** Neither scenario calls the other and this one must function fully with `treasury` absent. | Congruence could have been achieved by depending on a shared package. | Independent deployability preserved; each scenario is separately useful and separately sellable. Cost: congruence is maintained by test and discipline rather than by the compiler. | `TKE-P2-004` — extract a shared contract only once both have shipped enough to know what is genuinely common rather than coincidentally similar. |
| 2026-08-18 | **Balance is a projection, never a stored truth.** Every quantity is derived from an append-only journal; a cache may exist but the journal wins. | A stored total is an assertion nobody can audit. `money-ledger` made the same call for financial position. | The economy is fully explainable to a holder, which is the product's central promise. Corrections are compensating events. Cost: every read path must go through projection, and rebuild equality must be tested. | Never, under the current product shape. A performance problem would be solved by better projection, not by trusting a stored total. |
| 2026-08-18 | **Authority separation is a service boundary, not a permission check.** Two audience services: a minter service and a holder service. The holder service has no mint/grant/rule method at all. | A runtime role check is one bug away from being bypassed; an absent RPC is not. This mirrors `treasury`'s autonomy-toggle posture. | A holder with a valid token still cannot mint. The control is visible in the proto and in codegen. Cost: two audience service surfaces and two generated audience clients. Integration-only surfaces remain capability-minimal. | Never. If a legitimate holder-initiated mint appears, it is modelled as a *request* on the holder service that the minter service fulfils. |
| 2026-08-18 | **No real value, enforced by absence.** No price field, no payout path, no external transfer anywhere in the contract. | A policy check could be relaxed quietly in a later change; a missing capability cannot. This is also what keeps multi-holder balances clear of money transmission. | Adding a monetary path requires a visible contract change and a recorded decision here. Structurally asserted by `TKE-P0-014`. | `TKE-P2-001` only, and only after a recorded custody and regulatory decision. The structural test must be *amended*, never deleted. |
| 2026-08-18 | **SQLite, terminally — not as a starting point.** No shared resource of any kind. | Household volume with single-writer mutations. A household economy that requires infrastructure is not a household economy. | Laptop-runnable with nothing else started. Row-lock plus caller-supplied idempotency key follows the proven `landing-page-business-suite` credit-wallet pattern. | Concurrent external writers — realistically only `TKE-P2-001`. Nothing in P0/P1 produces contention. |
| 2026-08-18 | **`scenario-authenticator` is a hard, fail-closed dependency.** | The multi-holder isolation boundary is the failure that would end trust in a household product. Without a verifiable identity it is cosmetic. | Authenticated surfaces refuse rather than degrading when the authenticator is unreachable. Cost: the scenario is not usable standalone in the strictest sense. | Never. A client-side role flag is not an acceptable alternative at any point. |
| 2026-08-18 | **Isolation is enforced at the repository layer, not only the handler.** | A future handler written without the check would otherwise leak one holder's history to another. | Cross-holder queries return empty regardless of handler behavior; refusals do not disclose whether the other holder exists. Cost: every repository read carries a scoping parameter. | Never. |
| 2026-08-18 | **Rule programs use a closed condition vocabulary; caller-supplied code is never executed.** | A household economy that can execute caller code is a remote-execution surface wearing a reward-app costume. | Rule effects are explainable and refusals can name the rule that refused (`TKE-P0-003`). Cost: expressiveness is bounded by the vocabulary. | Never for arbitrary code. New conditions are added to the vocabulary instead. |
| 2026-08-18 | **No inference anywhere in the product.** Rule evaluation is deterministic. | An LLM-decided refusal cannot be explained to a child, which contradicts the transparency the scenario is built on. | Zero marginal cost, which is why the free/metered/gated split resolves to free for the whole core product. | Never for rule evaluation. A non-decisional convenience (suggesting catalog wording) could be reconsidered separately. |
| 2026-08-18 | **Operator entry is a first-class earning adapter, not a fallback.** | Several real earning sources will never have an API — a chore done offline is reported by a person. | The operator path and the programmatic path traverse identical code, asserted by test. Cost: no shortcuts for the UI path. | Never. This mirrors `money-ledger`'s treatment of manual entry. |
| 2026-08-18 | **Two audiences, one token contract.** The minter console is dense and operational; the holder view is sparse and child-legible. They share colors, type scale, status semantics and motion rules, and differ only in composition. | Forking the token contract would make the product feel like two products; forcing one density would make the holder view unusable by its actual reader. | The holder view *exceeds* the accessibility floor rather than relaxing it. Recorded in `DESIGN.md`. | Adoption of a different design kit. |
| 2026-08-18 | **Import of journal history is deliberately absent.** Export exists; import does not. | An importable journal would let a caller assert history the system never observed, defeating the audit property. | Migration from another tool is manual. | Never, while the journal is the audit authority. |
| 2026-08-18 | Requirement validation refs for unwritten tests live in `notes`, not `ref`. | `vrooli scenario requirements validate` errors on a `ref` pointing at a non-existent file, so a planned test cannot be declared through `ref`. | Registry validates clean at documentation stage; intended paths are preserved and recoverable. Cost: a manual move from `notes` to `ref` as each test lands. | The validator gaining a planned-ref affordance. Tracked in `PROBLEMS.md`. |
| 2026-08-19 | **Omit Treasury's `currency` field from `Grant`.** | Currency would make a token grant a monetary contract and violate `TKE-P0-014`. Token type identity supplies the local denomination. | Grant-to-mandate parity deliberately permits this one omission and tests that the omission stays recorded. | `TKE-P2-001`, after a custody and regulatory decision; amend the no-monetary-value test rather than deleting it. |
| 2026-08-19 | **Defer Treasury's `signature` field from `Grant`.** | A local-only household economy has no external counterparty to whom portable authority must be proven. | Local authority remains attributable through `authorizer`; parity deliberately permits the omission. | P1 delegated or externally relayed authority that creates a real verification boundary. |
| 2026-08-19 | **Public access DTOs belong to the access boundary, not to product domains.** | Reusing `mints`, `grants`, `holders`, `catalog`, and `redemption` messages directly made the technical access package a cross-domain wire dependency and failed proto-governance checks. | `MinterService` and `HolderService` are independently generated and stable; the access handler performs explicit translation while business logic remains in the seven product domains. Cost: DTO mappings must be maintained and tested. | A fleet-governed neutral contract package with a clearer ownership model. |
| 2026-08-19 | **Earning dedup outcomes are retained permanently at household scale.** | Expiring an idempotency key can turn an old adapter retry into a second grant. A bounded window therefore trades storage for silent balance inflation. | The database stores the adapter-scoped key, grant outcome, timestamps, and a reason digest rather than the raw payload. Replay remains safe indefinitely. Cost: rows grow monotonically until a non-household deployment justifies a governed archival design. | Sustained non-household adapter volume with measured storage pressure. |
| 2026-08-19 | **The redemption repository owns settlement; journal and catalog contribute transaction-bound operations.** | Reservation, inventory, debit, and audit history must become visible together or not at all, while the journal remains the balance authority and the catalog remains the inventory authority. | One SQLite transaction covers the redemption row, reservation, stock mutation, and journal event. Narrow transaction-bound adapters preserve domain ownership without permitting a relay failure to affect settlement. The optional notification relay runs only after commit. | A storage boundary that cannot share the same transaction; that requires an explicit outbox/saga design with equivalent failure-injection proof. |
| 2026-08-19 | **Actor provenance is stamped at the journal append choke point, using the shared runtime-attribution vocabulary.** | Every balance-affecting path must retain attribution even when agent verification is unavailable or invalid. Product authentication still owns authorization and supplies the operator subject, while verified runtime identity must remain distinguishable from operator activity. | The API-wide `api-core/provenance` middleware verifies agent claims through `cliutil.IdentityEnv`; the journal resolver records `verified`, `unavailable`, `invalid`, or `absent` on every new event and preserves verified agent subject/run identity. Unverified identity is recorded honestly rather than causing a household-token mutation to disappear. | A higher-blast-radius value rail where unverifiable callers must fail closed instead of being durably recorded. |
| 2026-08-19 | **Persist grant-rule row identifiers under their owning grant namespace.** | Built-in rule names such as `system:positive-amount` repeat for every grant, but the original SQLite primary key treated a rule id as globally unique and prevented a second grant. | Storage keys use `<grant-id>/<rule-id>` while the domain and wire contracts continue to expose the stable rule id. Multiple grants can carry the same declared rule without weakening uniqueness inside a grant. | A schema migration to a composite `(grant_id, id)` primary key. |
| 2026-08-19 | **CLI command ownership follows audience services, even when commands share a product-domain group.** | A convenient domain group such as `holders`, `catalog`, or `redemption` contains both operator and holder operations. Binding the whole group through one client would quietly collapse the minter/holder authority boundary. | Each command binds directly to its generated service descriptor: administration uses `MinterService`, holder view/browse/redeem uses `HolderService`, and earning uses `EarningService`. The `mints` group has a structural test forbidding holder-service bindings. Cost: mixed groups assemble selected primitive maps from two descriptors. | Never for convenience; a new operation moves only when its public authority contract moves. |
| 2026-08-19 | **Issued grant rules remain immutable; revoke and reissue rather than patching authorization evidence.** | The inherited access proto contained `UpdateGrantRule`, but no server behavior or product requirement defined how an edited rule should affect already-issued authority and historical decisions. | Phase 11 exposes every implemented lifecycle RPC and records `UpdateGrantRule` as an intentional CLI omission. Revocation plus a new idempotent grant is the explicit correction path. | A future requirement with temporal semantics for in-flight and prior redemptions, backed by tests and a recorded migration policy. |

## Grant/Mandate Parity Ledger

This table is the allowlist used by the generated-descriptor parity test. Any
new rename, omission, or Token Economy-only field requires a decision-log row
before the test may permit it.

| Treasury `Mandate` | Token Economy `Grant` | Disposition |
|---|---|---|
| `id` | `id` | Same role. |
| `book_id` | `token_type_id` | Renamed: the local economy root is a token type. |
| `budget_id` | `grant_source_id` | Renamed: the authority pool is a grant source. |
| `authorizer` | `authorizer` | Same role. |
| `cap_minor` | `amount_minor` | Renamed: this is an integer token quantity, never currency. |
| `currency` | omitted | Forbidden monetary field; decision recorded above. |
| `allowed_counterparties` | `allowed_catalog_scopes` | Renamed to the local redemption vocabulary. |
| `denied_counterparties` | `denied_catalog_scopes` | Renamed to the local redemption vocabulary. |
| `expires_at` | `expires_at` | Same role. |
| `signature` | omitted | Deferred local-only verification field; decision recorded above. |
| `issued_at` | `issued_at` | Same role. |
| `status` | `status` | Same state shape. |
| `idempotency_key` | `idempotency_key` | Same role. |
| `required_evidence` | `required_evidence` | Same role. |
| `recurrence_seconds` | `recurrence_seconds` | Same role. |
| `next_charge_at` | `next_issue_at` | Renamed: a grant issues; it does not charge. |
| `cancelled_at` | `cancelled_at` | Same role. |
| — | `holder_id` | Local addition: names the subject receiving the grant. |
| — | `rules` | Local addition: persists the closed server-evaluated policy vocabulary. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
