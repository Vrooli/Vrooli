# Team Docs Patterns

Reference for deciding whether a team should own shared documentation — and if so, in which of two distinct patterns. Cite this file when designing a new team or restructuring an existing one's doc surface.

This is canon. Cites `LAYERS.md` for where each layer lives and `PRIMITIVES.md` for the underlying definitions.

---

## Two patterns, not one

Teams that benefit from shared docs use one (or both) of these patterns. They are **not** interchangeable; each has its own write rules, readership, and health signals. Do not design "shared docs" generically — pick a pattern.

### Pattern A — Plan-of-record (canonical docs)

The docs **are** the plan. The team and other consumers read them as the durable source of truth.

- **Who writes:** operator-curated via approved decisions. Agents propose diffs; they never edit directly.
- **Who reads:** owning team + other teams + scenarios (cross-team readership is typical but not required).
- **Lifespan:** durable. Entries persist and evolve.
- **Growth direction:** grows over time as the plan expands.
- **Health signal:** entries are *stable*. High churn = something wrong with the plan.
- **Shape:** structured. One entity per file where possible, with an index at the top.
- **Example:** `docs/monetization/` — catalog bundles and add-ons are per-entity files; `CATALOG.md` and `REVENUE_LINES.md` are indexes. Also `docs/agent-system/` (this folder).

### Pattern B — Working notebook (debt docs) — historical

A team-scoped scratchpad for patterns the team has not yet crystallized. Entries are debt: they exist because the permanent solution (a skill, scenario feature, or config change) doesn't yet exist.

In the current architecture, **most working-notebook material has been replaced by the inbox-router-drain pattern**: team knowledge entries under topic prefixes, drained by router skills. See `INTAKE_PIPELINE.md` for the live pattern.

Notebook-as-markdown-file survives only in two narrow cases:

1. **Drafts adjacent to canon** — a `docs/<domain>/drafts/` subfolder where structure is being workshopped before being promoted to a permanent PoR file. These are short-lived: a draft either becomes canon or is deleted.
2. **Per-team running notes** that the team itself owns and that don't fit the inbox-router shape (e.g., a sequence of long-form workshop notes that aren't observation-shaped). These should still have a curator role and a clear retirement path; without those, they are guaranteed to become a debt junkyard.

When neither narrow case applies, prefer team knowledge entries under a topic prefix and a router skill that drains them.

---

## When to use Plan-of-record

Use it when **at least one** of these holds:

1. **Other teams/scenarios consume it.** (Strongest signal.) If another system reads it, it must exist and be authoritative.
2. **The operator needs a durable strategic frame to reason across entries at a long horizon.** Even with no external readers.
3. **The team itself needs a durable frame to anchor decisions against**, and would drift without one.

If none hold, skip plan-of-record. Decisions and knowledge entries already cover ephemeral insight. A doc nobody reads is a stale shrine.

---

## When to use a Working notebook (the narrow cases)

Use it **only if all four** hold:

1. **A curator role exists** (a team member whose job includes retirement). Without a curator, the notebook is a growing debt pile — worst of both patterns.
2. **Pattern-level insight arrives faster than decisions alone can absorb it.** Low-volume teams don't need this surface.
3. **Insights are crystallizable** — there's a realistic path for entries to become skills, scenario features, or config. Pure situational judgment belongs in knowledge entries, not a notebook.
4. **Existing surfaces don't already cover it.** If the team has knowledge entries, decisions, run lessons, or audits that handle the same material, a notebook duplicates work. **The inbox-router-drain pattern (`INTAKE_PIPELINE.md`) covers most signal-shaped material; if that pattern fits, use it instead of a notebook.**

If any of these fail, skip the notebook.

---

## The four axes that distinguish them

| Axis | Plan-of-record | Working notebook |
|---|---|---|
| Direction of flow | Operator → team (top-down) | Team → operator (bottom-up) |
| Write gate | Approval-required | Append-anyone |
| Ideal trajectory | Grow and stabilize | Shrink (entries retire into structure) |
| Cross-team readability | Yes — other teams/scenarios consume it | No — private working memory |

---

## Teams with both patterns

A mature team may run both. They coexist under two hard rules:

1. **Separate folders.** Plan-of-record lives at `docs/<team>/`; notebook lives at `docs/<team>/drafts/` or `docs/<team>/notebook/`. The folder separation is the primary disambiguator.
2. **An entry never lives in both at once.** When a notebook pattern is promoted into plan-of-record (or into permanent structure), it is **deleted from the notebook** as part of the promotion decision. Double-residency muddies the source of truth and both surfaces rot.

Supporting conventions:

- **Separate write rules**, stated explicitly in the team's `TEAM.md`. Plan-of-record = approval-gated; notebook = append-anyone.
- **Curator owns the promotion path:** a notebook entry has exactly three outcomes — promoted to plan-of-record, promoted to permanent structure (skill / scenario / config), or retired. Never "leave it and forget."
- **Use the right permanent structure** (per `LAYERS.md`): truth → plan-of-record, judgment → skills, deterministic execution → Actions, implementation → CLIs, missing capability → backlog/capability-gap, unverified learning → inbox/synthesis.
- **Cross-references are one-directional:** plan-of-record can reference notebook entries; notebook entries must NOT be cited as authoritative by other teams. If something is authoritative, it should already have been promoted.

---

## Traps to avoid

1. **Plan-of-record as stale shrine.** Adding docs "because it feels organized" when nothing consumes them. Readership is the load-bearing property; without it, decisions alone are better.
2. **Notebook without a curator.** Guaranteed to become a debt junkyard. If you can't name the curator, don't create the notebook.
3. **Monolithic plan-of-record docs when per-entity files are possible.** Editing a shared file to add/retire one entity invites accidental damage to others. Prefer one-entity-per-file with a generated-feeling index.
4. **Blurred write rules in a both-patterns team.** If members edit plan-of-record directly or propose changes to the notebook via decisions, the separation collapses. State the rules in `TEAM.md`.
5. **Treating "shared docs" as a single concept.** The generic term hides the pattern distinction, which is the most important thing to communicate when designing a team.
6. **Notebook-as-permanent-layer thinking.** The historical "hot buffer / living notebook / permanent structure" three-tier model is retired in favor of inbox/synthesis (per `INTAKE_PIPELINE.md`). Do not treat a notebook as anything other than transient debt.

---

## Quick decision checklist

When designing or restructuring a team:

1. **Does the team produce durable intent that something else reads?** → Plan-of-record.
2. **Does the team need to absorb signal-shaped observations that route to multiple surfaces?** → Inbox/synthesis pattern (`INTAKE_PIPELINE.md`), not a markdown notebook.
3. **Does the team generate long-form workshop material that wants to become structure, AND has a curator?** → Notebook (narrow case).
4. **Both PoR and notebook?** → Both, with the hard separation rules above.
5. **None of the above?** → No shared docs. Use decisions and knowledge entries.

When auditing an existing team:

- Plan-of-record without readers → challenge its existence.
- Notebook without a curator → challenge its existence or propose creating the curator role.
- Notebook material that is signal-shaped (observations awaiting routing) → migrate to the inbox/synthesis pattern.
- Entries living in both → challenge the duplication; pick one home.
- Monolithic plan-of-record where per-entity files would fit → propose splitting.

---

## Related skills

- `team-coordination-independent` / `team-coordination-leader-led` / `team-coordination-peer` — runtime coordination (orthogonal to doc architecture).
- `team-tool-mapping` — tool/skill assignment when team structure changes touch tool wiring.
- `documentation-health` — durable snapshot and clarity discipline for any doc surface.
