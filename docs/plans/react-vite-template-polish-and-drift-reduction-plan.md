# React-Vite Template Polish and Drift Reduction Plan

**Plan file**: `path:docs/plans/react-vite-template-polish-and-drift-reduction-plan.md`  
**Authored**: 2026-05-05  
**Owner**: meta-optimization / toolchain-validator  
**Target**: `path:templates/scenarios/react-vite/`  
**Status**: ready for implementation; no code changes from this plan have been applied yet

---

## 1. Purpose

This plan turns the 2026-05-05 review of `templates/scenarios/react-vite`
into an executable implementation program. The template is already strong:
Connect-RPC is the canonical wire path, the API owns business logic,
schemas are domain-owned, the CLI and UI are thin translation surfaces,
and endpoint metadata is generated from domain modules.

The remaining work is multiplier-sensitive polish:

1. Fix first-run/standalone CI correctness issues.
2. Remove stale docs and confusing lifecycle wording.
3. Add enforcement where the template currently relies on convention.
4. Reduce central coordinated edits for new domains and endpoints.
5. Improve generated-scenario ergonomics around i18n, CLI metadata,
   domain scaffolding, and the attachments reference slice.

The goal is not cosmetic cleanup. The target end state is a template where
new scenarios start green, future agents copy current examples rather than
stale ones, and common domain/endpoint changes require fewer manual files.

---

## 2. Required Reading

Run before executing any phase:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Reference files to read top-to-bottom:

- `path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md` — multiplier-aware audit lens.
- `path:scenarios/prompt-manager/store/teams/meta-optimization/notebook/template-fitness/react-vite-template/2026-05-04/RESULTS.md` — iteration-2 Connect-RPC measurement and remaining issues.
- `path:templates/scenarios/react-vite/README.md`
- `path:templates/scenarios/react-vite/template.json`
- `path:templates/scenarios/react-vite/.github/workflows/test.yml`
- `path:templates/scenarios/react-vite/.vrooli/service.json`
- `path:templates/scenarios/react-vite/docs/concepts/ARCHITECTURE.md`
- `path:templates/scenarios/react-vite/docs/internal/TESTING.md`
- `path:templates/scenarios/react-vite/docs/internal/SEAMS.md`
- `path:templates/scenarios/react-vite/docs/internal/REPLACING-NOTES.md`
- `path:templates/scenarios/react-vite/api/internal/modules/registry.go`
- `path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go`
- `path:templates/scenarios/react-vite/api/cmd/gen-endpoints/cli_commands_seed.json`
- `path:templates/scenarios/react-vite/cli/domains/domains.go`
- `path:templates/scenarios/react-vite/ui/src/i18n/locales.ts`
- `path:templates/scenarios/react-vite/ui/src/i18n/locales/locales.test.ts`
- `path:templates/scenarios/react-vite/ui/src/consts/selectors.ts`

---

## 3. Hard Rules

### 3.1 Greenfield Template Rule

This is greenfield template work. Existing generated scenarios do not get
compatibility shims from this plan.

Under `path:templates/scenarios/react-vite/**`:

- No compatibility wrappers, alias imports, deprecated exports, or old+new paths.
- No `// Deprecated:`, `// legacy`, `// compat`, `backward compatibility`, or
  similar comments.
- If a new template shape breaks an old generated scenario, that is out of
  scope; old scenarios update when next touched.

### 3.2 Additive Shared-Package Rule

Changes under shared packages such as `packages/cli-core`,
`packages/api-core`, and `packages/api-base` must be additive:

- Do not remove exported symbols.
- Do not change exported function signatures used by existing scenarios.
- New helpers may be added and then adopted by the template.

### 3.3 Dependency Rule

Do not add new third-party dependencies without explicit user permission.
This plan should be implementable with existing repo packages, Go stdlib,
existing npm packages, and local scripts.

### 3.4 Scenario Lifecycle Rule

All generated-scenario validation must use lifecycle commands:

```bash
cd scenarios/<generated-smoke>
make setup
make test
make start
make status
make stop
```

Never start generated scenario binaries directly outside lifecycle-managed
commands.

---

## 4. Problem Statement

The current template has a professional architecture, but several remaining
issues carry high replication risk:

1. `.github/workflows/test.yml` runs `pnpm install --frozen-lockfile` and
   caches `ui/pnpm-lock.yaml`, but the template source does not currently
   include `ui/pnpm-lock.yaml`.
2. `docs/internal/TESTING.md` contains a stale canonical health test snippet
   showing `server.Deps{Pinger: ...}` even though health now owns `Pinger`
   through `health.Module(...)`.
3. `.vrooli/service.json` describes the `develop` lifecycle as launching a
   Vite dev server, but the actual UI process serves the production bundle
   with `node server.js`.
4. API production-import guardrails ban `internal/testutil` imports but do
   not ban production files importing domain-local `mocks` packages.
5. `server.New` documents that `Clock` must be explicit but does not fail
   clearly if it is omitted.
6. `cli_commands_seed.json` is still hand-maintained as a third metadata
   source beside CLI registration and endpoint descriptors.
7. Adding a new domain still needs central edits in API registry, API main,
   CLI domains aggregator, UI app composition, selectors, locale catalogs,
   generated strings, proto schemas, endpoint manifest, and CLI metadata.
8. i18n locale parity is enforced, but adding strings still requires copying
   keys across three locale files manually.
9. The notes attachments slice is valuable as the REST-multipart exception
   reference, but it increases the size of the canonical example and deletion
   surface.

These are not severe for one scenario. They matter because this template is
the copy source for potentially many scenarios.

---

## 5. Scope

### 5.1 In Scope

- Template lockfile / standalone CI first-push correctness.
- Documentation repair for stale examples and lifecycle wording.
- Production import guardrails for domain `mocks` packages.
- Explicit `server.New` missing-dependency failure behavior.
- CLI command metadata generation from the real CLI registration surface.
- A domain scaffolding helper that reduces manual central edits.
- Locale synchronization tooling that keeps non-English catalogs in parity
  without hand-copying every key.
- A decision and implementation path for the notes attachments reference
  scope.
- Validation using template validation, local package tests, and a generated
  smoke scenario.

### 5.2 Out of Scope

- Migrating existing scenarios generated from older template revisions.
- Changing the template's base stack (React, Vite, Go, SQLite, Connect-RPC).
- Adding auth, resources, router libraries, or new product features.
- Replacing Connect-RPC or changing proto package conventions.
- Broad visual redesign of the UI.
- Introducing third-party i18n translation services or paid APIs.

---

## 6. Current Technical Context

### 6.1 Template Validation

Observed command:

```bash
vrooli scenario template validate
```

Observed result on 2026-05-05:

```text
Validated 2 scenario templates
```

### 6.2 Lockfile Gap

`path:templates/scenarios/react-vite/.github/workflows/test.yml`:

- Uses `cache-dependency-path: ui/pnpm-lock.yaml`.
- Runs `pnpm install --frozen-lockfile --ignore-workspace`.

`path:templates/scenarios/react-vite/ui/` currently has no `pnpm-lock.yaml`
in the template source.

### 6.3 Stale Testing Example

`path:templates/scenarios/react-vite/docs/internal/TESTING.md` shows:

```go
srv := server.New(server.Deps{Pinger: pinger, Clock: clock.System{}, /*…*/})
```

Actual current shape:

- `server.Deps` contains only `Clock` and `Logger`.
- `Pinger` is passed into `health.Module(pinger, service, version)`.

### 6.4 Lifecycle Wording Drift

`path:templates/scenarios/react-vite/.vrooli/service.json` says:

- `develop.description`: `Launch API and Vite dev server`.

Actual UI step:

- `cd ui && node server.js`
- The step serves `ui/dist/index.html`, not Vite dev server output.

### 6.5 Existing Guardrails

API guard:

- `path:templates/scenarios/react-vite/api/internal/testutil/no_prod_import_test.go`
- Bans production imports of `<module>/internal/testutil`.
- Exempts files inside `mocks/` directories.
- Does not currently fail when a production file imports
  `<module>/internal/<domain>/mocks`.

UI guard:

- `path:templates/scenarios/react-vite/ui/eslint.config.js`
- Bans production imports from `src/test-utils` and `features/<dom>/mocks`.

CLI guard:

- `path:templates/scenarios/react-vite/cli/internal/testutil/no_prod_import_test.go`
- Bans production imports of CLI `internal/testutil`.

### 6.6 Central Registries

Manual central touch points today:

- `api/main.go` for runtime module constructors.
- `api/internal/modules/registry.go` for endpoint and schema providers.
- `api/cmd/gen-endpoints/cli_commands_seed.json` for CLI command metadata.
- `cli/domains/domains.go` for CLI domain registration.
- `ui/src/App.tsx` for feature rendering.
- `ui/src/consts/selectors.ts` for test IDs.
- `ui/src/i18n/locales/*.json` for copy.
- `ui/src/consts/strings.generated.ts` via codegen.
- `.vrooli/endpoints.json` via codegen.

---

## 7. Target End State

After this plan lands:

1. Freshly generated scenarios include a valid UI lockfile or use a CI path
   that does not require one. First standalone GitHub Actions run succeeds
   without manual lockfile generation.
2. Template docs match the current Connect/module/server shape.
3. Lifecycle wording accurately describes whether the UI is served from a
   production bundle or a true dev server.
4. Production Go code cannot import scenario-local test packages or
   domain-local `mocks` packages.
5. `server.New` fails with an immediate, clear message when required
   cross-cutting dependencies are missing.
6. `.vrooli/endpoints.json` `cli_commands[]` entries are generated from the
   CLI's registered command tree, not hand-maintained JSON.
7. Adding a new domain can be started with a Vrooli-managed generator/helper
   that creates the standard folders/files and updates known central
   registrations.
8. Adding a UI string requires editing English once, then running a sync tool
   that inserts missing keys in other locale catalogs with deterministic
   placeholder values.
9. The attachments reference decision is documented and implemented:
   either it remains in the base template with sharper deletion/scaffolding
   docs, or it moves to a secondary example without bloating the base CRUD
   reference.
10. A generated smoke scenario validates setup, tests, start, health, and stop.

---

## 8. Implementation Strategy

### Phase 0 — Baseline and Safety Snapshot

**Goal:** Confirm the current state before changing the template.

Actions:

1. Record worktree state:
   ```bash
   git status --short
   ```
2. Validate the template:
   ```bash
   vrooli scenario template validate
   ```
3. Run targeted template tests where currently feasible:
   ```bash
   ( cd templates/scenarios/react-vite/api && go test -race ./... )
   ( cd templates/scenarios/react-vite/cli && go test -race ./... )
   ```
4. For UI, run only if dependencies are already installed. If not, defer
   until Phase 1 lockfile work:
   ```bash
   ( cd templates/scenarios/react-vite/ui && pnpm strings:check && pnpm type-check && pnpm lint && pnpm test:coverage )
   ```

Deliverable:

- Add an implementation note to this plan or `docs/internal/PROGRESS.md`
  after execution with exact commands and results.

### Phase 1 — Lockfile and Standalone CI Correctness

**Goal:** Ensure a generated standalone scenario passes its first GitHub
Actions run.

Preferred implementation:

1. From template UI:
   ```bash
   cd templates/scenarios/react-vite/ui
   corepack pnpm install --ignore-workspace
   ```
2. Commit `path:templates/scenarios/react-vite/ui/pnpm-lock.yaml`.
3. Verify placeholder substitution is acceptable in the lockfile:
   ```bash
   rg "{{SCENARIO_ID}}|{{SCENARIO_DISPLAY_NAME}}" pnpm-lock.yaml
   ```
   If placeholders appear, generate a smoke scenario and confirm the
   generator substitution pass rewrites them. If not, update the generator's
   substitution allowlist to include lockfiles.
4. Change template setup commands to prefer frozen lockfiles when a lockfile
   exists:
   - `template.json::postHooks` UI install command.
   - `.vrooli/service.json::lifecycle.setup.steps.install-ui-deps`.
5. Keep a non-frozen fallback only if required for local generation; if a
   fallback remains, document why in `docs/internal/TEMPLATE-MAINTENANCE.md`.

Validation:

```bash
( cd templates/scenarios/react-vite/ui && pnpm install --frozen-lockfile --ignore-workspace )
( cd templates/scenarios/react-vite/ui && pnpm strings:check && pnpm type-check && pnpm lint && pnpm test:coverage && pnpm build )
```

Acceptance:

- `ui/pnpm-lock.yaml` exists in the template.
- `.github/workflows/test.yml` install step succeeds locally with
  `--frozen-lockfile --ignore-workspace`.
- README / troubleshooting docs no longer imply lockfile generation is a
  manual first-push task.

### Phase 2 — Documentation and Lifecycle Drift Repair

**Goal:** Make generated docs match current code.

Actions:

1. Update `docs/internal/TESTING.md` canonical health snippet:
   - Construct `health.Module(pinger, "service", "version")`.
   - Construct `server.New(server.Deps{Clock, Logger}, healthModule)`.
   - Remove any reference to `server.Deps.Pinger`.
2. Search and update stale references:
   ```bash
   rg -n "Deps\\{Pinger|server\\.Deps\\{Pinger|Vite dev server|vite dev server|httptest.NewRecorder|backward compatibility" templates/scenarios/react-vite
   ```
3. Update `.vrooli/service.json` wording:
   - If keeping production-bundle serving: change `develop.description` to
     `Launch API and production UI server`.
   - If adding a true dev lifecycle path: add it deliberately and document
     which Make target uses it. Do not silently repurpose `make start`.
4. Remove greenfield-inconsistent compatibility wording from
   `ui/src/i18n/locales.ts` and `ui/src/i18n/index.ts`.
5. Reword testing docs so `httpx.NewLiveServer` is required for
   handler/client behavior, while narrow mux reachability tests may use
   `httptest.NewRecorder` when no socket semantics are under test.

Validation:

```bash
rg -n "Deps\\{Pinger|server\\.Deps\\{Pinger|backward compatibility|Vite dev server|vite dev server" templates/scenarios/react-vite
```

Acceptance:

- No stale health-wiring snippet remains.
- Lifecycle docs and manifest wording agree with actual commands.
- No greenfield-inconsistent compatibility prose remains under the template.

### Phase 3 — Guardrail Hardening

**Goal:** Turn current conventions into tests.

Actions:

1. Extend API `no_prod_import_test.go`:
   - Continue skipping files inside `mocks/` directories themselves.
   - Fail any non-test, non-mocks production file that imports a path
     containing `/mocks` under the module.
   - Keep the existing `internal/testutil` import ban.
2. Add equivalent CLI guard only if CLI gains domain-local mocks in this
   plan or in the future. Otherwise document the API-only scope.
3. Update `docs/internal/TESTING.md` and `docs/internal/SEAMS.md` to state:
   - Domain mocks are test-only.
   - Production code must never import `internal/<domain>/mocks`.
4. Change `server.New` to fail explicitly if `Clock` is nil:
   - Prefer `panic("server.New: Clock is required")` for programmer error.
   - Add tests in `api/internal/server/server_test.go`.
   - Keep `Logger` defaulting only if the template intentionally treats it
     as optional; otherwise make `Logger` required too and update all call
     sites. Recommended: keep `Logger` optional, make `Clock` required,
     because logging to `log.Default()` is safe while nil clock is not.

Validation:

```bash
( cd templates/scenarios/react-vite/api && go test -race ./... )
```

Acceptance:

- A production import of `internal/notes/mocks` fails the API test suite.
- `server.New(server.Deps{})` fails with a clear panic covered by tests.
- Existing API tests remain green.

### Phase 4 — Generate CLI Command Metadata

**Goal:** Remove hand-maintained `cli_commands_seed.json` as a drift source.

Design decision:

- The source of truth must be the CLI registration tree returned by
  `domains.CommandGroups(core)` and `domains.SubcommandGroups(core)`.
- The API should not import the CLI module directly.
- The generator can execute the CLI metadata dumper as a separate process
  or share a small additive `cli-core` introspection surface.

Recommended implementation:

1. Add additive command-tree introspection to `packages/cli-core/cliapp`:
   - A pure data type such as `CommandMetadata`.
   - A method/function that walks registered command groups and subcommand
     groups without invoking API calls.
   - Include command name, description, subcommand path, `NeedsAPI`, args,
     and whether it is built-in.
2. Add a scenario CLI hidden command or build-time command:
   - Example: `{{SCENARIO_ID}} __dump-commands --json`
   - It must not require API connectivity.
   - It must include built-ins such as `status` and `configure` when relevant.
3. Replace `api/cmd/gen-endpoints/cli_commands_seed.json` with generated
   command metadata:
   - Either `make endpoints` first invokes the CLI dumper and pipes a temp
     JSON file into `gen-endpoints`.
   - Or `gen-endpoints` accepts `--commands-cmd "../cli/{{SCENARIO_ID}} __dump-commands --json"`.
4. Preserve the cross-check:
   - Every endpoint `cli_mapping.command` must exist in generated CLI metadata.
   - Every generated CLI command that declares an endpoint id must point to an
     existing endpoint.
5. Remove `cli_commands_seed.json` from the template after the generated path
   is green. No compatibility seed fallback in the template.

Validation:

```bash
( cd packages/cli-core && go test -race ./... )
( cd templates/scenarios/react-vite/cli && go test -race ./... )
( cd templates/scenarios/react-vite && make endpoints )
git diff --exit-code templates/scenarios/react-vite/.vrooli/endpoints.json
```

Generated-smoke validation:

```bash
vrooli scenario generate react-vite --id cli-metadata-smoke --display-name "CLI Metadata Smoke" --description "CLI metadata generation smoke"
cd scenarios/cli-metadata-smoke
make endpoints
make test
make stop
```

Cleanup after smoke:

- Remove the generated scenario and all relocated proto artifacts from:
  - `packages/proto/schemas/cli-metadata-smoke`
  - `packages/proto/gen/go/cli-metadata-smoke`
  - `packages/proto/gen/typescript/js/cli-metadata-smoke`
  - `packages/proto/gen/python/cli_metadata_smoke`
- Run `( cd packages/proto && make generate )`.

Acceptance:

- `cli_commands_seed.json` no longer exists in the template.
- `make endpoints` regenerates `cli_commands[]` from real CLI registration.
- Missing endpoint↔CLI mappings fail with actionable errors.

### Phase 5 — Domain Scaffolding Helper

**Goal:** Make adding a domain a generated workflow rather than a long
copy-and-edit checklist.

Recommended command:

```bash
vrooli scenario domain add <scenario-id> <domain-name> \
  --template notes-crud \
  --display-name "Tasks"
```

Implementation home:

- `path:internal/cli/scenariocli/` for command surface.
- `path:internal/cli/scenariohandlers/` for filesystem/template runtime.
- Reuse existing template substitution helpers where possible.

Generated output for `tasks`:

- Proto:
  - `packages/proto/schemas/<scenario>/v1/tasks/tasks.proto`
- API:
  - `api/internal/tasks/{types,repository,sqlite,service,schema}.go`
  - `api/internal/tasks/schema.sql`
  - `api/internal/tasks/mocks/...`
  - `api/handlers/tasks/{adapter,connect_handler,module}.go`
  - tests beside each layer
- CLI:
  - `cli/domains/tasks/{register,handlers}.go`
  - handler tests
- UI:
  - `ui/src/api/tasks.ts`
  - `ui/src/features/tasks/TasksCard.tsx`
  - feature mocks/factories/tests
- Central updates:
  - `api/main.go`
  - `api/internal/modules/registry.go`
  - `cli/domains/domains.go`
  - `ui/src/App.tsx`
  - `ui/src/consts/selectors.ts`
  - `ui/src/i18n/locales/en.json`
  - Generated outputs: `strings.generated.ts`, `.vrooli/endpoints.json`,
    proto gen artifacts.

Design constraints:

- The helper must never use mass blind replacement. It should parse structured
  files where practical:
  - Go files: preferably small AST-aware insertion or tightly scoped marker
    comments.
  - JSON locale files: parse/write JSON deterministically.
  - TS selectors/App: use narrow textual insertion only if the file has stable
    anchors and tests cover it.
- The helper should print a "next manual edits" list for domain-specific
  business rules it cannot infer.
- Generated code should compile and tests should pass before scenario-specific
  customization, even if the domain is only CRUD skeleton.

Validation:

```bash
go test ./internal/cli/scenariohandlers ./internal/cli/scenariocli
vrooli scenario generate react-vite --id domain-helper-smoke --display-name "Domain Helper Smoke" --description "Domain helper smoke"
vrooli scenario domain add domain-helper-smoke tasks --template notes-crud --display-name "Tasks"
( cd packages/proto && make generate )
( cd scenarios/domain-helper-smoke && make setup && make test && make start && make status && make stop )
```

Acceptance:

- A new CRUD domain can be scaffolded with one command.
- The command updates all known central registrations.
- The generated scenario passes setup/test/start/status/stop.
- `REPLACING-NOTES.md` is updated to recommend the helper before manual copy.

### Phase 6 — Locale Synchronization Tooling

**Goal:** Keep locale parity while reducing manual copy work.

Recommended implementation:

1. Add `ui/scripts/sync-locales.mjs`.
2. Add scripts:
   - `pnpm locales:sync`
   - `pnpm locales:check`
3. Behavior:
   - `en.json` remains canonical.
   - For every other locale, insert missing keys with deterministic placeholder
     values copied from English and prefixed or annotated in a way that is
     obvious to translators.
   - Preserve existing translated values.
   - Remove extra keys only in explicit `--prune` mode, not by default.
   - Preserve CLDR plural variant rules already encoded in
     `locales.test.ts`.
4. Update `strings:gen` workflow if useful:
   - Do not make strings generation silently mutate non-English locales.
   - Prefer explicit `pnpm locales:sync && pnpm strings:gen`.
5. Document the workflow in:
   - `docs/internal/TESTING.md`
   - `docs/guides/troubleshooting.md`
   - `docs/internal/REPLACING-NOTES.md`

Validation:

```bash
( cd templates/scenarios/react-vite/ui && pnpm locales:check )
( cd templates/scenarios/react-vite/ui && pnpm locales:sync && pnpm strings:gen && pnpm strings:check )
( cd templates/scenarios/react-vite/ui && pnpm type-check && pnpm lint && pnpm test:coverage )
```

Acceptance:

- Adding an English string and running `pnpm locales:sync` updates all locale
  files deterministically.
- `locales.test.ts` still enforces key and interpolation parity.
- No translated existing values are overwritten by default.

### Phase 7 — Attachments Reference Scope Decision

**Goal:** Decide and implement the right canonical scope for the notes
reference.

Decision options:

**Option A — Keep attachments in the base template.**

Use if the REST multipart exception is important enough to be present in
every generated scenario as a reference. Required improvements:

- Keep the current co-located deletion guidance in `REPLACING-NOTES.md`.
- Add an explicit "attachments sub-resource map" table:
  - proto file
  - API service/repository/sqlite files
  - handler/module files
  - CLI attach handler
  - UI `AttachmentUpload`
  - selectors/strings
- Ensure the domain helper can generate CRUD-only domains without copying
  attachments by default.
- Add `--with-attachments` to the domain helper for scenarios that need the
  multipart reference.

**Option B — Move attachments to a secondary template/example.**

Use if the base domain-add/delete cost matters more than having multipart in
every generated scenario. Required changes:

- Remove attachments files from the base notes domain.
- Add a template-maintainer or generated guide such as
  `docs/internal/ADDING-MULTIPART-RESOURCE.md`.
- Optionally add a separate template variant only if the existing scenario
  generator supports it cleanly without duplicating the whole react-vite tree.

Recommended decision:

- Implement Option A first unless the direct measurement in this phase shows
  attachments dominate generated-scenario friction after the domain helper
  exists. Attachments are a useful reference, but the domain helper should
  make them opt-in for new domains.

Measurement before final decision:

```bash
# Use the existing template-fitness harness recipes.
# Directly re-measure scenario 2 (add domain) and scenario 5 (delete notes)
# after Phases 4-6 but before changing attachments scope.
```

Acceptance:

- The decision is recorded in `docs/internal/TEMPLATE-MAINTENANCE.md`.
- `REPLACING-NOTES.md` and `ARCHITECTURE.md` match the chosen scope.
- If attachments remain, the domain helper supports CRUD-only and
  attachments-enabled generation paths.

### Phase 8 — Template Maintenance Checklist

**Goal:** Make future template changes self-checking.

Actions:

1. Add a generated or documented checklist in
   `docs/internal/TEMPLATE-MAINTENANCE.md`.
2. Add a Make target if appropriate:
   ```bash
   make template-check
   ```
   or document root-level commands if a Make target is not desirable.
3. Checklist should include:
   ```bash
   vrooli scenario template validate
   rg -n "Deps\\{Pinger|server\\.Deps\\{Pinger|backward compatibility|Vite dev server|vite dev server" templates/scenarios/react-vite
   ( cd templates/scenarios/react-vite/api && go test -race ./... )
   ( cd templates/scenarios/react-vite/cli && go test -race ./... )
   ( cd templates/scenarios/react-vite/ui && pnpm install --frozen-lockfile --ignore-workspace && pnpm strings:check && pnpm locales:check && pnpm type-check && pnpm lint && pnpm test:coverage && pnpm build )
   ```
4. Include smoke scenario generation/cleanup instructions.

Acceptance:

- Future agents have one obvious validation sequence for template edits.
- The checklist includes cleanup instructions for relocated proto artifacts.

### Phase 9 — End-to-End Generated Smoke

**Goal:** Prove a fresh scenario generated from the final template works.

Actions:

1. Generate smoke:
   ```bash
   vrooli scenario generate react-vite \
     --id react-vite-polish-smoke \
     --display-name "React Vite Polish Smoke" \
     --description "Smoke scenario for react-vite template polish"
   ```
2. Run lifecycle:
   ```bash
   cd scenarios/react-vite-polish-smoke
   make setup
   make test
   make start
   make status
   make stop
   ```
3. Validate generated helper flows:
   ```bash
   vrooli scenario domain add react-vite-polish-smoke tasks --template notes-crud --display-name "Tasks"
   ( cd packages/proto && make generate )
   cd scenarios/react-vite-polish-smoke
   make setup
   make test
   make start
   make status
   make stop
   ```
4. Cleanup explicitly:
   ```bash
   rm -rf scenarios/react-vite-polish-smoke
   rm -rf packages/proto/schemas/react-vite-polish-smoke
   rm -rf packages/proto/gen/go/react-vite-polish-smoke
   rm -rf packages/proto/gen/typescript/js/react-vite-polish-smoke
   rm -rf packages/proto/gen/python/react_vite_polish_smoke
   ( cd packages/proto && make generate )
   ```
5. Verify zero residue:
   ```bash
   ls -d packages/proto/gen/{go,typescript/js}/*react-vite-polish-smoke* packages/proto/gen/python/*react_vite_polish_smoke*
   ```
   Expected: no matches.

Acceptance:

- Smoke scenario lifecycle passes.
- Domain helper output lifecycle passes.
- No generated scenario or proto residue remains.

---

## 9. Contract Decisions

### 9.1 CLI Metadata Contract

`cli_commands[]` in `.vrooli/endpoints.json` must be generated from actual
CLI command registration, not from hand-maintained JSON. Endpoint descriptors
may still carry `cli_mapping` because they explain which endpoint a command
mirrors, but command descriptions and command existence must come from the CLI.

### 9.2 Locale Contract

English remains canonical. Non-English locale files must stay complete, but
the tool may insert English fallback strings as placeholders. CI should fail
when locale catalogs are structurally out of sync, not when translations are
still pending.

### 9.3 Server Dependency Contract

`Clock` is required for `server.New`; `Logger` may default to `log.Default()`.
Missing required dependencies should fail immediately during tests or startup,
not later through nil-interface behavior.

### 9.4 Attachments Contract

Opaque bytes remain REST multipart exceptions. Metadata stays proto-typed.
Whether the attachments example remains in the base template or moves to
secondary guidance, Connect-RPC remains the default for Vrooli-owned typed
payloads.

---

## 10. Testing Plan

### 10.1 Template Validation

```bash
vrooli scenario template validate
```

### 10.2 API

```bash
( cd templates/scenarios/react-vite/api && go vet ./... )
( cd templates/scenarios/react-vite/api && go build ./... )
( cd templates/scenarios/react-vite/api && CGO_ENABLED=0 go build ./... )
( cd templates/scenarios/react-vite/api && go test -race ./... )
```

### 10.3 CLI

```bash
( cd templates/scenarios/react-vite/cli && go vet ./... )
( cd templates/scenarios/react-vite/cli && go build ./... )
( cd templates/scenarios/react-vite/cli && CGO_ENABLED=0 go build ./... )
( cd templates/scenarios/react-vite/cli && go test -race ./... )
```

### 10.4 UI

```bash
( cd templates/scenarios/react-vite/ui && pnpm install --frozen-lockfile --ignore-workspace )
( cd templates/scenarios/react-vite/ui && pnpm locales:check )
( cd templates/scenarios/react-vite/ui && pnpm strings:check )
( cd templates/scenarios/react-vite/ui && pnpm type-check )
( cd templates/scenarios/react-vite/ui && pnpm lint )
( cd templates/scenarios/react-vite/ui && pnpm test:coverage )
( cd templates/scenarios/react-vite/ui && pnpm build )
```

### 10.5 Shared Packages

Run only for touched packages:

```bash
( cd packages/cli-core && go test -race ./... )
( cd packages/api-core && go test -race ./... )
( cd packages/api-base && pnpm test || true )
```

If `packages/api-base` does not have a test script, replace with the package's
actual validation command and record that in the execution notes.

### 10.6 Generated Smoke

Use Phase 9 commands exactly.

---

## 11. Rollout and Validation Checklist

- [ ] Phase 0 baseline commands recorded.
- [ ] `ui/pnpm-lock.yaml` exists or CI no longer requires it.
- [ ] Template UI install passes with `--frozen-lockfile --ignore-workspace`.
- [ ] Stale health test docs fixed.
- [ ] Lifecycle wording matches actual UI command.
- [ ] Greenfield-inconsistent compatibility prose removed from template.
- [ ] API guard fails production imports from `internal/<domain>/mocks`.
- [ ] `server.New` required dependency behavior covered by tests.
- [ ] CLI command metadata generated from CLI registration.
- [ ] `cli_commands_seed.json` removed from template.
- [ ] `make endpoints` remains deterministic.
- [ ] Domain helper scaffolds a CRUD domain.
- [ ] Locale sync/check scripts added and documented.
- [ ] Attachments scope decision recorded and implemented.
- [ ] Template maintenance checklist updated.
- [ ] `vrooli scenario template validate` green.
- [ ] API, CLI, UI targeted gates green.
- [ ] Generated smoke lifecycle green.
- [ ] Generated smoke cleanup and proto regen complete.
- [ ] `git status --short` shows only intentional files.

---

## 12. Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Lockfile contains unsubstituted placeholders | Generated scenarios fail install | Verify with generated smoke; extend generator substitution for lockfiles if needed |
| CLI metadata dumper imports API or starts API | Codegen becomes slow/flaky | Dumper must walk registration metadata only and be `NeedsAPI=false` |
| Domain helper becomes a fragile mass-rewrite script | Template changes break generator | Use structured parsing where practical and stable anchors with tests elsewhere |
| Locale sync overwrites real translations | Translation loss | Preserve existing values by default; only insert missing keys; prune only with explicit flag |
| `server.New` panic breaks existing template tests | Temporary failures | Update tests and all template call sites in same phase |
| Attachments extraction balloons scope | Long-running plan | Make a measured decision; Option A (keep but make opt-in for generated domains) is the default |
| Generated smoke leaves proto residue | Future builds polluted | Follow explicit cleanup paths for Go, TS, Python generated artifacts; run `packages/proto && make generate` |

---

## 13. Non-Goals and Prohibited Patterns

Do not:

- Add compatibility shims in the template.
- Add third-party dependencies without permission.
- Create a generic `utils` dumping ground.
- Start scenarios by directly executing binaries.
- Hand-edit `.vrooli/endpoints.json` except as a generated artifact.
- Keep `cli_commands_seed.json` as a fallback after CLI metadata generation
  is working.
- Weaken locale parity, string registry, a11y, coverage, or test-utils
  quarantine gates.
- Lower coverage thresholds to make this plan pass.
- Use mass-update scripts to rewrite many files blindly.

---

## 14. Definition of Done

This plan is complete when:

1. All target phases have landed or any skipped phase has a written,
   evidence-backed reason in this file.
2. The template passes:
   ```bash
   vrooli scenario template validate
   ( cd templates/scenarios/react-vite/api && go test -race ./... )
   ( cd templates/scenarios/react-vite/cli && go test -race ./... )
   ( cd templates/scenarios/react-vite/ui && pnpm install --frozen-lockfile --ignore-workspace && pnpm locales:check && pnpm strings:check && pnpm type-check && pnpm lint && pnpm test:coverage && pnpm build )
   ```
3. Touched shared packages pass their package-specific tests.
4. A freshly generated smoke scenario passes lifecycle setup/test/start/status/stop.
5. A domain added by the new helper passes the same lifecycle gates.
6. Generated smoke scenario and proto artifacts are removed and proto codegen is
   regenerated.
7. `rg -n "Deps\\{Pinger|server\\.Deps\\{Pinger|backward compatibility|Vite dev server|vite dev server" templates/scenarios/react-vite`
   returns no stale matches except deliberately documented historical plan files
   outside the template.
8. `git grep -n "cli_commands_seed.json" templates/scenarios/react-vite` returns
   no matches after the CLI metadata generator is implemented.
9. Template diffs contain no `Deprecated:`, `legacy`, `compat`, or
   greenfield-violating comments.

