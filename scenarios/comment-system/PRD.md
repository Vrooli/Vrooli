# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Active
> **Canonical Reference**: scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md

## 🎯 Overview

**Purpose**: A universal comment system microservice that provides collaborative discussion capabilities to all Vrooli scenarios. This creates a social layer for the entire ecosystem, enabling user engagement, feedback collection, and community building across every generated app.

**Primary Users**:
- End users of Vrooli scenarios who want to discuss and collaborate
- Scenario developers who need to configure comment behavior
- System administrators managing content and moderation

**Deployment Surfaces**:
- REST API for programmatic access by other scenarios
- JavaScript SDK for easy integration into any web UI
- Admin dashboard for per-scenario configuration management
- CLI tool for operational management

**Intelligence Amplification**:
- User Feedback Loop: Comments become training data for understanding user needs and preferences
- Cross-Scenario Learning: User discussions reveal patterns of how capabilities connect
- Community Intelligence: Collective user knowledge enhances problem-solving capabilities
- Engagement Metrics: Comment activity guides priority decisions for scenario improvements

**Recursive Value** – New scenarios enabled:
1. Community-Manager: Automated community management with sentiment analysis and trend detection
2. User-Research-Hub: Aggregate user feedback across all apps for product insights
3. Social-Analytics: Deep analysis of user engagement patterns to optimize experiences
4. Collaborative-Workspaces: Multi-user scenarios with real-time discussion capabilities
5. AI-Moderation-System: Intelligent content moderation leveraging comment patterns and user behavior

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Core CRUD operations for comments | Create, read, update, delete operations for comments with proper authentication
- [x] OT-P0-002 | Thread/reply support with nested comment structure | Hierarchical comment organization with parent-child relationships
- [x] OT-P0-003 | Integration with session-authenticator for user authentication | Verify user tokens and retrieve user profile information
- [x] OT-P0-004 | Integration with notification-hub for comment notifications | Send notifications for replies, mentions, and new comments
- [x] OT-P0-005 | PostgreSQL persistence with robust schema design | ACID-compliant storage with proper indexing and relationships
- [x] OT-P0-006 | REST API for programmatic access by other scenarios | Complete API surface with health checks and versioning
- [x] OT-P0-007 | JavaScript SDK for easy integration into any web UI | Framework-agnostic SDK with comprehensive documentation
- [x] OT-P0-008 | Admin dashboard for per-scenario configuration management | UI for configuring auth requirements, moderation, and themes
- [x] OT-P0-009 | Markdown rendering support for comment display | Safe markdown parsing with XSS prevention
- [x] OT-P0-010 | Per-scenario authentication requirements | Configurable optional vs required login per scenario
- [ ] OT-P0-011 | API response time < 200ms for 95% of CRUD operations | Meet performance SLA targets under normal load
- [ ] OT-P0-012 | Database query time < 100ms for comment fetching | Optimized queries with proper indexing for 10k+ comments
- [ ] OT-P0-013 | Resource usage < 512MB memory, < 20% CPU | Efficient resource utilization under normal load
- [ ] OT-P0-014 | CLI tool with status, help, version commands | Operational commands for service management
- [ ] OT-P0-015 | CLI list and create commands for comment management | User-facing commands for comment operations
- [ ] OT-P0-016 | CLI config command for scenario configuration | Administrative commands for per-scenario settings
- [ ] OT-P0-017 | CLI moderate command for admin operations | Moderation commands for content management
- [ ] OT-P0-018 | Event publishing for comment.created | Emit events for downstream consumers like notification-hub

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Real-time comment updates via WebSocket | Live updates without polling for improved UX
- [ ] OT-P1-002 | Comment search and filtering capabilities | Full-text search across comment content
- [ ] OT-P1-003 | User mention system | @username notifications with proper user lookup
- [ ] OT-P1-004 | Comment edit history and version tracking | Audit trail of all comment modifications
- [ ] OT-P1-005 | Configurable comment themes per scenario | Custom styling to match host scenario branding
- [ ] OT-P1-006 | Rich media attachment support | Image and file uploads (configurable per scenario)
- [ ] OT-P1-007 | Comment voting/rating system | User engagement features for comment quality
- [ ] OT-P1-008 | Export functionality for comment data | Data portability for analysis and backup
- [ ] OT-P1-009 | Throughput of 1000 comments/second sustained | Scale to high-traffic scenarios
- [ ] OT-P1-010 | WebSocket latency < 50ms | Real-time message delivery performance

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Integration hooks for AI-moderation-system scenario | Automated content moderation via ML models
- [ ] OT-P2-002 | Comment analytics and engagement metrics | Insights dashboard for community health
- [ ] OT-P2-003 | Advanced formatting toolbar for comment composition | Rich text editing experience
- [ ] OT-P2-004 | Comment templates and quick replies | Productivity features for common responses
- [ ] OT-P2-005 | Bulk moderation operations | Administrative efficiency for large-scale moderation
- [ ] OT-P2-006 | Comment scheduling functionality | Delayed publishing for strategic timing
- [ ] OT-P2-007 | Mobile-responsive comment widget | Optimized mobile experience
- [ ] OT-P2-008 | Voice comments and transcription | Accessibility and convenience features
- [ ] OT-P2-009 | AI-powered comment suggestions | Intelligent reply recommendations
- [ ] OT-P2-010 | Cross-scenario user reputation system | Community-wide trust and credibility scores

## 🧱 Tech Direction Snapshot

**Preferred Stack**:
- **Architecture**: Microservice with Go API, React admin dashboard, vanilla JavaScript SDK
- **Backend**: Go for performance and type safety
- **Database**: PostgreSQL for ACID compliance and complex querying
- **Caching**: Redis (optional) for frequently accessed comments
- **Frontend**: React for admin dashboard, vanilla JavaScript SDK for universal embedding

**Data Storage**:
- PostgreSQL primary storage: comments, configurations, edit history
- Redis cache (optional): hot comments, real-time updates
- Prepared statements and connection pooling for efficiency

**Integration Strategy**:
- REST API for scenario-to-scenario communication
- JavaScript SDK for web UI embedding (framework-agnostic)
- Event publishing to notification-hub for user notifications
- Session-authenticator for user authentication and profiles

**API Surface**:
- GET/POST /api/v1/comments/{scenario_name} - Comment CRUD
- PUT/DELETE /api/v1/comments/{comment_id} - Comment updates
- GET/POST /api/v1/config/{scenario_name} - Configuration management
- GET /api/v1/health - Service health checks

**CLI Architecture**:
- Thin wrapper over Go API client
- Standard commands: status, help, version
- Custom commands: list, create, config, moderate
- Configuration: ~/.vrooli/comment-system/config.yaml

**Non-goals**:
- Direct file storage (delegate to future file-storage service)
- Built-in AI moderation (integrate with future ai-moderation-system)
- Custom authentication (delegate to session-authenticator)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- PostgreSQL: Essential for data persistence and ACID compliance
- Session-Authenticator Scenario: Required for user authentication and profile data
- Notification-Hub Scenario: Required for sending comment notifications and mentions

**Optional Resources**:
- Redis: Caching frequently accessed comments and real-time updates (fallback: direct database queries)

**Integration Requirements**:
- Session-authenticator API for token verification
- Notification-hub event API for notification delivery
- CORS configuration for cross-origin SDK usage

**Risks & Mitigations**:
- Database performance under high comment volume: Connection pooling, indexing, pagination
- Session-authenticator service unavailability: Graceful fallback to anonymous comments where configured
- Cross-origin issues with SDK: Comprehensive CORS configuration and testing
- WebSocket connection limits: Connection pooling and fallback to polling
- Spam and abuse: Rate limiting, integration hooks for AI moderation
- Data privacy: Compliance with user data deletion requests

**Launch Sequence**:
1. Core API implementation with PostgreSQL schema
2. Session-authenticator integration for authentication
3. Notification-hub integration for notifications
4. JavaScript SDK development and testing
5. Admin dashboard for configuration
6. CLI tool for operational management
7. Documentation and integration guides
8. Performance testing and optimization

**Downstream Enablement**:
- AI-Moderation-System: Automated content moderation with training data from comment patterns
- Community-Analytics: Deep insights into user engagement across all Vrooli scenarios
- Social-Features: Foundation for likes, follows, user reputation systems
- Collaborative-Tools: Multi-user scenarios with built-in discussion capabilities

## 🎨 UX & Branding

**Visual Style**:
- Adaptive color scheme: Matches host scenario theme
- Clean typography: Sans-serif, readable at small sizes
- Threaded layout: Nested comment structure with expand/collapse
- Subtle animations: Smooth expand/collapse, typing indicators

**Inspiration**: GitHub comments, Discord chat, modern commenting systems

**Personality**:
- Tone: Friendly and encouraging participation
- Mood: Focused and stays out of the way
- Target feeling: "Safe to contribute and discuss"

**Admin Dashboard Style**:
- Professional SaaS dashboard aesthetic
- Light color scheme for clarity
- Modern, clean, information-dense typography
- Multi-panel configuration view
- Fast, functional interface (no animations)

**Accessibility**:
- WCAG 2.1 AA compliance for keyboard navigation
- Screen reader support for all interactive elements
- Sufficient color contrast for readability
- Clear focus indicators for keyboard users

**Responsive Design**:
- Mobile-first design for comment widget
- Desktop-optimized admin dashboard
- Touch-friendly interaction targets
- Adaptive layouts for different screen sizes

**Brand Consistency**:
- Scenario Identity: Clean, universal design that adapts to host scenario
- Vrooli Integration: Consistent API patterns and error handling across ecosystem
- Professional Focus: Business tool that enhances other scenarios without distraction

## 📎 Appendix

**References**:
- README.md - User-facing overview and integration guide
- docs/api.md - Complete API specification
- docs/cli.md - CLI command documentation
- docs/sdk.md - JavaScript SDK documentation
- docs/integration.md - How to integrate into scenarios
- [CommonMark Specification](https://commonmark.org/) - Markdown parsing standard
- [WebSocket RFC 6455](https://tools.ietf.org/html/rfc6455) - Real-time communication
- [WCAG 2.1](https://www.w3.org/WAI/WCAG21/quickref/) - Accessibility guidelines

---

**Last Updated**: 2025-11-18
**Status**: Active
**Owner**: AI Agent (Claude Code)
**Review Cycle**: Every implementation milestone
