# De-scenario-ification checklist

Use this checklist after `react-component-library components ingest` returns a draft.

- Replace hard-coded colors, spacing, radii, and shadows with the catalog token contract.
- Remove scenario aliases, relative application imports, router calls, and ambient context dependencies.
- Make behavior explicit through typed props and callbacks.
- Preserve semantic elements, labels, keyboard behavior, focus treatment, and screen-reader names.
- Read the ingest parity report before promotion. Every `behavior-lost` finding
  means the origin exposed a hook, keyboard, ARIA, role, or listener signal
  absent from the harvested source; restore it or record an explicit waiver.
- For a component with relative companion imports, harvest the entry and every
  companion as one version unit. Do not inline or silently replace a hook/store
  just to make a single-file catalog entry compile.
- For subsystem-scale behavior (for example a voice stack), expose a small
  host-provided interface through props rather than importing scenario state.
- Add examples for default, constrained, error/disabled, and responsive states.
- Inspect the live preview in light and dark before promoting the draft to a release.
