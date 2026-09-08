# Command Center — Progress Log

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/command-center/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
The table retains selected dated milestones. Other milestones and their limitations
remain in the archive; consult it before resuming historical work.

Rolling log of material changes to the Command Center scenario. Append newest entries at the top.

| Date       | Author              | Δ Completeness | Description |
|------------|---------------------|----------------|-------------|
| 2026-09-07 | claude (Opus 5) | Lease contention fixed at the source | **Prep v7 drops the scenario-grouped fan-out added earlier the same day and goes back to a plain `gather`,** because the contention it worked around is now fixed in the layers that own it: `internal/scenarioruntime` opens the runtime registry with `_txlock=immediate` (its read-then-write transactions were upgrading a WAL snapshot mid-flight and raising SQLITE_BUSY_SNAPSHOT/517, which busy_timeout deliberately does not cover), `withRetryableTx` jitters its backoff, and `demand.SharedHolder` refcounts one lease per (consumer, scenario) so a program's concurrent bindings at one scenario take one lease instead of one each. |
| 2026-09-07 | claude (Opus 5) | Peer continuity + lease contention | **Prep v6 reads declared ledger topics instead of the retired handoff surface.** `portfolio_handoff`/`strategist_handoff` (`prompt-manager/team/handoff-latest`) are replaced by `portfolio_record`, `strategist_record` and a new `contrarian_record`, read from the `goal-portfolio-record`, `outcome-target-record` and `contrarian-scan` topics their producers actually publish; the `prompt-manager/team/handoff-latest` binding, the `previous_handoff` legacy checkpoint migration, the handoff `content_change` comparison, the `missing_evidence` error class and the `legacy` checkpoint state (also in `walk_records.go`) are gone. |
| 2026-09-06 | Codex | Walk v4 and friction repair | Corrected review-status and archived-completion selection; behavioral fixtures now exercise owner filters. Publish preserves unavailable evidence and accepts formatting without caller compaction. Full suite remains queued; programs/skill-set admission refused while that run owns the slot. Operator usefulness remains unmeasured; friction analysis is in PROBLEMS. |
| 2026-09-05 | Codex | Handoff change detection | Prep v3 fingerprints full handoff content before clipping excerpts and flags changed content for inspection. Timestamp-only updates remain unchanged; older truncated snapshots without fingerprints remain unknown. Test Genie `20260905-042040-2eb1130a` passes programs and skill-set; API unit command remains failed, with the broader cause unverified. |
| 2026-09-04 | Codex | Morning walk capability | Command Center now owns usage, prep, interactive walk and improve skills; director prep heartbeat delegates to the owned skill. |
