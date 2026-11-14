---
name: "notification-hub"
description: "Ask the operator for a decision, or tell them something, through Notification Hub so that a reader with none of your context can understand it and answer from a phone in under a minute."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  revision: 1
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
| Background | `--big-picture` | 2–6 short sentences: what this is about and why it matters now; why only the operator can decide; what happens after each answer; what happens if nobody answers. Define any term you must use in the same sentence. | A copy of the option list, paths, IDs, raw logs. |
| Assessment | `--technical-details` | Evidence, IDs, commands, file paths, packet locations, links to the run — everything the next agent needs to act on the answer. This is the only field where internal names belong. | Anything the operator needs in order to choose. |
| Choices | `--option key=label` (2–5) | Each label states what happens if chosen, as the owner's next action: `check=Yes — submit the eligibility check and run one test purchase`. Keys are short words (`check`, `skip`, `later`). Add a defer option only when deferring is a real answer. | `yes`/`no` keys with vague labels; a key named `other` (the Hub adds its own free-text "Other"). |
| Recommendation | `--recommend <key>` and `--recommend-reason` | Recommend whenever you have a view. The reason is one plain sentence about benefit or risk. | "Recommended." with no reason. |

Set the remaining flags this way:

| Flag | Rule |
|---|---|
| `--subject-key` | Always set it to a stable name for the situation (`release-payments-check`). It lets you fix the text later without a second push. |
| `--deadline` | The real time after which the answer stops being useful. Use an RFC 3339 time for a fixed date. |
| `--urgency` | `normal` for agent asks. `high` only when work is blocked right now. `critical` only for a real emergency; it can break quiet hours. |
| `--context-url` | Only a web page that opens on the operator's phone. Never a local file path or a `localhost` link; put those in `--technical-details`. |
| `--sensitivity-label` | `public`, `private`, `sensitive` or `critical`. Use `private` unless the text is safe on a lock screen. |
| `--idempotency-key` | Stable for these exact terms, so a retry does not create a duplicate. |
| `--default` | Only with `--reversible`, and never for spending, access, credentials or anything irreversible. Silence is not approval. |

Write all text in **plain language** (in the sense of the US Federal Plain
Language Guidelines): everyday words, short sentences, active voice, "you" for
the operator and "I" for yourself. One ask holds one decision. Send separate
asks for independent decisions so each one is a single tap.

## Check before you send

Read your question and big picture as a stranger would. If any answer is
"no", rewrite before sending:

1. Does the question alone say what I am deciding?
2. Can I tell why it matters and why it is mine to decide, without opening anything?
3. Does every option say what will happen next?
4. Is there a recommendation with a reason, and do I know what happens if I do not answer?
5. Is every ID, path and abbreviation either gone or explained?

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
  --deadline 2026-11-20T17:00:00Z --urgency normal \
  --sensitivity-label private --idempotency-key release-payments-eligibility-v1
```

## Revise, withdraw and check

| Need | Command |
|---|---|
| Fix wording or add context | Send the same `ask` again with the same `--subject-key` and the same options. The Hub updates the open ask in place and sends no new push. |
| Change the options | Send the `ask` again with the same `--subject-key`. The Hub replaces the open ask and pushes once. |
| Ask is obsolete | `notification-hub conversations withdraw --subject-key <key> --reason '<why>'` (or `--ask-id <id>`). |
| Ask sent without a subject key needs fixing | Withdraw it, then send a corrected ask with a `--subject-key`. |
| See what you sent or the answer | `notification-hub conversations get <ask-id>`; `conversations list --open`; `conversations wait --ask-id <id> --deadline 1h`. |

Before you send, run `notification-hub conversations list --open --query '<topic words>'`
and do not open a second ask about the same situation.

## Other routes to the operator

| Situation | Route | The rules above still apply to |
|---|---|---|
| A decision for an enrolled effort's goal home | `agent-manager effort decision raise` | `--question`, `--big-picture`, `--technical-details`, option labels |
| A research supervisor needs the operator | A needs-you item; Research Desk forwards it | The item title becomes the question and its body becomes the big picture. Write both for a stranger; keep IDs and paths in a final "Details" line. |
| Information only, no answer needed | `notification-hub notifications send --title ... --body ...` | Title says what happened; first body line says which one; say "No action needed". See `docs/concepts/NOTIFICATIONS.md` § Writing the title and body. |
| Scenario code asking for a decision | Publish a `*.decision_requested.v1` event | See `docs/concepts/INTEGRATIONS.md` § Decision requests. |

## Read receipt

After `conversations ask`, the CLI asks Prompt Manager whether your agent run
has read this skill. When it cannot find that read, it prints a reminder with
the read command and the revise command. The reminder never blocks the ask. To
be counted, read the skill through Prompt Manager:
`prompt-manager skill read notification-hub`. Reads through a native skill
directory are not visible to this check; if you already applied this skill,
ignore the reminder.

## Troubleshooting & Edge Cases

| Symptom | Cause | Fix |
|---|---|---|
| A price shows as `00` instead of `$100` | The shell expanded `$1` inside double quotes. | Quote text with single quotes. For an apostrophe inside, use `'\''` or `$'...'`. |
| `answer key "other" is reserved` | The Hub adds its own free-text "Other" answer. | Remove your `other` option. |
| `big_picture must be ... at most 4000` | Limits: big picture 4,000 characters, technical details 12,000, option label 200, at most 8 options. | Move evidence to technical details or the context page. |
| `default answer is allowed only for a reversible decision` | `--default` needs `--reversible`. | Drop the default unless the choice can truly be undone. |
| Ask is stored but no push arrived | Each source has a push budget (about six, then one per minute). The ask is still stored and listed. | Bundle related asks; do not resend. |
| Identical resend reports "already open" or "suppressed" | Same subject key and same terms: nothing new to send, or the operator dismissed these exact terms. | Change the substance, not the wording, before asking again. |
