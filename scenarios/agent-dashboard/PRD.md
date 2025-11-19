# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: P0 Complete ✅
> **Template**: PRD Control Tower v2.0

## 🎯 Overview

Real-time monitoring and basic control of all AI agents running within Vrooli resource containers. Provides a centralized cyberpunk-themed dashboard for developers to observe agent health, view logs, and perform basic agent lifecycle operations without touching the underlying resources or scenarios.

**Purpose**: Adds permanent capability for real-time agent discovery, monitoring, and control across all Vrooli resources with agent support.

**Primary Users**: Vrooli developers doing day-to-day development and debugging.

**Deployment Surfaces**: CLI (list/stop/logs commands), API (agent discovery/control endpoints), UI (cyberpunk dashboard with radar visualization).

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Discover all agents across resources | Enumerate agents across all installed Vrooli resources
- [x] OT-P0-002 | Display real-time agent status | Show active/inactive/error states with auto-refresh
- [x] OT-P0-003 | Show agent metadata | Display PID, uptime, command, capabilities
- [x] OT-P0-004 | Basic agent control | Stop agents via resource CLI integration
- [x] OT-P0-005 | Real-time log viewer | Log viewer with follow capability and streaming
- [x] OT-P0-006 | Auto-refresh polling | Poll every 30 seconds with rate limiting
- [x] OT-P0-007 | Cyberpunk-themed UI | Cyberpunk theme with scan animations
- [x] OT-P0-008 | Radar view with agent visualization | Radar with moving dots and color-coded health

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Agent cards with metrics | Display memory, CPU metrics when available
- [x] OT-P1-002 | Radar hover tooltips | Tooltips showing agent details on radar hover
- [x] OT-P1-003 | Agent search and filtering | Filter by name, type, or status
- [x] OT-P1-004 | Log export functionality | Download/export log files
- [x] OT-P1-005 | Keyboard shortcuts | Shortcuts for common operations with help modal
- [x] OT-P1-006 | Responsive design | Works on different screen sizes

### 🟢 P2 – Future / expansion

- [x] OT-P2-001 | Agent performance history | Session-only performance graphs with sparklines
- [x] OT-P2-002 | Multiple log viewers | Open multiple log viewers simultaneously
- [x] OT-P2-003 | Agent capability discovery | Search agents by capability
- [x] OT-P2-004 | Custom radar themes | Multiple theme options with localStorage persistence
- [x] OT-P2-005 | Agent grouping by resource | Group agents by resource type

## 🧱 Tech Direction Snapshot

**Preferred Stack**:
- Go API server for agent discovery and control
- Vanilla JavaScript frontend with cyberpunk theme
- No build step (direct browser execution)
- WebSocket optional for real-time updates (currently polling)

**Data Storage**:
- Browser memory only (session-only data)
- No persistent storage needed
- No database dependencies

**Integration Strategy**:
- Resource CLI abstraction layer (`resource-{name} agents list --json`)
- No direct resource access (uses CLI commands)
- Graceful degradation for non-standard resources

**Non-goals**:
- Not a production monitoring platform (dev tool only)
- Not storing historical data (session-only)
- Not replacing resource-specific tools (complements them)

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- None (purely monitors existing resources)

**Optional Resources**:
- Any Vrooli resource with agent support (ollama, claude-code, cline, autogen-studio, crewai, gemini, langchain, litellm, openrouter, whisper, comfyui, pandas-ai, parlant, huginn, opencode, autogpt)

**Risks**:
- Resource CLI changes may break discovery (mitigate with graceful error handling)
- Browser memory leaks with long sessions (mitigate with data cleanup)
- UI performance with many agents (mitigate with virtual scrolling)

**Launch Sequencing**:
1. Core agent discovery and status display
2. Agent control (stop operations) and log viewing
3. Cyberpunk UI with radar visualization
4. CLI interface for scripting

## 🎨 UX & Branding

**Visual Palette**: Cyberpunk dark theme inspired by Matrix command centers and GitHub dark mode. Primary colors: cyan/magenta/yellow on dark background.

**Typography**: Orbitron headers, Exo 2 body text, Share Tech Mono for code/logs.

**Accessibility Commitments**: Keyboard navigation, high contrast mode available, WCAG AA compliant.

**Voice/Personality**: Playful but functional. High-tech, futuristic, engaging. Target feeling: "Command center for my AI empire."

**Motion Language**: Scanning lines, data streams, radar sweep animations. Smooth position transitions with ease-in-out. Respects prefers-reduced-motion.

## 📎 Appendix

**References**:
- [Vrooli Resource CLI Standards](../../docs/resources.md)
- [Cyberpunk Design Patterns](https://cyberpunk.design)
- [LCARS Interface Guidelines](https://lcars.org)
