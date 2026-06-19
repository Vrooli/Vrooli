# Progress — Tunnel Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-06-18 | regen agent | done | **Phase 0** — Regenerated from `react-vite` 1.1.0 + vrooli-default design kit (old scenario preserved at `/tmp/tunnel-manager-OLD-reference`). Scaffold boots healthy (API+UI), builds green, raw unit/deps phases pass. Replaces the pre-1.0.0 REST/JSON scenario with a Connect-RPC + screaming-architecture foundation. |
| 2026-06-18 | regen agent | done | **Phase 1 (docs-first)** — Authored PRD (12 P0 / 7 P1 / 8 P2 targets; reframed as exposure broker + self-healing control plane), 10-module requirements registry (40 reqs, `prd-control-tower requirements validate` → healthy, 27/27 targets linked), DOMAINS map (7 domains + health), DECISIONS log (12 durable decisions), Gate-4 `service.json` (SQLite, fixed UI port 21240, cloudflared hostTool, optional redis), and all concept/internal/ops/business/reference docs (honest "planned" framing; example fences preserved for Phase 2 detemplate). Plan: `docs/plans/tunnel-manager-regen-adoption-plan.md`. |
| 2026-06-18 | impl agent | done | **Stage 1–2 (backend)** — Proto contracts for all 7 domains (`buf lint` 0, Go+TS codegen clean) + implemented routes/config/audit/tunnel/probes/recovery/exposure across proto→API (Connect + SQLite)→CLI (cli-core, `--json`). Fixed the `.vrooli.com` hardcode (host derives from `route.Domain`). Recovery = live backoff+circuit-breaker (single cloudflared-restart owner); exposure = tiered broker (CORE coreset closure + LEASED TTL; Expose/Extend/Revoke/Reconcile/IsExposed). New seams: `cmdrunner`, Cloudflare `IngressClient`. All Go build/vet/test green. |
| 2026-06-18 | impl agent | done | **Stage 3 (UX, Phases 10–14)** — Reimagined 5-surface UI replacing the placeholder shell: Overview, Exposure (the heart), Recovery & Events, Metrics, Audit. Shared primitives (`QueryState`, `StatusBadge`), per-surface i18n (en/ja/ar parity), document titles, a11y (axe green), React-Router v7 future flags. 180 vitest tests, coverage gate green, tsc + eslint clean. |
| 2026-06-18 | impl agent | done | **Stage 4 (Phases 15–16)** — `detemplate` removed the `notes` example (Gate 7 ✓); reconciled all 40 requirement `validation.ref`s to real tests (TIER-005 budget / TIER-006 idle-spindown honestly marked `not_implemented`, P2 out-of-scope). Hardening: width-bounded `ParseInt` (gosec G109 ×4), security-headers middleware + secured REST error writer, pnpm `minimumReleaseAge`. **`vrooli scenario test` = 15/18 phases green; completeness 83/100 (nearly_ready).** Remaining 3 reds (standards/tidiness/proto) are react-vite **template/fleet debt** — see PROBLEMS.md. Stage 5 (live CF adoption) is operator-attended, not started. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
