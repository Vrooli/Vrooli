# Problems / Risks

## Open Issues

- **Packager contract not implemented yet**: deployment-manager docs reference scenario-to-cloud as a stub; API/CLI contract needs to be finalized and implemented.
- **“Mini Vrooli” bundling semantics**: defining what to strip/include (scenarios/resources/packages) needs deterministic manifests and reproducible build rules.
- **Remote port overrides**: forcing fixed ports (UI 3000, API 3001, WS 3002) must be compatible with the lifecycle allocator and scenario assumptions.
- **Caddy + Let’s Encrypt edge cases**: HTTP-01 requires port 80/443 open and DNS already pointing at the VPS; preflight must be crisp and actionable.
- **Tooling mismatch (repo-level)**: `scenario-completeness-scoring` attempts to auto-rebuild and fails with `go.work` workspace errors; this is outside `scenario-to-cloud` scope but affects reporting.
- **Tooling mismatch (test-genie flags)**: `vrooli scenario status` invokes `test-genie` with a `-no-record` flag that the installed `test-genie` binary does not recognize; may require updating test-genie or the caller.
- **Browserless dependency**: UI smoke + lighthouse checks require `BROWSERLESS_URL` (typically `http://localhost:4110`); if Browserless is not running, status checks and some test phases will be blocked.
- **Requirement refs are still placeholders**: several `requirements/*/module.json` entries reference playbooks/tests that do not exist yet; business phase passes but emits warnings until those artifacts are added.

## Deferred (Explicit P2+)

- Rollback/blue-green, backups/restore, resource swaps to managed services, bastion hosts.
