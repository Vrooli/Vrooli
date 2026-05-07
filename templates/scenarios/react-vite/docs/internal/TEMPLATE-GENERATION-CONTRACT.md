# React-Vite Template Generation Contract

This document is for agents and engineers changing
`templates/scenarios/react-vite` itself. It is excluded from generated
scenarios by `template.json::copyExcludes`.

Generated scenarios should receive scenario-author guidance, not
template-maintainer notes. Put generated first-run guidance in
`docs/START-HERE.md`; keep generation mechanics and template-only
contracts here.

## Manifest Behavior

The template's `template.json` controls generation:

- `requiredVars` and `optionalVars` define supported generator flags and
  placeholder values.
- `startDocument` declares the generated scenario's first-read document.
  The generator prints it after creation, and template validation fails
  if the declared file is not present in the generated scenario.
- `docs` advertises reference documents in `vrooli scenario template
  show`.
- `copyExcludes` keeps template-only files out of generated scenarios.
- `relocations` copies rendered template content outside the scenario
  destination. The current template relocates `proto/` to
  `packages/proto/schemas/{{SCENARIO_ID}}/`.
- `postHooks` are optional generation follow-up commands. They are
  advertised after generation and run only when the user passes
  `--run-hooks`; template deep validation also runs them because it is
  the source of truth for first-run generated scenario health.

Unsupported manifest fields are ignored by the current Go decoder.
Add a field to `internal/cli/scenariocli.TemplateManifest` before
depending on it.

## Placeholder Contract

The generator substitutes `{{NAME}}` placeholders in text file content
and path components. Core values include:

- `SCENARIO_ID`
- `SCENARIO_ID_SNAKE`
- `SCENARIO_DISPLAY_NAME`
- `SCENARIO_DESCRIPTION`
- `SCENARIO_CATEGORY`
- `AUTHOR`
- `DATE`
- `CURRENT_DATE`
- `RANDOM_TOKEN`
- `REPO_ROOT_REL_FROM_API`
- `REPO_ROOT_REL_FROM_CLI`
- `PACKAGES_REL_FROM_API`
- `PACKAGES_REL_FROM_CLI`

Keep placeholders uppercase snake case. Template validation fails if a
generated text file or path still contains an unresolved placeholder.

## Relocated Proto Sources

The template source keeps proto files under:

```text
templates/scenarios/react-vite/proto/
```

Generated scenarios do not receive a local `proto/` directory. During
generation, `template.json::relocations` renders that tree to:

```text
packages/proto/schemas/<scenario-id>/
```

The relocation post-hook then runs `make generate` from
`packages/proto`. Generated docs should therefore tell scenario authors
to edit `packages/proto/schemas/<scenario-id>/...`.

## Template-Only Files

Use `copyExcludes` for files that should never appear in generated
scenarios. Current template-only files:

- `docs/internal/TEMPLATE-GENERATION-CONTRACT.md`
- `docs/internal/TEMPLATE-MAINTENANCE.md`

Do not rely on post-generation cleanup for template-only content. The
copy step should never write those files into the generated scenario.

## Generated Scenario Guidance

`docs/START-HERE.md` is the generated scenario's primary onboarding
document. It owns the first-session workflow and the scenario-author
instructions for replacing the example `notes` domain.

Keep `START-HERE.md` practical and generated-scenario-facing:

- initial validation commands
- PRD and requirements setup
- domain discovery and first vertical slice
- design-language setup
- resource/dependency decisions
- notes replacement and cleanup

Keep template-maintainer explanations here or in
`docs/internal/TEMPLATE-MAINTENANCE.md`.

## Example Domain Contract

The `notes` domain ships as a worked vertical slice. It is intentionally
generated so scenario authors can copy the pattern before deleting it.

When maintaining the example, preserve these properties:

- proto is the canonical wire contract
- API business logic lives in `api/internal/notes`
- Connect handler methods stay thin
- domain schema is owned by the domain folder
- CLI commands use declarative `cliapp.ArgSchema`
- UI strings and selectors stay centralized
- binary uploads remain the deliberate REST multipart exception
- deleting the notes folders plus small registration lines removes the
  product example without orphan schema, blob, CLI, selector, or i18n
  residue

If the mechanical replacement workflow changes, update
`docs/START-HERE.md` in the same change.

## Validation

Before finishing routine template changes, run shallow validation and
the generator tests:

```bash
vrooli scenario template validate --mode shallow --template react-vite
go test ./internal/cli/scenariohandlers ./internal/cli/scenariocli
```

Before marking first-run-sensitive template changes complete, run deep
validation. It generates a temporary real scenario, runs post hooks, and
executes test-genie against the generated physical scenario path. The
validator also passes logical placement so repo-relative docs and
standards checks evaluate the temp scenario as if it lived under
`scenarios/template-validation-react-vite-deep` in the Vrooli repo:

```bash
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive
```

Use `--retain-temp` for debugging only. Retained deep runs keep the temp
scenario and its generated relocation outputs so the reported test-genie
command can be rerun directly. The validator writes a marker file inside
the temp workspace and reports a run-specific cleanup command. Use the
marker-backed cleanup command after inspection:

```bash
vrooli scenario template cleanup --run <run-id>
```

For stale interrupted non-retained runs, use:

```bash
vrooli cleanup template-validation --older-than 24h
```

For broad template edits, also run the drift search:

```bash
rg "cmd/server|ParseInterspersed|PrintReportJSON|Pass [0-9]" templates/scenarios/react-vite
```
