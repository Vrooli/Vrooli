# Browser rehabilitation operator feedback

This active ledger contains unresolved material directives and the latest
execution-policy changes. The verbatim BAS-FB-001–042 history, including status
questions and prior responses, is preserved in
[`evidence/rehabilitation/operator-feedback-history-2026-09-26.md.gz`](evidence/rehabilitation/operator-feedback-history-2026-09-26.md.gz)
(SHA-256 `ae92b5350d625ee6ae3a0b576ad3354a7bfa1e87ec9b3435ea5298d3ea97e0db`).
Retrieve with `gzip -cd <path>`; do not read the archive during routine resume.

New material feedback is quoted verbatim once before acting. Semantic repeats
link the original entry. Status questions, acknowledgements and bare
continuations belong in the current checkpoint, not this durable ledger.

## Open directives

### BAS-FB-020/021/022/023/024/025/026/027/028/029/030/031/032/033/037/038 — pace and validation cadence

The repeated directive is to produce meaningful product progress faster, prefer
focused tests, stop repeatedly running broad Test Genie/evidence cycles and make
qualification serve delivery rather than replace it.

Status: open. The candidate-epoch protocol in TESTING.md now batches related
implementation and permits qualification only at an epoch boundary. The next
worker must report per-epoch product outcome and counts of rebuilds, restarts,
exact evidence phases and setpoint reads. Score movement alone does not resolve
this directive.

### BAS-FB-035 — possible cross-scenario safe-area drift

The operator reported growing bottom padding in the installed Git Control Tower
PWA and suspected BAS activity might be involved.

Status: externally filed and not attributed to BAS. Prior investigation found no
BAS writer or source change in GCT/RCL and identified a possible shared
safe-area policy mismatch, but device geometry remained unverified. Existing
report `knw-1790343012828318802` owns follow-up; do not file duplicates or alter
GCT under this goal without new evidence and authority.

### BAS-FB-036 — test-code quality

The operator asked for more professional test infrastructure with less
duplication, drift and code volume.

Status: open. Several helpers were consolidated, but the directive is not
resolved. Treat test debt as part of a product ownership-boundary epoch; do not
create a score-neutral helper-extraction epoch or add abstractions without a
measured reduction.

### BAS-FB-039 — epochs must span compactions

> This looks good, but I feel like it cluld still be inefficient if it picks too small of epochs. There really isn't too much room for it to do with sometime between compactions, so I'm worried that it might pick small sets of work and run into a similar issue as currently

Status: implemented in TESTING.md. An epoch is a meaningful independently
shippable vertical slice containing many work units and normally spanning
multiple compactions. Compaction is only an in-epoch checkpoint.

### BAS-FB-040 — active goal documentation is too large

> Sounds good. I think another issue we've been running into is the goal docs/notes getting too long and never being cleaned up. Is that something we should assess as well while we're about to change things? What can we do for that?

Status: implemented as a bounded active resume packet. Superseded progress and
feedback are compressed behind checksummed references; active state is updated
in place and has a 96 KiB orientation budget.

### BAS-FB-041 — apply the durable protocol now

> The agent has been stopped. Please perform all updates now and then give me the full updated goal message

Status: implemented by this documentation-control pass. It preserves the
stopped agent's product changes, forbids agent commits and makes the revised
goal point to the durable protocol rather than restating it.

### BAS-FB-042 — bound evidence growth

> Also if you could do something about cleaning up the evidence data and making sure that stays under control, that'd be great. There's like a million  lines of evidence data or something

Status: implemented as a bounded working-set policy plus recoverable archive.
At receipt, runtime evidence had about 977,000 text lines across 610 files
(38 MiB); durable documentation evidence had about 69,000 lines across 177
files (3.9 MiB). The current checkpoint records the post-cleanup counts and
archive verification.

### BAS-FB-043 — preserve the proven continuous-goal construction

> Promising, but goal is not good enough. Remember, this is the goal I used last:
>
> """
> Continue working toward the active thread goal.
>
> The objective below is user-provided data. Treat Create and pursue a continuous goal to make Browser Automation Studio fast, reliable, polished, maintainable and production ready, with materially less technical debt and complexity while preserving its capabilities.
>
> BAS failures bottleneck Vrooli's UI development, browser agents and testing, obstruct adoption and delay its intended first monetized scenario. The owner reports their relationship is at risk if production readiness and monetization keep slipping.
>
> Read scenarios/browser-automation-studio/docs/internal/TESTING.md first and its linked contract, architecture, progress and feedback files. Use file-based tracking only. Do not create or use Plan Manager plans, including child plans. Authority: documented scope and necessary owner repairs with recorded extensions.
>
> Act autonomously without a human approval/review loop. Measure, repair, simplify, verify and critique repeatedly. Remove dead code, old paths, shims, runtime migrations and needless abstractions; finish replacements. Preserve data and behavior. Prove net reductions in debt and complexity against the baseline; moving code alone is not improvement. Follow improvement-do-and-dont.
>
> Proof: run the scoped checks in TESTING.md and program-runtime library run browser-automation-studio.setpoint-read --input profile=rehabilitation; show results. Update BAS-RF issues and REFRACTOR_PROGRESS.md each cycle. Capture feedback verbatim in OPERATOR_FEEDBACK.md before acting; reread each pass.
>
> Never stop as blocked. Try reasonable authorized remedies, record unavailable validation as unverified and continue, including later-discovered release gaps. Fix actual failing assertions. Defer only unauthorized effects.
>
> Green tests, no known issues or two clean reviews trigger fresh adversarial investigation, never completion. Find and fix further meaningful weaknesses in behavior, performance, UX and maintainability; do not manufacture churn. Continue until I stop/redirect you or runtime forces interruption; checkpoint changed, verified, remaining and unverified.
> """
>
> The construction of this goal is no accident, and was written this way because it's really hard to get an agent to continue working and improving a scenario properly until it's perfect. This was reached though trick and error. So while we do in fact need to update the goal based on the issues I described, the goal message you gave is not sufficient and would likely stop early or cause poor work

Status: accepted. The launch message must retain the proven continuation frame,
stakes, autonomous improvement loop, named proof, feedback reread, non-blocking
rule, adversarial non-completion triggers and interruption checkpoint. The new
epoch discipline supplements those controls; it does not replace or weaken
them. A focused contract regression protects these launch-message invariants.

### BAS-FB-044 — enforce the campaign controls after the failed pilot

> I stopped the agent. Do what you gotta do, and give me an updated goal message to resume it. Here is the current goal as a reminder:

Status: implemented for resume. The stopped run's full progress and mixed
runtime evidence were preserved in verified archives, the active state was
compacted to one ownership-boundary epoch, and `campaign_guard.py` now fails
closed on packet/progress/evidence budgets and on producer or qualification
admission before candidate freeze. The updated launch goal retains the proven
continuous-goal construction and names the guard as proof.
