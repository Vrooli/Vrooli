# Product Requirements Document (PRD)

## Overview
Purpose: Unified personal lifestyle intelligence dashboard providing shared data model, correlation engine, and analytics layer for 9 domain-specific health/wellness scenarios
Target users: Single user (Matt) on personal Vrooli server — no multi-tenant, no auth, no SaaS
Deployment surfaces: Web UI (dashboard + domain deep-dives), REST API (events, queries, domains), Go SDK for domain integration
Value proposition: Cross-domain health insights that no single-domain app can provide, with zero-effort automation as the default interaction mode

## Operational Targets

### P0 – Must ship for viability
- [ ] OT-P0-001 | Shared event schema & storage | Time-series events in PostgreSQL with JSONB payloads, common envelope (timestamp, domain, event_type, payload), indexed by domain+timestamp
- [ ] OT-P0-002 | Domain registration & discovery | Dynamic registration, capability exposure, health status tracking for domain scenarios
- [ ] OT-P0-003 | Cross-domain query API | Time-range filters, domain filters, basic aggregations, correlation queries across all registered domains
- [ ] OT-P0-004 | Unified dashboard UI | Timeline view across all domains, per-domain sections linking to deep-dives, trend charts, responsive design
- [ ] OT-P0-005 | Daily brief system | Morning brief (what to do today), evening review (what happened, notable patterns), consolidated from all active domains

### P1 – Should have post-launch
- [ ] OT-P1-001 | Correlation engine | Automated statistical correlation across domains with significance tracking and configurable minimum data threshold
- [ ] OT-P1-002 | Experiment framework | N=1 experiments with hypothesis, intervention, measurement, duration, compliance tracking, outcome reports
- [ ] OT-P1-003 | Notification consolidation | Central scheduler collecting domain notifications into minimal daily touchpoints
- [ ] OT-P1-004 | Medical export | Comprehensive PDF health reports combining all domain data for healthcare providers
- [ ] OT-P1-005 | Natural language queries | Ollama-powered questions against event data ("How has my sleep been this month?")

### P2 – Future / expansion
- [ ] OT-P2-001 | Domain SDK | Go package with event schema types, registration client, helper functions for building new domain scenarios
- [ ] OT-P2-002 | Smart recommendations | AI-powered optimization suggestions using cross-domain data
- [ ] OT-P2-003 | Hardware integration framework | Abstraction layer for wearables and sensors (Bluetooth LE, WiFi APIs, device-specific adapters)
- [ ] OT-P2-004 | Historical import | Tools to import data from existing apps (MyFitnessPal exports, Apple Health data, etc.)

## Tech Direction Snapshot
Preferred stacks: Go-based API backend, React + TypeScript + Vite frontend, PostgreSQL time-series schema with JSONB
Preferred storage: PostgreSQL (lifestyle schema with events table), Redis for real-time state and aggregation caching
Integration strategy: Event-driven architecture with domain registration, Ollama for AI features (P1), Qdrant for semantic search (P2)
Non-goals: Multi-tenant support, authentication layer, GDPR compliance, gamification in dashboard (domains may gamify independently)

## Dependencies & Launch Plan
Required resources: PostgreSQL instance, Redis instance
Scenario dependencies: At least one domain scenario (nootropics-tracker recommended as first), optional Ollama for P1 features
Operational risks: Event schema evolution complexity, correlation engine surfacing noise as insight, domain scenarios failing independently
Launch sequencing: Shared event schema → Domain registration → Query API → Dashboard UI → Daily briefs → Correlation engine → Experiments

## UX & Branding
User experience: Dashboard-style interface optimized for quick daily check-ins, data-dense visualization, minimal required interaction
Visual design: Grafana meets Apple Health aesthetic, dark mode default, minimal chrome, maximum information density
Accessibility: Mobile-responsive for phone-based daily briefs, keyboard navigation, clear focus indicators
