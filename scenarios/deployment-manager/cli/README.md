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
deployment-manager help
deployment-manager status
deployment-manager --json profiles list
deployment-manager analyze "picker-wheel"
deployment-manager fitness "picker-wheel" --tier desktop
deployment-manager profiles create "demo" "picker-wheel" --tier 2
deployment-manager deploy-desktop --profile "demo" --dry-run
deployment-manager releases dossier "<release-id>"
deployment-manager releases health "<release-id>"
```

### Quick recipes
- Bootstrap and prepare review: `deployment-manager profiles create "demo" "picker-wheel" --tier 2`, then use `readiness-reviews prepare` with the exact review identity.
- Swap and re-score: `deployment-manager swaps list "picker-wheel"`, review the result, then use `swaps apply` with the exact profile ID.
- Desktop artifact dry-run: `deployment-manager deploy-desktop --profile demo --dry-run --timeout 10m` (local preparation only)
- Readiness applicability: `deployment-manager readiness-reviews prepare "<scenario>" "<profile-id>" "<commit>" "<artifact-digest>" stable linux --fact commercial_release=true --fact paid_release=true`
- Governed release lifecycle: `deployment-manager releases start <profile-id> --commit <hash> --version <version> --readiness-review-key <key> --candidate-id <id> --destination-revision-id <id>`
- Inspect a governed release: `deployment-manager --json releases dossier "<release-id>" && deployment-manager releases health "<release-id>"`

Electron builds run with pnpm by default (falls back to npm if pnpm is unavailable). Use `--timeout` to extend long-running builds (default 10m).

## Structure
- `app.go` — standard `cli-core` bootstrap via `NewStandardScenarioApp(...)`
- `domains/` — canonical registration layer for CLI domains
- `overview/` — analyze and fitness domain logic
- `profiles/` — typed profile CRUD, versions, import/export, and updates
- `swaps/` — swap discovery and application
- `deployments/` — typed deployment calls plus retained local preparation compatibility
- `releases/` — candidate-bound governed release lifecycle and recovery
- `approvals/`, `signing/`, `validations/` — approval gates and explicit retirement guidance for owner-specific workflows
- `cmdutil/` — shared flag/output helpers

The CLI auto-discovers the API base when the scenario runs via `vrooli scenario start deployment-manager`. Override with `DEPLOYMENT_MANAGER_API_BASE` or `deployment-manager configure api_base "<url>"`. Config and token files live under `~/.deployment-manager/` by default.
