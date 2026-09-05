# Deployment

Onboarding ships on every tier it configures. The flow is identical; only
catalog and state resolution differ.

| Tier | Catalog | Operator state | Entry |
|---|---|---|---|
| Repository install | Repository root | `.vrooli/operator-state.json` | `make start`, or the wizard opens after setup |
| Desktop bundle | Staged bundle catalog | App-data storage root | The app's first-run flow |
| Remote host / VPS | That host's catalog | That host's state | vrooli-bridge or scenario-to-cloud driving the non-interactive surface |

## Bundle requirements

A desktop bundle must stage **every** manifest class the wizard reads:
`catalog/scenarios/`, `catalog/resources/`, `catalog/internal/tools/`, and
`catalog/internal/safeguards/`. A bundle missing one fails the packaging check
rather than reaching an operator and failing at first launch.

The set to stage is not hand-maintained: it comes from `GET /api/v2/union` for
the bundled scenario's selection.

The desktop runtime supplies `BUNDLE_ROOT` and `VROOLI_STORAGE_ROOT` and
deliberately does not supply `VROOLI_ROOT` — a standalone app has no monorepo,
and writing into a repository it may not own would be worse.

## Credentials on a deployment target

A bundle must never embed a user credential. It provisions through the local
credential authority at run time.

A target with no graphical session has no native store, so the encrypted file
store is the authority there and must be initialized before provisioning. A
target with no working store of either kind is reported **unsupported** — it is
never silently downgraded, because credentials split across backends by session
health are invisible until a value cannot be found.

## Remote deployment

vrooli-bridge reaches the host and holds the connection; onboarding decides what
runs there. The boundary is a declarative selection document in, and a readiness
report plus a machine-readable exit code out, so provisioning can gate on a real
result instead of parsing prose.

## Release gating

Before a tier ships, the recorded journeys for that tier are handed to
deployment-manager as evidence. A tier with no journey evidence is not
releasable, regardless of unit coverage.
