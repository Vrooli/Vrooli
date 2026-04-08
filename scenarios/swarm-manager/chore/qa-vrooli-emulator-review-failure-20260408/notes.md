## Problem
GCT review-run failed for vrooli-emulator because the scenario directory is missing from the repo. This blocks QA from assessing readiness.

## Top Violations
- scenarios/vrooli-emulator/**: directory missing (repo check)
- git-control-tower review-run vrooli-emulator --json: status failed (job 57748813-411b-4e1c-9375-2edf5b1dd14f)

## Impact
No readiness assessment is possible, and backlog items depending on vrooli-emulator have no QA gate to clear. This leaves scope planning and dependency wiring unverified.

## Reproduction
git-control-tower review-run vrooli-emulator --json
ls scenarios/vrooli-emulator

## Success Criteria
- Scenario directory exists at scenarios/vrooli-emulator/** or is intentionally deprecated and removed from review queue
- git-control-tower review-run returns a readiness summary

## Proposed Action
1. Confirm whether vrooli-emulator has moved or been renamed.
2. Restore the scenario directory or update review-queue metadata to remove the entry.
3. Re-run GCT review to validate readiness once resolved.
