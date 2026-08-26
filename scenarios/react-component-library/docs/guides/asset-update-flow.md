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
3. Formatting is owned by version creation: ingests and draft-to-release copies
   canonicalize `experience-contract.json`, and use the scenario UI's Prettier
   configuration for `.ts`/`.tsx` artifacts when the formatter is installed.
   The Files editor's JSON pretty toggle is presentation-only; saving/promotion
   remains the consistency boundary.
4. Publish the draft only after validation with
   `react-component-library components version-publish <component-id> --version <draft>`.
   `CheckComponentVersion` validates the source, dependencies, story coverage,
   and any attached experience contract (including state-to-story references).
   Then run `react-component-library components index --json`. The indexer derives
   external token requirements and dynamic token families from the version source. A
   hash change to an existing released version is an integrity failure.
5. Run the catalog vocabulary and ramp-completeness gates, then run the
   version's component tests and the full scenario test suite.
6. For each target, run `adoptions preflight <component-id> <scenario>`. If
   the token verdict is blocking, run `adoptions tokens-sync <scenario>` and
   review collisions before retrying preflight.
7. Run `adoptions refresh` and classify the results. Clean, behind copies may
   be reconverged; modified copies require a human decision. A
   `source_drifted` result means the release record must be repaired or a new
   version must be published before adoption work proceeds.
8. Reapply clean adopters with the exact version and required confirmations.
   Reapply preserves opted-in suggested dependencies and removes orphaned
   files that left the new closure, unless another live adoption owns them.
   Cleanup also follows relative imports across released asset versions and
   adopted files, including stories. A protected cleanup item reports the
   importing asset/version, file, import specifier, scenario, and adoption id
   where applicable.
9. For related assets, use the batch apply surface so shared dependencies and
   target collisions are evaluated once.
10. Record the evidence in the plan/work record: index result, gate results,
   preflight results, refresh/reapply outcomes, and any intentional override.

## Importable adoption

Use a linked adoption when the released export surface is sufficient for the
target scenario. The governed link workflow installs the scenario's
`file:../../../packages/react-component-library` dependency, records
`mode=linked`, and writes the managed locale and selector obligations:

```bash
scenario-dependency-analyzer deps install node/@vrooli/react-component-library \
  --scenario <scenario> --surface ui \
  --version file:../../../packages/react-component-library --apply --json
react-component-library adoptions link <library-id> <scenario> \
  --version <version> --json
```

Linked records have no `adoption_files` rows. Imports use a stable versioned
subpath such as `@vrooli/react-component-library/components/Button/1.2.0`.
Run `react-component-library adoptions refresh --json` after linking so the
fork census and adopter gates reflect the live state.

If a scenario must change behavior outside the exported contract, use the
reason-bearing eject command. Eject changes the record to `mode=ejected` and
returns the adoption to the governed copy workflow; it never silently turns a
linked dependency into a private fork.

## Files editor library imports

The Files editor understands published imports such as
`@vrooli/react-component-library/Button/2.2.0`. Hovering an import resolves it
through the catalog and shows the asset id, pinned version, export kind, and
description. Go to Definition opens the version-pinned source in the same
Monaco viewer. Relative imports remain local to the current file and are
reported as unresolved; the editor does not pretend that a relative path is a
catalog dependency. An unresolved package import likewise produces a visible
diagnostic instead of an empty hover.

## Intentional forks and version retention

An adopter may declare a fork at apply time by supplying a reason and the
extension points that remain owned by the target scenario. The adoption record
stores `fork_status=declared-fork`; reconvergence reports the fork and never
overwrites it. A modified copy without that declaration is classified as
unintended drift until an operator reviews it. Mechanical token or layout
translation is reported separately and is not treated as a source fork.

Version cleanup is evidence-first. `versions plan-cleanup` produces the
candidate set and plan hash. Only candidates with no latest/draft role, direct
or mediated adoption, dependency pin, or source import may be retired. The
ledger row remains after retirement, and the retained release set is the only
set checked for released-source immutability. Never add a schema migration or
compatibility fallback to make cleanup pass: fresh schemas are declared next
to the code that interprets them.

## Worked bulk migration

For a migration touching several related assets, open one draft per asset and
keep the working set isolated from released directories:

```bash
react-component-library catalog draft open react-component-library:Button --bump patch --json
react-component-library catalog draft open react-component-library:Input --bump patch --json
react-component-library components refresh react-component-library:Button --version <button-draft>
react-component-library components refresh react-component-library:Input --version <input-draft>
react-component-library catalog draft promote react-component-library:Button --version <button-draft> --json
react-component-library catalog draft promote react-component-library:Input --version <input-draft> --json
```

Run the index, catalog gates, component tests, and adopter preflights after the
batch promotion. If review rejects the migration, use
`react-component-library catalog draft discard <component-id>`; never rewrite a
released directory to make the batch pass.
