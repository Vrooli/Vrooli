# Documentation Outlines — Lifestyle Dashboard

Outlines for scenario documentation to be created during processing.

## README.md Structure

```markdown
# Lifestyle Dashboard

> Unified personal lifestyle intelligence dashboard — the foundation for cross-domain health insights.

## Overview

- What it does: Foundation scenario for 9 domain-specific health scenarios
- Philosophy: Automated by default, customizable on demand
- Target: Personal use on Vrooli server

## Quick Start

- Prerequisites (Go, Node.js, SQLite)
- Installation
- Starting the dashboard
- Connecting first domain (nootropics-tracker)

## Features

### P0 (Core)
- Unified event storage
- Domain registration
- Cross-domain queries
- Dashboard UI (mobile-first)
- Daily briefs (morning/evening)
- Storage management

### P1 (Intelligence)
- Correlation engine
- Lifestyle Score
- Weekly digest
- Experiment framework
- Natural language queries

## Architecture

### Event Schema
- Common envelope fields
- Causality markers (is_intervention, hypothesis_id)
- SQLite storage rationale

### Domain Integration Contract
- Registration API
- Event emission
- Health checks
- Brief contributions

## Configuration

- Brief timing (morning/evening)
- Correlation threshold
- Lifestyle Score weights
- Storage location

## API Reference

- Link to OpenAPI spec or handler docs

## Related Scenarios

- nootropics-tracker (Phase 1)
- sleep-tracker (Phase 1, blocked on wearable)
- diet-nutrition (Phase 2)
- etc.
```

## RESEARCH.md Sections

```markdown
# Research Notes — Lifestyle Dashboard

## Storage Decision: SQLite vs PostgreSQL

### Decision
SQLite-only storage (accepted suggestion s-mmjrnthj)

### Rationale
- Personal single-user app — performance not a concern
- Portability for future mobile/Electron deployment
- Single file backup/restore
- No external database server management
- Follows storage-steer skill patterns

### Trade-offs
- Single-writer constraint (all writes via API)
- No built-in replication (acceptable for personal use)
- Migration path to PostgreSQL if ever needed for scale

## Correlation Engine Research

### Statistical Methods
- Pearson correlation for continuous variables
- Chi-squared for categorical
- P-value threshold: 0.05
- Minimum sample size: 14 (configurable)

### Prior Art
- Quantified Self community patterns
- N=1 experiment design from personal science literature

## Domain Integration Patterns

### Event-Driven Architecture
- Loose coupling via event schema
- Domains self-register on startup
- Dashboard discovers capabilities dynamically

### Similar Systems
- Apple Health HealthKit model
- Google Fit aggregation patterns
- Oura Ring cross-metric correlations
```

## PROBLEMS.md Entries

```markdown
# Known Issues and Limitations

## Current Limitations

### PROB-001: No Wearable Integration Yet
- **Status:** Deferred to P2
- **Impact:** Sleep tracker and activity data require manual entry
- **Workaround:** Design with dependency injection for future device support
- **Resolution:** Implement OT-P2-003 Hardware Integration Framework

### PROB-002: Single-Writer SQLite Constraint
- **Status:** By design
- **Impact:** All writes must go through API; no direct UI→DB writes
- **Workaround:** None needed — this is the intended architecture
- **Notes:** Enables future WAL mode and concurrent readers

### PROB-003: No Offline Support in P0
- **Status:** Deferred
- **Impact:** Dashboard requires network to fetch data
- **Workaround:** Morning/evening briefs designed to be screenshots/cached
- **Resolution:** Add service worker and local storage sync in P1

## Deferred Decisions

### Mobile App Type
- **Options:** PWA, React Native, Capacitor
- **Decision needed by:** P2 (when mobile becomes priority)
- **Current direction:** PWA for P0/P1, evaluate native for P2

### Wearable Device Selection
- **Options:** Oura Ring, Whoop, Apple Watch, Garmin
- **Decision needed by:** Before Sleep Tracker implementation
- **Current direction:** Oura Ring (sleep-focused data quality)
```

## PROGRESS.md Initial Entry

```markdown
# Development Progress

## Phase 1: Foundation + First Domain

### Milestone: Dashboard Core (P0)
- [ ] SQLite schema initialization
- [ ] Domain registration API
- [ ] Event CRUD API
- [ ] Cross-domain query API
- [ ] Dashboard UI shell
- [ ] Timeline view
- [ ] Per-domain sections
- [ ] Morning/evening brief generator
- [ ] Storage management settings

### Milestone: Nootropics Integration
- [ ] Nootropics Tracker scenario created
- [ ] Registration with dashboard
- [ ] Event emission working
- [ ] Cross-domain queries functional
- [ ] Correlation visible in UI

### Milestone: Intelligence (P1)
- [ ] Correlation engine
- [ ] Lifestyle Score calculation
- [ ] Weekly digest
- [ ] Experiment framework

## Status
- Current phase: Not started
- Last updated: [date]
```
