# ADR-002: Manifest-Driven Bundle Architecture

## Status

Accepted

## Context

Desktop, mobile, and cloud bundles need to package multiple components:
- UI assets (HTML, CSS, JS)
- API binaries (one per platform)
- Resource binaries (SQLite, embedded services)
- Secrets (some generated, some user-provided)
- Configuration (ports, health checks, migrations)

Early attempts used scripts to assemble bundles, but this approach had problems:
1. Scripts encoded assumptions that varied by scenario
2. No validation before packaging - failures discovered at runtime
3. No way to preview what a bundle would contain
4. Platform-specific logic scattered across multiple files

## Decision

Adopt a **manifest-driven architecture** where a single `bundle.json` file declares everything needed to assemble and run a bundle.

### Manifest Schema (v0.1)

```json
{
  "schema_version": "v0.1",
  "target": "desktop",
  "app": { "name": "...", "version": "...", "description": "..." },
  "ipc": { "mode": "loopback-http", "host": "127.0.0.1", "port": 47710 },
  "telemetry": { "file": "...", "upload_to": "..." },
  "ports": { "default_range": { "min": 47000, "max": 47999 }, "reserved": [] },
  "swaps": [{ "original": "postgres", "replacement": "sqlite", "reason": "..." }],
  "secrets": [{ "id": "...", "class": "...", "target": { "type": "env", "name": "..." } }],
  "services": [{
    "id": "...",
    "type": "ui-bundle|api-binary|resource",
    "binaries": { "darwin-arm64": { "path": "..." }, ... },
    "env": { ... },
    "ports": { "requested": [{ "name": "http", "range": { "min": ..., "max": ... } }] },
    "health": { "type": "http", "path": "/healthz", ... },
    "readiness": { "type": "dependencies_ready", ... },
    "dependencies": ["other-service-id"],
    "migrations": [{ "version": "...", "command": [...] }]
  }]
}
```

### Key Principles

1. **Single source of truth**: All bundle configuration in one file
2. **Declarative**: Describes *what* to bundle, not *how* to bundle it
3. **Validatable**: Schema can be checked before assembly
4. **Platform-agnostic**: Same manifest structure for desktop, mobile, cloud
5. **Versioned**: `schema_version` enables backward-compatible evolution

## Consequences

### Positive

- **Pre-flight validation**: Catch errors before bundling starts
- **Reproducibility**: Same manifest always produces same bundle
- **Tooling**: Easy to build visualization, diff, and validation tools
- **Documentation**: Manifest serves as executable documentation
- **Separation of concerns**: deployment-manager generates manifests; packagers consume them

### Negative

- **Schema maintenance**: Must version and migrate manifest schemas
- **Indirection**: Developers must learn manifest schema, not just write scripts
- **Complexity for simple cases**: Even trivial bundles need full manifest

### Neutral

- Manifest generation is separate from manifest consumption
- Multiple tools can generate manifests (CLI, UI, CI/CD)

## Alternatives Considered

### Convention-Based Assembly

Infer bundle structure from directory layout (e.g., `bundle/services/api/` automatically included).

**Rejected because**: Implicit behavior is hard to debug. No way to specify ports, health checks, or dependencies through directory structure.

### Dockerfile-Like DSL

A scripting language for bundle assembly (e.g., `COPY api /services/api`).

**Rejected because**: Turing-complete DSLs become maintenance burdens. Validation is harder than with declarative JSON.

### Per-Platform Configuration

Separate config files for each platform (desktop.json, mobile.json, etc.).

**Rejected because**: Duplication between files. Hard to keep in sync. Single manifest with platform-specific binaries is cleaner.

### YAML Instead of JSON

Use YAML for more human-readable manifests.

**Rejected because**: JSON has better tooling support (schema validation, IDE autocomplete). Comments can be added in a separate doc file.

## References

- [Example Manifests](../examples/manifests/)
- [Bundle API Documentation](../api/bundles.md)
- [Bundled Runtime Plan](/docs/plans/bundled-desktop-runtime-plan.md)
