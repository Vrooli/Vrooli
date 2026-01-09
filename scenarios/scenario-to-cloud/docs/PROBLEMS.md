# Problems / Risks

## Open Issues

- **Packager contract not implemented yet**: deployment-manager docs reference scenario-to-cloud as a stub; API/CLI contract needs to be finalized and implemented.
- **“Mini Vrooli” bundling semantics**: bundle embeds `.vrooli/cloud/manifest.json` + `.vrooli/cloud/bundle-metadata.json`, rewrites `.vrooli/service.json` to disable unused resources, and now generates a trimmed `go.work` so Go tooling won’t reference stripped modules — but still needs real VPS validation that `./scripts/manage.sh setup` honors it end-to-end.
- **Remote port overrides**: forcing fixed ports (UI 3000, API 3001, WS 3002) must be compatible with the lifecycle allocator and scenario assumptions.
- **Caddy + Let’s Encrypt edge cases**: HTTP-01 requires port 80/443 open and DNS already pointing at the VPS; preflight must be crisp and actionable.
- **Long-running deploy endpoints**: VPS setup/deploy can exceed typical HTTP client/server timeouts; P0 keeps sync endpoints (server WriteTimeout raised), but async job orchestration is likely needed for real deployments.
- **Inspect/log ergonomics**: P0 supports bounded log retrieval via `tail`, but does not yet support streaming (`--follow`) or advanced filtering (time ranges, multiple scenarios/resources).
- **Tooling mismatch (repo-level)**: `scenario-completeness-scoring` attempts to auto-rebuild and fails with `go.work` workspace errors; this is outside `scenario-to-cloud` scope but affects reporting.
- **Tooling mismatch (test-genie flags)**: `vrooli scenario status` invokes `test-genie` with a `-no-record` flag that the installed `test-genie` binary does not recognize; may require updating test-genie or the caller.
- **Browserless dependency**: UI smoke + lighthouse checks require `BROWSERLESS_URL` (typically `http://localhost:4110`); if Browserless is not running, status checks and some test phases will be blocked.
- **E2E validations are still placeholders**: playbooks exist for preflight/setup/deploy/logs, but real VPS E2E automation is not wired into the default suite yet.

## Deferred (Explicit P2+)

- Rollback/blue-green, backups/restore, resource swaps to managed services, bastion hosts.
