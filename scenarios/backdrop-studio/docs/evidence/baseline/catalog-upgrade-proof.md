# Catalog upgrade proof

Evidence that an install created before this plan receives the current catalog
without losing operator-authored work — the defect recorded as A3.

## Starting state

`data/backdrop-studio.db.preaudit`, a real install from before 2026-08-12:
**4 styles, 5 surfaces**, on the pre-plan schema. `Seed()` guarded on
`if styles == 0`, so this install would never have received a fifth style. Its
four ids (`arcade-noir`, `field-guided`, `horizon-ink`, `terrain-riso`) do not
appear in the current catalog at all.

An operator-authored row was inserted before the upgrade, with
`origin = 'operator'`.

## Reproduce

```bash
vrooli scenario stop backdrop-studio
cp data/backdrop-studio.db.preaudit data/backdrop-studio.db
rm -f data/backdrop-studio.db-shm data/backdrop-studio.db-wal
vrooli scenario start backdrop-studio
backdrop-studio catalog list --json  | jq '.styles   | length'
backdrop-studio surfaces list --json | jq '.surfaces | length'
```

## Result

| Check | Before | After |
|---|---|---|
| Styles | 4 | 21 — 16 seeded, 4 legacy retained, 1 operator |
| Surfaces | 5 | 9 |
| `applied_seed_version` | (absent) | 1 |
| Operator row | present | present, unchanged (`version=3`, `origin=operator`) |

Legacy styles are retained rather than deleted: a released backdrop may
reference one, so no row the current seed no longer contains is ever removed.

## What this caught

The first upgrade attempt failed at

```
catalog seed failed: catalog: seed v1 style "cyanotype-arcade":
NOT NULL constraint failed: backdrop_styles.payload
```

while the entire unit suite passed. `payload` is a whole-style JSON copy that
old installs carry and fresh ones never had, so every migration test — building
its database from the *current* schema — was testing the one shape no real
upgrade starts from. The column is now dropped during migration, and
`TestSeedMigratesAPrePlanInstall` builds its fixture from the verbatim pre-plan
schema so the gap cannot reopen.
