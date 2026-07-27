# Resource Interface Standards

This page describes the current expectations for resource-facing platform surfaces.

## Current Truth

The canonical project-level resource surface is the `vrooli resource` command group.

Resource-specific wrappers may still exist in some places, but they should not be treated as the default design center for new resource work unless explicitly retained.

## Expectations

At the platform level, a well-behaved resource should have:

- a clear canonical name
- discoverable metadata
- manifest-backed configuration where implemented
- status and validation behavior
- lifecycle behavior that is consistent with the control plane

## Useful Commands

```bash
vrooli resource list
vrooli resource info "<name>"
vrooli resource status
vrooli resource validate "<name>"
vrooli resource logs "<name>"
vrooli resource restart "<name>"
```

## Guidance

- prefer consistency with the root CLI
- avoid inventing special-case interfaces without strong reason
- keep compatibility shims isolated and documented as transitional when they still exist
- keep resource-specific wrappers subordinate to the canonical `vrooli resource ...` surface
- keep resource-local entrypoints thin when shared control-plane packages can own the behavior
- keep source-build tooling separate from deployed artifact delivery; an
  end-user resource path must not depend on Go or Bash
- expose target limitations, host requirements, and fallbacks before scenario
  deployment/runtime attempts the resource

## Health Checks Are Readiness Probes

The manifest `health_checks` entries are what orchestration ordering, status,
and consumer auto-recovery believe. They must therefore mean "able to serve
the primary capability" — a service whose model has not loaded, whose first
request would block on a large download, or whose admission path is closed is
not healthy, even if its process answers a liveness endpoint.

Declare semantics explicitly with the `kind` field (`readiness` or
`liveness`). Manifest-level checks are treated as readiness by default; point
them at a real readiness endpoint (for example `/ready`, not `/health`) when
the service distinguishes the two.

## Declared Degradation

A resource that can run in more than one mode (GPU/CPU device, model size,
engine fallback) must:

- expose the active mode on an info/status surface an operator and a consumer
  can query
- report running below its configured mode as visible degraded status, not as
  silent fallback

Silent mode switches are the most expensive failure class in fleet history:
they keep health green while quality or correctness quietly collapses.

## GPU and Capacity

A `gpu` block without a `capacity` block hides the resource from the capacity
broker's VRAM planning. Declare both, or record the deliberate decision to
keep the broker out. The fleet lint in
`internal/resources/manifest/runtime_pinning_realmanifest_test.go` enforces
this with an explicit exception list.

## Startup Timeouts

`orchestration.startup_timeout_seconds` must budget the worst normal case,
including first-run model/image downloads. Use `startup_time_estimate` to
describe the warm-start experience; never tune the enforced timeout to the
warm case, or the orchestrator will kill healthy first starts.

## Related

- [architecture.md](architecture.md)
- [deployment-contract.md](deployment-contract.md)
- [storage.md](storage.md)
