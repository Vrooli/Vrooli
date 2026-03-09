# Lifestyle Dashboard — Product Requirements Document

## 1. Overview

The Lifestyle Dashboard is a personal lifestyle intelligence system that unifies domain-specific health and wellness scenarios into a single analytics, automation, and correlation layer. It runs locally on a Vrooli server for a single user.

**Purpose:** Provide the shared data model, event bus, correlation engine, and unified UI that all lifestyle domain scenarios (nootropics, sleep, skincare, diet, exercise, meditation, socialization, learning, biomarkers) integrate with.

**Target user:** Single user (Matt), personal server deployment. No multi-tenant, no auth, no SaaS.

**Value proposition:** Cross-domain health insights that no single-domain app can provide, with zero-effort automation as the default interaction mode.

## 2. Operational Targets

### P0 — Core (Without these, the scenario fails)

**OT-P0-1: Shared Event Schema & Storage**
Define and implement the shared event model that all lifestyle domain scenarios emit to. Time-series events stored in PostgreSQL with JSONB payloads, indexed by domain and timestamp. Must support arbitrary domain-specific payloads while enforcing a common envelope (timestamp, domain, event_type, payload).

**OT-P0-2: Domain Registration & Discovery**
Implement a mechanism for domain scenarios to register themselves with the dashboard. The dashboard discovers active domains, their capabilities, and their health status. Supports dynamic registration (domains can start/stop independently).

**OT-P0-3: Cross-Domain Query API**
API endpoints that allow querying events across domains with time-range filters, domain filters, and basic aggregations. Must support queries like "show me sleep quality on days I took magnesium vs. days I didn't." This is the core intelligence capability.

**OT-P0-4: Unified Dashboard UI**
React UI with a unified timeline view showing events from all registered domains. Per-domain sections that link out to domain-specific UIs. Basic trend charts. Responsive design (usable on phone for quick checks).

**OT-P0-5: Daily Brief System**
Automated "morning brief" and "evening review" that consolidate information across all active domains into minimal daily touchpoints. Morning: what to do today. Evening: what happened, any notable patterns. Delivered via the UI (push notifications are P1).

### P1 — Important Enhancements

**OT-P1-1: Correlation Engine**
Automated statistical correlation analysis across domains. Surfaces insights with confidence levels. Tracks statistical significance to avoid presenting noise as insight. Configurable minimum data threshold before generating correlations.

**OT-P1-2: Experiment Framework**
Built-in support for N=1 experiments across any domain. Define hypothesis, intervention, measurement, duration. Automatically track compliance and measure outcomes. Generate experiment reports with statistical analysis.

**OT-P1-3: Consolidated Notification Scheduler**
Central scheduler that collects notifications from all domains and consolidates them into minimal daily touchpoints. Prevents notification fatigue from 9+ domain scenarios all wanting attention.

**OT-P1-4: Medical Export**
Generate comprehensive health reports combining data from all domains, formatted for sharing with medical professionals. PDF export with charts, trends, and raw data appendix.

**OT-P1-5: Natural Language Queries**
Use Ollama to allow natural language questions against the event data. "How has my sleep been this month?" or "What changed when I started taking creatine?"

### P2 — Nice to Have

**OT-P2-1: Domain SDK / Shared Types Package**
Go package that domain scenarios import to get the event schema types, registration client, and helper functions. Reduces boilerplate when building new domain scenarios.

**OT-P2-2: Smart Recommendations**
AI-powered recommendations that use cross-domain data to suggest optimizations. "Based on your data, taking magnesium 2 hours before bed improves your sleep score by 15%."

**OT-P2-3: Hardware Integration Framework**
Abstraction layer for wearable and sensor data ingestion. Support for common protocols (Bluetooth LE, WiFi APIs) and device-specific adapters (Oura, Whoop, smart scales, CGMs).

**OT-P2-4: Historical Import**
Tools to import historical data from existing apps/services (MyFitnessPal exports, Apple Health data, etc.) to bootstrap the system with existing data.

## 3. Technical Direction

- **API:** Go (consistent with Vrooli patterns)
- **UI:** React + Vite + TypeScript
- **Database:** PostgreSQL (shared Vrooli resource), `lifestyle` schema with time-series event table
- **Cache:** Redis for real-time dashboard state and event aggregation caching
- **AI:** Ollama for insight generation and NL queries
- **Vector search:** Qdrant for semantic search across health notes and literature

### Key Constraints
- Single user, local only — no auth layer, no encryption at rest beyond disk-level
- Must work with other lifestyle domain scenarios that may or may not be running
- Dashboard must be useful even with only 1-2 domains active (graceful degradation)
- Event schema must be extensible without schema migrations for new domains

## 4. Dependencies

- PostgreSQL resource (required)
- Redis resource (required)
- Ollama resource (P1 — for NL queries and recommendations)
- Qdrant resource (P2 — for semantic search)
- At least one domain scenario to be useful (nootropics-tracker recommended as first)

## 5. UX & Branding

- Clean, data-dense dashboard aesthetic. Think Grafana meets Apple Health.
- Dark mode default (late-night data checking is common).
- Mobile-responsive — most daily interactions will be phone-based (checking morning brief, logging quick data).
- Minimal chrome, maximum information density.
- No gamification in the dashboard itself — individual domains can gamify if appropriate.
