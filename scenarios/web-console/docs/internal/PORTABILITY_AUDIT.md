# web-console Cross-Platform Readiness Audit

## Last Updated
2026-03-18

## Target Tiers
- [x] Tier 2 Desktop (Electron) — ready (CORS, storage, static build all handled)
- [ ] Tier 3 Mobile — mobile clients use the remote/bridge terminal path
- [x] Tier 4 Cloud/SaaS
- [x] Tier 5 Enterprise

## Environment Variable Status
| Variable | Usage | Fallback? | Desktop-Ready? |
|----------|-------|-----------|----------------|
| VROOLI_ROOT | Not used | N/A | Yes |
| VROOLI_DATA | Not used directly | N/A (uses api-core/storage) | Yes |
| SCENARIO_ROOT | Not used | N/A | Yes |
| SCENARIO_DIR | Working dir inference | Falls back to cwd/home | Yes |
| PROJECT_ROOT | Working dir inference | Falls back to cwd/home | Yes |
| API_PORT | Server port | Defaults to 8080 via api-core/server | Yes |
| UI_PORT | CORS allowed origin | Defaults to 36233 | Yes |
| WC_DEFAULT_SHELL | Shell binary | Falls back to $SHELL, then /bin/sh | Yes |
| WC_DEFAULT_CWD | Working directory | Multi-step fallback chain | Yes |
| WHISPER_URL | Whisper ASR endpoint | Defaults to http://localhost:8090 | Yes |
| KOKORO_URL | Kokoro TTS endpoint | Defaults to http://localhost:8880 | Yes |
| OLLAMA_URL | Ollama AI endpoint | Defaults to http://localhost:11434 | Yes |
| OPENROUTER_API_KEY | OpenRouter fallback | Optional, gracefully absent | Yes |

## Resource Dependencies
| Resource | Fitness (Desktop) | Strategy | Alternative | Reasoning |
|----------|-------------------|----------|-------------|-----------|
| SQLite | 1.0 | Full Replace (already done) | N/A | Pure Go driver (modernc.org/sqlite), no CGO |
| Ollama | 0.0 (always-on) | Optional with fallback | OpenRouter | AI generation falls back to OpenRouter when Ollama unavailable |
| Whisper | 0.0 (server) | Optional | None (voice disabled) | Capability check gates voice features |
| Kokoro | 0.0 (server) | Optional | None (TTS disabled) | Capability check gates TTS features |

## Build Status
- [x] CGO_ENABLED=0 builds successfully
- [x] No CGO dependencies (uses modernc.org/sqlite, not go-sqlite3)
- [x] No hardcoded platform-specific paths
- [x] Cross-compilation ready (standard Go, pure-Go SQLite)

## Filesystem Status
- [x] Database path via api-core/storage (ClassData)
- [x] Voice config via api-core/storage (ClassState)
- [x] TTS config via api-core/storage (ClassState)
- [x] Hook token via api-core/storage (ClassState)
- [x] Session state via api-core/storage (ClassState)
- [x] Upload directory via api-core/storage (ClassCache)
- [x] All runtime storage paths resolve through `api-core/storage` class directories

## Network Status
- [x] API port configurable via API_PORT env var (default: 8080)
- [x] UI port configurable via UI_PORT env var (default: 36233)
- [x] CORS accepts both localhost and 127.0.0.1 on UI port
- [x] All service URLs configurable (WHISPER_URL, KOKORO_URL, OLLAMA_URL)
- [x] Offline capability documented in service.json (partial: AI/voice need network)

## Secret Classification
| Secret | Class | Handling |
|--------|-------|----------|
| Hook auth token | per_install_generated | Random 32-byte hex, persisted to storage ClassState |
| OPENROUTER_API_KEY | user_prompted | Optional env var, gracefully absent |
| POSTGRES_PASSWORD | infrastructure | Not needed — using SQLite |

## Issues Resolved
1. `upload_handler.go` — Hardcoded temp upload directory replaced with `api-core/storage` (ClassCache)
2. `voice_transcribe.go` — Package-level `whisperURL` now reads WHISPER_URL env var at init
3. `main.go:Handler()` — Added CORS middleware accepting both localhost and 127.0.0.1
4. `service.json` — Added `capabilities` section with `offline_capable` and AI provider documentation
5. `TerminalPane.tsx` — Fixed ESLint non-null assertion error (safe array access)
6. `Workspace.tsx` — Fixed missing `syncPaneUpdate` dependency in useEffect

## Known Limitations
1. Unix PTYs use creack/pty; Windows uses the native ConPTY adapter
2. Shell fallback is platform-resolved (`$SHELL`/`/bin/sh` on Unix, PowerShell on Windows)
3. `initSchema` reads SQL files relative to binary — desktop bundles must ship `api/internal/<domain>/` directory alongside binary (candidate for `//go:embed` in future)

## Required Changes for Tier 3 (Mobile)
1. PTY replacement needed — mobile cannot run shell processes
2. Would need a remote terminal proxy or WebSocket relay architecture
