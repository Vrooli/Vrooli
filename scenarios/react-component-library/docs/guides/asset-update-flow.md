# Asset update flow

Use this flow when changing a React Component Library asset that already has
adopters.

1. Start a draft from the current release with
   `react-component-library components version-begin <component-id> --bump patch`
   (or provide an explicit draft version). Work in the generated `*-draft.*`
   files; drafts are mutable and are not visible as released catalog versions.
2. Run `react-component-library components refresh <component-id> --version <draft>`
   while iterating. This performs a scoped preflight from disk without requiring
   a repository-wide reindex. The detail page shows the indexed version, source
   path, and whether the source hash has drifted from the indexed release.
3. Publish the draft only after validation with
   `react-component-library components version-publish <component-id> --version <draft>`.
   Then run `react-component-library components index --json`. The indexer derives
   `requiredTokens[]` and dynamic token families from the version source. A
   hash change to an existing released version is an integrity failure.
4. Run the catalog vocabulary and ramp-completeness gates, then run the
   version's component tests and the full scenario test suite.
5. For each target, run `adoptions preflight <component-id> <scenario>`. If
   the token verdict is blocking, run `adoptions tokens-sync <scenario>` and
   review collisions before retrying preflight.
6. Run `adoptions refresh` and classify the results. Clean, behind copies may
   be reconverged; modified copies require a human decision. A
   `source_drifted` result means the release record must be repaired or a new
   version must be published before adoption work proceeds.
7. Reapply clean adopters with the exact version and required confirmations.
   Reapply preserves opted-in suggested dependencies and removes orphaned
   files that left the new closure, unless another live adoption owns them.
8. For related assets, use the batch apply surface so shared dependencies and
   target collisions are evaluated once.
9. Record the evidence in the plan/work record: index result, gate results,
   preflight results, refresh/reapply outcomes, and any intentional override.
