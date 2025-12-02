# Scenario Completeness Scoring

A configurable health observatory for measuring and tracking scenario completeness across the Vrooli ecosystem. Provides resilient scoring with circuit breaker patterns to prevent single points of failure from tanking all scores.

## Overview

This scenario replaces the legacy JavaScript-based completeness scoring (formerly under `scripts/scenarios/lib/`, now removed) with a proper Go API and React UI that can be:
- Deployed alongside ecosystem-manager
- Packaged for desktop app deployment
- Configured without code changes
- Monitored for collector health

### Key Capabilities

- **Configurable Scoring**: Enable/disable individual scoring components and penalties
- **Circuit Breaker**: Auto-disable collectors that fail consecutively
- **Weight Redistribution**: Automatically rebalance weights when components are disabled
- **Score History**: Track scores over time to detect trends and stalls
- **What-If Analysis**: Simulate improvements before making changes
- **Health Monitoring**: Real-time status of all scoring collectors

## Quick Start

```bash
# From repo root
cd scenarios/scenario-completeness-scoring

# Install dependencies
pnpm install --dir ui
cd api && go mod tidy && cd ..

# Build and start via lifecycle
make start

# Or use vrooli CLI
vrooli scenario start scenario-completeness-scoring
```

### CLI installation (cross-platform)

Install or rebuild the Go CLI binary into `~/.vrooli/bin`:

```bash
# Use the checked-out cli-core (default)
./cli/install.sh

# Or pin a published cli-core version for reproducible installs
CLI_CORE_VERSION=latest ./cli/install.sh

# Without the repo, run directly:
go run github.com/vrooli/cli-core/cmd/cli-installer@latest \
  --module ./scenarios/scenario-completeness-scoring/cli \
  --name scenario-completeness-scoring \
  --install-dir ~/.vrooli/bin
```

## Architecture

```
scenario-completeness-scoring/
├── api/                    # Go API server
│   ├── main.go            # Entry point
│   ├── scoring/           # Score calculation logic
│   ├── collectors/        # Metric collectors (one per dimension)
│   ├── config/            # Configuration management
│   └── history/           # Score history storage
├── ui/                     # React + TypeScript + Vite
│   └── src/
│       ├── pages/         # Dashboard, Detail, Config views
│       └── components/    # Reusable UI components
├── cli/                    # CLI wrapper
├── .vrooli/               # Lifecycle configuration
├── requirements/          # Operational targets
└── docs/                  # Documentation
```

## Feature Set

### 1. Score Calculation (Core)

Calculate completeness scores across 4 dimensions (100 points total):

| Dimension | Weight | Components |
|-----------|--------|------------|
| **Quality** | 50% | Requirement pass rate (20), Target pass rate (15), Test pass rate (15) |
| **Coverage** | 15% | Test coverage ratio (8), Requirement depth (7) |
| **Quantity** | 10% | Requirements count (4), Targets count (3), Tests count (3) |
| **UI** | 25% | Template detection (10), Component complexity (5), API integration (6), Routing (1.5), Code volume (2.5) |

### 2. Configuration System

**Global Configuration** (`~/.vrooli/scoring-config.json`):
```json
{
  "components": {
    "quality": {
      "requirement_pass_rate": true,
      "target_pass_rate": true,
      "test_pass_rate": true
    },
    "coverage": {
      "test_coverage_ratio": true,
      "requirement_depth": true
    },
    "ui": {
      "template_detection": true,
      "e2e_metrics": false
    }
  },
  "penalties": {
    "invalid_test_location": true,
    "monolithic_test_files": true
  },
  "circuit_breaker": {
    "enabled": true,
    "failure_threshold": 3,
    "retry_interval_seconds": 300
  }
}
```

**Per-Scenario Overrides** (`scenarios/{name}/.vrooli/scoring-config.json`):
- Inherits from global config
- Can enable/disable specific components for that scenario only

### 3. Circuit Breaker Pattern

Prevents broken infrastructure (e.g., browser-automation-studio) from affecting all scores:

1. Collector attempts to gather metrics
2. If collector fails, increment failure counter
3. After N consecutive failures (default: 3):
   - Mark collector as "tripped"
   - Log warning
   - Redistribute weight to remaining collectors
4. Periodically retry tripped collectors
5. If retry succeeds, reset and re-enable

### 4. Score History & Trends

- Store score snapshots with timestamp, breakdown, and config used
- Detect improvements/regressions over time
- Identify stalls (unchanged score despite activity)
- Sparkline visualizations in UI

### 5. What-If Analysis

Simulate changes before implementing:
- "If you fix 4 failing targets, score goes from 58 → 67"
- "Disabling UI scoring would reweight to 70"
- Export simulations as ecosystem-manager tasks

### 6. Health Monitoring

Real-time collector status:
- 🟢 **OK**: Collector working normally
- 🟡 **Degraded**: Intermittent failures
- 🔴 **Failed**: Circuit breaker tripped, auto-disabled

## UX Concepts

### Dashboard (Home)
```
┌─────────────────────────────────────────────────────────────────┐
│  Scenario Health Overview                          [⚙️ Config]  │
├─────────────────────────────────────────────────────────────────┤
│  ⚠️ 2 collectors unhealthy: e2e-tests, lighthouse              │
│     [Auto-disabled] [View Details]                              │
├─────────────────────────────────────────────────────────────────┤
│  Scenario               Score   Trend    Classification         │
│  ─────────────────────────────────────────────────────────────  │
│  🟢 landing-manager      87     ↑ +12    Nearly Ready           │
│  🟡 prd-control-tower    58     → 0      Functional Incomplete  │
│  🟡 knowledge-observatory 52    ↓ -3     Functional Incomplete  │
│  🔴 git-control-tower    23     ↑ +5     Foundation Laid        │
├─────────────────────────────────────────────────────────────────┤
│  [Calculate All] [Export Report] [Compare Selected]             │
└─────────────────────────────────────────────────────────────────┘
```

### Scenario Detail View
```
┌─────────────────────────────────────────────────────────────────┐
│  prd-control-tower                              Score: 58/100   │
│  Category: automation          Classification: Functional       │
├─────────────────────────────────────────────────────────────────┤
│  Score Breakdown                                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Quality    ████████████░░░░░░░░  28/50  (56%)            │  │
│  │ Coverage   ██████░░░░░░░░░░░░░░   8/15  (53%)            │  │
│  │ Quantity   ████████░░░░░░░░░░░░   7/10  (70%)            │  │
│  │ UI         ███████████████░░░░░  15/25  (60%)            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  📈 History (last 30 days)                                      │
│  [sparkline chart]                                              │
│                                                                 │
│  🎯 Top Recommendations                          Impact         │
│  1. Fix 4 failing operational targets            +9 points      │
│  2. Add 3 more integration tests                 +4 points      │
│  3. Increase requirement depth                   +3 points      │
│                                                                 │
│  [What-If Analysis] [View Full Breakdown] [Configure]           │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration Panel
```
┌─────────────────────────────────────────────────────────────────┐
│  Scoring Configuration                                          │
├─────────────────────────────────────────────────────────────────┤
│  Presets: [Default] [Skip E2E] [Code Quality Only] [Custom]     │
├─────────────────────────────────────────────────────────────────┤
│  Component Toggles                           Status    Enabled  │
│  ─────────────────────────────────────────────────────────────  │
│  Quality                                                        │
│    ├─ Requirement Pass Rate                  🟢 OK      [✓]     │
│    ├─ Target Pass Rate                       🟡 Degraded [✓]    │
│    └─ Test Pass Rate                         🟢 OK      [✓]     │
│  UI                                                             │
│    └─ E2E/Lighthouse Metrics                 🔴 Failed  [ ]     │
│                                              (auto-disabled)    │
├─────────────────────────────────────────────────────────────────┤
│  Circuit Breaker: [✓] Auto-disable after 3 failures            │
│  Weight Redistribution: [✓] Redistribute disabled weights       │
├─────────────────────────────────────────────────────────────────┤
│  Scope: (○) Global  (●) This scenario only                      │
│  [Save] [Reset to Defaults]                                     │
└─────────────────────────────────────────────────────────────────┘
```

## API Reference

### Scoring Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/scores` | List all scenarios with current scores |
| GET | `/api/scores/{scenario}` | Get detailed score for one scenario |
| POST | `/api/scores/{scenario}/calculate` | Force recalculation |
| GET | `/api/scores/{scenario}/history` | Get score history |
| POST | `/api/scores/{scenario}/what-if` | Run what-if simulation |

### Configuration Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/config` | Get global scoring config |
| PUT | `/api/config` | Update global config |
| GET | `/api/config/scenarios/{scenario}` | Get scenario-specific overrides |
| PUT | `/api/config/scenarios/{scenario}` | Set scenario-specific overrides |
| GET | `/api/config/presets` | List available presets |
| POST | `/api/config/presets/{name}/apply` | Apply preset globally |

### Health Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/health` | Overall system health |
| GET | `/api/health/collectors` | Status of each collector |
| POST | `/api/health/collectors/{name}/test` | Test specific collector |
| GET | `/api/health/circuit-breaker` | View auto-disabled components |
| POST | `/api/health/circuit-breaker/reset` | Re-enable all components |

### Analysis Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/compare` | Compare multiple scenarios |
| GET | `/api/recommendations/{scenario}` | Get prioritized recommendations |
| GET | `/api/trends` | Cross-scenario trend analysis |

## Integration

### With Ecosystem-Manager

Ecosystem-manager will call this API instead of using its internal `pkg/autosteer/metrics*.go`:

```go
// Before: internal metrics collection
metrics, err := metricsCollector.CollectMetrics(scenarioName, phaseLoops, totalLoops)

// After: call completeness-scoring API
resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/scores/%s",
    completenessPort, scenarioName))
```

### With CLI

The existing `vrooli scenario completeness` command will be updated to call this API:

```bash
# Current: runs JavaScript directly
vrooli scenario completeness my-scenario

# Future: calls this API
vrooli scenario completeness my-scenario  # same command, uses API internally
```

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `API_PORT` | API server port | Auto-assigned |
| `UI_PORT` | UI server port | Auto-assigned |
| `DATABASE_PATH` | SQLite database location | `data/scores.db` |
| `CONFIG_PATH` | Global config location | `~/.vrooli/scoring-config.json` |
| `VROOLI_ROOT` | Vrooli repository root | Auto-detected |

## Development

```bash
# Run tests
make test

# Run API only
cd api && go run main.go

# Run UI dev server
cd ui && pnpm dev

# Format code
cd api && gofumpt -w .
cd ui && pnpm format
```

## Classification Thresholds

| Score Range | Classification | Description |
|-------------|----------------|-------------|
| 96-100 | Production Ready | Excellent validation coverage |
| 81-95 | Nearly Ready | Final polish and edge cases |
| 61-80 | Mostly Complete | Needs refinement and validation |
| 41-60 | Functional Incomplete | Needs more features/tests |
| 21-40 | Foundation Laid | Core features in progress |
| 0-20 | Early Stage | Needs significant development |

## Related Documentation

- [PRD.md](./PRD.md) - Product requirements and operational targets
- [docs/PROGRESS.md](./docs/PROGRESS.md) - Development progress log
- [docs/PROBLEMS.md](./docs/PROBLEMS.md) - Known issues and deferred ideas
- [docs/RESEARCH.md](./docs/RESEARCH.md) - Background research

## References

- Current Go implementation: `api/pkg/scoring`, `api/pkg/collectors`, `api/pkg/config`
- Legacy JS implementation (archived, removed after migration): `scripts/scenarios/lib/completeness.js`
- Legacy configuration file (archived): `scripts/scenarios/lib/completeness-config.json`
- Ecosystem-manager metrics: `scenarios/ecosystem-manager/api/pkg/autosteer/metrics*.go`
