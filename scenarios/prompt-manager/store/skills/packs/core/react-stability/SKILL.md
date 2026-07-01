## Steer focus: React Stability — CONSOLIDATED into `ui-health`

> **This skill has been consolidated into the `ui-health` steer skill.** Its crash-prevention rubric (strict-config, stability ESLint rules, hook discipline, defensive data access, error boundaries, runtime validation at boundaries) is now owned by the `ui-health` provider (`standard_tsconfig_strict`, `standard_eslint_stability`, and the `runtime_*` render group) and its remediation guidance lives as lenses §4.9–§4.10 of the consolidated skill.

**Load this instead:**

```bash
prompt-manager skill read ui-health
```

**Validate stability via the provider:**

```bash
# Static strict-config + stability-lint checks:
ui-health validate scenario <scenario> --static-only --json
# Live render / console-error / crash evidence (drives the UI through BAS):
ui-health validate scenario <scenario> --json
# Auto-fix the safe subset (e.g. tsconfig strict false→true):
ui-health fix run <scenario> --rule standard_tsconfig_strict --apply
```

Stability = crash prevention (null guards, hook discipline, error boundaries). For deployment-context correctness see the interop lenses; for stylistic judgment see the separate `ux` skill.
