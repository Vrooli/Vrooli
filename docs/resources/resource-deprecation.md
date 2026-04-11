# Resource Deprecation

Phase 2 introduces a first-class deprecation lifecycle for project-level resources.

The goal is to remove stale resource code and metadata from the active repo surface without losing short-term recoverability.

## What Deprecation Does

- archives the resource's active repo state under `~/.vrooli/archive/resources/`
- removes the resource from active repo discovery
- records deprecation metadata in `.vrooli/deprecated-resources.json`
- keeps restore explicit and quarantined rather than silently making the resource active again

## Current CLI Surface

```bash
vrooli resource deprecate <name>
vrooli resource list-deprecated
vrooli resource restore <name>
vrooli resource archive gc
```

## Restore Semantics

Restores are intentionally quarantined.

- `vrooli resource restore <name>` restores archived content into `.vrooli/restored-resources/<name>/`
- restored resources do not re-enter `vrooli resource list`
- promotion back into the active supported set remains an explicit later step

This prevents stale code from reappearing in the supported surface by accident.

## Retention Policy

- default retention: `90` days
- expired archives can be removed with `vrooli resource archive gc`
- metadata remains in `.vrooli/deprecated-resources.json` for traceability after purge

## Phase 2 First Batch

The first deprecated batch matches the clearly stale Phase 0 `deprecate` set:

- `autogen-studio`
- `erpnext`
- `langchain`
- `musicgen`

Each now has:

- deprecation metadata
- an archive record
- a replacement blueprint with the same canonical name

## Validation Bundle

Phase 2 validation is:

- `vrooli resource deprecate autogen-studio`
- `vrooli resource list-deprecated`
- `vrooli resource restore autogen-studio`
- `vrooli resource archive gc`
- `go test ./internal/resources`
- focused `cmd/vrooli` resource command tests
