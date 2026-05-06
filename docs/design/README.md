# Design Language System

Vrooli uses root-level `DESIGN.md` files as the canonical design contract for generated scenarios. New scenario work should read `DESIGN.md` before UI implementation and treat it as the source of truth for layout density, visual tone, tokens, accessibility constraints, and component behavior.

`docs/DESIGN_LANGUAGE.md` is obsolete. Do not create it for new scenarios and do not treat it as a compatibility path.

Vrooli follows the Google Labs `DESIGN.md` direction where it is stable enough to depend on: top-level YAML token groups such as `colors`, `typography`, `rounded`, `spacing`, and `components`, followed by Markdown rationale. Vrooli design kits may also include local extension blocks such as `tokens` and `constraints` when adapters need richer scenario-generation metadata. Official-style top-level token groups are the compatibility layer; Vrooli extension blocks are additive and must not contradict them.

## Structure

- Project-level governance lives in `docs/design/`.
- Reusable design kits live in `templates/design/<kit-id>/`.
- Each design kit has one canonical, stack-agnostic `DESIGN.md`.
- Stack-specific adapter assets live under `templates/design/<kit-id>/adapters/<adapter-id>/`.
- Generated scenarios receive `DESIGN.md` at the scenario root.

The core relationship is:

```text
design kit = design intent
adapter = stack-specific realization
template = consumer that declares which adapter it can apply
```

## Required UX State Contract

Every Vrooli design kit must define how generated UI handles asynchronous and degraded states. This is part of the design contract because silent failure, invisible loading, and ambiguous submits are user-facing design bugs.

At minimum, each kit should document:

- Loading, submitting, syncing, refreshing, empty, partial, stale, success, validation-error, request-error, unavailable, retrying, and disabled states where they apply.
- Immediate feedback for user-triggered network calls, long-running local tasks, generation steps, file operations, and mutations.
- Input preservation and field-level feedback for form failures.
- Retry or next-step affordances for failures the user can act on.
- Freshness, stale-data, and partial-data behavior for dashboards and displays.
- Privacy boundaries for errors, especially avoiding stack traces, secrets, raw tokens, and local paths in public UI.

The canonical prose should live in `DESIGN.md`. Stack-specific primitives, templates, tests, and future auditors should enforce it in implementation.

## Generation

Scenario generation selects a design kit explicitly with `--design <kit-id>` or uses the template default:

```bash
vrooli scenario generate react-vite \
  --id example \
  --display-name "Example" \
  --description "Example scenario" \
  --design vrooli-default
```

The `react-vite` template requires a compatible `react-vite-tailwind` adapter. It defaults to `vrooli-default`, but callers may select a different compatible kit:

```bash
vrooli scenario generate react-vite \
  --id command-display-example \
  --display-name "Command Display Example" \
  --description "Fullscreen command display" \
  --design vrooli-command-display
```

Use `vrooli-default` for normal scenario applications. Use `vrooli-command-display` for fullscreen war-room, kiosk, TV, and ambient display scenarios where the primary experience is an always-on visual dashboard rather than hands-on operational work.

The `landing-page-react-vite` template defaults to `vrooli-conversion-landing`:

```bash
vrooli scenario generate landing-page-react-vite \
  --id conversion-example \
  --display-name "Conversion Example" \
  --description "Conversion landing page" \
  --design vrooli-conversion-landing
```

Use `vrooli-conversion-landing` for pages that sell, validate, or capture demand for a specific scenario, bundle, app, download, demo, waitlist, or offer. It is not the general Vrooli marketing-site design.

## Validation

Local validation checks kit metadata, required files, adapter copy rules, official-style token groups, and the Vrooli UX state contract:

```bash
vrooli scenario design validate --all
```

The local validator is intentionally deterministic and does not require installing the upstream Google CLI. For compatibility checks against the alpha Google tooling, run the upstream linter separately when package installation is available:

```bash
npx @google/design.md lint DESIGN.md
```

## Related Docs

- [registry.md](registry.md) lists available design kits.
- [adapters.md](adapters.md) defines adapter contracts.
- [governance.md](governance.md) defines ownership, versioning, and review expectations.
