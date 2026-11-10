# Money Ledger architecture

Money Ledger is the record side of the monetization pair. It owns books, accounts, immutable postings, adapter receipts, and read-time financial position. Offer Desk may read this scenario; this scenario never imports offer intent.

## Runtime shape

`api/handlers/` is the typed Connect transport. `api/internal/ledger/` owns books, journal, reversals, goals, statements, and the schema. `api/internal/ingest/` owns adapter registration, manual entry, CSV import, receipts, and the only inbound money-event door. Position and statements are computed by the journal store at read time; no balance column exists. The CLI and UI use the same generated proto contracts.

Every posting has an adapter id, external id, occurred and fetched timestamps, currency, signed minor amount, and basis (`authoritative`, `derived`, or `operator-asserted`). A correction is a new reversing posting linked by `reversal_of`; edits and deletes are not capabilities.

## Write boundary

Adapters normalize into `MoneyEvent` values through the ingest store. Ordinary sources use `IngestEvent` or `ImportFile`; the Treasury adapter uses `AcceptCryptoReceipt` to attach exact chain evidence to the same journal writer. Adapter failures create a receipt and an `Availability` value with a reason and last-success timestamp; they do not create a zero-valued posting.

## Treasury receipts

The native USDC on Base adapter accepts only a verified `treasury` service principal from Scenario Authenticator with the `money-ledger:write` capability. The relying-party audience is `scenario-authenticator:money-ledger`. Preview remains an untrusted, read-only interpretation. A service acceptance asserts Treasury's verification; Money Ledger does not independently query chain consensus or infer a stablecoin's fiat value.

Every accepted receipt carries a typed Treasury check/wallet link, exact unsigned token amount, signed token delta, separately declared valuation and valuation delta, and payer address. Customer payments also carry payer identity, credit lot id and lot owner. The `no_chargeback` field records the rail's consumer-chargeback distinction; chain reorgs and accounting corrections remain possible. A credit lot cannot be attributed to two different payments.

| Action | Required accepted predecessor | Token and valuation effect |
|---|---|---|
| Original | None | Declared quantity and valuation |
| Replay, including a new verification check | Existing identical semantic receipt | Return the same receipt and posting |
| Re-inclusion | Current accepted receipt; new block | Zero while active; restore the accepted amount after removal |
| Removal | Current active inclusion | Negate accepted quantity and valuation |
| Correction | Current active inclusion | Difference between the replacement and accepted quantity/valuation |

All revisions name both the root `original_receipt_id` and immediate `previous_receipt_id`; stale predecessors fail. Scope, currency, asset metadata and payer/lot attribution remain fixed. Re-inclusion preserves the accepted quantity and valuation. Corrections retain their reason and declared basis. Ordinary journal reversal refuses token postings so fiat and token evidence cannot diverge.

`crypto_submissions` retains every Treasury verification identity, including deduplicated checks. `crypto_receipts` stores immutable exact-token columns and the complete typed receipt. Source submission, regular `MoneyEvent` posting, receipt and audit share one SQLite transaction. Decimal integer strings preserve uint256 quantities; fiat position still sums only declared signed minor units. `Posting.crypto_receipt_id` resolves to `GetCryptoReceipt`. Startup adds the new tables to existing databases without rewriting earlier postings.

## Read boundary

Position, runway, burn, statements, and goal verdicts are queries over postings. A partial position carries the missing adapter in the serialized response. Consumers must render the basis and caveat beside the figure.

## Startup and storage

The lifecycle starts the API and UI through the scenario Makefile. API startup applies the system and ledger schemas to the routed database. The SQLite file is non-regenerable operational data; see `docs/operations/RUNBOOK.md` for backup and restore.
