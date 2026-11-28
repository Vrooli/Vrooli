# Operator attention

The operator's only job in research is answering decisions, on a phone, with no context (R9). Ask rarely, ask well, never ask what agents can resolve.

## Ask or not

- Agents can resolve it or restore a permitted path: assign the work; do not ask.
- Only the operator holds it: prepare one concrete request and keep every other workstream moving. Before a credential ask, read `vrooli credentials list` (names and state only) and ask only for what is missing.
- Not reviewable yet: finish the preparation. Later evidence removed the need: mark it superseded; do not send it.
- A question inside a topic's scope belongs to that topic's manager. The research-review skill decides manager-level questions itself or routes them to the topic; the operator gets only a bright line (spend, outside contact, secrets, unrecoverable deletion) or a direction (what the end state is).

## Needs-you items

A topic keeps its decisions in `DECISIONS.md` (`## Needs you`); the research home in `ATTENTION.md` (`## Actionable now`, with `## Proposals awaiting approval` above it). `ATTENTION.md` lists only items that ask for action; status belongs in `STATE.md`. Research Desk reads them as a file contract (`scenarios/research-desk/docs/reference/FILE-CONTRACT.md` §5):

```markdown
### YTDLP-TARGET — Keep the video download tool uninstalled?
Two to six plain sentences for a stranger: what this is about, why only the operator
decides, what each answer leads to. No IDs, run names or paths.
Recommended: keep — Keep it uninstalled and use openly licensed footage instead.
Reason: Nothing is lost, and installing it needs a maintained source we do not have yet.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.
If silent: keep on 2026-11-27
Packet: work/YTDLP-TARGET/BRIEF.md
Explainer: work/YTDLP-TARGET/page.html
Details: IDs, paths and evidence for the agent that acts on the answer.
```

IDs match `[A-Z0-9][A-Z0-9_-]{0,127}`; option keys match `[a-z0-9-]{1,32}`, and `other` is dropped if listed: the Hub adds its own free-text answer to every ask, and no item is refused for naming it. `Recommended:` names one option; a decision needs `Packet:`. A notice has no `Recommended:`; a secret has `Secret: <where it goes>`, never the value. An item about a whole commission gains `Commission: <id>`.

`If silent: <option key> on <YYYY-MM-DD>` states what silence does, and the courier makes it so: the option is the Hub ask's default answer and the end of that day (UTC) its deadline, so silence applies it and the page says so. A bright-line item (spend, outside contact, secrets, unrecoverable deletion) has `If silent: none`: silence never approves it. A direction question carries a default like any other. `needs_you.py write` asks for the line only on items it adds, never on items already in the target, and never invent a default for an item already sent. Write it through the validator: `python3 -I scripts/needs_you.py write --file <draft> --section "Needs you" --target topics/<t>/DECISIONS.md`. `needs_you.py open` lists every open decision from files alone. Every reply to the operator opens with the open items.

## The packet behind a decision

The raiser writes `work/<ID>/BRIEF.md` and its page beside it. A fresh agent executes it on the answer with no other context:

```markdown
Checked: <UTC> <the premise, verified live, and how>
## Destination / ## What changes   identities, paths, the cause
## Commands   each: directory, as whom, the command, what it prints, its undo
## On each answer
- `<key>`: what it settles, what executes and its proof, or `nothing`   (one per option)
- `other`: what the operator's own words settle
```

`needs_you.py write` refuses a decision without it and the courier holds it back, because its executor would otherwise re-derive the decision. Every item's page (`notification-hub` skill, "Make the page") sits beside its brief and is named on `Explainer:`: one card per thing asked, how long it takes, what a no costs.

## Proposals

A `propose-then-build` topic sends each new maintained capability under `## Proposals awaiting approval` as `### PROPOSAL-<TOPIC>-<slug> — <title>` with: what the operator can do afterwards; what Vrooli lacks (the command that proves it); what the prototype really did and what it faked; what gets built, by resource and scenario name, with magnitude and showcase; a recommendation with options (build, smaller first step, defer, decline); and what the topic does meanwhile. Run `ecosystem-fit` on the shape first. On approval the proposal becomes a `build` workstream.

## Deliver and reconcile

- Research Desk forwards items and records `delivered <ID>` in `OPERATOR.md`. Not delivered by the second review: fix what its row in `research-desk needs-you` says; never send a decision by hand.
- An answer arrives as an `answer` entry in the owning level's `INBOX.md`: add it verbatim to the packet under `## Operator answer`, start the executor with exactly what `needs_you.py spawn <ID>` prints, move the item to `history/DECISIONS.md`, drain the entry.
- At most one reminder per item a day, only with new context. Answered, dismissed, withdrawn and expired items are distinct and none stays actionable; unreadable is `unknown`.
