# Money Ledger CLI reference

The CLI is the operator-friendly mirror of the generated Connect contract.
Every command supports `--json`; human output keeps basis, currency, source,
age, partiality, and reversal links visible.

## `ledger`

| Command | Purpose | Important flags |
|---|---|---|
| `books-list` / `books-create` | Read or create books. | `--name`, `--currency` |
| `accounts-list` / `accounts-create` | Read or create accounts. | `--book-id`, `--name`, `--kind` |
| `journal-list` / `journal-get` | Read postings and one audit trail. | `--book-id`, `--account-id`, `--posting-id`, `--limit` |
| `journal-reverse` | Append a linked reversal. | `--posting-id`, `--reason` |
| `journal-transfer` | Append paired transfer postings. | `--from-account-id`, `--to-account-id`, `--amount-minor`, `--currency`, `--external-id` |
| `position-show` | Compute position at read time. | `--book-id` |
| `statement-show` | Read a period statement. | `--book-id`, `--from`, `--to` |
| `goals-list` / `goal-declare` | Read or declare sustained goals. | `--book-id`, `--name`, `--metric`, `--comparator`, `--sustain-periods`, `--sustain-period-unit` |

## `ingest`

| Command | Purpose | Important flags |
|---|---|---|
| `adapters-list` | Read adapter availability and receipt age. | — |
| `adapter-register` | Register a manual or file adapter. | `--id`, `--name`, `--kind` |
| `event-ingest` | Admit one typed event idempotently. | `--external-id`, `--adapter-id`, `--account-id`, `--book-id`, `--amount-minor`, `--currency`, `--basis` |
| `adapter-run` | Run an adapter and retain its result. | `--adapter-id` |
| `file-import` | Import CSV through a file adapter. | `--adapter-id`, `--csv` |
| `operator-import` | Dry-run or apply typed operator inputs. | `--source-path`, `--source-mode`, `--adapter-id`, `--book-id`, `--account-id`, `--apply` |

Pending operator fields are absent, never zero; non-monetary quantities are
measures, not postings; derived rates such as MRR are refused as postings.
When a source fails, the command reports the reason and `partial` result rather
than inventing a balance.

## Lifecycle

Use `make start`, `make test`, `make logs`, and `make stop` for scenario
lifecycle. Use `vrooli scenario test money-ledger` for the comprehensive suite;
the run is server-owned.
