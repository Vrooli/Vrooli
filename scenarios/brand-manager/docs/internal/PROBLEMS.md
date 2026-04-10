# Problems & Deferred Ideas

## Open Issues
- ~~Auditor violations~~ **RESOLVED** (Phase 18-iter3): 0 security, 0 standards violations.
- modernc.org/sqlite registers as driver "sqlite" not "sqlite3" — api-core DriverSQLite constant doesn't match; using string literal "sqlite" as workaround
- ~~Version snapshot duplication~~ **RESOLVED** (Phase 19): Extracted `snapshotVersion()` shared helper; version snapshot creation degrades gracefully (logged warning) if it fails.
- ~~AI config boundary leak~~ **RESOLVED** (Phase 19): AI provider env vars moved from handler-layer `aiChain()` to `config.Load()`.
- 4 requirements pending (blocked on external dependencies): BM-REQ-AGENT-SPAWN/INSTRUCT/VALIDATE (need agent-manager service), BM-REQ-LIGHTHOUSE (need test-genie browserless)
- Completeness coverage subscore (6/15) limited by stale test-genie cache (48 tests tracked); actual test count is 503+ (269 Go + 214 UI + 20 CLI). A test-genie run would refresh this to ~93+ score.

## Deferred Ideas
- Framework-extensible scanner plugins (P2) — support Tailwind config, SCSS beyond CSS + JSON
- test-genie Lighthouse integration (P2) — supplemental WCAG accessibility checking
- Error rate metrics / counters — currently errors are logged but not counted; future observability improvement

## Known Risks
- OpenRouter image generation quality may be insufficient for professional logos — configurable models + user upload override as mitigation
- Agent-assisted application must mandate inline markers or fail — hard constraint to maintain validation guarantee
- CSS brand-manager: markers could be stripped by aggressive minification — document supported minifier configs, consider `/*!` prefix
