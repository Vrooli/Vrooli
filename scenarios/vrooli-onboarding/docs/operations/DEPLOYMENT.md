# Deployment

Onboarding ships on every tier it configures. The flow is identical; only
catalog and state resolution differ.

| Tier | Catalog | Operator state | Entry |
|---|---|---|---|
| Repository install | Repository root | `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` | `make start`, or the wizard opens after setup |
| Desktop bundle | Staged bundle catalog | App-data storage root | The app's first-run flow |
| Remote host / VPS | That host's catalog | That host's state | vrooli-bridge or scenario-to-cloud driving the non-interactive surface |

## Supported environment and target capabilities

The deployment tier describes where onboarding runs. The selected target
describes which machine it inspects or configures. These are separate choices;
the current implementation does not make every capability portable to every
target.

| Environment / target | Supported capability | Unsupported or unverified edge | Current evidence |
|---|---|---|---|
| Repository install, local target | Read local host facts and manifest-derived requirements; configure local safeguards and the local notification recipient. Project-scope requirements are included. | Applying local host changes still requires the owning host tool and operator consent. | `api/host.go`; `api/host_requirements_conformance_test.go` checks repository scope parity. |
| Desktop bundle, local target | Use staged scenario, resource, tool and safeguard catalogs; keep operator state in app-data storage; inspect the bundled machine. | Repository-root requirements and writes are excluded. The supported desktop OS/architecture set is determined by the bundle build and is not a universal cross-platform claim. | Bundle staging checks and the Bundle requirements contract below; `api/host.go` derives platform facts from the running host. |
| Remote host / VPS through vrooli-bridge | Discover registered nodes and expose a node as available only when its heartbeat, channel, protocol, dispatchability and required onboarding scopes are valid; inspect manifest-derived host requirements read-only. | Host-tool opt-in, safeguard and notification-recipient writes are unsupported by onboarding's local-only control-plane owners. The wizard disables those mutations; the target API rejects safeguard and notification-recipient procedures before dispatch. A discovered/offline node is not proof that remote setup or recovery is available. | `api/host_targets_test.go` covers online/offline target state, missing scopes and fail-closed source errors; `api/internal/targetproxy/interceptor_test.go` proves unsupported writes never reach Bridge; `ui/src/components/wizard/HostRequirementStep.test.tsx` verifies read-only display and disabled controls. |
| Local host with graphical session | Use a native credential store when the platform provider is present and usable. | A graphical session alone does not prove a credential provider is available; provider status must be evaluated. | Credential provider diagnosis is surfaced by readiness. |
| Headless host | Use the encrypted file credential store after it is initialized. | Credential provisioning is unsupported when neither native nor initialized encrypted-file storage is usable; no silent downgrade or split authority is allowed. | `docs/internal/SECURITY.md`; credential readiness and deployment runbook. |
| Physical device / TV | No Onboarding device-control capability is declared by this deployment contract. | Direct device configuration and recovery are unsupported here; this row does not qualify Portal's separate TV integration. | No Onboarding device-control owner or journey is declared in this scenario. |

The API currently returns the local machine as an available target and adds
bridge nodes only through the registered-node discovery path. Target
availability is an admission signal, not a promise that all host operations
are implemented for that target. In particular, safeguards and notification
recipient updates stay on the local host/operator-state owners. Do not infer
support for a provider, operating system, device, or remote write path from a
successful local readiness response.

## Bundle requirements

A desktop bundle must stage **every** manifest class the wizard reads:
`catalog/scenarios/`, `catalog/resources/`, `catalog/internal/tools/`, and
`catalog/internal/safeguards/`. A bundle missing one fails the packaging check
rather than reaching an operator and failing at first launch.

The set to stage is not hand-maintained: it comes from
`SelectionService/GetUnion` for
the bundled scenario's selection.

The desktop runtime supplies `BUNDLE_ROOT` and `VROOLI_STORAGE_ROOT` and
deliberately does not supply `VROOLI_ROOT` — a standalone app has no monorepo,
and writing into a repository it may not own would be worse.

If a resource catalog becomes unavailable after packaging, `ListDerivedResources`
returns a retryable unavailable status; operator choices remain in the separate
state root and are visible again when the catalog is restored. A malformed staged
resource manifest is reported as an invalid catalog rather than an empty selection.

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
