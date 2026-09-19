# Offer Desk runbook

Use `make setup`, `make start`, `make logs`, and `make stop`. Do not launch the API binary directly. The board is expected to remain useful if Money Ledger is stopped; its availability row should explain the missing source.

The SQLite catalog is reconstructible only from deliberate exports/imports, so back up the database using the same verified storage procedure as Money Ledger: stop the scenario for a file copy, include WAL/SHM companions, checksum the copy, open it read-only, and query nodes, triggers, facts, evaluations, and proposals. Keep dated copies on separate storage.

If an evaluation is stuck, inspect the latest evaluation row and the fact's observed time/staleness window. Missing and stale facts are expected unknowns, not false results. If a transition is refused, use the named legal transitions in the response. If promotion is refused, use the operator role or create a proposal for review.

## Verify the catalog reconciles with its sources

`offers catalog-verify --source-path docs/monetization --source-mode operator-supplied --json`
compares the declared source tree against the live graph and exits non-zero on drift.

Read the two fields together. `reconciled=true` alone does **not** mean a count
comparison ran: once the sources are compressed to judgment-only prose, no file
yields a countable record and the verifier reports `comparable=false` with
`not_comparable_reason`. The CLI prints that reason beside the verdict for
exactly this reason — a bare `reconciled=true` would otherwise read as "verified".

Duplicate `(kind, name)` identities are repaired with `offers catalog-merge`,
which is the only operation permitted to delete a node row. It dry-runs by
default; pass `--dry-run=false` to apply.

## Map an offer to its actuals

The board can only report earned-versus-intended for a node that carries a
ledger account. Imported nodes carry none, so the mapping is an explicit step:

```bash
money-ledger ledger accounts-list --book-id <book> --json     # find the account
offer-desk offers catalog-map-account --node-id <node> --account-id <account> \
  --reason "why this node reads this account"
```

An unmapped node is reported honestly — `money-ledger.actuals: no ledger account
mapping` — and is never shown as having earned zero. Clearing a mapping is
allowed (omit `--account-id`) and is audited like any other change.

## When the ledger is unreachable

The board keeps ranking catalog state and names the unavailable source with its
last-good age. It must never render an unavailable actuals read as a zero: an
active offer whose earnings cannot be read says "earnings unknown", while a
confirmed zero says "earning nothing". Those two rows are ranked differently on
purpose.
