# Browser rehabilitation operator feedback

This active ledger contains unresolved material directives and the latest
execution-policy changes. The complete verbatim ledger before this compaction
is preserved at
[`operator-feedback-history-2026-09-28.md.gz`](evidence/rehabilitation/operator-feedback-history-2026-09-28.md.gz)
with its adjacent manifest. Retrieve it with `gzip -cd <path>` only when a
current decision needs exact historical wording. Earlier BAS-FB-001–042
history remains in `operator-feedback-history-2026-09-26.md.gz`.

New material feedback is quoted verbatim once before acting. Status questions,
acknowledgements and semantic repeats belong in the current checkpoint, not
this ledger.

## Open directives

### BAS-FB-051 — smaller setup model; financial distress

> Gotcha. I'm going to switch you to a smaller model as well, since I can't afford this. Really your inefficiency is causing me financial distress. This is honestly ridiculous. You need to lock in and get this done

Status: open. Keep delivery on cheap Luna-medium agents and make supervision
infrequent, measured, and Sol-only. Do not spend premium review budget on
repeated setup. The end-to-end pilot and runtime adoption evidence remain open.

### BAS-FB-050 — supervisor cost and efficiency

> Status update? You've used another 15% of my weekly usage in the past few hours. You need to hurry up and get this done efficiently, and make sure the supervisor and orchestrators are also really efficient. That includes limiting how much the supervisor runs, since it's at least 10x more expensive to run than the orchestrator

Status: open. Batch review around a coherent candidate, prohibit unchanged-status
wakes, retain findings across compactions, and record actual token/charge usage
when available. Wall-clock pacing is a signal, not an automatic kill. Adoption
and a successful pilot remain unverified.

### BAS-FB-047 — efficiency and simplification are acceptance requirements

> Please make notes/logs that you have been running for about 8 hours and have blown through a quarter of my weekly usage. This is partly because you're Astra, which is an expensive codex model. So it's really important that the agents doing the main work are gpt-6-luna-medium, the supervisor is smarter and runs much less frequently, and the process as a whole is efficient and the agents don't have to spend a bunch of time repeating work, validation, or investigations (likely all still not achieved despite you working for so long already). Make sure you're focusing on efficiency as well as the agents actually progressing properly. And remember that the purpose of the supervisor is to make sure the orchestrators are efficient and progressing, the purpose of *you* is to make sure this process as a whole is set up properly and working correctly, and that you yourself are not progressing quickly or efficiently. Also keep in mind that agents always tend to do more work than needed and struggle to *simplify*. That means you too but especially the lower powered agents need more emphasis and push to make sure things are working clean and maintainably. Without the push for simplification, they will always add more and more code and that quickly becomes unsustainable and breaks.

Status: open. This is an operator-reported cost signal, not independently
measured billing. The current workflow uses GPT-6 Luna delivery, infrequent
GPT-6 Sol review, hard worker call/turn/time/token ceilings, and explicit
simplification gates. Runtime adoption and a successful pilot are still open.

### BAS-FB-045 — consolidate existing teams and supervision

> It seems we are building multiple systems here that need to converge. There's the goal skill and related docs that need updating to use epochs properly (possibly), and the while supervision thing. We have a setup with prompt-manager delivery teams where a team can orchestrate and another supervises. We never got that fully working, but that seems like what would need updating to get this all working properly. Do you agree? Like instead of building a new orchestration and supervision setup, we streamline and fix what we alreay have, and make sure the bas stuff becomes a team? And maybe we allow teams to store data in scenarios as well so we don't have to move stuff?

> To be clear, the ideal solution must be CLEANER and more MAINTAINABLE than everything we have now. We have tried getting this working in the past and have failed. I don't just mean the current goal, but the whole supervision thing. So you need to make sure we're actually fixing and maturing instead of tacking on new things and making it messier. And we need to make sure the actual orchestration is done with CHEAP luna medium agents, and supervision is INFREQUENT with sol agents. Note that the models recently changed, so what agent-manager has for the models is likely incorrect. I believe they are all gpt 6 tag variants now.

Status: open. Existing Prompt Manager teams, Agent Manager, Workspace Sandbox,
Test Genie and Program Runtime remain the owners. No duplicate scheduler,
controller, ledger or Plan Manager plan is allowed for this continuous effort.
Scenario-local state uses existing `repo-root` references; a new storage class
is not justified by current evidence.

### BAS-FB-036 — test-code quality

The operator requested professional test infrastructure with less duplication,
drift and code volume. Status: open. Treat test debt inside an ownership-boundary
epoch; do not create score-neutral helper extraction.

### BAS-FB-020/021/022/023/024/025/026/027/028/029/030/031/032/033/037/038 — cadence

The repeated directive is to produce meaningful product progress faster, use
focused checks, stop repeating broad evidence cycles, and make qualification
serve delivery. Status: open. The candidate-epoch protocol batches related work
and requires changed/verified/remaining/unverified checkpoint evidence.

## Resolved or accepted directives

The following entries remain verbatim in the 2026-09-28 archive and are
summarized here to keep the active packet bounded:

| IDs | Disposition |
| --- | --- |
| BAS-FB-054 | GPT-6 Luna/Sol are available in the owner-validated catalog; policy corrected and validated. |
| BAS-FB-053 | Plan Manager is optional and phase-specific; not required for this continuous effort. |
| BAS-FB-052 | Supervisor pacing uses owner-admitted token/charge budgets and no-overlap limits, not a fixed wall-clock cap. |
| BAS-FB-049 | Use one evidence-backed pace intervention; do not repeatedly nudge unchanged work. |
| BAS-FB-048 | GPT-6 Luna remains primary; older 5.6 is only an authorized fallback, not a downgrade. |
| BAS-FB-046 | Scoped sandbox repair and recovery were applied; BAS stays disabled until the pilot gate passes. |
| BAS-FB-039 | Epochs span meaningful vertical slices and compactions; compaction is only a checkpoint. |
| BAS-FB-040/041/042/043/044 | Bounded packet, evidence archive, durable goal protocol and historical-control dispositions are implemented. |

Retrieve the archive only for a decision that depends on exact wording. Do not
reopen a settled choice without new evidence.
