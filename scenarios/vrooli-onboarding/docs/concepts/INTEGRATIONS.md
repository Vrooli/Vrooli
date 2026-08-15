# Integrations

## What onboarding integrates with

| Counterpart | Direction | Contract |
|---|---|---|
| `internal/operatorstate` | Onboarding → | Field-scoped merge patches. The only write path for operator decisions |
| Credential authority | Onboarding → | One-way value relay; status metadata back. Never a read path for a value |
| Control plane (`hostreq`, `resources`, `lifecycle`) | Onboarding → | Apply delegates each action and receives an outcome plus a remediation |
| `secrets-manager` | Peer | Owns credential lifecycle, keyring repair, and recovery bundles. Onboarding provisions and reports; it does not duplicate that surface |
| `scenario-to-desktop` | ← Onboarding | Consumes the union export to decide bundle contents; supplies the bundle root and app-data state root at runtime |
| `vrooli-bridge` | ← Onboarding | Drives the non-interactive surface against a remote host and reads back a readiness report and exit code |
| `scenario-to-cloud` | ← Onboarding | Same surface for a VPS target |
| `vrooli-autoheal` | ← Onboarding | Reads the committed selection and the completion marker to know what should be running |
| `search-hub` | ← Onboarding | Indexes the operator-surface feed so a setting is findable by intent *(deferred)* |
| `experience-manager` | ← Onboarding | Validates the page, state, claim, and journey contract in `experience/` |
| `browser-automation-studio` | ← Onboarding | Records journey evidence per declared journey |
| `integration-hub` | — | **Deferred.** Owns connectors and connections |

## The bridge boundary

vrooli-bridge owns reaching a machine and holding the connection. Onboarding owns
deciding what runs there. Bridge sends node identity to `POST /api/v2/handoff`;
onboarding returns the effective capability-shaped selection, without exposing
operator-state internals or credential values. Bridge transfers that selection
through its private remote temporary-file path, then runs the onboarding CLI and
reads back the authoritative readiness report and machine-readable exit code.

Keeping it narrow is what lets the same surface serve a VPS, a LAN machine, and a
CI runner without bridge learning anything about scenarios or onboarding learning
anything about transport.

## Integration Hub — deferred

Connector definitions, connection instances, OAuth and device-code flows, and
per-context bindings for multi-instance scenarios are owned by integration-hub.

Until it ships, wizard step 4 is empty. Where a selected scenario declares an
`integrations[]` requirement, the step names it as deferred and creates no
binding — a fabricated connection produces state nothing can honour, discovered
at the worst possible time.

Scenarios that need an external service before integration-hub exists either use
a resource with a paste-string credential descriptor, or wait.

See [`connectors.md`](../../../../docs/configuration/integrations/connectors.md)
and [`external-auth.md`](../../../../docs/configuration/integrations/external-auth.md).
