# React-Vite Template Maintenance

This document is for engineers changing `templates/scenarios/react-vite`
itself. It is excluded from generated scenarios by
`template.json::copyExcludes`; generated scenarios should not carry
template-maintainer instructions.

## Audience Split

- **Template maintainers** edit files under
  `templates/scenarios/react-vite/`.
- **Scenario authors** edit generated scenarios under
  `scenarios/<scenario-id>/`.

When documentation differs by audience, keep the scenario-author path
in generated docs and keep template-only details here.

## Proto Sources

The template keeps proto relocation sources under:

```text
templates/scenarios/react-vite/proto/v1/
```

During `vrooli scenario generate`, `template.json::relocations` moves
that tree to:

```text
packages/proto/schemas/<scenario-id>/
```

Then the relocation post-hook runs `make generate` from
`packages/proto`. Generated scenario docs should therefore tell
scenario authors to edit `packages/proto/schemas/<scenario-id>/...`,
while template-maintainer docs can refer to `proto/v1/...`.

## Template-Only Files

Use `template.json::copyExcludes` for files that should exist only in
the template source tree. This is the preferred mechanism for
maintainer notes, migration scratch docs, and other guidance that would
confuse generated scenario authors.

Current template-only files:

- `docs/internal/TEMPLATE-GENERATION-CONTRACT.md`
- `docs/internal/TEMPLATE-MAINTENANCE.md`

Do not rely on post-generation shell hooks to remove template-only
files. Excluding them during copy is simpler and avoids briefly writing
files that should never be part of a generated scenario.

## Canonical Generated Docs

Avoid repeating mechanical instructions across generated docs. Use
these ownership rules:

- `docs/concepts/ARCHITECTURE.md` explains why the template is shaped
  this way.
- `docs/internal/TESTING.md` owns test patterns.
- `docs/internal/SEAMS.md` owns the seam/interface registry.
- `docs/reference/*.md` describes user-facing API, CLI, and config
  surfaces.
- `README.md` and `docs/QUICKSTART.md` stay short and link to the
  canonical docs.

When a template change alters the domain workflow, keep generated docs
short and generated-scenario-facing. If a template-only scratch guide is
needed, add it to `template.json::copyExcludes`.

## Validation

Before finishing template changes, run targeted checks:

```bash
vrooli scenario template validate
rg "cmd/server|ParseInterspersed|PrintReportJSON|Pass [0-9]" templates/scenarios/react-vite
```

For changes touching Go template runtime behavior, also run the
affected generator tests:

```bash
go test ./internal/cli/scenariohandlers ./internal/cli/scenariocli
```
