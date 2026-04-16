# Resource Templates

This directory contains templates for creating consistent and comprehensive resources in the Vrooli platform.

## Available Templates

Phase 3 now ships the canonical implementation templates described in the migration plan:

- `docker-service`
- `compose-service`
- `external-cli`
- `cloud-api`
- `desktop-app`
- `manual-resource`

The legacy `PRD.md` file remains available for design documentation, but new implementation scaffolds should come from `vrooli resource template ...`, not from copying an old resource directory.

## Template Usage

```bash
vrooli resource template list
vrooli resource template show docker-service
vrooli resource template generate docker-service --name demo-db
vrooli resource template generate --from-blueprint terraform --dry-run
```

## Validation

Phase 3 is only considered closed when the template system validates as a complete scaffold path rather than an illustrative sample.

```bash
go test ./internal/resources ./cmd/vrooli
go run ./cmd/vrooli resource template validate
go run ./cmd/vrooli resource template generate --from-blueprint terraform --dry-run
```

## Template Philosophy

Resource templates differ from scenario templates in important ways:

- **Resources** = Infrastructure components that enable scenarios
- **Scenarios** = Complete applications with business value

Resource scaffolds focus on:
- Infrastructure capabilities and reliability
- Integration patterns with other resources  
- Operational concerns (deployment, monitoring, scaling)
- Resource-specific management interfaces
- System-level configuration and optimization

That difference does not change the CLI contract. Canonical resource templates should still emit the same manifest-driven CLI surface shape as scenario templates:

- `cli/go.mod`
- `cli/install.sh`
- `cli/install.ps1`
- explicit `resource.json` `cli.install`
- explicit `resource.json` `cli.invoke`
- explicit `resource.json` `cli.freshness`

Resource CLIs remain thinner than scenario CLIs by design. The default bootstrap is `cliapp.NewResourceApp(...)` with delegated lifecycle commands, not `NewStandardScenarioApp(...)`.

## Quality Standards

All generated resource scaffolds should:
- ✅ Start from an approved canonical template
- ✅ Produce a valid placeholder `resource.json`
- ✅ Include a complete manifest-driven resource CLI scaffold
- ✅ Include config, docs, and test stubs
- ✅ Stay honest about portability and operational limits
