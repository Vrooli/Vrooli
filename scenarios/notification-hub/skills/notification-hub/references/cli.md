# Notification Hub CLI: the mechanics

Read this when `SKILL.md` is not enough: waiting in a loop, reading JSON, revising,
forms, other routes, troubleshooting. Every command's `--help` is authoritative. The
exact output of each command, one real example per exit status, is generated in
`cli-results.md`.

## Words

**Options** are what the operator picks (`--option key=label`, 2 to 8, or `--yes-no`).
The **answer** is what they reply. **Form questions** are typed extras under the
options (`--form`); their **choices** live in the form file. **Expires** (`--expires`,
default 24h) is when an unanswered ask closes. **Timeout** (`--timeout`, default 30m)
is how long your command waits.

## Defaults and fields

Everything but the question and options has a default: expires 24h, sensitivity
`private`, urgency `normal`, a printed idempotency key. `--body-file ask.md` (or `-`)
holds the cover in one file: first paragraph = question, `## Background` = big
picture, `## Details` = technical details.

| Flag | Rule |
|---|---|
| `--question` | One sentence answerable from the push alone, under 200 characters. No IDs, codes, run names, paths, bracketed tags. |
| `--big-picture` | 2–6 short sentences: what this is, why now, why it is the operator's, what each answer causes, what happens with no answer. Max 4,000 characters. |
| `--technical-details` | IDs, commands, paths, evidence for the agent that acts on the answer. The only place internal names belong. Max 12,000. |
| `--option key=label` | The label says what happens if chosen. Keys are short words. Never `other` (the Hub adds a free-text Other). |
| `--recommend`, `--recommend-reason` | Recommend whenever you have a view; the reason is one plain sentence (max 300). |
| `--subject-key` | Always. A stable name for the situation. |
| `--urgency` | `high` only when work is blocked now; `critical` only for an emergency (it may pass quiet hours). |
| `--default` | Only with `--reversible`; never for spending, access, credentials or anything irreversible. |
| `--context-url` | Only a page that opens on a phone. Never a local path or `localhost`. |
| `--from` | Leave it out; the CLI names you from your session or run label. Pass a short name only if that label misleads. |
| `--reassert-within` | With a subject key: the Hub closes the ask if you have not re-sent it within this long. |

## Revise, never ask again

Send `ask --subject-key <key>` with only what changes. Everything you leave out stays
(question, options, recommendation, texts, attachments, explainer, form), and the CLI
prints what it kept. A text flag replaces that text whole (`''` clears it). Text,
labels, the reason, files and the page update in place with no new push. The
**answer menu** (option keys, `--recommend` key, `--default`, `--reversible`,
urgency, sensitivity) is different: changing it replaces the ask and pushes once.

| Need | Command |
|---|---|
| Fix wording, add evidence | `ask --subject-key K --technical-details '...'` |
| Add or replace the page | `ask --subject-key K --explainer page.html` |
| Add or replace the packet | `ask --subject-key K --packet work/<ID>` (attaches `BRIEF.md`; `wait` prints the executor's start message) |
| Replace a file under a new name | `--remove-attachment old.md --attach new.md` (a new name alone adds a second file) |
| Drop one file / question | `--remove-attachment NAME` / `--remove-question ID` |
| Drop all files / questions | `--clear-attachments` (includes the page) / `--clear-questions` |
| Answer "I need more context" | `--context-response '...'` (see below) |
| Preview | add `--dry-run`: packet check and push text, nothing stored |
| Obsolete | `conversations withdraw --subject-key K --reason '...'` |
| Read it | `program-runtime library run notification-hub.conversation-read --input subject_key=K`; `conversations list --open` |

Before opening a new situation, `conversations list --open --query '<words>'`. Same
decision with new facts: revise. Answering one would not answer the other: new key.

`ask` reports an `outcome`: `created` (pushed), `updated` (in place, no push),
`replaced` (new menu, one push), `reasserted` (same terms, nothing changed),
`suppressed` (the operator dismissed these exact terms; change the substance),
`returned` (just answered; read it with `get`). A partial revision of an ask that was
answered or closed meanwhile is refused; send the whole ask to start a new one.

## Wait by exit status

`conversations wait <ask-id>`, `wait --subject-key K`, or `ask --and-wait`.

| Status | Meaning | You do |
|---|---|---|
| 0 | Answered, or the default applied | Act on the answer |
| 3 | The operator asked for more context; still open | `ask --subject-key K --context-response '...'`, then wait again |
| 4 | Your `--timeout` ran out; still open | Wait again or carry on |
| 5 | Closed without an answer (withdrawn, dismissed, expired; by id also `replaced`) | Read the reason; ask again only if the need remains |
| 1 | The command failed | Read stderr |

A wait survives dropped connections and the Hub's 10-minute limit. By subject key
it follows a replacement ask. Loop on 4:

```bash
while true; do
  notification-hub conversations wait --subject-key K --timeout 1h --json > answer.json
  case $? in 0|3|5|1) break ;; 4) continue ;; esac
done
```

Write JSON to a file (or `set -o pipefail`) before `jq`, or the exit status is lost.

| You want (`wait --json`) | Path |
|---|---|
| The choice | `.answer` (option key); `.note` free text; `.state` `answered`/`defaulted` |
| A form answer | `.responses[]?` with that `question_id`: `.selected[]`, `.text`, or `.sections[]` of `{section_key, verdict, note}` |
| Everything as text | `.answer_summary` |
| Their question (exit 3) | `.context_request.question` |
| Why it closed | `.reason` |
| Notes on files | `.attachment_notes[]` |

`get --json` wraps the ask as `.ask` (so `.ask.answer`). `ask --and-wait --json`
prints two documents; read the second with `jq -s '.[1]'`. Use `.responses[]?`:
the field is omitted when empty.

## More context

A keyed ask offers the operator "I need more context" (`--no-context-requests`
turns it off). `wait` then exits 3. Answer with
`ask --subject-key <key> --context-response '...'` only: any
revision marks the question answered, but only `--context-response` stores your
answer for them to read. Add other changes in the same command if the answer changes
the ask. Two runs waiting on one key both see status 3; the run that raised the ask
answers, the other uses `wait --until-answer`.

## Recover after a restart

Rebuild the subject key and run `program-runtime library run notification-hub.conversation-read --input subject_key=K` / `wait --subject-key K`.
Otherwise `conversations list --open --query '<words>'`. Access: by id, the runs of
the conversation that sent it; by exact subject key, any run of the same owner
account; `list` shows only your conversation's asks.

`list` sorts with `--sort newest|oldest|attention|due` and filters with `--state`,
`--urgency`, `--sender` (exact, repeatable) and `--query`. `--limit` (max 200) pages
with `--cursor`.

## Files and forms

`--attach PATH[=caption]` (max 24 incl. the page; 64 MiB a file, 256 MiB total,
500 files a folder). The name is the base name: name files for a stranger. The Hub
keeps its own copy and never serves your path.

`--form form.yaml` adds typed questions; the decision itself stays in the options.

```yaml
questions:
  - {id: layout, type: variant, prompt: Which layout?, recommended: cards,
     choices: [{key: cards, label: Cards, attachment: cards.png}, {key: table, label: Table, attachment: table.png}]}
  - {id: plan, type: section_review, prompt: Review each section., sections_from: plan.md, unanswered_verdict: approve}
  - {id: notes, type: text, prompt: Anything to change?}
```

Types: `single_choice`, `multi_choice`, `variant`, `section_review`, `text`. At most
12 questions, 12 choices, 40 sections (`##`/`###` headings, re-read when the file is
replaced). A form names files by attachment name; they must be attached. A broken
question is dropped with a warning; the rest stays.

## Packet check

Read stderr after every `ask`. `blocking`: a real send is refused, nothing is
stored; fix it and `--dry-run` again. `dropped`: the Hub left a part out.
`advice`: kept as sent, may not work as meant. Cover limits refuse the ask outright.

| Code | Fix |
|---|---|
| `explainer_missing` | Make the page (`SKILL.md`); until then the operator sees the Hub's plain page. |
| `packet_missing` | Write `work/<ID>/BRIEF.md` and send `--packet work/<ID>`. |
| `internal_id_in_question`, `path_in_question`, `long_question`, `missing_big_picture`, `missing_recommendation_reason` (blocking) | Rewrite that field; revise with just it. |
| `unattached_path`, `local_context_url` | Attach the file and name it in words; local paths go in technical details. |
| `explainer_restates` (blocking) | The page must add what the cover cannot hold: start from a starter in `examples/`, show the list or numbers with kit blocks, screenshots, the command, the cause. |
| `page_unreadable` (blocking) | Colour with `--hub-*` tokens; every rule that sets a background sets `color` too. Check light and dark. |
| `command_hidden` | Put the command on the page in `.hub-command`, not inside a sentence: where, the line, what it shows first, undo. |
| `page_no_picture` | Show the situation: a kit decision block, a `role="img"` chart, a table or a cropped screenshot. |
| `page_picture_uncropped` | Crop each screenshot to the part that changes: `height: 220px; object-fit: cover`. |
| `recurring_subject` | Name the cause; add one option that stops the repeats (a standing rule or a fix); show `.hub-recurrence`. |
| `explainer_network`, `explainer_no_viewport`, `explainer_form`, `explainer_unknown_attachment`, `explainer_too_large`, `explainer_not_html` | See the hard rules in `html-explainers.md`. |
| `attachment_blob_missing`, `attachment_limit`, `question_invalid` | Fix that file or question and resend it alone. |

## Page tools

`notification-hub pages check page.html [--cover-file cover.json] [--attach shot.png] [--form form.yaml]`
runs the same check as `--dry-run` on one page (the cover file is JSON: question,
options [{key, label}], recommended, recommendation_reason, big_picture); exit 1 when
the Hub would refuse it. `notification-hub pages shoot page.html [--out DIR] [--narrow]`
photographs it with the Hub's kit at 390 px in light and dark.

## Other routes

| Route | Who | Page | Answer comes back |
|---|---|---|---|
| `conversations ask` | An agent or host script; no identity or permission needed | `--explainer` | `wait`, `get` |
| `*.decision_requested.v1` event | Scenario code, `operatordecision.Facts` + `Raiser` | `Facts.Explainer` (inline, built with `operatordecision.NewPage`) | `notification_hub.ask.resolved.v1` to the scenario's webhook |
| `agent-manager effort decision raise` | An agent working for an enrolled effort | see its `--help` | Delivered into the raising run |
| Research needs-you item | A research supervisor (`### ID — question` in `STATE.md`) | `Explainer:` line naming an `.html` page | Written to the commission's `OPERATOR.md` |

Information only: `notification-hub notifications send --title ... --body ...` (title
says what happened; say "No action needed"). Each source has a push budget (about
six, then one per minute); asks are stored even when pushes are suppressed, so do
not resend.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `$100` shows as `00` | Single-quote text; the shell expanded `$1`. |
| `answer key "other" is reserved` | Remove your `other` option. |
| `default answer is allowed only for a reversible decision` | Drop `--default`, or add `--reversible` if it truly can be undone. |
| `already exists with different terms` | Drop `--idempotency-key`; use `--subject-key` to revise. |
| `your revision was not applied: ... was answered` | Read it with `notification-hub.conversation-read` and `subject_key=K`; send the whole ask to ask again. |
| Read-receipt reminder after a new ask | Read the skill through `prompt-manager skill read notification-hub`; it never blocks. |
