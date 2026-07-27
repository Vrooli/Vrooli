# Deployment Hub

This section is the canonical entrypoint for project-level deployment guidance.

Vrooli does not currently have one uniform deployment story across every target. Deployment guidance has to distinguish clearly between:

- what is supported now
- what is reference material
- what is active planning or longer-range direction

## Current Truth

The current production-ready path is the Tier 1 local stack.

That means:

- a full Vrooli installation on Linux, macOS, or Windows through WSL2 on
  infrastructure you control
- an authenticated public installer that supplies a prebuilt `vrooli` CLI and
  its exact matching source tree before `vrooli setup` installs host tools
- scenario and resource lifecycle managed through the current `vrooli` CLI and scenario-local Makefiles
- operator-controlled access patterns such as app-monitor and related remote-access surfaces where appropriate
- local or dev-server style operation as the real current baseline

For active operations on that path, use:

- [../QUICKSTART.md](../QUICKSTART.md)
- [../operations/production-guide.md](../operations/production-guide.md)
- [../operations/logging.md](../operations/logging.md)
- [../operations/troubleshooting.md](../operations/troubleshooting.md)

## Responsibility Planes

Deployment work is split across four planes. Each plane has one job. A component belongs to exactly one plane.

| Plane | Component | Job |
|-------|-----------|-----|
| Governance | `deployment-manager` | Decides whether a build may ship, and records what shipped |
| Ramp | `scenario-to-desktop`, `scenario-to-ios`, `scenario-to-android`, `scenario-to-cloud`, `scenario-to-extension`, `scenario-to-mcp` | Builds, packages, signs, publishes, and exercises artifacts for one target family |
| Evidence | `browser-automation-studio`, `workflow-health`, `vrooli-emulator`, `test-genie` | Produces the evidence a gate decides on |
| Reach | `vrooli-bridge` | Runs flows on machines and operating systems that are not the local host |

The direction of control matters. A ramp calls deployment-manager; deployment-manager does not call a ramp to start work. A ramp asks for a gate decision, publishes only when the gate allows it, and reports the result back.

What is true today:

- `scenario-to-desktop` calls deployment-manager for approvals, for the release gate, and for bundle manifest generation
- deployment-manager holds the release records, the approval records, and the per-platform release state
- `vrooli-bridge` provisions and runs allowlisted commands across trusted nodes on every operating system

What is direction, not current truth:

- a single evidence contract shared by every ramp and every execution surface
- cross-tier evidence comparison in deployment-manager
- automatic capture of user-flow evidence during ramp smoke tests

## Why A Tiered Model Exists

Vrooli needs a tiered deployment model because different targets have materially different constraints:

- local stack deployments can rely on the full platform
- portable bundles need dependency fitness, packaging, and runtime ownership
- cloud or hosted installs need clearer server assumptions and secret handling
- appliance-style deployments belong to strategy until they become real maintained offerings

The tier model prevents project docs from pretending that every target is equally mature.

## Deployment Tiers

A tier describes where a scenario runs. A ramp describes how it is packaged. The two are different axes, and several ramps can serve one tier.

| Tier | Meaning | Ramp | Status | Where To Read |
|------|---------|------|--------|---------------|
| 1 | Full local stack on a local machine or operator-controlled server | none; the platform runs the scenario directly | Canonical current path | [../operations/production-guide.md](../operations/production-guide.md) |
| 2 | Portable desktop bundles carrying scenario runtime pieces together | `scenario-to-desktop` | Roadmap direction; not general production canon | [../strategy/roadmap.md](../strategy/roadmap.md) |
| 3 | Mobile delivery targets | `scenario-to-ios`, `scenario-to-android` | Directional roadmap only | [../strategy/roadmap.md](../strategy/roadmap.md) |
| 4 | Server-hosted or SaaS-style installs | `scenario-to-cloud` | Reference and planning, not a fully standardized project-wide path | [reference/server-deployment.md](reference/server-deployment.md) |
| 5 | Enterprise or hardware-appliance style deployments | none yet | Strategic framing only | [../strategy/business-solutions.md](../strategy/business-solutions.md) |

`scenario-to-extension` and `scenario-to-mcp` package a scenario to run inside a host application rather than on a tier of its own. Their tier assignment is not decided. Do not assume one.

## Current Deployment References

Use these according to their role:

- [../QUICKSTART.md](../QUICKSTART.md): canonical fresh-machine installation path
- [../operations/production-guide.md](../operations/production-guide.md): canonical Tier 1 operating baseline
- [reference/server-deployment.md](reference/server-deployment.md): server-oriented deployment reference
- [storage.md](storage.md): specialized bundle-storage reference
- [../strategy/personal-ai-server.md](../strategy/personal-ai-server.md): exploratory infrastructure and appliance-style thinking
- [../strategy/roadmap.md](../strategy/roadmap.md): current desktop, mobile, and cross-platform direction

## Deployment Intelligence Direction

The long-term deployment direction is still important, but it is not current operational truth.

`deployment-manager` is the governance plane named above. The intended future shape is:

1. `deployment-manager` evaluates a target tier against a deployment profile.
2. It resolves the scenario dependency graph through `scenario-dependency-analyzer`, including resources and other scenarios.
3. It scores dependency fitness for the requested target.
4. It determines what must be bundled, swapped, regenerated, or kept remote.
5. A ramp packages the artifact and asks `deployment-manager` for a gate decision.
6. `deployment-manager` decides on the recorded evidence, then records the release.

Steps 1 through 4 exist today. Step 5 exists for `scenario-to-desktop` only. Step 6 records releases but does not yet decide on evidence beyond a human approval.

That direction explains the planning docs, but it should not be read as a finished current deployment system.

## Bundles And Portability

For desktop, mobile, or cloud-oriented bundle work, keep these rules explicit:

- bundles are not the default deployment model today
- heavy resources may need target-specific substitutions before packaging
- secrets handling changes by deployment tier
- runtime ownership, logs, data directories, and health checks must be defined by the target packaging flow rather than assumed

## Provider Notes

Provider-specific deployment guidance is still thin at the canonical layer.

If you need server-oriented assumptions today, start with:

- [reference/server-deployment.md](reference/server-deployment.md)
- [../operations/production-guide.md](../operations/production-guide.md)

Do not treat older package-and-ship or removed Kubernetes material as authoritative unless it is rebuilt into this section intentionally.

## Historical And Planning Material

Deployment docs elsewhere in the repository may still be useful, but they fall into two non-canonical classes:

- reference material that helps reason about one deployment shape
- planning material that describes target future behavior

Those docs are valuable only when they are read with their status in mind.
