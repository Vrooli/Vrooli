## Steer focus: React Coherence — CONSOLIDATED into `ui-health`

> **This skill has been consolidated into the `ui-health` steer skill.** Its coherence guidance (scope-driven state architecture, the sharing decision tree, styling/theming organization, design tokens, theme-refresh readiness) lives as lenses §4.2 and §4.6 of the consolidated skill; the design-token rubric is now owned by the `ui-health` provider (`standard_no_raw_hex` and the theming checks).

**Load this instead:**

```bash
prompt-manager skill read ui-health
```

**Validate coherence/theming via the provider:**

```bash
ui-health validate scenario <scenario> --static-only --json
```

Coherence = code organization, state management, and styling/theming structure. For deployment-context correctness see the interop lenses; for stylistic/visual judgment see the separate `ux` skill.
