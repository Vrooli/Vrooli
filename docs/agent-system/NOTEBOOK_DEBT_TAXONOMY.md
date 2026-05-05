# Notebook Debt Curation Taxonomy

Cross-team-readable canon for how notebook-debt curators (members that drain `<team>/notebook/*` prefixes) classify and resolve entries. Human-readable view of `path:docs/agent-system/notebook-debt-taxonomy.json`.

**Owner team:** meta-optimization. **Status:** canon. Operator-curated via meta-optimization decisions.

Cited by:
- `topics.json` for any member whose `intake[].taxonomy = "notebook-debt"` — currently `marketing-crew/brand-manager` (`marketing/notebook/*`) and `meta-optimization/debt-curator` (`meta-optimization/notebook/*`).

Notebook-debt curation is a *generic* drain pattern: the entries are not new external signals (so no classifier skill is required) but they still need an explicit decision-style outcome on each pass. The curator member iterates, picks one of {promotion-candidate, retirement-candidate, still-debt}, and acts.

**Use the notebook only when no typed inbox fits.** If an observation is a bug, capability gap, friction report with a concrete owner, or any other signal whose destination is known, write it to the appropriate typed inbox via `prompt-manager team knowledge-add` — see `path:docs/agent-system/TOPICS.md` for the registry. The notebook absorbs only the residual: half-formed ideas, workarounds without a clear owner, observations that don't fit existing taxonomies. Concrete observation types that recur should graduate from the notebook to their own typed inbox via `meta-self-improvement` decision; spotting graduation candidates is part of the curator's job.

## Signal types

| Signal type           | Definition                                                                 |
|-----------------------|----------------------------------------------------------------------------|
| promotion-candidate   | Entry has matured enough to promote to canon (PoR or owned-context decision). |
| retirement-candidate  | Underlying problem is solved, obsolete, or duplicated.                       |
| still-debt            | Entry remains unresolved but should stay.                                    |

## Evidence rules

- Notebook entries are debt, not authority. Never cite a notebook entry as canon.
- Promotion requires evidence convergence; retirement requires explicit superseding signal.
- Aging without resolution is acceptable; flagging without curation is not.

## Action selection

| Action            | When                                                                                          |
|-------------------|-----------------------------------------------------------------------------------------------|
| drop              | Duplicate / obsolete / out-of-scope. Retire (delete) the entry.                               |
| observe           | Entry remains valid debt; refresh metadata, no move.                                          |
| promote-to-canon  | Promote: file the matching owned-context decision.                                            |
| file-decision     | Operator should decide now.                                                                   |
| capability-gap    | Promotion blocked by missing capability; file capability-gap and leave the entry.             |

## Owned schemas

None — notebook curation does not introduce destination front-matter shapes; the destination depends on which canon surface the entry is being promoted to (governed by the consuming taxonomy or PoR).

## Pending method skills

`notebook-promotion`, `notebook-retirement`, `notebook-aging-pass` are referenced as canonical method names; until they ship, the curator applies inline judgment.
