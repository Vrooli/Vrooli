# codex-rollout

Reads Codex native rollout records and reports the operational token usage they contain
(input, cached input, output). The figures are observed usage, neither an invoice nor a
statement of subscription quota. Record lines are bounded by `MaxLineBytes`.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
