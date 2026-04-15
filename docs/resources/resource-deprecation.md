# Resource Deprecation

This page describes the current deprecation path for stale resources.

## Purpose

Deprecation removes a resource from the active supported surface without losing short-term recoverability.

If the capability should remain a future candidate instead of becoming deprecated, use blueprint-backed archival instead.

## Current CLI Surface

```bash
vrooli resource deprecate <name>
vrooli resource list-deprecated
vrooli resource restore <name>
vrooli resource archive gc
```

## Guidance

- use deprecation for stale active-surface resources that should leave supported use
- use blueprint archival when you want to preserve future capability without keeping the implementation active
- do not leave deprecated resources described as if they remain part of the normal supported runtime surface

## Related

- [resource-blueprint-archival.md](resource-blueprint-archival.md)
- [resource-blueprints.md](resource-blueprints.md)
