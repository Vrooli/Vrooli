# Experience Architecture Audit - 2026-03-11 (Updated)

## Scenario Purpose

**Purpose Statement**: This scenario helps Matt (single user) track personal lifestyle data across multiple health domains (sleep, exercise, diet, nootropics, etc.) so that he can gain cross-domain insights, identify correlations, and optimize his overall health through data-driven decisions.

**Value Proposition**: Cross-domain health insights that no single-domain app can provide, with zero-effort automation as the default interaction mode.

---

## Core Personas & Key Jobs

### 1. Daily User (Primary)

The single user checking in on their health status throughout the day.

| Job | Entry Point | Expected Flow | Success |
|-----|-------------|--------------|---------|
| Check today's health status | Dashboard | Dashboard → Score + Stats + Brief Preview visible | See lifestyle score, trend, recent events, brief summary |
| Review morning/evening brief | Dashboard brief preview OR Briefs nav | Briefs → Current/Morning/Evening | See consolidated summary across domains |
| Drill into specific domain | Dashboard domain card | Dashboard → Domain Detail | See domain health, recent events, capabilities |

### 2. Weekly Reviewer

The user reviewing their week's progress and making adjustments.

| Job | Entry Point | Expected Flow | Success |
|-----|-------------|--------------|---------|
| Review week's progress | Briefs → Weekly tab | Weekly Digest view | See week-over-week comparison, trends |
| Adjust score weights | Settings | Settings → Score Config | Configure domain importance |
| Clean up old data | Settings | Settings → Storage → Clear | Free up space, remove old events |

### 3. Event Inspector

The user investigating specific events or patterns.

| Job | Entry Point | Expected Flow | Success |
|-----|-------------|--------------|---------|
| Browse all events | Events nav | Events → Filter by domain + time | See chronological event list with filters |
| Investigate domain events | Domain Detail | Domain Detail → Events section | See domain-specific events |

---

## Current vs. Ideal Flows

### Flow 1: Daily Check-In ✅ RESOLVED

**Current Flow** (after Phase 18 iter 2):
1. User opens Dashboard → Sees lifestyle score, stats, timeline chart, AND brief preview
2. Brief preview shows current brief summary with direct link to full view
3. One-click access to briefs page from dashboard

**Status**: RESOLVED - Brief preview added to Dashboard

### Flow 2: Weekly Review ✅ RESOLVED

**Current Flow** (after Phase 18 iter 1):
1. User navigates to Briefs → Weekly tab
2. WeeklyDigestCard shows score trends, domain changes, highlights, focus areas
3. Complete "What Changed?" summary with actionable insights

**Status**: RESOLVED - Weekly Digest UI implemented

### Flow 3: Score Configuration ✅ RESOLVED

**Current Flow** (after Phase 18 iter 1):
1. User navigates to Settings
2. Score Configuration section shows all domain weights
3. User can adjust each domain (high/medium/low/none)
4. Changes persist and affect score calculation

**Status**: RESOLVED - Score Configuration UI implemented

---

## Navigation Audit

### Back Button Behavior ✅ RESOLVED

| Page | Back Target | Status |
|------|-------------|--------|
| DomainsPage | "/" (Dashboard) | OK |
| EventsPage | "/" (Dashboard) | OK |
| SettingsPage | "/" (Dashboard) | OK |
| DomainDetailPage | "/domains" | OK |
| BriefsPage | "/" (Dashboard) | ✅ FIXED - Added back button in Phase 18 iter 1 |

All pages now have consistent back navigation.

### Label-Destination Truthfulness

| Control | Label | Destination | Truthful? |
|---------|-------|-------------|-----------|
| Logo "Lifestyle Dashboard" | Logo | "/" Dashboard | Yes |
| Nav "Dashboard" | Dashboard | "/" | Yes |
| Nav "Domains" | Domains | "/domains" | Yes |
| Nav "Events" | Events | "/events" | Yes |
| Nav "Briefs" | Briefs | "/briefs" | Yes |
| Nav "Settings" | Settings | "/settings" | Yes |
| Domain card click | Interactive card | "/domains/{name}" | Yes |
| Brief preview | Interactive card | "/briefs" | Yes |

All navigation controls are truthful.

### Missing Entry Points ✅ ALL RESOLVED

1. ~~**Weekly Digest**: No UI entry point~~ → ✅ Added Weekly tab in BriefsPage
2. ~~**Score Configuration**: No UI entry point~~ → ✅ Added section in SettingsPage

---

## Friction Analysis

### Mechanical Friction ✅ RESOLVED

| Location | Issue | Severity | Status |
|----------|-------|----------|--------|
| Events filtering | ~~No time-range filter~~ | Medium | ✅ FIXED - Added time-range filter with presets + custom |
| Domain detail | No quick link back to all domains events | Low | Acceptable - can use nav |

### Cognitive Friction ✅ RESOLVED

| Location | Issue | Severity | Status |
|----------|-------|----------|--------|
| Dashboard | ~~No brief preview~~ | Medium | ✅ FIXED - BriefPreview component added |
| Settings | ~~No score config~~ | Medium | ✅ FIXED - Score Config section added |

### Discoverability Friction ✅ ALL RESOLVED

| Feature | Issue | Severity | Status |
|---------|-------|----------|--------|
| Weekly Digest | ~~API exists, NO UI~~ | High | ✅ FIXED - Weekly tab in BriefsPage |
| Score Weights | ~~API exists, NO UI~~ | High | ✅ FIXED - Score Config in SettingsPage |

---

## Implementation Status

### Completed (Phase 18)

1. ✅ **Weekly Digest UI**: Added "Weekly" tab to BriefsPage with WeeklyDigestCard component
2. ✅ **Score Configuration UI**: Added Score Configuration section to SettingsPage
3. ✅ **Back button to BriefsPage**: Added for navigation consistency
4. ✅ **Brief preview on Dashboard**: Added BriefPreview component to DashboardPage
5. ✅ **Time-range filter for Events**: Added comprehensive filtering (all/today/week/month/custom)

### Remaining (Future Phases)

1. **Quick actions from Dashboard**: Consider one-click actions (e.g., "Add event", "Start experiment")
2. **Event detail view**: Consider dedicated event detail page for deep inspection
3. **Domain comparison view**: Side-by-side domain comparison for correlation analysis

---

## Documentation History

| Date | Changes |
|------|---------|
| 2026-03-11 | Initial audit. Identified missing Weekly Digest and Score Config UI as critical gaps. |
| 2026-03-11 | Phase 18 iter 1: Resolved Weekly Digest UI, Score Config UI, BriefsPage back button. |
| 2026-03-11 | Phase 18 iter 2: Resolved Brief preview on Dashboard, Time-range filter for Events. All critical gaps addressed. |
