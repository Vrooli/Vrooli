# Progress — Security Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-01 | Claude (agi) | done | Generated scenario from react-vite (vrooli-default). Authored PRD (13 operational targets, validate=healthy) + requirements registry (13 modules, 19 reqs, 13/13 linked). Defined the architectural core: three Connect-RPC proto domains (`validation`, `dependencies`, `reindex`) under `packages/proto/schemas/security-health/v1/`, regenerated all trees, wired `qdrant`+`ollama` optional resource deps, wrote DOMAINS.md + INTEGRATIONS.md. API builds green. |
| 2026-06-01 | Claude (agi) | done | **Phase C — validation core.** `internal/validation`: substrate detector + Scanner interface + 5 runners (gitleaks/gosec/govulncheck/pnpm-audit/osv-scanner) behind a `Commander` exec seam; severity normalization (critical/high→ERROR, moderate→WARNING, low/info→INFO). `ValidationService.ValidateScenario` Connect handler + `security-health validate scenario <name> --json` CLI. Live-verified against the real scanners (all 4 working tools produce findings). Precise gating: govulncheck stdlib→WARNING (toolchain-only fix), osv-scanner→advisory INFO (no reachability), pnpm-audit dev-deps→WARNING (prod/dev split), gosec G115/G122/G703 over-firers→WARNING. security-health validates **itself** as passed=true. Installed gosec(latest)/govulncheck/osv-scanner with user approval. |
| 2026-06-01 | Claude (agi) | done | **Phase D — producer loop.** Added `FINDING_SOURCE_SECURITY=9` to findings.proto (regen all trees). test-genie `security` phase (`phase_security.go`, Optional, comprehensive preset, 180s) delegates to the CLI `--json`, maps to `FINDING_SOURCE_SECURITY`, skips gracefully when the producer is unavailable. EM `dimensions.json` source+phase maps + anti-drift fixture updated. `defaultEffortForSource(SECURITY)→MEDIUM`. doc `docs/phases/security/README.md`. test-genie + EM build/test green. |
| 2026-06-01 | Claude (agi) | done | **Phase E — dependency & vulnerability intelligence.** `internal/dependencies`: SQLite SBOM corpus, fleet discovery (go.mod via x/mod, pnpm-lock via yaml), osv-scanner vuln annotation, TEXT + structured-filter search (`--ecosystem`/`--vulnerable-only`/`--name-glob`), async reindex jobs (status/cancel) + 5-min reconcile loop. `DependencyService.Search/Status` + `ReindexService` handlers + `deps`/`reindex` CLI. Full test coverage. **AI/Qdrant semantic mode deferred** (MODE_AI degrades to TEXT) — see PROBLEMS.md. |
| 2026-06-01 | Claude (agi) | done | **Phase F — security console UI.** Replaced the placeholder routes with Posture (`features/posture`: scan a scenario via `ValidationService`, severity-grouped findings + remediation + pass/fail verdict + skipped-scanner list), Dependencies (`features/dependencies`: SBOM search with ecosystem + vulnerable-only filters over `DependencyService`, status strip, text-mode hint), and Secrets (PostureCard filtered to the `gitleaks` scanner + redaction note). Added the embeddable `PostureBadge` (`@vrooliWidget` slot INLINE) severity-rollup widget. New api clients `api/validation.ts` + `api/dependencies.ts`; severity→color/order contract in `features/posture/severity.ts`. Nav/routes/selectors/i18n (en/ja/ar) retargeted. 140 UI tests + `tsc` + `vite build` green. Incidental fixes (pre-existing scaffolding): router v7 future flags (test-setup throws on `console.warn`), distinct sidebar/bottom-nav landmark labels (`landmark-unique`), static `THEME_LABEL` map (unused-key audit). |
| 2026-06-01 | Claude (agi) | done | **Gate 7 — example domain removed.** Deleted the `notes`/attachment reference domain from API (`handlers/notes`, `internal/notes`, `main.go` + `internal/modules/registry.go` registration), CLI (`domains/notes`, `domains.go` + `cli/manifest.json` group + gen-endpoints seed), UI (`features/notes`, `api/notes.ts`, pages, selectors, strings), and the `security-health/v1/notes` proto (schema + all gen trees). Regenerated `.vrooli/endpoints.json`. `vrooli scenario orient` → `example-domain-removed` passes (5/5 globs absent); 7/8 required gates green (only `scaffold-health`/`make test` remains — see PROBLEMS.md). API+CLI+UI build & test green. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
