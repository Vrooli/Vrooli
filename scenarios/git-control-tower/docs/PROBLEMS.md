# Problems & Notes: git-control-tower

## Open Issues

- The scenario has been re-scaffolded from the `react-vite` template; OT-P0-001 has initial implementation + tests, but most operational targets remain unimplemented.
- Git operations are inherently risky; design must enforce repo-root path validation, explicit allowlists, and safe defaults for any mutating endpoint.
- Repository size and file counts can cause slow status/diff calls; plan pagination and caching before claiming “production-ready”.
- Some `test-genie` phases (smoke/performance) may require a running Browserless service; defer those phases until the shared test infrastructure is available.
- `vrooli scenario status git-control-tower` currently fails its optional `test-genie` structure probe due to an unsupported `-no-record` flag; the core scenario test lifecycle still uses `test-genie execute`.

## Deferred Ideas

- Multi-repo support (path switching, isolation, multi-tenant auth).
- Real-time UI updates (fsnotify + WebSocket channel) once baseline endpoints exist.
