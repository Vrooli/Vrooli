# Product Requirements Document (PRD)

> **Scenario**: web-console
> **Version**: 3.0.0
> **Last Updated**: 2026-02-13
> **Status**: Rewrite Planned
> **Template**: Canonical PRD v2.0.0

## 🎯 Overview

**Purpose**: Web Console delivers a full-fidelity terminal experience in the browser with pane-based workflows, durable sessions, and AI-assisted input generation for authenticated parent scenarios.

**Primary Users**:
- Operators and engineers running interactive CLI workflows (Claude Code, Codex, diagnostics)
- Parent scenarios embedding terminal capability via iframe behind authenticated proxying
- Mobile operators needing practical terminal controls without SSH clients

**Deployment Surfaces**:
- Go API for PTY session lifecycle, streaming, persistence, and policy controls
- Browser UI (Vite + xterm.js) with pane layout, launcher, floating keyboard toolbar, and drawer controls
- iframe bridge over `postMessage` for parent coordination and telemetry
- CLI + Makefile lifecycle commands for local development and testing

**Value Promise**: Zero-compromise browser terminal operations with resilient reconnect behavior, better multiterminal visibility, and faster command execution through configurable shortcuts and AI assist.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Pane-Based Terminal Workspace | Multi-pane terminal layout with simultaneous terminals, 2-column desktop and 1-column mobile behavior aligned with app-monitor workspace ergonomics
- [ ] OT-P0-002 | Production-Grade Web Terminal Fidelity | Browser terminal supports interactive CLIs with PTY resize, cursor reporting, binary-safe I/O, and reconnect-safe input handling
- [ ] OT-P0-003 | Durable Session Continuity | Default session expiration is never; reconnect restores live state, transcript history, and output generated while client was offline
- [ ] OT-P0-004 | Proxy-Correct Networking via api-base | UI uses shared api-base for HTTP/WebSocket routing under parent proxying with no direct-origin assumptions
- [ ] OT-P0-005 | AI Input with Provider Fallback | AI command generation uses Ollama first with OpenRouter fallback and deterministic failover behavior
- [ ] OT-P0-006 | New Terminal Launcher with Configurable Shortcuts | New-terminal flow offers empty shell and configurable shortcut entries; default entries include `claude --dangerously-skip-permissions` and `codex --yolo`
- [ ] OT-P0-007 | Mobile Terminal Usability Toolbar | Floating keyboard toolbar provides required terminal keys/chords for practical mobile usage
- [ ] OT-P0-008 | Sidebar/Drawer Controls Surface | Drawer exposes session/workspace status and core controls without blocking primary terminal workflow

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Session Policy Controls | Per-workspace/session expiration policy controls (never, preset TTL, custom) with explicit persistence behavior
- [ ] OT-P1-002 | Shortcut Profile Management | Shortcut catalog is configuration-driven (service/workspace/parent context) rather than hardcoded UI constants
- [ ] OT-P1-003 | AI Provider Policy Controls | Provider priority, timeout, and fallback policy are configurable with surfaced provider health
- [ ] OT-P1-004 | Operational Observability Coverage | Metrics and structured events cover lifecycle, reconnects, pane actions, and AI provider failovers

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Collaborative Session Modes | Optional observer/shared-view terminal modes for paired operations and escalations
- [ ] OT-P2-002 | Persisted Workspace Presets | Save/load named pane + shortcut presets for repeat workflows

## 🧱 Tech Direction Snapshot

- Preferred stacks/frameworks: Go 1.21+ API, Vite UI, xterm.js, shared `@vrooli/api-base` networking helpers
- Data + storage expectations: Durable local transcript/session persistence by default with provider adapters as extension points
- Integration strategy: Authenticated parent embedding + WebSocket streaming + `postMessage` bridge contract
- Non-goals: Built-in public auth surface, direct internet exposure, and replacing parent-level RBAC policy

## 🤝 Dependencies & Launch Plan

- Required resources: Go toolchain, POSIX runtime, `resource-ollama`, authenticated proxy headers, scenario-managed storage
- Optional resources: `resource-openrouter` fallback provider, Redis/Postgres durability adapters, parent callback integrations
- Operational risks: Session loss regressions, interactive CLI incompatibilities, provider outages, mobile input friction
- Launch sequencing: Ship P0 fidelity and durability first, then P1 policy/configuration and observability hardening, then P2 collaboration/presets

## 🎨 UX & Branding

- Look & feel: Utility-first terminal workspace with pane density and explicit state signaling
- Accessibility: WCAG 2.1 AA target, keyboard-only support, readable contrast, labeled controls for toolbar/drawer/launcher
- Voice & messaging: Operational and direct; prioritize clarity around session state, provider fallbacks, and proxy boundaries
- Branding hooks: Keep critical UI dependencies locally available to preserve offline and restricted-network reliability

## 📎 Appendix

- This PRD defines rewrite targets and intentionally supersedes legacy implementation scope. Detailed implementation notes belong in `README.md` and `PROBLEMS.md`.
