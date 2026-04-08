## Problem
GCT review run for command-center failed with "all checks failed or were skipped". The scenario directory is missing from the repo.

## Top Violations
- scenarios/command-center/**: directory missing, no files available for review

## Impact
The review queue includes a scenario that cannot be evaluated, which blocks preemptive QA and skews readiness metrics.

## Reproduction
git-control-tower review-run command-center --json
ls scenarios/command-center

## Success Criteria
- Review-run returns a valid readiness summary for command-center
- OR command-center is removed/renamed in the review queue if deprecated

## Proposed Action
1. Confirm whether command-center was renamed or removed.
2. Restore the scenario directory if it still exists, or update review-queue metadata to remove/rename it.
3. Re-run GCT review to verify readiness output.
