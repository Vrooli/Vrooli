# Money Ledger API endpoints

The machine-readable source of truth is [`../../.vrooli/endpoints.json`](../../.vrooli/endpoints.json).
Wire messages and services live in `packages/proto/schemas/money-ledger/v1`;
the generated Connect clients are used by the UI and CLI.

Every route is declared in the access policy (`api/internal/access/policy.go`)
and passes one gate before it runs; an undeclared route is refused. The
**Access** column below is that policy. *Open* routes need no identity;
*operator* means the person the host's owner-binding policy names as the owner;
*treasury* and *landing-page-business-suite* are declared service principals.
Agent runs are never writers. Browser requests from any other site are refused
before routing. Rationale and threat model:
[`../internal/SECURITY.md`](../internal/SECURITY.md).

A refused write answers `Unauthenticated` (HTTP 401) when no usable identity was
presented and `PermissionDenied` (403) when the identity may not do this, with a
message that says why. `AccessService/GetWriteAccess` reports the same decision
before a client offers a write.

## Operational endpoints

| Method | Path | Access | Purpose |
|---|---|---|---|
| GET | `/health` | open | Lifecycle/readiness health response. |
| GET | `/api/v1/capabilities/describe` | open | Typed capability metadata. |

## Access

All RPCs use `POST /vrooli.money_ledger.v1.access.AccessService/<Method>`.

| Method | Access | Behaviour |
|---|---|---|
| `GetWriteAccess` | open | Whether the presented caller may write, the actor writes are recorded as, or a typed refusal (`SIGNED_OUT`, `UNVERIFIED`, `NOT_OWNER`, `AGENT`, `UNDECLARED_SERVICE`, `CONFLICT`) with a reason; who is signed in; whether a browser session is present. Renews an expired browser session from its refresh cookie. |
| `SignIn` | open (session) | Forwards email, password and second factor to Scenario Authenticator for the `money-ledger` audience and sets HttpOnly session cookies. Returns `mfa_required` with a challenge, or the new session's write access, which may still refuse: signing in does not make a person the owner. Refused credentials are `Unauthenticated`, throttling `ResourceExhausted`, an unreachable provider `Unavailable`. |
| `SignOut` | open (session) | Clears the cookies and revokes the session; `revocation_confirmed` is false when the provider could not confirm. Idempotent. |

## Books and ledger

All RPCs use `POST /vrooli.money_ledger.v1.ledger.<Service>/<Method>`.

| Service / method | Access | Behaviour |
|---|---|---|
| `BooksService/CreateBook` | operator | Creates a currency-scoped book. |
| `BooksService/ArchiveBook` | operator | Archives a book; its journal is preserved. |
| `BooksService/ListBooks` | open | Lists books and currencies. |
| `BooksService/CreateAccount` | operator | Adds an account to a book. |
| `BooksService/ListAccounts` | open | Lists accounts for a book. |
| `JournalService/GetPosting` | open | Reads one posting and audit history. |
| `JournalService/ListPostings` | open | Reads immutable postings with filters. |
| `JournalService/ReversePosting` | operator | Appends a linked correction; never edits the original. |
| `JournalService/Transfer` | operator | Appends paired inter-account postings. |
| `PositionService/GetPosition` | open | Computes cash/runway and source availability at read time. |
| `PositionService/GetStatement` | open | Computes a period statement and coverage. |
| `PositionService/DeclareGoal` | operator | Declares a sustained financial goal. |
| `PositionService/ArchiveGoal` | operator | Archives a goal without deleting it. |
| `PositionService/ReparentGoal` | operator | Moves a goal to another active book. |
| `PositionService/ListGoals` | open | Returns goal thresholds, windows, progress, and verdicts. |

The journal records the verified caller as each write's actor. Requests carry
no `actor` field.

## Ingestion

All RPCs use `POST /vrooli.money_ledger.v1.ingest.IngestService/<Method>`.

| Method | Access | Purpose |
|---|---|---|
| `RegisterAdapter` | operator, treasury | Register a manual, file, or upstream adapter. |
| `ListAdapters` | open | Read adapter availability and receipt metadata. |
| `IngestEvent` | operator, treasury, landing-page-business-suite | Admit one typed event with amount, currency, basis, and provenance. Duplicate external IDs are reported without a second posting. |
| `RunAdapter` | operator | Execute an adapter and retain failure/last-success evidence. |
| `ImportFile` | operator | Import CSV through a registered file adapter. |
| `ImportOperatorInputs` | operator | Rehearse or apply typed operator inputs; a rehearsal reads a host file, so it is the owner's too. |
| `OperatorInputStatus` | open | The thirteen operator input fields with status and age. |
| `PreviewCryptoObservation` | open | Computes a crypto observation's effects without posting. |
| `AcceptCryptoReceipt` | treasury | Atomically records a verified crypto receipt and its posting. |
| `GetCryptoReceipt` / `ListCryptoReceipts` | treasury, operator | Reads accepted receipts, which carry customer credit attribution. |

Reserved adapter ids belong to one writer: `treasury` to Treasury, `manual` and
`transfer` to the operator. `treasury-base-usdc` is written only by
`AcceptCryptoReceipt`.

Connect errors use the canonical envelope. The UI renders validation and
request errors as text; partial source failures are represented in successful
responses by `partial` and `availability`, never by a fabricated zero.

## Change procedure

Add or change a proto first, implement the thin generated handler, **declare
the route in `access.Policy`** (the gate refuses an undeclared route and a test
fails), regenerate endpoint metadata with
`cd api && go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json`, and
update the CLI mapping (a write verb takes `--local-owner`). Do not hand-edit
`.vrooli/endpoints.json`. Validate with the API/contract tests and
[`cli-commands.md`](cli-commands.md).
