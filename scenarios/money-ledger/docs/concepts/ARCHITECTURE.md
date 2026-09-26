# Money Ledger architecture

Money Ledger is the record side of the monetization pair. It owns books, accounts, immutable postings, adapter receipts, and read-time financial position. Offer Desk may read this scenario; this scenario never imports offer intent.

## Runtime shape

`api/handlers/` is the typed Connect transport. `api/internal/ledger/` owns books, journal, reversals, goals, statements, and the schema. `api/internal/ingest/` owns adapter registration, manual entry, CSV import, receipts, and the only inbound money-event door. Position and statements are computed by the journal store at read time; no balance column exists. The CLI and UI use the same generated proto contracts.

Every posting has an adapter id, external id, occurred and fetched timestamps, currency, signed minor amount, and basis (`authoritative`, `derived`, or `operator-asserted`). A correction is a new reversing posting linked by `reversal_of`; edits and deletes are not capabilities.

## Write boundary

Adapters emit `MoneyEvent` values to `IngestService.IngestEvent` or `ImportFile`. Only the ingest store calls the journal admission method. Adapter failures create a receipt and an `Availability` value with a reason and last-success timestamp; they do not create a zero-valued posting.

## Read boundary

Position, runway, burn, statements, and goal verdicts are queries over postings. A partial position carries the missing adapter in the serialized response. Consumers must render the basis and caveat beside the figure.

## Startup and storage

The lifecycle starts the API and UI through the scenario Makefile. API startup applies the system and ledger schemas to the routed database. The SQLite file is non-regenerable operational data; see `docs/operations/RUNBOOK.md` for backup and restore.
