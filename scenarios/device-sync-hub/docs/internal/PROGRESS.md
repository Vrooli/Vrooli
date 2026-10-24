# Progress — Device Sync Hub

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/device-sync-hub/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-18 | codex | Cleared disposable local first-run hub state after backing up `data/device-sync-hub.db` to `data/backups/device-sync-hub.db.pre-owner-reset-20260618T223120Z` (`hub_owner` and `devices` now empty; authenticator accounts left intact). | Archived |
| 2026-06-17 | agi | First-run owner bootstrap + production-ready UI | Archived |
| 2026-06-17 | agi | Phase 5 (polished split-screen UI). | Archived |
| 2026-06-17 | agi | Phase 4 (CLI surface + programmatic compound-value seam). | Archived |
| 2026-06-17 | agi | Phase 6 (testing/validation/hardening) — most blockers resolved. | Archived |
| 2026-06-17 | agi | Phase 1 (greenfield rewrite — generate + charter + docs). | Archived |
| 2026-06-17 | agi | Phase 2 (API foundation + auth integration + devices/pairing/trust domain). | Archived |
| 2026-06-17 | agi | Phase 3 (transfer domain + retention + presence + quotas + realtime). | Archived |
| 2026-06-17 | agi | Phase 2 (API foundation + auth integration + devices domain). | Archived |

## Current State & Phase 6 Handoff

**DONE through Phase 5:** API (P2), transfer+realtime (P3), CLI (P4), and now the
**polished split-screen UI (P5)** — see the 2026-06-17 Phase 5 row. The product is
end-to-end drivable from a browser: join (redeem/request) → split-screen Transfer
(Receive top / Send bottom) → device management + pairing-code+QR issuance → realtime
presence/item/pairing events. All five UI DoD gates are green from `ui/`:
`strings:check`, `type-check`, `lint` (0 errors), `test` (163 passed), `build`.

**NEXT — Phase 6 (testing / validation / security hardening):** run the full
`git-control-tower baseline diff --scenario device-sync-hub --name dsh-rewrite`
(blocked in *this* environment by the unhealthy `scenario-authenticator` sibling —
DEPENDENCIES phase; pre-existing/environmental, not a code defect); exercise the
scaffold-health + e2e + smoke phases; security review of the dual-credential UI
(token storage in localStorage, owner-token paste, download/upload header handling).

**Phase 6 carve-outs / known gaps to harden or finish (deferred by plan, not P5 bugs):**
- **Owner login is token-paste only.** `OwnerSignIn` accepts a pasted owner JWT; a
  real login form posting to scenario-authenticator is deferred (the transfer core
  works with only a device token, so this is secondary).
- **QR is best-effort, not verified-scannable.** `lib/qr.ts` is a dependency-free
  byte-mode/level-L encoder; tests assert structural invariants, not decode
  round-trips. The code is always also shown as large copyable text.
- **Realtime is single-connection per tab**; no multi-tab coordination.
- **localStorage token storage** — fine for v1; revisit if a stricter storage policy
  is required.
- **4 pre-existing lint warnings** (`react-refresh/only-export-components`) on
  provider/context files; these are scaffold-pattern warnings, not errors, and the
  lint gate is green.

**Final Cleanup (later, per plan §7 Final):** remove the **notes backend + proto** and
the `/measures` mount once a real domain (transfer) owns a measure; add the dedicated
`settings` domain (owner-tunable retention default / quotas / at-rest-encryption
toggle), chunked/resumable upload (OT-P1-001), and at-rest blob encryption (OT-P1-007).
The Phase 5 UI already removed every *UI* reference to notes — only the backend
removal remains.

---

## Superseded — Phase 5 Handoff (pre-Phase-5)

**DONE through Phase 4:** charter+docs (P1), auth integration + devices/pairing/trust
(P2), the transfer + realtime domains with retention, quotas, presence, and
device-token trust (P3), and the **CLI surface + programmatic compound-value seam**
(P4 — see the 2026-06-17 Phase 4 row above). The API is feature-complete for the P0
server-relay path (pair → push file/text → pull on another device → retention purge →
revoke severs access) plus realtime presence/events, and is now fully drivable from
the CLI. `go build`/`vet`/`gofumpt`/`golangci-lint` and `go test ./...` (and `-tags
e2e`) are green across **both** the `api/` and `cli/` modules.

**RESOLVED — `cmd/gen-endpoints` crossCheck (was the one red through Phase 3):**
Phase 4 built `cli/domains/devices/` + `cli/domains/transfer/`, registered them in
`cli/domains/domains.go`, and regenerated `cli-commands.gen.json` +
`.vrooli/endpoints.json` via `make endpoints`. The API↔CLI parity crossCheck
(`api/cmd/gen-endpoints` test + `cli` `TestAPICLIParity`) is now **green** — every
manifest binding resolves to a registered command. The transfer CLI doubles as
the **programmatic delivery seam** other scenarios call ("deliver an artifact to a
device"), with `--json` proto-wire-shape output for scripting.

**Other still-open items (deferred by plan, not Phase-4 gaps):**
- `example-domain-removed` gate + the `notes` worked example: removed in **Final
  Cleanup** (plan §7 Final), once a real domain owns the `/measures` mount. Today
  `notes` still owns `/measures` in `main.go`; transfer has no measure yet. Before
  deleting `notes`, move a measure (e.g. `items.count`) onto a transfer measures
  registry so `main.go` is never left with a dangling mount.
- A dedicated **`settings`** domain (owner-tunable retention default / quotas /
  at-rest-encryption toggle) is in DOMAINS but not built — Phase 3 uses
  `transfer.Config` defaults (24h Held / 10-min Live / 5 GiB owner / 2 GiB device /
  1 MiB text). Build alongside the UI (Phase 5) or as a P3 follow-up.
- **Chunked/resumable upload** (OT-P1-001) not yet implemented; streaming multipart
  upload handles large files via temp-file spillover (the P0 path). Add the
  resumable-session endpoint in a P1 pass.
- **At-rest blob encryption** (OT-P1-007) toggle not yet implemented (P1).

**Deferred verification:** the full `git-control-tower baseline diff --scenario
device-sync-hub --name dsh-rewrite` (plan §6a/§10) was NOT re-run — its
DEPENDENCIES phase errors on the unhealthy `scenario-authenticator` sibling in THIS
environment (pre-existing, environmental). Phase-3 code correctness is covered by
the green `go test ./...` + e2e + lint. Run the baseline diff once the sibling is
healthy (or accept the known environmental DEPENDENCIES error when triaging).

**NEXT — Phase 5 (UI, per the plan §7 Phase 5):** replace the placeholder react-vite
shell with the polished, responsive, accessible split-screen UI — **Send (bottom) /
Receive (top)**, distinct accent colors. Send: drop/pick files or paste text → staged
cards with thumbnails + per-item retention/target controls + a Send button (calls the
REST multipart upload + `CreateTextItem`). Receive: empty-state when idle; fills with
downloadable cards/list (toggle view), search/sort/filter; owner-origin items expose
edit/remove; permissioned clear-all. Pairing UX: show QR/code (`IssuePairingCode`) +
an approve banner for inbound `RequestPairing` (accept-once / reject / reject-forever),
fed by the realtime SSE stream (`/api/v1/realtime` events: item-arrived/deleted,
presence-changed, pairing-requested). Settings: retention default, quotas,
at-rest-encryption toggle, device management (list/rename/revoke/sign-out), trust list.
Keep the durable START-HERE seams: design tokens, i18n (`SUPPORTED_LOCALES` /
`useTranslation`), a11y primitives (`role`/`aria-*`/`data-testid`), feature-folder
pattern; handle loading/error/empty everywhere. The TS client targets are the same
generated proto types the CLI/Go clients use (`packages/proto/gen/typescript/
device-sync-hub/v1/...`) plus the two REST byte edges. NOTE: the browser cannot send a
custom `X-Device-Token` header on an `EventSource`, so the SSE stream accepts the
device token via the `?token=` query param (already supported by `internal/deviceauth`).

## Superseded — original Phase 2 Handoff

**What is DONE (Phase 1):**
- Scaffold regenerated (react-vite + vrooli-default). `make setup` green.
- `PRD.md` published + validates ("No structural or linkage issues").
- `requirements/` 15 modules / 23 reqs, validates ("All requirements properly linked").
- Dependency decisions in `.vrooli/service.json`.
- `docs/concepts/DOMAINS.md` rewritten with the real domain map + ecosystem-fit.
- Concept docs (ARCHITECTURE/DATA/FLOWS/INTEGRATIONS) describe the intended design.
- Regression baseline `dsh-rewrite` captured (git-control-tower).

**Orientation gates:** 6/8 pass. The two open are **expected later-phase work**, not Phase-1 gaps:
- `example-domain-removed` — remove the template `notes` domain only AFTER the first real domain (`devices`) is green (Phase 2).
- `scaffold-health` (`make test`) — see residual failures below.

**Residual `make test` failures (all pre-existing scaffold state or environmental — none introduced by Phase 1):**
- **DEPENDENCIES (2 ERROR):** `scenario-authenticator` is unhealthy in THIS environment — missing binary `api/scenario-authenticator-api` + non-idempotent schema (`relation "idx_users_email" already exists`). Environmental; filed as an out-of-scope bug. Will pass once that sibling is built/healthy.
- **PROTO (4 ERROR):** scaffold example endpoints `health` + `notes_attach` lack proto-payload declarations/implementation proof. Resolved in Phase 2 (remove `notes`; give `health` + real domains proto-backed implementations).
- **STRUCTURE (1), UNIT (2):** fresh-scaffold/notes-example state; resolve as real domains land + `notes` is removed.
- **STANDARDS (high — "Security Headers"):** the stub API lacks security-headers middleware; add it when building the real API (Phase 2). Criticals (P0-missing-requirements) are already FIXED by the charter.

**NEXT — Phase 2 (per the plan `plan-manager plans render device-sync-hub-greenfield-rewrite-cross-device-file-text-transfer`):**
1. Build the API foundation + `auth` integration (scenario-authenticator client, fail-closed) + `devices` domain (devices/pairing_codes tables, code/QR pairing, request→approve, revocation). Wire `api-core/storage` + `blobstore` seams.
2. Prove `devices` green, THEN remove the `notes` example (clears `example-domain-removed` + most PROTO/STRUCTURE/UNIT residuals).
3. Add security-headers middleware (clears STANDARDS high).

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-06-17 — First-run owner bootstrap + production-ready UI**: GCT `baseline diff` CLI itself was blocked by another agent's transient uncommitted root-CLI WIP (`internal/cli/scenariocli/detemplate.go` false unused-import), so the afid comparison was run directly. **Authenticator:** fixed its documented schema-idempotency blocker (19 non-idempotent `CREATE INDEX`→`IF NOT EXISTS`); a deeper DB-connection-backoff hang remains → live cross-scenario hop deferred, bug filed `knw-1781722385713679749`, bootstrap logic fully proven against auth stubs/fakes. rec-e3c9d930bb5830da.

- **2026-06-17 — Phase 6 (testing/validation/hardening) — most blockers resolved.**: **DEFERRED/ENVIRONMENTAL:** (a) comprehensive `test-genie` run + GCT baseline diff — test-genie was under concurrent modification by another agent (unreliable); re-run when stable; (b) scenario-authenticator fails its own health check (sibling DB/migration env issue, out of scope per plan) → deps-phase runtime + baseline diff blocked; (c) 23/23 requirements still lack `[REQ:ID]` test tags (warning); (d) Go handler-wiring LOW_COVERAGE warnings (non-gating). **DEFERRED BY DESIGN (user decision):** P1s NOT built — settings domain, chunked/resumable upload, at-rest encryption (PRD P1/post-launch).

- **2026-06-17 — Phase 2 (API foundation + auth integration + devices domain).**: Built `api/internal/devices/` (types/schema/service/repository/sqlite/secrets/error-mapping + mocks): device registry, single-use conditional pairing-code claim, hub device-token issuance (SHA-256-hashed at rest), trust lifecycle pending→trusted→revoked, owner-scoped queries, best-effort authenticator session revoke. **All green:** `go build ./...`, `go test ./...` (incl. proto-connect parity + sqlite round-trips), `-tags e2e`, `golangci-lint` on new packages, proto module compiles. rec pending.
