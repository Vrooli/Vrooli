# Test Genie Phases

Test Genie uses a **progressive testing architecture** where each phase has specific responsibilities, timeouts, and dependencies. Phases execute sequentially, with earlier phases providing the foundation for later ones.

> **Single source of truth.** The canonical phase set, ordering, and metadata are declared in the catalog at [`api/internal/orchestrator/phases/catalog.go`](../../api/internal/orchestrator/phases/catalog.go) (names in `types.go`). The catalog currently registers **13 phases**. This document is derived from that catalog — when phases change, update the catalog and let the CLI, presets, and validation follow automatically.

## Phase Overview

```mermaid
graph TB
    subgraph "Static Phases (No Runtime Required)"
        P1[1. Structure<br/>Files, config, CLI]
        P2[2. Contracts<br/>cli/manifest proto bindings]
        P3[3. UI Health<br/>ui/manifest slots, overlays]
        P4[4. Standards<br/>scenario-auditor rules]
        P5[5. Dependencies<br/>Tools, resources]
        P6[6. Quality<br/>Static quality contracts]
        P7[7. Docs<br/>Markdown, mermaid, links]
    end

    subgraph "Runtime Phases (Scenario Running)"
        P8[8. Performance<br/>Build benchmarks, Lighthouse]
        P9[9. Smoke<br/>UI load, iframe-bridge]
        P10[10. Unit<br/>Go, Node, Python]
        P11[11. Integration<br/>API, CLI, BATS]
        P12[12. Playbooks<br/>BAS browser automation]
        P13[13. Business<br/>Requirements validation]
    end

    P1 --> P2 --> P3 --> P4 --> P5 --> P6 --> P7 --> P8 --> P9 --> P10 --> P11 --> P12 --> P13

    P13 --> SYNC[Requirements Sync]

    style P1 fill:#e8f5e9
    style P2 fill:#e8f5e9
    style P3 fill:#e8f5e9
    style P4 fill:#e8f5e9
    style P5 fill:#e8f5e9
    style P6 fill:#e8f5e9
    style P7 fill:#e8f5e9
    style P8 fill:#fff3e0
    style P9 fill:#fff9c4
    style P10 fill:#fff3e0
    style P11 fill:#fff3e0
    style P12 fill:#fff3e0
    style P13 fill:#fff3e0
```

## Phase Summary

Listed in catalog (execution) order:

| Phase | Timeout | Optional | Requires Runtime | Purpose |
|-------|---------|----------|------------------|---------|
| [Structure](structure/README.md) | 15s | No | No | Validate files, config, CLI setup |
| [Contracts](contracts/README.md) | 60s | No | No | Validate cli/manifest.json bindings against proto descriptors (cli-health) |
| [UI Health](ui-health/README.md) | 60s | No | No | Validate ui/manifest.json bindings, slot directories, and overlay rules (ui-health) |
| [Standards](standards/README.md) | 60s | No | No | Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config) |
| [Dependencies](dependencies/README.md) | 30s | No | No | Verify tools and resources |
| [Quality](quality/README.md) | 120s | No | No | Delegates lint/type/static-quality contracts to quality-health |
| [Docs](docs/README.md) | 60s | No | No | Validate Markdown, mermaid, links, portability |
| [Performance](performance/README.md) | 60s | Yes | Yes | Build benchmarks, Lighthouse audits |
| [Smoke](smoke/README.md) | 90s | Yes | Yes | UI load and iframe-bridge validation |
| [Unit](unit/README.md) | 60s | No | No | Run unit tests (Go, Node, Python) |
| [Integration](integration/README.md) | 120s | Yes | Yes | Test API, CLI, component interactions |
| [Playbooks](playbooks/README.md) | 120s | Yes | Yes | Execute BAS browser automation |
| [Business](business/README.md) | 180s | Yes | Yes | Validate requirements coverage |

## Static vs Runtime Phases

**Static phases** (Structure, Contracts, UI Health, Standards, Dependencies, Quality, Docs) can run without the scenario being started:
- Validate files exist and are well-formed
- Validate CLI/UI manifest bindings against proto descriptors and slot layout
- Enforce scenario standards (PRD/service.json/proxy setup)
- Check dependencies are installed
- Run static-quality contract checks through Quality Health
- Validate docs, links, and mermaid diagrams

**Runtime phases** (Performance, Smoke, Unit, Integration, Playbooks, Business) may require the scenario to be running:
- Performance and smoke tests need the UI server running
- Unit tests may need scenario context
- Integration and playbooks need API/UI endpoints accessible
- Test real component interactions

## Exit Codes

All phases use consistent exit codes:

| Code | Meaning |
|------|---------|
| 0 | Phase passed |
| 1 | Phase failed (test failures, validation errors) |
| 2 | Phase skipped (optional phase, runtime unavailable) |

## Running Phases

### Via CLI

```bash
# Run specific phases
test-genie execute my-scenario --phases structure,unit

# Run all phases (comprehensive)
test-genie execute my-scenario --preset comprehensive

# Quick check (structure + unit)
test-genie execute my-scenario --preset quick
```

### Via Makefile

```bash
cd scenarios/my-scenario
make test              # Run comprehensive preset
make test-quick        # Run quick preset
```

### Via REST API

```bash
API_PORT=$(vrooli scenario port test-genie API_PORT)
curl -X POST "http://localhost:${API_PORT}/api/v1/test-suite/my-scenario/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"phases": ["structure", "unit", "integration"]}'
```

## Configuration

Override phase settings in `.vrooli/testing.json`:

```json
{
  "phases": {
    "unit": {
      "timeout": 120,
      "coverageWarn": 85,
      "coverageError": 75
    },
    "performance": {
      "enabled": false
    }
  }
}
```

## Presets

Presets bundle phases for common use cases:

| Preset | Phases | Duration | Use Case |
|--------|--------|----------|----------|
| **quick** | structure, standards, docs, unit | ~1-2 min | Fast feedback during development |
| **smoke** | structure, standards, quality, docs, integration | ~4-5 min | Pre-push validation |
| **comprehensive** | every registered phase, including quality | ~10+ min | Full validation before release |

See [Presets Reference](../reference/presets.md) for custom preset configuration.

## Phase Documentation

Each phase has its own documentation folder with detailed guides:

- **[Structure](structure/README.md)** - File validation, CLI approaches
- **[Contracts](contracts/README.md)** - cli/manifest.json proto-binding validation via cli-health
- **[UI Health](ui-health/README.md)** - ui/manifest.json bindings, slot directories, overlay rules via ui-health
- **[Standards](standards/README.md)** - Standards enforcement via scenario-auditor
- **[Dependencies](dependencies/README.md)** - Tool and resource verification
- **[Quality](quality/README.md)** - Static quality contracts delegated to Quality Health
- **[Docs](docs/README.md)** - Markdown, mermaid, link, and portability validation
- **[Performance](performance/README.md)** - Build benchmarks, Lighthouse audits
- **[Smoke](smoke/README.md)** - UI load validation and iframe-bridge testing
- **[Unit](unit/README.md)** - Test runners, coverage, requirement tagging
- **[Integration](integration/README.md)** - CLI testing with BATS, API health checks
- **[Playbooks](playbooks/README.md)** - BAS browser automation workflows
- **[Business](business/README.md)** - Requirements validation and sync

## See Also

- [Architecture](../concepts/architecture.md) - Go orchestrator design
- [Presets](../reference/presets.md) - Preset configurations
- [API Endpoints](../reference/api-endpoints.md) - REST API reference
- [Troubleshooting](../guides/troubleshooting.md) - Debug common issues
