# Product Requirements Document (PRD)

## 🎯 Overview
- **Purpose**: Transform Vrooli's manual setup process into a guided, user-friendly onboarding experience
- **Primary users/verticals**: Non-technical users, new developers, returning configuration managers
- **Deployment surfaces**: Web UI, CLI, REST API
- **Value promise**: Reduce time-to-first-working-agent from hours to under 10 minutes through guided setup

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Resource Configuration Wizard | Complete guided setup flow for coding agents and AI providers with 100% validation coverage
- [x] OT-P0-002 | Configuration State Management | Generate and validate service.json with all dependencies resolved and proper schema compliance

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Health Monitoring Dashboard | Real-time resource status visualization with health checks and status indicators
- [x] OT-P1-002 | Progress Persistence System | Save and resume capability for partial onboarding progress across sessions

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Smart Flow Optimization | Intelligent setup order suggestions while maintaining flow flexibility
- [x] OT-P2-002 | User-Friendly Documentation | Context-aware help content with plain language descriptions replacing technical terms

## 🧱 Tech Direction Snapshot
- Preferred stacks: Go (API/CLI), React/TypeScript (UI)
- Data + storage expectations: Local file system (.vrooli/), HTTP endpoints for health checks
- Integration strategy: API-first architecture with thin CLI/UI layers
- Non-goals: Custom resource implementations, complex workflow automation

## 🤝 Dependencies & Launch Plan
- Required resources: running-resources.json, service.json, secrets.json
- Scenario dependencies: tunnel-manager, browser-automation-studio
- Operational risks: API key security, configuration file corruption, health check reliability
- Launch sequencing:
  1. Core API implementation
  2. CLI frontend
  3. Web UI deployment
  4. Integration validation

## 🎨 UX & Branding
- Look & feel: Clean, minimal interface with clear progress indicators
- Accessibility: WCAG 2.1 AA compliance, keyboard navigation support
- Voice & messaging: Friendly, instructional tone avoiding technical jargon
- Branding hooks: Standard Vrooli color scheme and component library

## 📎 Appendix
- Reference: GuidedTour implementation from browser-automation-studio
- Schema: service.json validation requirements
- Health check endpoint specifications