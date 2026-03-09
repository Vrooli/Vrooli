# Requirements Context — Lifestyle Dashboard

This document synthesizes the requirements from `archive/requirements/` with updates reflecting clarified decisions and accepted suggestions. Use this as context for `prd-control-tower requirements generate`.

## Architecture Change: SQLite

**Important:** The original requirements referenced PostgreSQL. Per accepted suggestion s-mmjrnthj, all storage requirements should use **SQLite** instead:
- Replace `PostgreSQL time-series table` → `SQLite events table`
- Remove Redis caching references (not needed for single-user)
- JSON payloads work identically in SQLite with `json()` functions

## Requirements Registry Summary

### Foundation Requirements (P0)

| ID | Title | Updated Notes |
|----|-------|---------------|
| LD-FUNC-001 | Shared event schema & storage | SQLite `events` table. Add `is_intervention` and `hypothesis_id` columns per s3. |
| LD-FUNC-002 | Domain registration & discovery | No changes from archive spec. |
| LD-FUNC-003 | Cross-domain query API | No changes from archive spec. |
| LD-FUNC-004 | Unified dashboard UI | **Mobile-first** per clarification q2. Add Lifestyle Score display. |
| LD-FUNC-005 | Daily brief system | No changes from archive spec. |

### Intelligence Requirements (P1)

| ID | Title | Updated Notes |
|----|-------|---------------|
| LD-FUNC-006 | Correlation engine | Configurable threshold per clarification q4. |
| LD-FUNC-007 | Experiment framework | Leverages causality markers (is_intervention, hypothesis_id) from s3. |

### New Requirements (from accepted suggestions)

| ID | Title | Source | Priority |
|----|-------|--------|----------|
| LD-FUNC-008 | Composite Lifestyle Score | Suggestion s1 | P1 |
| LD-FUNC-009 | Weekly digest | Suggestion s2 | P1 |
| LD-FUNC-010 | Storage management UI | Clarification q3 | P0 |

## Updated Child Requirements

### Events Module (LD-EVENT-*)

```json
{
  "requirements": [
    {
      "id": "LD-EVENT-SCHEMA",
      "title": "Common event envelope schema",
      "description": "All events follow envelope: id (UUID), timestamp (ISO-8601), domain (string), event_type (string), payload (JSON text). Causality fields: is_intervention (INTEGER 0/1), hypothesis_id (TEXT UUID nullable).",
      "notes": "Updated for SQLite types and causality markers (s3)"
    },
    {
      "id": "LD-EVENT-STORAGE",
      "title": "SQLite events table",
      "description": "Events stored in lifestyle.db SQLite file. JSON payloads via TEXT columns. Extensible without schema migrations when adding new domains.",
      "notes": "Changed from PostgreSQL to SQLite (s-mmjrnthj)"
    },
    {
      "id": "LD-EVENT-INDEX",
      "title": "Event indexing strategy",
      "description": "Composite index on (domain, timestamp). Index on event_type. Partial index on hypothesis_id WHERE NOT NULL.",
      "notes": "SQLite indexing syntax"
    }
  ]
}
```

### Dashboard UI Module (LD-UI-*)

```json
{
  "requirements": [
    {
      "id": "LD-UI-TIMELINE",
      "title": "Unified timeline view",
      "description": "Main dashboard displays chronological timeline of events from all registered domains. Events color-coded by domain. Expandable event cards show payload details.",
      "notes": "No changes"
    },
    {
      "id": "LD-UI-DOMAINS",
      "title": "Per-domain sections",
      "description": "Dashboard includes collapsible sections for each registered domain showing domain-specific summary, recent events, and link to domain deep-dive UI.",
      "notes": "No changes"
    },
    {
      "id": "LD-UI-TRENDS",
      "title": "Trend charts",
      "description": "Dashboard includes configurable trend charts with 7d/30d/90d time windows. Line charts for continuous metrics, bar charts for discrete events. Domain comparison overlays.",
      "notes": "No changes"
    },
    {
      "id": "LD-UI-RESPONSIVE",
      "title": "Mobile-first responsive design",
      "description": "Dashboard designed mobile-first for phone-based daily brief consumption. Touch-friendly targets. Simplified mobile view prioritizes daily brief and Lifestyle Score. Full features on tablet/desktop.",
      "notes": "Updated from 'mobile-responsive' to 'mobile-first' per q2"
    },
    {
      "id": "LD-UI-SCORE",
      "title": "Lifestyle Score display",
      "description": "Dashboard prominently displays daily 0-100 Lifestyle Score. Trend line showing score history. Tap to see domain breakdown.",
      "notes": "New requirement from s1"
    },
    {
      "id": "LD-UI-STORAGE",
      "title": "Storage management settings",
      "description": "Settings page shows database size, large files breakdown. Actions: clean all data, clean selected domains, export data backup, import backup.",
      "notes": "New requirement from q3 clarification"
    }
  ]
}
```

### Intelligence Module (LD-CORR-*, LD-EXP-*)

```json
{
  "requirements": [
    {
      "id": "LD-CORR-THRESHOLD",
      "title": "Configurable minimum threshold",
      "description": "User-configurable minimum sample size before generating correlations. Default: 14 data points. UI shows 'needs more data' for insufficient samples. Setting accessible in dashboard settings.",
      "notes": "Default 14, configurable per q4"
    },
    {
      "id": "LD-SCORE-CALC",
      "title": "Lifestyle Score calculation",
      "description": "Daily score 0-100 computed from all active domains. Domain weights: sleep (high), exercise (high), diet (medium), nootropics (medium), socialization (low). Weights adjustable per user. Score calculated at midnight for previous day.",
      "notes": "New requirement from s1"
    },
    {
      "id": "LD-DIGEST-WEEKLY",
      "title": "Weekly digest generation",
      "description": "Every Sunday 6pm, generate 'What Changed?' digest. Compare current week to rolling 4-week baseline. Show: domain-level deltas, new correlations discovered, Lifestyle Score trend. Delivery: in-app notification + email (if configured).",
      "notes": "New requirement from s2"
    }
  ]
}
```

## Validation Strategy

All requirements link to validation methods:

- **test**: Go unit tests in `api/*_test.go`
- **integration**: API endpoint tests with SQLite test database
- **e2e**: End-to-end workflows via BAS (if UI automation available)

Test tags like `[REQ:LD-EVENT-SCHEMA]` automatically update requirement status when tests run.

## Technical Constraints

### SQLite-Specific Patterns

```sql
-- JSON queries in SQLite
SELECT * FROM events
WHERE json_extract(payload, '$.substance') = 'caffeine';

-- Date handling (ISO-8601 strings)
SELECT * FROM events
WHERE timestamp >= datetime('now', '-7 days');

-- Single-writer constraint
-- All writes go through API, never direct from UI
```

### Mobile-First UI Constraints

- Minimum touch target: 44x44px
- Primary actions in thumb-reach zone
- Daily brief fits in single screen scroll
- Offline-first for brief viewing (P1)

## Requirement Coverage

| Category | Parent IDs | Child Count | Status |
|----------|------------|-------------|--------|
| Foundation | LD-FUNC-001 to 005 | 16 | not_started |
| Intelligence | LD-FUNC-006 to 007 | 6 | not_started |
| New (from enhance) | LD-FUNC-008 to 010 | 5 | not_started |
| **Total** | 10 parents | 27 children | |
