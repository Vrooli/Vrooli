# Enhanced Plan: Lifestyle Dashboard

## Overview

The Lifestyle Dashboard is the **foundation scenario** for a personal lifestyle intelligence system. It provides the shared data model, event bus, correlation engine, and unified UI that 9 domain-specific scenarios (nootropics, sleep, skincare, diet, exercise, meditation, socialization, learning, biomarkers) integrate with.

**Philosophy:** Automated by default, customizable on demand. Everything runs hands-off — the system tells you what to do and only asks for input when a genuine decision is required.

**Target:** Single user (Matt), personal Vrooli server. No multi-tenant, no auth, no SaaS complexity.

**Moat:** Cross-domain health correlations that no single-domain app can provide.

## Clarifications Applied

| Question | Answer | Impact |
|----------|--------|--------|
| Wearable device | None yet — use dependency injection for future device support + manual data entry | Wearable integration deferred to P2; P0/P1 designs must support manual entry |
| Mobile support | Both responsive, but **mobile is most important** | Mobile-first design, not desktop-primary. Daily briefs optimized for phone. |
| Data retention | Keep all data indefinitely + **UI storage management** (view size, clean data, manage large files) | Add settings page with storage management tools |
| Correlation threshold | Configurable minimum sample size (default: 14 data points) | User can tune threshold in settings |
| First domain scenario | Nootropics Tracker (lowest friction, no hardware requirement) | Phase 1 validates integration contract with nootropics |

## Suggestions Integrated

### Accepted

| Suggestion | Integration |
|------------|-------------|
| **Composite health score** (s1) | Add daily 0-100 "Lifestyle Score" to dashboard — weighted combination of all active domains. Sleep heavily weighted, socialization lighter. Personalize weights over time based on correlation with subjective well-being. No gamification — just trajectory tracking. |
| **Weekly digest** (s2) | Add "What Changed?" weekly summary (Sunday 6pm default). Compares current week to rolling 4-week baseline. Shows deltas (sleep up 8%, exercise consistency down), new correlations discovered, and medium-term trends that daily views miss. Single weekly touchpoint. |
| **Causality markers** (s3) | Event schema includes `is_intervention` (bool) and `hypothesis_id` (UUID, nullable). Events tagged as interventions automatically become candidates for correlation analysis. Makes experiment framework a first-class data model concern. |
| **SQLite-only storage** (s-mmjrnthj) | **Major architectural change:** Use SQLite as the only database (plus files/folders for large content). No PostgreSQL, no Redis. This dramatically improves portability for future mobile/Electron apps. Performance is not a concern for personal single-user use. Follows storage-steer skill patterns. |

### Not Accepted

*None — all suggestions were accepted.*

## Refined Scope

### Included (Must Have) — P0

1. **Shared Event Schema & Storage**
   - SQLite database with events table (JSONB-equivalent via JSON columns)
   - Common envelope: id (UUID), timestamp (ISO-8601), domain (string), event_type (string), payload (JSON)
   - Causality fields: is_intervention (bool), hypothesis_id (UUID nullable)
   - Indexed by domain+timestamp for efficient queries
   - Extensible without migrations when adding new domains

2. **Domain Registration & Discovery**
   - Dynamic domain registration via API
   - Capability exposure (what events a domain emits/consumes)
   - Health status tracking (periodic polling)
   - Domains can start/stop independently

3. **Cross-Domain Query API**
   - Time-range and domain filters
   - Basic aggregations (count, avg, sum by time bucket)
   - Correlation queries between event types

4. **Unified Dashboard UI**
   - Mobile-first responsive design (phone-optimized, desktop-capable)
   - Timeline view across all domains
   - Per-domain collapsible sections with deep-dive links
   - Trend charts (7d/30d/90d windows)
   - Daily Lifestyle Score display

5. **Daily Brief System**
   - Morning brief (default 7am): today's actions, experiments in progress, yesterday summary
   - Evening review (default 9pm): what happened, compliance rates, tomorrow preview
   - Cross-domain consolidation (domains contribute structured brief content)

6. **Storage Management UI**
   - Settings page showing database size and large files
   - Ability to clean all or selected data
   - Typical storage management operations

### Included (Should Have) — P1

7. **Correlation Engine**
   - Automated daily correlation analysis across domains
   - Statistical significance tracking (p-value, confidence interval)
   - Configurable minimum threshold (default: 14 data points)
   - "Needs more data" indicator for insufficient samples

8. **Weekly Digest**
   - "What Changed?" summary every Sunday
   - Compares to 4-week rolling baseline
   - Surfaces medium-term trends and new correlations

9. **Composite Lifestyle Score**
   - Daily 0-100 score combining all active domains
   - Domain weights configurable (sleep heavy, socialization light)
   - Trajectory visualization over time

10. **Experiment Framework**
    - N=1 experiment definition (hypothesis, intervention, measurement, duration)
    - Progress tracking with compliance rate
    - Outcome reports with statistical analysis

11. **Notification Consolidation**
    - Central scheduler preventing notification fatigue
    - Minimal daily touchpoints from 9+ domains

12. **Medical Export**
    - PDF health reports combining all domain data
    - Designed for sharing with healthcare providers

13. **Natural Language Queries**
    - Ollama-powered questions against event data
    - Examples: "How has my sleep been this month?" "What changed when I started creatine?"

### Excluded (Out of Scope)

- **Multi-tenant support** — personal use only, no auth layer
- **GDPR/HIPAA compliance** — no regulatory overhead for personal server
- **Cloud sync** — local-first, no third-party analytics
- **Gamification in dashboard** — domains may gamify independently, but dashboard is data-focused
- **PostgreSQL/Redis** — SQLite-only for portability

### Deferred (Future) — P2

- **Domain SDK** — Go package for building new domain scenarios (when we have 3+ domains)
- **Smart Recommendations** — AI-powered optimization suggestions
- **Hardware Integration Framework** — Wearable/sensor abstraction layer (after wearable purchase)
- **Historical Import** — Tools to import from MyFitnessPal, Apple Health, etc.

## Implementation Notes

### Technical Stack

- **API:** Go (Vrooli patterns, api-core/storage for filesystem)
- **UI:** React + Vite + TypeScript
- **Database:** SQLite (single file, `lifestyle.db`)
- **Filesystem:** api-core/storage classes for runtime state
- **AI:** Ollama (P1 features — NL queries, experiment design)
- **Vector:** Qdrant (P2 semantic search, if needed)

### Event Schema Design

```sql
CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    timestamp TEXT NOT NULL,           -- ISO-8601
    domain TEXT NOT NULL,              -- e.g., "nootropics", "sleep"
    event_type TEXT NOT NULL,          -- e.g., "intake.logged", "sleep.recorded"
    payload TEXT NOT NULL,             -- JSON blob
    is_intervention INTEGER DEFAULT 0, -- 0 = observation, 1 = intervention
    hypothesis_id TEXT,                -- UUID if part of experiment
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_events_domain_ts ON events(domain, timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_hypothesis ON events(hypothesis_id) WHERE hypothesis_id IS NOT NULL;
```

### Domain Integration Contract

- Emit events to `events` table via dashboard API
- Register via `POST /api/v1/domains/register` (name, capabilities, health endpoint)
- Expose `/api/v1/health` for health checks
- Optional: expose domain-specific UI routes (deep-dive views)
- Provide structured brief content via `GET /api/v1/brief/contribute`

### Key Constraints

- Works gracefully with 1+ domains active (no minimum domain count)
- No schema migrations when adding new domains (JSON payloads)
- SQLite single-writer constraint: API handles all writes, UI reads via API
- Mobile-first design: touch targets, readable on small screens

### SQLite Portability Benefits

Per accepted suggestion s-mmjrnthj and storage-steer skill:
- Single file database — easy backup, copy, restore
- No separate database server to manage
- Works identically on desktop (Electron), mobile (via capacitor/native), and server
- Well-tested Go driver (modernc.org/sqlite or mattn/go-sqlite3)
- Sufficient performance for single-user with thousands of events

## Success Criteria

- [ ] Dashboard displays unified timeline from 2+ active domains
- [ ] Cross-domain queries return meaningful correlations (e.g., nootropics x sleep)
- [ ] Daily brief consolidates information from all active domains
- [ ] New domain scenario can integrate in <1 hour following the contract
- [ ] Daily Lifestyle Score updates automatically from domain data
- [ ] Weekly digest generates and highlights meaningful changes
- [ ] Mobile view is fully functional for daily brief consumption
- [ ] SQLite database works correctly for all CRUD operations

## Readiness Gate

- [x] Vision document complete
- [x] PRD with operational targets defined
- [x] Domain scenarios documented
- [x] All 5 clarifying questions answered
- [x] All 4 suggestions reviewed and accepted
- [x] Technical approach validated (SQLite portability)
- [x] Scope clearly defined (P0/P1/P2 boundaries)
- [x] Success criteria measurable
- [x] Archive materials incorporated into staging artifacts

**Ready for processing:** Yes

## Build Sequence

1. **Phase 1:** Lifestyle Dashboard + Nootropics Tracker (validate integration)
2. **Phase 2:** Sleep Tracker (after wearable purchase) + Diet & Nutrition
3. **Phase 3:** Remaining domains based on value/effort

## Staging Artifacts Produced

- `enhance/prd-context.md` — Context brief for prd-control-tower consumption, incorporating all clarifications and accepted suggestions, with SQLite storage direction
- `enhance/requirements-context.md` — Synthesized requirements from archive/requirements/, updated for SQLite architecture
- `enhance/doc-outlines.md` — Documentation outlines for README, RESEARCH, and PROBLEMS entries
