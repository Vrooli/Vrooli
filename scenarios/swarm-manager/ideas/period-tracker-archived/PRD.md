# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-19
> **Status**: Canonical Specification
> **Source of Truth**: PRD Control Tower (API/UI/CLI)

## 🎯 Overview

**Purpose**: Privacy-first menstrual health tracking with local AI-powered insights, multi-tenant support for households, and cross-scenario integration capabilities. This provides a completely private, locally-hosted alternative to corporate period tracking apps that may share sensitive health data.

**Primary users**: Privacy-conscious individuals and households requiring secure menstrual health tracking without external data sharing

**Deployment surfaces**:
- CLI for quick logging and predictions
- API for cross-scenario integration (calendar blocking, wellness optimization)
- UI for daily tracking, pattern visualization, and multi-tenant management
- Event system for real-time notifications and cross-scenario workflows

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Secure multi-tenant period cycle tracking with encrypted local storage
- [x] OT-P0-002 | Basic symptom logging (flow, pain, mood, physical symptoms)
- [x] OT-P0-003 | Cycle prediction algorithm with pattern recognition
- [x] OT-P0-004 | Calendar integration for period predictions and blocking
- [x] OT-P0-005 | Privacy-first architecture with no external data sharing
- [x] OT-P0-006 | Simple, intuitive UI optimized for daily use

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | AI-powered pattern recognition using local Ollama models
- [ ] OT-P1-002 | Symptom correlation insights ("Your headaches correlate with Day 2 of cycle")
- [ ] OT-P1-003 | Medication and supplement reminder system
- [ ] OT-P1-004 | Export functionality for medical appointments
- [ ] OT-P1-005 | Partner dashboard with consensual, discrete sharing
- [ ] OT-P1-006 | Historical trend analysis and reporting

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Photo tracking for skin/symptom changes
- [ ] OT-P2-002 | Integration with fitness/mood tracking scenarios
- [ ] OT-P2-003 | Custom reminder scheduling
- [ ] OT-P2-004 | Advanced fertility window calculations
- [ ] OT-P2-005 | Research contribution opt-in with anonymization

## 🧱 Tech Direction Snapshot

- **Preferred UI/API stacks**: Go API server with React/TypeScript UI, Vite build tooling
- **Data storage expectations**: PostgreSQL for encrypted persistence with AES-256 encryption, Redis for session caching, multi-tenant isolation enforced at database level
- **Integration strategy**: Direct resource access (PostgreSQL, Redis, Ollama) via connection strings for performance; REST API and event-driven architecture for cross-scenario integration
- **Non-goals**: Cloud synchronization (privacy-first means local-only), social features, external API integrations, mobile native apps (web-first responsive design)

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- **PostgreSQL**: Encrypted storage of period data, symptoms, predictions with ACID compliance
- **scenario-authenticator**: Multi-tenant user authentication and authorization for household privacy

**Optional Resources**:
- **Redis**: Session management and temporary data caching (fallback: in-memory sessions)
- **Ollama**: Local AI pattern recognition and insight generation (fallback: statistical analysis)

**Scenario Dependencies**:
- Enhances **calendar** scenario with period predictions for automatic blocking
- Integrates with **wellness-optimizer** for hormonal cycle correlation
- Enables **partner-dashboard** for discrete period tracking sharing

**Launch Risks**:
- Privacy compliance requires thorough security audit before household deployments
- Multi-tenant authentication isolation must be validated under concurrent load
- Prediction algorithm accuracy depends on minimum 3-cycle training data
- Calendar integration requires bidirectional event handling

**Launch Sequencing**:
1. Core P0 features with encrypted local storage and basic predictions
2. Integration testing with scenario-authenticator and PostgreSQL
3. Security audit and penetration testing for privacy validation
4. Calendar scenario integration and event system
5. P1 AI features with Ollama pattern recognition
6. Partner/family sharing with granular privacy controls

## 🎨 UX & Branding

**Visual Palette**: Warm pastels with privacy-focused dark mode option, modern readable typography, clean spacious layouts optimized for daily quick entry

**Personality**: Supportive, informative, non-judgmental tone with calm, private, empowering mood. Users should feel safe, informed, and in control of their health data.

**Accessibility Targets**:
- WCAG 2.1 AA compliance for all UI components
- Menstrual health stigma awareness in language and iconography
- Mobile-first responsive design (primary use case: phone in private settings)
- Clear data ownership messaging and granular privacy controls
- Secure by default, convenience as opt-in

**Brand Alignment**: Professional health tool with warm, supportive personality. Feels integrated with Vrooli while respecting the sensitive nature of menstrual health data. Medical-grade privacy with consumer app usability.

## 📎 Appendix

### Evolution Path

**Version 1.0 (Current)**:
- Core cycle tracking and symptom logging
- Basic prediction algorithm
- Multi-tenant authentication
- Calendar integration

**Version 2.0 (Planned)**:
- AI-powered pattern recognition with local Ollama
- Advanced symptom correlation analysis
- Partner/family sharing features
- Export functionality for medical visits

**Long-term Vision**:
- Complete local health ecosystem hub
- Research data contribution with full anonymization
- Integration with wearable devices for automatic tracking
- AI health coaching based on cycle patterns

### Intelligence Amplification

**Recursive Value**: Enables health pattern recognition across Vrooli scenarios, provides temporal correlation insights for mood/energy/productivity patterns, establishes privacy-first health data handling patterns for future medical scenarios

**New Scenarios Unlocked**:
- Health Dashboard Hub (aggregate period data with sleep, exercise, mood)
- Wellness Optimization Agent (AI lifestyle suggestions based on cycle patterns)
- Partner/Family Health Coordinator (discrete notifications and support systems)
- Medical Report Generator (automated summaries for healthcare providers)
- Research Data Contributor (anonymized, opt-in women's health research)

### CLI Command Reference

**Core Commands**:
- `period-tracker status` – Show operational status and resource health
- `period-tracker help` – Display command help and usage
- `period-tracker version` – Show CLI and API version information

**Tracking Commands**:
- `period-tracker log-cycle [date] --flow <intensity>` – Log new period cycle
- `period-tracker log-symptoms [date] --symptoms <list> --mood <1-10> --pain <0-10>` – Log daily symptoms
- `period-tracker predictions <user_id> --days <90>` – Show upcoming cycle predictions
- `period-tracker patterns <user_id> --timeframe <3m|6m|1y>` – Show detected health patterns

### API Endpoints

**Core Endpoints**:
- `POST /api/v1/cycles` – Log new cycle start (returns predictions)
- `GET /api/v1/predictions/{user_id}` – Get cycle predictions for calendar integration
- `POST /api/v1/symptoms` – Log daily symptoms (returns pattern detection)
- `GET /api/v1/patterns/{user_id}` – Retrieve detected correlations and insights

**Performance SLAs**:
- Response time: < 200ms for 95% of requests
- Throughput: 100 operations/second per user
- Availability: 99.9% uptime
- Resource usage: < 512MB memory, < 10% CPU

### Data Models

**Primary Entities**:
- **User**: Multi-tenant user accounts with encrypted email, preferences, timezone
- **Cycle**: Period cycles with start/end dates, flow intensity, encrypted notes, prediction status
- **DailySymptom**: Daily symptom logs with mood rating, pain level, flow level, encrypted notes
- **Prediction**: Cycle predictions with confidence scores and algorithm versioning

### Event System

**Published Events**:
- `period.cycle.started` – New cycle logged (calendar, wellness-tracker subscribers)
- `period.prediction.updated` – Predictions recalculated (calendar, notification-hub subscribers)
- `period.pattern.detected` – New correlation found (wellness-optimizer, health-dashboard subscribers)

**Consumed Events**:
- `calendar.event.created` – Check for conflicts with predicted periods
- `mood.rating.logged` – Correlate mood patterns with cycle phases

### Revenue Model

**Pricing Tiers**:
- Individual: $5/month (single user, full privacy features)
- Household: $12/month (up to 4 users, shared authentication)
- Family: $20/month (unlimited users, advanced sharing controls)

**Trial Period**: 30 days full-featured trial

**Market Differentiator**: Only fully local, AI-powered period tracking with multi-tenant support and no external data transmission
