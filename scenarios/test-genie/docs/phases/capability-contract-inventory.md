# Phase Capability Contract — Inventory

This is the committed inventory of every Test Genie catalog phase and its
[Phase Capability Contract](../concepts/phase-capability-contract.md) posture. It
is drift-guarded: a test asserts the table matches the live catalog, so adding or
removing a phase forces this file to be updated.

## Posture

Most catalog phases are provider-delegated and compute their own
`MaturityAssessment`. `environment_conformance` is repository-scanned, and
`fault-injection` is a Test Genie validation-provider phase with a dedicated owner
handler. These exceptions remain in the catalog inventory and must keep their
contracts and remediation links explicit. Every phase carries a complete contract
(North Star + gated ladder + structured remediation doc), enforced by the
provider-conformance self-phase.

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
| `channel-conformance` | switchboard | 3 | provider-delegated | `scenarios/switchboard/docs/internal/PROBLEMS.md` |
| `cloud-production-health` | landing-page-business-suite | 1 | provider-delegated | `scenarios/test-genie/docs/phases/cloud-production-health/README.md` |
| `credential-leak-proof` | landing-page-business-suite | 1 | provider-delegated | `scenarios/landing-page-business-suite/docs/testing/credential-leak-proof.md` |
| `component-tests` | react-component-library | 1 | provider-delegated | `scenarios/test-genie/docs/phases/component-tests/README.md` |
| `contracts` | cli-health | 7 | provider-delegated (lighthouse) | `scenarios/test-genie/docs/phases/contracts/README.md` |
| `code-facts` | code-facts | 1 | provider-delegated | `scenarios/test-genie/docs/phases/architecture/README.md` |
| `dependencies` | scenario-dependency-analyzer | 6 | provider-delegated | `scenarios/test-genie/docs/phases/dependencies/README.md` |
| `docs` | knowledge-observatory | 7 | provider-delegated | `scenarios/test-genie/docs/phases/docs/README.md` |
| `event-capture-conformance` | vrooli-events | 1 | provider-delegated | `scenarios/test-genie/docs/phases/event-capture-conformance/README.md` |
| `fault-injection` | test-genie | 1 | test-genie-native (validation-provider) | `scenarios/data-backup-manager/docs/reference/support-matrix.md` |
| `go-code-graph` | go-code-graph | 1 | provider-delegated | `scenarios/test-genie/docs/phases/architecture/README.md` |
| `measures` | measures-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/measures/README.md` |
| `model-eval` | ai-gateway | 1 | provider-delegated | `scenarios/ai-gateway/docs/operations/model-eval.md` |
| `monetization-conformance` | landing-page-business-suite | 1 | provider-delegated | `scenarios/test-genie/docs/phases/monetization-conformance/README.md` |
| `performance` | performance-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/performance/README.md` |
| `programs` | program-runtime | 3 | provider-delegated | `scenarios/test-genie/docs/phases/programs/README.md` |
| `portability` | scenario-dependency-analyzer | 1 | provider-delegated | `scenarios/test-genie/docs/phases/portability/README.md` |
| `proto` | proto-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/proto/README.md` |
| `provider-conformance` | test-genie | 2 | provider-delegated (self) | `scenarios/test-genie/docs/phases/provider-conformance/README.md` |
| `quality` | quality-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/quality/README.md` |
| `search` | search-hub | 4 | provider-delegated | `scenarios/test-genie/docs/phases/search/README.md` |
| `security` | security-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/security/README.md` |
| `skill-set` | prompt-manager | 3 | provider-delegated | `scenarios/test-genie/docs/phases/skill-set/README.md` |
| `soak` | audio-tools | 1 | provider-delegated | `scenarios/audio-tools/docs/test-genie/soak/README.md` |
| `stripe-test-outcomes` | landing-page-business-suite | 1 | provider-delegated | `scenarios/test-genie/docs/phases/stripe-test-outcomes/README.md` |
| `storage` | storage-manager | 5 | provider-delegated | `scenarios/test-genie/docs/phases/storage/README.md` |
| `structure` | structure-health | 5 | provider-delegated | `scenarios/test-genie/docs/phases/structure/README.md` |
| `typescript-code-graph` | typescript-code-graph | 1 | provider-delegated | `scenarios/test-genie/docs/phases/architecture/README.md` |
| `environment_conformance` | envresolve-go | 4 | repository-scanned | `scenarios/test-genie/docs/phases/env-conformance/README.md` |
| `tidiness` | tidiness-manager | 4 | provider-delegated | `scenarios/test-genie/docs/phases/tidiness/README.md` |
| `templates` | template-manager | 4 | provider-delegated | `scenarios/test-genie/docs/phases/templates/README.md` |
| `experience` | ui-health | 6 | provider-delegated | `scenarios/test-genie/docs/phases/experience/README.md` |
| `experience-contract` | experience-manager | 4 | provider-delegated | `scenarios/test-genie/docs/phases/experience-contract/README.md` |
| `consumer-contract` | react-component-library | 1 | provider-delegated | `scenarios/test-genie/docs/phases/consumer-contract/README.md` |
| `unit` | unit-health | 6 | provider-delegated | `scenarios/test-genie/docs/phases/unit/README.md` |
| `workflow` | workflow-health | 6 (phase-level) | provider-delegated | `scenarios/test-genie/docs/phases/workflow/README.md` |

`contracts` (cli-health) is the reference lighthouse adopter proven end to end
(run → scorecard → doc-search topic → structured doc). `provider-conformance`
(test-genie) is the recursion case: Test Genie's own descriptor is validated by
the same contract it enforces on every other phase.
