# Programs

## North Star

Every declared program has a readable contract, a valid source, and evidence that its governed behavior can be checked.

## The rungs and their gates

L0 means the target cannot be inspected. L1 means declarations are readable. L2 means the declaration and fixture checks are clean. Static checks run by default; fixture checks run when execution is requested.

## What each finding means

`programs.scenario_missing` means the requested scenario path is absent. `programs.fixture_unavailable` means execution evidence was requested but the program fixture directory was not available.

## The canonical fix

Restore the scenario declaration and its committed program fixtures, then rerun `vrooli scenario test <scenario> --phases programs`.

## How to verify

Run `test-genie provider-contract check programs program-runtime --json`, then run `vrooli scenario test program-runtime --phases programs` and inspect the advisory scorecard.
