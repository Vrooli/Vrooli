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
