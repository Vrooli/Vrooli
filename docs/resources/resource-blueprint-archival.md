# Blueprint Archival

Blueprint archival is the workflow for removing an implementation from the active resource surface while preserving the capability as a blueprint-backed future candidate.

## Current CLI Surface

```bash
vrooli resource archive-to-blueprint <name>
vrooli resource list-blueprint-archived
vrooli resource restore-blueprint <name>
vrooli resource archive gc-blueprints
```

## Use This When

- the implementation should leave the active repo surface
- the capability should remain preserved as a future blueprint-backed candidate

## Do Not Confuse With Deprecation

- blueprint archival preserves future candidate status
- deprecation removes the resource from the active supported surface without treating it as an active future blueprint-backed path by default

## Guidance

- use blueprint archival when future capability preservation matters more than short-term recoverability of the current implementation
- keep archived resources out of normal support claims and day-to-day resource guidance

## Related

- [resource-blueprints.md](resource-blueprints.md)
- [resource-deprecation.md](resource-deprecation.md)
