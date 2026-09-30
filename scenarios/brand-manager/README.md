# Brand Manager

Brand meaning and the logo-to-icons pipeline for every Vrooli scenario.

brand-manager turns a proposed mark into an applied, validated identity. It
stores brands and their version history, logo **candidates** (explore, import,
refine with lineage, pick, reject, restore), the **container styles** and
**product lines** that compose a mark into a tile, the **target profiles**
(`web-public-v1`, `electron-v1`) that define every icon a scenario ships, the
**render** package that composes filter-free variant SVGs, and the **apply**
domain that writes those targets into a scenario's `/public/*` layout and
`platforms/electron/assets`. The branding validation provider
(`declared-icon-targets`) proves the applied set. Pixel and vector operations
(rasterize, vectorize, icon packing, object removal, generation) belong to
**image-tools**; brand-manager calls them over Connect and never reaches into
another scenario's storage.

The logo-refresh flow is five commands, or the Logo page in the UI:

```bash
brand-manager candidates explore --brand-id aquila --brief "a desktop constellation app" --concept "constellation eagle line art" --variations 2
brand-manager candidates list --brand-id aquila
brand-manager candidates refine cand-123 --instruction "remove the drop shadow"
brand-manager candidates pick cand-123
brand-manager apply run --brand-id aquila --scenario web-console --elements icons
```

This scenario provides the standard full-stack Vrooli shape:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/internal/PROGRESS.md`)

> **Start here:** open [`docs/START-HERE.md`](docs/START-HERE.md). It
> owns the first-session initialization protocol — charter, requirements,
> domain map, design language, placeholder replacement, and first real
> vertical slice. Run `make orient` for a machine-readable gate status.
> The pipeline design itself is in
> [`docs/concepts/BRAND-ASSET-PIPELINE.md`](docs/concepts/BRAND-ASSET-PIPELINE.md).

## What's In This Scenario

- Go API (`api/`), Go CLI (`cli/`), and React/Vite UI (`ui/`) coordinated
  through generated proto contracts.
- Domains: `brands` (with version history), `candidates`, `styles` (container
  styles + product lines), `assets`, `assignments`, `apply`, `design`,
  `discovery`, `generation`, and the served branding `validation` provider.
- The logo pipeline: candidate explore/refine/pick, deterministic render of
  mark + container style, and apply to the `/public/*` and electron target sets.
- SQLite by default; image-tools is the declared runtime dependency for
  generation, rasterization, vectorization and icon packing.
- UI/CLI guardrails for i18n, accessibility, API base resolution, declarative
  command args, generated Connect clients, and report-shaped output.
- The Logo page: a candidate gallery grouped by concept with lineage, compare,
  pick/reject/restore, refine (instruction, vectorize, masked removal), an
  explore form, and a live target-preview sheet.
- Root-level `DESIGN.md` plus generated UI token assets from the selected
  design kit.
- A documentation contract in `docs/manifest.json`, with stubs for
  domains, flows, data, integrations, monetization, deployment,
  runbooks, observability, security, performance, and durable
  decisions.

## Placeholders vs. Durable Scaffolding

The generated scaffold is intentionally not the product. When you build
the real UX, treat these as **placeholders** to replace:

- The `notes` domain (proto, API, CLI, UI feature) — a worked vertical
  slice meant to be copied once and then deleted.
- The `AppShell` and the centered single-panel home page in `ui/src/`.
- The bare-minimum settings surface (currently just locale switching).

Treat these as **durable seams** to preserve, even as you rewrite the
visual layout:

- i18n wiring (`SUPPORTED_LOCALES`, `useTranslation`, `setLocale`).
- Accessibility primitives (`role`, `aria-*`, `data-testid` selectors).
- Design tokens (`bg-app-background`, `rounded-panel`, etc.).
- The feature-folder pattern under `ui/src/features/<name>/`.
- The proto → API → CLI → UI vertical-slice shape.

**Connect-RPC is the default transport.** Every domain endpoint goes
through a proto service and generated Connect handlers/clients. If
you find yourself writing `Path: "/api/v1/..."` as a literal string in
an `EndpointDescriptor`, stop — use a proto service method instead.
Codegen rejects literal Paths that lack an explicit `RESTException`
tag; the four allowed REST reasons (multipart upload, webhook
receiver, third-party shape, ops probe) are enumerated in
`api/internal/module/module.go`. The notes attachments endpoint is
the worked REST example.

[`docs/START-HERE.md`](docs/START-HERE.md) describes the replacement
workflow in full.

## Running The Scenario

```bash
# Build API + UI, install pnpm deps, install scenario CLI
make setup   # wraps `vrooli scenario setup`

# Start API + UI in the background
make start   # wraps `vrooli scenario start`
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full clone-to-running flow.

Run tests with `make test` (which runs `vrooli scenario test`) or invoke
`test-genie execute brand-manager --preset comprehensive` directly for
finer-grained presets.

## Documentation Map

| Need | Start Here |
|---|---|
| Initialize after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at this scenario's root |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| Understand the architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Map product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Track workflows, data, and integrations | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md), [`docs/concepts/DATA.md`](docs/concepts/DATA.md), [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Capture monetization and launch strategy | [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md), [`docs/business/GO-TO-MARKET.md`](docs/business/GO-TO-MARKET.md) |
| Prepare deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Write tests | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Add or update seams/fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Configure env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Add API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Add CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |

## Working Rules

1. **Read [`docs/START-HERE.md`](docs/START-HERE.md) first.** It owns the first implementation workflow.
2. **Run `make orient`** as a progress check — it reports initialization gates from `.vrooli/orientation.json`.
3. **Update `PRD.md` and `requirements/`** before feature work. Operational targets drive code + tests.
4. **Read root `DESIGN.md` before UI work.** Tokens, motion, and status semantics are binding; specific component lists in the design are illustrative — implement everything your scenario actually needs.
5. **Update `docs/concepts/DOMAINS.md`** before adding product code.
6. **Keep `docs/manifest.json` accurate.** Durable docs should be registered there with a truthful maturity value.
7. **Append progress entries** to `docs/internal/PROGRESS.md` whenever you land work.
8. **Add resources** to `.vrooli/service.json` only when needed; this scenario ships with no resource dependencies (SQLite is in-process).
9. **Keep boundaries**: only edit within this scenario's directory.

## pnpm Everywhere

This scenario assumes pnpm. If you run another package manager, convert
lockfiles yourself before committing. Scripts use `pnpm` directly (no
`npm` fallbacks) to reduce drift.

## Need Inspiration?

Open `scenarios/browser-automation-studio/` to see the same template
shape taken to completion.
