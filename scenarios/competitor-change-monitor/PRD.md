# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Real-time competitive intelligence platform that autonomously tracks changes in competitor websites, GitHub repositories, and public data sources, providing AI-powered analysis and strategic alerts for high-impact changes
- **Primary users/verticals**: Product managers, business strategists, competitive intelligence teams, market analysts, executive leadership
- **Deployment surfaces**: CLI (manual scanning and competitor management), API (programmatic access to change data and analysis), UI (professional BI dashboard with timeline views and alerts)
- **Value promise**: Provides early warning system for competitor moves with intelligent change categorization, enabling proactive strategic responses and data-driven competitive positioning decisions

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Multi-source monitoring | Track changes across websites, GitHub repos, and public APIs with automated detection
- [ ] OT-P0-002 | Change detection and storage | Detect content differences and persist complete change history in PostgreSQL
- [ ] OT-P0-003 | AI-powered analysis | Categorize changes and assess business impact using Ollama-powered analysis
- [ ] OT-P0-004 | Alert system | Deliver configurable alerts for high-impact changes via notifications and UI indicators
- [ ] OT-P0-005 | Scheduled scanning | Automated periodic checks via internal scheduler
- [ ] OT-P0-006 | Business intelligence dashboard | Professional dark-themed UI with timeline views, data tables, and real-time notification panel

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Trend analysis | Historical change tracking with pattern recognition across competitors
- [ ] OT-P1-002 | Custom alert rules | User-configurable thresholds and filters for different change types
- [ ] OT-P1-003 | Export capabilities | Export change reports and analysis to common formats (CSV, JSON, PDF)
- [ ] OT-P1-004 | Competitive comparison | Side-by-side comparison views for pricing, features, and positioning
- [ ] OT-P1-005 | Integration webhooks | Push change notifications to Slack, Teams, and custom webhooks

### 🟢 P2 – Future / expansion ideas
- [ ] OT-P2-001 | Predictive analytics | ML-based prediction of competitor strategy shifts based on change patterns
- [ ] OT-P2-002 | Social media monitoring | Track competitor presence on Twitter, LinkedIn, and other platforms
- [ ] OT-P2-003 | Mobile app | iOS/Android app for on-the-go alerts and quick analysis review
- [ ] OT-P2-004 | Advanced scraping | Deep content analysis including pricing tables, feature matrices, and legal documents
- [ ] OT-P2-005 | Market intelligence APIs | Expose competitive intelligence to product-manager-agent and roi-fit-analysis scenarios

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (change detection and analysis orchestration), React UI (professional BI dashboard with dark theme)
- Data + storage expectations: PostgreSQL (competitor profiles, monitoring targets, change history, and alert configurations), Redis (content caching and real-time monitoring state), Qdrant (semantic search for change patterns and similarity analysis)
- Integration strategy: API-driven orchestration with internal scheduler; shared services (ollama for AI analysis, web-scraper for content extraction, notification-dispatcher for alerts); CLI for manual operations and testing
- Non-goals / guardrails: Not a full web scraping platform (focus on competitive intelligence), not a general-purpose monitoring tool (focus on strategic business changes), avoid overwhelming target sites with requests (use rate-limiter workflow)

## 🤝 Dependencies & Launch Plan
- Required resources: postgres (competitor and change storage), ollama (AI-powered change analysis), qdrant (pattern recognition and semantic search), browserless (web scraping fallback), agent-s2 (complex scraping scenarios)
- Scenario dependencies: None (standalone capability that other scenarios can leverage)
- Operational risks: Target site rate limiting or blocking could affect monitoring reliability; AI analysis quality depends on Ollama model capabilities; false positives in change detection could cause alert fatigue
- Launch sequencing: Phase 1 - Deploy with manual scanning and validation (2 weeks testing), Phase 2 - Enable automated scheduled scanning with conservative intervals (1 month monitoring), Phase 3 - Add alert rules and integrate with downstream scenarios (ongoing expansion)

## 🎨 UX & Branding
- Look & feel: Professional business intelligence aesthetic with dark theme and accent colors for alert levels (green for informational, yellow for noteworthy, red for critical, urgent red for immediate action), clean sortable data tables, timeline view of competitor changes, real-time notification panel
- Accessibility: High contrast color coding for alert severity, keyboard navigation for all dashboard operations, screen reader support for change summaries, color-blind friendly alert indicators using icons + colors
- Voice & messaging: Professional, analytical, data-driven intelligence - "Your strategic early warning system for competitive landscape shifts"
- Branding hooks: Alert level badges with clear iconography (🟢 Info, 🟡 Watch, 🔴 Critical, 🚨 Urgent), competitor cards with last-scanned timestamps and change counts, change categories with business impact scores

## 📎 Appendix (optional)

### Change Categories
- **Pricing**: Price adjustments, new tiers, promotional offers
- **Features**: New capabilities, feature removals, product updates
- **Marketing**: Messaging changes, positioning shifts, campaign launches
- **Legal**: Terms of service, privacy policy, compliance updates
- **Repository**: Code commits, releases, architecture changes

### Success Metrics
- **Coverage**: Monitor >20 competitors across multiple sources
- **Latency**: Detect changes within 6 hours of publication
- **Accuracy**: <5% false positive rate on high-impact alerts
- **Actionability**: >70% of critical alerts lead to strategic discussion or response
- **Reliability**: >99% uptime for scheduled scanning workflows

### API Endpoints
- `POST /api/competitors` - Add new competitor
- `GET /api/competitors` - List all monitored competitors
- `POST /api/scan/{id}` - Trigger manual scan
- `GET /api/changes` - Get recent changes with filters
- `GET /api/analysis/{id}` - Get AI-powered change analysis
- `GET /api/alerts` - Retrieve alert history

### CLI Commands
```bash
competitor-monitor add <name> <url>
competitor-monitor list
competitor-monitor scan [competitor-id]
competitor-monitor analyze <competitor-id>
competitor-monitor alerts [--since date]
```
