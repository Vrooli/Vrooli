# Research method release

Controlled evaluation receipts are the only authority for selecting a research
method. A receipt records the candidate revision hash, report hash, and passing
status. Promotion checks all three against the immutable report before changing
the current selector.

Promotion is compare-and-swap: the caller supplies the revision currently
expected to be selected. A concurrent or stale caller receives a conflict and
cannot overwrite the newer release. Rollback uses the same guard and only
selects a revision retained in the release history.

Evaluation observations use `provenance=test` and are excluded from operational
learning cohorts. A passing controlled receipt supports software release
selection; it does not establish an organic operator benefit claim.
