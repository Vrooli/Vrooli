# Test Genie Phases

Test Genie uses a **11-phase progressive testing architecture** where each phase has specific responsibilities, timeouts, and dependencies. Phases execute sequentially, with earlier phases providing the foundation for later ones.

## Phase Overview

```mermaid
graph TB
    subgraph "Static Phases (No Runtime Required)"
        P1[1. Structure<br/>15s timeout<br/>Files, config, CLI]
        P2[2. Standards<br/>60s timeout<br/>scenario-auditor rules]
        P3[3. Dependencies<br/>30s timeout<br/>Tools, resources]
        P4[4. Lint<br/>30s timeout<br/>Type checking, linters]
        P5[5. Docs<br/>60s timeout<br/>Markdown, mermaid, links]
    end

    subgraph "Runtime Phases (Scenario Running)"
        P6[6. Smoke<br/>90s timeout<br/>UI load, iframe-bridge]
        P7[7. Unit<br/>60s timeout<br/>Go, Node, Python]
        P8[8. Integration<br/>120s timeout<br/>API, CLI, BATS]
        P9[9. Playbooks<br/>120s timeout<br/>BAS browser automation]
        P10[10. Business<br/>180s timeout<br/>Requirements validation]
        P11[11. Performance<br/>60s timeout<br/>Build benchmarks, Lighthouse]
    end

    P1 --> P2 --> P3 --> P4 --> P5 --> P6 --> P7 --> P8 --> P9 --> P10 --> P11

    P7 -.->|fail-fast| ABORT[Abort]
    P11 --> SYNC[Requirements Sync]

    style P1 fill:#e8f5e9
    style P2 fill:#e8f5e9
    style P3 fill:#e8f5e9
    style P4 fill:#e8f5e9
    style P5 fill:#e8f5e9
    style P6 fill:#fff9c4
    style P7 fill:#fff3e0
    style P8 fill:#fff3e0
    style P9 fill:#fff3e0
    style P10 fill:#fff3e0
    style P11 fill:#fff3e0
```

## Phase Summary

| Phase | Timeout | Optional | Requires Runtime | Purpose |
|-------|---------|----------|------------------|---------|
| [Structure](structure/README.md) | 15s | No | No | Validate files, config, CLI setup |
| [Standards](standards/README.md) | 60s | No | No | Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config) |
| [Dependencies](dependencies/README.md) | 30s | No | No | Verify tools and resources |
| [Lint](lint/README.md) | 30s | No | No | Type checking and linting (Go, TS, Python) |
| [Docs](docs/README.md) | 60s | No | No | Validate Markdown, mermaid, links, portability |
| [Smoke](smoke/README.md) | 90s | Yes | Yes | UI load and iframe-bridge validation |
| [Unit](unit/README.md) | 60s | No | No | Run unit tests (Go, Node, Python) |
| [Integration](integration/README.md) | 120s | Yes | Yes | Test API, CLI, component interactions |
| [Playbooks](playbooks/README.md) | 120s | Yes | Yes | Execute BAS browser automation |
| [Business](business/README.md) | 180s | Yes | Yes | Validate requirements coverage |
| [Performance](performance/README.md) | 60s | Yes | Yes | Build benchmarks, Lighthouse audits |

## Static vs Runtime Phases

**Static phases** (1-5) can run without the scenario being started:
- Validate files exist and are well-formed
- Enforce scenario standards (PRD/service.json/proxy setup)
- Check dependencies are installed
- Run type checking and linting
- Validate docs, links, and mermaid diagrams

**Runtime phases** (6-11) require the scenario to be running:
- Smoke tests need UI server running
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
| **smoke** | structure, standards, lint, docs, integration | ~4-5 min | Pre-push validation |
| **comprehensive** | All 11 phases | ~10+ min | Full validation before release |

See [Presets Reference](../reference/presets.md) for custom preset configuration.

## Phase Documentation

Each phase has its own documentation folder with detailed guides:

- **[Structure](structure/README.md)** - File validation, CLI approaches
- **[Standards](standards/README.md)** - Standards enforcement via scenario-auditor
- **[Dependencies](dependencies/README.md)** - Tool and resource verification
- **[Lint](lint/README.md)** - Type checking and linting (Go, TypeScript, Python)
- **[Docs](docs/README.md)** - Markdown, mermaid, link, and portability validation
- **[Smoke](smoke/README.md)** - UI load validation and iframe-bridge testing
- **[Unit](unit/README.md)** - Test runners, coverage, requirement tagging
- **[Integration](integration/README.md)** - CLI testing with BATS, API health checks
- **[Playbooks](playbooks/README.md)** - BAS browser automation workflows
- **[Business](business/README.md)** - Requirements validation and sync
- **[Performance](performance/README.md)** - Build benchmarks, Lighthouse audits

## See Also

- [Architecture](../concepts/architecture.md) - Go orchestrator design
- [Presets](../reference/presets.md) - Preset configurations
- [API Endpoints](../reference/api-endpoints.md) - REST API reference
- [Troubleshooting](../guides/troubleshooting.md) - Debug common issues
