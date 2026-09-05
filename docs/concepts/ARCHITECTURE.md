# Vrooli Architecture

> **Owner:** `director-swarm` (drift detection via `vision-walk-prep` + `vision-update` decision context). **Author:** operator-direct. **Status:** sketch-level today; canonical-technical-reference expansion is tracked as a swarm-manager backlog candidate (flagged at vision walk #4, 2026-04-27). When expanded, this becomes the canonical "how Vrooli actually works" reference for technical readers, agents, and architectural-question answers. Until then, treat this as a pointer / starting-point and supplement with `README.md` and `VISION.md` for the broader picture.

This document describes the current platform architecture at a high level.

## Core Model

Vrooli is built around a compounding loop:

1. agents use resources to solve problems
2. solutions become scenarios, workflows, packages, or patterns
3. those artifacts become reusable capabilities
4. future work starts from a stronger base

The platform is therefore best understood as a **local software foundry** rather than a single application.

## Core Architectural Principles

### Wrap-not-use

**Agents should use scenarios, not external tools directly.** External tools (git, browsers, web APIs, search engines, etc.) are systematically replaced by Vrooli scenarios that wrap them. The long-run direction is to forbid direct external-tool use entirely; agents go through the scenario or fail. (Capability and reliability are not yet sufficient to enforce that hard rule today — the trajectory is set, the enforcement is gradual.)

**The maturation pattern (proven on GCT, BAS, others):**

1. Start as a simple wrapper — minimal logic, cheap to build because scenario templates and generation tooling keep improving.
2. Add custom capabilities incrementally as needs arise: permissions, analytics, identity-aware policies, integration with other scenarios, custom protections.
3. Eventually wrap the underlying tool's CLI itself with a script that warns or blocks direct use.

**Canonical examples:**

- **Git → Git Control Tower (GCT).** Wraps `git`. Already blocks destructive ops by agents. Coming: per-commit run-attribution from agent-manager workspace sandbox, auto-generated commit messages, auto-PR generation, identity-gated permissions, usage analytics.
- **Browser / web → Browser Automation Studio (BAS).** Wraps browser-use. Adds end-to-end UI testing, screenshot + video capture, known-issue handling, integration with scenario UIs.
- **Sandboxing → agent-manager workspace-sandbox.** Per-run file-change attribution feeds GCT. Coding-agent processes themselves run inside the sandbox via the `runner.Launcher` seam in protected mode (default since 2026-04-28); see `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md`.

**Why this isn't "extra work" in the long run:**

- Scenario templates + reliable generation make initial wrapping cheap.
- A wrapper starts as a simple passthrough and gains custom capability only as needed — never speculatively.
- Identity comes from agent-manager (agents are spawned through it), so permission/analytics layers fall out naturally.
- Sandboxed agents can perform approved privileged work through controlled scenario surfaces instead of direct host access. For example, `vrooli scenario start|restart|stop` from a protected workspace is proxied through workspace-sandbox to the host lifecycle command, while arbitrary Docker, systemd, or filesystem mutation remains unavailable unless a Vrooli scenario deliberately wraps it.
- Each wrapper becomes a control point for future capability layering. Strategic value compounds.

**Corollary — internal scope discipline.** The wrap-not-use principle also applies to internal domain boundaries. Each Vrooli scenario stays in its own lane: domain-specific CLIs (marketing-publisher commands, swarm-manager backlog commands, etc.) live in their own scenarios, not bolted onto generic platforms like prompt-manager. Generic team / coordination primitives belong in prompt-manager; everything else in its domain scenario.

### Scenarios as substrate

Scenarios are the unit of accumulation, not just the unit of execution. Every solved problem crystallizes into a scenario; every scenario becomes a permanent capability future scenarios can compose. The platform's intelligence is the tech tree of scenarios it has built, plus the agents that build new ones.

This is why scenarios are dual-purpose by design — each is simultaneously a product (revenue-generating), a capability (composable), and a test (validates underlying resources work together). Treating scenarios as ephemeral tasks would lose the compounding.

### Operator steers, agents execute

The morning vision walk is the steering interface; the rest of the system runs on agent loops with structured decision channels. Operator authority is asserted through accepted decisions, not direct execution. Agents respect approval boundaries even when the agent is technically capable of acting unilaterally — the boundary is the contract, not a capability gap.

## Primary Layers

### Control Plane

The root control plane is the Go-native `vrooli` CLI and its supporting project internals.

Its responsibilities include:

- setup and development lifecycle
- scenario lifecycle management
- resource lifecycle management
- package governance
- diagnostics and maintenance
- repo-contract validation and path resolution

The most important project entrypoints live under:

- `cmd/`
- `internal/`
- `packages/`

## Resources

Resources provide raw capability.

Typical categories include:

- AI and inference
- relational, cache, vector, and object storage
- browser and workflow automation
- secrets and infrastructure helpers
- supporting execution environments

Resources are not the end product. They are the capability layer that scenarios compose.

## Scenarios

Scenarios are the application and orchestration layer.

A scenario may include:

- UI
- API
- CLI
- tests
- manifests
- deployment metadata
- initialization or runtime assets

Some scenarios are user-facing products. Others are meta-scenarios that improve the platform itself.

### Web Console terminal device and size-lease boundary

Web Console keeps terminal size authority per WebSocket connection while using a
browser-local device id to group that device's connections. A reconnect from the
same device can reclaim the session's size lease after the prior socket fails a
liveness probe; the superseded socket is then closed by the server. Device labels
and device classes are recognition-only metadata and never authorize an action.
The live device roster is a projection of session connections exposed by
`DeviceService` and refreshed through the existing lifecycle event stream.

## Supervision Authority

Vrooli computes one supervision set; consumers do not maintain private core
lists. The operator grants the root authority in
`.vrooli/operator-state.json` at `core.seed`. The control plane starts with
those scenario seeds and follows enabled scenario and resource dependency
edges whose effective supervision intent is `must_start` or `try_start`.
`ignore` edges do not enter the closure. The computed result is an output, not
a second stored declaration.

Run the authority directly:

```bash
vrooli supervision-set
vrooli supervision-set --kind resource --json
```

Each returned member includes an attribution chain ordered from that member
back to the seed that granted its inclusion. Each link names the declaring
scenario, the dependency kind, the effective intent, and the manifest source;
the final link is `core.seed`. This makes “why is qdrant supervised?” a query,
not a source-code investigation. `scenario-dependency-analyzer core-set` is a
compatibility presentation of the same computation.

### Supervision-intent precedence

Dependency manifests still carry the legacy `enabled`, `required`, and
`startup_policy` fields. The scenario parser derives exactly one intent using
this table:

| Declaration | Effective intent | Required acknowledgement |
|---|---|---|
| `enabled: false` | `ignore` | none; disabled wins over retained metadata |
| enabled, `required: true`, policy absent or `must_start` | `must_start` | none |
| enabled, `required: true`, `startup_policy: try_start` | `must_start` | `supervision_precedence: required` |
| enabled, `required: false`, policy absent or `try_start` | `try_start` | none |
| enabled, `required: false`, `startup_policy: must_start` | `must_start` | `supervision_precedence: startup_policy` |
| enabled, `required: false`, `startup_policy: ignore` | `ignore` | none |
| enabled, `required: true`, `startup_policy: ignore` | invalid | rejected by manifest validation |

The conservative default for an enabled optional edge is `try_start`: silent
capability loss is harder to observe than the cost of an attempted start. Run
`vrooli scenario validate` to enforce the table and the explicit precedence
markers.

### Ownership records outrank ancestry

Process ancestry is evidence about how a process was launched, not authority
over who owns it. A live managed-resource record, resource companion record,
or scenario process record is an ownership record. When its PID and process
start time match the host process, maintenance classifies that process as
tracked even if its parent has exited or belongs to another process tree. A
record for a dead or reused PID grants no protection. If the host cannot supply
the start-time evidence needed for that check, the classifier reports the
unsupported condition instead of guessing.

This principle prevents `vrooli cleanup orphans` from killing a valid daemon
because ancestry made it look detached. Inspect the computed result before any
cleanup:

```bash
vrooli orphans --json
```

### Boot recovery

The processes that restart everything else (the autoheal loop, the runtime
supervisor, the emergency watchdog) run from native units rendered from one
typed definition, `platformgo.ServiceDefinition`, validated by the native
manager before they are enabled, and converged by a host safeguard on every
`vrooli setup`. Their argv comes from the `cliinvoke` catalog and carries no
global flags; a retired global is tolerated with a warning for a grace period
and every registered invoker's argv is parsed through the real root parser in
a test. `vrooli setup status --json --phase readiness` re-inspects the three
units while the host is healthy, so a render the manager would refuse is seen
before the next boot rather than after it. See
[native service definitions](../reference/native-service-definitions.md) and
[CLI invokers](../reference/cli-invokers.md).

### Autoheal consumption and safety interlock

Autoheal reloads `vrooli supervision-set --json` at startup and every 30
seconds. `must_start` produces a critical target check; `try_start` produces a
warning-level target check. The autoheal operator config may add targets, but
it cannot remove or disable a canonical member. A source failure retains the
in-memory last-known-good non-empty set and degrades the
`vrooli-supervision-set-source` check. Startup fails closed when no good set
has ever loaded.

Correct ownership classification prevents one false orphan path, but it cannot
make independently scheduled checks atomic. The heal interlock therefore
guards the action boundary separately. After a successful start-like action,
it refuses a dangerous action from a different check against the same target
for 30 seconds. The same check may verify or repair its own lifecycle. The
refusal is returned as typed action evidence and follows the normal durable
action-log path. This second guard contains cross-check races even if a future
classifier or check regresses.

Automatic recovery is bounded. After the configured consecutive-failure limit
(three by default), the check is suspended and autoheal raises a durable
incident. Health observation continues, but recovery actions stop until an
operator fixes the cause and runs `vrooli-autoheal check resume <check-id>`.
`vrooli-autoheal check suspended` shows the active suspension and its retained
attempt history.

### Outage record

For a supervised member, the first explicit unavailable observation opens one
`outage_records` interval. Repeated unavailable observations keep that interval
open. The next explicit available observation closes it. A degraded resource
that is still serving counts as available, and an unreadable supervision
source fabricates neither an opening nor a closing observation.

One typed query returns the two windowed aggregates required to answer both
“how long?” and “how often?”:

```bash
vrooli-autoheal measure outages --member-id resource-qdrant --window-hours 24
```

The CLI JSON response includes `totalUnavailableSeconds` and
`distinctOutageCount`; it also reports `openOutageCount`. Protobuf source uses
the corresponding snake-case field names. Zero-valued fields may be omitted by
protobuf JSON. An open interval
is measured through the query window end without being mutated. The intervals
and aggregates are stored in autoheal's bounded SQLite ledger and survive an
API restart.

The investigation that established this model, including the oscillation
timeline, rival authorities, attempt counts, and rejected fixes, is preserved
as [AI resource fleet supervision origin evidence](../reports/ai-resource-fleet-supervision-readout-2026-08-28.html).

## Dependency & Isolation Model

Scenarios are isolated at the dependency layer so that upgrading one scenario never forces a migration in another, and so a scenario can in principle be built with any language, framework, or package manager — the platform contract is **process-level** (`.vrooli/service.json`, Makefile lifecycle targets, health endpoints), not package-level.

There are two deliberately different dependency classes:

### Third-party dependencies — fully isolated

- Every scenario UI is a **standalone pnpm project**: its own `package.json`, its own committed `pnpm-lock.yaml`, its own `node_modules`. Bumping React (or anything else) in one scenario touches nothing else. pnpm's content-addressable store keeps this cheap — identical versions are hard-linked once on disk, not duplicated per scenario.
- Every scenario API is its own Go module (`go.mod` per scenario) with independently pinned third-party versions.
- The repository structure and the executable policies for this dependency topology are owned by [Structure Health's generated rule catalog](../../scenarios/structure-health/docs/reference/structure-rules.md) and [coverage matrix](../../scenarios/structure-health/docs/reference/structure-rule-coverage.md). This project-level page explains the architecture; it does not restate enforcement claims.

### First-party shared packages — deliberately source-coupled

Shared Vrooli packages are consumed **by source path**, not by published version:

- JS: `"@vrooli/api-base": "file:../../../packages/api-base"` in scenario UIs
- Go: `replace github.com/vrooli/api-core => ../../../packages/api-core` in scenario `go.mod`s

This means every scenario builds against HEAD of `packages/*`. A breaking change there propagates fleet-wide at the next build/install — the opposite of the third-party isolation above. **This is an accepted tradeoff, not an oversight**: source coupling is what makes the compounding loop cheap (one improvement to a shared package is instantly available to every scenario, with no release ceremony), and the fleet is small enough that conformance discipline is cheaper than version management. The guards are additive-evolution discipline on shared package APIs (see the API surface manifest & conformance work) and buf breaking-change checks on the shared proto contracts.

**Revisit trigger:** if fleet-wide breakage from shared-package changes starts costing more than release ceremony would (recurring multi-scenario build breaks of the kind conformance checks don't catch), move `packages/*` to versioned releases that scenarios pin and upgrade deliberately.

### React component restyling

Library components expose a stable consumer styling seam:

- `className` is merged with the component default through `cn` from `ClassMerge`.
- The consumer ref reaches the outermost host element through `forwardRef` (or the documented imperative-ref seam for components that expose a command handle).
- Token and variant declarations live in a co-located stylesheet and are selected by `data-*` attributes. A component does not use a static inline style object for token values, because inline declarations defeat a consumer stylesheet override.
- `style` remains the native `CSSProperties` escape hatch for consumer-owned or computed values such as measured dimensions, positions, and transforms. Components do not invent a token-specific `style` type or overload the standard DOM prop.

`ValidateRestyleContract` enforces these rules against the active catalog sources. The package build copies version-local CSS into `dist`, so the contract applies equally to linked adopters and package consumers.

### React component automation selectors

Every exported library component emits a root `data-testid` equal to its
catalog id. Additional selectors use the same dotted namespace, for example
`navigation.sidebar.close`. Adopter registries are generated from those
values and expose semantic fields such as `root`, `close`, and
`resizeHandle`; positional names such as `id2` are not a stable contract.

The BAS flows and cases consume the adopter selector registry, so a selector
rename is complete only when the registry, its generated manifest, and the
owning flow references are regenerated together. `selector-coverage` checks
the library source, while `selectors-adopted` checks that linked scenarios
carry the generated registry and semantic naming convention.

## Meta-Systems

Vrooli increasingly relies on scenarios that improve other scenarios, operator workflows, and governance loops.

This includes areas such as:

- testing and requirement validation
- deployment planning
- issue tracking and backlog control
- swarm or team coordination
- browser automation and operator assist surfaces

## Deployment Reality

Project-level docs should distinguish between current and future deployment maturity.

- Tier 1 local/dev stack is the primary mature path today.
- Desktop, mobile, SaaS, and appliance targets are important, but their maturity depends on packaging, dependency fitness, and tier-specific constraints.

Use [../deployment/README.md](../deployment/README.md) as the canonical deployment truth.

## Cross-Platform Direction

The platform should now be documented as a cross-platform Go-native control plane.

That does not mean every scenario or every resource is equally portable yet. It means:

- the project-level contract is cross-platform
- the root CLI is Go-native
- repo-aware behaviors should be described in contract-backed terms
- shell-era assumptions should not be presented as the authoritative model

## Documentation Boundaries

Project-level architecture docs should explain:

- how the platform is organized
- how resources and scenarios relate
- what the root control plane is responsible for
- where to look next for deeper system-specific detail

They should not duplicate:

- scenario-specific design docs
- resource-specific implementation detail
- active implementation plans, unless clearly marked as plans
# Asset-first catalog validation

The React Component Library gate loop is asset-first: it resolves each asset's
universal, kind, and asset-level rules, then invokes registered runners with a
scoped asset set. Corpus-scoped graph rules run once. Evidence is reusable only
when the asset revision and resolved rule-set digest both match, and findings
carry the rule source and declaring file for traceability.
