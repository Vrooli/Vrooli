# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Canonical Specification

## 🎯 Overview

**Purpose**: The funnel-builder adds the permanent capability to create, deploy, and optimize multi-step conversion funnels (sales, marketing, onboarding) with visual drag-and-drop editing, branching logic, lead capture, and comprehensive analytics. This becomes Vrooli's universal customer acquisition and conversion engine that any scenario can leverage to monetize or qualify users.

**Primary Users**:
- Marketers building lead generation funnels
- Product teams creating onboarding flows
- Sales teams building qualification processes
- Business owners monetizing scenarios

**Deployment Surfaces**:
- UI: Visual drag-and-drop builder and analytics dashboard
- API: RESTful endpoints for funnel creation and execution
- CLI: Command-line tools for funnel management
- Integrations: Embeddable in other Vrooli scenarios

**Intelligence Amplification**: This capability provides agents with conversion optimization intelligence, lead qualification patterns, user journey mapping, A/B testing framework, and revenue attribution for measuring business impact.

**Recursive Value**: Enables saas-billing-hub (checkout funnels), app-onboarding-manager (onboarding flows), survey-monkey (conditional surveys), roi-fit-analysis (prospect qualification), and competitor-change-monitor (lead magnets).

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Visual drag-and-drop funnel builder | React/TypeScript UI with intuitive step creation and ordering
- [x] OT-P0-002 | Multiple step types support | Quiz, form, content, and CTA step types with customizable fields
- [x] OT-P0-003 | Lead capture and storage | PostgreSQL-backed lead storage with full response tracking
- [x] OT-P0-004 | Linear funnel flow execution | Sequential step progression with session management
- [x] OT-P0-006 | API endpoints for funnel operations | Complete CRUD and execution endpoints with JSON responses
- [x] OT-P0-007 | Mobile-responsive funnel display | Mobile-first responsive design for funnel execution

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Branching logic support | User response-based conditional routing between steps
- [x] OT-P1-003 | Analytics dashboard | Conversion rates, drop-off points, and lead metrics visualization
- [x] OT-P1-004 | Template library | Pre-built professional funnels for common use cases
- [x] OT-P1-005 | Dynamic content personalization | User-specific content rendering based on responses
- [x] OT-P1-006 | Lead export functionality | CSV and JSON export for external processing

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | AI-powered copy generation | Ollama-based compelling funnel content creation
- [ ] OT-P2-002 | Predictive analytics | ML-based conversion optimization recommendations
- [ ] OT-P2-003 | Webhook integrations | External service notifications for lead events
- [ ] OT-P2-004 | Advanced segmentation | User targeting and audience-specific funnel variants
- [ ] OT-P2-005 | Multi-tenant authentication | Scenario-authenticator integration for isolated tenants
- [ ] OT-P2-006 | A/B testing framework | Variant testing with statistical significance tracking

## 🧱 Tech Direction Snapshot

**UI Stack**: React + TypeScript + Vite for drag-and-drop builder and analytics dashboard. Component-based architecture with real-time preview. Mobile-first responsive design matching Typeform/ConvertFlow UX patterns.

**API Stack**: Go for high-performance REST API. Gin framework for routing. Direct PostgreSQL integration for persistence. Health check endpoints for lifecycle monitoring.

**Data Storage**: PostgreSQL for funnels, steps, leads, and analytics. JSONB columns for flexible step content and branching rules. Optional Redis for session caching in high-traffic deployments.

**Integration Strategy**: RESTful API for cross-scenario usage. CLI wraps API endpoints for agent access. Embeddable iframe support for external deployments. Optional Ollama integration for AI copy generation.

**Non-Goals**: Not a general-purpose form builder. Not replacing scenario-authenticator (multi-tenancy optional). Not a marketing automation platform (focused on conversion funnels only).

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- postgres: All data persistence (funnels, steps, leads, analytics)
- Resource initialized via schema.sql and seed.sql with 4 professional templates

**Optional Resources**:
- redis: Session caching for high-traffic scenarios (PostgreSQL fallback available)
- ollama: AI-powered copy generation for funnel steps (template fallback available)

**Scenario Dependencies**:
- scenario-authenticator: Multi-tenant support (deferred to v2.0, single-tenant mode works)

**Launch Risks**:
- Database connection failures mitigated with exponential backoff retry and circuit breaker
- Port conflicts handled by Vrooli lifecycle dynamic allocation
- UI bundle size (700KB) mitigated with tree shaking, code splitting planned for v2.0
- CORS issues addressed with proxy configuration and trusted proxy disabled

**Launch Sequence**:
1. PostgreSQL resource startup and schema initialization
2. API server startup with health check validation
3. UI dev server or production build serving
4. CLI installation and PATH configuration
5. Template seeding and verification

## 🎨 UX & Branding

**Visual Style**: Modern SaaS inspired by Typeform and ConvertFlow. Light color scheme with customizable accent colors. Clean typography with high readability. Spacious layouts focusing on one step at a time. Subtle animations for step transitions.

**Builder Experience**: Drag-and-drop interface with clear visual hierarchy. Live preview panel alongside builder. Template gallery with thumbnail previews. Prominent CTAs and progress indicators. Professional but approachable aesthetic.

**Funnel Display**: One step per screen to maintain focus. Clear progress bar showing completion percentage. Smooth transitions between steps. Mobile-first responsive design ensuring cross-device consistency.

**Personality**: Friendly but professional tone. Confident and trustworthy mood. Target feeling: "This is easy and I'm making progress." Accessibility: WCAG 2.1 Level AA compliance with color contrast ratios and keyboard navigation.

**Brand Promise**: Local-first conversion optimization without SaaS fees. AI-native features for intelligent copy generation. Infinitely customizable and open source. Seamlessly integrates with Vrooli ecosystem.

## 📎 Appendix

**Performance Targets**
- Response Time: < 200ms for step transitions
- Throughput: 1000 concurrent funnel sessions
- Conversion Rate: > 15% for optimized funnels
- Resource Usage: < 512MB memory, < 10% CPU

**Value Proposition**
- Revenue Potential: $10K-$50K per deployment as standalone SaaS
- Cost Savings: Replaces $200-500/month third-party funnel tools
- Reusability Score: 10/10 - Every revenue scenario benefits from conversion funnels
- Market Differentiator: Local-first, AI-native, infinitely customizable

**Evolution Path**
- v1.0 (Current): Core builder, basic analytics, linear flows, 4 templates
- v2.0 (Planned): Branching logic, A/B testing, advanced analytics, AI copy generation, webhooks
- Long-term: Predictive optimization via ML, cross-funnel journey mapping, revenue attribution across Vrooli ecosystem

**External References**
- [ConversionXL Institute - Funnel Optimization](https://conversionxl.com/)
- [BJ Fogg's Behavior Model](https://behaviormodel.org/)
- [Optimizely A/B Testing Guide](https://www.optimizely.com/optimization-glossary/ab-testing/)
- [HubSpot Lead Generation](https://www.hubspot.com/lead-generation)

**Internal References**
- Resource postgres: `scripts/resources/resource-postgres/`
- Resource ollama: `scripts/resources/resource-ollama/`
- Scenario authenticator: `scenarios/scenario-authenticator/`
- Lifecycle system: `scripts/lib/lifecycle/`
- Testing architecture: `docs/testing/architecture/PHASED_TESTING.md`

**Design Inspiration**
- Typeform: Simple one-question-at-a-time flow
- ConvertFlow: Visual funnel builder with drag-and-drop
- ClickFunnels: Complete funnel ecosystem
- Leadpages: Landing page and lead capture focus

**Related Scenarios**
- saas-billing-hub: Checkout funnels for monetization
- app-onboarding-manager: User onboarding flows
- survey-monkey: Survey funnels with conditional logic
- roi-fit-analysis: Lead qualification funnels
- competitor-change-monitor: Lead magnet funnels
