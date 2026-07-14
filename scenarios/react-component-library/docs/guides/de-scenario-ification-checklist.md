# De-scenario-ification checklist

Use this checklist after `react-component-library components ingest` returns a draft.

- Replace hard-coded colors, spacing, radii, and shadows with the catalog token contract.
- Remove scenario aliases, relative application imports, router calls, and ambient context dependencies.
- Make behavior explicit through typed props and callbacks.
- Preserve semantic elements, labels, keyboard behavior, focus treatment, and screen-reader names.
- Add examples for default, constrained, error/disabled, and responsive states.
- Inspect the live preview in light and dark before promoting the draft to a release.
