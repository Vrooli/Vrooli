## Practice focus: Workshop Decision Sync

Conversational decision triage for Swarm Manager workshop questions. Use this when the operator wants to answer backlog workshop decisions without going through the Swarm Manager UI.

Required reading:
- `prompt-manager skill read skill-principles`
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

---

### 2. Operating Principle

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
- running actions outside the answer / skip / clarify-spawn surface

---

### 3. Required Inputs

Start by reading the staged handoff:

```bash
Read the file: scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/last-handoff.md
```

Then fetch live queue state:

```bash
swarm-manager backlog pending-questions --source workshop --json
```

Do not trust the handoff blindly. Treat it as a performance optimization, not as the source of truth.

---

### 4. Freshness Rules

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
- Continue the session; never fail closed just because the handoff is stale.

---

### 5. Session Structure

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

### 6. Conversation Pattern

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

### 7. Persisting an Answer

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

The implementation surface may use the CLI, API client, or file helpers available in the environment, but the semantic contract must remain fetch-patch-save through `workshop/save`.

After a successful save:
- confirm the selected option briefly
- advance to the next decision in the current item if one exists
- otherwise advance to the next item

If save fails:
- tell the operator plainly
- do not pretend the answer persisted
- stay on the same decision until it is retried or skipped

---

### 8. Async Clarification Flow

If the operator asks something the brief cannot already answer:

1. Offer async clarification:
   - `I don’t have enough context to answer that cleanly. I can spawn a clarification and move on.`
2. If the operator accepts, run:

```bash
swarm-manager backlog clarify --kind <kind> --name <name> --round <round> --item <item_id> --message "<operator question>"
```

3. Mark the current decision mentally as clarification-pending.
4. Move on immediately. Do not wait for resolution inside this session.

On a later prep run, the resolved thread's `LatestImpact.ContextNote` should be fed back into the next staged brief.

---

### 9. Skip Rules

Support these operator commands exactly:

- `skip`
  - skip only the current decision
- `skip item`
  - skip every remaining decision in the current backlog item
- `skip initiative`
  - skip every remaining decision in the current initiative

After any skip, state what is being skipped and move on without argument.

---

### 10. Completion

At the end of the session:

- report how many decisions were answered
- report any items or initiatives skipped
- report any clarifications spawned
- do not create follow-up work automatically

If no valid live decisions remain after freshness checks, say so plainly:

`There are no live open workshop decisions to walk through right now.`

---

### 11. Quality Bar

This skill should feel like a professional operator workflow:

- fast
- bounded
- context-rich
- non-repetitive
- explicit about what persisted and what did not

Never blur the line between recommendation and decision. The operator decides.

See `TEST-PLAN.md` before merging behaviour changes.
