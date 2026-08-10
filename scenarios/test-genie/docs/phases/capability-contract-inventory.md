# Phase Capability Contract — Inventory

This is the committed inventory of every Test Genie catalog phase and its
[Phase Capability Contract](../concepts/phase-capability-contract.md) posture. It
is drift-guarded: a test asserts the table matches the live catalog, so adding or
removing a phase forces this file to be updated.

## Posture

**Every catalog phase is provider-delegated.** The catalog is built entirely from
provider `.vrooli/test-genie.json` descriptors (`loadDefaultCatalogFromDescriptors`),
so there is **no Test-Genie-native phase** today: each phase is owned by a provider
scenario that computes and returns a `MaturityAssessment`. The native-phase
exemption mechanism described in the contract is therefore **reserved but unused** —
if a future native phase is added that cannot fit a provider ladder, it is marked
with an explicit, documented exemption rather than silently skipped. Until then,
every phase carries a complete contract (North Star + gated ladder + structured
remediation doc), enforced by the provider-conformance self-phase.

## Inventory

Capabilities = the number of per-capability ladders the descriptor declares (or
the phase-level ladder height when the provider uses a single ladder). Every phase
below is provider-delegated and declares a maturity ladder; every phase's
`docs.path` target follows the required remediation-doc skeleton.

| Phase | Provider | Capabilities | Posture | docs.path |
|-------|----------|--------------|---------|-----------|
| `ai-conformance` | ai-gateway | 4 | provider-delegated | `scenarios/test-genie/docs/phases/ai-conformance/README.md` |
| `agent-conformance` | agent-manager | 1 | provider-delegated | `scenarios/test-genie/docs/phases/agent-conformance/README.md` |
| `api` | api-health | 6 | provider-delegated | `scenarios/test-genie/docs/phases/api/README.md` |
| `architecture` | architecture-cartographer | 5 | provider-delegated | `scenarios/test-genie/docs/phases/architecture/README.md` |
| `branding` | brand-manager | 6 | provider-delegated | `scenarios/test-genie/docs/phases/branding/README.md` |
| `business` | business-health | 4 | provider-delegated | `scenarios/test-genie/docs/phases/business/README.md` |
| `component-tests` | react-component-library | 1 | provider-delegated | `scenarios/test-genie/docs/phases/component-tests/README.md` |
| `contracts` | cli-health | 7 | provider-delegated (lighthouse) | `scenarios/test-genie/docs/phases/contracts/README.md` |
| `dependencies` | scenario-dependency-analyzer | 6 | provider-delegated | `scenarios/test-genie/docs/phases/dependencies/README.md` |
| `docs` | knowledge-observatory | 7 | provider-delegated | `scenarios/test-genie/docs/phases/docs/README.md` |
| `event-capture-conformance` | vrooli-events | 1 | provider-delegated | `scenarios/test-genie/docs/phases/event-capture-conformance/README.md` |
| `experience` | experience-manager | 4 | provider-delegated | `scenarios/test-genie/docs/phases/experience/README.md` |
| `measures` | measures-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/measures/README.md` |
| `performance` | performance-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/performance/README.md` |
| `portability` | scenario-dependency-analyzer | 1 | provider-delegated | `scenarios/test-genie/docs/phases/portability/README.md` |
| `proto` | proto-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/proto/README.md` |
| `provider-conformance` | test-genie | 2 | provider-delegated (self) | `scenarios/test-genie/docs/phases/provider-conformance/README.md` |
| `quality` | quality-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/quality/README.md` |
| `search` | search-hub | 4 | provider-delegated | `scenarios/test-genie/docs/phases/search/README.md` |
| `security` | security-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/security/README.md` |
| `storage` | storage-manager | 5 | provider-delegated | `scenarios/test-genie/docs/phases/storage/README.md` |
| `structure` | structure-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/structure/README.md` |
| `tidiness` | tidiness-manager | 4 | provider-delegated | `scenarios/test-genie/docs/phases/tidiness/README.md` |
| `templates` | template-manager | 4 | provider-delegated | `scenarios/test-genie/docs/phases/templates/README.md` |
| `ui-health` | ui-health | 6 | provider-delegated | `scenarios/test-genie/docs/phases/ui-health/README.md` |
| `unit` | unit-health | 6 | provider-delegated | `scenarios/test-genie/docs/phases/unit/README.md` |
| `workflow` | workflow-health | 6 (phase-level) | provider-delegated | `scenarios/test-genie/docs/phases/workflow/README.md` |

`contracts` (cli-health) is the reference lighthouse adopter proven end to end
(run → scorecard → doc-search topic → structured doc). `provider-conformance`
(test-genie) is the recursion case: Test Genie's own descriptor is validated by
the same contract it enforces on every other phase.
