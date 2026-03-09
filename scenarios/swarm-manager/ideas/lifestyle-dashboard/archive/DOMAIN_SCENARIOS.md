# Lifestyle Domain Scenarios — Reference

This document describes the domain scenarios that integrate with the lifestyle dashboard. Each will be its own swarm-manager backlog item when ready to build, referencing this document and the dashboard's integration contract.

## Build Order & Rationale

| Phase | Scenario | Rationale |
|-------|----------|-----------|
| 1 | Nootropics Tracker | Lowest friction (adds to existing caffeine habit), immediate feedback loop, tight correlation with sleep |
| 1 | Sleep Tracker | Highest passive signal, benefits every other domain. Blocked on wearable purchase. |
| 2 | Diet & Nutrition | Optimizes existing good habits with minimal changes |
| 2 | Exercise Planner | Structures existing activity |
| 3 | Skincare Manager | Requires more behavior change, benefits from biomarker data |
| 3 | Biomarkers & Preventive Care | High medical value, works best with several domains feeding data |
| 3 | Meditation & Focus | Lower urgency, benefits from sleep/nootropics data |
| 3 | Learning & Brain Games | Lower urgency |
| 3 | Socialization & Mental Health | Most subjective, hardest to automate |

## Per-Domain Specifications

### Nootropics Tracker

**Events emitted:**
- `nootropics.intake.logged` — substance, dose, timestamp
- `nootropics.protocol.started` — protocol name, substances, schedule
- `nootropics.protocol.completed` — protocol name, duration, compliance rate
- `nootropics.experiment.result` — experiment ID, outcome metrics

**Current baseline:**
- Caffeine pills on weekdays (timing and dose untracked)
- Small amounts of cannabis for creativity sessions (frequency untracked)

**MVP scope:**
- Substance catalog (start with caffeine, cannabis, then expand)
- Daily protocol: what to take, when, with what
- One-tap intake logging via notification action
- Correlation dashboards (substance × sleep, substance × focus)
- Simple experiment runner: "try X for 2 weeks, compare to baseline"

**Integration contract:**
- Emits events to `lifestyle.events` table
- Registers with dashboard via `/api/v1/domains/register`
- Exposes `/api/v1/health` for dashboard health checks

---

### Sleep Tracker

**Events emitted:**
- `sleep.night.recorded` — duration, quality score, stages, HRV, timestamps
- `sleep.nap.recorded` — duration, timestamp

**Dependency:** Wearable device (Oura Ring recommended for sleep-specific data quality)

**MVP scope:**
- Wearable sync (device-specific adapter)
- Nightly sleep score display
- Trend charts (7d, 30d, 90d)
- Automatic correlation with other domain events

---

### Diet & Nutrition Planner

**Events emitted:**
- `nutrition.meal.logged` — meal type, foods, macros, micros
- `nutrition.plan.generated` — weekly plan, target macros
- `nutrition.grocery.generated` — shopping list

**MVP scope:**
- One-time diet analysis (log current typical meals)
- Gap identification (missing micronutrients)
- Minimal-change suggestions (keep current base, tweak 1-2 things)
- Weekly grocery list generation
- NOT a full meal planner — respect the "I eat the same things" pattern

---

### Exercise & Activity Planner

**Events emitted:**
- `exercise.workout.completed` — type, duration, exercises, sets/reps/weight
- `exercise.plan.generated` — weekly plan, periodization phase
- `exercise.activity.recorded` — steps, active minutes (from wearable)

**MVP scope:**
- Goal-based program generation (strength, cardio, flexibility)
- Daily "here's your workout" with specific exercises
- Completion tracking (did you do it: yes/no/modified)
- Automatic periodization (progressive overload, deload weeks)

---

### Skincare Manager

**Events emitted:**
- `skincare.routine.completed` — products used, AM/PM
- `skincare.condition.logged` — skin state, photos (optional)
- `skincare.experiment.result` — product trial outcome

**MVP scope:**
- Skin profile questionnaire (one-time)
- Prescribed AM/PM routine
- Product recommendations with purchase links
- Compliance tracking via daily checklist
- Single-variable experiments (change one product, measure impact)

---

### Biomarkers & Preventive Care

**Events emitted:**
- `biomarkers.bloodwork.recorded` — test name, values, reference ranges
- `biomarkers.metric.recorded` — weight, blood pressure, etc.
- `biomarkers.screening.scheduled` — type, date

**MVP scope:**
- Screening schedule (blood work every 6 months, etc.)
- Manual blood work entry (from lab results)
- Trend visualization with reference ranges
- Anomaly flagging
- Comprehensive health report export (PDF) combining all domain data

---

### Meditation & Focus

**Events emitted:**
- `focus.session.completed` — type, duration, quality rating
- `focus.streak.updated` — current streak length

**MVP scope:**
- Short guided sessions (5-10 min)
- Daily prompt/reminder
- Streak tracking
- Correlation with sleep and nootropics

---

### Learning & Brain Games

**Events emitted:**
- `learning.session.completed` — type, duration, score
- `learning.review.due` — spaced repetition items due

**MVP scope:**
- Daily brain training prompt (simple cognitive exercises)
- Spaced repetition for custom items
- Performance trending

---

### Socialization & Mental Health

**Events emitted:**
- `social.interaction.logged` — person, type, quality rating
- `social.mood.logged` — mood score, notes
- `social.reminder.triggered` — relationship maintenance prompt

**MVP scope:**
- Contact list with last-interaction dates
- "You haven't talked to X in Y days" reminders
- Periodic mood check-ins
- Mood trend correlation with other domains
