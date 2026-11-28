# Python uv + React + Vite Template Changelog

This changelog records changes to `templates/scenarios/python-uv-react-vite`.
It is not copied into generated scenarios (see `template.json::copyExcludes`).
Generated scenarios record their origin template and version in
`.vrooli/service.json::generation.template`. The template inherits most files
from `react-vite` through `baseTemplate`; read `react-vite/CHANGELOG.md` for
changes to inherited files. Versioning follows the rules in
`scenarios/template-manager/docs/factory/TEMPLATE-MAINTENANCE.md`.

## 1.0.1 — 2026-11-07

Generated scenarios can build their inherited Go CLI.

### Changed
- Template Manager now inherits a base template's relocations for the paths a
  child inherits. This template therefore relocates react-vite's `proto/`
  schemas into `packages/proto/schemas/<scenario>/` and runs codegen. Before
  this, the inherited CLI imported proto packages that were never generated.
- `.vrooli/service.json` emits `"dependencies": {"resources": {}}`, which the
  service schema accepts. The old value, `{}`, failed schema validation.
- Generated Go modules are converged through
  `scenario-dependency-analyzer deps reconcile --module <dir> --apply`.

### Migration (for agents updating older scenarios)
- [ ] If the scenario's CLI imports `packages/proto/gen/go/<scenario>/...`, make sure
      `packages/proto/schemas/<scenario>/` exists. Copy it from the template's
      base `proto/` with placeholders rendered, or regenerate the scenario.
      Then run `(cd packages/proto && make generate)`.
- [ ] Change `.vrooli/service.json` `dependencies` to `{"resources": {}}` if
      it is `{}`.
- [ ] Run `scenario-dependency-analyzer deps reconcile --scenario <name> --apply`.
