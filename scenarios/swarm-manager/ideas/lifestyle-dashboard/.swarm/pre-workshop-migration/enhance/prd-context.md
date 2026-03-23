# PRD Context Brief — Lifestyle Dashboard

## What This Is

A unified personal lifestyle intelligence dashboard that serves as the foundation layer for 9 domain-specific health/wellness scenarios. It provides:
- Shared event schema and time-series storage (SQLite)
- Domain registration and discovery
- Cross-domain correlation engine
- Unified analytics UI (mobile-first)
- Daily brief system (morning/evening touchpoints)
- Weekly digest with trend analysis
- Composite daily Lifestyle Score

## Target User

Single user (Matt) on a personal Vrooli server. No multi-tenant, no authentication, no SaaS complexity. Designed for personal health optimization with an automation-first philosophy.

## Value Proposition

Cross-domain health insights that no single-domain app can provide. Example queries:
- "Show me sleep quality on days I took magnesium vs. days I didn't"
- "What changed when I started taking creatine?"
- "How does my exercise consistency correlate with focus scores?"

The composite Lifestyle Score provides at-a-glance health trajectory without gamification.

## Technical Direction

- **API:** Go backend (Vrooli patterns)
- **UI:** React + Vite + TypeScript
- **Database:** SQLite (single file, `lifestyle.db`) — chosen for portability to mobile/Electron
- **Filesystem:** api-core/storage for runtime state outside deploy directory
- **AI:** Ollama for NL queries and insights (P1)
- **Vector:** Qdrant for semantic search (P2, if needed)

### Why SQLite (Not PostgreSQL/Redis)

Per accepted suggestion s-mmjrnthj:
- This is a personal app — performance is not a concern
- SQLite + files is more portable than PostgreSQL
- Makes future mobile (Capacitor) or desktop (Electron) apps trivial
- Single file backup/restore
- No external database server to manage
- Follows storage-steer skill guidance for simple, single-user scenarios

## Operational Targets

### P0 — Must ship for viability

| ID | Target | Description |
|----|--------|-------------|
| OT-P0-001 | Shared Event Schema & Storage | SQLite events table with JSON payloads, common envelope (id, timestamp, domain, event_type, payload), indexed by domain+timestamp. Includes causality fields (is_intervention, hypothesis_id) for experiment tracking. |
| OT-P0-002 | Domain Registration & Discovery | Dynamic registration via API, capability exposure, health status tracking. Domains start/stop independently. |
| OT-P0-003 | Cross-Domain Query API | Time-range filters, domain filters, basic aggregations, correlation queries. |
| OT-P0-004 | Unified Dashboard UI | Mobile-first responsive design. Timeline view, per-domain sections, trend charts, daily Lifestyle Score. |
| OT-P0-005 | Daily Brief System | Morning brief (what to do today), evening review (what happened, patterns). Cross-domain consolidation. |
| OT-P0-006 | Storage Management UI | Settings page showing database size, ability to clean data, manage large files. |

### P1 — Should have post-launch

| ID | Target | Description |
|----|--------|-------------|
| OT-P1-001 | Correlation Engine | Automated statistical correlation with significance tracking. Configurable minimum threshold (default: 14 data points). |
| OT-P1-002 | Weekly Digest | "What Changed?" summary comparing current week to 4-week baseline. Surfaces medium-term trends. |
| OT-P1-003 | Composite Lifestyle Score | Daily 0-100 score from all active domains. Weighted (sleep heavy, socialization light). Personalizable. |
| OT-P1-004 | Experiment Framework | N=1 experiments with hypothesis, intervention, measurement, duration, compliance tracking, outcome reports. |
| OT-P1-005 | Notification Consolidation | Central scheduler preventing notification fatigue from 9+ domains. |
| OT-P1-006 | Medical Export | PDF health reports combining all domain data for healthcare providers. |
| OT-P1-007 | Natural Language Queries | Ollama-powered questions against event data. |

### P2 — Future expansion

| ID | Target | Description |
|----|--------|-------------|
| OT-P2-001 | Domain SDK | Go package with event schema types, registration client, helpers. |
| OT-P2-002 | Smart Recommendations | AI-powered optimization suggestions based on cross-domain data. |
| OT-P2-003 | Hardware Integration Framework | Wearable and sensor abstraction layer (dependency injection ready). |
| OT-P2-004 | Historical Import | Tools to import from MyFitnessPal, Apple Health, etc. |

## Dependencies

- **SQLite** — embedded, no external resource required
- **Ollama resource** — P1 features (NL queries)
- **Qdrant resource** — P2 features (semantic search)
- **At least one domain scenario** — nootropics-tracker recommended first

## Key Constraints

- Single user, local only — no auth layer needed
- Must work with domains starting/stopping independently
- Dashboard must be useful with only 1-2 domains active (graceful degradation)
- Event schema must be extensible without migrations (JSON payloads)
- SQLite single-writer: all writes go through API
- Mobile-first: touch-friendly, readable on small screens

## UX Direction

- **Mobile-first responsive** — phone is primary for daily briefs
- Dashboard aesthetic: Grafana meets Apple Health
- Dark mode default (late-night data checking common)
- Minimal chrome, maximum information density
- No gamification in dashboard (domains may gamify independently)
- Daily Lifestyle Score as primary health indicator

## Domain Scenarios (for reference)

Nine domains integrate with this dashboard:

| Phase | Domain | Notes |
|-------|--------|-------|
| 1 | Nootropics Tracker | First to build — lowest friction, no hardware |
| 1 | Sleep Tracker | Blocked on wearable purchase, highest passive signal |
| 2 | Diet & Nutrition | Optimizes existing habits |
| 2 | Exercise Planner | Structures existing activity |
| 3 | Skincare Manager | Requires behavior change |
| 3 | Biomarkers & Preventive Care | High medical value |
| 3 | Meditation & Focus | Benefits from sleep/nootropics data |
| 3 | Learning & Brain Games | Lower urgency |
| 3 | Socialization & Mental Health | Most subjective |

## Accepted Suggestions Summary

1. **Composite Lifestyle Score** — Daily 0-100 weighted score from all domains
2. **Weekly Digest** — "What Changed?" comparing to 4-week baseline
3. **Causality Markers** — Event schema supports is_intervention and hypothesis_id
4. **SQLite-only Storage** — Replaces PostgreSQL/Redis for portability

## Clarified Decisions

| Topic | Decision |
|-------|----------|
| Wearable device | None yet; design for dependency injection + manual entry |
| Mobile support | **Mobile-first** (not just responsive) |
| Data retention | Keep indefinitely + UI storage management tools |
| Correlation threshold | Configurable (default: 14 data points) |
| First domain | Nootropics Tracker |

## P2 Monetization Notes (future reference)

When ready to commercialize:
- Individual domains can be cleaned up for SaaS
- Dashboard becomes the premium "pro" tier
- Regulatory compliance (HIPAA, GDPR) would need to be added
- Multi-tenant auth layer required
- SQLite → PostgreSQL migration path for scale
