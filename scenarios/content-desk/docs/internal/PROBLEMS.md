# Problems — Content Desk

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-07-28 — Retired scenario leaves dangling skill references

`campaign-content-studio` was moved out of the repo to
`/tmp/campaign-content-studio-retired-2026-07-27`. Six skill files plus one
marketing catalogue document still reference it: its own skill
(`skill.json`, `SKILL.md`), and cross-references inside the `funnel-builder`,
`social-media-scheduler`, `video-studio`, and `seo-optimizer` skills, plus
`docs/marketing/catalogs/rich-media/README.md`.

**Impact:** agents discovering those skills will be pointed at a scenario that
no longer exists. Not blocking for this scenario's work, but it is live
misinformation in the skill index.

**Next step:** retire the `campaign-content-studio` skill and repoint the five
cross-references, once `content-desk` is real enough to be the referent.

### 2026-07-28 — `video-studio` skill points at a scenario that was never built

Discovered while mapping media boundaries. The `video-studio` skill is marked
`active` in the registry and its `skill.json` carries `scenario: null`; no
`scenarios/video-studio` exists. It is unrelated to this scenario's scope but
is the same class of drift as the entry above, and it will mislead any agent
that discovers it.

**Next step:** an owner decision — build, retire, or mark planned. Not this
scenario's call.

### 2026-07-28 — Claim taxonomy is uncalibrated

The eight claim kinds and the evidence-strength rules were designed against
worked examples, not against a real draft. What counts as a claim may be wrong
in both directions: too broad makes the gate exhausting, too narrow lets a
misleading statement through as "framing".

**Impact:** the taxonomy is the load-bearing product decision and it has never
met real input.

**Next step:** publish one post manually end to end and classify its assertions
by hand before implementing the gate. This is deliberately sequenced ahead of
implementation in the PRD launch plan.

### 2026-07-28 — business-health API degrades under repeated wizard calls

During charter authoring, `StartSession` and `SubmitAnswers` slowed from
sub-second to 36–39 seconds, causing client-side EOF errors that looked like
failures while the server was in fact completing the work. A scenario restart
cleared it. The API process had been up since 2026-07-18.

**Impact:** not a `content-desk` defect, but it cost a confusing debugging
cycle and would mislead the next agent to author a charter.

**Next step:** file to scenario-qa against `business-health` if it recurs;
capture is the restart, and the symptom to recognise is slowness rather than
error.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
