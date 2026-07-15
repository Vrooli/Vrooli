## Practice focus: Workshop Decision Sync

Conversational decision triage for Swarm Manager workshop questions. Use this when the operator wants to answer backlog workshop decisions without going through the Swarm Manager UI.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `prompt-manager skill read swarm-manager-backlog-tools`

---

### 1. When to Use This Skill

Use this skill when:
- The operator wants a short decision session for open workshop questions
- The operator says they want to answer Swarm Manager decisions conversationally
- The operator wants a lighter-weight complement to UI-based workshop review

Do not use this skill for:
- prompt-manager team decisions such as morning-vision-walk phases
- creating or restructuring backlog items
- brainstorming entirely new initiatives
- portfolio or initiative-wide progress triage
- choosing, switching, or running Swarm Manager operating modes

This skill may be used directly by the operator or invoked as a focused decision-drain routine by a broader Swarm Manager operations session. In either case, its authority is the same: walk through live workshop decisions and persist only the operator's explicit answers.

---

### 2. Caller Contract

When another skill or session invokes this skill, the caller may provide optional scope:

- `initiative`: drain only decisions for one initiative
- `kind` + `name`: drain only one backlog item
- `max_decisions`: stop after this many live decisions are answered, skipped, or clarified

If no scope is provided, use the staged handoff plus live queue priority order.

Caller responsibilities:
- decide whether decision-draining is the right next operator workflow
- perform portfolio, initiative, or operating-mode analysis before or after this skill runs
- handle any follow-up signals returned by this skill

This skill's responsibilities:
- revalidate staged briefs against the live queue
- present one scoped workshop decision at a time
- persist answers through the workshop-save contract
- return a concise completion handoff to the caller/operator

Do not let caller scope expand the mutation surface. A scoped invocation may reduce which decisions are shown; it must not allow backlog restructuring, initiative metadata changes, mode switches, phase starts, queueing, or proposal application.

---

### 3. Operating Principle

This is an operator-decision surface, not an autonomous decision-maker.

You present the decision, relevant context, and bounded options. The operator chooses. You persist exactly what they chose using the existing Swarm Manager workshop-round contract.

Allowed actions:
- read prep handoff
- read live pending workshop questions
- answer one decision
- skip one decision
- skip one backlog item
- skip one initiative
- spawn a clarification and move on

Forbidden actions:
- choosing an option on the operator's behalf
- creating, deleting, or reprioritizing backlog items
- modifying initiative metadata
- switching initiative operating modes
- starting, refreshing, canceling, completing, or reconciling operating-mode rounds
- applying backlog-sync proposals
- running actions outside the answer / skip / clarify-spawn surface

If decision triage reveals that an item is stale, mis-scoped, blocked by a mode mismatch, or likely better handled through an initiative operating mode, capture that as a follow-up signal in the completion handoff. Do not act on it inside this skill.

---

### 4. Required Inputs

Start by reading the staged handoff:

```bash
Read the file: scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/last-handoff.md
```

Then fetch live queue state:

```bash
swarm-manager backlog pending-questions --source workshop --json
```

If the caller supplied scope, apply it to the live queue before presenting decisions:

```bash
swarm-manager backlog pending-questions --source workshop --initiative "<initiative>" --json
```

For item scope, fetch the live queue and filter to the exact `kind` and `name`. The live queue is authoritative; the staged handoff only enriches matching decisions.

Do not trust the handoff blindly. Treat it as a performance optimization, not as the source of truth.

---

### 5. Freshness Rules

For each staged brief:

1. Match it to the live queue by `kind`, `name`, `round`, and `item_id`.
2. Recompute the decision hash from live `topic + text + context + options`.
3. Drop the brief if:
   - the live decision is missing
   - the hash changed
   - the live decision now has `selected` populated

If the handoff is missing or every staged brief is stale, say so explicitly and run lazy inline prep:

- Tell the operator: `The staged handoff is missing or stale. I’m doing lazy prep now, about 60 seconds.`
- Use the live pending-questions output directly.
- Limit lazy prep to 5-8 decisions.
- If caller scope is narrower than the staged handoff, prefer scoped live decisions over unscoped staged briefs.
- Continue the session; never fail closed just because the handoff is stale.

---

### 6. Bounded Drain Rules

This skill should support both open-ended and bounded drains:

- open-ended: continue until no scoped live decisions remain or the operator stops
- initiative-bounded: drain only decisions under one initiative
- item-bounded: drain only decisions under one backlog item
- count-bounded: stop after `max_decisions` live decisions have been answered, skipped, or sent to clarification

When a bound is reached, stop cleanly and report remaining live decisions in the completion handoff. Do not silently continue into another initiative or item just because staged briefs are available.

---

### 7. Session Structure

Walk the operator through decisions grouped by:

1. initiative
2. backlog item
3. decision

Present only one decision at a time.

Maintain a current-focus stack:
- current initiative
- current backlog item
- current decision

If the operator asks a clarifying question, answer only from the current decision context. Do not pivot to a different decision or initiative unless the operator explicitly says to skip or switch.

When the initiative changes, give the initiative summary once.
When the backlog item changes, give the backlog summary once.
Do not repeat those summaries on every decision inside the same item.

---

### 8. Conversation Pattern

For each decision:

1. State where the decision lives:
   - initiative
   - backlog item
   - decision topic
2. Give the decision text/context in concise conversational language.
3. List the options, preserving recommendation flags if present.
4. Mention any clarification note from prior async clarification work.
5. Ask the operator for one of:
   - an answer
   - `skip item`
   - `skip initiative`
   - `clarify`

Do not overtalk. This skill is for quick operator throughput.

---

### 9. Persisting an Answer

When the operator chooses an option:

1. Fetch the current round JSON for the item.
2. Patch only the matching decision item:
   - set `selected`
   - set `freeform` only when applicable
   - set `notes` if the operator gave rationale worth preserving
3. Save the full round through the existing endpoint:

```bash
POST /api/v1/backlog/{kind}/{name}/workshop/save
```

The implementation surface may use the CLI, API client, or file helpers available in the environment, but the semantic contract must remain fetch-patch-save through `route:workshop/save`.

After a successful save:
- confirm the selected option briefly
- advance to the next decision in the current item if one exists
- otherwise advance to the next item

If save fails:
- tell the operator plainly
- do not pretend the answer persisted
- stay on the same decision until it is retried or skipped

---

### 10. Async Clarification Flow

If the operator asks something the brief cannot already answer:

1. Offer async clarification:
   - `I don’t have enough context to answer that cleanly. I can spawn a clarification and move on.`
2. If the operator accepts, run:

```bash
swarm-manager backlog clarify --kind "<kind>" --name "<name>" --round "<round>" --item "<item_id>" --message "<operator question>"
```

3. Mark the current decision mentally as clarification-pending.
4. Move on immediately. Do not wait for resolution inside this session.

On a later prep run, the resolved thread's `LatestImpact.ContextNote` should be fed back into the next staged brief.

---

### 11. Skip Rules

Support these operator commands exactly:

- `skip`
  - skip only the current decision
- `skip item`
  - skip every remaining decision in the current backlog item
- `skip initiative`
  - skip every remaining decision in the current initiative

After any skip, state what is being skipped and move on without argument.

---

### 12. Completion

At the end of the session:

- report how many decisions were answered
- report how many decisions were skipped
- report any items or initiatives skipped
- report any clarifications spawned
- report how many staged briefs were dropped as stale, if known
- report how many live decisions remain in the active scope, if known
- report notable follow-up signals for the caller/operator
- do not create follow-up work automatically

If no valid live decisions remain after freshness checks, say so plainly:

`There are no live open workshop decisions to walk through right now.`

When invoked by a broader operations session, end with a compact handoff in this shape:

```json
{
  "skill": "workshop-decision-sync",
  "scope": {
    "initiative": null,
    "kind": null,
    "name": null,
    "max_decisions": null
  },
  "answered_decisions": 0,
  "skipped_decisions": 0,
  "skipped_items": [],
  "skipped_initiatives": [],
  "clarifications_spawned": [],
  "stale_briefs_dropped": 0,
  "remaining_live_decisions": 0,
  "notable_followup_signals": []
}
```

Use `notable_followup_signals` only for observations the caller may need, such as:
- decision triage suggests the backlog item is stale or mis-scoped
- multiple decisions point to the same unresolved initiative-level question
- the operator repeatedly skips an initiative
- the item appears better suited to initiative-level operating-mode work

Do not turn these signals into actions inside this skill.

---

### 13. Quality Bar

This skill should feel like a professional operator workflow:

- fast
- bounded
- context-rich
- non-repetitive
- explicit about what persisted and what did not

Never blur the line between recommendation and decision. The operator decides.

See `TEST-PLAN.md` before merging behaviour changes.
