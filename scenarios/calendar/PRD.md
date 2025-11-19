# Calendar - Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Production-Ready
> **Template**: Canonical PRD v2.0.0

## 🎯 Overview

**Purpose**: Universal scheduling intelligence that enables any scenario to create, manage, and coordinate time-based events, recurring schedules, and temporal workflows. This becomes Vrooli's temporal coordination brain, transforming scenarios from isolated tools into time-aware, interconnected systems.

**Primary Users**: Business professionals, project managers, executive assistants, automation scenarios requiring time-based orchestration.

**Deployment Surfaces**:
- REST API (28+ endpoints)
- CLI commands (event create/list, schedule chat/optimize)
- Professional web UI (calendar views, drag-drop scheduling)
- Event-driven automation hooks
- Natural language scheduling interface

**Intelligence Amplification**: Agents can now reason about time, schedule cascading workflows, coordinate resources across scenarios, and create sophisticated automation that respects temporal constraints. This enables building enterprise-grade scheduling applications, automating recurring business processes, and creating intelligent time-management systems that improve through usage patterns.

**Recursive Value**: Enables new scenarios like enterprise project managers with milestone dependencies, automated customer onboarding sequences, resource booking systems, smart home orchestration, and business intelligence schedulers.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Multi-user event creation | Multi-user event creation with authentication via scenario-authenticator
- [x] OT-P0-002 | Recurring event patterns | Recurring event patterns (daily, weekly, monthly, custom)
- [x] OT-P0-003 | Event reminders | Event reminders via notification-hub integration
- [x] OT-P0-004 | Natural language scheduling | Natural language scheduling through chat interface with Ollama
- [x] OT-P0-005 | Event-triggered automation | Event-triggered code execution for scenario automation
- [x] OT-P0-006 | PostgreSQL storage | PostgreSQL storage with full CRUD operations
- [x] OT-P0-007 | AI-powered search | AI-powered schedule search using Qdrant embeddings
- [x] OT-P0-008 | REST API access | REST API for programmatic access by other scenarios
- [x] OT-P0-009 | Professional UI | Professional calendar UI with week/month/agenda views

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Schedule optimization | Schedule optimization suggestions with AI
- [x] OT-P1-002 | Conflict detection | Smart conflict detection and resolution
- [x] OT-P1-003 | Timezone handling | Timezone handling for distributed teams
- [x] OT-P1-004 | Event categorization | Event categorization and filtering
- [x] OT-P1-005 | Bulk operations | Bulk operations and batch scheduling
- [x] OT-P1-006 | iCal import/export | iCal import/export for external calendar sync
- [x] OT-P1-007 | Event templates | Event templates for common meeting types
- [x] OT-P1-008 | RSVP functionality | Attendance tracking and RSVP functionality

### 🟢 P2 – Future / expansion

- [x] OT-P2-001 | External calendar sync | External calendar synchronization (Google Calendar, Outlook)
- [x] OT-P2-002 | Meeting preparation | Meeting preparation automation (agenda creation, document gathering)
- [x] OT-P2-003 | Travel time calculation | Travel time calculation and buffer insertion
- [x] OT-P2-004 | Resource booking | Resource double-booking prevention
- [x] OT-P2-005 | Advanced analytics | Advanced analytics on scheduling patterns and productivity
- [ ] OT-P2-006 | Voice scheduling | Voice-activated scheduling through audio scenarios

## 🧱 Tech Direction Snapshot

**Architecture**: Hybrid storage with PostgreSQL for structured event data and Qdrant for semantic search vectors. Go API server with React UI, JWT authentication, and event-driven automation.

**UI/API Stack**:
- API: Go with database/sql, Qdrant client, Ollama integration
- UI: React + Vite, responsive calendar views, drag-drop scheduling
- CLI: Go binary with natural language support

**Data Storage**:
- PostgreSQL: Events, users, recurring patterns, reminders, templates, resources, attendees
- Qdrant: Event embeddings for AI-powered semantic search
- In-memory: Real-time conflict detection cache

**Integration Strategy**:
- Direct API calls for authentication (scenario-authenticator), notifications (notification-hub), and NLP (Ollama)
- Event webhooks for scenario automation triggers
- Standard REST endpoints for cross-scenario event scheduling

**Non-Goals**:
- Video conferencing platform (integrates with existing tools)
- Email client (delegates to notification-hub)
- Task management (separate scenario concern)
- Time tracking (separate analytics scenario)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- PostgreSQL: Event storage, recurring patterns, metadata
- Qdrant: Vector embeddings for semantic search
- scenario-authenticator: Multi-user authentication and authorization
- notification-hub: Event reminders and notifications

**Optional Resources**:
- Ollama: Natural language scheduling (fallback to basic command parsing)

**Cross-Scenario Dependencies**:
- Provides time-based workflow orchestration to ALL scenarios
- Consumed by project-manager, customer-onboarding, automation scenarios

**Risks**:
- Timezone complexity (mitigated with industry-standard libraries and extensive testing)
- Notification delivery failure (mitigated with retry mechanisms and multiple delivery channels)
- AI scheduling accuracy (mitigated with confidence scoring and user confirmation)
- Database performance with large calendars (mitigated with proper indexing and pagination)

**Launch Sequencing**:
1. Core event CRUD with PostgreSQL (P0)
2. Authentication and multi-user support (P0)
3. Recurring patterns and reminders (P0)
4. UI with calendar views (P0)
5. Natural language scheduling with Ollama (P0)
6. Advanced features: conflict detection, optimization, analytics (P1/P2)

## 🎨 UX & Branding

**Visual Style**: Clean, professional calendar interface inspired by Google Calendar and Calendly. Light theme with dark mode support, readable sans-serif typography (Inter or similar), responsive grid layout with sidebar navigation.

**Interaction Design**: Fast, keyboard-friendly interactions (Linear-style), smooth drag-and-drop for event scheduling, subtle transitions, calendar-native gestures (swipe between months, quick event creation).

**Accessibility**: WCAG 2.1 AA compliance, full keyboard shortcuts, screen reader support, high color contrast, focus indicators, semantic HTML structure.

**Voice & Personality**: Professional, efficient, helpful. Users should feel in control of their time. Calm, organized, productive tone. Clear, actionable error messages and suggestions.

**Performance Promise**: Sub-200ms API responses (typically 7-10ms), sub-500ms semantic search, sub-2s natural language processing. Smooth 60fps UI animations.

## 📎 Appendix

**Supporting Documentation**: See README.md for setup guide, API specifications, and usage examples. Development history tracked in docs/PROGRESS.md. Detailed requirements in requirements/index.json.

**Related Scenarios**: Integrates with notification-hub (reminders), scenario-authenticator (multi-user), and provides temporal orchestration for downstream scenarios.
