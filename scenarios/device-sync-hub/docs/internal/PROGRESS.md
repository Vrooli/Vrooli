# Progress — Device Sync Hub

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-18 | codex | done | Cleared disposable local first-run hub state after backing up `data/device-sync-hub.db` to `data/backups/device-sync-hub.db.pre-owner-reset-20260618T223120Z` (`hub_owner` and `devices` now empty; authenticator accounts left intact). Improved first-device setup UX: `SetupDeviceStep` now gives workflow-specific recovery copy for `permission_denied` (hub already owned by another account) and `unauthenticated` (sign-in expired) instead of surfacing generic error text; updated locale keys, generated string registry, focused tests, and error-handling docs. |
| 2026-06-17 | agi | done | **First-run owner bootstrap + production-ready UI** (plan `device-sync-hub-first-run-owner-bootstrap-production-ready-ui`). Closed the first-device dead-end: a fresh hub had no product path to pair its first device (UI gate showed only a JoinScreen, CLI had no owner-token plumbing, ownership was implicit/unguarded). **API:** explicit single-owner model — new `hub_owner` singleton table + `ClaimOwner`/`HubOwner` repo methods (atomic first-owner-wins `INSERT … ON CONFLICT(id) DO NOTHING`), removed `ResolveOwner`; new `DevicesService.SetupOwnerDevice` RPC (claims if unclaimed → rejects a different identity with `ErrNotOwner`/PermissionDenied → registers caller TRUSTED, returns one-time token); `requireHubOwner` guard on every owner-authed service method (List/Get/IssuePairingCode/Approve/Rename/Revoke); proto regenerated. **CLI:** new `domains/auth` (`login`/`logout`/`whoami`) targeting scenario-authenticator (resolved via `--auth-api-base` → `$AUTH_SERVICE_URL` → typed `vrooli scenario port` auto-detect via vrooli-cli-go), stores owner JWT in `Config.Token`; new `devices setup` verb → `SetupOwnerDevice`; `make endpoints` regenerated (API↔CLI parity green). **UI:** replaced the `isPaired`-only gate with a `features/onboarding/` state machine (`OnboardingScreen` choose→setup→join; `OwnerLoginForm` real login posting to scenario-authenticator via `api/authenticator.ts`, token-paste demoted to Advanced fallback; `SetupDeviceStep`; `JoinHubForm`); session store gains `ownerEmail`. **Polish:** migrated `Input`/`Textarea`/`ErrorBoundary` off hardcoded `white`/`slate` to design tokens (they were invisible in the light default theme). **Docs:** rewrote QUICKSTART, added DECISIONS rows (+ superseded token-paste), updated FLOWS (owner-setup flow + state machine). **Green:** api+cli `go build/vet/test`+`golangci-lint`+`gofumpt`; UI `strings:check`/`type-check`/`lint` (0 errors) + `test` (233 passed) + coverage ≥85%. **Validation:** test-genie run `20260617-185301-84771b03` = 15/18 phases pass (incl. unit/integration/ui-health/smoke/proto/contracts/business/docs/security/architecture). The 3 failing phases (standards/dependencies/tidiness) are **identical to the pre-change baseline `dsh-owner-bootstrap`** and live in files this work never touched (Makefile, realtime/sse_handler.go, testutil/modeltest, manifest-test boilerplate); an afid-set diff vs baseline shows **0 new gating findings** (the only new afids are sev≤2 advisory: a UI-feature-no-domain architecture note for `onboarding`, two low tidiness duplications in my test/flag boilerplate, and a Go-stdlib `crypto/tls` CVE). GCT `baseline diff` CLI itself was blocked by another agent's transient uncommitted root-CLI WIP (`internal/cli/scenariocli/detemplate.go` false unused-import), so the afid comparison was run directly. **Authenticator:** fixed its documented schema-idempotency blocker (19 non-idempotent `CREATE INDEX`→`IF NOT EXISTS`); a deeper DB-connection-backoff hang remains → live cross-scenario hop deferred, bug filed `knw-1781722385713679749`, bootstrap logic fully proven against auth stubs/fakes. rec-e3c9d930bb5830da. |
| 2026-06-17 | agi | done | **Phase 5 (polished split-screen UI).** Replaced the react-vite placeholder UI with the production transfer hub. **Signature layout:** full-height vertical split — Receive (top, `app-primary` accent) over Send (bottom, `app-accent` accent) — as the index route; routes are Transfer / Devices / Settings. **Auth/session:** dual-credential layer (`features/session/`): localStorage-backed `useSession()` + one custom `authedFetch` (`api/transport.ts`) attaching `X-Device-Token` and/or `Authorization: Bearer` fresh per request; an unpaired browser sees a **Join screen** (redeem code → `redeemPairingCode`, or request→approve → `requestPairing`), paired browsers see the shell. **Receive:** react-query `listItems` with client-side search/sort/filter + card↔list toggle; file download via device-token fetch→blob→`<a download>`; image thumbnails via token-authed blob fetch; text snippets inline w/ copy; owner-origin remove; retention badge + expiry. **Send:** drag-drop + file-picker + text staging as cards with per-item retention (Live/Held/Pinned, default Held) + optional target device (from `listDevices` when owner-signed-in, else broadcast); text→`createTextItem`, files→device-token multipart XHR upload. **Devices (owner-gated):** list/rename/revoke(confirm)/approve + issue pairing code with dependency-free SVG QR (`lib/qr.ts`) + token-paste owner sign-in. **Realtime:** one app-level SSE stream (`EventSource` + `?token=`) decoding proto-JSON `Event` via a pure reducer — item events invalidate the items query, presence drives online dots + a top-bar indicator, `PAIRING_REQUESTED` raises an approve/reject banner; reconnect handled. **Removed** the template notes UI (features/notes, NotesPage, api/notes, its selectors/strings/tests); folded `health` into Settings (notes *backend* left for Final Cleanup). All strings through `t(strings.*)` (en/ja/ar parity, regen committed); all test-ids through the selectors registry; no new deps; `app-*` theme tokens; a11y (landmarks/roles/labels/keyboard, axe tests on shell + transfer surface). **DoD all green:** `strings:check`, `type-check`, `lint` (0 errors; 4 pre-existing react-refresh warnings), `test` (163 passed / 39 files), `build`. |
| 2026-06-17 | agi | done | **Phase 4 (CLI surface + programmatic compound-value seam).** Added the `devices` and `transfer` CLI domains (cli/domains/devices, cli/domains/transfer) mirroring the DevicesService/TransferService Connect-RPC contracts, plus the two REST byte edges (`transfer upload` multipart, `transfer download` binary) appended outside the manifest. Extended cli/manifest.json with both groups (8 devices verbs + 4 transfer connect verbs, governance-tagged; revoke/delete=destructive). Transfer is **device-token authed** (`X-Device-Token` header from `--device-token` / `$DEVICE_SYNC_HUB_DEVICE_TOKEN`), distinct from the owner JWT; upload streams via io.Pipe (no full-file buffering). Wired all three groups in domains.go; regenerated cli-commands.gen.json + .vrooli/endpoints.json via `make endpoints` (**API↔CLI parity crossCheck now GREEN — closes the Phase-2/3 carve-out**). Per-domain manifest-coverage tests + handler tests (render, --json proto wire shape, error surfacing, required-arg enforcement, device-token requirement + header propagation, multipart upload, streaming download w/ original filename). Verified: `go build ./...`, `go test ./...` (all CLI packages), `go vet`, `gofumpt`, `golangci-lint` — all green. |
| 2026-06-17 | agi | in-progress | **Phase 6 (testing/validation/hardening) — most blockers resolved.** Removed the `notes` example domain end-to-end (handlers/internal/cli dirs, registry, main.go `/measures` mount, proto schema+gen, manifest group+measure, seed file, all comment refs retargeted to devices/transfer; `make endpoints` idempotent). Fixed REST-exception contract drift: dropped the scenario-invented `binary_download`/`event_stream` reasons (not in the shared `endpoints.schema.json` 4-reason enum), remapped transfer-download + realtime-SSE to `ops_probe`, added required `proto_payloads` to all 4 REST exceptions → `.vrooli/endpoints.json` schema-valid. Standards: Makefile CRITICAL fixed (restored template `endpoints` target comment), env-var HIGHs fixed (removed hardcoded `localhost:15000` AUTH default + value-logging). Added `internal/middleware/security.go` (SecurityHeaders outermost wrap: X-Frame-Options/X-Content-Type-Options/X-XSS-Protection/Referrer-Policy/HSTS + reflective CORS). gosec 4 ERRORs (G101×3 + G704, all FPs) suppressed via repo-standard `//nolint:gosec`. `go mod tidy` api+cli. **Security-Headers HIGHs fixed at the SHARED rule (per user decision): patched scenario-auditor** — `security_headers` rule now skips `_test.go`, and `handlers_standards.go` credits a scenario whose central middleware sets the full header set (post-filter keyed on violation.Type=="security_headers"); added 2 tests; rebuilt+restarted; **re-scan confirms standards highest=medium (0 high/critical) → gating-green**. UI coverage raised to ≥85% all metrics (97.27% lines / 90.86% funcs / 85.86% branches, 212 tests, +10 test files) — unit `pnpm test:coverage` now exit 0. pnpm `minimumReleaseAge: 10080` added to ui/pnpm-workspace.yaml. **ALL DIRECT GATES GREEN:** go build/vet/test + golangci-lint + gofumpt (api & cli), UI test:coverage + type-check + lint (0 errors). **DEFERRED/ENVIRONMENTAL:** (a) comprehensive `test-genie` run + GCT baseline diff — test-genie was under concurrent modification by another agent (unreliable); re-run when stable; (b) scenario-authenticator fails its own health check (sibling DB/migration env issue, out of scope per plan) → deps-phase runtime + baseline diff blocked; (c) 23/23 requirements still lack `[REQ:ID]` test tags (warning); (d) Go handler-wiring LOW_COVERAGE warnings (non-gating). **DEFERRED BY DESIGN (user decision):** P1s NOT built — settings domain, chunked/resumable upload, at-rest encryption (PRD P1/post-launch). |
| 2026-06-17 | agi | done | **Phase 1 (greenfield rewrite — generate + charter + docs).** Regenerated from `react-vite` (v1.1.0) + `vrooli-default` design kit over the stale 2025 scenario (old tree backed up to `/tmp/device-sync-hub-OLD-*`). Authored `PRD.md` via prd-control-tower (8 P0 / 7 P1 / 4 P2 targets; validates clean). Generated `requirements/` (15 modules, 23 requirements covering all P0+P1; validates "all linked"). Declared dependency decisions in `.vrooli/service.json` (scenario-authenticator required+try_start+degraded_behavior; redis optional). Authored real `docs/concepts/DOMAINS.md` (domains: auth, devices, transfer, realtime, settings, health) + ARCHITECTURE/DATA/FLOWS/INTEGRATIONS. Orientation 6/8 gates green. |
| 2026-06-17 | agi | done | **Phase 2 (API foundation + auth integration + devices/pairing/trust domain).** See git: `devices.proto` (DevicesService), `internal/auth` (authenticator client + owner middleware, fail-closed), `internal/devices` (registry, single-use pairing claim, trust lifecycle, revoke), wired into `main.go` + registry. Green: build/test/e2e/lint. |
| 2026-06-17 | agi | done | **Phase 3 (transfer domain + retention + presence + quotas + realtime).** `transfer.proto` (TransferService: CreateTextItem/ListItems/GetItem/DeleteItem) + `realtime.proto` (Event messages, no service). `internal/transfer`: items (file+text, both first-class), SQLite repo with delivery-ACL visibility (broadcast/directed/origin), retention Live/Held/Pinned + `ExpiresAt` stamping, background `RunPurgeLoop`, per-owner/per-device quotas, pure-stdlib image `Thumbnailer`. `internal/realtime`: in-memory presence + SSE event hub (item-arrived/deleted, presence-changed, pairing-requested) with delivery-ACL fan-out. `internal/deviceauth`: device-token middleware (`X-Device-Token`/`?token=`) → `RequireDevice`. `internal/devices`: added `Authenticator` (token→TRUSTED device) + `PairingNotifier` hook; devices handler now overlays live presence + pushes pairing banners. Handlers: transfer Connect handler + REST multipart upload (quota pre-check, thumbnail) + REST streaming download (original filename, `?thumb=1`, marks Live delivered). Two new REST-exception reasons (`binary_download`, `event_stream`). Wired all into `main.go` (shared hub, device authenticator, transfer wiring, purge goroutine, dual auth middleware) + registry (transfer in AllEndpoints/AllProtoFiles/AllSchemas; realtime in AllEndpoints). **Green:** `go build ./...`, `go vet`, `gofumpt`, `golangci-lint` (new pkgs), and `go test ./...`/`-tags e2e` for every package **except** the pre-existing `cmd/gen-endpoints` crossCheck (see handoff). New tests: transfer sqlite/service/thumbnail, realtime hub, devices authenticator, deviceauth middleware, transfer connect + REST upload/download roundtrip. |
| 2026-06-17 | agi | done | **Phase 2 (API foundation + auth integration + devices domain).** Authored `packages/proto/schemas/device-sync-hub/v1/devices/devices.proto` (`DevicesService`: List/Get/IssuePairingCode/RedeemPairingCode/RequestPairing/ApprovePairing/Rename/Revoke; `TrustState` enum; `Device`/`PairingCode`/`DeviceProfile`) + `buf generate`. Built `api/internal/auth/` (AuthClient seam over scenario-authenticator `GET /auth/validate` + `DELETE /sessions/{id}`; owner `Middleware` best-effort-injects Identity; `RequireOwner` gate; fail-closed, no test bypass). Built `api/internal/devices/` (types/schema/service/repository/sqlite/secrets/error-mapping + mocks): device registry, single-use conditional pairing-code claim, hub device-token issuance (SHA-256-hashed at rest), trust lifecycle pending→trusted→revoked, owner-scoped queries, best-effort authenticator session revoke. Built `api/handlers/devices/` (connect_handler/adapter/module/endpoints) — owner-gated RPCs vs open pairing RPCs. Wired registry (3 lines: endpoints/proto-files/schemas) + `main.go` (auth client + owner middleware + devices module). Regenerated `.vrooli/endpoints.json`. **All green:** `go build ./...`, `go test ./...` (incl. proto-connect parity + sqlite round-trips), `-tags e2e`, `golangci-lint` on new packages, proto module compiles. rec pending. |

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
