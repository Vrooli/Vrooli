# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Complete & Production Ready
> **Template Compliance**: Canonical PRD Template v2.0.0

## 🎯 Overview

**Purpose**: Professional-grade data visualization generation with customizable styling, supporting all major chart types (bar, line, pie, scatter, area, gantt, heatmap, treemap, candlestick). This becomes the foundational charting engine that every business report, research analysis, financial dashboard, and data presentation scenario can leverage programmatically.

**Primary Users**:
- Business analysts requiring professional visualizations
- Research teams needing publication-quality charts
- Report generation scenarios requiring programmatic chart creation
- Financial analysis tools needing specialized chart types
- Other Vrooli scenarios requiring data visualization capabilities

**Deployment Surfaces**:
- **CLI**: Command-line interface for automation and cross-scenario integration
- **API**: RESTful endpoints for chart generation, style management, and export
- **UI**: Web interface for style creation, preview, and template management
- **Automations**: Integration with N8N workflows for orchestrated chart generation

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Core chart types: bar, line, pie, scatter, area charts with configurable data inputs
- [ ] OT-P0-002 | JSON/CSV/direct data object ingestion with automatic data type detection
- [ ] OT-P0-003 | Default professional styling themes (light, dark, corporate, minimal)
- [x] OT-P0-004 | CLI interface for programmatic chart generation by other scenarios
- [ ] OT-P0-005 | Export capabilities: PNG, SVG formats for different use cases
- [x] OT-P0-006 | PostgreSQL integration for chart template and style persistence
- [ ] OT-P0-007 | Web UI for style management and preview with mock data

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Advanced chart types: gantt, heatmap, treemap charts
- [ ] OT-P1-002 | Candlestick charts for financial data
- [x] OT-P1-003 | Custom style builder with live preview and color palette management
- [ ] OT-P1-004 | Chart animation and interactivity options for web displays
- [x] OT-P1-005 | PDF export with vector graphics for print-quality reports
- [ ] OT-P1-006 | Chart composition (multiple charts in single canvas)
- [ ] OT-P1-007 | Data transformation pipeline (aggregation, filtering, sorting)
- [x] OT-P1-008 | Template library with industry-specific presets

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Real-time data streaming for live dashboard updates
- [ ] OT-P2-002 | 3D chart capabilities for advanced data visualization
- [ ] OT-P2-003 | Chart annotation system for explanatory text and arrows
- [ ] OT-P2-004 | Multi-language support for chart labels and legends
- [ ] OT-P2-005 | Chart versioning and diff capabilities
- [ ] OT-P2-006 | Integration with external design systems (Material, Bootstrap themes)

## 🧱 Tech Direction Snapshot

**Architecture**: API-first design with Go backend for performance, TypeScript/React UI for style management

**Chart Rendering**: D3.js for maximum flexibility and professional output quality over Chart.js (simplicity) or Plotly (built-in interactivity). D3.js provides finest control over output quality and customization needed for professional publications.

**Data Storage**: PostgreSQL for chart templates, custom styles, and generation history. Database-backed approach enables programmatic style creation and sharing across scenarios versus file-based templates.

**Export Pipeline**: browser-automation-studio (BAS) CaptureService for PNG generation — the renderer serves the chart HTML over its own HTTP surface and calls BAS via Connect-RPC to screenshot it (800x600 default), with fallback to Go native generation when BAS is unavailable; native SVG output from D3.js; PDF export with vector graphics support.

**Performance Target**: <2000ms for complex charts with 1000+ data points. Current: 15-18ms typical generation time.

**Integration Strategy**:
- Shared N8N workflows for chart generation orchestration and data processing
- CLI commands for cross-scenario integration
- Direct API access for custom chart rendering requirements

**Non-goals**:
- Real-time collaborative chart editing (P2+)
- Chart animation authoring tools (P2+)
- Built-in data source connectors (handled by consuming scenarios)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- `postgres` – Chart template, style, and generation history persistence
- `n8n` – Chart generation pipeline orchestration (optional, fallback to direct API)

**Scenario Dependencies**:
- `browser-automation-studio` – PNG export via CaptureService screenshot (optional; falls back to Go native generation)

**Optional Resources**:
- `minio` – Generated chart asset storage (fallback: local filesystem with cleanup policies)
- `redis` – Style and template caching (fallback: in-memory LRU cache)

**Upstream Dependencies**: None - foundational capability that other scenarios build upon

**Downstream Enablement**: Unlocks business-reports, research-assistant, financial-analyzer, marketing-analytics, project-management-visualizer scenarios

**Launch Sequencing**:
1. PostgreSQL schema initialization and seed data
2. API service deployment with health checks
3. CLI installation and cross-scenario integration testing
4. UI deployment for style management
5. N8N workflow activation for orchestrated generation

**Risk Mitigation**:
- Chart rendering performance degradation → Implement caching, optimize rendering pipeline
- Export quality inconsistencies → Comprehensive quality validation for all formats
- Data format compatibility → Robust parsing with clear error messages
- Style drift prevention → Version control for templates and styles

## 🎨 UX & Branding

**Visual Style**: Modern data visualization tool aesthetic (inspired by Tableau, Power BI) with clean, professional interface that inspires confidence in output quality

**Color Palette**: Light-mode primary with support for dark, corporate, and minimal themes. Accessible color palettes with WCAG 2.1 AA contrast ratios for all chart outputs.

**Typography**: Modern sans-serif for UI (system fonts), configurable typography in chart outputs supporting professional, technical, and creative presentations

**Motion Language**: Subtle animations in UI, optional chart animations for web displays (fade-in, progressive reveal, interactive hover states)

**Accessibility Commitments**:
- WCAG 2.1 AA compliance for all chart outputs and management interface
- Color-blind friendly default palettes
- Screen reader support for UI
- Keyboard navigation for style management

**Voice & Personality**: Professional, confident, focused. Messaging emphasizes data clarity and visualization quality. Technical accuracy without complexity.

**User Experience Promise**: "Transform raw data into professional visualizations with a single command" - minimal configuration, maximum quality output

## 📎 Appendix

**Reference Materials**: README.md, docs/api.md, docs/cli.md, api/internal/<domain>/storage/postgres/schema.sql, requirements/index.json

**Related Scenarios**: business-reports, research-assistant, financial-analyzer, marketing-analytics, project-management-visualizer

**External References**: D3.js Documentation, WCAG 2.1 Accessibility Guidelines, Color Theory for Data Visualization
