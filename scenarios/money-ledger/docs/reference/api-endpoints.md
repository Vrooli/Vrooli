# Money Ledger API endpoints

The machine-readable source of truth is [`../../.vrooli/endpoints.json`](../../.vrooli/endpoints.json).
Wire messages and services live in `packages/proto/schemas/money-ledger/v1`;
the generated Connect clients are used by the UI and CLI.

## Operational endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Lifecycle/readiness health response. |
| GET | `/api/v1/capabilities/describe` | Typed capability metadata. |

## Books and ledger

All RPCs use `POST /vrooli.money_ledger.v1.ledger.<Service>/<Method>`.

| Service / method | Request → response | Behaviour |
|---|---|---|
| `BooksService/CreateBook` | `CreateBookRequest → CreateBookResponse` | Creates a currency-scoped book. |
| `BooksService/ListBooks` | `ListBooksRequest → ListBooksResponse` | Lists books and currencies. |
| `BooksService/CreateAccount` | `CreateAccountRequest → CreateAccountResponse` | Adds an account to a book. |
| `BooksService/ListAccounts` | `ListAccountsRequest → ListAccountsResponse` | Lists accounts for a book. |
| `JournalService/GetPosting` | `GetPostingRequest → GetPostingResponse` | Reads one posting and audit history. |
| `JournalService/ListPostings` | `ListPostingsRequest → ListPostingsResponse` | Reads immutable postings with filters. |
| `JournalService/ReversePosting` | `ReversePostingRequest → ReversePostingResponse` | Appends a linked correction; never edits the original. |
| `JournalService/Transfer` | `TransferRequest → TransferResponse` | Appends paired inter-account postings. |
| `PositionService/GetPosition` | `PositionRequest → PositionResponse` | Computes cash/runway and source availability at read time. |
| `PositionService/GetStatement` | `StatementRequest → StatementResponse` | Computes a period statement and coverage. |
| `PositionService/DeclareGoal` | `DeclareGoalRequest → DeclareGoalResponse` | Declares a sustained financial goal. |
| `PositionService/ListGoals` | `ListGoalsRequest → ListGoalsResponse` | Returns goal thresholds, windows, progress, and verdicts. |

## Ingestion

All RPCs use `POST /vrooli.money_ledger.v1.ingest.IngestService/<Method>`.

| Method | Purpose |
|---|---|
| `RegisterAdapter` | Register a manual, file, or upstream adapter. |
| `ListAdapters` | Read adapter availability and receipt metadata. |
| `IngestEvent` | Admit one typed event with amount, currency, basis, and provenance. Duplicate external IDs are reported without a second posting. |
| `RunAdapter` | Execute an adapter and retain failure/last-success evidence. |
| `ImportFile` | Import CSV through a registered file adapter. |

Connect errors use the canonical envelope. The UI renders validation and
request errors as text; partial source failures are represented in successful
responses by `partial` and `availability`, never by a fabricated zero.

## Change procedure

Add or change a proto first, implement the thin generated handler, refresh
endpoint metadata through the scenario lifecycle, and update the CLI mapping.
Do not hand-edit `.vrooli/endpoints.json`. Validate with the API/contract
tests and [`cli-commands.md`](cli-commands.md).
