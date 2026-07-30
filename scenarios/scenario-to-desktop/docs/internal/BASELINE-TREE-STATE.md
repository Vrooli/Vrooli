# Desktop deployment readiness baseline tree state

Captured: 2026-07-29

- Commit: `e35ea541cd1a848cb3fbda28e0b1d193b3e30d9a`
- Working tree: dirty before plan execution (402 porcelain entries); no existing change was reset, stashed, or overwritten.
- Concurrent-plan checkpoint: `/tmp/vrooli-desktop-readiness-resource-deployment-20260729` (four files copied from the then-untracked `packages/resource-deployment/`).
- Scenario-to-desktop baseline run: `20260729-045600-569f50f4` — **FAIL**, 19/20 phases passed; the sole failing phase was `ui-health` due to a pre-existing stale `ui/dist/index.html` relative to `packages/proto/gen/typescript/buf/validate/validate_pb.ts`.
- Secrets-manager baseline run: `20260729-050004-91c7830d` — **FAIL before phase execution**. Lifecycle reported Vault stopped/unhealthy but startup could not bind `127.0.0.1:8200` because the port was already in use.

The full pre-plan porcelain listing is intentionally recoverable from the worktree and commit above; this record identifies the precise immutable revision, checkpoint, and server-owned test runs used for comparison.
