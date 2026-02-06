# Product Requirements Document (PRD)
> Version 2.0 — Operational Targets aligned with the scenario-generator standard. Detailed implementation guidance now lives in `requirements/`.

## 🎯 Overview
- Gamified household chore tracking system that transforms mundane cleaning tasks into an engaging experience with points, achievements, and playful UX
- Serves families, households, and individuals who want to build healthy cleaning habits and make task completion more rewarding
- Shipping surfaces include the Go API, React/Vite dashboard, and CLI wrapper for flexible interaction modes (quick mobile check-ins, family leaderboards, automated scheduling)
- Core value: eliminate chore conflicts, build consistent habits through gamification, and make household management fun rather than friction-heavy

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Core Chore Management | Support CRUD operations for chores with configurable difficulty levels, point values, frequency patterns, and category tags; expose both API and CLI interfaces
- [ ] OT-P0-002 | Family Profile System | Enable multiple user profiles with age-appropriate task assignments, individual statistics tracking, and customizable reward preferences
- [ ] OT-P0-003 | Gamification Engine | Implement points system, achievement unlocks, streak tracking, and family leaderboards, powered by token-economy scenario
- [ ] OT-P0-004 | Task Completion Flow | Deliver satisfying completion interactions including confetti animations, sound effects, instant point updates, and achievement notifications

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | AI-Powered Scheduling | Integrate with Ollama to generate intelligent weekly chore schedules based on completion patterns, user preferences, and household rhythms
- [ ] OT-P1-002 | Habit Analytics Dashboard | Surface completion rates, streak visualizations, trend graphs, and behavioral insights to encourage consistency
- [ ] OT-P1-003 | Custom Rewards Marketplace | Allow families to define purchasable rewards using earned points (screen time, special treats, choosing dinner). Powered by using token-economy scenario's API to do fungible and non-fungible token swaps
- [ ] OT-P1-004 | Reminder & Notification System | Provide configurable reminders via UI notifications and optional integration hooks for external notification scenarios

### 🟢 P2 – Future / expansion ideas
- [ ] OT-P2-001 | Seasonal Challenges | Introduce limited-time events and special achievement campaigns (spring cleaning bonuses, holiday decoration challenges)
- [ ] OT-P2-002 | Social Features | Enable sharing achievements with friends, friendly competition between households, and collaborative family goals
- [ ] OT-P2-003 | Smart Home Integration | Connect with IoT devices to auto-detect task completion (vacuum robot finished, dishwasher cycle complete)

## 🧱 Tech Direction Snapshot
- React + TypeScript + Vite UI, Go API, PostgreSQL persistence, Redis for real-time leaderboards, and Ollama integration for AI scheduling
- Lifecycle-managed ports, pnpm workspaces, and shared packages (`@vrooli/api-base`, `@vrooli/iframe-bridge`) keep the scenario aligned with react-vite template
- CLI should proxy API endpoints for quick task operations; mobile-first UI design prioritizes rapid check-ins and status views
- Requirements, tests, and docs must cite `[REQ:ID]` for traceability so requirement coverage can be tracked programmatically

## 🤝 Dependencies & Launch Plan
- Hard dependencies: PostgreSQL (user/chore data), Redis (leaderboards/real-time stats); Ollama optional for AI scheduling features
- Setup order: install CLI → build Go API → bootstrap PostgreSQL schema/seed → install UI deps → build UI bundle; lifecycle commands enforce this order
- Launch readiness requires phased tests, health checks for API/UI, and `requirements/index.json` kept in sync with PRD operational targets
- Operational risks center on gamification balance (too easy/hard point systems) and family adoption; mitigate via configurable difficulty curves and onboarding flows

## 🎨 UX & Branding
- Playful, colorful aesthetic inspired by Animal Crossing meets productivity tools; bright pastels with vibrant accent pops for achievements
- Animated confetti on task completion, cute character mascots, satisfying sound effects that reinforce positive behavior
- Mobile-first responsive design with quick-tap completion buttons, swipe gestures, and thumb-friendly navigation zones
- WCAG AA contrast compliance with colorblind-friendly status indicators (icons + text labels, not color-only signals)
- Encouraging, supportive voice in all UI copy; celebrate progress without shame or pressure
