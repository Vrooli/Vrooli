# PS2 checkout and Money Ledger proposal

Status: **running test-only prototype; cross-owner technical review remains**.
This increment is a local journey harness with a sandbox-provider fake, the
LPBS service-order store, and the real Money Ledger SQLite journey fixture. It
does not authorize promotion, publication, provider calls, live payment,
account creation, or legal-rights acceptance.

## Outcome under review

Extend the existing `company-market-scan` offer prototype into one browser
sandbox order whose identity is stable across the order, delivery job, provider
receipts, and Money Ledger event. The buyer first authorizes. A governed
delivery verifier records success before capture is requested. A successful
sandbox capture is reconciled exactly once into an isolated Money Ledger
journey database. Failed verification voids the authorization. Unknown capture
or ledger outcomes remain pending and retry under the original idempotency
identities.

This is the first vertical slice toward PS2's four-door showcase. It does not
complete the target of four sandbox purchases from browser, MCP, MPP/x402, and
Agent Manager, or four reconciled jobs.

## Owners and paths

| Boundary | Proposed owner | Existing path and responsibility |
|---|---|---|
| Offer and fixed price snapshot | Offer Desk | `api/internal/serviceoffer/offer.go`; one `Record` projects page and agent-door descriptions. The existing record remains staged, rights-open, and operator-pending. |
| Order/job identity, provider receipt idempotency, verification gate, capture outbox | LPBS `commerce` | `api/internal/commerce/service_order.go` and service-order tables in `api/internal/commerce/operations_schema.sql`; separate from existing Stripe subscription, credit-top-up, and contribution sessions. Not wired to the production API. |
| Accepted payment facts | Provider adapter owned with LPBS checkout | `ServiceOrderSandboxProvider` is provider-neutral. Its only implementation in this increment is a test fake whose receipts explicitly identify sandbox mode. No Stripe adapter or external provider call. |
| Journal posting and duplicate detection | Money Ledger | `api/handlers/ingest/module.go` and `api/internal/ingest/store.go`; use `IngestService.IngestEvent`, with the stable `(adapter_id, external_id)` identity. Posting corrections are new reversal entries. |
| Isolated journey fixture | Money Ledger owns fixture; LPBS owns its orchestration in tests | `scenarios/money-ledger/api/journey_server_test.go::TestJourneyFundingLedgerServer` starts actual Money Ledger handlers over SQLite at `VROOLI_JOURNEY_LEDGER_DB`, creates a test-only revenue account, and exposes a path-derived fixture identity on the loopback test listener. LPBS `journeytest/startFundingLedger` supplies a fresh `t.TempDir()` database and stops it. |

Offer Desk is not proposed as a payment processor or an accounting writer. It
owns neither provider credentials nor Money Ledger's database. LPBS must not
write Money Ledger tables directly; it calls the typed ingestion API and
persists its own retryable outbox state.

## Contract and state transitions

The Offer Desk contract draft is in
[`api/internal/serviceoffer/checkout_contract.go`](../../api/internal/serviceoffer/checkout_contract.go).
The LPBS service contract pins order ID, job ID, offer ID/version, purchase
door, sandbox mode, exact minor-unit price, and currency before authorization.
The four accepted test-door labels follow the showcase journeys: `browser`
(person on a phone), `browsing_agent` (same human-facing checkout used by an
agent), `mcp`, and `agent_manager_x402` (Agent Manager using the local x402
fork). The x402 value is a payment rail, not a fifth buyer door. Unknown or
caller-invented door values are rejected before persistence. The labels do not
assert that the corresponding network/client integration is implemented.
Transition event IDs are durable and payload-hashed; an identical replay is a
no-op and a reused ID with a changed payload is rejected.

| Required journey | Door value | Current evidence | Still required |
|---|---|---|---|
| Person on phone | `browser` | Local Playwright order through the test harness and disposable Money Ledger fixture | Maintained responsive LPBS guest checkout and a reconciled order |
| Browsing agent | `browsing_agent` | Door identity is accepted by the contract; no agent has exercised checkout | Human-facing site route, payer-authorized order contract, consented agent run, and one reconciled order |
| MCP client | `mcp` | Offer projects an MCP tool descriptor only | Real MCP client call through the maintained checkout and one reconciled order |
| Agent Manager over x402 fork | `agent_manager_x402` | No Agent Manager call is part of this prototype | Treasury-owned x402 test route/fork, Agent Manager run, and one reconciled order |

All four journeys must use four durable order/job IDs and result in four
Money Ledger receipts in the isolated test fixture. The raw MPP/x402 route and
Stripe machine-payment surface remain separate implementation tasks; this
table does not claim that either is delivered by the current sandbox
interfaces.

### Maintained MCP implementation dependency review

The current `mcp` door value is only an enum/label. The maintained checkout does
not expose an MCP endpoint or accept an MCP client call. For a Go-owned LPBS
endpoint, recommend the official Go SDK at `github.com/modelcontextprotocol/go-sdk`
version `v1.8.0`: its upstream release notes say this release hardens transport
resource limits, session cleanup, deadlocks and teardown, while retaining support
for the latest `2026-07-28` protocol. The SDK module requires Go 1.25; LPBS API
declares Go 1.26, so the module language baseline is compatible. The official
repository describes the SDK as maintained with Google. [SDK repository](https://github.com/modelcontextprotocol/go-sdk),
[v1.8.0 release notes](https://github.com/modelcontextprotocol/go-sdk/releases),
[v1.8.0 module requirements](https://raw.githubusercontent.com/modelcontextprotocol/go-sdk/v1.8.0/go.mod).

The upstream license file contains Apache-2.0 terms and states that contributions
without relicensing consent remain MIT-licensed. This is a mixed licensing notice
that needs the operator's legal/rights acceptance before dependency approval or
installation. No license was accepted here. No Go module was added and no package
manager ran. Scenario Dependency Analyzer's dry-run for
`go/github.com/modelcontextprotocol/go-sdk@v1.8.0` in LPBS `api` returned
`unrecorded`, `blocked=true`, `installed=false`; its suggestion to use
`approve-observed` is not applicable yet because the module is not observed in
`go.mod` or source.

**Exact next route after operator license acceptance:** record the package through
the SDA governance owner with exact version `v1.8.0`, `range-policy=exact`, and
LPBS `api` scope; then run the same SDA `deps install` with `--apply` and retain
the dependency resolution/security receipt. Only after those gates should LPBS
implement a maintained MCP endpoint and client journey. The endpoint must load
the same Offer Desk record/version and call LPBS's stable order identity,
authorize-then-verify-then-capture service; Money Ledger remains the sole writer
of its journal. Any owner review should also name the human checkout payer
authorization contract. The contract's current `mcp` door identity alone proves
none of these boundaries.

Proposed successful path:

```text
created -> authorized -> verified -> capture_pending
                                   capture_pending -> ledger_pending -> reconciled
```

Proposed failure paths:

```text
created -> declined
authorized -> verification_failed -> voided
verified -> capture_pending (same provider key on retry)
capture_pending -> ledger_pending (sandbox receipt and durable outbox)
ledger_pending -> reconciled (typed Money Ledger receipt)
```

The service rejects capture before `verified` and reconciliation before a
durable capture receipt. Capture receipt and ledger outbox are recorded in one
LPBS transaction. A refund or correction after capture belongs
to a separate provider-refund and Money Ledger reversal contract; it is not
modeled as undoing a capture. Delivery verification itself must be supplied by
the governed delivery owner before LPBS adds a capture request.

The provider interface uses `order_id` and a stable idempotency key for
authorize, capture, and void; receipts include stable provider event/receipt
IDs. The Money Ledger event should use a stable external ID derived from the
order and capture (for example `service-order:<order-id>:capture:<capture-id>`)
and a sandbox-specific adapter and category. LPBS outbox replay reuses the
identical event; Money Ledger's `(adapter_id, external_id)` uniqueness makes a
lost response safe to retry and rejects a changed amount under the same key.

## Isolated Money Ledger fixture

The existing fixture is real Money Ledger code and a real SQLite database, not
a fake ledger service. Its parent test owns the file path under `t.TempDir()`;
the child test opens that database, applies Money Ledger schemas/migrations,
creates a book, accounts, and adapter, mounts the real ledger and ingest
modules, and serves only on an `httptest` listener. The fixture lifetime ends
when the test sends `POST /__journey/stop` and joins the subprocess.

The fixture's `TEST ONLY isolated funding` book retains `loss`, `refund`, and
`fee` expense accounts and adds a `TEST ONLY service-order sandbox` revenue
account. Its identity endpoint returns the fixture ID derived from the
temporary SQLite path. LPBS refuses non-loopback endpoints and endpoints whose
identity does not match. The writer calls typed ingest; LPBS never opens or
writes the Money Ledger SQLite file directly.

## Persistence, rollback, and replay

1. LPBS creates the durable order and job ID before calling the sandbox
   provider. `(door, offer_id, offer_version, amount_minor, currency)` are
   immutable on retries; a changed payload with the same client idempotency key
   is a conflict.
2. Provider receipts are stored by stable provider event ID. Rebinding the ID
   to a different order, event kind, or payload digest is rejected. This
   increment exercises synchronous receipts; it does not implement a provider
   webhook endpoint or callback inbox.
3. Verification failure does not capture. LPBS requests a void with the same
   order and authorization identity; a lost void response remains pending
   until the provider confirms it.
4. Successful sandbox capture creates a durable ledger outbox event. A failed or
   uncertain Money Ledger call leaves the outbox row pending. Retry with the
   exact same MoneyEvent; do not mark delivered until the typed response
   contains the accepted posting and duplicate status.
5. A mistaken accepted posting is corrected only with Money Ledger's
   `JournalService.ReversePosting`; neither service edits/deletes the posting.
   The isolated journey database is discarded after test cleanup, not rolled
   back against the live ledger.

## Acceptance evidence for this increment

Offer Desk contract tests cover stable offer/order identity,
verification-before-capture, receipt replay, payload conflict, and
serialization. LPBS `journeytest/service_order_test.go` adds a runnable
Playwright browser harness over the durable LPBS order/job identity and the
actual Money Ledger SQLite journey fixture. It reads the staged
`company-market-scan` ID/version/price from its test record, authorizes via a
test fake, accepts a successful test verifier receipt, requests one simulated
capture, and reconciles one typed journal posting. The provider and verifier
are test fakes; this is not the Offer Desk page, a maintained LPBS checkout
route, or the production delivery verifier.

The focused service-order suite proves:

- begin retry reuses durable order/job IDs and calls authorization once;
- browser retry after a simulated lost order response reuses its original key;
  rapid second clicks are ignored while the submission is in flight. The
  browser harness now runs at a 390×844 mobile viewport with touch enabled,
  verifies the order control is visible without horizontal page overflow, and
  completes the same isolated ledger journey;
- changed quote terms with the same idempotency key conflict;
- verification refusal voids authorization and makes zero capture calls;
- capture refusal leaves the order capture-pending and reuses the same provider
  idempotency key;
- the browser flow creates one test-only posting and exact MoneyEvent replay
  returns the original posting as a duplicate;
- the fixture uses a fresh temporary SQLite path, and the writer rejects
  non-loopback endpoints and mismatched fixture identities.

Run evidence is in the research checkpoint for
`steward-channel-agent-commerce-20261113-2`. The LPBS journey uses a disposable
PostgreSQL test container plus the Money Ledger SQLite subprocess. Browser
execution uses the installed local Chrome binary. Tests stop containers and
fixture subprocesses during cleanup.

## Scope and boundaries

In scope for this increment: one browser-door sandbox journey harness; typed
provider-neutral contract; an LPBS durable order row with stable job ID; the
test-only revenue account in a temporary SQLite fixture; and focused tests.
Later work must wire the Offer Desk page to a maintained LPBS route and
exercise the other three doors.

Out of scope: Stripe calls or test charges, live-mode support, real payments,
external accounts, customer/prospect contact or data, publication/listing,
rights acceptance, n8n, production Money Ledger writes, and treating a
simulated sandbox receipt or isolated posting as revenue. The sample offer's
legal and operator gates remain open/pending.

## Review questions for the owners

1. LPBS owner review: confirm `internal/commerce/service_order.go` is the
   right durable order/job owner, that these `OperationsSchema` tables belong
   in that owner, and that an API route should be added only after the sandbox
   boundary is accepted.
2. Delivery owner: identify the maintained verifier that can supply a trusted,
   durable success receipt. The current journey uses a fixed test verifier and
   does not prove report delivery or citation validation.
3. Money Ledger owner: review the test-only revenue account and
   `/__journey/identity` addition to `TestJourneyFundingLedgerServer`; confirm
   the path-derived fixture ID is sufficient evidence for the loopback typed
   writer.
4. Money Ledger owner: confirm the derived basis,
   `test.service_order.sandbox` category, test-only description, and
   `IngestEvent` duplicate response used by this isolated fixture.
5. Offer Desk owner: confirm the offer version and proposed price snapshot are
   stable inputs LPBS should consume without consulting this preview on retry.

These are owner-boundary decisions. This proposal is ready for review as one
item. No LPBS production API route, Money Ledger production API, or real
provider integration was changed; the Money Ledger edit is limited to its
isolated journey fixture test.

## PS2 continuation · four-door acceptance and x402 ordering · 2026-11-14

The acceptance gate for PS2 is the full four-door outcome, not expansion from the current slice:

- Start from one validated, versioned Offer Desk record and derive the human page, guest-checkout quote, MCP tool descriptor, MPP/x402 price declaration, and OpenAPI/`llms.txt` projections from that same record. Every purchase freezes the same offer ID, version, price and currency before provider work.
- Complete four independent test-mode orders through the actual surfaces: a person on a phone, a consented browsing agent using the same human checkout, a maintained MCP endpoint exercised by an MCP client, and an Agent Manager run that pays via Treasury on the local x402 fork.
- Each order receives its own durable order ID and job ID. Each door proves an identical request replay returns the original identities and a changed payload under that idempotency identity conflicts. All four delivery jobs must verify before capture; all four must reconcile to exactly one accepted Money Ledger receipt linked back to its job.
- The Money Ledger posting owner must be explicit per rail. LPBS owns the browser, browsing-agent and MCP sale postings. For x402, Treasury owns the seller-side inbound posting; LPBS stores the Treasury admission and links the job to Treasury's accepted Money Ledger posting ID. LPBS must not post a second seller-side entry for that capture. An Agent Manager buyer-side spend, if recorded, is a distinct expense in its own account/book and is not the seller's order receipt.

### Treasury x402 extension required for this promise

The current inbound route does not yet meet the delivery-verification promise: Treasury `Gate.Admit` calls facilitator `Verify`, then `Settle`, and only then returns the admission to the resource handler. The local CF4 fork fixture likewise submits `transferWithAuthorization` while processing the paid retry. That is settlement-before-resource; a later LPBS verification failure could not void it.

The official x402 v2 `exact` scheme supports an `authorization` flow (verify → resource → settle) where the transfer method permits it, and recommends that flow to leave the payer uncharged when the handler fails. For this PS2 service, the maintained composition must therefore persist a unique LPBS order/job identity, have Treasury verify the signed authorization without settling, run delivery and record its verifier receipt, then settle only after success. Unknown settlement remains recoverable by replay under the same payment/order/job identities. Verification failure leaves the authorization un-settled. This must be proved using the local Base fork in test mode; no mainnet or live payment is in scope.

Money Ledger currently receives a Treasury-owned `ledger_emissions` row for each settled inbound x402 admission. LPBS's existing service-order reconciler also emits a typed Money Ledger event. For x402 sales, Treasury should remain the one seller-side posting writer and LPBS should link the accepted posting to its order/job; it must not create a second seller-side entry for the same sale. Treasury's `MoneyLedgerEmitter` currently receives a typed ingest response but discards the posting ID, `ledger_emissions` stores only accepted status/time, and `ListEmissions` is operator-authenticated. The proposal therefore includes a typed, service-authenticated receipt lookup (or equivalent owner boundary) that returns the accepted posting ID to LPBS. The order/job linkage and one-entry invariant need a direct test against the isolated Money Ledger fixture. Browser, browsing-agent and MCP orders retain LPBS's test-only Money Ledger writer. Agent Manager's buyer expense, if separately recorded, uses its buyer account/book and is not counted as the seller-side order receipt.

Treasury's receipt read route is `GET /api/v1/x402/receipts?external_id=treasury-x402-inflow%3A<tx>`. It exposes `external_id`, `posting_id`, Money Ledger `receipt_id`, `accepted_at`, `amount_minor`, `currency`, and `basis` only for one uniquely accepted outbox row; a missing, ambiguous or unaccepted row is not represented as success. The route requires a service principal with audience `scenario-authenticator:treasury-x402-receipt` and scope `treasury:x402-receipt-read`. LPBS declares it in `scenarios/landing-page-business-suite/.vrooli/service-principals.json`; its read client/token source is implemented, wired into test-only service-order reconciliation and tested against local HTTP/Money Ledger fixtures. No credential has been issued and no sync apply was run.

The cross-owner deferred-settlement API is implemented as `POST /api/v1/x402/authorizations/verify` followed by `POST /api/v1/x402/authorizations/{admission_id}/settle`, with every call bound to the LPBS `job_id`, `resource_id` and `order_id`. Verify accepts the signed payment and returns API-level `authorized` with a server-derived `mode`; LPBS fails closed unless this is `local-fork`. Settle does not resend the signed payload: Treasury retains it through the protected Credential Authority recovery store and persists only a logical recovery reference with the durable admission, then resolves and digest-checks it during settlement/recovery. The payload is not stored in LPBS or plain Treasury SQLite; terminal settlement cleanup is retryable. On failed LPBS verification, `POST /api/v1/x402/authorizations/{admission_id}/cancel` terminally cancels only the matching still-verified authorization, clears its protected payload and never calls the facilitator. Cancellation refuses any admission whose settlement is in progress or unknown, leaving that case for same-identity settlement recovery. Treasury remains the only seller-side Money Ledger writer and receipt readback is a separate, read-only service identity. Treasury's owner has implemented the handler/gate/storage routes; LPBS has a test-only adapter and deferred-capture/cancel/reconcile tests against local stubs. The focused Treasury Gate/handler tests also pass. Neither the actual local-fork composition nor Agent Manager purchase/receipt path has been run.

The x402 order identity is `landing-page-business-suite-treasury-x402-order` with audience `scenario-authenticator:treasury-x402-order` and capabilities `treasury:x402-authorize`, `treasury:x402-settle` and `treasury:x402-cancel`. The cancellation capability clears protected authorization input after a failed LPBS verifier without settlement. The receipt-reader identity remains separate and read-only. LPBS also declares `landing-page-business-suite-money-ledger-writer` on `scenario-authenticator:money-ledger` with only `money-ledger:write` for browser/agent/MCP test captures; x402 seller receipts continue to be written only by Treasury. No credential was issued. The refreshed complete global sync preview lists six audience registrations, 33 principal additions, 33 missing credentials, and 35 credential entries total (two already issued), a rewrite of two Scenario Authenticator profile settings, and one restart. The supervisor's attention item records this exact preview and retains the recommendation to hold because unrelated fleet-wide registrations and credentials are part of the apply. No auth profile was manually changed.

This is a maintained Treasury/LPBS extension plus a Money Ledger receipt-link contract. Treasury route/storage code is implemented and locally unit-tested; LPBS client linkage is exercised with a Treasury HTTP stub and isolated Money Ledger fixture. The real Treasury service, local Base fork, Agent Manager runner and accepted receipt in one composed purchase remain unverified; this does not claim that PS2's x402 door meets its showcase promise.

### Updated local evidence

The existing one-door browser slice was rerun on 2026-11-14:

`cd scenarios/landing-page-business-suite/api && go test ./journeytest -run '^TestServiceOrderSandboxBrowserJourney$' -count=1 -v` — PASS in 8.71 s. It used a temporary Postgres container, the real Money Ledger test SQLite server and local Chrome, with a fake sandbox provider and fixed verifier. The fixture cleaned up its Postgres container. This remains a harness-only browser journey; it is not a maintained checkout route or four-door proof.

The current Treasury outbound rail baseline was also rerun:

`cd scenarios/treasury/api && TREASURY_BASE_FORK_TEST=1 go test ./internal/rail/x402 -run '^TestCF4LocalBaseForkReconcilesExactPayment$' -count=1 -v` — PASS in 1.28 s. It used credential-free public Base RPC for a finalized block read and a disposable local Anvil fork; local EIP-3009 settlement and ambiguous-receipt recovery passed. This is Treasury rail evidence only: it is not a normal Agent Manager run, an inbound resource that defers settlement until verification, an LPBS service order/job, or a PS2 Money Ledger receipt.

LPBS also has service-contract journey coverage across all four recognized door identities:

`cd scenarios/landing-page-business-suite/api && go test ./journeytest -run '^TestServiceOrderSandboxFourDoorContractUsesOneOfferAndFourReceipts$' -count=1 -v` — PASS in 9.61 s on 2026-11-14. The test reads the same `company-market-scan` v1 offer snapshot for `browser`, `browsing_agent`, `mcp` and `agent_manager_x402`; for every door it verifies exact begin replay preserves order/job IDs, changed job input conflicts under the same key, verification precedes capture, terminal retries preserve capture/posting IDs, and the real isolated Money Ledger fixture contains exactly four unique test postings. This is order-domain/ledger contract coverage, not a real browsing-agent, MCP-client, or Agent Manager integration; those surfaces remain acceptance gaps.

The same checkpoint reran these LPBS journeys with the x402 consumer wired to the real client/provider adapters and a local fake Treasury HTTP server:

`cd scenarios/landing-page-business-suite/api && go test ./journeytest -run '^(TestServiceOrderSandboxBrowserJourney|TestServiceOrderSandboxFourDoorContractUsesOneOfferAndFourReceipts|TestServiceOrderTreasuryX402LinksOneAcceptedSellerPostingAfterVerification)$' -count=1 -v` — PASS in 37.812 s. It covers the mobile-width browser harness, four test order-domain identities/receipts, and a deferred x402 path that waits for LPBS verification before calling the fake Treasury settle route and links its accepted seller posting. This is still a composed stubbed journey: the Treasury process, its protected recovery store, local Base fork facilitator and Agent Manager runner are not exercised. `go test ./internal/commerce -run '^TestTreasuryX402' -count=1 -v` also passed the LPBS service-token, HTTP client and receipt-reader tests after aligning the adapter with Treasury's current API contract.

The failed-verifier branch now also simulates one temporary cancellation/secret-cleanup failure and retries the same durable admission: `cd scenarios/landing-page-business-suite/api && go test ./journeytest -run '^TestServiceOrderTreasuryX402LinksOneAcceptedSellerPostingAfterVerification$' -count=1 -v` — PASS in 7.02 s. It ends in `voided` only after Treasury confirms cancellation, with no second settlement or seller posting. `go test ./internal/commerce -run '^TestTreasuryX402' -count=1 -v` passes the LPBS contract tests for exact settlement binding and `202 unknown`, `settling` and `recovering` responses remaining retryable. These fakes prove LPBS behavior only; actual Treasury protected-store recovery and the local fork remain unverified in this composed path.

The official protocol basis is [x402 v2 exact scheme](https://github.com/x402-foundation/x402/blob/main/specs/schemes/exact/scheme_exact.md) and [x402 v2 specification](https://github.com/x402-foundation/x402/blob/main/specs/x402-specification-v2.md). In particular, the Base-fork proof must demonstrate the authorization flow rather than calling a settlement-first `admit` endpoint.
