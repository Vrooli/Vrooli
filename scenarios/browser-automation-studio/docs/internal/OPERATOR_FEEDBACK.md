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
