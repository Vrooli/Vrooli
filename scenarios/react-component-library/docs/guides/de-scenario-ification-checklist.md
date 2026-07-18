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

## Catalog metadata contract

Ingest scaffolds this contract so a fresh draft lands with the same catalog
fields authored components carry — `slot` and `category` default when omitted,
and each version folder ships a one-story `story.json` stub. Promotion is
not finished until you replace the defaults and the stub with real values:

- **Design affinities.** Declare `designStyles` in `component.json` — one entry
  per relevant style (`vrooli-default`, `vrooli-command-display`,
  `vrooli-conversion-landing`) with an `affinity` of `native`, `compatible`, or
  `discouraged` and a short `reason`. Authored components carry 2-3. A promoted
  component with none reads "No design affinities declared" in the detail view
  and the reindex raises a `missing_design_affinity` conformance finding. Do not
  invent new style ids — an unknown id raises `stale_design_style`.
- **Examples.** Replace the scaffolded `default` stub with 3+ meaningful
  examples that exercise the real prop surface (state, size/variant, header
  slots, callbacks). Empty or single-stub example sets are not catalog-complete.
- **Tags, slot, category.** Confirm `slot` (`ui-primitive`, `ui-pattern`, or
  `layout-nav`), a real `category` in place of the `uncategorized` default, and
  descriptive `tags` so the component is discoverable in search and list views.
