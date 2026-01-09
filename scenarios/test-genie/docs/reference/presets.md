# Test Presets Reference

## Overview

Test Genie provides preconfigured presets that bundle common testing patterns. Presets make it easy to run the right tests for your situation without remembering individual phase configurations.

## Available Presets

### Quick

**Purpose**: Fast sanity check during development

```bash
test-genie execute my-scenario --preset quick
```

**Phases included:**
| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates files and config | 15s |
| Standards | scenario-auditor standards rules | 60s |
| Docs | Validates Markdown, mermaid, links | 60s |
| Unit | Runs unit tests | 60s |

**Total time**: ~2 minutes

**Use when:**
- Making quick code changes
- Running in pre-commit hooks
- Need fast feedback

**Skips:**
- Integration tests (requires running scenario)
- Performance tests
- Business logic tests

---

### Smoke

**Purpose**: Verify core functionality works

```bash
test-genie execute my-scenario --preset smoke
```

**Phases included:**
| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates files and config | 15s |
| Standards | scenario-auditor standards rules | 60s |
| Lint | Type checking and linting | 30s |
| Docs | Validates Markdown, mermaid, links | 60s |
| Integration | Basic connectivity tests | 120s |

**Total time**: ~4-5 minutes

**Use when:**
- Before pushing to remote
- After dependency updates
- Quick integration verification

**Skips:**
- Full business logic tests
- Performance benchmarks

---

### Comprehensive

**Purpose**: Full validation before release

```bash
test-genie execute my-scenario --preset comprehensive
```

**Phases included:**
| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates files and config | 15s |
| Standards | scenario-auditor standards rules | 60s |
| Dependencies | Checks resources available | 30s |
| Lint | Type checking and linting | 30s |
| Docs | Validates Markdown, mermaid, links | 60s |
| Smoke | UI handshake / iframe-bridge | 90s |
| Unit | Runs unit tests | 60s |
| Integration | Full API/UI testing | 120s |
| Playbooks | BAS browser automation | 120s |
| Business | End-to-end workflows | 180s |
| Performance | Benchmarks and load tests | 60s |

**Total time**: ~10+ minutes

**Use when:**
- Before merging PRs
- Pre-deployment validation
- Full test coverage needed

**Includes everything.**

---

## Preset Comparison

```mermaid
graph LR
    subgraph Quick
        Q1[Structure]
        Q2[Standards]
        Q3[Docs]
        Q4[Unit]
    end

    subgraph Smoke
        S1[Structure]
        S2[Standards]
        S3[Lint]
        S4[Docs]
        S5[Integration]
    end

    subgraph Comprehensive
        C1[Structure]
        C2[Standards]
        C3[Dependencies]
        C4[Lint]
        C5[Docs]
        C6[Smoke]
        C7[Unit]
        C8[Integration]
        C9[Playbooks]
        C10[Business]
        C11[Performance]
    end

    style Quick fill:#e8f5e9
    style Smoke fill:#fff3e0
    style Comprehensive fill:#e3f2fd
```

| Feature | Quick | Smoke | Comprehensive |
|---------|-------|-------|---------------|
| Structure validation | ✅ | ✅ | ✅ |
| Standards enforcement | ✅ | ✅ | ✅ |
| Dependency check | ❌ | ❌ | ✅ |
| Unit tests | ✅ | ✅ | ✅ |
| Integration tests | ❌ | ✅ | ✅ |
| Business logic | ❌ | ❌ | ✅ |
| Performance | ❌ | ❌ | ✅ |
| Requirements sync | ❌ | ❌ | ✅ |
| **Typical time** | ~1 min | ~4 min | ~8 min |

## Custom Presets

Define custom presets in `.vrooli/testing.json`:

```json
{
  "presets": {
    "ci-fast": {
      "phases": ["structure", "unit"],
      "timeout": 120,
      "failFast": true
    },
    "nightly": {
      "phases": ["structure", "dependencies", "unit", "integration", "business", "performance"],
      "timeout": 3600,
      "failFast": false,
      "syncRequirements": true
    }
  }
}
```

Use custom presets:
```bash
test-genie execute my-scenario --preset ci-fast
```

## Phase Configuration

### Override Phase Timeouts

```json
{
  "phases": {
    "unit": {
      "timeout": 120,
      "enabled": true
    },
    "performance": {
      "timeout": 300,
      "enabled": false
    }
  }
}
```

### Skip Phases

```bash
# Skip specific phases
test-genie execute my-scenario --preset comprehensive --skip performance
test-genie execute my-scenario --preset comprehensive --skip standards

# Or in config
{
  "phases": {
    "performance": {
      "enabled": false
    }
  }
}
```

## See Also

- [Custom Presets Guide](../guides/custom-presets.md) - Step-by-step guide to creating custom presets
- [Phases Overview](../phases/README.md) - Detailed phase definitions
- [Phased Testing](../guides/phased-testing.md) - Understanding phases
- [CLI Commands](cli-commands.md) - CLI reference
