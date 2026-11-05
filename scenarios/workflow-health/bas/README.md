# BAS Automation

Store automation workflows here. Keep it short:

- `cases/<operational-target>/<surface>/` mirrors operational targets (rename folders as needed).
- `flows/` contains multi-surface user flows.
- `actions/` hosts fixtures referenced via `@fixture/<slug>`.
- `seeds/` includes optional setup scripts when deterministic data is required.

Each workflow JSON must include:

```json
{
  "metadata": {
    "description": "What the workflow validates",
    "requirement": "REQ-ID",
    "version": 1
  }
}
```

Reference selectors via `@selector/<key>` from `ui/src/consts/selectors.ts`. After adding or moving a workflow, run from the scenario directory:

```bash
test-genie registry build
```

This regenerates `bas/registry.json`, which is tracked so other agents can see which files exist, which requirements they validate, and what fixtures they depend on. (Only `bas/cases/**` are executed by the Playbooks phase — `flows/` and `actions/` are reusable building blocks.)

## Performance-capture flows (`intent: "performance"`)

A `bas/flows/` entry can double as a **performance-capture target** for the
`performance-health` scenario. Mark it `metadata.labels.intent: "performance"`
and keep it **assertion-free** — it only drives an interaction so a perf trace
can span it. The loop:

```bash
# 1. Author bas/flows/<slug>.json (intent:performance, no asserts).
#    Use literal [data-testid=...] selectors — @selector tokens do NOT resolve
#    on the capture path.
# 2. Drive it inside a profile-mode perf trace:
performance-health audit run <scenario> --workflow <slug>
# 3. Analyze the returned trace, then optionally set a per-flow budget:
performance-health analysis analyze --trace <key>
performance-health budget set --flow <slug> --lcp-max-ms 2500 --ratchet
```

Reusable perf interaction helpers live in `actions/`:

- `perf-scroll-ancestor` — scroll a virtualized list's nearest scrollable ancestor.
- `perf-drag-horizontal` — stepped horizontal drag of a resize handle / divider.

See `bas/flows/perf-example-scroll.json` for a starting point.

> **Rule:** NEVER bind a perf flow as a requirement `automation` validation —
> that would run it in the functional Playbooks pass/fail suite. Perf flows are
> continuous, out-of-band capture targets, not assertions.
