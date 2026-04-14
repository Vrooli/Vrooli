# Resource Blueprints

Resource blueprints preserve capability knowledge without implying that a resource is currently implemented or supported in the active surface.

## What A Blueprint Is

- a structured record under `.vrooli/resources/blueprints/`
- a way to preserve future resource capability
- a CLI-inspectable artifact

## What A Blueprint Is Not

- not an implemented resource
- not evidence that `vrooli resource start <name>` is supported
- not a substitute for actual validation or runtime behavior

## Current CLI Surface

```bash
vrooli resource blueprint list
vrooli resource blueprint info <name>
vrooli resource blueprint search <query>
vrooli resource blueprint validate
```

## When To Use Blueprints

Use a blueprint when:

- the capability is worth preserving
- there is not yet a supported active implementation
- the resource should remain part of future planning without pretending to be live

## Related

- [resource-templates.md](resource-templates.md)
- [resource-blueprint-archival.md](resource-blueprint-archival.md)
