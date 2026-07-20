# Resource Interface Standards

This page describes the current expectations for resource-facing platform surfaces.

## Current Truth

The canonical project-level resource surface is:

```bash
vrooli resource ...
```

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
vrooli resource info <name>
vrooli resource status
vrooli resource validate <name>
vrooli resource logs <name>
vrooli resource restart <name>
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

## Related

- [architecture.md](architecture.md)
- [deployment-contract.md](deployment-contract.md)
- [storage.md](storage.md)
