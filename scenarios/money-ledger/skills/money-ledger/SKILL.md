---
name: "money-ledger"
description: "Record an expense, income or transfer and read the money position: books, accounts, immutable journal postings, period statements, and financial goals. Import a CSV of transactions or admit a typed money event; reverse a posting instead of editing it."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["money-ledger", "expense", "income", "transaction", "bookkeeping", "accounting", "budget", "statement", "journal", "csv-import", "finance"]
  requires:
    scenarios: ["money-ledger"]
    commands: ["money-ledger ledger position-show", "money-ledger ingest event-ingest"]
  origin:
    kind: "authored"
---

## Money Ledger

Postings are immutable: correct a mistake with a reversing entry, never an edit.
Financial observations stay UNKNOWN until the operator supplies inputs, so report
an unknown honestly rather than estimating.

| You want | Command |
|---|---|
| Record one expense, income or other money event | `money-ledger ingest event-ingest` |
| Import a bank or card CSV | `money-ledger ingest file-import` (after `ingest adapter-register` for a file adapter) |
| Move money between accounts | `money-ledger ledger journal-transfer` |
| Undo a posting | `money-ledger ledger journal-reverse` |
| Where do I stand | `money-ledger ledger position-show` |
| A period statement | `money-ledger ledger statement-show` |
| See postings | `money-ledger ledger journal-list` / `journal-get` |
| Books and accounts | `ledger books-list`, `books-create`, `accounts-list`, `accounts-create` |
| Goals | `ledger goals-list`, `goal-declare` |
| Which operator inputs are missing | `money-ledger ledger operator-input-status` |

Run `money-ledger <group> <command> --help` for the typed fields; `ingest
operator-import` and `adapter-run` can rehearse before they apply.
