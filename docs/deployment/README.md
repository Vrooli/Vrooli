# Deployment Hub

This is the project-level source of truth for how Vrooli reaches a target
machine. It explains the deployment model, ownership boundaries, and current
support status. It does not duplicate scenario-specific build instructions.

## Start here

| Question | Read |
| --- | --- |
| What deployment path is supported today? | [Current support](#current-support) |
| What happens when a scenario is packaged? | [Deployment flow](#deployment-flow) |
| How do I produce a desktop application? | [Scenario-to-desktop documentation](../../scenarios/scenario-to-desktop/docs/OVERVIEW.md) |
| How does deployment-manager decide whether a build may ship? | [deployment-manager documentation](../../scenarios/deployment-manager/docs/README.md) |
| What may a resource require on a target host? | [Resource deployment contract](../resources/deployment-contract.md) |
| How are credentials handled? | [Credential configuration](../configuration/secrets.md) |
| What evidence is required for desktop claims? | [Desktop evidence and communication contract](../reference/scenario-to-desktop-evidence-and-tier-contract.md) |

## Two tier vocabularies

Vrooli currently uses “tier” in two different contexts. Documentation must
qualify the context instead of using a bare number.

| Vocabulary | Tier 1 | Tier 2 | Owner |
| --- | --- | --- | --- |
| Deployment target | Local Vrooli stack | Portable desktop application | Deployment Hub and deployment-manager |
| Commercial delivery | Bundle apps | Self-hosted full Vrooli runtime | Monetization plan of record |

These are different axes. A commercial Tier 1 app can be produced by the
deployment Tier 2 desktop ramp. Use `deployment Tier 1`, `deployment Tier 2`,
`commercial Tier 1`, or the descriptive name when the distinction matters.
The commercial definitions live in [monetization tiers](../monetization/strategy/TIERS.md);
the technical definitions below are the deployment definitions.

## Current support

The current production baseline is the **deployment Tier 1 local stack**:

- a full Vrooli installation runs on infrastructure the operator controls;
- scenarios and resources use lifecycle-managed processes;
- production UI bundles are served during normal operation;
- app-monitor and other explicitly configured access paths provide remote use;
- credentials are resolved through the platform credential authority.

The **deployment Tier 2 desktop ramp** is implemented, but it is not a blanket
claim that every scenario or platform is production-ready.

| Capability | Current status |
| --- | --- |
| Electron wrapper generation | Implemented |
| External-server thin client | Supported as a deployment mode; the target server must be reachable and validated |
| Bundled private runtime | Implemented for eligible dependency plans; release promotion requires target-specific evidence |
| Shared Tier 1 resource | Consent and broker lease contract exists; release evidence is fixture/environment dependent |
| Tier 2 peer protocol | Unsupported; the provider resolver is not a peer protocol |
| Linux native desktop journey | The primary validated path when the required display and capture tools are available |
| Windows and macOS | Build/package claims are separate from native runtime claims and require target evidence |
| Mobile, hosted cloud, and appliance targets | Directional or reference material, not the current universal deployment path |

The words `compile`, `package`, `conditional`, `degraded`, `supported`, and
`promotable` are not interchangeable. A build can compile without being
eligible for a target, and an artifact can be useful for validation without
being promotable.

## Deployment planes

Each plane has one responsibility:

| Plane | Owner | Responsibility |
| --- | --- | --- |
| Governance | `deployment-manager` | Profiles, dependency fitness, approvals, release records, and promotion gates |
| Ramp | `scenario-to-desktop` and other `scenario-to-*` scenarios | Target-specific generation, packaging, signing, publication, and native execution |
| Dependency intelligence | `scenario-dependency-analyzer` | Dependency graph, resource requirements, target fitness inputs, and swap candidates |
| Credential authority | `secrets-manager` plus the platform credential packages | Credential declarations, classification, secure storage, recovery, and diagnosis |
| Evidence | `test-genie`, browser-automation-studio, workflow-health, and the ramp | Machine assertions and reviewer-visible evidence |
| Reach | `vrooli-bridge` | Execution and artifact transfer on trusted remote machines; it does not define desktop-session evidence |

The direction of control is from ramp to governance: a ramp asks
deployment-manager for a decision. deployment-manager does not secretly run a
packager or grant a release based only on a build result.

## Deployment flow

For a target-specific release, the normal flow is:

1. The author declares scenario dependencies and target intent.
2. Dependency intelligence resolves the scenario/resource graph.
3. deployment-manager scores target fitness and records required swaps,
   limitations, host requirements, and secret strategies.
4. The ramp produces a target manifest and stages only the selected artifacts.
5. The ramp runs preflight and native smoke journeys on the target.
6. Evidence is attached to the exact source revision and artifact set.
7. deployment-manager evaluates the release gate and records the decision.
8. The ramp publishes only when the gate permits publication.

For desktop, the manifest is the boundary between planning and execution. It
must state what is bundled, what remains remote, what is conditional, and what
is unsupported. The desktop runtime must not discover an undeclared dependency
after installation and silently change the deployment shape.

## Choosing a desktop mode

| Mode | Use when | Runtime ownership | Offline claim |
| --- | --- | --- | --- |
| Bundled private | The app must work without a Vrooli server and every required dependency has an eligible local route | Desktop supervisor owns only verified private services | Allowed only when every required capability is local and evidenced |
| External-server thin client | Users connect to a shared Tier 1 server | Tier 1 owns the API, resources, data, and credentials | Not available |
| Shared resource | A desktop app may reuse a running Tier 1 or authenticated peer provider | Broker owns authorization and leases; desktop does not own provider lifecycle | Depends on the provider |
| Tier 2 peer | Apps need to call one another directly | Not currently supported as a release capability | Not claimable |

The [desktop evidence contract](../reference/scenario-to-desktop-evidence-and-tier-contract.md)
defines the required assertions and evidence for these modes.

## Authoring rules

Scenario authors should provide:

- an honest deployment profile and dependency declarations;
- `bas/` workflows for the user journeys that matter;
- target-specific limitations and secret declarations;
- migrations or documented data compatibility rules for any target swap.

Scenario authors should not add packaging, release gates, or cross-platform
runtime supervision to the scenario itself. Those responsibilities belong to
the ramp, deployment-manager, and the shared resource/credential contracts.

## Related references

- [Scenario deployment guidance](../scenarios/DEPLOYMENT.md)
- [Resource deployment contract](../resources/deployment-contract.md)
- [Managed release authority](../configuration/release-authority.md)
- [Server deployment reference](reference/server-deployment.md)
- [Storage guidance](storage.md)
