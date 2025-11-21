# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Active
> **Template**: Canonical PRD Template v2.0.0

## 🎯 Overview

The Bookmark Intelligence Hub transforms scattered social media bookmarks into organized, actionable intelligence. It automatically discovers, extracts, categorizes, and suggests actions on bookmarks across X, Reddit, TikTok, and other platforms. This capability enables other Vrooli scenarios to leverage curated content intelligence for workflows like recipe extraction, workout planning, research assistance, and content recommendations.

**Purpose**: Add permanent capability to automatically process social media bookmarks into structured, actionable data that feeds other scenarios.

**Primary Users**: Knowledge workers who consume social media for professional and personal growth; teams managing shared content across platforms.

**Deployment Surfaces**:
- CLI for automation and scripting
- REST API for scenario integration
- Web UI for profile management and action approval
- Event bus for real-time cross-scenario notifications

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Multi-tenant authentication system with profile-specific bookmark processing
- [ ] OT-P0-002 | Modular social platform integration system supporting X, Reddit, TikTok
- [ ] OT-P0-003 | AI-powered content categorization with user-defined buckets
- [ ] OT-P0-004 | Action suggestion engine with approval/rejection workflow
- [ ] OT-P0-005 | Cross-scenario integration API for other scenarios to consume organized data
- [ ] OT-P0-006 | Real-time bookmark processing with configurable polling intervals

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Integration with data-structurer scenario for unstructured text processing
- [ ] OT-P1-002 | Integration with scenario-authenticator for multi-tenant support
- [ ] OT-P1-003 | Intelligent learning system that improves categorization over time
- [ ] OT-P1-004 | Bulk action approval for similar content types
- [ ] OT-P1-005 | Export functionality to external systems (CSV, JSON, API)
- [ ] OT-P1-006 | Advanced filtering and search across organized bookmarks

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Browser extension for real-time bookmark categorization
- [ ] OT-P2-002 | Mobile app for on-the-go bookmark management
- [ ] OT-P2-003 | Advanced analytics on bookmark patterns and productivity outcomes
- [ ] OT-P2-004 | Integration with calendar systems for time-based action scheduling
- [ ] OT-P2-005 | Collaborative bookmark sharing within teams

## 🧱 Tech Direction Snapshot

**Architecture**: Go API + React UI following Vrooli scenario patterns
- API: Go 1.21+ with database/sql for PostgreSQL, standard library HTTP server
- UI: React + Vite with TypeScript, TailwindCSS for styling
- CLI: Go binary wrapping API client for thin-wrapper pattern

**Data Storage**: PostgreSQL for all persistent data (bookmark metadata, categorization rules, user preferences, action history)

**Integration Strategy**:
1. Prefer shared workflows via Huginn agents (social media scraping patterns)
2. Use resource CLIs (resource-huginn, resource-browserless, resource-postgres)
3. Direct platform APIs only when necessary (authenticated bookmark access)

**Resource Dependencies**: postgres (required), huginn (required), browserless (required fallback), qdrant (optional semantic matching), ollama (optional local LLM)

**Non-goals**:
- Not building a generic social media management tool
- Not storing full social media history (only bookmarks)
- Not replacing existing bookmark managers (focused on intelligence extraction)

## 🤝 Dependencies & Launch Plan

**Required Before Launch**:
- PostgreSQL resource operational
- Huginn resource with bookmark scraping agents
- Browserless resource for fallback content extraction
- Basic scenario-authenticator integration for multi-tenant support

**Scenario Dependencies**:
- scenario-authenticator: Multi-tenant user management (P1 - has fallback single-user mode)
- data-structurer: Unstructured content processing (P1 - has fallback regex extraction)

**Launch Sequence**:
1. Core bookmark processing engine with X/Reddit/TikTok support
2. Basic categorization with keyword matching
3. Action suggestion system with manual approval
4. Cross-scenario API for consumption by recipe-book, workout-plan-generator
5. Enhanced ML-based categorization (P1)
6. Learning system for continuous improvement (P1)

**Risks**:
- Social media API rate limiting (mitigated with intelligent backoff + browserless fallback)
- Bot detection blocking (mitigated with user-agent rotation + stealth mode)
- Content extraction accuracy (mitigated with multiple extraction methods + user feedback)

## 🎨 UX & Branding

**Visual Style**: Modern productivity dashboard with social media integration aesthetics
- Clean, professional interface that doesn't feel overwhelming despite complex data
- Light color scheme with platform-specific accent colors
- Modern typography, dashboard layout, subtle animations
- Desktop-first with tablet support, mobile view for approval actions

**Tone & Personality**: Friendly but focused - users should feel organized and in control of their digital consumption

**Accessibility**: WCAG 2.1 AA compliance
- Keyboard navigation for all functions
- Screen reader compatibility
- Color contrast meeting accessibility standards
- Focus indicators and skip links

**Brand Identity**: Professional productivity tool that bridges social media consumption and productive action-taking within the Vrooli ecosystem

## 📎 Appendix

**Intelligence Amplification**

Every bookmark processed adds to growing intelligence about user interests, content patterns, and action preferences. Future agents leverage this to:
- Pre-populate scenario inputs based on past bookmark categories
- Suggest relevant content from the organized bookmark database
- Learn from user approval/rejection patterns
- Build comprehensive user interest profiles
- Create cross-scenario workflows triggered by bookmark categories

**Recursive Value - What This Enables**

After this exists, these become possible:
1. Content Recommendation Engine using bookmark patterns
2. Personal Knowledge Graph with semantic relationships
3. Automated Workflow Triggers based on content types
4. Cross-Platform Content Synthesis into unified reports
5. Smart Content Scheduling based on timing patterns

**Data Models**

- **BookmarkProfile**: User profiles with platform configs and categorization rules (PostgreSQL)
- **BookmarkItem**: Individual bookmarks with content, metadata, categories, confidence scores (PostgreSQL)
- **ActionItem**: Suggested actions linked to bookmarks with approval status (PostgreSQL)
- **CategoryRule**: User-defined categorization rules with keywords and patterns (PostgreSQL)

**API Endpoints**

- `POST /api/v1/bookmarks/process` - Process new bookmarks (< 2s response time)
- `GET /api/v1/bookmarks/query` - Query organized data (< 500ms response time)
- `POST /api/v1/actions/approve` - Bulk approve/reject actions (< 1s response time)
- `GET /api/v1/profiles` - Manage bookmark profiles
- `GET /api/v1/categories` - Manage categorization rules

**CLI Commands**

- `bookmark-intelligence-hub status` - Show operational status
- `bookmark-intelligence-hub profile create|list|update|delete` - Manage profiles
- `bookmark-intelligence-hub process <profile_id>` - Process bookmarks
- `bookmark-intelligence-hub query <profile_id>` - Query organized data
- `bookmark-intelligence-hub actions list|approve|reject|bulk` - Manage actions
- `bookmark-intelligence-hub categories list|create|update|delete` - Manage rules

**Cross-Scenario Integration**

*Provides To*:
- recipe-book: Auto-adds recipes from bookmarked content via API
- workout-plan-generator: Suggests workouts from fitness bookmarks via events
- research-assistant: Provides research material from organized bookmarks
- content-recommendation: Feeds bookmark patterns for personalization

*Consumes From*:
- scenario-authenticator: User profile management and authentication
- data-structurer: Structured extraction from unstructured content

**Performance Targets**

| Metric | Target |
|--------|--------|
| API Response Time | < 500ms (95th percentile) |
| Throughput | 1000 bookmarks/hour |
| Categorization Accuracy | > 85% initially, > 95% with learning |
| Resource Usage | < 2GB memory, < 50% CPU |

**Security & Privacy**

- Social media credentials encrypted at rest
- Profile-based access control via scenario-authenticator
- Complete audit trail of all processing and approvals
- Local content analysis (no external data sharing without consent)

**Related Documentation**

- README.md: User-facing setup and usage
- docs/api.md: Complete API specification
- docs/cli.md: CLI command reference
- docs/integration.md: Cross-scenario integration guide

**References**

- Canonical PRD Template: scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md
- Huginn Documentation: Agent-based automation patterns
- Browserless API: Headless browser automation
