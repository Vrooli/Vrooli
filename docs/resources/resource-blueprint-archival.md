# Blueprint-Only Resource Archival

Blueprint-only archival is the cleanup workflow for resources whose old `resources/<name>/` implementation should leave the repo, while the capability itself remains preserved as a blueprint candidate.

This is intentionally different from deprecation:

- **Blueprint-archived** means the old implementation is removed from the active repo surface, but the capability remains a live candidate through `.vrooli/resource-blueprints/<name>.json`
- **Deprecated** means the resource has left the active surface and is tracked as deprecated rather than as a future blueprint-backed candidate

## What Archive-To-Blueprint Does

- requires a matching blueprint record to exist first
- archives the current implementation under `~/.vrooli/archive/resources/`
- removes the active `resources/<name>/` directory
- removes transitional registry/config state for that resource
- records metadata in `.vrooli/blueprint-archived-resources.json`
- keeps restore explicit and quarantined rather than silently making the resource active again

## Safety Gates

`vrooli resource archive-to-blueprint <name>` will refuse to run when:

- the resource has no blueprint record
- the resource is still enabled or required in `.vrooli/service.json`
- any scenario manifest still references the resource
- the resource has already been deprecated

Those guards are there so blueprint-only archival does not accidentally remove an active dependency.

## Current CLI Surface

```bash
vrooli resource archive-to-blueprint <name>
vrooli resource list-blueprint-archived
vrooli resource restore-blueprint <name>
vrooli resource archive gc-blueprints
```

## Restore Semantics

Restores are intentionally quarantined.

- `vrooli resource restore-blueprint <name>` restores archived content into `.vrooli/restored-resources/<name>/`
- restored resources do not re-enter `vrooli resource list`
- promotion back into the active supported set remains an explicit later step

## Retention Policy

- default retention: `90` days
- expired blueprint archives can be removed with `vrooli resource archive gc-blueprints`
- metadata remains in `.vrooli/blueprint-archived-resources.json` for traceability after purge

## Recommended Use

Use blueprint-only archival when:

- the repo implementation is stale or speculative
- the capability knowledge is still worth preserving
- a future implementation should come from `blueprint -> template -> implementation`

Use deprecation instead when the capability should leave the active surface without remaining part of the blueprint-backed future candidate set.
