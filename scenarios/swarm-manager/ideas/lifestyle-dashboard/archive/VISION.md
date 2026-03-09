# Lifestyle Dashboard — Vision Document

## The Big Picture

This is the master plan for a category of Vrooli scenarios that collectively form a **personal lifestyle intelligence system**. The lifestyle dashboard is the unifying scenario that ties everything together, providing cross-domain analytics, a shared data model, and the automation backbone.

### Core Philosophy

**Automated by default, customizable on demand.**

The target user (Matt) is self-admittedly lazy about lifestyle optimization despite knowing it matters. These scenarios must succeed under that constraint:

- Everything runs hands-off. No daily logging obligations, no engagement guilt.
- The system tells you what to do, ships you what you need, and only asks for input when a genuine decision is required.
- Each domain can be deep-dived and customized for people who enjoy that (think Notion's appeal), but the default path requires zero effort.
- Over time, manual steps (skincare routine, meal/nootropics prep, exercise form checking) can be delegated to humanoid robots — the system is designed with that future in mind.

### Why Dashboard First

The cross-domain correlations are the moat. Sleep quality affecting exercise recovery, diet impacting skin, nootropics interacting with sleep — no single-domain app captures these. Building the shared data layer first ensures:

1. Every domain scenario has a clear integration contract from day one
2. Cross-domain insights are architecturally possible, not retrofitted
3. The event model is designed for correlation, not just per-domain storage

## Domain Scenarios

Each domain is its own scenario that plugs into the dashboard. They should be built independently but conform to the dashboard's integration contract.

### 1. Nootropics Tracker
**What it does:** Manages structured nootropic protocols, tracks what you take and when, runs N=1 experiments with measurable outcomes.
**Current state:** Matt takes caffeine pills on weekdays and uses small amounts of weed for creativity. No structured plan.
**MVP automation:** Morning notification of what to take. Log intake with one tap. Correlate with sleep/focus data automatically.
**Key insight:** Adding compounds to an existing caffeine habit has near-zero friction. This is the lowest-friction entry point.

### 2. Sleep Tracker
**What it does:** Passively tracks sleep via wearable integration, correlates with all other domains.
**Current state:** No wearable. No tracking.
**MVP automation:** Fully passive once a wearable is acquired. Just wear it and sleep.
**Dependency:** Requires a wearable device (e.g., Oura Ring, Whoop, Apple Watch). This is the main blocker.
**Key insight:** Sleep is the highest-signal passive data source. Nearly every other domain benefits from sleep correlation.

### 3. Skincare Manager
**What it does:** Manages skincare routines, runs product experiments, tracks skin condition over time.
**Current state:** Matt does basic skincare but knows it's not enough. Too lazy to research and optimize.
**MVP automation:** Prescribe a routine based on skin profile. Ship products. Daily checklist notification. Track compliance. Experiment with one variable at a time.
**Key insight:** End state is "be shipped products and told exactly what to do." The scenario should progressively reveal complexity only if the user wants it.

### 4. Diet & Nutrition Planner
**What it does:** Meal planning, micronutrient optimization, grocery lists, recipe management.
**Current state:** Matt eats well but planned micronutrients once and just sticks to the same foods.
**MVP automation:** Analyze current diet once, identify gaps, suggest minimal changes. Generate weekly grocery lists. Don't try to overhaul everything — optimize at the margins.
**Key insight:** The "I planned once and stick to it" pattern is actually good — the scenario should respect inertia and suggest small, high-impact tweaks rather than complete diet overhauls.
**Archived references:** `nutrition-tracker-archived`, `recipe-book-archived`, `make-it-vegan-archived` in swarm-manager ideas. These have useful data modeling ideas but were designed for consumer SaaS — strip the multi-tenant/auth overhead.

### 5. Exercise & Activity Planner
**What it does:** Structured exercise programming, form tracking, activity monitoring, flexibility/mobility routines.
**Current state:** Matt exercises but with no structured plan.
**MVP automation:** Generate a structured plan based on goals and available equipment. Daily "here's your workout" notification. Track completion. Periodize automatically.
**Key insight:** The plan should be opinionated and just tell you what to do. Progressive overload logic can be automated. Form checking is a future robot-delegation opportunity.

### 6. Meditation & Focus
**What it does:** Guided meditation/mindfulness sessions, focus tracking, attention training.
**Current state:** Not currently practiced.
**MVP automation:** Schedule short sessions. Guided audio/prompts. Track completion. Correlate with productivity/sleep.
**Archived references:** `morning-vision-walk-archived` has relevant concepts around daily mindfulness.

### 7. Socialization & Mental Health
**What it does:** Relationship maintenance reminders, social activity tracking, mood logging, mental health check-ins.
**Current state:** Not tracked.
**MVP automation:** Periodic check-ins. Relationship maintenance reminders (when did you last talk to X?). Mood trends.
**Archived references:** `personal-relationship-manager-archived`, `date-night-planner-archived` have relevant data models.

### 8. Learning & Brain Games
**What it does:** Spaced repetition, brain training, skill tracking, learning path management.
**Current state:** Not structured.
**MVP automation:** Daily brain training prompt. Spaced repetition for things you want to remember. Track cognitive performance over time.
**Archived references:** `study-buddy-archived` (40% functional), `mind-maps-archived` (10/10 reusability score).

### 9. Biomarkers & Preventive Care
**What it does:** Track blood work, health metrics, preventive screening schedules, trend analysis, medical professional export.
**Current state:** Not tracked.
**MVP automation:** Reminder schedule for blood work. Input results. Trend over time. Flag anomalies. Export reports for doctor visits.
**Key insight:** This is the most medically valuable domain. The ability to export a comprehensive health report (combining data from all domains) to a doctor is a unique differentiator.

## Dashboard Responsibilities

The lifestyle dashboard itself is NOT just a viewer — it's the intelligence and integration layer:

### Data Model & Event Bus
- Define a shared event schema that all domain scenarios emit to (e.g., `lifestyle.intake.logged`, `lifestyle.sleep.recorded`, `lifestyle.exercise.completed`)
- Store events in a time-series-friendly format in PostgreSQL
- Enable cross-domain queries (e.g., "show me sleep quality on days I took magnesium vs. days I didn't")

### Correlation Engine
- Automated correlation analysis across domains
- Surface insights like "your sleep quality drops 20% on days you skip exercise"
- Statistical significance tracking — don't surface noise as insight

### Automation Orchestrator
- Central scheduler for all domain notifications/reminders
- Consolidate into minimal daily touchpoints (don't send 9 separate notifications)
- "Morning brief" and "evening review" as the primary interaction patterns

### Analytics & Visualization
- Unified timeline view across all domains
- Per-domain deep-dive dashboards
- Trend analysis with configurable time windows
- Exportable health reports for medical professionals

### Integration Contract
- Define the API/event interface that domain scenarios must implement
- Provide a Go SDK or shared types package for domain scenarios to import
- Handle registration/discovery of domain scenarios

## Prioritization for Building

**Phase 1 — Foundation:**
1. Lifestyle Dashboard (shared data model, event bus, basic UI shell)
2. Nootropics Tracker (lowest friction, immediate value, tight feedback loop)
3. Sleep Tracker (highest passive signal, but blocked on wearable purchase)

**Phase 2 — High Impact:**
4. Diet & Nutrition (optimize existing habits)
5. Exercise Planner (structure existing activity)

**Phase 3 — Expansion:**
6. Skincare Manager
7. Biomarkers & Preventive Care
8. Meditation & Focus
9. Learning & Brain Games
10. Socialization & Mental Health

## Technical Architecture

### Shared Infrastructure
- **Database:** PostgreSQL (shared Vrooli resource) with a `lifestyle` schema
- **Event storage:** Time-series events table with JSONB payloads, indexed by domain + timestamp
- **Cache:** Redis for real-time dashboard state
- **AI:** Ollama for insight generation, experiment design, and natural language queries
- **Vector search:** Qdrant for semantic search across health literature and personal notes

### Per-Domain Pattern
Each domain scenario follows the same structure:
- Go API that emits events to the shared `lifestyle` schema
- Implements a registration endpoint the dashboard discovers
- Exposes domain-specific endpoints for its own UI
- Can run standalone but gains cross-domain value when dashboard is present

### Key Design Decisions
- **Personal use only:** No auth, no multi-tenant, no GDPR compliance overhead. Just works for one user on one server.
- **Local-first:** All data stays on the local Vrooli server. No cloud sync. No third-party analytics.
- **Experiment framework:** Built into the data model from day one. Every domain can run A/B-style N=1 experiments (e.g., "try this nootropic stack for 2 weeks, measure sleep quality delta").

## Existing Archived Scenarios

These archived backlog items contain potentially useful data models and research. They should be referenced but not directly migrated — their consumer SaaS framing doesn't match the personal-use philosophy:

| Archived Item | Useful For | Caveat |
|---|---|---|
| `nutrition-tracker-archived` | Diet data model, macro schema | Over-engineered for multi-user |
| `recipe-book-archived` | Recipe storage, semantic search | Too feature-rich for MVP |
| `make-it-vegan-archived` | Ingredient analysis patterns | Narrow scope |
| `morning-vision-walk-archived` | Mindfulness interaction model | Voice-first may not be needed |
| `study-buddy-archived` | Spaced repetition algorithm | 40% functional, worth reviewing |
| `mind-maps-archived` | Knowledge graph patterns | High reusability but scope creep risk |
| `personal-relationship-manager-archived` | Relationship data model | Good reminders pattern |
| `date-night-planner-archived` | Activity planning | Narrow scope |
| `chore-tracking-archived` | Habit/routine tracking | Gamification patterns useful |

## Future Vision

- **Robot integration:** As humanoid robots become available, many manual lifestyle tasks (skincare application, meal prep, form checking, vitals measurement) can be delegated. The system's automation-first design makes this a natural extension.
- **Monetization:** Individual domain scenarios can be cleaned up and offered as SaaS products if they prove valuable. The personal-use-first approach means they'll be battle-tested before any commercialization.
- **Hardware integration:** Wearables, smart scales, blood glucose monitors, etc. can feed data directly into the event bus as they're acquired.
