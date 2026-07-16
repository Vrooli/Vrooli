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

## Why A Tiered Model Exists

Vrooli needs a tiered deployment model because different targets have materially different constraints:

- local stack deployments can rely on the full platform
- portable bundles need dependency fitness, packaging, and runtime ownership
- cloud or hosted installs need clearer server assumptions and secret handling
- appliance-style deployments belong to strategy until they become real maintained offerings

The tier model prevents project docs from pretending that every target is equally mature.

## Deployment Tiers

| Tier | Meaning | Status | Where To Read |
|------|---------|--------|---------------|
| 1 | Full local stack on a local machine or operator-controlled server | Canonical current path | [../operations/production-guide.md](../operations/production-guide.md) |
| 2 | Portable desktop bundles carrying scenario runtime pieces together | Roadmap direction; not general production canon | [../strategy/roadmap.md](../strategy/roadmap.md) |
| 3 | Mobile delivery targets | Directional roadmap only | [../strategy/roadmap.md](../strategy/roadmap.md) |
| 4 | Server-hosted or SaaS-style installs | Reference and planning, not a fully standardized project-wide path | [reference/server-deployment.md](reference/server-deployment.md) |
| 5 | Enterprise or hardware-appliance style deployments | Strategic framing only | [../strategy/business-solutions.md](../strategy/business-solutions.md) |

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

The intended future shape is:

1. A deployment-aware scenario or control surface evaluates a target tier.
2. It resolves the scenario dependency graph, including resources and other scenarios.
3. It scores dependency fitness for the requested target.
4. It determines what must be bundled, swapped, regenerated, or kept remote.
5. It produces or drives the target-specific packaging or server rollout path.

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
