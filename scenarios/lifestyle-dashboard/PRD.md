# Product Requirements Document (PRD)

## Overview
Purpose: Unified personal lifestyle intelligence dashboard providing shared data model, correlation engine, and analytics layer for 9 domain-specific health/wellness scenarios
Target users: Single user (Matt) on personal Vrooli server — no multi-tenant, no auth, no SaaS
Deployment surfaces: Web UI (dashboard + domain deep-dives), REST API (events, queries, domains), Go SDK for domain integration
Value proposition: Cross-domain health insights that no single-domain app can provide, with zero-effort automation as the default interaction mode

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Shared event schema & storage | SQLite events table with JSON payloads, common envelope (id UUID, timestamp ISO-8601, domain, event_type, payload JSON), indexed by domain+timestamp. Includes causality markers: is_intervention (bool), hypothesis_id (UUID nullable)
- [ ] OT-P0-002 | Domain registration & discovery | Dynamic registration via API, capability exposure, health status tracking for domain scenarios. Domains start/stop independently
- [ ] OT-P0-003 | Cross-domain query API | Time-range filters, domain filters, basic aggregations, correlation queries across all registered domains
- [ ] OT-P0-004 | Unified dashboard UI | Mobile-first responsive design. Timeline view across all domains, per-domain collapsible sections linking to deep-dives, trend charts (7d/30d/90d), daily Lifestyle Score display
- [ ] OT-P0-005 | Daily brief system | Morning brief (7am default: what to do today, experiments in progress, yesterday summary), evening review (9pm default: what happened, compliance rates, tomorrow preview), cross-domain consolidation
- [ ] OT-P0-006 | Storage management UI | Settings page showing database size and large files, ability to clean all or selected data, manage large files

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Correlation engine | Automated statistical correlation across domains with significance tracking (p-value, confidence interval). Configurable minimum threshold (default: 14 data points). "Needs more data" indicator for insufficient samples
- [ ] OT-P1-002 | Weekly digest | "What Changed?" summary every Sunday (6pm default) comparing current week to rolling 4-week baseline. Shows deltas, new correlations discovered, medium-term trends
- [ ] OT-P1-003 | Composite Lifestyle Score | Daily 0-100 score combining all active domains. Weights configurable (sleep heavy, socialization light). Trajectory visualization over time
- [ ] OT-P1-004 | Experiment framework | N=1 experiments with hypothesis, intervention, measurement, duration. Progress tracking with compliance rate. Outcome reports with statistical analysis. Leverages causality markers (is_intervention, hypothesis_id)
- [ ] OT-P1-005 | Notification consolidation | Central scheduler collecting domain notifications into minimal daily touchpoints
- [ ] OT-P1-006 | Medical export | Comprehensive PDF health reports combining all domain data for healthcare providers
- [ ] OT-P1-007 | Natural language queries | Ollama-powered questions against event data ("How has my sleep been this month?", "What changed when I started creatine?")

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Domain SDK | Go package with event schema types, registration client, helper functions for building new domain scenarios
- [ ] OT-P2-002 | Smart recommendations | AI-powered optimization suggestions using cross-domain data
- [ ] OT-P2-003 | Hardware integration framework | Abstraction layer for wearables and sensors with dependency injection for future device support (Oura, Whoop, Apple Watch, Garmin)
- [ ] OT-P2-004 | Historical import | Tools to import data from existing apps (MyFitnessPal exports, Apple Health data, etc.)

## Tech Direction Snapshot
Preferred stacks: Go-based API backend, React + TypeScript + Vite frontend, SQLite single-file database (lifestyle.db)
Preferred storage: SQLite (single file, JSON text columns for payloads) — chosen for portability to mobile/Electron. No PostgreSQL, no Redis
Integration strategy: Event-driven architecture with domain registration, Ollama for AI features (P1), Qdrant for semantic search (P2)
Non-goals: Multi-tenant support, authentication layer, GDPR compliance, gamification in dashboard (domains may gamify independently), PostgreSQL/Redis dependencies

## Dependencies & Launch Plan
Required resources: SQLite (embedded, no external resource required)
Scenario dependencies: At least one domain scenario (nootropics-tracker recommended as first), optional Ollama for P1 features
Operational risks: Event schema evolution complexity, correlation engine surfacing noise as insight, domain scenarios failing independently, SQLite single-writer constraint (all writes via API)
Launch sequencing: Shared event schema → Domain registration → Query API → Dashboard UI → Daily briefs → Storage management → Correlation engine → Experiments

## UX & Branding
User experience: Dashboard-style interface optimized for quick daily check-ins, data-dense visualization, minimal required interaction. Composite Lifestyle Score as primary health indicator
Visual design: Grafana meets Apple Health aesthetic, dark mode default, minimal chrome, maximum information density
Accessibility: Mobile-first design (phone is primary for daily briefs), touch-friendly targets (44x44px minimum), keyboard navigation, clear focus indicators
