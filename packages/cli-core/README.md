# CLI Core

Shared helpers for scenario CLIs: installer, stale-checker, HTTP client, and app/command scaffolding.

## Install a scenario CLI

Use the shared installer instead of per-scenario scripts:

```bash
# From the repo root
./packages/cli-core/install.sh scenarios/scenario-completeness-scoring/cli --name scenario-completeness-scoring
```

- Binaries install to `~/.vrooli/bin` by default; override with `--install-dir <path>`.
- To pin a published cli-core version instead of local sources, set `CLI_CORE_VERSION`, e.g.:
  `CLI_CORE_VERSION=v0.0.1 ./packages/cli-core/install.sh scenarios/scenario-completeness-scoring/cli --name scenario-completeness-scoring`.
- `APP_ROOT` can override repo root detection when the script is run from other directories.

Under the hood this runs `cmd/cli-installer`, which embeds build fingerprints for stale-check detection.
