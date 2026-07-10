# Resource Templates

This directory contains templates for creating consistent and comprehensive resources in the Vrooli platform.

## Available Templates

Phase 3 now ships the canonical implementation templates described in the migration plan:

- `docker-service`
- `compose-service`
- `external-cli`
- `native-cli`
- `cloud-api`
- `desktop-app`
- `manual-resource`

The legacy `PRD.md` file remains available for design documentation, but new implementation scaffolds should come from `template-manager resource-template ...`, not from copying an old resource directory.

## Template Usage

```bash
template-manager resource-template list
template-manager resource-template show docker-service
template-manager resource-template generate docker-service --var RESOURCE_NAME=demo-db
template-manager resource-template generate --from-blueprint terraform --dry-run
```

## Validation

Phase 3 is only considered closed when the template system validates as a complete scaffold path rather than an illustrative sample.

```bash
go test ./internal/resources ./cmd/vrooli
template-manager resource-template validate
template-manager resource-template generate --from-blueprint terraform --dry-run
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

The one explicit exception is `native-cli`: use it when the resource is a repo-owned Go binary with a real operator command surface. In that archetype, `cli/main.go` still stays thin, but `cli/internal/app` and resource-domain packages become first-class rather than optional placeholders.

## Quality Standards

All generated resource scaffolds should:
- ✅ Start from an approved canonical template
- ✅ Produce a valid placeholder `resource.json`
- ✅ Include a complete manifest-driven resource CLI scaffold
- ✅ Include docs plus baseline Go lint/test scaffolding
- ✅ Stay honest about portability and operational limits
