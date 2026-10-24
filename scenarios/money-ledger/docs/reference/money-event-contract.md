# The money-event contract

The single inbound shape this scenario admits. Every source is an adapter that
satisfies it; there is no privileged path (`OT-P0-004`). This document is the
public specification — the ledger is its reference implementation, not its
definition.

`business/MONETIZATION.md` records the contract as a packaging candidate on the
grounds that it inverts the integration-count problem: a published shape makes
integration count something other people can contribute to. That argument only
holds if the shape is actually written down, which is what this file is for.

## The shape

| Field | Type | Required | Meaning |
|---|---|---|---|
| `external_id` | string | yes | Stable id **in the source system**. Together with `adapter_id` it is the idempotency key. |
| `adapter_id` | string | yes | Which adapter produced this event. |
| `account_id` | string | yes | The account the amount posts against. |
| `book_id` | string | yes | The accounting entity. An account belongs to exactly one book (`JRNL-001`). |
| `amount_minor` | int64 | yes | **Signed**, in minor units. Never a float. Direction is the sign, not a separate field. |
| `currency` | string | yes | ISO 4217. Never converted (`CUR-001`); currencies are reported separately. |
| `occurred_at` | timestamp | yes | When the money moved. Not when it was fetched. |
| `fetched_at` | timestamp | yes | When the adapter read it. The pair `(occurred_at, fetched_at)` is what makes staleness computable. |
| `basis` | enum | yes | `authoritative` \| `derived` \| `operator_asserted`. See below. |
| `description` | string | no | Free text for a human. |
| `category` | string | no | Assigned by rule or operator; a rule may assign a category but never an amount (`CAT-001`). |
| `reversal_of` | string | no | The posting this one reverses. Corrections are new events, never edits. |

## The three rules that make it a contract rather than a struct

**1. `basis` is mandatory and has exactly three values.** `authoritative` means
the source system is the system of record for this figure. `derived` means it
was computed from something else. `operator_asserted` means a human typed it.
A consumer that cannot tell these apart cannot judge the figure, which is the
failure this whole scenario exists to prevent. There is no fourth value and no
default — an adapter that cannot state a basis is not conformant.

**2. Ingestion is idempotent on `(adapter_id, external_id)`.** Re-running an
adapter over an overlapping window must produce no duplicate postings
(`OT-P0-007`). This is a database constraint, not adapter etiquette.

**3. An adapter that cannot run reports unavailable with a reason and an age.
It never reports zero.** A failed read and a real zero are different facts and
must stay different at every layer above (`POS-004`). This is the single
requirement most likely to be violated by a well-meaning implementation.

## Conformance

An implementation is conformant when all four hold:

1. Every emitted event carries all required fields above, with a basis drawn
   from the three declared values.
2. Re-emitting the same `(adapter_id, external_id)` over an overlapping window
   adds no posting.
3. A failure mode — no credentials, upstream down, malformed response —
   produces an availability record with a reason and a last-success age, and no
   postings.
4. The adapter emits events and nothing else. Adapters may not write balances,
   positions, or derived rates (`CTR-005`). A derived rate such as MRR is
   refused at the door rather than stored as a posting.

The conformance drills live in `api/internal/ingest/store_test.go`; the manual
and file adapters are the reference conformant implementations, and they are
ordinary adapters rather than a degraded fallback (`OT-P0-006`).

## Status

The contract is `candidate` as a packaged asset, and the bar for promoting it is
stated in `../business/MONETIZATION.md`: **a contract with one implementation is
a file, not a standard.** It stays candidate until at least one adapter exists
that we did not write. Publishing this specification does not change that
status; it is the prerequisite, not the evidence.
