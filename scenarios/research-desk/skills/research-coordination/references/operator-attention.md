# Operator attention

The operator's only job in research is answering decisions. They read on a phone, with no
context. Ask rarely, ask well, and never ask what agents can resolve.

## Ask or not

- Agents can resolve it, or restore a permitted path: assign that work; do not ask.
- Only the operator holds it (money, credentials, rights, private facts, taste, an explicitly
  human-only decision): prepare one concrete request and keep every other workstream moving.
- The proposal or evidence is not reviewable yet: finish the preparation first.
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
`Commission: <id>`; one you delivered yourself, `Delivered: <route> <UTC>`. Edit a copy, then write
it through the validator, which replaces the file:
`python3 -I scripts/needs_you.py write --file <draft> --section "Needs you" --target topics/<t>/DECISIONS.md`.
`needs_you.py open` lists every open decision from files alone. Every reply to the operator opens
with the open items; a clause in a status paragraph is not a delivery.

## The packet behind a decision

Whoever raises a decision writes its packet; they understand it best. `Packet:` points
to a brief a fresh agent can execute on approval with no other context: what gets built or
changed, where, the scale, the evidence and starting points, the showcase, and what each option
changes.

## Decision pages

Every item gets a page. Make it by the `notification-hub` skill, "Make the page"
(`prompt-manager skill read notification-hub`, or `scenarios/notification-hub/skills/notification-hub/SKILL.md`),
save it inside the research home, and name it on the item's `Explainer:` line; Research Desk
sends it with the item (experience engineering's `present-ux-decisions` skill is a model).

## Proposals

A `propose-then-build` topic sends each new maintained capability as a proposal under
`## Proposals awaiting approval`, headed `### PROPOSAL-<TOPIC>-<slug> — <title>`. Make it easy
to say yes:

- **In plain words**: what the operator can do afterwards that they cannot today.
- **What Vrooli lacks**: each missing piece, and whether an existing resource or scenario really
  owns it (the command that proves it) or nothing does. Software a prototype installed for itself
  is still missing.
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
  the commission's `OPERATOR.md`. With no such entry by the second review, deliver once yourself:
  `notification-hub conversations ask` with the plain question, `--big-picture` (for a cold
  reader), `--technical-details` (paths, IDs), at least two `--option key=label`,
  `--recommend` with a reason, `--explainer <page>`, a stable `--idempotency-key` and
  `--subject-key`, and `--expires`.
  If that fails, mark the item "not delivered"; your replies carry it.
- Read the ask's state before every delivery and at the next unit: `research-desk needs-you`.
  Answered, dismissed, withdrawn and expired are distinct; none stays actionable; unreadable is
  `unknown`, never "pending".
- An answer arrives as an `answer` entry in the owning level's `INBOX.md`. Add it verbatim to the
  packet under `## Operator answer`, start the executor ("Execute the approved decision `<ID>`.
  Read `<abs>/BRIEF.md`; the answer is `<key>`." plus the stance), move the item to
  `history/DECISIONS.md`, then drain the entry. Silence, expiry or a delivery receipt never grants
  approval.
- At most one reminder per item per day, and only with new context or a real deadline.
