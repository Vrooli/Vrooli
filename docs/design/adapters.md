# Design Adapters

Design adapters translate a design kit into stack-specific assets. Adapter IDs describe the rendering stack, not just a language.

Good adapter IDs:

- `react-vite-tailwind`
- `react-vite-css`
- `python-pywebview-css`
- `rust-egui`
- `swiftui`

Avoid language-first buckets such as `react/`, `python/`, or `rust/`. The implementation target is normally a stack, framework, and styling approach together.

## Adapter Contract

Each adapter lives under:

```text
templates/design/<kit-id>/adapters/<adapter-id>/
```

Each adapter must include `adapter.json`:

```json
{
  "id": "react-vite-tailwind",
  "requires": "react-vite-tailwind",
  "copy": [
    { "from": "tokens.css", "to": "ui/src/design-tokens.css" },
    { "from": "tailwind.theme.json", "to": "ui/tailwind.theme.json" }
  ]
}
```

Copy sources are adapter-relative. Copy destinations are generated-scenario-relative. The generator validates that neither path escapes its allowed root.
