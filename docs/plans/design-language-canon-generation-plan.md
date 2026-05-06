# Design Language Canon + Scenario Generation Implementation Plan

## Purpose

Implement Vrooli's canonical design-language system for new scenario generation. The target is a greenfield hard cut to `DESIGN.md` as the only supported design-definition contract for generated scenarios, backed by project-level governance docs, reusable design kits, stack-specific adapters, generation-time selection, and validation.

This plan exists so a future implementation agent can execute without needing this conversation history.

## Required Reading

Run these skills before implementation:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Recommended extra skills for UI-facing portions:

```bash
prompt-manager skill read react-coherence ux documentation-health
```

Read these local references:

- path:scenarios/swarm-manager/initiatives/design-language-foundation/initiative.json
- path:scenarios/swarm-manager/initiatives/design-language-foundation/orchestration-summary.md
- path:templates/scenarios/react-vite/template.json
- path:templates/scenarios/react-vite/docs/START-HERE.md
- path:templates/scenarios/react-vite/README.md
- path:templates/scenarios/react-vite/docs/manifest.json
- path:templates/scenarios/react-vite/ui/src/styles.css
- path:templates/scenarios/react-vite/ui/tailwind.config.ts
- path:templates/scenarios/react-vite/ui/src/components/AppShell.tsx
- path:internal/cli/scenariocli/template_contracts.go
- path:internal/cli/scenariocli/template_parsers.go
- path:internal/cli/scenariocli/help.go
- path:internal/cli/scenariohandlers/template_runtime.go
- path:internal/cli/scenariohandlers/template_runtime_test.go
- path:internal/cli/scenariocli/scenariocli_test.go
- path:scenarios/brand-manager/README.md
- path:scenarios/brand-manager/api/handlers/design_language.go
- path:scenarios/react-component-library/UI_IMPLEMENTATION.md

Read these external references and verify the current state before depending on them:

- https://blog.google/innovation-and-ai/models-and-research/google-labs/stitch-design-md/
- https://github.com/google-labs-code/design.md

## Problem Statement

Vrooli currently has no canonical project-wide UI design language contract for generated scenarios. The `design-language-foundation` initiative identifies the same drift pattern that already happened with voice canon: agents may produce individually plausible UI, but without a shared enforced canon, scenarios drift visually and structurally.

Existing related material is incomplete or out of date:

- `docs/marketing/` contains external brand and image guidance, not application UI design-system governance.
- `templates/scenarios/react-vite` has Tailwind, primitive UI components, and shell styling, but no canonical design contract for agents to follow.
- `scenarios/brand-manager` references `docs/DESIGN_LANGUAGE.md`; that is obsolete for this direction and must be cut over, not preserved as an alternate standard.
- `scenarios/react-component-library` is an implementation arm candidate, but it is not yet driven by a canonical `DESIGN.md`.
- Scenario generation cannot list design languages, select one, copy its canonical document, apply stack-specific style assets, or validate conformance.

The result is that each new scenario starts with avoidable design ambiguity. The highest-leverage fix is to define design intent before feature UI work starts and make scenario generation install that intent automatically.

## Scope

In scope:

- Create a project-level design governance area under `docs/design/`.
- Create reusable design kits under `templates/design/`.
- Establish `DESIGN.md` as the canonical stack-agnostic design contract.
- Support optional stack-specific adapters under each design kit.
- Extend scenario template manifests so templates declare the adapter they can consume and the default design kit they should use.
- Extend scenario generation so agents can pass `--design <kit-id>`, use a template default, or explicitly opt out only where allowed.
- Copy `DESIGN.md` to the generated scenario root as `scenarios/<scenario-id>/DESIGN.md`.
- Copy adapter assets into template-specific paths when a compatible adapter exists.
- Add CLI surfaces for listing, showing, and validating design kits.
- Update `react-vite` template docs and UI touchpoints so agents know to consult `DESIGN.md` before UI work.
- Add deterministic validation for design-kit structure, metadata, adapter compatibility, copied files, and unresolved placeholders.
- Plan the Brand Manager and React Component Library alignment as part of the hard cut to `DESIGN.md`.

Out of scope for the first implementation pass:

- Migrating every existing scenario to `DESIGN.md`.
- Installing or vendoring the Google `design.md` CLI without explicit dependency approval.
- Building a full LLM-judged design-linter before the deterministic contract is stable.
- Making Tailwind, React, or any specific UI framework universal.
- Preserving `docs/DESIGN_LANGUAGE.md` as a compatibility alias for new scenario generation.

## Current Technical Context

Scenario template generation is centralized enough for a clean implementation:

- `TemplateManifest` in path:internal/cli/scenariocli/template_contracts.go owns template metadata.
- `GenerateOptions` currently carries destination, force, dry-run, hooks, and placeholder values.
- `ParseGenerateArgs` in path:internal/cli/scenariocli/template_parsers.go is the right seam for a `--design` flag.
- `runGenerate` in path:internal/cli/scenariohandlers/template_runtime.go builds values, preflights output, copies template files, runs relocations, verifies unresolved placeholders, validates the generated scenario, and optionally runs hooks.
- `runTemplateValidate` already performs a temp generation pass for every template and is the right place to validate default design-kit integration.
- `RenderTemplateShowResponse` can surface design defaults and adapter requirements.

There is no existing `docs/design/` folder. That is acceptable and justified because the existing project docs taxonomy has no equivalent home for UI design-language governance. `docs/marketing/` is external brand/voice/channel canon; design-language governance needs its own docs area because it defines application UI contracts, generation behavior, adapters, linting, and ownership.

There is no existing `templates/design/` folder. That is also justified because design kits are scaffoldable assets, not project governance docs. They should live beside `templates/scenarios/`, but not inside a single scenario template, because the same design language must be reusable across React, Rust, Python, desktop, mobile, and future stacks.

The `react-vite` template currently has no `DESIGN.md` reference in:

- path:templates/scenarios/react-vite/template.json
- path:templates/scenarios/react-vite/README.md
- path:templates/scenarios/react-vite/docs/manifest.json
- path:templates/scenarios/react-vite/docs/START-HERE.md
- path:templates/scenarios/react-vite/ui/src/styles.css
- path:templates/scenarios/react-vite/ui/tailwind.config.ts
- path:templates/scenarios/react-vite/ui/src/components/AppShell.tsx

Those are the initial template touchpoints to update once the design system exists.

## Target End State

Project governance:

```text
docs/design/
  README.md
  registry.md
  adapters.md
  governance.md
```

Design-kit assets:

```text
templates/design/
  vrooli-default/
    metadata.json
    DESIGN.md
    adapters/
      react-vite-tailwind/
        adapter.json
        tokens.css
        tailwind.theme.json
        README.md
```

Future kits follow the same shape:

```text
templates/design/
  <kit-id>/
    metadata.json
    DESIGN.md
    adapters/
      <adapter-id>/
        adapter.json
        ...
```

Generated scenarios receive:

```text
scenarios/<scenario-id>/
  DESIGN.md
  README.md
  docs/
    START-HERE.md
    ...
  ui/
    src/design-tokens.css
    tailwind.theme.json
    ...
```

The canonical scenario-level design file is the scenario root `DESIGN.md`, not `docs/DESIGN.md`. This keeps it at the same contract level as `README.md`, `AGENTS.md`, `PRD.md`, and other agent-facing scenario entry points. Scenario docs should reference it, but the root file is the source of truth.

## Implementation Strategy

### Phase 1: Define the Design Docs Contract

Add `docs/design/` with:

- `README.md`: explains the design-language system, why it exists, and the relationship between project governance, design kits, adapters, generated scenario `DESIGN.md`, Brand Manager, and React Component Library.
- `registry.md`: lists available design kits and their intended use. This can start as Markdown, but should mirror `metadata.json` content so it can later be generated.
- `adapters.md`: defines adapter IDs as stack-specific realizations, not language folders. Examples: `react-vite-tailwind`, `react-vite-css`, `rust-egui`, `python-pywebview-css`, `swiftui`.
- `governance.md`: defines ownership, review expectations, versioning, when a kit change is breaking, and how generated scenarios should handle local modifications.

Update:

- path:docs/README.md
- path:docs/manifest.json

The docs should explicitly state that `docs/DESIGN_LANGUAGE.md` is obsolete and that new scenario work must use root-level `DESIGN.md`.

### Phase 2: Add the First Design Kit

Create `templates/design/vrooli-default/` with one canonical `DESIGN.md` and one initial adapter for `react-vite-tailwind`.

The kit metadata should be stack-neutral:

```json
{
  "id": "vrooli-default",
  "name": "Vrooli Default",
  "version": "0.1.0",
  "default": true,
  "description": "Default Vrooli product UI language for professional scenario applications.",
  "tags": ["vrooli", "application", "dashboard", "professional"],
  "adapters": {
    "react-vite-tailwind": {
      "path": "adapters/react-vite-tailwind",
      "supports": ["templates/scenarios/react-vite"]
    }
  }
}
```

The canonical `DESIGN.md` should follow the Google design.md direction where practical: YAML front matter for tokens and structured constraints, followed by Markdown rationale. Because the external spec is alpha, keep Vrooli's validator deterministic and local. Do not introduce a new external dependency without approval.

The adapter should contain concrete assets for `react-vite`:

- `adapter.json`: declares copy operations, required template adapter ID, and validation expectations.
- `tokens.css`: CSS custom properties generated from or consistent with `DESIGN.md`.
- `tailwind.theme.json`: Tailwind-compatible theme extension data.
- `README.md`: explains what the adapter owns and what scenario authors may customize.

### Phase 3: Add Design-Kit Contracts to the CLI Layer

Add new structs near the scenario template contracts:

- `DesignKitManifest`
- `DesignAdapterManifest`
- `DesignKitInfo`
- `DesignCopyRule`
- `DesignValidationReport`
- `DesignValidationIssue`

The exact placement can be either a new file under path:internal/cli/scenariocli or a closely named sibling to `template_contracts.go`. Keep contracts in `scenariocli`; keep filesystem/runtime behavior in `scenariohandlers`.

Validation rules:

- `templates/design/<kit-id>/metadata.json` exists.
- Metadata `id` matches folder name.
- Exactly one default kit is allowed unless the CLI command is validating a single kit.
- `DESIGN.md` exists at kit root.
- Each adapter path is relative and stays inside the kit.
- Each declared adapter has `adapter.json`.
- Copy rules stay inside the generated scenario destination.
- Copy rules do not overwrite files outside declared destinations.
- Referenced source files exist.
- No copied text file contains unresolved `{{PLACEHOLDER}}` after rendering.
- Adapter IDs are stack-specific and must match the consuming template's declared adapter.

### Phase 4: Add Design CLI Commands

Add a new command group under `vrooli scenario design`:

```bash
vrooli scenario design list
vrooli scenario design show <kit-id>
vrooli scenario design validate [<kit-id>|--all]
```

Recommended behavior:

- `list`: show ID, name, default marker, version, tags, adapter IDs.
- `show`: show metadata, canonical `DESIGN.md` path, adapter paths, supported templates, and copy rules.
- `validate`: run deterministic validation and exit non-zero on issues.
- Support JSON output using existing CLI output patterns.

Avoid adding a top-level `vrooli design` command initially. Design selection is currently scenario-generation behavior, so `vrooli scenario design` keeps the surface discoverable near `scenario template` and `scenario generate`.

### Phase 5: Extend Template Manifests and Generation

Extend `TemplateManifest` with a `Design` block:

```json
{
  "design": {
    "adapter": "react-vite-tailwind",
    "default": "vrooli-default",
    "required": true
  }
}
```

Keep template manifests declarative. The design kit adapter should own file-level copy rules so framework-specific artifacts live near the design kit, not scattered across scenario templates.

Extend generation flags:

```bash
vrooli scenario generate react-vite \
  --id example \
  --display-name "Example" \
  --description "Example scenario" \
  --design vrooli-default
```

Selection rules:

- If `--design <kit-id>` is provided, use that kit.
- If `--design none` is provided, allow it only when the template declares design as optional.
- If no `--design` is provided and the template declares a default, use the default.
- If a required design has no compatible adapter for the template, fail before writing files.
- If an optional design has no compatible adapter, copy only `DESIGN.md` when useful and print a clear warning.
- In dry-run mode, show the selected design kit and adapter copy operations.

Generation order:

1. Load template manifest.
2. Parse generation args, including `--design`.
3. Resolve template values.
4. Resolve design kit and adapter.
5. Preflight scenario destination, relocations, and design copy targets together.
6. Copy template.
7. Copy root `DESIGN.md`.
8. Apply adapter copy rules.
9. Verify unresolved placeholders across the generated scenario.
10. Run relocations.
11. Run generated scenario validation.
12. Run hooks if requested.

Preflighting design copy targets with template and relocation targets matters so generation never partially commits when a design adapter collides with a template file.

### Phase 6: Wire `react-vite` to the Design System

Update path:templates/scenarios/react-vite/template.json:

- Add `design.adapter = "react-vite-tailwind"`.
- Add `design.default = "vrooli-default"`.
- Add `design.required = true`.
- Add docs pointers for the design docs and generated scenario `DESIGN.md`.

Update template docs:

- path:templates/scenarios/react-vite/docs/START-HERE.md: make Design Language a real initialization gate once generation installs `DESIGN.md`.
- path:templates/scenarios/react-vite/README.md: document that new scenarios start from root `DESIGN.md`.
- path:templates/scenarios/react-vite/docs/manifest.json: include `DESIGN.md` as an agent-facing contract document.
- path:templates/scenarios/react-vite/docs/concepts/ARCHITECTURE.md if present or created by template docs: reference design boundaries from architecture, but do not duplicate design rules.

Update UI touchpoints sparingly:

- path:templates/scenarios/react-vite/ui/src/styles.css: import or define generated design tokens and include a short comment pointing to root `DESIGN.md`.
- path:templates/scenarios/react-vite/ui/tailwind.config.ts: consume `tailwind.theme.json` or clearly map CSS variables into Tailwind theme extension.
- path:templates/scenarios/react-vite/ui/src/components/AppShell.tsx: avoid hardcoded visual language where tokens/components should be used.
- path:templates/scenarios/react-vite/ui/src/components/ui/button.tsx and sibling primitives: ensure variants map to design tokens.

Do not add noisy comments throughout the UI. Put references where agents naturally modify global styles, theme config, shell layout, and primitive components.

### Phase 7: Hard Cut Brand Manager to `DESIGN.md`

Treat existing Brand Manager `docs/DESIGN_LANGUAGE.md` references as obsolete.

Required migration work:

- Update path:scenarios/brand-manager/README.md so it names root `DESIGN.md` as the canonical scenario design file.
- Update path:scenarios/brand-manager/api/handlers/design_language.go or replace it with a `DESIGN.md` generator/export path.
- Rename API concepts from "design language markdown" to "DESIGN.md export" where practical.
- Ensure any generated Markdown follows the canonical `DESIGN.md` structure with YAML front matter plus Markdown rationale.
- Do not preserve `docs/DESIGN_LANGUAGE.md` as a new-generation compatibility path.

This can be a later implementation phase if the initial generator work is already large, but the final definition of done requires no current docs to instruct agents to create `docs/DESIGN_LANGUAGE.md`.

### Phase 8: Align React Component Library

Make `scenarios/react-component-library` the implementation arm of `DESIGN.md`, consistent with the swarm-manager initiative.

Work items:

- Update its docs to state that component primitives implement selected design kits, starting with `vrooli-default`.
- Add or plan package/API boundaries that consume tokens from `DESIGN.md` or generated adapter assets.
- Ensure component examples and exported primitives do not become an independent design canon.
- Add a future path for validating component variants against design-token names and accessibility constraints.

### Phase 9: Add Generation-Time Guardrails

After deterministic design-kit validation and generation integration are stable, add design conformance guardrails.

First layer:

- Deterministic `design validate` checks.
- Scenario generated-file checks.
- Required-file checks in template validation.
- Accessibility checks that can be locally computed, such as token contrast pairs declared in `DESIGN.md`.

Second layer:

- A `design-linter` scenario or command that evaluates generated UI against the selected `DESIGN.md`.
- LLM-judged conformance should be introduced only after deterministic rules define what must be blocked versus warned.
- Keep this as the second instance of the broader agent-generation-guardrails substrate; do not extract a generic guardrails scenario until a third mature use case exists, matching the swarm-manager initiative guidance.

## Contract Decisions

- `DESIGN.md` is the canonical design contract for new generated scenarios.
- `docs/DESIGN_LANGUAGE.md` is obsolete and must not be supported as a parallel standard for new scenario generation.
- Project-level governance lives in `docs/design/`.
- Copyable design-kit assets live in `templates/design/`.
- A design kit is design intent; an adapter is a stack-specific realization; a scenario template is a consumer.
- Each design kit has one canonical root `DESIGN.md`.
- Adapters do not get their own duplicated `DESIGN.md`.
- Adapter IDs are stack-level, not language-level.
- Generated scenarios receive `DESIGN.md` at the scenario root.
- Template manifests declare the adapter they consume and whether design is required.
- Design adapter copy rules live with the design kit, not inside each scenario template.
- External `design.md` tooling may inform the format, but Vrooli validation must work offline and without unapproved dependencies.
- Use greenfield contracts. Do not add compatibility shims unless a concrete existing production scenario needs one and the user explicitly approves it.

## Testing Plan

Unit tests:

```bash
go test ./internal/cli/scenariocli ./internal/cli/scenariohandlers
```

Template and design validation:

```bash
vrooli scenario design validate --all
vrooli scenario template validate
```

CLI behavior tests should cover:

- `scenario design list` human output.
- `scenario design list --json`.
- `scenario design show vrooli-default`.
- `scenario design validate --all`.
- `scenario generate react-vite --design vrooli-default --dry-run`.
- Unknown design kit fails before writing.
- Required design with missing adapter fails before writing.
- Optional design with no adapter behaves according to the documented fallback.
- `--design none` fails for `react-vite` while design is required.
- Template validation exercises the default design kit.

Generated scenario smoke test:

```bash
vrooli scenario generate react-vite \
  --id design-smoke \
  --display-name "Design Smoke" \
  --description "Scenario generation smoke test for DESIGN.md integration" \
  --design vrooli-default \
  --dry-run
```

Then run a real throwaway generation when implementation is ready:

```bash
vrooli scenario generate react-vite \
  --id design-smoke \
  --display-name "Design Smoke" \
  --description "Scenario generation smoke test for DESIGN.md integration" \
  --design vrooli-default
```

Validate generated files:

- `scenarios/design-smoke/DESIGN.md` exists.
- Adapter assets exist in the expected UI paths.
- `README.md`, `docs/START-HERE.md`, and `docs/manifest.json` reference root `DESIGN.md`.
- No generated file contains unresolved `{{PLACEHOLDER}}`.
- `ui/src/styles.css` and `ui/tailwind.config.ts` consume adapter assets.
- `make test` works from `scenarios/design-smoke` if generation hooks completed successfully.

Clean up throwaway scenarios using the repo's proto cleanup warning from path:AGENTS.md. For a scenario ID with hyphens, remember proto output paths use both hyphenated and underscored names:

```bash
rm -rf scenarios/design-smoke
rm -rf packages/proto/schemas/design-smoke
rm -rf packages/proto/gen/go/design-smoke
rm -rf packages/proto/gen/typescript/js/design-smoke
rm -rf packages/proto/gen/python/design_smoke
( cd packages/proto && make generate )
```

Documentation validation:

- `docs/manifest.json` includes the new `docs/design/` documents.
- `templates/scenarios/react-vite/docs/manifest.json` includes the generated root `DESIGN.md` contract.
- No current docs instruct new scenario agents to create or use `docs/DESIGN_LANGUAGE.md`.

## Rollout / Validation Checklist

- [ ] `docs/design/` exists and is linked from project docs.
- [ ] `templates/design/vrooli-default/DESIGN.md` exists.
- [ ] `templates/design/vrooli-default/metadata.json` validates.
- [ ] `templates/design/vrooli-default/adapters/react-vite-tailwind/adapter.json` validates.
- [ ] `vrooli scenario design list` works.
- [ ] `vrooli scenario design show vrooli-default` works.
- [ ] `vrooli scenario design validate --all` works and fails on intentionally broken fixtures.
- [ ] `TemplateManifest` supports a `design` block.
- [ ] `ParseGenerateArgs` supports `--design`.
- [ ] `RenderTemplateShowResponse` shows design defaults and adapter requirements.
- [ ] `runGenerate` resolves and preflights design copy operations before writing files.
- [ ] `runTemplateValidate` validates template default design integration.
- [ ] `templates/scenarios/react-vite/template.json` requires `vrooli-default` through `react-vite-tailwind`.
- [ ] Generated `react-vite` scenarios include root `DESIGN.md`.
- [ ] Generated `react-vite` UI consumes adapter assets.
- [ ] `vrooli scenario template validate` passes.
- [ ] Relevant Go tests pass.
- [ ] Brand Manager docs no longer present `docs/DESIGN_LANGUAGE.md` as the current standard.
- [ ] React Component Library docs identify `DESIGN.md` as upstream canon.

## Risks + Mitigations

Risk: Google's `design.md` spec is alpha and may change.

Mitigation: Use the direction, not an unpinned runtime dependency. Keep Vrooli's first validator local and deterministic.

Risk: `docs/design/` and `templates/design/` drift.

Mitigation: `docs/design/registry.md` should point to kit metadata, and `vrooli scenario design validate --all` should be the source of truth for scaffoldable assets.

Risk: Adapter folders multiply and become inconsistent.

Mitigation: Require `adapter.json`, explicit copy rules, supported template declarations, and validation. Do not create language-first buckets.

Risk: Design copy operations overwrite template files unexpectedly.

Mitigation: Preflight adapter copy targets together with scenario destination and relocations. Fail before writing unless `--force` explicitly allows replacement.

Risk: Brand Manager keeps producing obsolete `DESIGN_LANGUAGE.md`.

Mitigation: Treat that as migration debt in the same initiative. The final state must emit or manage `DESIGN.md`, not a compatibility alias.

Risk: Design comments in UI become noisy and ignored.

Mitigation: Add references only in high-leverage touchpoints: global styles, theme config, app shell, and primitive components.

Risk: Tailwind assumptions leak into the design canon.

Mitigation: Keep `DESIGN.md` stack-agnostic. Tailwind belongs only in the `react-vite-tailwind` adapter.

## Non-Goals / Prohibited Patterns

- Do not create duplicated `DESIGN.md` files per framework adapter.
- Do not use language-first folders such as `templates/design/react/` or `templates/design/python/`.
- Do not make `docs/DESIGN_LANGUAGE.md` a supported new-generation path.
- Do not bury generated scenario design canon under `docs/`; use root `DESIGN.md`.
- Do not install design tooling dependencies without explicit permission.
- Do not make React Component Library an independent source of design truth.
- Do not use marketing image style docs as the UI design-system canon.
- Do not add broad compatibility shims for speculative old paths.

## Definition of Done

This initiative is complete when:

- Project docs define `DESIGN.md` governance under `docs/design/`.
- At least one design kit, `vrooli-default`, exists under `templates/design/`.
- The design kit has one canonical `DESIGN.md` and a validated `react-vite-tailwind` adapter.
- `vrooli scenario design list/show/validate` are implemented and tested.
- `vrooli scenario generate react-vite --design vrooli-default` generates a scenario with root `DESIGN.md` and adapter assets.
- `react-vite` uses a default required design kit when no `--design` is provided.
- `--design none` is rejected for `react-vite` while design is required.
- `vrooli scenario template validate` validates design integration.
- Brand Manager no longer points new work at `docs/DESIGN_LANGUAGE.md`.
- React Component Library documentation treats `DESIGN.md` as upstream canon.
- All relevant tests and validation commands pass.
