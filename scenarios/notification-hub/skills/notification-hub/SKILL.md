---
name: "notification-hub"
description: "Ask the operator for a decision through Notification Hub as a page they can answer cold from a phone: lead with the choice, show the situation, say what each answer does."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  revision: 14
  modes: ["tools"]
  tags: ["notification-hub", "decisions", "asks", "operator", "notifications", "writing", "operator-legibility", "html-explainer"]
  requires:
    scenarios: ["notification-hub", "program-runtime"]
    commands: ["notification-hub conversations ask", "program-runtime library run"]
  origin:
    kind: "authored"
---

## Make a decision the operator can answer cold

Sending an ask needs no identity, grant or permission request. The Hub stores every
ask; a per-source bucket only delays device pushes, so never resend because a push
was held back.

The operator reads your decision on a phone, between other things, with none of
your context: not your chat, files, IDs or plan. A hard-to-read decision waits, and
so does every agent behind it.

**A decision is a page plus a packet.** The page shows what is going on, what you
recommend and what each answer does. The packet is the brief a worker needs to carry
out the answer, `work/<ID>/BRIEF.md` (`docs/agent-system/LEVELS.md`), sent with
`--packet work/<ID>`. The Hub attaches it, the page gains "What happens when you
approve", and `wait` prints the message that starts the executor. An ask sent
without a page is shown with the Hub's plain page, which only repeats your question
and options; the Hub advises `explainer_missing` and `packet_missing`.

**Never ship your judgment as the sender's.** When you re-present another agent's
decision, keep its recommendation. Your concern goes in one line labelled
"Second opinion".

## Make the page

One self-contained HTML file, sent with `--explainer page.html`. The Hub shows it
sealed above the answer buttons, with a starter kit of colours and blocks. The
operator wants it to look like a finished product, not a form. **The page must add
what the cover cannot hold.** A page that repeats the question and options is
refused (`explainer_restates`). Build it like this:

1. **Lead with the decision**: what is being chosen, what you recommend, why, in
   one sentence each.
2. **Show the situation**: the one picture, table or chart that makes the choice
   clear, with real captures and real numbers. Many items become a few groups and a
   bar of how the whole splits (`.hub-proportion`, `.hub-groups`), never a long card
   list. A UI proposal is shown in the actual app at production finish, not a wireframe.
   Show each picture once: a choice between screens gets one picture per option,
   cropped to the part that changes, not a gallery and then the same screens again.
   A table (`.hub-matrix`) holds a word, a number or a mark per cell; reasons go in
   one line under it.
3. **A command the operator must run to carry out an answer goes on the first
   screen** in `.hub-command`: where to run it, the exact line with Copy, what it
   does step by step, what it shows before it changes anything, and the undo. Never
   only in `--technical-details`: the phone does not show it. Write the real line,
   never a placeholder such as `<id>`. A command that is only possible later (how to
   undo, how to turn something off one day) is one sentence in the card it belongs
   to, not a `.hub-command` block.
4. **Name the cause.** Say why this decision exists. When the cause can be fixed,
   one option fixes it (a standing rule, a policy, a repair), not only today's
   symptom. If you have asked this before, show it with `.hub-recurrence` and offer
   the option that stops the repeats.
5. **One card per answer**, each with what happens next, its cost or risk, and a
   "Choose this" button calling `hub.prefill("answer", "<key>")`.
6. **What happens when you approve**, from the packet, and **what happens with no
   answer**, stated as a real choice with its consequence.
7. **IDs, paths and jargon** go in `--technical-details` or a collapsed
   `<details>`. A command the operator runs is not jargon (step 3).
8. **Keep it short**: about four phone screens. Say each fact once; if a section
   repeats the cover or another section, cut it.

**Colours: light and dark.** The operator reads in both. Colour with the kit's
tokens (`var(--hub-surface)`, `var(--hub-fg)`, `var(--hub-muted)`,
`var(--hub-primary)`). Every rule that sets a background also sets `color` in the
same rule. Never a literal background (`white`, `#fff`) on its own. A page with
text the operator cannot read in either theme is refused (`page_unreadable`).

Nothing loads from the network: attach pictures and name them in `src`. No form
controls; under 2 MiB. Kit blocks and contract: `references/html-explainers.md`.
Start from the starter that fits and replace every word and number:

| Starter (`examples/`) | Use for |
|---|---|
| `approve-action.html` | One action: effect, risk, undo |
| `choose-between-drafts.html` | Two or three finished options side by side |
| `plan-tradeoffs.html` | Several options against a few criteria |
| `compare-before-after.html` | A change as before and after, with a slider |
| `rollout-timeline.html` | A plan over time, with a stop rule |
| `media-walkthrough.html` | Video, photos or a 3D model, with steps |
| `batch-by-group.html` | A list or batch: groups the operator switches on (with a `--form` multi_choice), a bar of the split, the cause |
| `run-a-command.html` | The operator runs a command: where, the line, what it shows first, undo, why only they can |
| UX redesigns | `~/.vrooli/research/topics/experience-engineering/skills/present-ux-decisions/starter.html` |

Six recurring pages have templates instead: send the template name and one JSON
object (`--template NAME --data-file data.json`), never HTML. The page draws its
charts from the data, works on a phone in light and dark, records the operator's
choices for the answer bar, and marks any section whose data is missing as
unavailable. Fill every field from real sources and leave out what you do not
know; the schema documents each field.

| Template (`kit/templates/`) | Use for | Schema, sample data |
|---|---|---|
| `usage-mode.html` | Decision: a usage window will run out; burn chart, each option's outcome, how often this came back | `kit/schemas/usage-mode.schema.json`, `kit/samples/usage-mode.json` |
| `daily-briefing.html` | Status update (`--kind status`): target, hours by lane, friction, pace tiers that switch the slot schedule | `kit/schemas/daily-briefing.schema.json`, `kit/samples/daily-briefing.json` |
| `opportunity-roadmap.html` | Status update: every line on one timeline, steps, first-income ranges, slip, weights | `kit/schemas/opportunity-roadmap.schema.json`, `kit/samples/opportunity-roadmap.json` |
| `structural-change.html` | Decision: loosen or keep checks that keep stopping work, on a map of hours lost against harm | `kit/schemas/structural-change.schema.json`, `kit/samples/structural-change.json` |
| `evidence-decision.html` | Decision: review one reusable evidence comparison (image, audio, video, text, fields, UI or measurement) | `kit/schemas/evidence-decision.schema.json`, `kit/samples/evidence-decision.json` |
| `research-report.html` | Status update when a research commission ends: workstream states, measured results, scorecard progress with judged levels, feedback read-back, next lines; per-workstream weights | `kit/schemas/research-report.schema.json`, `kit/samples/research-report.json` |

Preview with `node kit/tools/render.mjs <template> data.json --out page.html`, then
`notification-hub pages check page.html` sends a non-storing packet check. Review
the result before `pages shoot`: shooting writes local screenshots under separate
filesystem authorization. See `kit/README.md` for answer keys and render contract.

## Read one conversation

Use `program-runtime library run notification-hub.conversation-read`
with `--input ask_id=<id>` or `--input subject_key=<key>`. Owner access remains
enforced; keep returned private content out of Memory, logs and evidence.

A topic that presents one kind of decision often keeps its own presentation skill in
its `skills/` folder and points back here. Scenario code sets
`operatordecision.Facts.Explainer` (built with `operatordecision.NewPage`) and
`Facts.Packet`; the Hub attaches the page as `decision.html`.

## Write the cover

The push and the decision list never show the page, so the cover carries the
decision in text too:

| Flag | Write |
|---|---|
| `--question` | One sentence answerable from the push alone, under 200 characters, with no IDs, run names, paths or tags. |
| `--big-picture` | 2–6 short sentences: what this is, why now, why only they can decide, what each answer causes, what happens with no answer. |
| `--option key=label` | Each label says what happens if chosen (`check=Yes — submit the check and run one test purchase`). |
| `--recommend`, `--recommend-reason` | Recommend whenever you have a view; one plain sentence on benefit or risk. |
| `--technical-details` | IDs, paths and evidence for the agent that acts: the only place internal names belong. A command the operator runs also goes on the page. |

Write "you" for the operator and "I" for yourself, in everyday words. One ask holds
one decision.

## Group checks with one cause

When several decisions exist because of the same underlying cause, give each the
same stable `--decision-family` key and the same plain-language `--decision-cause`.
The inbox groups those asks under the cause while keeping every check separately
answerable. Use `--decision-kind` to say what work the operator is doing: `judge`,
`approve`, `review`, or `do`. Do not group unrelated choices just because they
come from the same sender or project.

```bash
notification-hub conversations ask \
  --subject-key release-guards:network \
  --question 'May the network guard pass for this release?' --yes-no \
  --decision-family release-guards \
  --decision-cause 'The release is waiting on three independent safety checks.' \
  --decision-kind review
```

Use the same family and cause with a different `--subject-key` and question for
each remaining check.

## Send, revise, wait

```bash
notification-hub conversations ask \
  --subject-key release-payments-eligibility \
  --question 'May I check whether Vrooli qualifies for Stripe Managed Payments and run one test purchase (no real money)?' \
  --big-picture 'To sell the first paid version we need a way to take payments. ... If you do not answer, the release plan waits here.' \
  --option 'check=Yes — submit the eligibility check and run one test purchase' \
  --option 'hold=No — leave payments undecided; the release plan waits' \
  --recommend check --recommend-reason 'It costs nothing, can be undone, and shows whether sales tax is handled for us.' \
  --technical-details 'Plan item F49-MOR-CHECKOUT.' \
  --explainer page.html --packet work/F49-MOR-CHECKOUT --dry-run   # drop --dry-run to send
```

- **Always set `--subject-key`**, a stable name for the situation, so the ask is
  one living thing you can find after a restart.
- **Revise; never ask again.** `ask --subject-key K` with only what changes updates
  the open ask with no new push. Changing option keys or the recommended key
  replaces it and pushes once.
- **Wait by exit status.** `conversations wait --subject-key K` (or `ask --and-wait`):
  0 answered, 3 more context wanted (reply with `ask --subject-key K
  --context-response '...'`), 4 your wait timed out, 5 closed unanswered, 1 failed.
- **Read stderr**: the packet check names what the Hub left out or what reads poorly.
  Findings marked `blocking` refuse the send (a recommendation with no reason, a page
  that restates the cover, a page that cannot be read); nothing is stored. Fix each
  one, `--dry-run` again, then send.

Before a new situation, `conversations list --open --query '<words>'`. Loops, JSON,
forms, attachments, withdrawing, routes: `references/cli.md`; outputs:
`references/cli-results.md`.

## Check before you send

Run the ask with `--dry-run` until nothing is `blocking` (`notification-hub pages
check page.html` checks a page on its own). Then photograph it with `notification-hub
pages shoot page.html`, open every screenshot, and read the page and the cover as a
stranger, at 390 px, in light **and** dark:

1. Does the first screen say what I am deciding, what you recommend, and how to
   answer?
2. Can I see each option (including "keep as is") and what it affects, in real
   pictures or numbers?
3. Does every option say what happens next, approval what gets carried out, and
   no answer what that choice means?
4. Is every ID, path and term gone, explained, or folded into "Where these pictures
   come from"?
5. Is every word readable in both themes, and is the command (if any) on the
   first screen with where, what and undo?
6. Does it say why this is being asked, and does one option fix the cause?
7. Does the cover still make sense without the page, and does the packet let a
   worker start without asking you anything?
