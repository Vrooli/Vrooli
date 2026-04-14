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
```

## Guidance

- prefer consistency with the root CLI
- avoid inventing special-case interfaces without strong reason
- keep compatibility shims isolated and documented as transitional when they still exist
