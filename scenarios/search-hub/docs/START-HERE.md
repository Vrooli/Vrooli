# Start Here — Search Hub

This is the operator and contributor orientation for Search Hub. It records
the lifecycle, architecture, quality gates, and handoff evidence for the
federated retrieval control plane. The scenario is already implemented; use
the gates below as maintenance and certification checks, not as instructions
to replace a starter application.

Run the scenario lifecycle through `make setup`, `make start`, `make test`,
and `make stop`. Use `search-hub maturity scan --json` for the current search
certification report and `cli-health validate scenario search-hub --json` for
the CLI contract.

Search Hub owns registry metadata, routing, eval mirrors, telemetry, health,
and operator surfaces. Provider scenarios own their corpora and retrieval
implementations. The UI and CLI are translation layers over typed API
contracts; they are not alternate policy implementations.

The proto schemas, provider descriptor contract, failure semantics,
routing/eval policy, and lifecycle commands are authoritative. UI copy,
layout details, and examples are replaceable as long as they preserve
accessible degraded/error states and the documented ownership boundaries.

## Initialization Protocol

### Gate 0 — Scaffold Health

- [ ] Run `make setup` from this scenario directory.
- [ ] Run `make start`.
- [ ] Run `make status` and confirm API/UI lifecycle metadata resolves.
- [ ] Run `make test`.
- [ ] Fix lifecycle, generation, codegen, dependency, or test failures
      before product work.

**Exit criteria:** the generated scenario boots, reports status, and
passes the template test lifecycle.

### Gate 1 — Charter

Do not hand-write the final `PRD.md` when AI generation is available.
Drive the `business-health` wizard as the canonical PRD authoring path so the file
matches the Vrooli PRD standard and can drive requirement generation.

- [ ] Write a brief context file for the scenario:

```bash
cat > /tmp/prd_context_search-hub.md <<'EOF'
Overview:
- Purpose:
- Primary users / verticals:
- Deployment surfaces:
- Value promise:

P0 operational targets:
- ...

P1 operational targets:
- ...

P2 operational targets:
- ...

Tech direction snapshot:
- Preferred stacks / frameworks:
- Data + storage expectations:
- Integration strategy:
- Non-goals / guardrails:

Dependencies & launch plan:
- Required resources:
- Scenario dependencies:
- Operational risks:
- Launch sequencing:

UX & branding:
- Look and feel:
- Accessibility bar:
- Voice and messaging:
- Branding hooks:
- PWA install surface: keep the seeded `ui/public/site.webmanifest`,
  `apple-icon-180.png`, `favicon-196.png`, and maskable manifest icons
  valid; replace the generic icons when final product branding exists.
EOF
```

- [ ] Generate and publish the PRD:

```bash
business-health wizard start  # (was: prd generate) search-hub \
  --context-file /tmp/prd_context_search-hub.md \
  --publish \
  --json
```

- [ ] Validate the PRD:

```bash
vrooli scenario requirements validate search-hub --json
```

- [ ] Read the published `PRD.md` and confirm it captures the intended
      permanent capability, users, operational targets, dependencies,
      risks, and UX direction.
- [ ] Treat `PRD.md` as read-only after this gate except for automated
      checkbox updates.

**Exit criteria:** `PRD.md` exists, validates, and contains stable
P0/P1/P2 operational targets that can drive requirements and tests.

### Gate 2 — Requirements Registry

Generate requirements from the PRD operational targets. Requirements
are the implementation-facing measurement layer; tests should later tag
`[REQ:ID]` so sync tooling can update status.

The generated scenario includes a tiny starter module at
`requirements/01-foundation/module.json` only so first-run validation has
a valid registry to inspect. Do not treat it as product scope. Replace
it with PRD-specific modules during this gate, or regenerate the whole
registry from the PRD and remove the starter import from
`requirements/index.json`.

- [ ] Optionally write requirement-generation context:

```bash
cat > /tmp/requirements_context_search-hub.md <<'EOF'
Validation approach:
- Unit:
- Integration:
- Business / BAS:
- Performance:

Technical constraints:
- ...

Requirement details:
- ...
EOF
```

- [ ] Generate requirements:

```bash
business-health wizard apply  # (was: requirements generate) search-hub \
  --context-file /tmp/requirements_context_search-hub.md \
  --json
```

- [ ] Validate requirements:

```bash
vrooli scenario requirements validate search-hub --json
```

- [ ] Confirm `requirements/index.json` imports real numbered
      `requirements/<number>-<target>/module.json` files that mirror
      the PRD operational targets, not just the starter foundation
      module.
- [ ] Confirm each P0/P1 target has at least one linked requirement and
      each requirement has a concrete validation strategy.

**Exit criteria:** the requirements registry validates and gives future
implementation agents clear `[REQ:ID]` targets for tests.

### Gate 3 — Domain Map

Use `docs/concepts/DOMAINS.md` as the durable domain-map artifact. Do
the thinking before coding.

- [ ] Name the real bounded contexts for this scenario.
- [ ] For each domain, identify the data it owns, proto operations, API
      behavior, CLI commands, UI surface, storage needs, and test
      evidence.
- [ ] Confirm each domain maps back to at least one operational target
      or requirement.
- [ ] Update `docs/concepts/DOMAINS.md` so the Domain Inventory and
      Domain Details sections describe this scenario rather than only
      generic reference domains.
- [ ] Review `docs/concepts/DATA.md`, `docs/concepts/FLOWS.md`, and
      `docs/concepts/INTEGRATIONS.md`; fill the relevant sections or
      leave explicit deferred/not-applicable entries.

**Exit criteria:** you can explain what the first real domain is, why
it exists, and which files it will touch before writing code.

### Gate 4 — Dependency Decisions

- [ ] Keep SQLite unless a domain truly needs a shared resource.
- [ ] If adding resources or scenario dependencies, document the reason
      in `PRD.md` during the charter gate and in
      `docs/concepts/INTEGRATIONS.md` before editing
      `.vrooli/service.json`.
- [ ] Confirm no dependency is added only because a sample domain
      happens to use a local SQLite store.

**Exit criteria:** `.vrooli/service.json` reflects only dependencies
the real scenario needs.

### Gate 5 — Design Language

Generation installs root-level `DESIGN.md` from the selected design kit.
Treat that file as the UI source of truth before building screens.

**Binding vs. illustrative.** Read `DESIGN.md` carefully:

- *Binding* (must follow): tokens, color roles, typography scale,
  spacing, radius, motion rules, status-color semantics,
  accessibility floors, responsive transformations, and the overall
  feel/density target.
- *Illustrative* (examples, not exhaustive): any concrete list of
  components, page surfaces, settings, or copy. If the design lists
  "preferred primitives" or shows an example settings page, those are
  **shape hints**, not a feature checklist. Implement the full set of
  features and surfaces your scenario actually needs, even if the
  design doc does not enumerate them.

Concretely: if `DESIGN.md` shows a settings page with three example
controls, and your scenario also needs theme, locale, accessibility,
account, or notification settings, build all of them. The design
governs how they should look and behave, not which ones exist.

- [ ] Read `DESIGN.md` and confirm it fits this scenario's users,
      density, workflow, and accessibility needs.
- [ ] If the scenario needs a different language, regenerate with a
      compatible `--design <kit-id>` or intentionally update
      `DESIGN.md` before UI implementation.
- [ ] Keep `ui/src/design-tokens.css`, `ui/tailwind.theme.json`,
      `ui/tailwind.config.ts`, and reusable `ui/src/components/ui/`
      primitives aligned with `DESIGN.md`.
- [ ] Replace the placeholder `AppShell` and home page. Design the
      real shell, navigation, and surfaces your scenario needs.
      Preserve the durable seams listed above (i18n, locale switcher
      behavior, accessibility selectors, design tokens) inside
      whatever new layout you build.
- [ ] Audit settings. The placeholder shell ships with the bare
      minimum (currently just locale switching). Inventory every
      preference your scenario needs (theme, font scale, locale,
      a11y, account, notifications, scenario-specific toggles) and
      build the full settings surface — do not constrain yourself to
      whatever happens to be shown as an example in `DESIGN.md`.
- [ ] Do not create `docs/DESIGN_LANGUAGE.md`; root `DESIGN.md` is the
      canonical design contract.

**Exit criteria:** UI work has a reviewed root `DESIGN.md`, the
placeholder shell has been replaced with scenario-specific layout, the
settings surface covers everything this scenario actually needs, and
the global styles, Tailwind theme, primitives, selectors, and
accessibility tests all point back to the design contract.

### Gate 5b — Business And Operations Stubs

These documents start as stubs. Review them early so missing business,
deployment, security, and telemetry assumptions are visible before
implementation hardens around them.

- [ ] Review `docs/business/MONETIZATION.md` and
      `docs/business/GO-TO-MARKET.md`; mark them deferred or fill the
      scenario-specific hypothesis.
- [ ] Review `docs/operations/DEPLOYMENT.md`,
      `docs/operations/RUNBOOK.md`, and
      `docs/operations/OBSERVABILITY.md`; confirm local-run assumptions
      are accurate.
- [ ] Review `docs/internal/SECURITY.md`,
      `docs/internal/PERFORMANCE.md`, and
      `docs/internal/DECISIONS.md`; fill any risk, budget, or durable
      choice discovered during initialization.
- [ ] Keep `docs/manifest.json` maturity values aligned with the real
      state of these documents.

**Exit criteria:** every generated documentation stub is either active,
deferred, or explicitly not-applicable for a reason.

### Gate 6 — First Real Search Hub Vertical Slice

- [ ] Add or extend one real Search Hub surface beside the existing
      provider, routing, evaluation, metrics, or maturity surfaces.
- [ ] **Start in proto.** Author `packages/proto/schemas/search-hub/v1/<domain>/<domain>.proto`
      with a `service` block FIRST, run `make generate`, then write
      handlers/CLI/UI against the generated `*Procedure` constants and
      `*Service` clients. If you find yourself writing `Path:` as a
      literal string in an `EndpointDescriptor`, stop — codegen will
      reject it. The only exceptions are the four `RESTReason` values
      in `api/internal/module/module.go`.
- [ ] **Register the proto file** in `api/internal/modules/registry.go`
      by appending a `ProtoFileEntry` to `AllProtoFiles()`. The global
      parity test (`TestProtoConnectParity`) then covers the new
      domain automatically — no per-domain parity test required.
- [ ] Mirror the pattern across proto, API domain, API handler module,
      CLI domain, UI API client, UI feature, i18n strings, selectors,
      and tests.
- [ ] Run code generation, endpoint generation, string generation, and
      tests as needed.
- [ ] Run `make test`.

**Exit criteria:** the first real domain is green across API, CLI, UI,
and scenario tests.

### Gate 7 — Preserve The Search Hub Boundary

- [ ] Keep provider corpora and scenario-owned search data in the
      owning scenario; Search Hub owns registry metadata and policy.
- [ ] Keep API handlers and CLI/UI adapters thin over generated
      contracts; do not duplicate routing, evaluation, or maturity
      business rules in presentation layers.
- [ ] Remove stale starter-template references when a surface is
      retired, and regenerate endpoint/string artifacts rather than
      editing generated output by hand.
- [ ] Verify the changed surface with focused searches, package tests,
      and the scenario's lifecycle test path.
- [ ] Run `make test`.

**Exit criteria:** the Search Hub boundary remains generic and the
changed surface is represented by real contracts, adapters, and tests.

### Gate 7b — Contribute Back To The Component Canon

Adoption takes what the canon already has. This gate closes the other half:
anything generic this scenario built goes back, so the next scenario adopts it
instead of rebuilding it. Skipping this is how a shared library starves while
every scenario grows its own copy of the same component.

It runs after the example domain is gone, because a component is harvested from
this tree and the tree has to be real first.

- [ ] Inventory every component built under `ui/src/components/` and
      `ui/src/features/` during Gates 6–7. The test is one question:
      **would a scenario in a different product area use this?** A thread list
      is generic; a thread list that reads this scenario's own descriptor shape
      is not — extract the generic half and promote that.
- [ ] Prefer raising an existing asset over adding a new one. When the gap is
      missing states rather than a missing component, add the states your
      experience spec already declares. Never edit a released version
      directory:

```bash
react-component-library components draft-begin <component>
# add the stories/states your experience spec declares
react-component-library components test <component-id>
react-component-library components draft-publish <component>
```

      A raised component improves every scenario already consuming it, which a
      new asset does not.
- [ ] Promote a scenario-local component, carrying its experience contract with
      it so the canon inherits the claims rather than losing them:

```bash
react-component-library components ingest search-hub <tsx-path> <slug> \
  --experience-contract experience/components/<component>.json \
  --display-name "<Name>" --slot <slot>
```

- [ ] Validate before calling anything canonical:

```bash
react-component-library components test <component-id>
react-component-library components style-fit <component-id> search-hub
react-component-library catalog evidence capture <asset-id>
```

- [ ] Read the promotion gate. It requires parity, examples, dependency
      closure, drift evidence, and a clean replacement adoption in the origin
      scenario:

```bash
react-component-library workflows promotion-readiness <asset-id> \
  --origin-scenario search-hub
```

- [ ] Adopt the published version back, then delete the local original:

```bash
react-component-library adoptions apply <component-id> search-hub <adopted-path>
```

      Promotion is not finished while this scenario still runs its own copy.
      That is a fork with extra steps, and `promotion-readiness` reports it as
      an unmet origin replacement.
- [ ] Record what stays local and why, in `docs/internal/DECISIONS.md`. A
      genuinely scenario-specific component is a fine outcome; an unrecorded
      one is indistinguishable from a fork.
- [ ] Record what the canon still lacks. If this scenario needed a component
      that does not exist and could not build it here, write it where a library
      maintainer will read it, not only in this scenario's head.
- [ ] Run `make test` after adopting back — an adoption changes real imports.

**Exit criteria:** every generic component this scenario built is either
published to the canon and adopted back, or recorded in
`docs/internal/DECISIONS.md` with the reason it stayed local. No scenario-local
component duplicates one that exists in the canon.

### Gate 8 — Progress Handoff

- [ ] Append a concise row to `docs/internal/PROGRESS.md` after
      meaningful changes.
- [ ] Record known gaps or intentional deviations in
      `docs/internal/PROBLEMS.md`.
- [ ] Include validation evidence in the final handoff.

**Exit criteria:** future agents can reconstruct what happened, what is
complete, and what remains.

## Architecture Rules

- Business logic belongs in `api/internal/<domain>/`.
- Wire contracts belong in
  `packages/proto/schemas/search-hub/v1/<domain>/`.
- UI and CLI are translation layers over the API; they do not own
  business rules.
- Domain-owned schemas live next to the domain code.
- Generated files are regenerated, not hand-edited.
- Provider corpora are not Search Hub-owned data; add provider behavior in the
  owning scenario and keep Search Hub policy generic.

Read `docs/concepts/ARCHITECTURE.md` before changing structure, and
read `docs/internal/TESTING.md` before adding non-trivial tests.

## Extending Search Hub Domains

There is no starter domain to replace. When adding a new Search Hub surface,
follow the existing provider, routing, eval, metrics, or maturity domain
patterns and preserve the thin-router boundary.

For a normal proto-backed Search Hub domain:

1. Add proto messages and a service under
   `packages/proto/schemas/search-hub/v1/<domain>/`, then run
   `make generate` from `packages/proto`.
2. Add the domain's internal service and focused tests under
   `api/internal/<domain>/`; only add persistence when that domain owns
   durable Search Hub metadata.
3. Add `api/handlers/<domain>/` with a thin generated Connect handler,
   `module.go`, conversion helpers if needed, endpoint descriptors, and
   tests.
4. Register the domain schema/endpoints in
   `api/internal/modules/registry.go` and mount the module in
   `api/main.go`.
5. Add `cli/domains/<domain>/` with declarative `cliapp.ArgSchema`
   commands that call generated Connect clients, then register the
   domain in `cli/domains/domains.go`.
6. Add endpoint-to-command seed rows in
   `api/cmd/gen-endpoints/cli_commands_seed.json`, then run
   `make endpoints`.
7. Add `ui/src/api/<domain>.ts` and `ui/src/features/<domain>/` with typed
   feature components, mocks, selectors, i18n strings, and tests.
8. Run string/code generation as needed, then run `make test`.

If a future domain needs opaque binary uploads, keep bytes on a REST
multipart edge and keep metadata proto-typed; do not make corpus content a
Search Hub responsibility.
