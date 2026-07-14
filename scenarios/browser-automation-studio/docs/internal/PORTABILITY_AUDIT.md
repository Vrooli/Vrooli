# browser-automation-studio Cross-Platform Readiness Audit

## Last Updated
2026-04-16 — written immediately after the SQLite-only migration unblocked Tier-2 storage portability.

## Target Tiers
- [x] Tier 1 Local — fully supported (production today via app-monitor + Cloudflare tunnel).
- [~] Tier 2 Desktop (Electron) — storage layer ready; **blocked** on bundling the Playwright driver and the MinIO swap. `service.json` `tier-2-desktop.fitness_score` = 0.5.
- [ ] Tier 3 Mobile — `service.json` declares `status: blocked`, `fitness_score: 0.1`. Browser automation requires Chromium; not addressable on iOS/Android.
- [~] Tier 4 SaaS / Cloud — supported with the same caveats as Tier 1 plus orchestration of the Playwright driver and MinIO. `fitness_score: 0.6`.
- [ ] Tier 5 Enterprise / Appliance — derived from Tier 4 with hardened secrets; not separately audited.

## Environment Variable Status

| Variable | Usage | Fallback? | Desktop-Ready? | Notes |
|---|---|---|---|---|
| `VROOLI_ROOT` | `api/database/connection.go:224`, `api/main.go:740` | Yes — falls back to `repocontract.FindRepoRootFromEnvOrCWD()` | ⚠️ Partial — both fallbacks expect the monorepo layout. Desktop bundles need to inject this or replace `resolveScenarioRoot` with a bundle-aware helper. | The schema file is loaded relative to scenario root, so a bundle must either ship the `initialization/` tree at the expected relative path or override the loader. |
| `VROOLI_DATA` | Referenced only in `database/sqlite_smoke_test.go:56` (test isolation, asserts it is *ignored*) | n/a | n/a | Not a runtime input. Resolver uses `ProfileAuto` instead. |
| `BAS_SQLITE_PATH` | `api/database/connection.go:113` | Yes — falls back to `DATABASE_URL=file:…` then `storage.NewResolver` | ✅ | Desktop bundles can inject an Electron-managed path. |
| `DATABASE_URL` | `api/database/connection.go:115` (file: prefix only) | Yes | ✅ | |
| `API_PORT` / `UI_PORT` | `internal/scenarioport/scenarioport.go:189`, `automation/session/manager.go:343`, `handlers/record_mode.go`, `sidecar/supervisor/process.go:104`, etc. | Mixed — some sites have defaults, some warn-and-default | ⚠️ Partial | Electron wrapper must inject both listener ports. `sidecar/supervisor/process.go:107` is the only site that logs a warning instead of crashing on missing `API_PORT`. |
| `PLAYWRIGHT_DRIVER_URL`, `MINIO_*` | Various | Hard requirement | ❌ for desktop | Neither the driver process nor MinIO is bundleable as-is — see Resource Dependencies table. |

**No code currently uses `VROOLI_DESKTOP_MODE`, `APP_DATA_DIR`, or `BUNDLE_ROOT`.** Adopting these would unblock the bundle-aware variant of `resolveScenarioRoot`.

## Resource Dependencies

| Resource | Fitness (Desktop) | Strategy | Alternative | Reasoning |
|---|---|---|---|---|
| **SQLite (embedded)** | ✅ 1.0 | Already embedded | n/a | `modernc.org/sqlite` is pure-Go; CGO_ENABLED=0 builds verified clean (audit ran `CGO_ENABLED=0 go build ./...` → exit 0). |
| **MinIO** | 🟡 ~0.4 | Runtime swap | Filesystem store rooted under `paths.ResolveRecordingsRoot()` | `service.json` lists this as a 2-day adaptation. Storage abstraction (`api/storage/interface.go`) already has the seam. |
| **Playwright driver** | 🟡 ~0.6 | Bundle into app | Compile the Playwright driver into the Electron bundle via `ENGINE=playwright` + `PLAYWRIGHT_DRIVER_URL` (already supported at runtime per `README.md`) | The runtime path exists; what's missing is shipping the Playwright driver inside the Electron bundle and wiring its lifecycle from main.ts. `service.json` lists this as a 5-day adaptation. |
| **Ollama** | 🟢 already optional | Optional with OpenRouter fallback | OpenRouter | `service.json` already declares Ollama as `required: false` and OpenRouter as `required: true`. Matches the `cross-platform-readiness/SKILL.md` §3.1 special-case for Ollama. |
| **OpenRouter** | ✅ network-only | n/a | n/a | Per `cross-platform-readiness/SKILL.md` §7, `OPENROUTER_API_KEY` is `user_prompted` — already classified that way in `service.json` `tier-1-local.secrets`. |

## Build Status
- [x] **CGO_ENABLED=0 builds successfully.** Verified by audit: `CGO_ENABLED=0 go build ./...` → exit 0. No `mattn/go-sqlite3` or `// #cgo` references remain.
- [x] No CGO imports in production code (`grep "// #cgo\|import \"C\""` empty in `api/`).
- [x] SQLite driver = `modernc.org/sqlite` everywhere (verified across `api/database/connection.go`, all three credits test files, `services/recording/persistence/sqlite_test.go`, and the new `services/uxmetrics/repository/sqlite_test.go`).
- [ ] **Cross-compilation not yet exercised in CI.** No `GOOS=darwin`/`GOOS=windows` build matrix exists. Worth adding before Tier 2 work begins.

## Filesystem Status
- [x] Runtime filesystem writes route through `api-core/storage` for both DB (`scenarioDBPath`) and recording artifacts (`resolveScenarioStoragePath`).
- [x] `wire.BuildDependencies` and `handlers.InitDefaultDeps` resolve recording roots via `paths.ResolveRecordingsRoot(log)` — no scenario-local `./data` writes.
- [x] No `DATA_DIR` / `./data` / hardcoded user-home paths in `api/` (verified by grep).
- [ ] `storage.WriteFileAtomic` not currently used. Acceptable today (see STORAGE_AUDIT.md item 2) but worth adopting for execution-result JSON writes if we want clean crash semantics on desktop.

## Network Status
- [x] **CORS accepts both `localhost` and `127.0.0.1`** (`api/middleware/cors.go:83-87`).
- [x] Listener ports configured via `API_PORT`/`UI_PORT` env vars throughout. Most sites have inline defaults; `sidecar/supervisor/process.go:107` warns rather than crashes.
- [ ] No explicit `offline_capable` declaration in `service.json` `capabilities`. AI features require OpenRouter (or local Ollama if enabled); core workflow execution does not require network beyond the target page load. Worth declaring formally per `cross-platform-readiness/SKILL.md` §6.3.

## Secret Classification
Already declared in `service.json` `tier-1-local.secrets`:
- `openrouter/api-key` — `user` (prompt at install time)

The previous `browser-automation-studio/postgres-url` secret is gone; SQLite has no auth surface.

## Branding (deployment readiness)
- [x] `service.displayName = "Vrooli Ascension"`
- [ ] Logo SVG / rasterized assets — not yet present in `bundle/` or `.vrooli/`
- [ ] Favicon multi-size — not audited
- [ ] Color system / typography spec — not audited

These are out of scope for the postgres-removal audit but the next obvious cross-platform readiness chunk if bundling is queued.

## Issues Found
1. **`resolveScenarioRoot` is monorepo-aware only** (`api/database/connection.go:222-238`, `api/main.go:740`). Both branches assume `repocontract.FindRepoRootFromPath` / `FindRepoRootFromEnvOrCWD` will land in the Vrooli monorepo. A bundled Electron build won't have that layout. Adding a `VROOLI_DESKTOP_MODE` / `BUNDLE_ROOT` branch (per skill §2.2) is a prerequisite for Tier 2.
2. **Schema file is loaded by relative path** (`initialization/storage/sqlite/schema.sql` joined to scenario root). For a desktop bundle, either embed the schema with `//go:embed` or ship the `initialization/` tree alongside the binary.
3. **No cross-compilation CI step.** `CGO_ENABLED=0 go build` works on Linux today but no Windows/macOS matrix exists to catch regressions.
4. **`offline_capable` missing from `service.json` `capabilities`.** Should be declared with limitations note ("AI features require OpenRouter network egress").

## Required Changes for Tier 2 (Desktop)
In rough priority order:

1. **Bundle-aware scenario-root resolver.** Add `if os.Getenv("VROOLI_DESKTOP_MODE") == "true" { return os.Getenv("BUNDLE_ROOT"), nil }` short-circuit at the top of `resolveScenarioRoot()` in `api/database/connection.go` and `api/main.go`. Files: `api/database/connection.go`, `api/main.go:740`.
2. **Embed the schema file.** Replace the `os.ReadFile(schemaPath)` in `initSchema()` with a `//go:embed initialization/storage/sqlite/schema.sql` byte slice — eliminates the relative-path concern entirely. File: `api/database/connection.go:194`.
3. **Bundle Playwright driver into Electron** and wire its lifecycle from main.ts. Already supported at runtime via `ENGINE=playwright` + `PLAYWRIGHT_DRIVER_URL`; `service.json` lists this as 5 effort-days. Files: `playwright-driver/`, the desktop wrapper (lives outside this scenario, in `scenario-to-desktop`).
4. **MinIO → filesystem-store swap.** `service.json` lists this as 2 effort-days. The `api/storage/interface.go` seam is already in place; need a `FilesystemScreenshotStorage` implementation that writes under `paths.ResolveRecordingsRoot()`.
5. **Cross-compilation CI matrix** — `linux/amd64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, all with `CGO_ENABLED=0`.
6. **Declare `offline_capable: false` with limitations** in `service.json` `capabilities` (per skill §6.3 pattern).

## Required Changes for Tier 3 (Mobile)
Not addressable. Browser automation requires Chromium; iOS sandboxing forbids spawning child browser processes; Android Chromium control via WebView is too restricted for general automation. The sensible mobile play is a **client app that talks to a remote BAS server**, not a self-contained mobile bundle. `service.json` already declares this status accurately.

## Areas Not Yet Audited
- `playwright-driver/` (Node.js) cross-platform readiness — separate audit needed for the driver process itself.
- UI bundle (`ui/`) for hardcoded `localhost` references in built JS — not greppable from this audit pass.
- Branding assets (logo, favicon, color system) per skill §8.2 branding checklist.
