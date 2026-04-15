# Resource Templates

Resource templates are the canonical scaffolding mechanism for new implemented resources.

## Current CLI Surface

```bash
vrooli resource template list
vrooli resource template show <template>
vrooli resource template validate
vrooli resource template generate <template> --name <name>
vrooli resource template generate --from-blueprint <name>
```

## Current Rule

New resource work should start from:

1. blueprint
2. template
3. implementation

It should not start from copying an old `resources/<name>/` directory unless you are explicitly working in transitional cleanup.

## Canonical Template Kinds

- `docker-service`
- `compose-service`
- `external-cli`
- `cloud-api`
- `desktop-app`
- `manual-resource`

Migration-only adapter pattern:

- `legacy-adapter`

`legacy-adapter` is transitional and should not become the default pattern or the normal starting point for new resources.

See [architecture.md](architecture.md) for the ownership model and template-kind expectations each scaffold should converge toward.

## Guidance

- improve canonical templates when patterns repeat
- keep template choice aligned with actual integration shape
- validate template manifests before treating generated work as canonical
- treat generated resource code as a starting point that still needs lifecycle, manifest, and validation review
- prefer shared control-plane/runtime packages over resource-local reinvention
- generated resources should start with the standard `cli/main.go` native entrypoint and storage-safe manifest paths
- keep retained shell compatibility isolated and clearly transitional

## Related

- [architecture.md](architecture.md)
- [resource-blueprints.md](resource-blueprints.md)
- [integration-cookbook.md](integration-cookbook.md)
