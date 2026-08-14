# Asset update flow

Use this flow when changing a React Component Library asset that already has
adopters.

1. Make the source change in a new version directory. Do not edit a released
   version in place.
2. Run `react-component-library components index --json`. The indexer derives
   `requiredTokens[]` and dynamic token families from the version source. A
   hash change to an existing released version is an integrity failure.
3. Run the catalog vocabulary and ramp-completeness gates, then run the
   version's component tests and the full scenario test suite.
4. For each target, run `adoptions preflight <component-id> <scenario>`. If
   the token verdict is blocking, run `adoptions tokens-sync <scenario>` and
   review collisions before retrying preflight.
5. Run `adoptions refresh` and classify the results. Clean, behind copies may
   be reconverged; modified copies require a human decision. A
   `source_drifted` result means the release record must be repaired or a new
   version must be published before adoption work proceeds.
6. Reapply clean adopters with the exact version and required confirmations.
   Reapply preserves opted-in suggested dependencies and removes orphaned
   files that left the new closure, unless another live adoption owns them.
7. For related assets, use the batch apply surface so shared dependencies and
   target collisions are evaluated once.
8. Record the evidence in the plan/work record: index result, gate results,
   preflight results, refresh/reapply outcomes, and any intentional override.
