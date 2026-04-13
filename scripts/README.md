# Scripts Directory

## Status

The project-level `manage.sh` path is gone. Repo-root orchestration now lives in the Go control plane:

- `cmd/vrooli`
- `cmd/vrooli-api`
- `internal/*`
- root `Makefile` targets that invoke the installed `vrooli` binary

If you are trying to set up or run the repo root, use:

```bash
make setup
make dev
make build
vrooli --help
```

## What Still Lives Here

The `scripts/` tree remains for shared shell helpers that are still consumed outside the project-level control plane.

- `scripts/resources/`
  Resource-side shell frameworks, templates, validators, and compatibility helpers.
- `scripts/scenarios/`
  Scenario scaffolds and scenario-specific tooling that has not moved into a native package yet.
- `scripts/lib/`
  Shared shell helpers still referenced by resources and scenarios. Host setup is no longer owned here; this tree is migration debt, not the canonical project orchestration path.

## Cleanup Boundary

Do not treat everything under `scripts/` as dead.

- Some files under `scripts/lib/` are still referenced by resource shells.
- UI package scripts should use the shared `packages/api-base/bin/vrooli-ui-run.js` launcher instead of `scripts/lib/ui-guard.sh`; a few non-UI legacy callers may still exist until their own cleanup pass lands.
- Deleted host-setup surfaces under the old `scripts/lib` bootstrap path must not be reintroduced.
- Project-level Bash entrypoints such as `scripts/manage.sh` are already deleted and should not be reintroduced.

When cleaning this directory, prefer:

1. Deleting root-only historical artifacts.
2. Moving shared survivors toward resource/scenario-owned locations.
3. Updating docs to describe the Go-native root flow accurately.
