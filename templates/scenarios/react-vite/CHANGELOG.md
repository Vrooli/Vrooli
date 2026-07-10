# React-Vite Template Changelog

This changelog records changes to `templates/scenarios/react-vite`. It is
**not copied into generated scenarios** (see `template.json::copyExcludes`);
generated scenarios record their origin template id + version in
`.vrooli/service.json::generation.template`, and agents updating an older
scenario read this file from the live template tree.

## How to read this file (agents updating an older scenario)

1. Open the target scenario's `.vrooli/service.json` and read
   `generation.template.version` — that is the version it was generated from.
2. Read every entry below with a version **greater than** that one. The
   **Migration** block in each entry is the actionable punch list; the
   other sections explain *why* and link to the skills that own the
   detailed rules.
3. Treat the linked skills as authoritative. The changelog is a router,
   not a substitute for the skill content.
4. After applying migrations, update the scenario's
   `.vrooli/service.json::generation.template.version` to the latest
   template version so the next update loop knows where it now stands.

## Versioning rules (template maintainers)

The template version follows semantic versioning, anchored to the
*generated-scenario contract*:

- **Major** — anything that an existing generated scenario would have to
  *actively change* to stay aligned: file/folder reorganization, removed
  or renamed required documents, transport-protocol changes
  (REST → Connect-RPC), required-tool additions, manifest schema
  changes, orientation step removals.
- **Minor** — new opt-in capabilities, new optional documents, new
  example domains, new skills introduced, expansions to
  `requiredVars`/`optionalVars` with defaults that keep existing
  generated scenarios working.
- **Patch** — typo fixes, doc clarifications, illustrative-example
  rewordings, dependency bumps without API impact.

Every version bump in `template.json::version` **must** ship with a
matching entry below in the same change. Bumping the version without an
entry, or adding an entry without bumping, are both bugs — the update
loop relies on these two being kept in lockstep. See
`scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md` for the full maintenance
contract.

## Entry shape

Each entry has the form:

```markdown
## <version> — <YYYY-MM-DD>

<one-sentence summary of the release theme>

### Breaking
- <change that an existing scenario must react to>

### Added
- <new capability or doc>

### Changed
- <non-breaking refinement>

### Removed
- <something removed from the template>

### Migration (for agents updating older scenarios)
- [ ] <concrete, verifiable step>
- [ ] <... point at the skill that owns the detail>
```

Omit empty sections. Keep migration steps imperative and verifiable —
"convert all domain APIs to Connect-RPC (see screaming-architecture-audit
skill)" beats "modernize the API layer".

---

## 1.6.0 — 2026-07-07

Makes the generated UI baseline inherit the component canon and mobile-safe
experience floors instead of a placeholder shell.

### Added
- Adopted-provenance component primitives under `ui/src/components/ui/` for
  `Button`, `Card`, `DataTable`, `EmptyState`, `Input`, `Select`,
  `StatusBadge`, `BottomNav`, and `SidebarShell`.
- Safe-area and touch-target utility classes expected by the component canon.

### Changed
- `AppShell` now starts from `min-h-dvh`, uses the governed sidebar and fixed
  safe-area bottom navigation, and keeps locale controls out of the top bar.
- Starter dashboard, notes, and settings pages consume governed primitives; the
  notes example list is now a searchable/sortable `DataTable`.
- Generated docs now steer agents toward adopt-not-hand-roll UI growth through
  `react-component-library`.

### Migration (for agents updating older scenarios)
- [ ] Copy the updated shell pattern: `min-h-dvh` root, overflow-safe main,
      fixed safe-area bottom nav, and Settings-owned locale switching.
- [ ] Adopt governed primitives from `react-component-library` for shared UI
      surfaces instead of adding new raw local primitives. Start with
      `Button`, `Card`, `DataTable`, `EmptyState`, `Input`, `Select`,
      `StatusBadge`, `BottomNav`, and `SidebarShell` where applicable.
- [ ] Move any header locale switcher to Settings and verify mobile chrome
      labels remain single-line at 390px width.
- [ ] Update `.vrooli/service.json::generation.template.version` to `1.6.0`.

## 1.5.0 — 2026-07-05

Seeds generated UI scenarios with a first-class experience contract so the
experience phase can validate UX intent from birth.

### Added
- `experience/` with an L0 `scenario-experience-spec/v1` registry and page
  specs for the generated dashboard, notes example, and settings routes.
- `experience-contract` orientation step requiring the generated experience
  registry and dashboard page spec to exist.
- Generated-scenario guidance in `README.md`, `docs/START-HERE.md`, and
  `docs/concepts/UI-ARCHITECTURE.md` for growing the L0 scaffold into
  priorities, claims, bindings, states, and journeys.

### Changed
- `template-manager detemplate <scenario>` now removes the notes page spec and
  prunes its registry entry with the rest of the notes example domain.

### Migration (for agents updating older scenarios)
- [ ] Add an `experience/` folder with `index.json`, `README.md`, and one
      `pages/<page>.json` L0 spec per real UI route.
- [ ] Run `experience-manager spec validate <scenario> --json`; fix any
      route, PRD reference, or registry parity findings before adding claims.
- [ ] If the scenario still carries the notes example, ensure detemplate removes
      `experience/pages/notes.json` and the `notes` registry entry.
- [ ] Update `.vrooli/service.json::generation.template.version` to `1.5.0`.

## 1.4.0 — 2026-07-02

Moves generated scenarios to the current test-genie unit policy profile so
Unit Health can validate discovered API, CLI, and UI surfaces consistently.

### Added
- `.vrooli/testing.json::unit.policy_profile` with required roles for Go API,
  Go CLI, and React/Vite UI surfaces.
- Reference configuration docs for the unit policy profile and monotonic
  customization rule.

### Changed
- `.vrooli/testing.json` now references the test-genie schema location.
- Generated scenario requirements and START-HERE guidance align with the
  policy-profile based unit phase.
- The notes flow verifier test and mocks were updated for the current
  development-toolchain validator expectations.

### Removed
- The generated notes replay Go artifact that is now regenerated from the
  checked flow model instead of maintained by hand.

### Migration (for agents updating older scenarios)
- [ ] Replace the old `.vrooli/testing.json::unit.languages` block with
      `unit.policy_profile` from the template, preserving only stricter
      scenario-specific additions or explicit waivers.
- [ ] Add or verify the required `api`, `cli`, and `ui` test utility roots and
      production-import guard tests named in the profile.
- [ ] Update docs/reference/configuration.md with the unit policy-profile
      section if the scenario carries generated configuration docs.
- [ ] Update `.vrooli/service.json::generation.template.version` to `1.4.0`.

## 1.3.0 — 2026-06-30

Adds baseline PWA install metadata and placeholder icons to generated
React/Vite scenarios.

### Added
- `ui/public/site.webmanifest` with standalone display mode, relative
  `start_url`/`scope`, theme/background colors, and maskable icon
  declarations.
- Generic placeholder PNGs for `apple-icon-180.png`,
  `favicon-196.png`, `manifest-icon-192.maskable.png`, and
  `manifest-icon-512.maskable.png`.
- Mobile install metadata in `ui/index.html`, including manifest,
  Apple touch icon, theme color, and iOS standalone-mode tags.

### Changed
- `README.md` and `docs/START-HERE.md` now call out the seeded PWA
  branding surface as durable scaffolding that should be replaced with
  scenario-specific branding when available.
- PWA/native-readiness baseline uses relative install asset and
  service-worker cache URLs for proxy/tunnel deployments, and the
  translucent iOS status-bar metadata now has matching viewport-fit and
  safe-area CSS tokens.

### Migration (for agents updating older scenarios)
- [ ] Add a web app manifest under `ui/public/site.webmanifest` using
      relative `start_url` and `scope` values so root tunnels and proxied
      paths both resolve correctly.
- [ ] Add PNG app icons for Apple touch, favicon, and 192/512 maskable
      manifest icons. Use scenario-specific artwork when available;
      otherwise use a neutral placeholder and document that Brand Manager
      should replace it later.
- [ ] Add the manifest, icon, theme-color, and iOS standalone meta tags
      to `ui/index.html`, then rebuild the UI bundle.
- [ ] If `apple-mobile-web-app-status-bar-style` is `black-translucent`,
      include `viewport-fit=cover` and real `env(safe-area-inset-*)`
      usage in UI CSS.
- [ ] Update `.vrooli/service.json::generation.template.version` to
      `1.3.0`.

## 1.2.0 — 2026-06-17

Makes the `notes` example domain mechanically removable and its removal
machine-verifiable: one marker vocabulary, one command, one residue gate —
replacing the manual deletion checklist.

### Added
- `template.json::exampleDomain` (`marker`, `paths`, `jsonPrune`) declares
  the example domain so it can be removed wholesale. New
  `template-manager detemplate <scenario>` strips fenced `EXAMPLE-DOMAIN`
  blocks (docs **and** code), deletes the listed files/dirs (including the
  relocated proto schema + generated artifacts), prunes the comment-less
  JSON (i18n locales, CLI manifest), runs finalizers, and refuses if a
  non-example file still imports a to-be-deleted package. `--dry-run` and
  proto-typed `--json` supported; idempotent.
- `example-domain-removed` orientation step gained a `text_absent_tree`
  check that fails if any `EXAMPLE-DOMAIN` marker survives anywhere in the
  tree.
- The marker vocabulary (fenced block / trailing line marker / manifest
  `paths` / manifest `jsonPrune`) is specified in
  `scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md` and
  `scenarios/template-manager/docs/factory/TEMPLATE-GENERATION-CONTRACT.md`.

### Changed
- Every generated doc now concentrates its `notes` content in one fenced
  `EXAMPLE-DOMAIN:notes` block with a binding zone that describes the real
  `health` domain plus abstract guidance, instead of interleaving `notes`
  as product scope. Example-only code registration lines/blocks carry
  `EXAMPLE-DOMAIN:notes` markers.
- `docs/START-HERE.md` Gate 7 replaces the manual 13-step deletion
  checklist with `template-manager detemplate <scenario>` + the orient gate.

### Migration (for agents updating older scenarios)
- [ ] If the scenario still carries the `notes` example, run
      `template-manager detemplate <scenario>` (preview with `--dry-run`
      first), then `make test` and `template-manager orient <scenario>`.
- [ ] If the scenario already removed `notes` manually, no action — the
      new gate passes when no `EXAMPLE-DOMAIN` marker is present.

## 1.1.0 — 2026-06-12

Makes scenario UI dependency isolation structural instead of
flag-discipline-dependent, and trues up the template's own lockfile.

### Added
- `ui/pnpm-workspace.yaml` — a **workspace boundary file**. pnpm resolves
  its workspace by walking up the directory tree to the first
  `pnpm-workspace.yaml`; without a boundary, a plain `pnpm install` run in
  a scenario's `ui/` silently joins the repo-root `packages/*` workspace
  (wrong scope, ignored overrides, stray root lockfile). The boundary file
  stops the walk, so plain `pnpm install` is always scoped to the UI.
  `--ignore-workspace` keeps working unchanged and remains in the
  lifecycle commands; the boundary removes the failure mode when the flag
  is forgotten.

### Changed
- `ui/pnpm-lock.yaml` regenerated so it matches `ui/package.json` again
  (added `react-router-dom` and `@types/node`, which had drifted in
  without a lockfile update). `pnpm install --frozen-lockfile` now passes
  at generated-scenario depth. Note the template UI is **not installable
  in place** under `templates/` — its `file:../../../packages/*`
  references resolve relative to the *generated* location
  (`scenarios/<id>/ui`); refresh the lockfile from a generated scenario
  (or a temporary copy at `scenarios/<tmp>/ui` depth) and copy it back.
- `docs/guides/troubleshooting.md` pnpm section updated for the boundary
  file.

### Migration (for agents updating older scenarios)
- [ ] Copy `ui/pnpm-workspace.yaml` from the template into the scenario's
      `ui/` directory (content is scenario-agnostic; keep the comment).
- [ ] Verify isolation: from the scenario's `ui/`, run
      `corepack pnpm install --frozen-lockfile` **without**
      `--ignore-workspace`; it must scope to the UI only (no
      "Scope: N workspace projects" banner) and create no
      `pnpm-lock.yaml` at the repo root.
- [ ] Update `.vrooli/service.json::generation.template.version` to
      `1.1.0`.

## 1.0.1 — 2026-05-14

Adds the first PWA install scaffold and placeholder app icons so generated UI
scenarios have a valid web app metadata surface from the start.

### Added
- `ui/public/site.webmanifest` plus placeholder Apple, favicon, and maskable
  manifest icon PNGs.
- Mobile install metadata in `ui/index.html`.
- START-HERE and README guidance telling scenario authors to keep the seeded
  PWA files valid and replace generic icons once product branding exists.

### Changed
- `template.json::version` bumped from `1.0.0` to `1.0.1`.

### Migration (for agents updating older scenarios)
- [ ] Add `ui/public/site.webmanifest`, Apple touch icon, favicon, and maskable
      manifest icons from the template or replace them with scenario-specific
      branded assets.
- [ ] Add the generated mobile install metadata to `ui/index.html`.
- [ ] Record the PWA install surface in the scenario's README or START-HERE
      branding guidance.
- [ ] Update `.vrooli/service.json::generation.template.version` to `1.0.1`.

## 1.0.0 — 2026-05-12

First versioned release. Establishes the template's contract with
generated scenarios and the changelog discipline that future updates
follow. Earlier 0.x revisions are not itemized — treat any scenario
whose `generation.template.version` starts with `0.` as needing the full
1.0.0 migration below.

### Breaking
- **Transport is proto + Connect-RPC, not REST/JSON.** Every domain
  defines its wire contract in `packages/proto/schemas/<scenario-id>/`
  and adopts the generated Connect handlers in API, the generated
  clients in UI, and the generated types in CLI. Binary upload remains
  the deliberate REST multipart exception. Drift between API/UI/CLI is
  the failure mode this collapses.
- **Screaming architecture by domain.** Business logic lives in
  `api/internal/<domain>/`; UI lives under `ui/src/features/<domain>/`;
  CLI commands live under `cli/domains/<domain>/`. Cross-domain shared
  code goes through explicit seams in `docs/internal/SEAMS.md`, not
  ambient utility folders.
- **Documentation manifest is authoritative.** `docs/manifest.json`
  declares every required and scenario-specific document and is the
  single source of truth for the UI docs navigation. New documentation
  must be registered in the manifest in the same change.
- **Temporal flows are formally verified.** Anything with a temporal
  flow uses the reference shape under `api/internal/notes/flow/` (or
  equivalent in a real domain): Quint model, generated artifacts, and
  the flow-verifier scenario for conformance checks.
- **Onboarding via START-HERE + orientation.** First-session work
  follows `docs/START-HERE.md` and the gated orientation steps declared
  in `template.json::orientation`. Existing scenarios without these
  files must add them.

### Added
- `CHANGELOG.md` (this file) and the version-bump discipline documented
  in `scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md`.
- `docs/START-HERE.md` and the orientation step list as the canonical
  first-session contract.
- `docs/manifest.json` schema and required-doc list covering README,
  QUICKSTART, ARCHITECTURE, DOMAINS, SEAMS, TESTING, PROGRESS, PROBLEMS,
  and reference docs.
- Reference `notes` domain implementing the full proto → API → CLI → UI
  vertical slice plus a temporal-flow example.
- Proto relocation under `packages/proto/schemas/<scenario-id>/` with
  `make generate` post-hook.
- `flow-verifier` scenario CLI as the canonical temporal-flow
  toolchain. Generated scenarios shell out to
  `flow-verifier verify check` (see `make temporal-models` and the
  `flow-verify` test command); host-level Quint and Java requirements
  live in `flow-verifier`'s own `hostTools`, not in generated
  scenarios.

### Changed
- CLI structure now uses declarative `cliapp.ArgSchema` per domain.
- UI strings and selectors are centralized for i18n/a11y testability.
- `template.json::version` bumped from 0.1.0 to 1.0.0 to mark the
  versioning system going live.

### Migration (for agents updating older scenarios)
- [ ] Read the scenario's `.vrooli/service.json::generation.template`
      block; record the current version for the audit trail.
- [ ] Define every domain's wire contract under
      `packages/proto/schemas/<scenario-id>/`; remove REST/JSON
      handlers in favor of Connect-RPC. See the proto-codegen + Connect
      adoption patterns in the reference `notes` domain.
- [ ] Reorganize sources by domain: `api/internal/<domain>/`,
      `ui/src/features/<domain>/`, `cli/domains/<domain>/`. Run the
      screaming-architecture-audit skill to verify.
- [ ] Add `docs/START-HERE.md`, `docs/manifest.json`, and every
      required document the manifest declares. Register any
      scenario-specific docs in the manifest.
- [ ] For any temporal flow in the scenario, add a `flow.json`
      contract and generated conformance artifacts under
      `<domain>/flow/`, then verify via
      `flow-verifier verify check --root .` (the flow-verifier
      scenario owns the Quint/Java toolchain). Run the
      temporal-flow-audit skill.
- [ ] Add `docs/internal/SEAMS.md` documenting all cross-domain seams;
      add `docs/internal/PROGRESS.md` and `docs/internal/PROBLEMS.md`.
- [ ] Update `.vrooli/service.json::generation.template.version` to
      `1.0.0` once the migration is green
      (`make test`, shallow + deep template validation pass).
