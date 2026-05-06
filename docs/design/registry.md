# Design Kit Registry

This registry mirrors design-kit metadata under `templates/design/`. Use the CLI as the deterministic source of truth:

```bash
vrooli scenario design list
vrooli scenario design show vrooli-default
vrooli scenario design show vrooli-command-display
vrooli scenario design show vrooli-conversion-landing
vrooli scenario design validate --all
```

## Kits

| ID | Name | Version | Default | Adapters | Intended Use |
| --- | --- | --- | --- | --- | --- |
| `vrooli-default` | Vrooli Operational Console | `0.2.0` | yes | `react-vite-tailwind` | Dense, responsive, customizable operational UI for generated Vrooli scenarios. |
| `vrooli-command-display` | Vrooli Command Display | `0.2.0` | no | `react-vite-tailwind` | Fullscreen war-room, kiosk, TV, and ambient command-center displays. |
| `vrooli-conversion-landing` | Vrooli Conversion Landing | `0.2.0` | no | `react-vite-tailwind` | High-converting landing pages for scenarios, bundles, apps, downloads, demos, and waitlists. |

## Rules

- A kit is stack-neutral design intent.
- A kit owns exactly one canonical `DESIGN.md`.
- Adapters may translate the kit for specific stacks, but they do not define separate design languages.
- Only one kit may be marked as the default.
- Add a new kit only when the product intent or interaction model differs materially from existing kits.
