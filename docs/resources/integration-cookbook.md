# Resource Integration Cookbook

This page gives practical platform-level guidance for integrating resources into Vrooli.

## Primary Principle

Resources should be integrated as capability providers with clear manifests, lifecycle behavior, and validation expectations. They should not be documented as ad hoc one-off shell islands unless they are explicitly transitional.

## Inspect Existing Resources

```bash
vrooli resource list
vrooli resource info <name>
vrooli resource status
```

Check whether the resource already exists before designing a new one.

## Recommended Path For New Resource Work

1. inspect or create a blueprint
2. choose a canonical template
3. scaffold from the template
4. implement lifecycle and validation behavior
5. validate manifests and schema artifacts

Useful commands:

```bash
vrooli resource blueprint list
vrooli resource blueprint info <name>
vrooli resource template list
vrooli resource template show <template>
vrooli resource template generate <template> --name <name>
vrooli resource template validate
vrooli resource validate <name>
vrooli resource schema validate
vrooli resource schema sync
```

## Guidance

- prefer canonical templates over copying old resource directories
- keep manifests honest about capability and support level
- use blueprint and archive flows for speculative or retired capability instead of pretending everything is active
- move resource-specific implementation detail into the resource once it stops being cross-resource guidance

## Related

- [resource-blueprints.md](resource-blueprints.md)
- [resource-templates.md](resource-templates.md)
- [resource-deprecation.md](resource-deprecation.md)
