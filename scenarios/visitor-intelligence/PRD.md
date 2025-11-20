# Product Requirements Document (PRD)

> **Template Version**: 2.0.0
> **Last Updated**: 2025-11-19
> **Status**: Canonical Specification
> **Scenario**: visitor-intelligence

## 🎯 Overview

Visitor-intelligence provides real-time website visitor identification, behavioral tracking, and retention marketing automation for all Vrooli scenarios. This capability transforms anonymous traffic into actionable visitor profiles, creating a permanent intelligence layer that remembers and learns from every interaction across the entire ecosystem.

**Primary Users**:
- Developers integrating visitor tracking into Vrooli scenarios
- Marketers analyzing visitor behavior and running retention campaigns
- Product managers optimizing conversion funnels

**Deployment Surfaces**:
- API: RESTful endpoints for visitor data retrieval and management
- CLI: Visitor profile queries and analytics commands
- UI: Real-time dashboard showing live visitor activity and behavioral insights
- JavaScript Library: One-line integration script for any scenario

**Permanent Capability Added**:
Every Vrooli scenario gains visitor context awareness—agents can understand user behavior patterns, preferences, and intent without explicit input. This creates compound intelligence where scenarios can proactively adapt to user needs, trigger retention workflows, and share insights across the entire ecosystem.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | JavaScript tracking pixel with fingerprinting | Implement client-side fingerprinting achieving 40-60% visitor identification rate across sessions
- [ ] OT-P0-002 | Real-time visitor session tracking with PostgreSQL storage | Track visitor sessions with events, page views, and behavioral data stored in PostgreSQL
- [ ] OT-P0-003 | RESTful API for visitor data retrieval and management | Provide GET/POST endpoints for tracking pixels, visitor profiles, and analytics queries
- [ ] OT-P0-004 | One-line integration script for any scenario | Distribute tracking script via simple script tag integration
- [ ] OT-P0-005 | Privacy-compliant data collection with GDPR controls | Implement consent management, data retention limits, and purging capabilities

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Real-time dashboard showing live visitor activity | Build UI displaying current visitors, session counts, and top pages
- [ ] OT-P1-002 | Visitor profile enrichment with behavioral insights | Analyze visitor patterns to generate insights about intent and preferences
- [ ] OT-P1-003 | Event-driven retention trigger system | Publish visitor.identified and visitor.abandoned events for downstream scenarios
- [ ] OT-P1-004 | Cross-scenario visitor journey tracking | Track visitor movement across multiple Vrooli scenarios in unified profiles
- [ ] OT-P1-005 | Audience segmentation and export tools | Enable filtering, segmentation, and CSV export of visitor cohorts

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Predictive analytics using Ollama for intent inference | Use local LLMs to predict visitor intent and conversion probability
- [ ] OT-P2-002 | WebSocket real-time notifications | Push live visitor activity updates to connected dashboard clients
- [ ] OT-P2-003 | Advanced fingerprinting with Canvas/WebGL | Enhance identification rate using canvas fingerprinting and WebGL parameters
- [ ] OT-P2-004 | Integration with external marketing tools | Export visitor data to email platforms and CRM systems

## 🧱 Tech Direction Snapshot

**API Stack**: Go 1.21+ with standard library HTTP server, database/sql for PostgreSQL, go-redis client

**UI Stack**: React + Vite with TypeScript, real-time charts using lightweight visualization library

**Data Storage**:
- PostgreSQL for persistent visitor profiles, events, and sessions with optimized indexing
- Redis for real-time session caching, rate limiting, and hot visitor data

**Integration Strategy**:
- Direct database access for high-performance tracking (sub-100ms response time requirement)
- Event publishing to n8n for retention triggers and cross-scenario workflows
- JavaScript library for client-side fingerprinting and event collection

**Non-Goals**:
- Not building a full-featured analytics platform like Google Analytics
- Not replacing specialized A/B testing tools
- Not handling payment or transaction tracking (scenarios handle their own business logic)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- **postgres**: Visitor profiles, events, and session data storage (direct SQL via Go database/sql)
- **redis**: Real-time session caching and rate limiting (TCP connection to Redis instance)

**Optional Local Resources**:
- **ollama**: Visitor intent analysis and behavioral insights (fallback to rules-based classification)

**Launch Prerequisites**:
1. PostgreSQL schema initialized with visitors, visitor_events, and visitor_sessions tables
2. Redis available for session caching
3. Database migrations tested with sample visitor data
4. JavaScript tracking library bundled and served from API endpoint

**Risks**:
- High tracking volume could overwhelm database → Mitigation: Redis caching, batch writes, connection pooling
- Browser fingerprinting blocked by privacy extensions → Mitigation: Graceful degradation to session-based tracking
- GDPR compliance violations → Mitigation: Built-in consent management and automatic data retention limits

## 🎨 UX & Branding

**Visual Style**: Modern analytics dashboard inspired by Mixpanel and Amplitude—dark color scheme, clean typography, data-dense layouts with subtle animations

**Personality**: Technical and focused tone, empowering users with actionable data insights

**Accessibility Commitments**: WCAG AA compliance for dashboard components, keyboard navigation support, sufficient color contrast for data visualization

**User Experience Promise**:
- Developers: Single script tag integration, no complex setup
- Marketers: Clean interface with powerful filtering and export capabilities
- Privacy-conscious: Clear consent controls, transparent data handling, self-hosted alternative to commercial tools

**Responsive Design**: Desktop-first dashboard for analytics workloads, mobile-optimized tracking script for universal browser support

## 📎 Appendix

### Intelligence Amplification
Every Vrooli scenario gains visitor context awareness—agents can understand user behavior patterns, preferences, and intent without explicit input. This creates compound intelligence where scenarios can proactively adapt to user needs, trigger retention workflows, and share insights across the entire ecosystem. Agents become predictive rather than reactive.

### Recursive Value – New Scenarios Enabled
1. **Personalized Content Engine**: Dynamically adapt scenario UIs based on visitor behavior patterns
2. **Retention Campaign Orchestrator**: Automated email/SMS campaigns triggered by specific visitor actions
3. **A/B Testing Platform**: Real-time experimentation with visitor segmentation
4. **Behavioral Analytics Dashboard**: Cross-scenario user journey analysis and optimization
5. **Intent Prediction System**: ML models to predict visitor actions before they happen

### Business Value
- **Revenue Potential**: $15K - $50K per deployment (similar to retention.com pricing model)
- **Cost Savings**: Eliminates need for external visitor identification services
- **Market Differentiator**: Privacy-first, self-hosted alternative to commercial tracking tools
- **Reusability Score**: 10/10—every Vrooli scenario can leverage visitor intelligence

### Performance Targets
- Tracking pixel response time: < 50ms
- API throughput: 1000+ events/second
- Visitor identification accuracy: 40-60%
- Memory usage: < 2GB, CPU < 25%

### Related Documentation
- README.md: Integration guide and overview
- docs/api.md: Complete API specification
- docs/privacy.md: GDPR compliance guide
- docs/integration.md: Scenario integration examples
