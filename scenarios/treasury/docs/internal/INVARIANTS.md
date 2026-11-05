# Invariants — Treasury

These are cross-domain rules whose violation would change Treasury's security,
financial-integrity, or custody posture. Each rule is named at its enforcement
site and exercised through the violating path, not only through a handler.

## onlyOperatorBeneficiaryCanBeRepresented

**Statement.** Treasury can represent any number of books, budgets, mandates,
and instruments for one operator beneficiary, but it cannot represent a second
beneficiary or attach a mandate or instrument to a different book than its
parent authority. A third-party balance therefore has no persistence shape.

**Mechanism.** `db-constraint`: the beneficiary registry is a singleton guarded
by a `CHECK`; books reference that singleton; budgets reference books; and
mandate/instrument insert and update triggers require their parent authority to
belong to the same book. Subordinate tables contain no beneficiary column.

**Enforcement anchors.** `path:api/internal/book/schema.sql:1`,
`path:api/internal/mandate/schema.sql`, and
`path:api/internal/instrument/schema.sql`.

**Violation evidence.** `path:api/internal/book/sqlite_test.go` performs raw SQL
inserts, bypassing services and handlers, and proves that a second beneficiary,
an unknown-book budget, a cross-book mandate, and a cross-book instrument are
all rejected. It also reads subordinate table metadata and proves no alternate
beneficiary column exists. Requirement: `TRS-P0-010`.

**Change rule.** Relaxing this invariant is a legal/custody decision, not an
implementation convenience. It requires explicit operator direction and legal
review before the schema or test may change.

## Replay / Idempotency Invariants

### oneExternalCallPerSettlementKey

**Statement.** Retrying an outbound settlement key cannot construct or dispatch
a second payment. The first caller commits `calling` before invoking the rail;
later callers return the same durable terminal or unknown record.

**Mechanism.** `settlements.idempotency_key` is unique and
`settlement.SQLiteRepository.Claim` performs insert plus `ready -> calling` in
one serializable transaction. Terminal completion, immutable evidence, and the
Money Ledger outbox row commit together. An x402 response lost after signature
dispatch retains the exact signed payload in Credential Authority and its
digest plus logical reference under the `calling` fence before the paid HTTP
request. Unknown resolution loads that exact payload in memory and may replay
it but may never sign a replacement. New x402 calls carry a preflight-only
phase marker; an interrupted call with that marker and no recovery reference is
known to have stopped before paid dispatch. Legacy unmarked calls stay unknown.

**Safe retry.** Repeat `Settle` with the identical settlement id,
authorization id, instrument id, and idempotency key. For an x402 `unknown`,
use `ResolveUnknown`, which re-resolves the scoped credential and replays only
the retained signature.

**Unsafe retry.** Never change identifiers under an existing idempotency key,
retry an x402 payment with newly generated authorization timing/nonce, or infer
failure from an HTTP timeout.

**Evidence.** `path:api/internal/settlement/sqlite_test.go` and
`path:api/internal/rail/x402/client_integration_test.go`. The x402 payload is
provisioned to Credential Authority before paid dispatch by
`RecordX402RecoveryInput`; SQLite never stores the payload. Terminal completion
removes the protected value after recording the terminal receipt. Requirements:
`TRS-P0-011`, `TRS-P1-001`.

### oneInboundAdmissionPerSignedPayload

**Statement.** One signed x402 payload can settle and admit exactly one declared
price. Concurrent or later replays cannot call facilitator settlement twice or
append a second inflow event.

**Mechanism.** Verification is side-effect free. After it succeeds,
`x402_inbound_admissions.payload_digest` is claimed in a serializable
transaction before `/settle`. The settled admission and its positive Money
Ledger outbox row commit together. A payload already bound to another price is
rejected; a settled replay returns the original receipt; a `calling` or
`unknown` replay refuses rather than guessing.

**Safe retry.** Replay the identical `Payment-Signature` after a settled
response. It returns the existing admission. An unknown admission needs a
future chain-aware reconciliation operation; it must not be settled again by
the ordinary admission path.

**Evidence.** `path:api/internal/rail/x402/facilitator_test.go`,
`path:api/internal/rail/x402/facilitator_integration_test.go`, and
`path:api/internal/rail/x402/facilitator_concurrency_test.go`. Requirement:
`TRS-P1-002`.

## Cross-references

- [`DECISIONS.md`](DECISIONS.md) — durable operator-funds-only decision
- [`SECURITY.md`](SECURITY.md) — third-party custody threat
- [`../concepts/DATA.md`](../concepts/DATA.md) — storage ownership
