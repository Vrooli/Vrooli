# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Provide a configurable, resilient health observatory for measuring and tracking scenario completeness scores across the Vrooli ecosystem. Replaced the legacy JS-based scoring (formerly under `scripts/scenarios/lib/`) with this scenario; the legacy directory has now been removed after the migration.
- **Primary users/verticals**:
  - Ecosystem-manager (programmatic API consumer for autosteer decisions)
  - Human developers (understanding why scenarios score as they do)
  - AI agents (actionable guidance on what to improve)
  - CLI users (via `vrooli scenario completeness`)
- **Deployment surfaces**: Go API, React UI, CLI integration
- **Value promise**:
  - Prevent single points of failure from tanking all scores (circuit breaker pattern)
  - Provide visibility into scoring calculations and what-if analysis
  - Track score history and trends to detect stalls
  - Enable granular configuration without code changes
  - Portable for desktop app deployment as ecosystem-manager dependency

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Core Score Calculation | Calculate completeness scores with 4 dimensions: Quality (50%), Coverage (15%), Quantity (10%), UI (25%)
- [ ] OT-P0-002 | Component Toggle API | Enable/disable individual scoring components and penalties via configuration
- [ ] OT-P0-003 | Per-Scenario Overrides | Support scenario-specific configuration that overrides global defaults
- [ ] OT-P0-004 | Score Retrieval API | GET endpoint to retrieve current score and breakdown for any scenario
- [ ] OT-P0-005 | Circuit Breaker | Auto-disable collectors that fail N consecutive times; prevent broken infrastructure from tanking scores
- [ ] OT-P0-006 | Weight Redistribution | When components are disabled, redistribute weights proportionally to remaining components
- [ ] OT-P0-007 | Graceful Degradation | Return partial scores when some collectors fail rather than failing entirely
- [ ] OT-P0-008 | Health Status API | Endpoint showing which collectors are healthy, degraded, or failed

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Score History Storage | Store score snapshots over time with timestamp, breakdown, and config used
- [ ] OT-P1-002 | Trend Analysis | Detect score improvements/regressions and stalls (unchanged score despite activity)
- [ ] OT-P1-003 | What-If Analysis | Simulate "if you fix X, your score would be Y" predictions
- [ ] OT-P1-004 | Dashboard UI | Visual overview of all scenarios with scores, trends, and health status
- [ ] OT-P1-005 | Configuration UI | Toggle switches for components with health indicators and presets
- [ ] OT-P1-006 | Scenario Detail View | Drill-down showing full breakdown, history chart, and recommendations
- [ ] OT-P1-007 | Prioritized Recommendations | Ranked list of improvements with estimated point impact
- [ ] OT-P1-008 | Presets System | Pre-defined configurations (e.g., "Skip E2E", "Code Quality Only", "Full Scoring")
- [ ] OT-P1-009 | Bulk Score Refresh | Recalculate scores for all scenarios in one operation

### 🟢 P2 – Future / expansion ideas

- [ ] OT-P2-001 | Cross-Scenario Comparison | Side-by-side comparison of multiple scenarios
- [ ] OT-P2-002 | Category Benchmarking | "Your scenario scores below average for its category"
- [ ] OT-P2-003 | What-If Simulator UI | Interactive UI for simulating changes and seeing impact
- [ ] OT-P2-004 | Export Reports | Generate markdown/JSON reports for sharing
- [ ] OT-P2-005 | Webhook Notifications | Alert when scores change significantly or collectors fail
- [ ] OT-P2-006 | Integration with Ecosystem-Manager | Direct API integration replacing `pkg/autosteer/metrics*.go`
- [ ] OT-P2-007 | CLI Enhancement | `vrooli scenario completeness` calls this API instead of JS
- [ ] OT-P2-008 | Score Badges | Embeddable badges showing scenario health for READMEs
- [ ] OT-P2-009 | Custom Collectors | Plugin system for adding new scoring dimensions
- [ ] OT-P2-010 | Anomaly Detection | Alert when scores change unexpectedly

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**:
  - API: Go with standard library HTTP server
  - UI: React + TypeScript + Vite (via react-vite template)
  - Storage: SQLite for score history (simple, portable), JSON files for configuration
- **Data + storage expectations**:
  - Score snapshots stored in SQLite database
  - Configuration in `~/.config/vrooli/scenario-completeness-scoring/scoring-config.json` (global) and per-scenario tracked defaults
  - No external database dependencies for portability
- **Integration strategy**:
  - REST API as primary interface
  - Ecosystem-manager calls API directly
  - CLI wrapper for `vrooli scenario completeness`
  - Future: shared Go library for direct import
- **Non-goals / guardrails**:
  - Does NOT run tests (that's browser-automation-studio's job)
  - Does NOT manage requirements (that's the scenario itself)
  - Does NOT execute agents (that's ecosystem-manager's job)
  - Purely scoring, configuration, and insights

## 🤝 Dependencies & Launch Plan

- **Required resources**: None (self-contained, uses SQLite)
- **Scenario dependencies**: None for core functionality; optional integration with:
  - `ecosystem-manager` (primary consumer)
  - `browser-automation-studio` (for e2e test status, but gracefully handles unavailability)
  - `tidiness-manager` (for code quality metrics, optional)
- **Operational risks**:
  - Migration from JS-based scoring requires careful validation
  - Circuit breaker thresholds need tuning based on real-world failure patterns
- **Launch sequencing**:
  1. P0: Core API with scoring and configuration
  2. Validation: Compare outputs against the archived JS datasets to confirm parity before the JS implementation was retired
  3. P1: UI and history tracking
  4. Integration: Update ecosystem-manager to use this API
  5. Deprecation: Legacy JS scoring removed; CLI now routes to this scenario instead

## 🎨 UX & Branding

### Dashboard View (Home)
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
│  ...                                                            │
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
│  │ Penalties  ━━━━━━━━━━━━━━━━━━━━  -0     (disabled)       │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  📈 History (last 30 days)                                      │
│  [sparkline chart showing score over time]                      │
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
│  Coverage                                                       │
│    ├─ Test Coverage Ratio                    🟢 OK      [✓]     │
│    └─ Requirement Depth                      🟢 OK      [✓]     │
│  UI                                                             │
│    ├─ Template Detection                     🟢 OK      [✓]     │
│    ├─ Component Complexity                   🟢 OK      [✓]     │
│    └─ E2E/Lighthouse Metrics                 🔴 Failed  [ ]     │
│                                              (auto-disabled)    │
│  Penalties                                                      │
│    ├─ Invalid Test Location                  🟢 OK      [✓]     │
│    ├─ Monolithic Test Files                  🟢 OK      [✓]     │
│    └─ ...                                                       │
├─────────────────────────────────────────────────────────────────┤
│  Circuit Breaker: [✓] Auto-disable after 3 consecutive failures │
│  Weight Redistribution: [✓] Redistribute disabled weights       │
├─────────────────────────────────────────────────────────────────┤
│  Scope: (○) Global  (●) This scenario only                      │
│                                                                 │
│  [Save] [Reset to Defaults]                                     │
└─────────────────────────────────────────────────────────────────┘
```

### What-If Simulator
```
┌─────────────────────────────────────────────────────────────────┐
│  What-If Analysis: prd-control-tower                            │
├─────────────────────────────────────────────────────────────────┤
│  Current Score: 58                                              │
│                                                                 │
│  Simulate Changes:                                              │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ [✓] Fix 4 failing operational targets      → +9 points    │ │
│  │ [✓] Add 5 new tests                        → +3 points    │ │
│  │ [ ] Replace template UI                    → +10 points   │ │
│  │ [ ] Disable UI scoring entirely            → reweight     │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  Projected Score: 70 (+12)                                      │
│  New Classification: Mostly Complete                            │
│                                                                 │
│  [Apply Config Changes] [Export as Task]                        │
└─────────────────────────────────────────────────────────────────┘
```

- **Accessibility**: WCAG 2.1 AA compliance; keyboard navigation; screen reader support
- **Voice & messaging**: Technical but approachable; focus on actionability over metrics jargon
- **Branding hooks**: Consistent with Vrooli design system; uses standard color coding for health states

## 📎 Appendix

### API Design

#### Scoring Endpoints
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/scores` | List all scenarios with current scores |
| GET | `/api/scores/{scenario}` | Get detailed score for one scenario |
| POST | `/api/scores/{scenario}/calculate` | Force recalculation |
| GET | `/api/scores/{scenario}/history` | Get score history |
| POST | `/api/scores/{scenario}/what-if` | Run what-if simulation |

#### Configuration Endpoints
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/config` | Get global scoring config |
| PUT | `/api/config` | Update global config |
| GET | `/api/config/scenarios/{scenario}` | Get scenario-specific overrides |
| PUT | `/api/config/scenarios/{scenario}` | Set scenario-specific overrides |
| GET | `/api/config/presets` | List available presets |
| POST | `/api/config/presets/{name}/apply` | Apply preset globally |

#### Health & Diagnostics Endpoints
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/health` | Overall system health |
| GET | `/api/health/collectors` | Status of each collector |
| POST | `/api/health/collectors/{name}/test` | Test specific collector |
| GET | `/api/health/circuit-breaker` | View auto-disabled components |
| POST | `/api/health/circuit-breaker/reset` | Re-enable all components |

#### Analysis Endpoints
| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/compare` | Compare multiple scenarios |
| GET | `/api/recommendations/{scenario}` | Get prioritized recommendations |
| GET | `/api/trends` | Cross-scenario trend analysis |

### Scoring Dimensions (from existing JS implementation)

**Quality (50 points)**
- Requirement pass rate: 20 points
- Operational target pass rate: 15 points
- Test pass rate: 15 points

**Coverage (15 points)**
- Test coverage ratio: 8 points
- Requirement depth: 7 points

**Quantity (10 points)**
- Requirements count vs threshold: 4 points
- Targets count vs threshold: 3 points
- Tests count vs threshold: 3 points

**UI (25 points)**
- Template detection: 10 points
- Component complexity: 5 points
- API integration: 6 points
- Routing: 1.5 points
- Code volume: 2.5 points

### Circuit Breaker Behavior

1. Collector attempts to gather metrics
2. If collector fails, increment failure counter
3. After N consecutive failures (configurable, default: 3):
   - Mark collector as "tripped"
   - Log warning
   - Redistribute weight to remaining collectors
4. Periodically retry tripped collectors (configurable interval)
5. If retry succeeds, reset failure counter and re-enable
6. Expose status via `/api/health/circuit-breaker`

-### References
- Current Go implementation: `scenarios/scenario-completeness-scoring/api/pkg/scoring`
- Configuration data: `scenarios/scenario-completeness-scoring/api/pkg/config` + `~/.config/vrooli/scenario-completeness-scoring/scoring-config.json`
- Legacy JS implementation (removed after migration): `scripts/scenarios/lib/completeness.js`
- Legacy configuration file (removed after migration): `scripts/scenarios/lib/completeness-config.json`
- Ecosystem-manager metrics: `scenarios/ecosystem-manager/api/pkg/autosteer/metrics*.go`
