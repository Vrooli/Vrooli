# Workshop Decision Sync — Manual Test Plan

Operator-driven smoke checklist. Run after material behaviour changes to either:

- the `workshop-decision-sync` skill (this directory's `SKILL.md`), or
- the `workshop-decision-prep` heartbeat
  (`scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/HEARTBEAT.md`).

The Go integration test
`scenarios/swarm-manager/api/internal/backlog/workshop_decision_prep_integration_test.go`
is the daily-CI safety net for the data seam. This plan is the periodic UX
validation — neither substitutes for the other.

---

## 1. Setup

Goal: a Swarm Manager state with three backlog items across two initiatives,
each item carrying two unresolved workshop decisions (six open decisions
total). Mirror the structural fixture used by the integration test
(`seedTwoInitiativesThreeItems`).

```bash
# Pick a clean workspace.
swarm-manager backlog list --json | jq '.items | length'   # baseline

# Create three idea items across two initiatives.
swarm-manager backlog create --kind idea --name alpha --title Alpha \
  --initiative north-star --priority 1
swarm-manager backlog create --kind idea --name beta  --title Beta \
  --initiative north-star --priority 2
swarm-manager backlog create --kind idea --name gamma --title Gamma \
  --initiative side-quest --priority 3
```

Seed each item with a round-001 containing two unresolved decisions. Either
write `workshop/round-001.json` directly (matching the shape used in the Go
test) or run a real workshop heartbeat with seeded inputs. Confirm the queue
state:

```bash
swarm-manager backlog pending-questions --source workshop --json \
  | jq '[.items[].questions | length] | add'
# Expected: 6
```

Trigger the prep heartbeat:

```bash
prompt-manager team heartbeat-trigger director-swarm workshop-decision-prep
```

> **What would invalidate this section:** the `pending-questions` CLI / endpoint
> shape changes, the heartbeat trigger command is renamed, or `workshop-decision-prep`
> moves out of `director-swarm`. Refresh the commands rather than discarding
> the section.

---

## 2. Inspection of `last-handoff.md`

After the heartbeat completes, open
`scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/last-handoff.md`
and confirm:

- [ ] Sections grouped in the order: **initiative → backlog item → decision**
      (matches `HEARTBEAT.md` "Output Contract").
- [ ] Each decision block contains all five machine-checkable fields:
      `kind`, `name`, `round`, `item_id`, `content_hash`.
- [ ] `recommendation_surface: conversational` is present on every decision.
- [ ] `options:` preserve any `recommended=true` flags from the live round
      (do not invent recommendations; do not drop them).
- [ ] Every decision has `anticipated_questions` with 2–4 Q/A pairs grounded
      in the item description / decision context (no hallucinated facts).
- [ ] `Generated at:` carries an RFC3339 UTC timestamp.

> **What would invalidate this section:** HEARTBEAT.md changes the output
> contract (e.g. renames a field, moves grouping levels). Update the
> bullets here to match before merging that change.

---

## 3. Conversational Walk-through

Open a focused operator session that loads the skill:

```bash
prompt-manager skill read workshop-decision-sync
```

Drive the session and verify:

- [ ] **(a) Initiative summary printed once per initiative transition.** When
      moving from the first decision in `north-star` to the second decision
      inside the same initiative, the skill must NOT repeat the initiative
      summary. When it eventually transitions to `side-quest`, the new
      initiative summary appears exactly once.
- [ ] **(b) Backlog summary printed once per item transition.** Same rule as
      above, scoped to backlog item.
- [ ] **(c) Focus stack — clarifying questions.** Mid-decision, ask the skill
      a clarifying question about that decision. The reply must answer from
      the current decision context only. It must NOT pivot to a different
      decision or initiative unless you say `skip` / `skip item` / `skip
      initiative` / `switch`.
- [ ] **(d) Async clarify — offer.** Ask something the staged brief cannot
      cleanly answer (e.g. an implementation-detail question outside the
      brief's scope). The skill must offer to spawn a `swarm-manager backlog
      clarify` and explicitly say it will move on without waiting.
- [ ] **(e) Async clarify — accept.** Accept the offer. The skill must run
      the CLI and proceed to the next decision in the same turn (or the
      one immediately after), without blocking on the clarification result.
- [ ] **(f) No autonomous answers.** At no point does the skill choose an
      option on the operator's behalf or persist a save without an explicit
      operator pick.

> **What would invalidate this section:** SKILL.md changes the focus-stack
> contract, the offer-language for clarify, or the skip command set. Update
> the bullets to match (and tighten / loosen the operator script as needed).

---

## 4. Round-trip Proof

Inside the same conversational session:

- [ ] Pick one decision and answer it conversationally (provide an option
      key like `A`).
- [ ] The skill confirms persistence briefly and advances to the next
      decision in the same item (or the next item if the answered decision
      was the item's last open one).
- [ ] After the session, run:

```bash
swarm-manager backlog pending-questions --source workshop --json \
  | jq '[.items[].questions | length] | add'
# Expected: 5 (one fewer than the original 6)
```

- [ ] Re-trigger the prep heartbeat and confirm the answered decision is
      no longer in `last-handoff.md`.

> **What would invalidate this section:** the persistence path moves off
> `POST /api/v1/backlog/{kind}/{name}/workshop/save`, or the prep heartbeat
> stops running freshness checks against the live `pending-questions`
> output. Update the commands and the expected-count math to match.

---

## 5. Pass / Fail Rubric

A run **passes** only when ALL of the following are true:

- Every checkbox in sections 2, 3, and 4 is checked.
- No decision was answered without an explicit operator pick.
- No initiative or item summary was repeated within the same parent group.
- A clarification offer was made for the unanticipated question and, upon
  acceptance, ran without blocking the session.
- The post-session pending-questions count drops by exactly the number of
  decisions the operator answered (no stragglers, no double-drops).

A run **fails** if any of:

- The skill answers a decision autonomously.
- The skill pivots focus mid-decision without an explicit operator command.
- `last-handoff.md` is missing one of the machine-checkable fields, or its
  grouping order is wrong.
- The async clarify path blocks the session waiting for a result.
- `pending-questions` does not drop the answered decision.

If the run fails, file the failure mode against either `SKILL.md` or
`HEARTBEAT.md` (whichever owns the broken contract) before merging the
behavioural change that the smoke run was gating.

> **What would invalidate this section:** the contract surface changes (a
> new operator command, a relaxed "skill may answer" rule, etc.). Update
> the pass/fail bullets so the rubric reflects what "good" actually means
> after the change.
