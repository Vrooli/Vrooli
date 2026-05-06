# Design Language System

Vrooli uses root-level `DESIGN.md` files as the canonical design contract for generated scenarios. New scenario work should read `DESIGN.md` before UI implementation and treat it as the source of truth for layout density, visual tone, tokens, accessibility constraints, and component behavior.

`docs/DESIGN_LANGUAGE.md` is obsolete. Do not create it for new scenarios and do not treat it as a compatibility path.

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

## Related Docs

- [registry.md](registry.md) lists available design kits.
- [adapters.md](adapters.md) defines adapter contracts.
- [governance.md](governance.md) defines ownership, versioning, and review expectations.
