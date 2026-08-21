# Decision: the `component-tests` timeout is raised to 1200s

**Date:** 2026-08-20 · **Status:** decided, pending re-measurement

## The measurement was censored by its own ceiling

| Measure | Value |
| --- | --- |
| Samples | 61 |
| p50 wall clock | 305 s |
| p90 wall clock | **600.003 s** |
| Timeout | 600 s |
| Samples at or over the ceiling | **11 of 61 (18%)** |

A p90 that lands three milliseconds past the timeout is not a measurement of how
long the phase takes. It is a measurement of the timeout. The real upper tail is
unknown, because every run that would have revealed it was killed at the
ceiling.

## Two consequences, both bad

**Runs are being killed.** 18% of samples hit the ceiling. Those appear as phase
failures whose cause is the deadline, not the code under test.

**The phase is excluded from every batch.** Test Genie's deadline guard keeps a
phase out of a batch when `p90 × contentionAllowance >= timeout`. With
`contentionAllowance = 1.5`, that reads `600 × 1.5 = 900 >= 600` — always true.
So the most expensive component phase runs alone in every suite, and the guard
that was meant to protect a phase with no headroom is instead permanently
excluding one whose headroom was never measured.

## The decision

Raise the timeout from 600 s to **1200 s**.

This is a **measurement** decision, not a performance one. The purpose is to
un-censor the distribution so the true p90 becomes observable. Nothing here
claims the phase should take 20 minutes.

Why 1200 s specifically: it is roughly 4× the measured p50 of 305 s, which is
wide enough that the tail should fall inside it, while still bounding a runaway
at a value an operator would notice. If the true p90 lands near 700 s, the
deadline guard reads `700 × 1.5 = 1050 < 1200` and the phase becomes batchable
again — restoring the guard's intent rather than disabling it.

## What must happen next

1. Re-measure after roughly 20 runs under the new ceiling. If the p90 again
   lands at the ceiling, the phase has an unbounded case and the answer is to
   reduce what it does, not to raise the ceiling a second time.
2. If the uncensored p90 is comfortably below 800 s, lower the timeout to about
   1.5× that value so the guard stays meaningful.

Raising a timeout twice without diagnosis converts a visible failure into a
slower invisible one. This decision is explicitly bounded to one round.

## Related

The 28 failures among these 61 samples are now reusable: a failure under an
unchanged cache identity is served from the phase cache rather than re-derived,
so the repeat cost of this phase falls without waiting on the timeout question.
