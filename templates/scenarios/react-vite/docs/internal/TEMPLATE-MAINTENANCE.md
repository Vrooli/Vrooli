# React-Vite Template Maintenance

This document is for engineers changing `templates/scenarios/react-vite`
itself. It is excluded from generated scenarios by
`template.json::copyExcludes`; generated scenarios should not carry
template-maintainer instructions.

## Binding Contract vs. Illustrative Example

Every doc, scaffold, and design asset this template ships mixes two
kinds of guidance, and the distinction shapes how scenario authors and
agents downstream interpret the template. When you edit any
template-owned file, always ask: **is this line a binding contract that
generated scenarios must respect, or is it an illustrative example
meant to communicate shape?** Make that classification obvious in the
text — agents over-fit to whichever framing they see.

- **Binding contracts** in a generated scenario: design tokens, color
  roles, status-color semantics, motion rules, accessibility floors,
  responsive transformations, i18n/a11y seams, the proto → API → CLI
  → UI vertical-slice shape, lifecycle/health wiring, the placement
  of business logic in `api/internal/<domain>/`, the durable
  documentation contract in `docs/manifest.json`. These should read
  prescriptively — "must", "use", "do not".
- **Illustrative examples** in a generated scenario: the `notes`
  domain, the placeholder `AppShell` and home page, the bare-minimum
  settings surface, any specific component list inside `DESIGN.md`,
  any sample copy or sample preferences. These should read as
  examples — "for example", "such as", "this is one shape; build
  what your scenario needs".

Failure modes we have actually observed:

- Agents leave the placeholder `AppShell` and home page intact and
  only bolt new components onto them, because no doc told them the
  shell itself is a placeholder.
- Agents implement exactly and only the settings shown as examples in
  `DESIGN.md` (and delete pre-existing settings like locale switching
  that were not listed there), because the design's example controls
  read prescriptively.
- Agents read READMEs in template-author voice ("Use this template
  to…") and act as if they are editing the template rather than
  a generated scenario.

When editing template files, prevent these failure modes proactively:

1. If a paragraph names specific components, settings, pages, or copy,
   hedge it ("for example", "such as", "illustrative") or wrap it in
   an explicit "Example primitives" / "Example surfaces" header.
2. If something in the scaffold is a placeholder, mark it with a
   `PLACEHOLDER` comment at the top of the file, and reference it
   from `docs/START-HERE.md` so the replacement workflow is visible
   in the initialization gates.
3. If a doc currently reads as template-author voice ("Use this
   template…", "When you copy…"), rewrite it as scenario-voice ("This
   scenario…") so it reads naturally inside a generated scenario.
   Template-author guidance belongs in this file or in
   `TEMPLATE-GENERATION-CONTRACT.md`, both of which are excluded from
   generated scenarios by `copyExcludes`.
4. When adding new sections to `DESIGN.md` (or any of the design kits
   under `templates/design/`), put the binding tokens/rules in one
   place and the illustrative component lists in another, clearly
   labeled. Do not interleave.

This principle is the single most common cause of generated-scenario
quality drift. Apply it whenever you touch a template file.

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

- `CHANGELOG.md`
- `docs/internal/TEMPLATE-GENERATION-CONTRACT.md`
- `docs/internal/TEMPLATE-MAINTENANCE.md`

Do not rely on post-generation shell hooks to remove template-only
files. Excluding them during copy is simpler and avoids briefly writing
files that should never be part of a generated scenario.

## Versioning and Changelog

The template advertises its version via `template.json::version`. That
string is stamped into every generated scenario's
`.vrooli/service.json::generation.template.version` and is the anchor
the update loop uses to figure out what migrations a stale scenario
needs. The version is therefore part of the binding contract, not
cosmetic.

**Every template change that affects generated scenarios must bump the
version and add a matching entry to `CHANGELOG.md` in the same
commit.** The two are kept in lockstep so an agent updating an older
scenario can read a contiguous, complete history.

Semver discipline, anchored to the generated-scenario contract:

- **Major** — a generated scenario must *actively change* to stay
  aligned. Examples: file/folder reorganization, removed or renamed
  required documents, transport-protocol changes
  (REST → Connect-RPC), required-tool additions, manifest schema
  changes, orientation step removals.
- **Minor** — new opt-in capabilities, new optional documents, new
  example domains, new skills introduced, expansions to
  `requiredVars`/`optionalVars` with defaults that keep existing
  generated scenarios working without any change.
- **Patch** — typo fixes, doc clarifications, illustrative-example
  rewordings, dependency bumps without API impact.

When in doubt about whether something is breaking, ask: *would an agent
running an "update this older scenario to match the template" loop have
to do anything because of this change?* If yes, it is at least minor;
if yes-and-it-requires-edits-not-just-additions, it is major.

The changelog entry format is documented at the top of `CHANGELOG.md`.
Each entry's **Migration** block is the single most important section —
it must be a concrete, verifiable checklist that points at the skill
that owns each detail (e.g. screaming-architecture-audit,
temporal-flow-audit). The changelog routes; the skills authoritatively
describe.

If you are uncertain whether your change qualifies, default to writing
the entry. An extra patch-level entry costs nothing; a missing breaking
entry silently breaks every future update loop.

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
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive --warning-policy report
```

Deep validation defaults to `--warning-policy report`: test-genie
failures still fail the command, while non-fatal phase warnings are
reported in grouped output and JSON. Use `--warning-policy fail` for a
release-quality gate where Lighthouse, browser-console, dependency, or
standards warnings should block completion. Use `ignore` only for local
debugging when warning visibility is intentionally not part of the
check. Standards warning fixes may require either template changes or
standards/Test Genie classification changes; do not edit standards rules
without approval.

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
