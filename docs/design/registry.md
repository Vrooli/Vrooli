# Design Kit Registry

This registry mirrors design-kit metadata under `templates/design/`. Use the CLI as the deterministic source of truth:

```bash
vrooli scenario design list
vrooli scenario design show vrooli-default
vrooli scenario design validate --all
```

## Kits

| ID | Name | Version | Default | Adapters | Intended Use |
| --- | --- | --- | --- | --- | --- |
| `vrooli-default` | Vrooli Operational Console | `0.1.0` | yes | `react-vite-tailwind` | Dense, responsive, customizable operational UI for generated Vrooli scenarios. |

## Rules

- A kit is stack-neutral design intent.
- A kit owns exactly one canonical `DESIGN.md`.
- Adapters may translate the kit for specific stacks, but they do not define separate design languages.
- Only one kit may be marked as the default.
