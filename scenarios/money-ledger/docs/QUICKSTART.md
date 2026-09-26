# Quickstart — Money Ledger

This is the first useful path for a new operator: start the scenario, create a
book/account, record one typed event, and inspect the resulting position.

## Start

From the scenario directory:

```bash
make setup
make start
make status
```

Open the reported UI port, or use `make open`. The lifecycle owns ports and
health checks; do not run the API or UI binaries directly.

## Record the first event through the console

1. Open **Accounts** and create a book with an ISO currency such as `USD`.
2. Create a cash account in that book.
3. Open **Journal**, complete Date, Account, Signed amount (minor units),
   Currency, Description, External event ID, and Basis, then choose **Record
   event**.
4. Return to **Dashboard**. The position/runway and goal surfaces show the
   read-time result with currency, basis, source, and completeness.

The journal is append-only. To correct the event, use its **Reverse posting**
action and supply a reason; the original remains visible and linked.

## Equivalent CLI path

```bash
money-ledger ledger books-create --name Operating --currency USD --json
money-ledger ledger accounts-create --book-id <book-id> --name Cash --kind cash --json
money-ledger ingest adapter-register --id manual --name Manual --kind manual --json
money-ledger ingest event-ingest --external-id first-event --adapter-id manual \
  --account-id <account-id> --book-id <book-id> --amount-minor 125000 \
  --currency USD --basis operator-asserted --description "First recorded event" --json
money-ledger ledger position-show --book-id <book-id> --json
```

Repeat ingestion with the same external ID to receive a duplicate receipt,
not a second posting. If an adapter is unavailable, the CLI and console show a
named gap and `partial=true`; they never replace the missing value with zero.

## Lifecycle and tests

```bash
make logs
make test
make stop
```

For the comprehensive server-owned suite, use `vrooli scenario test
money-ledger` and wait once on the returned run ID. See
[`reference/cli-commands.md`](reference/cli-commands.md) and
[`concepts/UI-ARCHITECTURE.md`](concepts/UI-ARCHITECTURE.md) for the complete
surface contract.
