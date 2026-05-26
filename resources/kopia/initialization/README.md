# Kopia Initialization

This is the `initialization` well-known path for the kopia resource (declared in
`.vrooli/repo-contract.json` → `resource.well_known_paths.initialization`).

The kopia resource needs **no build-time initialization assets**: the binary is
provisioned by the manifest's `install.platforms` block, repositories are
created on demand via `resource-kopia repo create`, and all secrets are sourced
at runtime from the vault resource. There are therefore no seed files, schema
migrations, or bootstrap fixtures to ship here.

Runtime state (the repository registry, per-repo kopia config files, and content
caches) is created lazily under host-scoped storage-class roots, never inside
the repo tree. See [`../docs/OPERATIONS.md`](../docs/OPERATIONS.md).

This file exists to keep the well-known path present and self-documenting.
