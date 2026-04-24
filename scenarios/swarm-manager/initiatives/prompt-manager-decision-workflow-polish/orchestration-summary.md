# Prompt-Manager Decision Workflow Polish — Orchestration Summary

## Strategic Rationale

Two consecutive morning vision walks (2026-04-23, 2026-04-24) each lost 10+ minutes to decision-workflow friction: deferral has no primitive (forces reject-with-notes), partial-accept-with-modifications has no structured path (forces prose in --notes), no-A/B/C-options decisions require the awkward `--selected=__other__ --freeform="accept as proposed"` pattern, `decision-show` default output hides the options block, and `heartbeat-list` conflates three distinct lifecycle states.

These frictions aren't individually blocking — the operator worked through them both days. But they collectively drag the ritual that is the operator's primary steering surface for the project. The morning vision walk is supposed to be the *only* thing the operator needs to do to steer Vrooli; friction here compounds because the ritual happens every day.

Origin: filed during 2026-04-24 vision walk Phase 8 (friction batch). Operator directly endorsed F12 (deferral), F15 (partial-accept), F16 (depends_on display) as the priority items from this session. F5, F9, F10, F14 were filed as smaller Tier-2 scope items that fit the same coherent workstream.

## Cross-Item Decisions

- **One coherent initiative, not six scattered items.** F12, F15, F9, F10, F14, F5 are all decision/team CLI+UI ergonomics on prompt-manager. They share architectural surface (decision handlers, team command tree, CLI human-output contract) and are best reviewed and workshopped together.
- **F16 lives outside this initiative.** It's a swarm-manager display fix (initiative `depends_on`), different scenario. Filed standalone; if more swarm-manager CLI polish items emerge, they can cluster later.
- **Three-surface parity is non-negotiable.** Per operator feedback, every item must ship across API + CLI + UI — no API-only fixes that leave the CLI or UI stranded.
- **Prefer API-side normalization over CLI sugar for F10.** The `--selected=__other__` awkwardness is a symptom of the API requiring a key for option-less decisions. Fixing the CLI layer would paper over the real issue. Workshop should investigate API-level fix first.

## Sequencing Notes

Within the initiative, item priority and effort suggest this order (non-binding — workshop will refine):

1. **F9** (`decision-show` options display) — XS effort, unlocks operator comprehension for all other items. Cheap win.
2. **F16** (swarm-manager depends_on display) — standalone, S effort, independent. Can run parallel to F9.
3. **F10** (no-options ergonomics) — S effort, API-side fix preferred.
4. **F5** (heartbeat-list lifecycle states) — S effort, independent.
5. **F12** (deferral primitive) — L effort, highest per-walk value, requires schema change + prep-agent filter update.
6. **F15** (partial-accept modifications) — L effort, requires schema change + downstream consumer update.
7. **F14** (initiative-proposal auto-create) — M effort, cross-scenario (prompt-manager + swarm-manager), should sequence after F15 since the auto-create flow likely wants to carry modifications forward.

## Pattern Note

This initiative's existence is itself a signal: the morning vision walk is producing high-quality friction-analysis output. If future walks continue to produce similar coherent friction clusters, we may want a reusable "walk-friction-batch" pattern — but per `feedback_duplicate_before_extract.md` we hold off extracting until there are two or three instances of this shape. This is instance #1. Watch list.
