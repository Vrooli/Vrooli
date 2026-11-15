# Money Ledger CLI reference

The CLI is the operator-friendly mirror of the generated Connect contract.
Every command supports `--json`; human output keeps basis, currency, source,
age, partiality, and reversal links visible. The machine-readable source of
truth is [`../../cli/manifest.json`](../../cli/manifest.json).

## Writing as the owner

Money Ledger refuses every write without a verified, authorised identity (see
[`../internal/SECURITY.md`](../internal/SECURITY.md)). Every write command
therefore takes `--local-owner`: it exchanges this machine's own credential for
the operator's Scenario Authenticator session through the local authenticator
socket and presents it for that one request. Without the flag a write goes out
with no identity and is refused with a sentence saying what to do. The flag is
unavailable inside an agent run: an agent that needs a record changed asks the
operator.

`--local-owner` admits the operator only when the host's owner-binding policy
names the machine-linked account as the owner. Check what the owner session may
do, or why not, before writing:

```bash
money-ledger ledger write-access --local-owner
```

Without the flag, `write-access` reports what an anonymous caller may do.

## `ledger`

| Command | Effect | Purpose | Important flags |
|---|---|---|---|
| `write-access` | read | Whether this caller may write, as whom, and why not. | `--local-owner` |
| `books-list` | read | List books and currencies. | `--include-archived` |
| `books-create` | write | Create a currency-scoped book. | `--name`, `--currency`, `--local-owner` |
| `books-archive` | write | Archive a book; its journal is preserved. | `--book-id`, `--local-owner` |
| `accounts-list` | read | List accounts in a book. | `--book-id` |
| `accounts-create` | write | Create an account. | `--book-id`, `--name`, `--kind`, `--local-owner` |
| `journal-list` / `journal-get` | read | Read postings, or one posting and its audit trail. | `--book-id`, `--account-id`, `--posting-id`, `--limit` |
| `journal-reverse` | write | Append a linked reversal; never edits the original. | `--posting-id`, `--reason`, `--local-owner` |
| `journal-transfer` | write | Append paired transfer postings. | `--from-account-id`, `--to-account-id`, `--amount-minor`, `--currency`, `--external-id`, `--description`, `--local-owner` |
| `position-show` | read | Compute position at read time. | `--book-id` |
| `statement-show` | read | Read a period statement. | `--book-id`, `--from`, `--to` |
| `goals-list` | read | List goal thresholds, progress and verdicts. | `--book-id`, `--include-archived` |
| `goal-declare` | write | Declare a sustained goal. | `--book-id`, `--name`, `--metric`, `--comparator`, `--threshold-minor` or `--threshold-ratio`, `--sustain-periods`, `--sustain-period-unit`, `--local-owner` |
| `goals-archive` | write | Archive a goal without deleting it. | `--goal-id`, `--local-owner` |
| `goal-reparent` | write | Move a goal to another active book. | `--goal-id`, `--book-id`, `--local-owner` |
| `operator-input-status` | read | The thirteen operator input fields with status and age. | `--book-id` |

## `ingest`

| Command | Effect | Purpose | Important flags |
|---|---|---|---|
| `adapters-list` | read | Adapter availability and receipt age. | — |
| `adapter-register` | write | Register a manual or file adapter. | `--id`, `--name`, `--kind`, `--local-owner` |
| `event-ingest` | write | Admit one typed event idempotently. | `--external-id`, `--adapter-id`, `--account-id`, `--book-id`, `--amount-minor`, `--currency`, `--basis`, `--local-owner` |
| `adapter-run` | write | Run an adapter over its configured source. | `--adapter-id`, `--local-owner` |
| `file-import` | write | Import CSV through a file adapter. | `--adapter-id`, `--csv`, `--local-owner` |
| `operator-import` | write | Rehearse or apply typed operator inputs. A rehearsal still needs the owner because it reads a host file. | `--source-path`, `--source-mode`, `--adapter-id`, `--book-id`, `--account-id`, `--apply`, `--local-owner` |
| `crypto-preview` | read | Preview a crypto observation's effects without posting. | `--observation` |

Pending operator fields are absent, never zero; non-monetary quantities are
measures, not postings; derived rates such as MRR are refused as postings.
When a source fails, the command reports the reason and `partial` result rather
than inventing a balance.

## Lifecycle

Use `make start`, `make test`, `make logs`, and `make stop` for scenario
lifecycle. Use `vrooli scenario test money-ledger` for the comprehensive suite;
the run is server-owned.
