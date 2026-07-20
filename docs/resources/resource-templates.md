# Resource Templates

Resource templates are the canonical scaffolding mechanism for new implemented resources.

## Current CLI Surface

```bash
template-manager resource-template list
template-manager resource-template show <template>
template-manager resource-template validate
template-manager resource-template generate <template> --var RESOURCE_NAME=<name>
template-manager resource-template generate --from-blueprint <name>
```

## Current Rule

New resource work should start from:

1. blueprint
2. template
3. implementation

It should not start from copying an old `path:resources/<name>/` directory unless you are explicitly working in transitional cleanup.

## Canonical Template Kinds

- `docker-service`
- `compose-service`
- `external-cli`
- `native-cli`
- `cloud-api`
- `desktop-app`
- `manual-resource`

Migration-only adapter pattern:

- `legacy-adapter`

`legacy-adapter` is transitional and should not become the default pattern or the normal starting point for new resources.

See [architecture.md](architecture.md) for the ownership model and template-kind expectations each scaffold should converge toward.

`managed-service` is a planned archetype for a Vrooli-managed local process
that should not require Docker. Do not force such a resource into
`docker-service` merely because a native process template is not yet available.

## Guidance

- improve canonical templates when patterns repeat
- keep template choice aligned with actual integration shape
- prefer `cloud-api`, `external-cli`, or a native/managed runtime over Docker
  when they provide the real supported runtime contract
- treat Docker/Compose as explicit host-runtime requirements, not invisible
  defaults for desktop-bundle use cases
- validate template manifests before treating generated work as canonical
- treat generated resource code as a starting point that still needs lifecycle, manifest, and validation review
- prefer shared control-plane/runtime packages over resource-local reinvention
- generated resources should start with the standard native resource CLI
  scaffold: `cli/main.go`, `cli/go.mod`, and Go-native test/configuration
  structure; shell installation entrypoints are transitional only
- generated resource manifests must include an explicit top-level `cli` block
  with command, adapter, artifact, source-build, and distribution metadata
- templates must declare target deployment profiles or explicitly defer them as
  an incomplete scaffold; do not make an unqualified desktop-support claim
- generated templates expose `RESOURCE_CLI_COMMAND` so the installed command name is declared in manifest data instead of inferred
- resource templates should match scenario templates at the manifest/install contract level, not at the app architecture level
- generated resource CLIs remain thin control-plane delegates built on `cliapp.NewResourceApp(...)`, not scenario-style API clients
- `native-cli` is the deliberate exception: it is for repo-owned Go resource binaries whose real operator surface lives in `cli/internal/...`
- mature resources must not require Bash for lifecycle, configuration,
  diagnostics, or tests; retained shell compatibility is isolated, explicitly
  transitional, and has a removal criterion

The target layout, archetype baselines, proposed manifest shape, and required
deployment/readiness evidence are defined in
[deployment-contract.md](deployment-contract.md). Treat its schema examples as
the target design until the manifest schema and templates implement them.

## Related

- [architecture.md](architecture.md)
- [resource-blueprints.md](resource-blueprints.md)
- [integration-cookbook.md](integration-cookbook.md)
