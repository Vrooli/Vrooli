# deployment-manager CLI

Go-native, cross-platform CLI built on `packages/cli-core` for automating deployment-manager APIs.

## Install

```bash
# From repo root
./packages/cli-core/install.sh scenarios/deployment-manager/cli --name deployment-manager
# Binary lands in ~/.vrooli/bin by default (adds build fingerprint + timestamp).
```

## Usage

```bash
deployment-manager --help
deployment-manager status
deployment-manager --json status                    # global JSON toggle
deployment-manager --format table profiles          # table output for humans
deployment-manager analyze picker-wheel --format json
deployment-manager fitness picker-wheel --tier desktop
deployment-manager profile create demo picker-wheel --tier 2
deployment-manager profile export demo --output demo-profile.json
deployment-manager profile diff demo
deployment-manager profile rollback demo --version 1
deployment-manager deploy demo --dry-run
deployment-manager logs demo --level error
deployment-manager secrets template demo --format env
```

### Quick recipes
- Bootstrap and validate: `deployment-manager profile create demo picker-wheel --tier desktop && deployment-manager validate demo`
- Export/import for AI agents: `deployment-manager profile export demo --output demo.json` then `deployment-manager profile import demo.json --name demo-copy`
- Swap and re-score: `deployment-manager swaps list picker-wheel && deployment-manager profile swap demo add postgres sqlite && deployment-manager fitness picker-wheel --tier desktop`
- Desktop deploy dry-run: `deployment-manager deploy-desktop --profile demo --dry-run --timeout 10m && deployment-manager estimate-cost demo`
- Debug a deploy: `deployment-manager deploy demo --dry-run && deployment-manager logs demo --level error`

Electron builds run with pnpm by default (falls back to npm if pnpm is unavailable). Use `--timeout` to extend long-running builds (default 10m).

## Structure
- `overview/` — status, analyze, fitness
- `profiles/` — profile CRUD, versions, swaps, secrets, diff/save/rollback
- `swaps/` — swap discovery and application
- `deployments/` — deploy, deploy-desktop, validate, cost-estimate, logs (packagers/package kept as compatibility stubs)
- `cmdutil/` — shared flag/output helpers

The CLI auto-discovers the API base when the scenario runs via `vrooli scenario start deployment-manager`. Override with `DEPLOYMENT_MANAGER_API_BASE` or `deployment-manager configure api_base <url>`. Config and token files live under `~/.deployment-manager/` by default.
