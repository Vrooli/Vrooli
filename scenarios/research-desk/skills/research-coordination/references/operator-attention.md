# Operator attention

The operator's only job in research is answering decisions. They read on a phone, with no
context. Ask rarely, ask well, and never ask what agents can resolve.

## Ask or not

- Agents can resolve it, or restore a permitted path: assign that work; do not ask.
- Only the operator holds it (money, credentials, rights, private facts, taste, an explicitly
  human-only decision): prepare one concrete request and keep every other workstream moving.
  Before a credential or account ask, read `vrooli credentials list` (names and state only) and
  ask only for what is missing.
- Not reviewable yet: finish the preparation first.
- Later evidence removed the need: mark it superseded; do not send it.

## Needs-you items

Each level keeps its open decisions ([levels](../../../../../docs/agent-system/LEVELS.md)): a
topic in `DECISIONS.md` (`## Needs you`), the supervisor in the research home's `ATTENTION.md`
(`## Actionable now`; `## Proposals awaiting approval` above it). Research Desk reads them as a file
contract (`scenarios/research-desk/docs/reference/FILE-CONTRACT.md` §5):

```markdown
### YTDLP-TARGET — Keep the video download tool uninstalled?
Two to six plain sentences for a stranger: what this is about, why only the operator
decides, what each answer leads to, what happens without one. No IDs, run names or paths.
Recommended: keep — Keep it uninstalled and use openly licensed footage instead.
Reason: Nothing is lost, and installing it needs a maintained source we do not have yet.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.
Packet: work/YTDLP-TARGET/BRIEF.md
Explainer: work/YTDLP-TARGET/page.html
Details: IDs, paths and evidence for the agent that acts on the answer.
```

IDs match `[A-Z0-9][A-Z0-9_-]{0,127}`; option keys match `[a-z0-9-]{1,32}`;
`Recommended:` names one option, and a decision needs `Packet:`. A notice has no `Recommended:`;
a secret has `Secret: <where it goes>`, never the value. An item about a whole commission gains
`Commission: <id>`. Edit a copy, then write it through the validator:
`python3 -I scripts/needs_you.py write --file <draft> --section "Needs you" --target topics/<t>/DECISIONS.md`.
`needs_you.py open` lists every open decision from files alone. Every reply to the operator opens
with the open items, never a clause in a status paragraph.

## The packet behind a decision

The raiser writes `work/<ID>/BRIEF.md` and its page beside it, never a workstream brief. A fresh
agent executes it on the answer with no other context:

```markdown
Checked: <UTC> <the premise, verified live, and how; a credential: what `vrooli credentials list` shows>
## Destination / ## What changes   identities, paths, the cause; earlier asks like it as pointers
## Commands   each: directory, as whom, the command, what it prints, its undo; `operator-only` where so
## On each answer
- `<key>`: what it settles, what executes and its proof, or `nothing`   (one per option)
- `other`: what the operator's own words settle
```

`needs_you.py`, the desk and the steward refuse a decision without it. When scope or premise changes,
update the brief and `Checked:` with the item.

## Decision pages

Every item's page (`notification-hub` skill, "Make the page") sits beside its brief and is named
on `Explainer:`. It shows one card per thing asked (its state, what it
unlocks, numbered steps), how long it takes, and what a no costs.

## Proposals

A `propose-then-build` topic sends each new maintained capability as a proposal under
`## Proposals awaiting approval`, headed `### PROPOSAL-<TOPIC>-<slug> — <title>`. Make it easy
to say yes:

- **In plain words**: what the operator can do afterwards that they cannot today.
- **What Vrooli lacks**: each missing piece, and whether a resource or scenario really owns it
  (the command that proves it). Software a prototype installed for itself is still missing.
- **What the prototype really did**: what ran, on what, and what was simulated.
- **What gets built**: every resource, scenario and extension by name, with magnitude and showcase.
- **Recommendation and options**: build, a smaller first step (say what it leaves out), defer,
  decline; and what the topic does meanwhile (it never idles).

Size the build to the need: vendor software Vrooli keeps running belongs in a new resource, state
no owner holds in a new scenario. "No new scenario" is a property of a design, not a merit. Run
`ecosystem-fit` on the shape first. On approval the proposal becomes a `build` workstream at the
proposed scale; a decline is recorded under Not this.

## Review requests

A request to judge finished work is one self-contained HTML page: what was asked for, the
deliverable and its full source in collapsible blocks, what it was tested on and who made that
target, hours spent, weak points, and verdict controls. Apply the trade's bar first: would a
professional send it to a paying client? If not, finish it.

## Deliver and reconcile

- Research Desk forwards `DECISIONS.md` and `ATTENTION.md` items and records `delivered <ID>` in
  the commission's `OPERATOR.md`. Not delivered by the second review: fix what its row in
  `research-desk needs-you` says. Never send a decision by hand; that skips every check.
- `research-desk needs-you` reads ask states: answered, dismissed, withdrawn and expired are
  distinct and none stays actionable; unreadable is `unknown`, never "pending".
- An answer arrives as an `answer` entry in the owning level's `INBOX.md`. Add it verbatim to the
  packet under `## Operator answer`, start the executor with exactly what `needs_you.py spawn
  <ID>` prints, move the item to `history/DECISIONS.md`, then drain the entry. Silence, expiry
  or a delivery receipt never grants approval.
- At most one reminder per item a day, only with new context or a real deadline.
