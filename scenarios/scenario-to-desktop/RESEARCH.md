# Research Notes

## Related Scenarios & Resources
- **deployment-manager** and **scenario-dependency-analyzer** will consume desktop telemetry and bundle manifests once offline builds mature.
- **app-monitor**/**Cloudflare tunnel** remain the primary exposure path for thin clients; **scenario-auditor** references deployment telemetry for health signals.
- Resources that matter for bundling: `postgres` (build metadata), `redis` (cache), `browserless` (screenshot validation), and eventually local model runtimes for offline inference.

## Open Questions
- How will bundle manifests express dependency swaps (e.g., Ollama/Postgres replacements) so desktop runtime can honor them without shipping heavy services?
- Should telemetry storage move from JSONL to a lightweight embedded store when offline bundles land to enable richer diagnostics?
- What minimum `vrooli` shim is needed on desktop to keep CLI compatibility when no global install exists?

## External Inspiration
- Electron auto-update ecosystems (Squirrel/MSIX/PKG + differential updates) for release channels.
- Portable runtime supervisors (Tauri sidecar handling, Deno deploy kits) for ideas on embedding binaries without root privileges.
- Offline-first app packaging patterns from VS Code and Obsidian (data dir versioning, extension sandboxing) for future bundled mode. 
