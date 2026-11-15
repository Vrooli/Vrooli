# API Endpoints — Notification Hub

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/notification-hub/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/notification-hub/v1/shared/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today: `invalid_request` (400),
`not_found` (404), `internal` (500). Add to the proto enum when a new
REST-exception failure mode appears.

---

## System

### `GET /health`

Service health check. Returns API readiness plus dependency status.
Also mounted at `/api/v1/health` for client callers.
This is an operational REST exception by design: lifecycle systems,
load balancers, and curl probes must be able to read it without a Connect
client.

| | |
|---|---|
| **Auth** | None |
| **Response** | `Response { status: string, readiness: bool, service: string, timestamp: string, version: string, uptime_seconds: int64, dependencies: map<string, DependencyStatus> }` |
| **Errors** | None — always returns 200 and reports an unhealthy status if a dependency fails |
| **CLI** | `notification-hub status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/notification-hub/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Notification Hub read boundary

Ordinary reads do not require an interactive owner login, with one exception:
asks. Creating an ask does not require identity, scope, grant, or permission;
`GetAsk`, `ListAsks` and `Wait` are separate read operations that need a verified
identity and show an ask only to the owner, to the verified agent run (for its
conversation lineage, or by the exact subject key of an ask of the same owner), or
to the service that raised it (see "Who reads an ask" below). Recipient-scoped
procedures (notification `Get`/`List`, delivery
timeline/analytics, routing channel status, recipient metadata, devices, quiet
windows, escalation chain, and preference reads) use the recipient configured
for this Hub instance and return only matching recipient data; an unset
recipient yields no recipient data. The client cannot choose a recipient.
Recipient metadata lookup never creates a missing recipient row. The existing
routing-status call also refreshes its bounded status snapshot, using configured target IDs or `local` for cache keys. Notification detail receipts omit push provider IDs for every caller on this public configured-recipient projection. Catalog reads
expose source/category metadata only. Ask creation and answers,
notification send/relay, subscription/credential operations, and
recipient/device/preferences writes retain their existing identity checks.
Other endpoints are unchanged by this read rule. In particular, reading an ask
never authorizes its answer.

### Quiet windows

`SetQuietWindow` replaces the recipient's window for one weekday (0 is Sunday).
`start` and `end` are 24-hour `HH:MM` strings read in `timezone`, which must be
an IANA zone. The window belongs to the weekday it starts on. If `end` is
earlier than `start` it is overnight: it covers `start` until midnight on
`weekday` and midnight until `end` on the next weekday, so Saturday 22:00 to
07:00 holds Sunday morning and Sunday 22:00 to 07:00 holds Monday morning.
Equal `start` and `end`, malformed times and unknown zones return
`INVALID_ARGUMENT` and leave the stored window untouched. While any window is
active a notification, an ask, or an ask's escalation step is held and
re-evaluated by the recovery sweep; `critical` passes only when every active
window has `critical_override` set, which matters when an overnight window and
the next day's own window overlap. `ListQuietWindows` returns one row per weekday
and `DeleteQuietWindow` removes a row by id.

### Who reads an ask

`GetAsk`, `ListAsks` and `Wait` resolve the caller first; a caller with no
verified identity gets `Unauthenticated` (a loopback caller is not an identity
except as the owner in `personal_local` mode). The caller is one of:

| Reader | How it is verified | Sees |
|---|---|---|
| Owner | Cloudflare Access login, the loopback owner in `personal_local`, or the owner's scenario-authenticator bearer or cookie | Every ask of the configured recipient, with the source drift check on attachments |
| Agent run | The verified agent identity token (`X-Agent-Identity-Token`). A request that presents one is that run and never the owner. | The asks of its verified conversation: those sent by the run (`asks.created_by = run:<run id>`) and by the runs that share its signed `conversation_id` (a recovered run, a child run, a re-attached harness session); a token with no conversation claim sees its own run only. Naming an exact `subject_key`, also the ask of that key sent by a run of the same owner account (`GetAsk`, `Wait`, `Withdraw`; never in a list). Attachments with `source_status: unknown` |
| Service | A scenario-authenticator service token with the exclusive audience `scenario-authenticator:notification-hub-ask-reader` and scope `notification-hub:ask-read`, from a principal named `<source>-ask-reader` | Only the event-sourced asks whose source is `<source>`, for example `vrooli-autoheal-ask-reader` reads the asks of `vrooli-autoheal`; attachments as for a run |
| Sender | A scenario-authenticator service token with the exclusive audience `scenario-authenticator:notification-hub` and scope `notification-hub:send`, from a principal named `<source>-notification-hub` | The asks that sender sent (`created_by = service:<source>`); never another source's, and not the event-sourced asks of its own source (that is the service reader's) |

An ask the reader may not see is `NotFound`, exactly like an ask that does not
exist, and `ListAsks` (including `query` and cursors) returns only the caller's
conversation lineage. Each ask records the verified identity that sent it
(`created_by`: `run:<id>` or `owner:<subject>`), its verified `created_conversation`
and its verified owner account `created_subject`; an ask sent with no identity, an
event-sourced ask, and every ask stored before these columns existed have none and
are owner-only for anyone outside their lineage. `Withdraw` follows the same rule:
by `ask_id` the caller must be in the ask's lineage; by `subject_key`, a run of the
same owner may close it. A `<source>-notification-hub` service sender closes the asks
it sent (`created_by = service:<source>`), by id or subject key, and never an event ask
or another source's; an identityless caller is `Unauthenticated`. A producer that sends an ask whose subject key belongs to
another owner account and conversation gets the packet-check advice
`ask_read_restricted`: it cannot read or wait on that ask. Ask-reader service principals are declared by the reading scenarios and issued with
`scenario-authenticator service-principals sync --apply`.

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.notification_hub.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference (see the fenced example below); `template-manager detemplate
<scenario>` removes it once your real domains are green.

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/notification-hub/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `handlers/<domain>/connect_handler.go`; keep it thin.
3. Update endpoint metadata in `handlers/<domain>/module.go`.
4. If the endpoint has a CLI mirror, bind it (or list it in `omitted[]`
   with a reason) in `cli/manifest.json` — the single source of truth for
   the CLI surface.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and the API↔CLI mapping
contract (every Connect endpoint is bound or omitted in `cli/manifest.json`).

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns


## Request search and continuation

`ConversationsService/ListAsks` reads the current recipient's retained requests without
changing their state. `query` is a literal substring search over question/title, source,
answer/options, answer note and requester reference. `state_filter` accepts `all` (default),
`awaiting` (pending and escalated), `pending`, `escalated`, `answered`, `defaulted`, `expired`
and `withdrawn`. Search and state filtering apply on the server before pagination.

`limit` is a page size (default 50, maximum 200), not a total-result cap. Continue with
`next_cursor` as the next request's `cursor`; an empty value means the current query reached
the end. Cursors bind to recipient, query, state, order and filters; changing those requires a fresh first page.
The compatibility `open_only` shortcut selects awaiting requests; conflicting filters fail
with InvalidArgument. Invalid/foreign-query cursors fail rather than silently restarting.

`order` is one of four, each valid with any `state_filter`, and each ends with the larger or
smaller id so a page boundary is exact:

| `order` | Sort key |
|---|---|
| `ORDER_NEWEST` (default) | `created_at` descending |
| `ORDER_OLDEST` | `created_at` ascending |
| `ORDER_ATTENTION` | escalated first, then pending asks whose `deadline` is at or before the page's reference time plus the closing window (6 hours; an expired deadline not yet processed counts; the nearest `deadline` first), then urgency (critical, high, normal, low), then the earliest `deadline` (an ask with none last), then `created_at` descending |
| `ORDER_DUE` | the earliest `deadline` (none last), then urgency, then `created_at` descending |

An unknown order fails with InvalidArgument. A cursor is valid only for the order, search,
state, reader and filters it came from. A page is a live keyset on the whole sort key: an
ask that becomes urgent after a page was served sorts before the cursor and shows on
refresh, not midway through older pages, and an ask answered meanwhile is simply gone.
Every persisted timestamp is stored as fixed-width UTC text with a nine-digit fraction
(`2026-11-14T12:00:00.500000000Z`), so SQLite's text order is time order, within a second
too; `deadline` and `created_at` in responses use that form.

The closing window is measured from the instant the first page was read: the
cursor carries that time, so asks do not cross into the band between pages while
the clock moves, and a page never repeats or skips a row. A list read afresh uses
the new time. Every `ListAsks` response carries `closing_window_seconds`, and each
open ask inside the window carries `closing_soon`, so a client names and marks the
band from the Hub's answer and keeps no copy of the number. `closing_soon` is set only by
`ListAsks`.

`urgencies` and `senders` filter on the server before pagination: only asks of those
urgencies (`low`, `normal`, `high`, `critical`) and from those senders (the `Ask.sender` text,
matched exactly). Values of one filter combine with OR, the two filters with AND, and an
unknown urgency fails with InvalidArgument. `query` also matches the sender's name. Every
response carries `facets`: the number of asks each urgency and each sender holds under the
request's `query` and `state_filter`, where a facet's counts apply the other facet's filter
but not its own (so choosing `high` leaves the urgency counts unchanged and narrows the
senders). A value the request selected is listed even at zero; senders are bounded to the 20
largest plus the selected ones.

`Ask.sender` is who the operator reads as asking, never empty. The requester declares it with
`AskRequest.sender` (1 to 60 characters, unverified, display only; unset on a revision keeps
the ask's sender). A caller's tooling may instead pass `AskRequest.default_sender`, such as the
label of the session the request came from: the Hub uses it, repaired to fit those rules, only for
an ask it creates that declares no `sender`. A revision never renames from it, and a replacement
keeps the name of the ask it replaces. Without either the Hub names it from what it verified: the service's name
("Research Desk") for an event-sourced ask or a verified service sender, "Agent run" for a
verified run that declared none, otherwise "Unnamed sender". It never decides who may read
an ask.

Every `ListAsks` response carries `total_awaiting`: the exact number of pending and
escalated asks the reader may see (all of the recipient's for the owner, only the run's own
lineage for a run), whatever `query`, `state_filter` or `cursor` the request had. Native
permission requests are not counted. The count, and any page over awaiting asks, read the
partial index `idx_asks_open_attention`, so their cost follows the open asks, not the history.

`NotificationsService/Get` returns `native_consent_review_url`, set only when the Hub
recognises the notification as an Agent Manager permission request (source `agent-manager`,
category `authorization-requested`, a UUID dedupe key that matches the approval path of the
link, an https or loopback link without credentials, query or fragment: the same test as
`ListNativeConsents`). It asks Agent Manager nothing; the page it names shows the live state.
It is empty for every other notification, whatever shape their `context_url` has.

Pages use a live newest-first keyset with ID as the equal-time tiebreaker. Newly created
requests appear on refresh. State transitions can change a live result set between reads;
this is not a frozen snapshot. History exposes the full lifecycle and Needs you fixes the
state to awaiting. Both share search, accessible Load more/retry and detail-return context.
The product target is `../concepts/FLOWS.md`, amendment
`NH-DESIGN-20261025-HISTORY-COMPLETE`; older latest-200 screenshots are historical.

## Decision packet attachments

Attachments are files and folders an ask carries so the operator can open them
inside the Hub. The design, limits and robustness rules are in
[`../concepts/DECISION_PACKETS.md`](../concepts/DECISION_PACKETS.md); this section
is the wire reference. Proto: `AskAttachment`, `AskAttachmentEntry`,
`StatAttachmentBlobs`, `PutAttachmentBlob`, `PacketWarning`.

| Endpoint | Auth | Notes |
|---|---|---|
| `ConversationsService/StatAttachmentBlobs` | Verified producer (an agent run, the owner or a `<source>-notification-hub` service sender; `Unauthenticated` without, `PermissionDenied` for a service token that is not a sender) | Up to 1,000 lowercase hex SHA-256 hashes; returns those the Hub lacks. A hash that is present is kept alive for 24 more hours. A blob row whose bytes were lost counts as missing. |
| `ConversationsService/PutAttachmentBlob` | Identity optional | `sha256`, `media_type` (the sender's declaration, kept for the record, never served), `content` (at most 64 MiB). The Hub recomputes the hash; a mismatch, a malformed hash or an oversize file is `InvalidArgument`. `existed` is true when the Hub already held the bytes. `ResourceExhausted` when 1 GiB of uploads no ask references yet is already waiting (they expire after 24 hours). `Unavailable` when the Hub has no attachment store. Two uploads run at once, and one that finds no slot within 30 s gets HTTP `503` with `Retry-After`. Anonymous callers upload directly by hash and cannot inspect the shared inventory. |
| `ConversationsService/Ask` | Direct producer | `attachments[]` reference blobs by hash. An attachment the Hub cannot accept is left out and reported in `warnings[]`; the ask is never refused for it. Anonymous asks keep uploaded content but discard `source_path` and `source_host`. |
| `ConversationsService/GetAsk` | Owner, sending run, or raising service | Returns `attachments[]` with `preview_kind`, `source_status`, `source_checked_at`, `truncated`, and `attachment_count`. `source_status` is checked against the sender's path only for the owner; other readers get `unknown`. |
| `ConversationsService/ListAsks` | Owner, sending run, or raising service (own asks only) | Rows carry `attachment_count` only. |
| `GET /api/v1/asks/{ask_id}/attachments/{name}` | Verified owner (as `Answer`) | REST exception `browser_surface`. `?path=<entry>` selects a file of a folder attachment. Also answers `HEAD`. |

### Limits

Attachments per ask 24; one file 64 MiB; all attachment bytes of an ask 256 MiB; files
in a folder 500; one explainer per ask. Sizes are the Hub's own measurement of the
uploaded bytes, never the sender's claim. Every Connect procedure other than
`PutAttachmentBlob` accepts at most 8 MiB per message, measured after
decompression, so a small gzip body cannot expand past it. Uploads that no ask
references yet are capped at 1 GiB in total.

### Warning codes

`warnings[]` is a list of `PacketWarning{code, field, message, severity, carried_over}`. Severity
`dropped` means the Hub left the named part out; `advice` means it kept the part.
`carried_over` is true on a revision's finding that the open ask already had, word for word,
about a part the revision did not touch (callers fold these into a count).

| Code | Severity | Meaning |
|---|---|---|
| `attachment_blob_missing` | dropped | A file (or folder entries) whose bytes the Hub does not hold. Upload with `PutAttachmentBlob` first. |
| `attachment_limit` | dropped | More than 24 attachments (counted after invalid ones are dropped), or more than 256 MiB in one ask. |
| `attachment_invalid` | dropped / advice | Unknown kind, a file without a valid `sha256`, a folder with no valid entry (dropped); an unknown role (advice, listed as evidence). |
| `attachment_name_changed` | advice | The name was sanitized or made unique (a repeat is numbered before its extension: `notes-2.md`). The warning says what it became. |
| `attachment_source_ignored` | advice | `source_path` was not an absolute path. |
| `attachment_folder_truncated` | advice | A folder kept its first 500 files and has `truncated = true`. |
| `attachment_explainer_limit` | advice | A second explainer, or a folder marked explainer, is listed as evidence. |
| `attachment_remove_unknown` | advice | `remove_attachments` named an attachment the ask does not have; nothing was removed for it. |

### Content route

`GET /api/v1/asks/{ask_id}/attachments/{name}` streams the Hub's stored copy.

- **Auth.** A verified owner identity, through the same path `Answer` uses: a
  Cloudflare Access login (the `Cf-Access-Jwt-Assertion` header Cloudflare adds to
  every request, including `<img>` and `<video>`), the loopback owner in
  `personal_local` mode, or the owner's bearer token or cookie. No verified owner
  is `401`; an Access assertion that fails verification is `401` with no fallback.
  This is stricter than `GetAsk`, which also admits the sending run and the raising
  service (see "Who reads an ask"). With an owner: an unknown ask,
  another owner's ask, an unknown attachment, an unknown or outside `?path=`, a
  `?path=` on a file, and a folder without `?path=` are all the same
  `404 not found`, so the route reveals nothing about what exists.
- **Ranges.** `Range` is honored (206, `Content-Range`), `HEAD` and `If-None-Match`
  work, so video and audio seek and a revisited file is not downloaded again.
- **Headers.** `Content-Type` is the Hub's own classification (shared classifier
  `packages/api-core/filekind`); a type the sender supplied is never served, and a
  file the classifier cannot place is `application/octet-stream`.
  `X-Content-Type-Options: nosniff`; `Cache-Control: private, no-cache` with
  `ETag: "sha256-<hash>"`: the URL names the ask and attachment, not the bytes, and a
  revision can attach different content under the same name, so a browser always
  revalidates and an unchanged file answers `304`. No `?v=` is needed on the URL;
  `Content-Security-Policy: sandbox; default-src 'none'; img-src data:; media-src data:; style-src 'unsafe-inline'`;
  `Content-Disposition: inline; filename=...` (`filename*=utf-8''...` for a name that
  is not plain ASCII).
- **Reachability.** The UI server proxies every `/api/*` path to the API unchanged,
  including `Range` and conditional headers, so the route works through the UI
  origin and the Cloudflare Access path like the other `/api/v1` routes.

## Decision packet answer form and context requests

The answer form (typed questions under the main options) and "I need more
context" are specified in
[`../concepts/DECISION_PACKETS.md`](../concepts/DECISION_PACKETS.md). Proto:
`AskQuestion`, `AskQuestionChoice`, `AskQuestionSection`, `AskQuestionResponse`,
`AskSectionVerdict`, `AskContextRequest`, `RequestContext`.

| Endpoint | Auth | Notes |
|---|---|---|
| `ConversationsService/Ask` | Direct producer | `deadline` and `sensitivity_label` are optional: the Hub applies 24 h and `private`, and a request with no deadline can be sent again without conflicting over it. A revision (a `subject_key` with an open ask) keeps the open ask's attachments and questions it does not mention: what it sends replaces the part of the same attachment name or question id, `clear_attachments` and `clear_questions` drop every current part first, `remove_attachments` drops the named attachments (a name the ask lacks is reported as `attachment_remove_unknown`), `remove_questions` drops the questions with the named ids (an id the ask lacks is `question_remove_unknown`), `context_response` (1-4,000 characters) answers the operator's open context questions and is stored with them (with no open question it is dropped and reported as `context_response_unused`), and a replacement ask carries the merged packet over. A direct request's `idempotency_key` is a retry only when it is the key of the latest request applied to that subject's ask; an event id, delivered in any order, keeps naming the ask for every revision it carried. A question of type `section_review` may set `sections_from` (an attachment name, or `name/path` inside a folder): the Hub derives its sections from that Markdown file's headings and derives them again whenever a revision replaces the file. The response carries `state`. `accepts_context_requests` declares that the requester answers the operator's questions by answering it (`context_response`) under the same `subject_key`; without a key it is dropped (`context_requests_need_subject_key`). `questions[]` is the answer form. A question that breaks a rule is left out (`question_invalid`, severity `dropped`, the message names the rule); one of an unknown type is kept as a free-text question (`question_type_unknown`, advice). `warnings[]` also carries the packet check's advice. Neither fails the ask. Questions are part of the ask's text: changing them on a subject ask updates it in place without a new push, and a retry of a plain ask with different questions is `AlreadyExists`. |
| `ConversationsService/Answer` | Configured recipient | `responses[]` answer the ask's questions. An unknown `question_id`, a repeated question, or a response of the wrong shape (see below) is `InvalidArgument`, and the ask stays open. A required question must be answered unless `answer` is `other`. A response with no content is ignored. `attachment_notes[]` (`attachment`, `path`, `note`; at most 24) pin notes to the ask's attachments or a folder entry; a note naming something the ask does not have is `InvalidArgument`. If the requester revised the ask while the answer was being made, nothing is stored and the call fails `FailedPrecondition`, message prefix `ask_changed:`, error metadata `Hub-Error-Reason: ask_changed`; reload the ask and submit again. |
| `ConversationsService/RequestContext` | Configured recipient (as `Answer`) | `ask_id`, `question` (1-2,000 characters). The ask must be open (`FailedPrecondition` otherwise), belong to the caller (`NotFound` otherwise), and its requester must have declared `accepts_context_requests` (`FailedPrecondition` with `Hub-Error-Reason: context_not_accepted` otherwise; `Ask.can_request_context` says which). Repeating a question that is still unaddressed returns the existing request. At most 5 unaddressed requests per ask: the sixth is `ResourceExhausted`. |
| `ConversationsService/Wait` | Owner, sending run, or raising service (as `GetAsk`) | Returns `state`, `answer`, `reason`, `note`, `responses[]`, `attachment_notes[]` and `answer_summary` once the ask is resolved. With `return_on_context_request` it also returns while the ask is open and an unaddressed context request exists: `state` is still `pending`, `context_request` is the oldest unaddressed one, and `reason` says so. Without the flag a context request never ends the wait. One call lasts at most 10 minutes whatever the `deadline`: it then returns the open ask with a `reason` beginning `wait_slice`, and the caller waits again. The route has an operation budget, so a response written after the server's ordinary 30 s write deadline is not cut. |
| `ConversationsService/CheckAsk` | Direct producer (as `Ask`) | Dry run of `Ask`: validates, applies the revision merge, runs the packet check, and returns `warnings[]`, the notification `push_title` and `push_body`, `revises_ask_id` (the open ask a keyed request would revise), and the `sender` the operator would see with its `sender_source` (`declared`, `default`, `kept`, `service`, `agent_run` or `unnamed`), worked out by the rule a real send uses. Stores nothing and uploads nothing: the request's attachments are taken as stored at the `sha256` and size they state, and `file_contents` (sha256 to bytes) supplies the explainer page and the Markdown file a review follows, read from the request and kept nowhere. |
| `ConversationsService/GetAsk` | Owner, sending run, or raising service | Adds `questions[]`, `responses[]`, `attachment_notes[]`, `context_requests[]` (oldest first, with `addressed_at`, `addressed_revision` and the requester's `response`), `answer_summary`, `question_count`, `open_context_request_count` and `can_request_context` (the ask is open and its requester declared it answers questions; the UI shows the question button only then). |
| `ConversationsService/ListAsks` | Owner, sending run, or raising service (own asks only) | Rows carry `question_count`, `open_context_request_count` and `answer_summary`, not the form itself. |

### Response shapes

| Question type | Valid response |
|---|---|
| `single_choice`, `variant` | `selected` holds at most one of the question's choice keys; `text` may add a note. |
| `multi_choice` | `selected` holds any of the choice keys (repeats collapse); `text` may add a note. |
| `section_review` | `sections` holds `{section_key, verdict, note}` for the sections the operator reviewed; verdict is `approve`, `change`, or `drop`. The Hub stores and returns one verdict per section in the review's order: a section left alone takes the question's `unanswered_verdict` (`approve` when empty). |
| `text`, any unknown type | `text` only. |

A required question counts as answered when a choice was selected, at least one
section was given a verdict, or text was written; a note alone does not answer a
choice question. `text` and notes are at most 4,000 characters.

### Packet warning codes of the form and the check

| Code | Severity | Meaning |
|---|---|---|
| `context_requests_need_subject_key` | dropped | `accepts_context_requests` was sent without a `subject_key`. |
| `question_invalid` | dropped | A question (or, with field `questions[i].recommended`, just its recommendation) broke a rule. |
| `question_type_unknown` | advice | Kept; the Hub shows it as free text. |
| `question_attachment_unknown` | advice | A choice or section points at an attachment name the ask does not have. |
| `internal_id_in_question`, `path_in_question`, `long_question`, `missing_big_picture`, `missing_recommendation_reason`, `local_context_url`, `unattached_path` | advice | The cover reads poorly to a reader with no context. |
| `explainer_not_html`, `explainer_too_large`, `explainer_network`, `explainer_no_viewport`, `explainer_form`, `explainer_unknown_attachment` | advice | The explainer page will not work as intended in its sealed frame. |

Warnings are computed for every `Ask` response, including a repeat of an
earlier request.
