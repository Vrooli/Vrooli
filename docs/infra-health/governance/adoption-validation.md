# Infra Health Adoption Validation

## Validation Commands

Use these commands after changing this plan of record:

```bash
prompt-manager graph operating-model validate --team infra-health --id infra-health-operating-model
prompt-manager graph operating-model diff --team infra-health --id infra-health-operating-model
prompt-manager graph operating-model coverage --team infra-health --id infra-health-operating-model
```

For a quick local tree check, also run:

```bash
find docs/infra-health -maxdepth 3 -type f | sort
prompt-manager graph topics
```

To confirm the migration onto the instrument has not regressed — no member
prompt or PoR path may route back to a retired document:

```bash
grep -rn "RELIABILITY_TARGETS\|INSTRUMENTATION_ROADMAP\|CROSS_PLATFORM_LEDGER" \
  scenarios/prompt-manager/store/teams/infra-health/members \
  scenarios/prompt-manager/store/teams/infra-health/shared \
  scenarios/prompt-manager/store/teams/infra-health/team.json \
  scenarios/prompt-manager/store/teams/infra-health/graph-presentation.json \
  docs/infra-health/operating/ | grep -v "/logs/"
infrastructure-manager coverage validate --json
```

`coverage validate` must report `ok: true`. The grep must return nothing except
the known-outstanding hits recorded in the 2026-08-20 changelog entry —
`team.json`'s `planOfRecord` path and the two `graph-presentation.json` nodes,
all three still routing agents to the retired `CROSS_PLATFORM_LEDGER.md`, plus
the matching `@node` declarations in `operating/OPERATING_MODEL.md:103-104` that
the presentation file maps by id. Those move together or not at all. One further
hit is permitted and is not routing: `operating/OPERATING_MODEL.md:175` names the
ledger only to say it is retired and takes no new entries. A hit on anything else
is a regression.
All three files still exist and each carries a retirement banner, so the banner,
the retirement table in `README.md`, the hubs in `strategy/` and `evidence/`,
the manifest retirement records and the changelog are permitted references. No
entry may be added to any of them, and nothing outside this folder may route a
reader to one as current truth.

The computed cross-platform surface that replaced the ledger must itself read:

```bash
vrooli capability ledger --json
vrooli capability fleet blocked --json
```

## Expected Clean State

> **Known failing since the 2026-08-20 migration.** `prompt-manager graph
> operating-model validate --team infra-health` currently exits non-zero with two
> errors, both needing changes outside this folder: `por_manifest_invalid`
> (`json: unknown field "status"` — the retirement records and the `instrument`
> block are not in the `PlanOfRecordManifest` schema, which decodes with
> `DisallowUnknownFields`) and `graph_unknown_node_kind` for the `instrument:`
> node at `operating/OPERATING_MODEL.md:100`. Treat a *different* error as a new
> regression; treat these two as outstanding until the schema admits them.

- `README.md` is the only top-level prose canon file.
- `manifest.json` declares every durable module and names the instrument scenario.
- `operating/OPERATING_MODEL.md` exists and is registered in the infra-health team plan-of-record.
- Reliability targets live in the **instrument**, not this folder: `scenarios/infrastructure-manager/setpoint/reliability-setpoint.json` plus each owner's `docs/spaces/<projection>-space.md`. A target reintroduced under `strategy/` is a regression.
- `strategy/RELIABILITY_TARGETS.md` and `evidence/INSTRUMENTATION_ROADMAP.md` are retired: present, banner-first, `required: false`, and referenced by no member prompt. `manifest.json` carries their retirement records.
- `evidence/CROSS_PLATFORM_LEDGER.md` is retired: present, banner-first, `required: false`, and holding no entries. A row appended to it is a regression.
- Instrumentation gaps are computed (`coverage open-loop`); cross-platform state is computed (`vrooli capability ledger`). Neither is a list under `evidence/`.
- Editing authority, validation notes, and migration history live under `governance/`.

## Enforcement Scope

Validation treats [`../manifest.json`](../manifest.json) as the only structural contract for this plan of record. New durable canon must be registered in the manifest, placed under the most specific standard module, and edited only through accepted infra-health decisions.
