# Preview composition migration

Use this procedure to migrate existing stories without changing the public
asset API.

## Classification

For each `story.json` and `story.tsx` pair, record the asset kind and hierarchy
rung, current frame, current harness export, fixtures, expectations,
interactions, and screenshot states. Then choose the smallest valid outcome:

| Outcome | Meaning |
|---|---|
| Direct | Subject is clear without additional composition |
| Shared harness | A documented family expresses the behavior with an injected subject |
| Local harness | The behavior is unique and cannot be expressed generically |
| Frame | Context materially changes interpretation |
| Fixture-backed | Deterministic external state is required |
| Exception | Migration is intentionally deferred with an owner and revisit condition |

Migrate representative assets first: one foundation, runtime hook, primitive,
component, navigation/pattern, overlay, and page-template asset. MorphingIcon,
Button, and FilterBar are the initial behavior and composition probes.

## Safe sequence

1. Capture the current story's rendered behavior, expectations, interactions,
   and screenshots.
2. Select a frame only after compatibility validation returns a supported
   region and fixture set.
3. Select a shared harness only after its family contract matches the behavior.
4. Keep the local harness if migration would remove meaningful behavior or
   accessibility evidence.
5. Format JSON and TSX, run focused validation, and compare screenshots.
6. Update the migration ledger and exception record.
7. Run adoption and dependency-closure checks to prove Preview-only artifacts
   remain excluded.

Never resolve a canonical story through “latest”. Pin the exact frame and
shared-harness versions. Publish a new story or asset version when composition
changes. Do not edit released asset bytes.

## Rollback

Rollback means restoring the prior story contract and local harness reference,
not deleting a released version. Keep the failed evidence and reason in the
migration ledger. A migration is complete only when the new story preserves
the old contract and has stronger visual, accessibility, and interaction
evidence.

## Complete inventory

The migration ledger is generated from the checked-in story contracts so it
cannot silently omit a newly added version. Run:

```bash
cd scenarios/react-component-library/ui
node scripts/preview-composition-inventory.mjs > /tmp/preview-composition-inventory.json
```

The output contains one entry for every `library/**/story.json`, including its
stable library/version identity, hierarchy, story IDs, composition roles,
current frame, fixture, shared harness, local harness presence, diagnostics,
and proposed disposition. It also contains one `storyRecords` row per story,
with expectation and interaction counts, raw-child detection, review-set
coverage, production-access checks, migration status, and an exception record
for every local harness that still needs behavior-equivalence review. The
current checked-in evidence snapshot is
`docs/evidence/preview-composition-inventory.json`; it covers every story
contract and story record found at capture time. Counts are time-dependent.
Review the inventory before each migration batch and retain the batch evidence;
do not copy derived counts into permanent prose.

Run a bounded, resumable review batch with stable story keys:

```bash
node scripts/preview-composition-inventory.mjs \
  --batch-size 25 \
  --batch-index 0 \
  --state /path/to/migration-state.json
```

The state file is read-only input with this shape:

```json
{ "completedStoryKeys": ["react-component-library:Button@2.2.0#async-save"] }
```

The report excludes completed keys, emits the next resumable key, and returns
only the selected batch. The workflow must record a key only after its before
manifest, focused test, capture, review disposition, and adoption check exist.
This makes interruption safe: rerunning the same state and batch does not
duplicate completed stories or silently skip unfinished ones.
