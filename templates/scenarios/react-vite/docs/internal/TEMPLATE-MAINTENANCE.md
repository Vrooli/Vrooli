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

Use shallow validation for routine template-source changes. It checks
manifest shape, placeholder substitution, default design copy,
relocations, generated module paths, and generated start-document
presence without running the full scenario suite:

```bash
vrooli scenario template validate --mode shallow --template react-vite
go test ./internal/cli/scenariohandlers ./internal/cli/scenariocli
```

Use deep validation before marking broad or first-run-sensitive template
changes complete. It generates a temporary real scenario, runs post
hooks, invokes test-genie against the generated physical scenario path,
and passes logical placement so repo-relative documentation and
standards checks behave as if the generated scenario lived under
`scenarios/template-validation-react-vite-deep`. Temporary output is
cleaned by default:

```bash
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive
```

Use `--retain-temp` only while debugging a failed deep run. The command
keeps the generated temp workspace and its shared relocation outputs so a
direct `test-genie execute ... --scenario-path ...` rerun can resolve the
same generated proto artifacts. Deep validation writes a marker file to
the retained workspace and reports a cleanup command with the run id.
Preview or clean retained/interrupted runs with:

```bash
vrooli scenario template cleanup --dry-run
vrooli scenario template cleanup --run <run-id>
vrooli cleanup template-validation --older-than 24h
```

Cleanup is marker-backed and skips retained runs unless you target a run
id or pass `--include-retained`. If cleanup removes proto relocation
artifacts, it regenerates `packages/proto` outputs.

For broad template edits, also run the drift search:

```bash
rg "cmd/server|ParseInterspersed|PrintReportJSON|Pass [0-9]" templates/scenarios/react-vite
```
