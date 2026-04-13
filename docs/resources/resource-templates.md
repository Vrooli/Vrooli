# Resource Templates

Resource templates are the Phase 3 mechanism for starting new implemented resources without cloning an old `resources/<name>/` directory.

They sit between blueprints and full implementation work:

1. choose or inspect a blueprint
2. scaffold from an approved canonical template
3. fill in the resource-specific lifecycle details
4. validate before promoting the resource into active maintenance

## Current CLI Surface

```bash
vrooli resource template list
vrooli resource template show docker-service
vrooli resource template validate
vrooli resource template generate docker-service --name demo-db --dry-run
vrooli resource template generate --from-blueprint terraform --dry-run
```

## Phase 3 Closeout State

Phase 3 is complete when template generation is the default supported starting point for new implemented resources, not directory cloning.

The closeout validation bundle is:

```bash
go test ./internal/resources ./cmd/vrooli
go run ./cmd/vrooli resource template validate
go run ./cmd/vrooli resource template list
go run ./cmd/vrooli resource template show docker-service
go run ./cmd/vrooli resource template generate --from-blueprint terraform --dry-run
```

## Canonical Templates

- `docker-service`
  Use for single-container services such as `postgres` or `redis`.
- `compose-service`
  Use for resources that require a multi-service runtime graph.
- `external-cli`
  Use for tools centered on an installed executable and version detection.
- `cloud-api`
  Use for hosted APIs where auth, config, and connectivity checks matter more than local runtime ownership.
- `desktop-app`
  Use for native applications with strong platform-specific install and validation rules.
- `manual-resource`
  Use when Vrooli can document and probe a dependency but should not claim full automation.
- `legacy-adapter`
  Transitional only. Use when an existing shell-backed resource still needs migration or deprecation.

## Template Layout

Each canonical template lives under `templates/resources/<template>/` and includes:

- `template.json`
- `README.md`
- `resource.json`
- `config/defaults.json`
- `config/schema.json`
- `test/smoke.json`
- `test/integration.json`
- `docs/OPERATIONS.md`

Some templates also include archetype-specific files such as `compose.yaml`, credential notes, or migration notes.

## Template Policy

- Start from an approved canonical template.
- Improve the canonical template when a pattern repeats.
- Do not create ad hoc template folders casually.
- Treat `legacy-adapter` as a shrinking backlog mechanism, not a stable end state.

## Blueprint Integration

Blueprints already carry `suggested_template`. Phase 3 makes that field operational:

- `vrooli resource template generate --from-blueprint <name>` uses the blueprint recommendation automatically.
- If you specify both a template and a blueprint, they must agree.
- Blueprint validation now enforces explicit `integration_kind -> suggested_template` recommendation rules so the catalog and template system cannot drift silently.

This keeps blueprint recommendations honest and prevents the template enum from drifting between data and tooling.
