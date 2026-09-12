# Money Ledger runbook

## Start, stop, and inspect

Use `make setup`, `make start`, `make logs`, and `make stop` from this scenario directory. Do not launch the API binary directly. Health is exposed through the scenario lifecycle and API health endpoint.

## Backup the journal

The SQLite database is non-regenerable. Resolve the active scenario data path with the storage resolver used by `api/main.go`; do not guess from a home-directory path. While the API is stopped, copy the database and its WAL/SHM companions to a restricted backup directory, then record the UTC timestamp and SHA-256 checksum. Keep at least three dated copies on separate storage.

For a live backup, use SQLite's backup-capable tooling against the database connection rather than copying only the main file. Verify the backup by opening it read-only and querying `books`, `accounts`, `postings`, and `ingest_receipts`.

## Restore

Stop the scenario, preserve the damaged database as a dated quarantine copy, and restore the selected verified backup plus its checksum record. Start with `make start`, run the health check, query the latest receipt and posting count, and run `make test`. A restore is not complete until the journal count and latest posting ids match the backup verification record.

## Adapter outage

Read adapters and receipts first. A failed adapter should be shown as unavailable with a reason and age. Do not repair an outage by inserting a zero or editing postings. After the source is healthy, rerun its window; idempotency makes overlap safe.
