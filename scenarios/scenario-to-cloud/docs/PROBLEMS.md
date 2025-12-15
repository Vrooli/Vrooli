# Problems / Risks

## Open Issues

- **Packager contract not implemented yet**: deployment-manager docs reference scenario-to-cloud as a stub; API/CLI contract needs to be finalized and implemented.
- **“Mini Vrooli” bundling semantics**: defining what to strip/include (scenarios/resources/packages) needs deterministic manifests and reproducible build rules.
- **Remote port overrides**: forcing fixed ports (UI 3000, API 3001, WS 3002) must be compatible with the lifecycle allocator and scenario assumptions.
- **Caddy + Let’s Encrypt edge cases**: HTTP-01 requires port 80/443 open and DNS already pointing at the VPS; preflight must be crisp and actionable.

## Deferred (Explicit P2+)

- Rollback/blue-green, backups/restore, resource swaps to managed services, bastion hosts.

