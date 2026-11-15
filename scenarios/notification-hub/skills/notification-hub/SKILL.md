---
name: "notification-hub"
description: "Ask the operator for a decision, or tell them something, through Notification Hub so that a reader with none of your context can understand it and answer from a phone in under a minute."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  revision: 7
  modes: ["tools"]
  tags: ["notification-hub", "decisions", "asks", "operator", "notifications", "writing", "operator-legibility"]
  requires:
    scenarios: ["notification-hub"]
    commands: ["notification-hub conversations ask"]
  origin:
    kind: "authored"
---

## Write decisions the operator can answer cold

The operator reads your decision on a phone, between unrelated tasks, with
none of your context: not your chat, not your files, not your IDs, not your
plan. Every word you send must make sense to that reader. If the operator has
to open a file, recall an ID, or guess what "the prepared kit" is, the decision
waits — and so does your work.

This skill is how agents apply the
[operator legibility standard](../../../../docs/agent-system/SWARM_MANAGER_WORK.md#operator-legibility)
to Notification Hub. Scope: decisions (`conversations ask`), revisions and
withdrawals of them, and information-only notifications. Out of scope:
answering asks (operator only), recipients, channels and quiet hours.

## Choose the shape before you write

Decide what the operator will look at before you write a word. A page of text is the right shape for most decisions. It is the wrong shape when the choice rests on a comparison, and then an HTML **explainer** (one page, shown above the answer buttons) is faster for the operator to read and for you to be right about.

| The decision is... | Send |
|---|---|
| A yes or no, or one option with one line of context | Text only (`--yes-no`). Do not build a page. |
| A choice among 3 or more options, or 2 options whose differences are numbers (cost, time, risk, effort) | **An explainer** with a trade-offs table or side-by-side cards. |
| A plan, rollout or schedule to approve | **An explainer** with a timeline. |
| A change to approve, where seeing it matters | **An explainer** with a before/after slider. |
| Images, video, a 3D model or a long file to look at | **An explainer** with figures (or `--attach` alone for one file). |
| A document to approve part by part | `--attach plan.md` and a `--form` review of its headings. |
| A fact to confirm or a short text to supply | Text, plus one `--form` question when the answer has parts. |

To send an explainer, copy the starter page that matches the decision, replace its words and numbers with yours, and add it to the ask. You write no CSS:

```bash
cp "$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/plan-tradeoffs.html" explainer.html
# edit explainer.html: your options, your numbers, your recommendation
notification-hub conversations ask --subject-key K ... --explainer explainer.html
```

| Starter page (in `examples/`) | Use for |
|---|---|
| `plan-tradeoffs.html` | Options against a few criteria (a table that becomes cards on a phone) |
| `choose-between-drafts.html` | Two or three finished options side by side, with buttons that fill in the answer |
| `rollout-timeline.html` | A plan over time, with a size comparison and a stop rule |
| `compare-before-after.html` | A change shown as before and after |
| `media-walkthrough.html` | Video, photos or a 3D model, with steps |

The question, big picture, options and recommendation still carry the whole decision in plain text: the push and the decision list never show the page. The page adds the comparison. Read `references/html-explainers.md` once for the rules. `ask --dry-run` runs the explainer through the packet check, and it suggests an explainer when your text looks like one of the shapes above.

## Send a decision in one command

The smallest useful ask is three flags:

```bash
notification-hub conversations ask --yes-no \
  --subject-key backup-elements-blocked \
  --question 'Retry tonight'"'"'s backup to the Elements drive now?'
# ask_id=7f3a... state=pending outcome=created idempotency_key=ask-9c1e... next: notification-hub conversations wait 7f3a...
```

Everything else has a default: the ask expires in 24h (`--expires`), sensitivity
`private`, urgency `normal`, and an idempotency key that is printed. Add what the
decision needs: your own `--option key=label` (2 to 8) instead of `--yes-no`,
`--recommend`, `--big-picture`, `--technical-details`, files, an explainer, an
answer form. Or put the cover in one markdown file and send `--body-file ask.md`
(`-` reads standard input): the first paragraph is the question, `## Background`
is the big picture, `## Details` is the technical details. `--dry-run` runs the
packet check and shows the notification the operator would get, and the sender they would see with where that name came from (your `--from`, your session label, kept from the open ask, or named by the Hub; `--json`: `sender`, `sender_source`). It stores
nothing, uploads no file and creates no ask, so a real send afterwards conflicts
with nothing. To run the check, the CLI does send the Hub each attachment's hash
and size, plus the content of the explainer page and of any Markdown file a
review follows; the Hub reads those for that one check and keeps none of them.
It cannot be combined with `--and-wait`.

Words used in this skill and in `--help`: **options** are what the operator
picks from (`--option`); **answer** is what they reply; **form questions** are
the typed extras under the options, and their **choices** are inside the form
file; **expires** is when an unanswered ask closes; **timeout** is how long
your command waits.

Three rules cover almost everything else:

1. **Always set `--subject-key`**, a stable name for the situation
   (`backup-elements-blocked`). It makes the ask one living thing instead of a
   pile of copies, and it is how you find it again after a restart.
2. **Revise with the same `--subject-key`; never ask again.** To fix wording or
   add evidence, run `ask --subject-key <key>` with only what changes, for
   example `--technical-details '...'`. To answer the operator's "I need more
   context" question, use `--context-response` (see "When the operator asks for
   more context"). Whatever you leave
   out stays as the open ask has it: the question, the options, the
   recommendation, the texts, the attachments, the explainer and the form, and
   the CLI prints what it kept on stderr. A text flag replaces that text whole
   (resend the old words with your addition; `--big-picture ''` clears it). The
   open ask updates in place, with no new push, whatever you change except the
   **answer menu**: the option keys, the recommended key (`--recommend`), the
   default, `--reversible`, the urgency and the sensitivity label. A new menu
   replaces the ask with a new one and pushes once. Option labels, the
   recommendation reason and every text are not menu, so they update in place. Asking
   again under a new subject key opens a second ask and leaves the first one open.
3. **Wait by exit status.** `conversations wait <ask-id>` (the id is the first
   argument or `--ask-id`, and `ask` prints it as `ask_id=...`), or
   `wait --subject-key <key>`, or `ask --and-wait`, blocks until the answer or
   until `--timeout` (default 30 minutes) and exits with one of these statuses.
   `ask --and-wait` prints the send's result first and the wait's result after
   it; a send that failed exits 1 before any wait, with no `ask_id=` line:

| Status | Meaning | You do |
|---|---|---|
| 0 | An answer is available: the operator answered, or the default applied | Read the printed answer summary and act |
| 3 | The operator asked for more context; the ask is still open | Answer with `ask --subject-key <key> --context-response '...'` (see below), then wait again |
| 4 | Your wait timed out (`--timeout`); the ask is still open and has not expired | Wait again, or carry on and check later with `get` |
| 5 | The ask closed without an answer: withdrawn, dismissed or expired. By id, a revision with new options also ends here, as withdrawn with the reason `replaced` | Read the reason; ask again only if the need remains |
| 1 | The command itself failed: bad flags, the Hub unreachable, or a refused read | Read the error on stderr |

A wait carries on across a dropped connection and the Hub's ten-minute limit on a
single wait; it ends only with one of the statuses above. By `--subject-key` it
follows a replacement ask (new options) instead of reporting status 5.

**Choosing `--timeout`.** An ask lives for `--expires` (24h by default) and a
person answers when they can, so a wait that ends with status 4 is normal, not a
failure. Pick a `--timeout` that matches how long your work can idle, and wait by
subject key in a loop that treats 4 as "still open":

```bash
set -o pipefail
while true; do
  notification-hub conversations wait --subject-key backup-elements-blocked --timeout 1h --json > answer.json
  case $? in
    0) break ;;                                  # answered: read answer.json
    3) echo "the operator asked a question: answer it with --context-response" >&2; break ;;
    4) continue ;;                               # still open: wait again
    *) break ;;                                  # 5: closed without an answer; 1: the command failed
  esac
done
```

Write the output to a file, or use `set -o pipefail`, when you pipe `wait` into
`jq`: without it the pipeline's status is `jq`'s and the exit status above is lost.

What a program can rely on. The exact documents, one real example per status,
are in `references/cli-results.md` (generated from the CLI's own output, so it
cannot drift):

| Command | Last line of output | `--json` |
|---|---|---|
| `ask` | `ask_id=... state=... outcome=... [idempotency_key=...] [next: ...]` | `{ask_id, state, outcome, warnings[]}`, every field present; `outcome` is one of the six in "Revise, withdraw and check" |
| `wait` | `ask_id=... state=... [answer=...]` | The same document on exit 0, 3, 4 and 5: `{ask_id, state, answer, note, reason, responses[], answer_summary, attachment_notes[], context_request}`, a field with no value left out |
| `get` | the whole ask in text | `{ask: {...}}`; a field with no value is left out, so read a missing count as 0 |
| `withdraw` | `Withdrew ask <id>.`, then what changed and the next command | `{ask_id, state}`; `--reason` is required |
| `ask --dry-run` | `Dry run: ...`, then the notification the operator would get | `{revises_ask_id, push_title, push_body, warnings[]}`, a field with no value left out |
| `ask --and-wait` | the `ask` output, then the `wait` output; the last line is the wait's | two documents in a row: the `ask` document, then the `wait` document |

Where to read things in `wait --json`:

| You want | Path |
|---|---|
| The operator's choice | `.answer` (an option key); `.note` is their free text; `.state` is `answered` or `defaulted` |
| A form answer | `.responses[]` entry with that `question_id`: `.selected[]` (single_choice, multi_choice, variant), `.text` (text), `.sections[]` of `{section_key, verdict, note}` (section_review) |
| Everything as plain text | `.answer_summary` |
| Their question (exit 3) | `.context_request.question` |
| Why it closed (exit 5, 4) | `.reason`; `.state` is `withdrawn`, `dismissed`, `expired` or still `pending` |

For example `wait --subject-key K --json | jq -r '.responses[]? | select(.question_id=="layout") | .selected[]'`.

`ask --and-wait --json` prints two documents one after the other (the send's, then
the wait's), so read the second one with `jq -s` (slurp, which puts both in an
array). Save the output to a file first: the exit status is the wait's, and a pipe
would replace it with `jq`'s.

```bash
notification-hub conversations ask --subject-key K --yes-no --question '...' --and-wait --json > out.json
status=$?                          # 0 answered, 3 more context, 4 timed out, 5 closed
jq -s '.[1].answer' out.json       # the option key, for example "no"
jq -s '.[0].ask_id' out.json       # the id, from the send's document
```

Where the answer sits depends on the command, not on the data. `wait --json` prints
the wait result, so the answer is `.answer`. `get --json` prints the Hub's whole ask
under one `ask` key, so the same answer is `.ask.answer`.
Use `.responses[]?` (with the `?`): `responses` is left out of the JSON when there is
nothing in it, and `.responses[]` alone then fails. It holds an entry only for a form
question the operator answered: a question they left alone (an optional one) has no
entry, and an ask whose default applied has no `responses` at all, only `.answer`.
The main answer is `.answer`, never a `.responses[]` entry. All JSON uses the
field names of the Hub's API in snake_case. Warnings and guidance go to stderr,
so stdout stays one JSON document.

### Recover your asks after a restart

If your conversation was lost, the asks you sent are still open. Re-derive the
subject key (it is a name for the situation, so you can rebuild it) and ask the
Hub where it stands:

```bash
notification-hub conversations get --subject-key backup-elements-blocked   # state, answer, open context requests
notification-hub conversations wait --subject-key backup-elements-blocked  # resume waiting; follows a replacement ask
```

If you cannot rebuild the subject key, list what is open and search by topic words:
`notification-hub conversations list --open --query 'backup'`.

### Sort and filter a list

`list` shows newest first. Choose another order with `--sort`, and narrow it with
filters; every sort takes every filter, filters combine with AND, and several
values of one filter with OR.

| Flag | Values |
|---|---|
| `--sort` | `newest` (default), `oldest`, `attention`, `due` |
| `--state` | `all` (default), `awaiting` (pending or escalated), `pending`, `escalated`, `answered`, `defaulted`, `expired`, `withdrawn`, `dismissed`. `--open` is `--state awaiting`; combining it with another state is an error. |
| `--urgency` | `low`, `normal`, `high`, `critical`; repeat the flag or separate with commas (`--urgency high,critical`) |
| `--sender` | A sender exactly as the list prints it; repeat for several (`--sender 'Research Desk' --sender 'Agent run'`). Never split on commas. |
| `--query` | A literal substring of the question, source, options, answer, note or requester |

`attention` is what needs the operator first: escalated asks, then pending asks
closing within 6 hours (an expired deadline not yet processed counts; the
nearest close first), then urgency (critical, high, normal, low), then the
earliest deadline (an ask with none last), then newest. The window is the Hub's
(`closing_window_seconds` in `--json`), and each ask in it carries
`closing_soon: true`. `due` is the earliest deadline first, then urgency, then newest. Both
tie-break to a fixed order, so a page boundary never repeats or skips an ask. A
closed ask is never escalated or closing, so for it `attention` is urgency, deadline, newest.

```text
$ notification-hub conversations list --open --sort attention --urgency high,critical
Summary:
  2 asks
  By urgency: critical 1, high 1. By sender: Research Desk 1, Storage manager 1
Asks:
  a1 [escalated] Free the disk? | from Storage manager, critical, due 2026-11-14T18:00:00Z
  a3 [pending] Rename the project? | from Research Desk, high
```

Each row ends with the sender, an urgency other than `normal`, and the deadline of
an ask that still waits. The summary's counts are the choices the filters offer: a
count applies every filter except its own, so picking `high` leaves the sender
counts those of high asks and the urgency counts unchanged. With `--json` they are
`facets.urgencies` and `facets.senders`, each a list of `{value, count}` (at most 20
senders, the ones you chose always included), and each ask has `sender`. More
asks than `--limit` (default 50, at most 200) give a `next_cursor` and a hint:
repeat the command with `--cursor <it>`.

Who can reach an ask: by **ask id**, the runs of the conversation that sent it
(your run, and the runs recovered from it, spawned by it or re-attached to the
same harness session), so a second run of the same conversation can `get`,
`wait` and `withdraw` by id. By **exact `--subject-key`**, any run of the same
owner account. `list` shows only your conversation's asks, and the Hub never
lists other agents' asks to you. A shell with no verified identity is refused.

Before you open a new ask, list what is open. It is the same situation, and you
revise it, when the decision is the same decision with new facts or wording. It
is a different situation, and gets its own subject key, when answering one would not
answer the other.

## What the operator sees

| Surface | Shows | Consequence for you |
|---|---|---|
| Phone push | "Decision required", your **question**, your **recommended** option | The question alone must tell the operator what is being decided. |
| Ask page | question, **big picture**, options (recommended first), recommendation reason, **technical details**, context link | Big picture explains; technical details are for the agent that picks it up later. |
| Decision list | the question, cut short | Put the decision itself in the first words, not a tag or ID. |

## Field rules (SBAR)

Structure every ask as **SBAR** (Situation, Background, Assessment,
Recommendation), the hand-off format built for a reader who walks in cold.

| SBAR | Flag | Write | Never put here |
|---|---|---|---|
| Situation | `--question` | One sentence the operator can answer from the push alone. Name the choice and the thing it affects in everyday words. Aim for under 200 characters. | Internal IDs or codes (`O7`, `S-14`, `DB1`, `CF3`), run or commission names, bracketed tags, file paths, "approve the prepared X" without saying what X is. |
| Background | `--big-picture` | 2–6 short sentences: what this is about and why it matters now; why only the operator can decide; what happens after each answer; what happens if nobody answers. Define any term you must use in the same sentence. Leaving it out only earns `missing_big_picture` advice; the ask is still sent. | A copy of the option list, paths, IDs, raw logs. |
| Assessment | `--technical-details` | Evidence, IDs, commands, file paths, packet locations, links to the run — everything the next agent needs to act on the answer. This is the only field where internal names belong. | Anything the operator needs in order to choose. |
| Options | `--option key=label` (2 to 8) | Each label states what happens if chosen, as the owner's next action: `check=Yes — submit the eligibility check and run one test purchase`. Keys are short words (`check`, `skip`, `later`). Add a defer option only when deferring is a real answer. Use `--yes-no` only when the question itself says what yes causes (the labels are just Yes and No); otherwise write the labels. | `yes`/`no` keys with vague labels; a key named `other` (the Hub adds its own free-text "Other"). |
| Recommendation | `--recommend <key>` and `--recommend-reason` | Recommend whenever you have a view. The reason is one plain sentence about benefit or risk. | "Recommended." with no reason. |

Set the remaining flags this way:

| Flag | Rule |
|---|---|
| `--subject-key` | Always set it to a stable name for the situation (`release-payments-check`). It lets you revise the ask in place, find it again after a restart, and wait on it by name. It also tells the Hub you will answer the operator's "I need more context" question (the question button appears only then; `--no-context-requests` turns it off when nothing will read the question). |
| `--expires` | When an unanswered ask closes. Default 24h. Set it only when the answer stops being useful sooner or later: a duration (`6h`) or an RFC 3339 time. It is the ask's life, not your wait. |
| `--timeout` | On `wait` and `ask --and-wait`: how long your command waits (default 30m). When it runs out the ask stays open and the command exits 4. |
| `--from` | Leave it out. The operator sees who is asking and sorts and filters the list by that name, and the CLI sets it for you from the label of your Web Console session, or of your run when someone named it (`research · steward-channel-ux-tooling`; a trailing run date such as `-20261114` is dropped so one role keeps one sender across days, and the generic `This machine` and the made-up `claude-code session <id>` labels are ignored), looked up once per command within about a second. When it cannot (a session nobody named, a service down), the Hub names the sender itself: your service's name, or `Agent run`, and the ask is sent all the same. Pass `--from '<short name>'` (1 to 60 characters, `Treasury steward`) only when the session label would mislead. The Hub does not verify the name and never grants access by it. The session label only names an ask when it is created: a revision never renames an ask, and only a `--from` you give changes the name of an open one. |
| `--urgency` | Default `normal`. `high` only when work is blocked right now. `critical` only for a real emergency; it can break quiet hours. |
| `--context-url` | Only a web page that opens on the operator's phone. Never a local file path or a `localhost` link; put those in `--technical-details`. |
| `--sensitivity-label` | Default `private`. `public`, `sensitive` or `critical` only when the text needs them. |
| `--idempotency-key` | Leave it out. The CLI makes one and prints it, so that a retry of the same send returns the same ask: an ask without a subject key gets a key from its terms and its `--expires` as you wrote it (`6h` stays `6h` on a resend), a keyed ask a fresh key each send, because the subject already makes a repeat of identical terms change nothing. The Hub counts a key as a retry only when it is the latest request applied to that ask, so returning to earlier wording is a revision, not a replay. |
| `--reassert-within` | With `--subject-key`, optional: the Hub closes the ask if you have not sent it again within this long (1m to 720h, such as `15m`). Use it only when you will keep re-sending while the decision is needed; each re-send with the same terms returns outcome `reasserted`. |
| `--default` | Only with `--reversible`, and never for spending, access, credentials or anything irreversible. Silence is not approval. |

Write all text in **plain language** (in the sense of the US Federal Plain
Language Guidelines): everyday words, short sentences, active voice, "you" for
the operator and "I" for yourself. One ask holds one decision. Send separate
asks for independent decisions so each one is a single tap.

## Decision packets

A decision packet is the ask plus what the operator needs to answer it cold:
attached files, an optional HTML explainer and an optional answer form. The cover
fields above stay the same and still carry the whole decision. Add a packet part
only when it saves the operator time. A packet part never blocks an ask. The
packet check on stderr reports each finding with one of two severities, so read
stderr after every `ask`: `dropped` means the Hub left that part out (a missing
or oversize file, too many attachments, a broken form question, a second
explainer listed as evidence); `advice` means the part is kept exactly as sent
and may not work as you meant (every cover-text rule and every explainer rule). The cover fields are different: their limits (big
picture 4,000 characters, technical details 12,000, option label 200, at most 8
options, `--default` only with `--reversible`) refuse the ask with an error and
nothing is sent. On a revision the packet check folds findings the open ask
already had into one line (`2 earlier warnings still apply`); `--json` lists
them all, marked `carried_over`.

| Part | Flag | Use it when | Notes |
|---|---|---|---|
| Attachments | `--attach PATH` (repeat) | The operator should see a file, folder, image, video, 3D model or draft instead of a path in text. | A folder is one attachment. The Hub keeps its own copy; it never serves your host path. The attachment's name is the file's base name, so name files for a stranger (`draft-b-second-opinion.md`, not `out2.md`), and sending a file with the same name on a revision **replaces** that one. `PATH=LABEL` adds only a display caption; it never changes the name. A folder's name is the folder's base name, cleaned the same way (letters, digits, `.`, `_`, `-`; anything else becomes `-`). Replacement compares names, not kinds, so a folder replaces a file only when its cleaned name equals the file's whole name, `notes.md` for instance; otherwise it is a new attachment. At most 24 attachments, the explainer included. |
| Explainer | `--explainer PAGE.html` | Layout carries the meaning: side by side options, before and after, a timeline, trade-offs, a media walkthrough. | It is its own attachment: do not also `--attach` it. Most decisions need none. Read `references/html-explainers.md` first (`path:scenarios/notification-hub/skills/notification-hub/references/html-explainers.md`); it has the rules, the starter kit and five worked examples. The hard rules in one line: a 390 px viewport meta tag, nothing loaded from the network (no `http:`, `https:` or `//` address in `src`, `poster`, `srcset`, `<link href>`, CSS `url()` or `@import`, or `fetch()`), every attachment it names is attached, no form controls, under 2 MiB, one `.html` file. These are advice, not refusals: the Hub still shows the page, and the packet check names what will probably break. |
| Answer form | `--form FORM.yaml` | The decision has parts that one option cannot answer: choose a layout, review each section of a plan, or answer a short question. | The main options stay the decision's answer. A question adds detail. Types: `single_choice`, `multi_choice`, `variant` (choices shown side by side with their attachment), `section_review` (approve, change or drop each section), `text`. |

A small form file:

```yaml
questions:
  - id: layout
    type: variant
    prompt: Which layout should the settings page use?
    required: true
    choices:
      - {key: cards, label: Card stack, attachment: cards.png}
      - {key: table, label: Table, attachment: table.png}
    recommended: cards
    recommendation_reason: Cards read best on a phone.
  - id: plan
    type: section_review
    prompt: Review each section of the plan.
    sections_from: plan.md
  - id: notes
    type: text
    prompt: Anything I should change?
```

A `single_choice` or `multi_choice` question lists `choices` the same way, without
attachments (`label` defaults to the `key`; `detail` adds one line under it):

```yaml
questions:
  - id: rollout
    type: single_choice            # multi_choice lets the operator tick several
    prompt: How should we roll it out?
    choices:
      - {key: staged, label: Staged, detail: One site a day}
      - {key: all-at-once, label: All at once}
    recommended: staged
  - id: plan
    type: section_review
    prompt: Review each section of the plan.
    sections_from: plan.md         # an attached Markdown file
    unanswered_verdict: approve    # approve (default), change or drop
```

What comes back in `wait --json`: a choice question gives `.responses[]` an entry
`{"question_id": "rollout", "selected": ["staged"]}`, one key for `single_choice`
and `variant`, several for `multi_choice` (a typed note, if the operator adds one,
is in `.text`). A review gives `{"question_id": "plan", "sections": [{"section_key": "storage", "verdict": "approve"}, ...]}`
with an entry for every section. A question the operator left alone has no entry.

How `section_review` looks to the operator: each `##` and `###` heading of the file
is one card with Approve, Change and Drop, a note box and a "Read this
section" button that opens the file at that heading. The verdict named by
`unanswered_verdict` is already selected on every card, and a line above the cards
says so ("Sections you skip count as “Approve”"). A section the operator does not touch is therefore sent with that
verdict, and the answer summary folds those into one line (`Other sections: approve`).
Choose `approve` when silence means you may go on, `change` or `drop` when it must
not. The Hub reads only the first 40 headings of the file; later ones get no card,
so split a longer plan into two files or two questions.

Form rules:

1. Put the decision in the main `--option` list. Do not hide it in a question.
2. Write each `prompt` as a plain question, under 500 characters. Use the same plain language as the cover.
3. Add `recommended` and a `recommendation_reason` whenever you have a view. "Accept recommendations" fills in all of them.
4. Use `sections_from` with an attached Markdown plan to review it by its `##` and `###` headings. The Hub reads the headings and **reads them again whenever a revision replaces that file**, so the review always matches the plan the operator opens, though the revision never mentions the form. If the new plan has no headings, or the file was removed, the review is left out and the packet check says so.
5. Keep to 12 questions and 12 choices per question, and 40 sections per review (`sections_from` keeps the first 40 headings). An unknown type shows as a text box, so the operator can still answer.
6. A form refers to files by their **attachment name**: the file's base name, or `name/path` for a file inside an attached folder. The file must be attached to the ask, either by `--attach` in this command or already on the open ask; a `variant` choice's `attachment` and a review's `sections_from` that name something not attached are reported (`question_attachment_unknown`) and the operator sees no image or sections. A local path that you also give to `--attach` is accepted as a spelling of its name, resolved against the directory you run the command from (the form file's own directory plays no part). A local path you did not attach is an error that says to add `--attach`.

What happens to a packet when a part it points at goes away. A revision is never refused for this; the packet check runs on the merged packet and names what now points at nothing:

| You remove or replace | What the Hub does |
|---|---|
| A file the explainer shows or opens | The revision applies; `explainer_unknown_attachment` names the file. Attach it again to fix the page. |
| A file a `variant` choice shows | The revision applies; the choice stays and `question_attachment_unknown` names the file. Attaching it again makes the image show again. |
| The Markdown file a review follows (`sections_from`) | Removed: the review is left out and the check says so. Replaced: the sections are read again from the new file, by heading; a renamed or new heading is a new section. |
| A file the operator pinned a note to | Notes are stored only when the operator submits their answer, and are checked against the files the ask holds then. A replaced file keeps its notes (the name is the same); a note on a file that was removed is refused with an error when they submit, and they take the note off. |

The operator's section verdicts are stored only when they submit the answer, not
while they work. If you revise the ask while they are answering, their answer is
refused (`ask_changed`), the page reloads the revised ask and keeps their draft, and
they check it and submit again.

The operator can always answer with "Other" and free text, so no packet part is
a gate. "Accept recommendations" copies your recommendations; make them ones you
would stand behind.

Notes the operator pins to a file arrive with their answer, in
`attachment_notes[]`, naming the attachment. The Hub checks them against the
files the ask holds when they answer, so a revision that replaced the file keeps
the notes pointing at it, and nothing is stored about a file before then.

### When the operator asks for more context

The operator can press "I need more context" on an ask that has a subject key,
because `ask` declares, whenever it has one, that you will answer. An ask sent
with `--no-context-requests`, or without `--subject-key`, shows no such button,
so the operator decides from what it says. The ask stays open and `wait` returns
early with the operator's question and exit status 3. Do this:

1. Read the question that `wait` printed (`context_request.question` in the JSON).
2. Answer it with `conversations ask --subject-key <key> --context-response '<your answer>'`. The operator reads your answer under their question. It is the only place the answer goes: not the big picture, not the technical details. Nothing else about the ask changes, so you resend nothing. If the answer also changes the ask, add that to the same command (a revised `--question` or `--big-picture`, `--attach`, `--remove-attachment`, `--form`, `--remove-question`); everything you leave out stays as it is.
3. Wait again: `conversations wait --subject-key <key>`, or send the answer and wait in one command with `ask --subject-key <key> --context-response '<your answer>' --and-wait`.

One rule: **any revision marks the operator's open question answered, but only
`--context-response` stores your answer for them to read.** A revision without
it leaves the question answered with no written answer, which tells the operator
nothing. The Hub sends no new push, shows the operator "Your question / Answer",
and the next wait does not return the old question. A later question and answer
are kept beside the first. If the operator has no open question, your response is
not stored and a warning (`context_response_unused`) says so. Add `--until-answer`
to `wait` only when you want it to ignore context requests.

Two runs waiting on the same subject key both get status 3 for one question. The
run that raised the ask answers it; the other keeps waiting with
`wait --subject-key <key> --until-answer`, which does not return for questions.
If both answer, the second answer finds no open question and is dropped with the
warning, so nothing is stored twice.

## Check before you send

Read your question and big picture as a stranger would. If any answer is
"no", rewrite before sending:

1. Does the question alone say what I am deciding?
2. Can I tell why it matters and why it is mine to decide, without opening anything?
3. Does every option say what will happen next?
4. Is there a recommendation with a reason, and do I know what happens if I do not answer?
5. Is every ID, path and abbreviation either gone or explained?
6. If I attached files, are they named for a stranger, and does the text say what each one is instead of showing its path?
7. If I added an explainer, does the decision still read well without it? Did I check it at 390 px wide in light and dark? Do all the files it names exist as attachments?
8. If I added a form, is the decision still in the main options, and does every question have a plain prompt?

## Example

Before — sent with no big picture, no recommendation reason:

```text
question: [S-14 · Vrooli release, web console first → Define the supported public release and its evidence] Approve a Stripe Managed Payments eligibility check and one staging test checkout, with Polar as fallback?
options:  yes=Approve the bounded eligibility and staging test only | no=Keep current Stripe Checkout and Polar as unapproved options
```

After:

```bash
notification-hub conversations ask \
  --question 'May I check whether Vrooli qualifies for Stripe Managed Payments and run one test purchase (no real money)?' \
  --big-picture 'To sell the first paid version of Vrooli we need a way to take payments. Stripe Managed Payments would act as the seller of record, which means Stripe also handles sales tax for us, but Stripe must first confirm we qualify. I want to submit that eligibility check and make one purchase in a test environment. No real money moves and nothing becomes public. If Stripe says no, I will prepare the same test with Polar, a similar service, and ask you again. If you do not answer, the release plan stays on hold at this step.' \
  --technical-details 'Steward objective S-14 (release, web console first), plan item F49-MOR-CHECKOUT. Packet and evidence: <path>. Current state: Stripe Checkout configured, Managed Payments not applied for.' \
  --option 'check=Yes — submit the eligibility check and run one test purchase' \
  --option 'hold=No — leave payments undecided; the release plan waits' \
  --recommend check \
  --recommend-reason 'It costs nothing, can be undone, and is the only way to learn whether sales tax is handled for us.' \
  --subject-key release-payments-eligibility \
  --expires 2026-11-20T17:00:00Z
```

The expiry is the only default overridden: this answer stops being useful on a
fixed date. Urgency, sensitivity and the idempotency key keep their defaults.

## Revise, withdraw and check

| Need | Command |
|---|---|
| Fix wording or add context | `ask --subject-key <key> --technical-details '...'` (or `--question`, `--big-picture`). The Hub updates the open ask in place and sends no new push. Anything you leave out stays: question, options, texts, attachments, explainer, form. The CLI prints what it kept on stderr. |
| Add or replace a file or question | Send it with the same name (`--attach` file base name, explainer, question `id`): it replaces that one and the rest stay. The match is by name only. |
| Replace a file under a new name | `--remove-attachment old.md --attach new.md`. A file with a different name is a **new** attachment: `--attach new.md` alone adds it and leaves `old.md` in place, so the operator sees both. |
| Remove one file | `--remove-attachment NAME` (repeat for more; the explainer by its name too). The other files stay. A name the ask has no file of is reported and changes nothing. |
| Remove one form question | `--remove-question ID` (repeat for more). The other questions stay. An id the ask has no question of is reported and changes nothing. |
| Answer the operator's "I need more context" question | `--context-response '<answer>'`, alone or with any change above. See "When the operator asks for more context". |
| Remove every file or every form question | `--clear-attachments` (includes the explainer) or `--clear-questions`, with whatever you send in their place. |
| Change the options, the recommendation or the default | `ask --subject-key <key> --option ... --option ...` with the new list, or a new `--recommend`, `--default`, `--reversible`, `--urgency` or `--sensitivity-label`. These make a new answer menu: the Hub replaces the open ask with a new one (`outcome=replaced`), carries its packet over, and pushes once. |
| Change only why you recommend it | `ask --subject-key <key> --recommend-reason '<new reason>'`. The recommended key is unchanged, so the ask updates in place with no push. Send `--recommend X` alone and the Hub treats it as a menu change when `X` differs from the open ask's key: new push. A revision that sends no `--option` keeps the open ask's options. |
| Preview any of these | Add `--dry-run`: it shows the notification of the revision merged over the open ask, and its packet check, and changes nothing. |
| Ask is obsolete | `notification-hub conversations withdraw --subject-key <key> --reason '<why>'`, or `withdraw <ask-id> --reason '<why>'`; `--reason` is required. A run waiting on it ends with status 5 and sees your reason. Withdrawing needs the same access as reading (see "Recover your asks after a restart"). |
| Ask sent without a subject key needs fixing | Withdraw it, then send a corrected ask with a `--subject-key`. |
| See what you sent or the answer | `notification-hub conversations get <ask-id>` or `get --subject-key <key>`; `conversations list --open` (`--sort attention` for what needs the operator first); `conversations wait <ask-id>`. |

What `ask` reports as its `outcome` when you send under a subject key (an ask
without one is always `created`):

| `outcome` | What happened |
|---|---|
| `created` | A new ask. The operator was pushed. |
| `updated` | The open ask changed in place, or your `--context-response` was stored with the operator's question. No push. |
| `replaced` | Your options changed: a new ask took over from the old one, which ended (a wait by id gets status 5, a wait by subject key follows). One push. |
| `reasserted` | Same terms as the open ask: nothing changed. This is also what a retry of the same send returns, and what `--reassert-within` re-sends should return. |
| `suppressed` | The operator dismissed these exact terms; no new ask was made. Change the substance, not the wording, before asking again. |
| `returned` | The ask was just answered, so you get that ask back instead of a new one. Read the answer with `get`. |

If the ask was answered or closed after your last look and you send only a
revision, nothing is applied and the error says what became of the ask (read it
with `get --subject-key <key>`). Send the whole ask (question and options) to
start a new one.

Before you send a new situation, run `notification-hub conversations list --open --query '<topic words>'`
and do not open a second ask about the same situation: revise the one that is open.

## Other routes to the operator

Three routes raise a decision. They end in the same Hub ask and the same
operator screen; they differ in who raises it and how the answer gets back.

| | `conversations ask` | `*.decision_requested.v1` event | `agent-manager effort decision raise` |
|---|---|---|---|
| Raised by | An agent or script, from a shell | A scenario's backend, with `operatordecision.Facts` (and a `Raiser` for retries) | An agent working for an enrolled effort |
| Needs | No identity or permission; requests are stored even when the source's device-push bucket is empty | A signed webhook from vrooli-events | The effort's goal home |
| Minimum input | `--question` and `--yes-no` or two `--option` | `Question` and two `Options` | `--id`, `--question`, two `--option` |
| Defaults | Expiry 24h, sensitivity private, urgency normal, derived idempotency key | The Hub's: deadline 24h, sensitivity private, urgency normal (set `Urgency: "high"` when blocked); `Recommended` optional | Urgency high (the CLI's `--urgency` default) |
| Files, explainer, form | Yes (`--attach`, `--explainer`, `--form`) | No: a signed webhook has no verified sender and carries no file bytes | No |
| Revise in place | Same `--subject-key` | Same `SubjectKey` (a `Raiser` re-send) | Raise the decision again |
| Answer reaches you | `wait` or `ask --and-wait`, by exit status; `get` | `notification_hub.ask.resolved.v1` to the scenario's webhook (`operatordecision.Webhook` and a `DomainEffect`) | Delivered into the raising run |
| Operator questions | Answer by revising | Only if the producer declares `AcceptsContextRequests` and implements `ContextResponder` | Not offered |
| Pick it when | You are an agent that needs a decision and will wait or look again | Scenario code decides, and the effect applies the answer without an agent | The decision belongs to an enrolled effort's goal home |

Other situations:

| Situation | Route | The rules above still apply to |
|---|---|---|
| A research supervisor needs the operator | A needs-you item; Research Desk forwards it | The item title becomes the question and its body becomes the big picture. Write both for a stranger; keep IDs and paths in a final "Details" line. |
| Information only, no answer needed | `notification-hub notifications send --title ... --body ...` | Title says what happened; first body line says which one; say "No action needed". See `docs/concepts/NOTIFICATIONS.md` § Writing the title and body. |
| Scenario code asking for a decision | Publish a `*.decision_requested.v1` event | See `docs/concepts/INTEGRATIONS.md` § Decision requests. |
| Scenario code sending a notice or ask directly, as itself | Send through the Hub API/CLI without obtaining permission or provisioning a sender principal. The Hub applies the configured device-push bucket to the source; an exhausted bucket suppresses device pushes while the notification or ask remains stored. | Use a stable idempotency key where retries are possible. A send being accepted does not approve the action described by an ask; the operator's answer remains the decision. |

### A host program that is not an agent run

A timer or service-manager process can send notifications and decision asks
without agent identity, an authorization token, or an operator permission step:

```bash
notification-hub notifications send --title 'Backup finished' --body 'No action needed.'
notification-hub conversations ask --yes-no --subject-key backup-retry --question 'Retry the backup now?'
```

The Hub stores each accepted request even when its source's device-push bucket
is empty. Bucket capacity and drain rate are configurable; when the bucket has
capacity again, later requests can push to devices. Keep routine outcomes in the
Hub and use an ask only when an operator decision is required. Do not request
permission to send through the Hub. A decision request still asks the operator
to decide the described action; it does not grant permission to send messages.

## Read receipt

After a new ask (not after a revision), the CLI asks Prompt Manager whether your
agent run has read this skill. When it cannot find that read, it prints a
reminder with the read command and the revise command. The reminder never blocks
the ask. To be counted, read the skill through Prompt Manager:
`prompt-manager skill read notification-hub`. Reads through a native skill
directory are not visible to this check; if you already applied this skill,
ignore the reminder.

## Troubleshooting & Edge Cases

| Symptom | Cause | Fix |
|---|---|---|
| A price shows as `00` instead of `$100` | The shell expanded `$1` inside double quotes. | Quote text with single quotes. For an apostrophe inside, use `'\''` or `$'...'`. |
| `answer key "other" is reserved` | The Hub adds its own free-text "Other" answer. | Remove your `other` option. |
| `big_picture must be ... at most 4000` or `at most 8 answers are allowed` | The cover fields have hard limits that refuse the ask (unlike the packet check): big picture 4,000 characters, technical details 12,000, option label 200, 2 to 8 options. | Shorten it, move evidence to technical details or the context page, and send again. |
| `default answer is allowed only for a reversible decision` | `--default` needs `--reversible`. | Drop the default unless the choice can truly be undone. |
| Ask is stored but no push arrived | Each source has a push budget (about six, then one per minute). The ask is still stored and listed. | Bundle related asks; do not resend. |
| Packet check says `explainer_may_help` | The text has the outline of a comparison: 3 or more options with long labels, a background over 1,000 characters, or several images with nothing to compare them. | If the choice rests on a comparison, copy a starter page from `examples/` and revise with `--explainer` (see "Choose the shape before you write"). Ignore it when the options are not a comparison. |
| Packet check says `internal_id_in_question`, `path_in_question`, `long_question` or `missing_big_picture` | The cover text breaks the field rules above. | Rewrite the field and run `ask --subject-key <key>` again with just that field. |
| Packet check says `unattached_path` or `local_context_url` | The text names a local file that is not attached, or the context link opens only on this computer. | Attach the file and describe it in words. Move local paths to `--technical-details`. |
| `Left out ...` lines on stderr after `ask`, or `Packet check: N left out` | A file, folder, explainer or form was unreadable, too large (file 64 MiB, total 256 MiB, folder 500 files) or invalid. The ask was still sent without it. | Fix the part the line names. Run `ask --subject-key <key>` again with only the fixed part; the rest of the ask stays. |
| Packet check says `attachment_blob_missing`, `attachment_limit` or `question_invalid` | The Hub left that part out of the packet. | Fix the file, count or question that the message names, and resend. The rest of the packet was kept. |
| Packet check says `explainer_network`, `explainer_no_viewport`, `explainer_form`, `explainer_unknown_attachment`, `explainer_too_large` or `explainer_not_html` | The page breaks an explainer rule. | See "Hard rules" in `references/html-explainers.md`. Fix the page and resend. |
| `conversations wait` exits with status 3 | The operator asked for more context. The ask is still open. | Follow "When the operator asks for more context". |
| `conversations wait` exits with status 4 or 5 | 4: your wait timed out (`--timeout`) and the ask is still open. 5: the ask closed without an answer. | 4: wait again. 5: read the reason line; ask again only if the need remains. |
| `already exists with different terms` | You gave an `--idempotency-key` that an earlier ask already used for other text. | Drop `--idempotency-key` (the CLI derives one). To change an open ask, add `--subject-key` and run it again. |
| Identical resend reports outcome `reasserted`, `suppressed` or `returned` | Same subject key and same terms: `reasserted` means the ask is already open and nothing changed; `suppressed` means the operator dismissed these exact terms; `returned` means it was just answered. | `reasserted`: nothing to do. `suppressed`: change the substance, not the wording, before asking again. `returned`: read the answer with `get`. |
| `your revision was not applied: the ask with the subject key ... was answered` | The operator answered (or the ask closed) after your last look, so no open ask was left to revise. | Read it with `get --subject-key <key>`. To ask again, send the whole ask. |
