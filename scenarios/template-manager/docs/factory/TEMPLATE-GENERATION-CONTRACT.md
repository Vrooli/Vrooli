# React-Vite Template Generation Contract

This document is for agents and engineers changing
`templates/scenarios/react-vite` itself. Template Manager owns this
factory-audience contract; generated scenarios receive scenario-author
guidance instead.

> Cutover note: command examples use the Template Manager surfaces planned for
> the hard cutover phases. Until those phases land, the equivalent
> `vrooli scenario ...` commands remain the live engine behind the additive
> APIs.

Generated scenarios should receive scenario-author guidance, not
template-maintainer notes. Put generated first-run guidance in
`docs/START-HERE.md`; keep generation mechanics and template-only
contracts here.

When editing any template-owned file, also read the **Binding Contract
vs. Illustrative Example** section in
`scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md`. It is the single most
load-bearing editorial principle for template maintenance: every line
generated scenarios inherit must be unambiguously a binding contract
*or* an illustrative example, and the framing must match. Mis-framing
is the root cause of placeholder-shell-not-replaced and
settings-taken-too-literally bugs we have actually seen downstream.

## Manifest Behavior

The template's `template.json` controls generation:

- `requiredVars` and `optionalVars` define supported generator flags and
  placeholder values.
- `version` identifies the template version recorded in generated
  scenario provenance. It follows semver against the
  *generated-scenario contract* and must be bumped — with a matching
  `CHANGELOG.md` entry — for any change generated scenarios would have
  to react to. See **Versioning and Changelog** in
  `scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md` for the full rule.
- `startDocument` declares the generated scenario's first-read document.
  The generator prints it after creation, and template validation fails
  if the declared file is not present in the generated scenario.
- `orientation` declares the generated scenario's temporary
  initialization checklist. The generator renders it to
  `.vrooli/orientation.json`; `template-manager orient <scenario>` reads
  that file, evaluates generic checks, and removes only declared cleanup
  paths when explicitly finalized.
- `docs` advertises reference documents in `template-manager template
  show`.
- `copyExcludes` keeps template-only files out of generated scenarios.
- `relocations` copies rendered template content outside the scenario
  destination. The current template relocates `proto/` to
  `packages/proto/schemas/{{SCENARIO_ID}}/`.
- `postHooks` are optional generation follow-up commands. They are
  advertised after generation and run only when the user passes
  `--run-hooks`; template deep validation also runs them because it is
  the source of truth for first-run generated scenario health.
- `exampleDomain` declares the illustrative `notes` domain so it can be
  removed mechanically by `template-manager detemplate`. `marker` is the
  domain slug (`notes`); `paths` enumerates the example-only files/dirs
  to delete wholesale (template-relative; the `proto/` entry is resolved
  through the same relocation mapping the generator applied). See the
  **Example Domain Contract** below and the **Example Domain Markers**
  section in `scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md`.

Unsupported manifest fields are ignored by the current Go decoder.
Add a field to `internal/cli/scenariocli.TemplateManifest` before
depending on it.

Generated scenarios also receive durable provenance in
`.vrooli/service.json::generation`. That metadata records the template
id/version, generation timestamp, and selected design kit/adapter. Do
not remove it during orientation finalization; it is the durable link
between a scenario and the template contract that created it.

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

Use `copyExcludes` for files that should never appear in generated scenarios.
The current template-only file is `CHANGELOG.md`; factory-audience contracts
and guides live under `scenarios/template-manager/docs/`.

`CHANGELOG.md` is intentionally excluded so generated scenarios read
the *live* template changelog when an update loop needs to figure out
what migrations apply between the version recorded in their
`.vrooli/service.json::generation.template.version` and the current
template version. Shipping a frozen copy into each scenario would only
go stale.

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
`scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md`.

## Example Domain Contract

The `notes` domain ships as a worked vertical slice. It is intentionally
generated so scenario authors can copy the pattern before deleting it.

When maintaining the example, preserve these properties:

- proto is the canonical wire contract
- template example proto files keep `@template react-vite/example` until a
  generated scenario intentionally adopts or replaces that contract
- API business logic lives in `api/internal/notes`
- Connect handler methods stay thin
- domain schema is owned by the domain folder
- CLI commands use declarative `cliapp.ArgSchema`
- UI strings and selectors stay centralized
- binary uploads remain the deliberate REST multipart exception
- every example-only artifact carries an `EXAMPLE-DOMAIN:<marker>` marker
  (doc-block fence, `exampleDomain.paths` entry, or trailing
  registration-line comment) so `template-manager detemplate` removes the
  product example in one idempotent command without orphan schema, blob,
  CLI, selector, or i18n residue, and the residue gate can verify zero
  markers survive

The marker vocabulary and its three placements are specified in the
**Example Domain Markers** section of
`scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md`. Removal is mechanical and
verifiable, not a manual checklist:

```bash
template-manager detemplate <scenario>          # strip + delete + finalize
template-manager detemplate <scenario> --dry-run # preview, no writes
```

If the mechanical replacement workflow changes, update
`docs/START-HERE.md` in the same change.

## Validation

Before finishing routine template changes, run shallow validation and
the generator tests:

```bash
template-manager template validate --mode shallow --template react-vite
go test ./internal/cli/scenariohandlers ./internal/cli/scenariocli
```

Before marking first-run-sensitive template changes complete, run deep
validation. It generates a temporary real scenario, runs post hooks, and
executes test-genie against the generated physical scenario path. The
validator also passes logical placement so repo-relative docs and
standards checks evaluate the temp scenario as if it lived under
`scenarios/template-validation-react-vite-deep` in the Vrooli repo:

```bash
template-manager template validate --mode deep --template react-vite --test-preset comprehensive --warning-policy report
```

Deep validation defaults to `--warning-policy report`, so a passing
test-genie suite can still surface grouped non-fatal warnings in human
and JSON output. Use `--warning-policy fail` when validating a template
release candidate so unresolved phase warnings block the command. Warning
findings may point to template issues or to Test Genie/standards
classification gaps; standards rule edits require explicit approval.

Use `--retain-temp` for debugging only. Retained deep runs keep the temp
scenario and its generated relocation outputs so the reported test-genie
command can be rerun directly. The validator writes a marker file inside
the temp workspace and reports a run-specific cleanup command. Use the
marker-backed cleanup command after inspection:

```bash
template-manager template cleanup --run <run-id>
```

For stale interrupted non-retained runs, use:

```bash
vrooli cleanup template-validation --older-than 24h
```

For broad template edits, also run the drift search:

```bash
rg "cmd/server|ParseInterspersed|PrintReportJSON|Pass [0-9]" templates/scenarios/react-vite
```
