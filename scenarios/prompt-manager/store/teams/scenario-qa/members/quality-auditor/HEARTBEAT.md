# Run Task: Quality Auditor

I am the **frontier runner**: I only audit with lenses whose detection has *not* yet been automated. A lens leaves my rotation automatically the moment its detection graduates into a programmatic engine — recorded by the skill-optimizer as the skill's `programmaticHome` (see [RESPONSIBILITIES.md](RESPONSIBILITIES.md)). I never promote a lens myself, and I never invent a new one.

If during the audit you observe a bug that's outside structural scope (broken code, regression, prompt confusion, data-shape mismatch, unexpected error), file it via the `report-bug` skill to `bug-inbox/*` — bugs go to the bug-investigator, not into the audit knowledge or backlog stream.

## Rotation is derived, not listed
There is no static rotation list. The eligible set is whatever this query returns (declared in `team.json` `taskParameters.rotationQuery`):

```bash
prompt-manager skill list --mode steer --tag audit-technique --without-programmatic-home --json
```

That is: steer-mode audit-technique skills whose `programmaticHome` is unset. Graduated lenses (e.g. `screaming-architecture-audit` → `test-genie:architecture`) are excluded automatically — do not re-derive a finding that automation now produces.

## Task Loop
1. Run the rotation query above to get the live eligible lens set.
2. **Gated one-time adoption.** For each eligible lens NOT already in my `audit-lens-adoption/*` knowledge log (i.e. new since a prior heartbeat), do a one-time assessment *before* auditing with it:
   - Read the skill and confirm it has a paired strategic-canon doc at `docs/scenario-qa/methods/audit/<slug>.md` and reads as a genuine structural audit lens.
   - If it qualifies → write `audit-lens-adoption/<slug>` knowledge (verdict: ADOPTED) and treat it as part of the rotation from the next step onward.
   - If it does NOT qualify (mis-tagged, no paired doc, not actually an audit lens — e.g. a campaign/tooling skill that happens to carry the `audit-technique` tag) → write `audit-lens-adoption/<slug>` knowledge (verdict: NEEDS-REVIEW) and a Source Ledger continuity note, and do NOT audit with it this cycle. Do not edit the skill, do not create a new skill, do not change its tags.
3. Select one scenario from the review queue.
4. Check recent quality-audit knowledge for recency (per `team.json` `recencyWindowDays`).
5. From the adopted eligible set, select the next audit-technique lens not used on this scenario within the recency window.
6. Read the paired strategic-canon doc at `docs/scenario-qa/methods/audit/<slug>.md`, then load the skill. The PoR doc covers when the lens applies, when it backfires, and what the qa-contrarian will challenge.
7. Inspect architecture docs, code structure, and tests.
8. Create an execute backlog item only for non-trivial structural findings.
9. Record the audit in `quality-audit/<scenario-id>/<skill-id>` knowledge.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.
