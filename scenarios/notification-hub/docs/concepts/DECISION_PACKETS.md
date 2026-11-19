# Decision Packets

A decision packet is everything the operator needs to answer one ask cold, on a
phone, without the requester's context. It has three parts that improve
independently:

| Part | What it holds | Owner |
|---|---|---|
| **Cover** | question, big picture, recommendation, technical details | Writing rules: `skills/notification-hub/SKILL.md` |
| **Attachments** | files and folders, opened inside the Hub; at most one sealed HTML **explainer** | This document; viewer: `packages/file-viewer` |
| **Answer form** | the main options plus optional typed questions | This document |

The proto contract is `packages/proto/schemas/notification-hub/v1/conversations/conversations.proto`
(`AskAttachment`, `AskQuestion`, `AskQuestionResponse`, `AskContextRequest`,
`PacketWarning`, and the blob and context RPCs).

## Raising and waiting

`notification-hub conversations ask` needs a question and options (`--yes-no`, or
2 to 8 `--option`); the rest has a default, so the minimum is
`ask --yes-no --subject-key K --question Q`.

Three words are kept apart everywhere (CLI help, the skill, this document):
**options** are what the operator picks from, and **answer** is their reply;
**form questions** are the typed extras under the options, whose **choices** live
in the form file; **expires** (`--expires`) is when an unanswered ask closes, and
**timeout** (`--timeout`) is how long a `wait` or `ask --and-wait` command waits.
The old flag names for these (`deadline` and `wait-for`, with their two dashes)
are gone. They fail as unknown options, and the error names the new flag.

| Part | Default | Where it lives |
|---|---|---|
| Expiry (`--expires`, wire field `deadline`) | 24 h, `DefaultAskDeadline` | The Hub: a request with no `deadline` gets it, and its deadline is then not part of the ask's identity, so the same request sent again returns the same ask. A request that names a deadline keeps the old rule: a changed deadline is a changed term. |
| Sensitivity label | `private`, `DefaultAskSensitivity` | The Hub, for a request with no label. |
| Urgency | `normal`, `DefaultAskUrgency` | The Hub, for a request that names none, and the CLI's manifest default, so every route agrees. Until 2026-11-14 the Hub's default was `high`; a producer whose work is blocked on the answer now says `high` itself (the agent-manager effort decision does). A `normal` ask is held back from a recipient whose per-source minimum urgency is `high`, which a `high` one was not. |
| Idempotency key | Derived: `ask-` plus 20 hex characters. An ask without a subject key hashes the request without its absolute deadline and key, plus the `--expires` text as the agent wrote it; a keyed ask gets a fresh random key per send | The CLI, printed. Without a subject key the same terms and the same `--expires` text give the same key, so a retry returns the same ask. The expiry is keyed as written, not as the time it resolves to: `6h` resolves to a different time on every send, which would make every resend a new ask, while leaving it out of the key would let a resend with another expiry return the old ask unchanged. A keyed ask needs no terms-derived key: the subject already makes identical terms change nothing. An explicit `--idempotency-key` reused for different terms is `AlreadyExists`, and the CLI says what to do next. |

**What a key means to the Hub.** A direct request (the CLI, or `Ask` over Connect)
sends a key only so that its own retry is recognised, and a retry is always of the
latest request. The Hub therefore replays a key only when it is the key of the
latest request applied to that subject's ask; the same key with different terms is
a conflict, but a key an earlier request used is just a request returning to
earlier terms and is applied (A, then B, then A again changes the ask three times).

One edge stays: a caller that chooses its own key per set of terms (the CLI does
not: a keyed `ask` gets a fresh key every time) and returns to earlier terms
whose **options** differ from the open ask's, with the earlier key, gets
`AlreadyExists`. A change of options is a replacement, which creates a new ask and
a new notification, and the notification's identity is (requester, key): the key
of A's first notification is still taken. Nothing is stored. Send a new key for
the return, as for any request that is not a retry.
An event (`*.decision_requested.v1`) is different: it is delivered at least once and
in any order, so every event id the ask has carried still names it, and a late
delivery of an earlier revision changes nothing.

A revision says only what changes, and the Hub does the merge. `AskRequest` marks
`recommended`, `recommendation_reason`, `default_answer`, `reversible`, `urgency`,
`context_url`, `big_picture`, `technical_details`, `reassert_within_seconds` and
`accepts_context_requests` as `optional`, so presence is on the wire: when a
request carries a `subject_key` that has an open ask, a field it leaves unset (and
a `question` or `options` it leaves empty) keeps the open ask's value, and a field
it sets, even to the empty string, takes that value. The Hub reads the open ask,
merges the request over it and writes the result in one write transaction, with
the attachments and form questions, so two revisions that arrive together are
applied one after the other and neither undoes the other. A request with no open
ask takes each unset field's default and is refused without a question and
options (`no open ask has the subject key ... to revise`). When the open ask was
answered or closed after the sender's last look, the refusal says what became of
it (`your revision was not applied: the ask with the subject key ... was answered
"yes"`) and names `conversations get --subject-key` to read it; sending the whole
ask starts a new one. A whole ask sent within a short while of the answer, with
the same terms, returns that ask (`returned`).

Two rules keep a merge coherent. The recommendation and the default answer name
options, so they are kept only when the options are: a request that sends its own
`options` has new options, which replace the ask, and its old recommendation and
default answer are not carried over. The recommendation's
reason belongs to the answer it explains, so it is kept only with the
recommendation. `reversible` is kept on its own.

Event producers (`operatordecision.Facts`) send the whole decision and leave
nothing unsaid, so they stay full replacements; only the Connect `Ask` path has
presence. The CLI sends only the flags a command gave, and a keyed ask gets a fresh
idempotency key on every send: a key made from the terms would be an earlier
revision's key when an agent returns to earlier wording, and the Hub ignores an
event it has already applied.

`--body-file FILE` (or `-` for standard input) reads the cover from markdown: the
first paragraph is the question, text before the first `##` heading and
`## Background` make the big picture, `## Details` is the technical details. A
leading `# title` line is ignored, any other `##` section is reported on stderr and
left out, and giving the same part as a flag too is an error.

`ConversationsService/CheckAsk` is the dry run behind `ask --dry-run`. It runs the
request through the same validation, revision merge (`sealAsk`) and packet check as
`Ask`, then returns the warnings, the notification title and body the operator
would get (of a revision, the merged ask), and the open ask a keyed request would
revise (`revises_ask_id`). It stores no ask, notification, attachment or blob row.
It also uploads nothing: it takes each attachment of the request as stored, at the
`sha256` and `size_bytes` the request states, and reads the two kinds of file the
check looks inside from `CheckAskRequest.file_contents` (sha256 to bytes): the
explainer page, and the Markdown file a review question follows. Those bytes are
read from the request and kept nowhere, and a file the request does not include is
read from the Hub's store if it is there. The CLI sends only those two kinds,
each up to 2 MiB.

`AskResponse` carries `state` (the ask's state after the call) and the CLI's output
ends with one line, `ask_id=... state=... outcome=... [idempotency_key=...] next: ...`.
`--json` prints `{ask_id, state, outcome, warnings[]}`, every key always present.

### Waiting

`conversations wait <ask-id>`, `wait --subject-key K` and `ask --and-wait` exit
with one status per outcome:

| Status | Meaning |
|---|---|
| 0 | An answer is available: the operator answered, or the default applied. |
| 3 | The operator asked for more context; the ask is open (see Context requests). |
| 4 | The caller's own wait (`--timeout`, default 30 minutes) ran out with the ask still open. The ask has not expired. |
| 5 | The ask closed without an answer: withdrawn, dismissed, or expired. A wait by id also ends here when a revision with new options replaced the ask (withdrawn, reason `replaced`); a wait by `--subject-key` follows the replacement instead. |

Status 1 is a failure to wait at all (unreachable Hub, refused read, bad flags). With
`--json` the same document is printed on 0, 3, 4 and 5. One real example of each, with
the exact JSON paths (`.answer`, `.responses[]`, `.context_request.question`,
`.reason`), is in `skills/notification-hub/references/cli-results.md`, which a test
regenerates from the CLI's own output.

One `Wait` call never lasts longer than ten minutes (`hub.MaxWaitSlice`), whatever
deadline the caller gave: the Hub then returns the ask as it is, still open, with
the reason `wait_slice`. Maintenance preparation may also return a pending ask
early with that prefix, so the caller can resume waiting after the service returns.
The server's ordinary 30 s write deadline would otherwise
cut a longer response, which a caller saw as a bare EOF; `Wait` is mounted with an
operation budget (`apiserver.WithOperationBudget`) of the slice plus a margin. The
CLI asks again until its own deadline, and also carries on across a dropped
connection (`Unavailable`, `DeadlineExceeded`, `Aborted`, EOF) with a pause that
grows from 1 s to 15 s. A refusal (`Unauthenticated`, `NotFound`) is never retried.

## Robustness rules

These rules hold everywhere. They exist so that a broken attachment, form, or
explainer can never stop a decision from reaching the operator or lose the
operator's answer.

1. **Sending never fails because of a broken packet part.** An attachment whose
   blob is missing, an invalid question, or an explainer that breaks a rule is
   left out or kept with advice, and `AskResponse.warnings` says so. The cover
   fields keep their hard validation. One exception, for agents only: a direct
   ask (no producer source) that has a recommendation without a reason, a page
   that only restates the cover, or a page the operator cannot read is refused
   whole before anything is stored (see "Decision quality" below). The agent is
   still running and can fix it; the refusal names each finding and its fix, and
   `--dry-run` reports it first. A scenario's event-built decision is never
   refused for these: it gets advice, because no agent stands behind it.
2. **The operator can always answer.** "Other" with free text and "Accept
   recommendations" are always available. A required question binds only when
   the operator picks one of the requester's options.
3. **Unknown question types degrade to text.** A reader that does not know a
   question type shows its prompt with a free-text box and returns the text.
4. **Explainers inform; the Hub form answers.** An explainer page can pre-fill
   the form through the bridge, but only the Hub's own submit button answers.
5. **Attached bytes are copied; linked artifacts are bounded reads.** An
   attachment is served from the Hub's stored copy. A local path in decision
   prose opens only after the owner taps it, only for an ask addressed to that
   owner, only beneath an allowed artifact root, and only for a regular file no
   larger than 64 MiB. Symlink escapes and directories are refused. The default
   roots are `.vrooli/work`, the steward delegation folder and the research home
   (so Research Desk packets open); add operator-owned
   roots with `NOTIFICATION_HUB_LINKED_ARTIFACT_ROOTS` (platform path-list
   syntax). Missing roots leave the decision answerable and show an in-Hub
   unavailable message.

## Attachments

### Sending

Agents use `notification-hub conversations ask` with `--attach PATH[=LABEL]`
(repeatable; a file or a folder), `--explainer PAGE.html`, and `--form FILE`
(YAML or JSON questions; `sections_from: <attached .md>` makes a review follow its
`##`/`###` headings), and, on a revision, `--remove-attachment NAME` and `--remove-question ID` (repeatable)
to drop one attachment or form question. The exact shapes are in
`docs/reference/cli-commands.md` § Notification Hub decisions.

The sender (the CLI, or any client) reads the files with its own permissions and
uploads each distinct file once:

1. `StatAttachmentBlobs(sha256[])` returns the hashes the Hub lacks.
2. `PutAttachmentBlob(sha256, media_type, content)` stores each missing blob.
   The Hub recomputes the hash and refuses a mismatch.
3. `Ask(..., attachments[])` references blobs by hash. A folder is one
   attachment with `entries[]` (relative paths), not one attachment per file.

Adding or revising an ask and uploading its attachment bytes require no identity
or approval. `PutAttachmentBlob` accepts identityless callers within the same
hash validation, size, quota and concurrency limits. `StatAttachmentBlobs`
remains verified because it reveals which content hashes the Hub already holds;
an identityless CLI skips that lookup and uploads each file directly by hash.
The Hub accepts attachment metadata but strips `source_path` and `source_host`
for an identityless sender, so it never probes a caller-selected host path.
Owner-only attachment viewing and owner-only answers remain protected.

**An event carries no files.** A `*.decision_requested.v1` event cannot say who
published it (the webhook proves only that vrooli-events delivered it), so the
Hub builds its ask without attachments, whoever the event names as its source.
A scenario that wants a page or files on its decisions sends `Ask` as its own
sender. The service sender also closes its own ask with `Withdraw`, by ask id or
subject key, and reads it back with `GetAsk` and `ListAsks`; it sees the asks it
sent, never the event asks of its source (those are the ask-reader's).

| Limit | Value | When exceeded |
|---|---|---|
| Attachments per ask | 24 | Extra attachments dropped, warning |
| One file | 64 MiB | Upload refused (`InvalidArgument`) |
| Uploaded bytes no ask references yet | 1 GiB | Upload refused (`ResourceExhausted`); they expire after 24 hours |
| File uploads in flight | 2 | The next waits up to 30 s, then `503` with `Retry-After` |
| All attachment bytes in one ask | 256 MiB | Attachments after the limit dropped, warning |
| Files in one folder | 500 | Folder kept with the first 500 and `truncated = true` |
| Explainers per ask | 1 | Later explainers become evidence, warning |
| Attachment name | 1–96 of `A-Z a-z 0-9 . _ -`, starts with a letter or digit | Name sanitized, warning |

### Storage

Blobs live under `attachments/` in the directory the database lives in (storage
entry `decision-attachments`; see `DATA.md`, Where The Files Live), keyed by
SHA-256, through `api-core/blobstore`. Rows:

- `ask_attachments`: one row per attachment of an ask's current revision.
- `ask_attachment_entries`: folder entries.
- `attachment_blobs`: hash, size, the sender's declared media type (kept for the
  record, never served), first stored time, last touched time.

A revision keeps what it does not mention. When a request for a subject key meets
an open ask, the attachments it sends replace the open ask's attachment of the
same name (a sent explainer replaces the explainer) and are added when new; every
other current attachment stays. An attachment's name is its file's base name, so
sending a file with the same name replaces that one (`PATH=LABEL` changes only the
caption). `AskRequest.remove_attachments` drops the named attachments, and
`clear_attachments` drops every current one (the explainer included); both apply
before the sent ones, so a name that is removed and sent again is replaced. A name
the ask has no attachment of is reported (`attachment_remove_unknown`, advice) and
changes nothing. A resend
whose resulting attachments differ (names, labels, roles, sources or content)
counts as a text change: the open ask is updated in place with no new push. An
identical resend changes nothing. A replacement ask (changed options)
carries the merged packet over. A request with no open ask (the first send, or a
send after the operator dismissed or the requester withdrew the ask) carries only
what it sends. The merge is made inside the write transaction that reads the open ask's packet
and stores the result, so revisions that arrive at the same time apply one after
the other and none loses a part another added; hashing and blob lookups for what
a request sends happen before, outside the lock. An ask without a subject key keeps the attachments of its first
send; a retry under the same idempotency key does not change them.

A blob that no ask references is deleted by the maintenance sweep (every minute)
24 hours after it was last *touched*: uploaded, confirmed present by
`StatAttachmentBlobs`, or referenced by an ask. Touching on confirmation is what
makes "the Hub has it, do not upload" safe to rely on: the blob cannot be swept
between the sender's `Stat` and its `Ask`. The Hub's daily retention pass
(`Service.Retain`, run by the maintenance loop) deletes resolved asks 90 days after
their resolution together with their attachment rows, so their blobs become
unreferenced and the same sweep, in the same pass, reclaims them. A blob that
another ask still references stays. See `DATA.md`, Retention And Deletion.

The Hub ignores the sender's `size_bytes` and `media_type`: sizes come from its own
copy, and the media type is the shared classifier's alone. The sender's declaration
is never served, because the type decides how the operator's browser treats the
bytes (`text/html` would run them): a file the classifier cannot place is
`application/octet-stream`, and a sniff that finds HTML in a file with no known
extension is `text/plain`. Names are sanitized and kept unique (a repeat is
numbered before its extension, `notes-2.md`, as the CLI does), `source_path` must be
absolute, folder entries must be relative and free of `..`, and a request may not
claim `preview_kind` or `source_status`. Invalid attachments are dropped before the
24-attachment limit is applied, so they never take a valid one's place.

### Serving

`GET /api/v1/asks/{ask_id}/attachments/{name}` (REST exception
`browser_surface`), with `?path=<entry>` for a file inside a folder.

- Owner-only, the same authorization as `Answer` (`identity.Subject`: a Cloudflare
  Access login, the loopback owner in `personal_local`, or the owner's bearer or
  cookie), then the ask is read for that owner. This is stricter than `GetAsk`,
  which also admits the run that sent the ask and the service that raised it
  (`../reference/api-endpoints.md`, "Who reads an ask"): attachment
  bytes are the larger and more private payload. No verified owner is `401`.
  With an owner, every refusal (unknown ask, another owner's ask, unknown
  attachment, unknown or outside `?path=`, `?path=` on a file, a folder without
  `?path=`) is the same `404`, so the route reveals nothing about what exists.
- `http.ServeContent`, so byte ranges, `HEAD` and `If-None-Match` work (the `ETag`
  is `"sha256-<hash>"`, the content address).
- Headers: `X-Content-Type-Options: nosniff`, `Cache-Control: private, no-cache`,
  `Content-Security-Policy: sandbox; default-src 'none'; img-src data:; media-src data:; style-src 'unsafe-inline'`
  so that opening a file directly can never run it as a page of the Hub's origin, and
  `Content-Disposition: inline; filename=...` (a name that is not plain ASCII is sent
  as `filename*=utf-8''...`, RFC 2231).
- The URL names the ask and the attachment, not the bytes: a revision can attach
  different content under the same name. So a response is never reused without
  asking (`no-cache`), and the `ETag` makes the question cheap: an unchanged file
  answers `304` with no body. Clients need no `?v=` on the URL.

### Preview kind and drift

The Hub classifies every file and folder entry with the shared classifier
`packages/api-core/filekind` (the same one Web Console uses), so both apps pick
the same viewer for the same file. The kind is returned as `preview_kind`.

When the sender's host matches the Hub's host, `GetAsk` compares each
attachment's `source_path` with the stored copy (size, then hash) and reports
`source_status`: `fresh`, `changed`, `missing`, or `unknown`. `fresh` means the
content was hashed and matches; a size match alone is never `fresh`. `unknown`
means it could not be compared: another host, an unreadable path, or the hashing
budget of this read ran out. An `unknown` status carries no `source_checked_at`.
The Hub reads the source path only to compare it; it never serves it. Only a
only attachments with a retained source path are compared, and only for the
owner: a run or service reading an ask gets `unknown`, so the
check can never tell a caller whether a path exists or what a file contains.

For a folder, `missing` means the folder is gone, `changed` means any attached
file is missing, a different size, or different content (one difference settles it,
whatever the other files could be compared to), `fresh` means every file was
compared and matches, and `unknown` is any other outcome; files added to the source
since are not looked for. Hashing is bounded: at most 64 MiB newly hashed per
`GetAsk`, a source file is read for no more than 64 MiB, and results are remembered
per (path, size, modification time) so a repeat read costs nothing. The preview kind itself is fixed when
the ask is stored (the classifier may sniff the first 512 bytes of a file with an
unmapped extension), so a list or a folder of 500 files costs no reads.

## Answer form

The main options stay the decision's single answer key, so every existing
requester and event consumer keeps working. Questions add typed detail.

| Type | Operator sees | Response |
|---|---|---|
| `single_choice` | radio cards | `selected: [key]` |
| `multi_choice` | checkboxes | `selected: [keys...]` |
| `variant` | choices side by side, each showing its attachment (image, video, model) | `selected: [key]` |
| `section_review` | each section with Approve / Change / Drop and a note | `sections: [{section_key, verdict, note}]` |
| `text` | text box | `text` |
| anything else | prompt with a text box | `text` |

Limits: 12 questions, 12 choices per question, 40 sections per review, prompt
500 characters, help 1,000 characters, response text 4,000 characters. Smaller
fields: choice label 200 and detail 300, section title 200, summary 300 and
anchor 200, recommendation reason 300. All are counted in characters.

**Validation.** The Hub checks the form when the ask is sent. A question that
breaks a rule is left out and reported as `question_invalid` (the message names
the rule): an id or key outside `A-Z a-z 0-9 . _ : -` (1-64), a repeated id, key
or section key, a missing prompt or type, a choice question without choices, a
review without sections, a value over its limit, an invalid `unanswered_verdict`,
or a question past the twelfth. A recommendation that does not fit its question
(an unknown choice key, the wrong shape) is dropped alone and the question stays
(`question_invalid` with field `questions[i].recommended`). A question of an
unknown type is kept as a free-text question (`question_type_unknown`); its
choices and sections are not stored. A choice or section whose `attachment` is
not attached to the ask keeps its question and gets `question_attachment_unknown`.

**Answering.** `AnswerRequest.responses` are checked against the ask's
questions: an unknown or repeated question id, a choice that is not one of the
question's keys, more than one choice for a single-choice question, a verdict
other than approve, change or drop, or the wrong shape for the type (choices for
a text question, section verdicts for a choice question) is `InvalidArgument`
and the ask stays open. A response with no content is ignored. A required
question must be answered unless the main answer is `other`; it counts as
answered when a choice was selected, at least one section was given a verdict, or
text was written (a note alone does not answer a choice question). "Other" never
needs the form, but responses it carries are still checked. A late answer to a
defaulted ask is validated the same way.

**A review that follows a file.** A `section_review` question with
`sections_from` names an attachment (its name, or `name/path` for a file inside a
folder). The Hub reads that file's `##` and `###` headings into the sections (the
GitHub-style slug is the anchor, a key-safe form of it the key; at most 40, with an
advice warning when there are more; files up to 1 MiB) and reads them again every
time a request is sealed, so a revision that replaces the file changes the review
with it, though the revision never mentions the form. The sections a request sends
with `sections_from` are ignored. If the file is no longer attached, is a folder
without a path, or has no headings, the question is left out (`question_invalid`,
dropped) and the rest of the form stays. If the file cannot be read at the moment
(its blob is gone) the earlier sections are kept with advice. The operator's
verdicts are stored against the section keys the ask has when they answer, and
`recommended` is checked against them, so a recommendation naming a heading the new
plan lacks is dropped with a warning. The CLI no longer reads Markdown: it sends
`sections_from` and the Hub does the rest.

**A revised form.** If the requester revises the ask between the moment the operator
opens it and the moment the answer is stored, the answer was checked against a form
that is gone. The Hub stores nothing and `Answer` fails with `FailedPrecondition`
whose message starts `ask_changed:` and whose Connect error metadata
`Hub-Error-Reason` is `ask_changed` (a resolved ask is also `FailedPrecondition`
but has no such reason). The ask is still open: the client reloads it, keeps the
operator's draft where it still fits the new form, and lets them submit again.

Questions are part of the ask's text. A subject ask whose questions change is
updated in place without a new push, and an ask retried with the same event id
but a different form is a conflict, like any other changed term. The form is not
kept per revision: `GetAsk` shows the current revision's questions. A revision
keeps the open ask's questions it does not mention: a sent question takes the
place of the one with its id, a new one goes last, and
`AskRequest.clear_questions` drops them all first.

A section the operator leaves alone takes the question's `unanswered_verdict`
(`approve` when empty). The Hub stores a review as one verdict for every section,
in the review's order, so a requester reads a complete review and never applies
the default itself. A review with no verdict at all is no response. The summary
lists the sections that differ from the unanswered verdict (or carry a note) and
then "Other sections: approve".

### Notes on attachments

While viewing an attachment the operator can pin a note to it ("Hair is too
glossy" on `character.glb`). Notes travel as `AnswerRequest.attachment_notes`
(attachment name, folder entry path, note; at most 24 notes of 2,000
characters) and come back on `GetAsk`, `Wait`, the resolved event, and in the
summary. A note naming an attachment the ask does not have, a path on a file
attachment, or a path that is not an entry of the folder is `InvalidArgument`
(and the ask stays open); a note with no text is ignored, and a second note for
the same attachment and path is refused. A note on a folder without a path is a
note on the whole folder. Nothing is stored about a note before the operator
answers: it is checked against the attachments the ask holds at that moment, by
attachment name. A revision that replaces a file keeps its name, so a note on it
lands on the new file; a note on a file or folder entry that the revision removed
is refused, and the operator reloads the ask like for any revised form.

`recommended` on a question is an `AskQuestionResponse`; "Accept
recommendations" copies the main recommendation and every question
recommendation into the draft.

### What the requester gets back

`GetAsk`, `Wait`, and the `notification_hub.ask.resolved.v1` event carry
`responses`, `attachment_notes`, and `answer_summary`. The summary is plain text the requester can
paste into its own log:

```text
Answer: Approve with changes (approve)
Note: Keep the retention period short.
Layout: Card stack (cards)
Plan sections:
  - Retention: change — keep 30 days, not 90
  - Billing: drop
  - Other sections: approve
Notes on files:
  - character.glb: Hair is too glossy.
```

The summary exists only for an operator's answer: an ask still open, or closed
by default, expiry, withdrawal or dismissal, has none. Each response's label is
its question's prompt without the closing `?`, `:` or `.`, cut at 80 characters.
A choice prints as `Label (key)` (several joined by commas) and a note after
it as ` — note`; continuation lines of a multi-line note or text are indented
two spaces (four inside a list). A file note names its attachment, and a note on
a folder entry names `folder/path`.

## Context requests

"I need more context" keeps the ask open and sends the operator's question to
the requester.

**Who can be asked.** The operator is never left waiting on a sender that cannot
reply, so the Hub offers the question only for an ask whose requester declared it
answers: `AskRequest.accepts_context_requests` for a direct ask, and
`accepts_context_requests` in the `decision_requested` payload for an event
(`operatordecision.Facts.AcceptsContextRequests`). The CLI `ask` declares it whenever
`--subject-key` is given; `--no-context-requests` opts out. `Ask.can_request_context` is
true while the ask is open and its current revision declared it; the UI shows the
button only then, and otherwise tells the operator to decide with what the ask
shows (answering is always possible). `RequestContext` on an undeclared ask is
`FailedPrecondition` with `Hub-Error-Reason: context_not_accepted`, and nothing
is stored or published. The answer to a question is a revision under the same
subject key, so a declaration without a `subject_key` is dropped with the
`context_requests_need_subject_key` warning. A revision restates the declaration
like the rest of the packet: leaving it out updates the ask in place (no push)
and closes the button; sending it again reopens it.

The alternative, keeping the button for every ask and recording the question
with an honest "this sender may not reply", was rejected: for a producer with no
code that reads the question, the operator would wait for an answer that cannot
come. A declared producer can still be slow or down; the request stays visible as
unaddressed with its time, and the operator can answer at any point.

1. The UI calls `RequestContext(ask_id, question)`. The Hub stores an
   `AskContextRequest` and publishes `notification_hub.ask.context_requested.v1`
   (through the same outbox as resolutions) with the ask's source,
   requester_ref, subject_key, correlation, and the question.
2. `Wait(..., return_on_context_request: true)` returns while the ask is open
   when an unaddressed request exists (`state` stays `pending`; `context_request`
   is the oldest one). Without the flag a request never ends a wait. Until the
   requester revises the ask, every flagged `Wait` returns immediately, so a
   caller that does not revise should drop the flag. The CLI sets it by default.
   `notification-hub conversations wait` exits with status **3** in that case,
   prints the operator's question on stdout, and prints the revise command on
   stderr (`--until-answer` turns this off).
3. The requester answers the question with a **context response**: the same `ask`
   with the same `--subject-key` and `--context-response '<answer>'`
   (`AskRequest.context_response`, 1-4,000 characters). The answer is a distinct
   thing, not a part of the ask: it is not written into the big picture, the ask's
   terms do not change, and the requester does not resend any text. It can stand
   alone, or ride with any change the answer needs (a revised `--question`,
   `--attach`, `--remove-attachment`, `--form`, `--remove-question`); everything the
   request leaves out stays. The Hub stores the response with each question that
   was open (one answer covers every question open when it was sent) and marks them
   addressed, so the packet cover shows "Your question: ... / Answer: ..." where the
   big picture is, and the next `wait` does not return the old question
   (`ask --and-wait` after an answer ends at its own timeout, not at once). The
   questions stay: every round of question and answer is kept and readable in
   `AskContextRequest.response` (`get` lists them oldest first).

   A request with only a context response changes no term, so the ask gets no new
   revision; its outcome is `updated`, and `addressed_revision` is the ask's current
   revision. When the request does change the ask, `addressed_revision` is the
   revision it became. A revision that sends no response still addresses the
   questions, with an empty `response`; a re-send with identical text and no response
   only re-asserts the ask and addresses nothing. A response sent while the operator
   has no open question is not stored: the Hub applies the rest of the request and
   returns a `context_response_unused` warning (severity `dropped`).

   The revision uses a **new idempotency key** (each revision's event id is the ask's
   current `requester_ref`); it updates the ask in place and sends no new push.
   Reusing the key of the current revision with different text is an identity
   conflict (`AlreadyExists`) and changes nothing, and redelivering an earlier
   revision's key returns the ask unchanged, so a retried answer is stored once.
   A replacement (changed options) is a new ask, and the old ask is closed: the
   replacement answers the questions open on the old ask, and the Hub copies the
   old ask's questions and their answers to the new ask, so the operator finds them
   on the ask they now see.

Only the ask's recipient may ask, only while the ask is open. A question is 1-2,000
characters. Repeating a question that is still unaddressed returns the existing
request (a double tap is one request), and an ask holds at most 5 unaddressed
requests; the sixth is `ResourceExhausted` until the requester revises the ask.
The event is published within moments (an in-process drain after the request,
with the one-minute sweep as the retry), and carries the ask's `requester_ref` and
correlation as they are when it is published.

## Packet check

`AskResponse.warnings` returns advice and drops; `CheckAskResponse.warnings` (the
dry run) also returns `blocking`, the findings that would refuse an agent's ask.
Apart from those (next section), a packet part never blocks an ask; the cover
fields' limits (big picture 4,000 characters, technical details 12,000,
option label 200, 2 to 8 options, a default only with `reversible`) are validation
and do refuse it with an error, and nothing is sent.

On a revision the Hub also marks which findings are news. A finding that the open
ask already had, word for word, about a part the revision did not touch (it left
that field unsaid, and sent no explainer) comes back with `carried_over: true`. The
CLI folds those into one line (`2 earlier warnings still apply (same as before)`)
and prints the rest in full; `--json` lists every finding with the flag. Findings
about what the request itself sent (an attachment the Hub left out, a question it
dropped) are never carried over, and neither are those of a first send. Checks:

| Code | Rule |
|---|---|
| `internal_id_in_question` | A field the operator reads (the question, the big picture, the recommendation reason or an option label; `field` names which) contains internal-looking IDs (`S-14`, `O7`, `F49-MOR-CHECKOUT`, `BAS-065`, dated slugs such as `run-20261114`, hashes of 12 or more hex digits, UUIDs) or the question starts with a bracketed tag (`[BAS] ...`). Ordinary text is left alone: years, dates, times, money, versions (`v2`, `V2.1`), quarters (`Q3`), Apple chips (`M1`), latency percentiles (`P95`), and standards such as `UTF-8`, `SHA-256` and `GPT-4`. |
| `path_in_question` | The question or big picture contains a file path (under a host root such as `/home` or `/tmp`, `~/`, `./`, a Windows drive, any absolute path with a directory and a file extension, or a repository file such as `api/main.go`); attach the file and name it in words. URLs and API routes such as `/api/v1/asks` are not paths. |
| `long_question` | The question is longer than 200 characters. |
| `missing_big_picture` | No big picture. |
| `missing_recommendation_reason` | A recommendation without a reason. Blocking for an agent's ask. |
| `local_context_url` | `context_url` is a local path or loopback URL. |
| `unattached_path` | `technical_details` names a local file that no attachment's `source_path` covers (the path itself or a file inside an attached folder). Paths in the question or big picture are `path_in_question`, attached or not. |

The cover rules (the first five rows) are api-core's `operatordecision.CheckCover`. A
scenario that raises decisions asserts `operatordecision.CoverProblems(facts)` (or
`PayloadCoverProblems` on the published event) is empty in its tests, so a producer
never sends a cover the Hub would advise against.
| `attachment_blob_missing` | Dropped: the referenced blob is not stored (for a folder, only the entries whose blobs are missing). |
| `attachment_limit` | Dropped: too many attachments (counted after invalid ones are dropped) or too many bytes. |
| `attachment_invalid` | Dropped: unknown kind, a file without a valid `sha256`, or a folder with no valid entry. Advice for an unknown role (listed as evidence). |
| `attachment_name_changed` | Kept: the name was sanitized or made unique. |
| `attachment_source_ignored` | Kept: `source_path` was not absolute, so it was ignored. |
| `attachment_folder_truncated` | Kept: more than 500 files, the first 500 stay. |
| `attachment_explainer_limit` | Kept as evidence: a second explainer, or a folder marked explainer. |
| `attachment_remove_unknown` | Advice: `remove_attachments` names an attachment the ask does not have; nothing was removed for it. |
| `question_invalid` | Dropped: a question breaks a rule (the message says which). |
| `context_requests_need_subject_key` | Dropped: `accepts_context_requests` without a `subject_key`. |
| `question_type_unknown` | Kept; shown as free text. |
| `question_attachment_unknown` | Advice: a choice or section names an attachment the ask does not have; the question is kept. |
| `explainer_missing` | Advice: the ask has no explainer; it is shown with the Hub's page (below). |
| `packet_missing` | Advice: a direct ask names no `packet` (the brief that carries out the answer). |
| `explainer_*` | Explainer rules below. |

### Decision quality

These rules judge whether the decision is worth the operator's attention, not
only its shape. Three refuse an agent's ask (severity `blocking`); the rest
advise. A scenario's `*.decision_requested.v1` decision only gets advice from
all of them. A revision is refused only for a finding about a part it changed:
one carried over from the open ask stays advice.

| Code | Severity for an agent | Rule |
|---|---|---|
| `missing_recommendation_reason` | blocking | A recommendation with no reason (in the table above). |
| `explainer_restates` | blocking | The page adds under 40 content words to the cover (question, big picture, recommendation reason, option keys and labels), or under a quarter of its words and under 80. Common words and page furniture do not count; each image, video, SVG, canvas or `[data-hub-compare]` counts as 30 words. The fix names the kit blocks and the skill's starters. |
| `page_unreadable` | blocking | Some text is below 3:1 contrast against its background, in a scheme the frame shows the page in. The Hub runs the page's CSS cascade over the kit (selectors, specificity, inheritance, `var()` and `color-mix()`) in light and dark; text that is hidden, or positioned over something with no opaque background, is not judged. The message names up to three texts with their colours and the fix: colour with `--hub-*` tokens, and set a background's text colour in the same rule. |
| `command_hidden` | advice | An answer has the operator run something (a terminal, "you will run", "I'll run", `sudo`), and neither the page nor the question or big picture shows the command. On the page, a command counts in `.hub-command`, a `<pre>`, or a `<code>`/`<kbd>`/`<samp>` that is the whole of its block; one inside a sentence does not. The fix is the kit's `.hub-command` block (`operatordecision.Page.Command`) on the first screen. `technical_details` does not count. |
| `page_no_picture` | advice | The page shows nothing: no `img`, `svg`, `canvas` or `video`, no `role="img"` or `role="figure"` element, no table of two or more rows, and none of the kit's decision blocks (`.hub-proportion`, `.hub-groups`, `.hub-matrix`, `.hub-timeline`, `.hub-compare`, `.hub-recurrence`, `.hub-command`). Option cards, callouts and chips arrange words and do not count. Not reported when `explainer_restates` is. |
| `page_picture_uncropped` | advice | Two or more images are shown whole: no `height` or `max-height` length, `aspect-ratio` or `height` attribute bounds them, so each full screenshot takes a phone screen. The fix crops each to the part the decision is about. |
| `recurring_subject` | advice | The third ask from one sender within 14 days on the same subject key or a similar question (numbers ignored), and no option stops the repeats (a standing rule, a policy, "from now on", a root-cause fix). The fix names the earlier dates and the kit's `.hub-recurrence` block. |
| `related_open_decision` | advice | A similar question is open. A verified producer sees identifiers only for asks in its normal read audience; an owner can see matches across sources. A verified agent run may instead receive a content-free signal for another conversation of the same owner. Anonymous requests receive no advice. The advisory check considers at most 200 recent open decisions and reports up to three visible matches. |

For a new direct decision by a verified run, the Hub also checks all open direct
decisions of the same owner and recipient inside the write transaction. A similar
question from another conversation causes a generic failed-precondition response
before an ask or notification is stored. The sender should have the owner review
the existing decision and provide its subject key for a revision, or, after
determining the choices really are distinct, retry with `--allow-related`
(`allow_related` in the API). This is a sender's explicit judgment, not a grant
to read or revise another ask. Similarity does not merge decisions, transfer an
answer, or change an approval's scope. Revisions of an existing subject and
event-sourced requests follow their existing lifecycle rules.

A refused ask returns `FailedPrecondition` with the header
`Hub-Error-Reason: ask_refused` and the message `refused, nothing was sent:
code (field): message | ... | Fix each one, check with --dry-run, then send
again.` Nothing is stored, the open ask a revision targeted is left as it was, and
an idempotent retry of the same request is refused the same way. The CLI prints
one finding per line; a dry run lists them first under "Would refuse the ask".

**Theme.** The frame shows a page in a scheme its own colours work in.
`theme.ts` (UI) and `packages/api-core/pagescheme` (the API and the page tools) read the page's CSS the same way:
a declared `color-scheme`, then its own dark styles (`prefers-color-scheme`,
`[data-theme=dark]`, a theme-change script) mean it handles both; otherwise
literal root or body colours, then unpaired literal colours, decide light or
dark. A one-scheme page gets that `data-theme` and `data-hub-scheme`, the bridge
stops passing theme changes to it, and the frame's `color-scheme` matches, so the
browser's own defaults agree with the page. The kit colours `html`, not `body`,
so a page that sets its root colours replaces the kit's for all its text.

Feature-loss redesign decisions use the same packet check. Their versioned JSON card and
phone-rendered page are checked together for evidence, choices, recommendation, exact
finding/build identity, and answer actions. See [Feature loss decision cards](FEATURE_DECISION_CARDS.md).

## HTML explainers

An explainer is an attachment with `role = ROLE_EXPLAINER` and an `.html`
file. The Hub shows it at the top of the decision in a sealed frame. Every
decision is expected to have one (`explainer_missing` advises when it does not).

**No decision is bare.** An ask with no explainer is read with the Hub's page,
`decision-summary.html`, made when the ask is read by
`operatordecision.CoverPage` from the cover as it is now: question, big picture,
option cards with the recommendation and reason, how to answer, "What happens
when you approve" (with the packet's brief when attached) and what happens with
no answer. It is never stored; the first page the sender attaches takes its
place.

**Packet.** `AskRequest.packet` (`operatordecision.Facts.Packet`, CLI
`--packet`) is the absolute path of the ready-to-run brief, normally
`work/<ID>/BRIEF.md` (`docs/agent-system/LEVELS.md`). It is shown by `get`,
returned by `wait` with the executor's start message, and carried in the
resolution event as `packet`. The CLI attaches the brief as evidence.

**From scenario code.** A `*.decision_requested.v1` event carries no uploaded
file, so a producer sends its page inline as `explainer_html`
(`operatordecision.Facts.Explainer`, built with `operatordecision.NewPage`; at most
720 KiB as JSON encodes it). The Hub stores it as a blob and attaches it as the
explainer `decision.html`; a re-send under the same subject key replaces it by
that name. A page the store refuses is left out with a `dropped` warning and the
ask is still created.

A decision event can also carry an answer form, `questions`
(`operatordecision.Facts.Questions`: `single_choice`, `multi_choice` or `text`,
naming no file). The Hub checks it as it checks an agent's `--form`, so a broken
question is dropped with a warning. A page's `.hub-groups` block fills the
`multi_choice` question it names. The answers return as `responses` on
`notification_hub.ask.resolved.v1` and on the ask the raising scenario reads back.
`operatordecision.VerifiedResolution` takes them from that readback
(`DecisionResolution.Response(id)`).

**Sealing.** The UI fetches the page and renders it in an
`<iframe sandbox="allow-scripts">` (no `allow-same-origin`, so the page has an
opaque origin and no access to the Hub's cookies, storage, or API) through
`srcdoc`. Before rendering, the Hub injects, at the start of `<head>`:

1. `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data: blob:; media-src data: blob:; font-src data:; style-src 'unsafe-inline'; script-src 'unsafe-inline'">`.
   A page cannot loosen it: every policy on a page is enforced.
2. A one-rule baseline, `@layer hub-base { html, body { margin: 0; } }`, always
   present, so the browser's 8px body margin never stacks on the Hub's frame.
3. The starter kit stylesheet inside `@layer hub-kit`, so any style the page
   writes itself wins. `<meta name="hub-kit" content="off">` leaves it out.
4. The bridge script (below) and the kit behaviours.

**Layout.** On a phone the frame runs the full width of the screen under one
caption line ("Agent's page, sealed"), so the page's own text sits at the same
gutter as the decision above it. The kit's `body` padding (`0.75rem 1rem 1rem`)
is that one gutter; a page that pads its own body or wraps itself in a margin
stacks a second one. Beside the answer panel on a wide screen the same page
sits in a bordered panel.

**Attachments by name.** `src`, `href`, and `poster` attributes whose value is
an attachment name (or `name/entry/path` inside a folder) are replaced with
`data:` URLs, up to 12 MiB per file and 40 MiB per page. A larger file keeps a
placeholder that opens the Hub viewer when tapped.

**Bridge.** The page talks to the Hub only through `postMessage`. The injected
script exposes:

| Call | Effect |
|---|---|
| `hub.open(name, path?)` | Opens an attachment in the Hub viewer. Elements with `data-hub-open="name"` do this on tap. |
| `hub.prefill(questionId, response)` | Fills one form question in the draft (`response` is an `AskQuestionResponse` in JSON form, or a choice key string). Also `hub.prefill("answer", key)` for the main option. The operator still submits. |
| `hub.theme` | `"light"` or `"dark"`; the page root also gets `data-theme`. A page locked to one scheme (`data-hub-scheme`) keeps it. |
| `hub.note(key, label)` | Opens a Hub-owned note sheet. Saved notes and uploaded file references are sent under `changes.notes[key]`. |
| `hub.openAsk(id)` | Opens another Hub ask at `/asks/<id>`. |
| `hub.context.asks[id]` | Live `{state, askedAt, answeredAt, question}` for up to 20 distinct ids named in page data `asks[]`; inaccessible ids have `state: "unknown"`. `hub.context.asksTruncated` reports when additional ids were omitted. |
| `await hub.asset("folder-name/relative/path")` | Resolves an existing entry in an attached folder to a data URL (up to 12 MiB); invalid, missing or larger entries return `null`. |

Height is reported automatically. The Hub ignores any other message.

**Rules the packet check tests** (advice; the decision-quality rules above
also judge the page):

| Code | Rule |
|---|---|
| `explainer_not_html` | The explainer is a folder or not an HTML file (judged by its `.html`/`.htm` name, its media type, or its first bytes). |
| `explainer_too_large` | Over 2 MiB of HTML. |
| `explainer_network` | Loads resources from `http:`/`https:`/`//`: `src`, `srcset` and `poster` attributes, `<link href>`, CSS `url()` and `@import`, and `fetch()` of a literal address. Anchors (`<a href>`) and `xmlns` declarations are not resource loads and are not flagged. |
| `explainer_no_viewport` | No `<meta name="viewport">`; it may not fit a phone. |
| `explainer_form` | Contains `<form>`, `<input>`, `<select>`, or `<textarea>` in its markup (comments and script text are not scanned); answers belong in the Hub form. |
| `explainer_unknown_attachment` | Names a file that is not attached: a bare `src`, `href` or `poster` value, `data-hub-open`, or `hub.open("name")`. Fragments, `data:`/`mailto:`/other schemes and network URLs are not attachment names. |

Writing guidance, the starter kit's building blocks, and worked examples are in
`skills/notification-hub/references/html-explainers.md`.

## Served-feature keep/drop cards

UI Health feature-loss cards use an attached `research-feature-decision/v2` JSON model,
a rendered HTML explainer, and a matching keep/drop answer form. The packet checker verifies
the card and page agree on the feature, finding version, measured build pair, embedded
evidence and answer choices; it returns actionable `feature_card_*` warnings when a required
part is missing. The one-command CLI path and the later proof-bound disposition flow are
documented in [`FEATURE_DECISION_CARDS.md`](FEATURE_DECISION_CARDS.md) and
[`ui-health's preservation guide`](../../../ui-health/docs/concepts/FEATURE-PRESERVATION.md).

For EX12 capture comparisons and region notes, see [`FEATURE_REGION_FEEDBACK.md`](FEATURE_REGION_FEEDBACK.md).
