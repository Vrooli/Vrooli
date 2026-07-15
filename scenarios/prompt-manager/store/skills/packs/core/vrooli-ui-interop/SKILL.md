## Steer focus: Vrooli UI Interop — CONSOLIDATED into `ui-health`

> **This skill has been consolidated into the `ui-health` steer skill.** Its compliance rubric is now owned by the `ui-health` provider (the rule engine + `scenarios/ui-health/.vrooli/maturity.json`), and its remediation guidance — the convergence slot pattern, iframe-safe scroll/viewport, and spatial navigation — lives as lenses §4.2–§4.4 of the consolidated skill. The old `app-monitor interop` / `app-monitor rules` commands referenced here no longer exist; UI-interop validation moved to ui-health.

**Load this instead:**

```bash
prompt-manager skill read ui-health
```

**Verify interop compliance via the provider (replaces the old `app-monitor interop <scenario>`):**

```bash
# Static report including every interop_* finding (no browser):
ui-health validate scenario "<scenario>" --static-only --json

# Preview/apply the safe-subset interop fixers:
ui-health fix run "<scenario>" --json
```

ui-health is the single authority for UI-interop checks (deployment-context correctness across localhost / Cloudflare tunnel / app-monitor proxy-iframe). The `ui-health` skill's decision model routes each `interop_*` finding to its remediation lens.
