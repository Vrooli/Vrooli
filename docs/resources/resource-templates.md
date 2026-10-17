# Resource Templates

Resource templates are the canonical scaffolding mechanism for new implemented resources.

## Current CLI Surface

```bash
vrooli resource scaffold --name <name> --driver <archetype>
```

## Current Rule

New resource work should start from:

1. choose one of the four supported archetypes
2. generate the scaffold
3. implement the resource-specific behavior

It should not start from copying an old `resources/<name>/` directory unless you are explicitly working in transitional cleanup.

## Supported Archetypes

- `managed-service`
- `external-cli`
- `cloud-api`
- `native-cli`

See [architecture.md](architecture.md) for the ownership model and layout
expectations each scaffold must satisfy.

`managed-service` is the canonical archetype for a Vrooli-managed local
process that should not require Docker. Its generated manifest declares the
signed server artifact, provider policy, and bundled-service delivery profile;
the shared driver owns lifecycle and supervision. Its policy defaults are
target-aware: `control-plane` selects the Vrooli-owned shared host and
`desktop-bundle` selects a private bundled process. Desktop shared reuse is an
explicit, consented override rather than a manifest-wide default.

Its `managed_service` block should also declare acquisition when the service
is distributed from a clean machine:

```json
{
  "managed_service": {
    "artifact": {
      "path": "server/example",
      "version": "1.0.0",
      "layout": "file",
      "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    },
    "acquisition": {
      "kind": "url",
      "targets": [
        {
          "when": { "os": "linux", "arch": "amd64" },
          "url": "https://example.invalid/example-linux-amd64.tar.gz",
          "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
          "archive": "tar.gz",
          "layout": "file",
          "bin_path": "example"
        }
      ]
    }
  }
}
```

Use a directory layout and tree digest when the executable loads sibling
libraries or data. Keep targets ordered from the most specific predicate to a
portable fallback, and declare an explicit unsupported reason for every
claimed platform without a target.

## Guidance

- improve canonical templates when patterns repeat
- keep template choice aligned with actual integration shape
- prefer the archetype whose runtime contract matches the resource
- validate the generated manifest before adding resource-specific behavior
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
the target design until the canonical template implements them.

## Declaring acceleration

No template scaffolds an `acceleration` block, because most resources do no
accelerated work and an unused block invites a copied, unmeasured VRAM
reservation. Each template's `README.md` documents the block with a worked
example instead, and the JSON schema supplies field completion and descriptions
in an editor.

Add one when the resource genuinely runs work on a device:

```json
"acceleration": {
  "backends": ["cuda", "cpu"],
  "require": "preferred",
  "cuda": { "min_compute": "8.9", "env": { "MY_RESOURCE_DEVICE": "cuda" } },
  "cpu":  { "env": { "MY_RESOURCE_DEVICE": "cpu" } }
}
```

Three rules the schema enforces, each of which existed as a real fleet defect:

- **Every backend you name needs a config block.** A backend listed without one
  is a typo, not a declaration.
- **A `vram` claim needs `yield_when_idle` and a `profile` ending at
  `floor_bytes`.** Two resources shipped a reservation the broker could never
  ask them to release.
- **`claim.profile.apply.verb` must be `capacity`.** One broker contract had
  four call signatures across the fleet; the broker had to know which resource
  it was addressing in order to address it.

Measure the rung amounts. An unmeasured `preferred_bytes` reserves capacity
nobody can plan around — one resource reserved 5 GiB of a 16 GiB card while
holding zero device bytes.

## Related

- [architecture.md](architecture.md)
- [resource-blueprints.md](resource-blueprints.md)
- [integration-cookbook.md](integration-cookbook.md)
