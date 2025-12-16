# Problems & Notes: git-control-tower

## Open Issues

- The scenario has been re-scaffolded from the `react-vite` template; operational targets are preserved, but no target is implemented yet.
- Git operations are inherently risky; design must enforce repo-root path validation, explicit allowlists, and safe defaults for any mutating endpoint.
- Repository size and file counts can cause slow status/diff calls; plan pagination and caching before claiming “production-ready”.

## Deferred Ideas

- Multi-repo support (path switching, isolation, multi-tenant auth).
- Real-time UI updates (fsnotify + WebSocket channel) once baseline endpoints exist.

