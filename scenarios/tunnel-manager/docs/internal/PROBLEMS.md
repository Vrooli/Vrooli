# Problems — Tunnel Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-06-19 — Production readiness truth map gaps

**Symptom:** Fresh Tunnel Manager instances could report or imply remote-mode readiness even when Cloudflare credentials were absent; some docs still described Phase 1 planned behavior while the PRD marked P0/P1 targets complete; probe/recovery scheduling and UI flows needed reconciliation with live behavior.

**Root cause:** The regenerated implementation landed broad API/CLI/UI surfaces before a production truth pass. Config docs used canonical `CLOUDFLARE_*` names while runtime code read `CF_*`; the config service defaulted to remote mode; exposure/recovery/probe scheduling and UI setup workflows needed to be reconciled against the product contract.

**Workaround:** Treat config readiness as the first source of truth. As of the 2026-06-19 config pass, canonical `CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_TUNNEL_ID`, and `CLOUDFLARE_API_TOKEN` are supported, fresh config defaults to local mode, and `config sync --dry-run` reports missing remote setup instead of failing when remote credentials are absent. As of the 2026-06-20 Greenfield config/secrets slice, legacy `CF_*` names are intentionally not accepted and credential resolution uses canonical env override, then scenario-scoped and shared user secret files under the operator runtime home. As of the follow-up exposure passes, `exposure` composes the same production config-service builder as `config`, so remote-mode `Expose`/`Reconcile` use the configured Cloudflare ingress client instead of an unwired config service; the API also starts a cancellable exposure scheduler that runs CORE reconcile and expired-lease reaping at boot and on `TUNNEL_MANAGER_EXPOSURE_RECONCILE_INTERVAL`. As of the probe/recovery pass, probes run at boot and periodically by default (`TUNNEL_MANAGER_PROBE_INTERVAL`), while background recovery evaluation is implemented but opt-in (`TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED`) because an acted evaluation restarts cloudflared. As of the UI redesign slices, Overview shows config readiness/mode/missing fields; Settings owns local/remote mode, Cloudflare readiness, sync preview, and sync apply; Exposure owns search, CORE/LEASED filtering, reconcile feedback, lease actions, and route-classification badges; Diagnostics/Metrics owns latest tunnel metrics, route classification counts, diagnostic-signal limits, probe history, and manual scrape/probe actions; Audit owns fixed-port compliance summaries, status filtering, and remediation hints; and Recovery owns the state-machine summary, breaker/backoff risk, next operator action, manual force warning, and event details.

As of the Phase 7 follow-up reconciliation, PRD/requirements no longer mark the full OT-P1-001 scope as complete: current probe-pair classification is implemented and tested, while DNS-failure and Cloudflare-outage isolation are explicitly tracked as not implemented until resolver/upstream signals exist.

**Real fix:** Remaining production-readiness work is advisory/deferred: implement richer DNS/Cloudflare outage classification signals when resolver/upstream inputs exist, integrate scenario-authenticator aud-scoped tokens before granting direct privileged mutation access to non-operator cross-scenario callers, and validate live Cloudflare behavior with an attended operator run. Time-series/history retention is no longer TBD: metrics/probes keep a rolling 14-day window, and recovery events keep a rolling 90-day window. Service-layer static-token authz is no longer TBD: `TUNNEL_MANAGER_AUTHZ_ENFORCED=1` fail-closes privileged mutation RPCs behind `TUNNEL_MANAGER_OPERATOR_TOKEN` or fallback `API_TOKEN`.

**Owner:** unassigned.

**Refs:** `api/internal/config/{types.go,cfclient.go,service.go,production.go}`, `api/handlers/{config,exposure,probes,recovery}/module.go`, `api/internal/{exposure,probes,recovery}/scheduler.go`, `packages/proto/schemas/tunnel-manager/v1/config/config.proto`, user plan `tunnel-manager-production-readiness-redesign`.

### 2026-06-18 — Superseded: product implementation was not yet built during docs-first phase

**Symptom:** Historical during the regeneration docs-first phase: API/CLI/UI described domains and endpoints that did not exist as code yet.

**Root cause:** Intentional. Phase 1 is documentation-first (charter → requirements → domain map → docs). Implementation is Phase 2.

**Workaround:** Superseded. Product domains are now implemented and validation-green; keep this entry only as historical context for why early docs were planned-contract heavy.

**Real fix:** Done by the implementation and validation-hardening slices recorded in `PROGRESS.md`.

**Owner:** unassigned. **Refs:** `docs/plans/tunnel-manager-regen-adoption-plan.md`, `PRD.md`.

### 2026-06-18 — Superseded: `make test` reported fleet-reds from template/example content

**Symptom:** Historical during regeneration: `make test` exited 1 with dependencies/unit/tidiness ERROR findings even though raw unit and dependencies phases passed.

**Root cause:** test-genie fleet analysis flags the template's own example/scaffold content — `notes` domain (`TEST_HELPER_FROM_PRODUCTION`, duplicated blocks, low coverage), formal-flow testutil cyclomatic complexity, the UI coverage gate (App.tsx/profiler 0%), and the pnpm `minimumReleaseAge` policy. None originate from this regen.

**Workaround:** Superseded. `vrooli scenario test tunnel-manager` now passes 18/18 phases.

**Real fix:** Done by detemplate, real domain coverage, pnpm `minimumReleaseAge`, and validation-hardening slices.

**Owner:** unassigned. **Refs:** `coverage/latest/findings.json`.

### 2026-06-18 — `prd-control-tower` generate/validate blocked

**Symptom:** `prd-control-tower prd generate` returns `ORPHANED_CRITICAL_TARGETS`; `prd validate` returns `blocked`.

**Root cause:** Known prd-control-tower issue (also hit by image-tools). Unrelated to PRD content.

**Workaround:** PRD authored directly to the canonical v2.0 template (the documented fallback); the orientation charter gate (placeholder-absence) passes. `requirements validate` works and returns healthy.

**Real fix:** Re-run `prd-control-tower` once the tool issue is resolved; until then the hand-authored PRD is authoritative.

**Owner:** unassigned. **Refs:** `PRD.md`.

### 2026-06-18 — Cloudflare hostname cap unconfirmed

**Symptom:** The exact maximum public-hostname count per tunnel is unknown (operator estimate ~100 via dashboard; docs don't state a hard cap).

**Root cause:** Cloudflare docs cover ingress config shape, not limits; the dashboard limit likely differs from API/config-managed limits.

**Workaround:** Tiered exposure (core + leased) is cap-robust regardless. Hostname-budget management is parked at OT-P2-001.

**Real fix:** Phase 3 — confirm the real cap against the live Cloudflare plan; promote OT-P2-001 to P0 if the cap is low.

**Owner:** unassigned. **Refs:** `PRD.md` (note under P2).

### 2026-06-19 — SUPERSEDES the residual-red entry above: validation gates are green

**Symptom:** The prior 15/18 suite state is resolved. `vrooli scenario test tunnel-manager` now reports **18/18 phases green** with completeness 85/100 (`nearly_ready`). The previously red `standards`, `tidiness`, and `proto` phases pass.

**Root cause:** The failures were a mix of stale generated metadata, analyzer-recognition shape, and local maintainability debt:
- **standards:** the security-header analyzer only credits literal `w.Header().Set(...)` calls in files that write responses. The REST error writer set the right headers through a local header variable, so the analyzer still flagged it.
- **tidiness:** the scanner blocked on high-complexity/high-duplication helper code and repeated manifest coverage test scaffolding.
- **proto:** `proto-health` was still serving stale tunnel-manager surface data that included the removed `NotesService`.

**Fix shipped:** Regenerated tunnel-manager endpoint metadata, refreshed proto descriptor inputs, restarted `proto-health`, changed the REST error writer to analyzer-visible header calls, extracted the shared scheduler loop, deduped CLI manifest service-coverage tests, and simplified high-complexity/high-duplication test helpers.

**Remaining advisory debt:** Standards still reports medium/low warnings (env validation heuristics, hardcoded local values, root-health heuristic, test-file warnings). Proto still reports warnings for template-sourced `errors.proto`/`health.proto` and unsupported REST proof for `/health`. These do not block the suite.

**Baseline:** `git-control-tower baseline diff --scenario tunnel-manager --name tunnel-manager-production-readiness-redesign --wait` returned `Overall: preexisting`: standards cleared, no regressions, and only an inherited smoke baseline failure remains.

**Owner:** tunnel-manager maintainers for advisory cleanup; fleet/tooling owners for analyzer/template warning polish. **Refs:** `api/internal/httpx/errors.go`, `api/internal/scheduler/loop.go`, `cli/internal/manifesttest/manifesttest.go`, `packages/proto/gen/descriptor/image.binpb`, `.vrooli/endpoints.json`.

### 2026-06-20 — Credential setup API/UI still needs dynamic re-resolution

**Symptom:** Tunnel Manager now has a Greenfield credential store backend, but operators still cannot provision Cloudflare credentials through dedicated CLI/UI setup commands.

**Root cause:** The first consolidation slice deliberately stopped at the backend seam and production resolver. The config service still constructs the Cloudflare ingress client at service-build time, so adding write RPCs before dynamic re-resolution would save credentials but could leave remote sync using the old nil/stale client until restart.

**Workaround:** Local mode remains fully usable without Cloudflare credentials. Remote-mode credentials can be provided through canonical `CLOUDFLARE_*` env overrides, or through the operator runtime-home secret files used by the new `CredentialStore` backend.

**Real fix:** Continue `tunnel-manager-greenfield-config-and-secrets-consolidation`: add dynamic credential/ingress resolution inside the config service, then expose write-only credential status/set/clear/validate RPCs with CLI and UI setup flows.

**Owner:** unassigned. **Refs:** `api/internal/config/credentials.go`, `api/internal/config/production.go`, plan `tunnel-manager-greenfield-config-and-secrets-consolidation`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
