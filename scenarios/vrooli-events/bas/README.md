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

Reference selectors via `@selector/<key>` from `ui/src/consts/selectors.ts`. The
test suite discovers workflows from this directory when it runs. Keep
`metadata.requirement` and selector references valid so the generated test
inventory can show which requirements and fixtures each workflow covers.
