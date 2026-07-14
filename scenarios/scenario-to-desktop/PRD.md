# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

**Purpose**: Transform any existing Vrooli scenario into a professional native desktop application. Provides comprehensive Electron templates, cross-platform packaging configuration, development tooling, and automated distribution pipelines so that any scenario can become a standalone desktop app with native OS integration, offline capability, and professional-grade UX.

**Primary users/verticals**:
- Developers and product teams building desktop applications from Vrooli scenarios
- Enterprise teams needing offline-capable or system-integrated deployments
- Creators packaging AI scenarios for distribution via app stores or direct installers

**Deployment surfaces**:
- API: Go HTTP server for desktop generation, build tracking, and pipeline orchestration
- CLI: `scenario-to-desktop` binary with generate, build, test, and package commands
- UI: React + Vite dashboard for template selection, build monitoring, and configuration
- Templates: Electron-based scaffolding (basic, advanced, kiosk, multi-window)

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Generate complete Electron desktop applications | API generates template files successfully with required assets and configs
- [ ] OT-P0-002 | Multi-framework scaffolding with Electron primary | Electron fully implemented; Tauri/Neutralino tracked as P2 enhancements
- [ ] OT-P0-003 | Cross-platform packaging | Template configs exist; full builds require electron-builder in target environments
- [ ] OT-P0-004 | Development tooling | Make targets, CLI commands, and test infrastructure are in place
- [ ] OT-P0-005 | Integration with scenario APIs | Templates include secure IPC and API integration patterns
- [ ] OT-P0-006 | Native OS features | Menus, tray, notifications, and file dialogs implemented in templates
- [x] OT-P0-007 | Auto-updater hooks | electron-updater wiring included; release pipeline pending

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Code signing + notarization | Automate per-platform signing workflows
- [ ] OT-P1-002 | App store submission automation | Microsoft Store + Mac App Store pipelines
- [ ] OT-P1-003 | Multi-window workflows | Advanced window orchestration and state persistence
- [ ] OT-P1-004 | Performance monitoring + analytics | In-app metrics and build telemetry dashboards
- [ ] OT-P1-005 | Enterprise deployment features | MSI/PKG installers and silent install flows

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Visual desktop app builder | Drag-and-drop configuration UI
- [ ] OT-P2-002 | Plugin architecture for templates | Extensible template + feature packs
- [ ] OT-P2-003 | Kiosk mode enhancements | Dedicated hardware management and remote controls

## 🧱 Tech Direction Snapshot

Preferred stacks and architectural intent:

- **API**: Go HTTP server with screaming architecture (domain handlers for build, pipeline, records, scenario, system, tools)
- **UI**: React + Vite SPA with iframe-bridge interop for Vrooli dashboard embedding
- **Templates**: Electron-first with template variants (basic, advanced, kiosk, multi-window); Tauri/Neutralino deferred to P2
- **Build system**: electron-builder for packaging; Wine/Flatpak for cross-platform Windows builds on Linux
- **Data storage**: Filesystem-based (template files, build artifacts, deployment telemetry); optional postgres/redis for history and caching
- **Integration**: Secure IPC between Electron main/renderer processes; scenario API communication via bundled runtime or thin-client proxy
- **Non-goals**: Mobile targets, web-only deployment (those belong in other scenarios)

## 🤝 Dependencies & Launch Plan

**Required resources**: None mandatory; optional dependencies include browser-automation-studio (UI testing), postgres (template/build history), and redis (build cache)

**Scenario dependencies**:
- deployment-manager for bundled builds and tier-2 automation
- app-monitor for proxy URL discovery and external access

**Operational risks**:
- Cross-platform packaging requires electron-builder plus Wine/macOS CI runners
- Code signing and notarization require manual certificate setup
- Electron security vulnerabilities require regular dependency updates
- Large bundle sizes (~100-200MB) inherent to Electron; Tauri alternative planned for P2

**Launch sequencing**:
1. Start scenario-to-desktop (`make start` or `vrooli scenario start scenario-to-desktop`)
2. Configure proxy URL for target scenario + validate connectivity
3. Generate wrapper and build installers for target platforms
4. Distribute installers and ingest telemetry for feedback loops

## 🎨 UX & Branding

**Visual style**: Dark-themed technical interface inspired by VS Code and modern desktop development tools. Professional typography, smooth transitions, and desktop-application layout patterns.

**Tone**: Professional and capable — users should feel confident they are producing production-ready desktop software.

**Accessibility**: Generated desktop apps inherit native OS accessibility features (screen reader support, keyboard navigation, high-contrast mode). The scenario UI follows WCAG 2.1 AA guidelines.

**Brand alignment**: Professional development tooling aesthetic; seamless integration with Vrooli's scenario development and deployment workflow.

## 📎 Appendix

For detailed technical specifications relocated from this PRD, see:
- [API Contract & Data Models](docs/reference/api-contract.md) — endpoint schemas, data models, and event interfaces
- [CLI Interface Contract](docs/reference/cli-commands.md) — command structure, flags, and usage
- [Architecture](docs/concepts/ARCHITECTURE.md) — system design and template architecture
- [API Architecture](docs/reference/api-architecture.md) — handler structure, pipeline system, and tool execution
- [Known Issues](docs/internal/PROBLEMS.md) — limitations, security considerations, and tech debt
- [Implementation Progress](docs/internal/PROGRESS.md) — session-by-session development history
