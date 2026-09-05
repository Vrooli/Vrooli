# Problems / Risks

## Work ladder

- Rung: W0
- Evidence: `swarm-manager goals list --json` returned no goal whose name, title, or description names `scenario-to-cloud`; the active user-owned Plan Manager execution `one-privileged-moment-one-interactive-surface-a-cross` is the authoritative work item for this change but is not represented in the swarm-manager goal store.
- Blocker: W0 contract-to-goal comparison is not independently verifiable through the swarm-manager gate; continue under the explicit Plan Manager objective and do not claim the scenario contract is independently reconciled.
- Measured: 2026-08-14

## Open Issues

- **Packager contract not implemented yet**: deployment-manager docs reference scenario-to-cloud as a stub; API/CLI contract needs to be finalized and implemented.
- **Native mini-Vrooli VPS validation**: bundle embeds `.vrooli/cloud/manifest.json` + `.vrooli/cloud/bundle-metadata.json`, rewrites `.vrooli/service.json` to disable unused resources, generates a trimmed `go.work`, and now uploads a deployment-local native `vrooli` binary to the target VPS. Unit coverage is good, but we still need disposable-VPS E2E validation that the native upload + `vrooli setup --yes yes --environment production` flow works end-to-end.
- **Remote port overrides**: forcing fixed ports (UI 3000, API 3001, WS 3002) must be compatible with the lifecycle allocator and scenario assumptions.
- **Caddy + Let’s Encrypt edge cases**: HTTP-01 requires port 80/443 open and DNS already pointing at the VPS; preflight must be crisp and actionable.
- **Long-running deploy endpoints**: VPS setup/deploy can exceed typical HTTP client/server timeouts; P0 keeps sync endpoints (server WriteTimeout raised), but async job orchestration is likely needed for real deployments.
- **Inspect/log ergonomics**: P0 supports bounded log retrieval via `tail`, but does not yet support streaming (`--follow`) or advanced filtering (time ranges, multiple scenarios/resources).
- **Tooling mismatch (repo-level)**: `scenario-completeness-scoring` attempts to auto-rebuild and fails with `go.work` workspace errors; this is outside `scenario-to-cloud` scope but affects reporting.
- **Tooling mismatch (test-genie flags)**: `vrooli scenario status` invokes `test-genie` with a `-no-record` flag that the installed `test-genie` binary does not recognize; may require updating test-genie or the caller.
- **Browser-automation dependency**: UI smoke + lighthouse checks require `browser-automation-studio` (the Playwright driver) to be running; if it is not, status checks and some test phases will be blocked.
- **E2E validations are still placeholders**: playbooks exist for preflight/setup/deploy/logs, but real VPS E2E automation is not wired into the default suite yet.

## Deferred (Explicit P2+)

- Rollback/blue-green, backups/restore, resource swaps to managed services, bastion hosts.
