# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Active
> **Template**: Canonical PRD v2.0.0

## 🎯 Overview

App Monitor provides comprehensive real-time monitoring and control for all running scenarios and resources within the Vrooli ecosystem through a distinctive Matrix-themed cyberpunk interface.

**Purpose**: Enable operators to oversee system health, manage application lifecycle, and diagnose issues across the entire Vrooli deployment from a unified dashboard.

**Primary Users**:
- System administrators managing Vrooli deployments
- Developers monitoring scenario health during development
- DevOps teams overseeing production environments

**Deployment Surfaces**:
- Web UI with real-time WebSocket updates
- REST API for programmatic access
- Docker integration for container monitoring
- Cloudflare tunnel for remote access (Tier 1 deployments)

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Real-time scenario status monitoring | Display running/stopped/error states for all scenarios with live updates
- [x] OT-P0-002 | Application lifecycle control | Start, stop, and restart scenarios through UI and API
- [x] OT-P0-003 | Resource health tracking | Monitor status of core resources (PostgreSQL, Redis, Ollama, etc.)
- [x] OT-P0-004 | Performance metrics display | Show CPU, memory, network, and disk usage per scenario
- [x] OT-P0-005 | Live log streaming | Stream real-time logs from running scenarios with filtering
- [x] OT-P0-006 | Matrix cyberpunk UI theme | Deliver distinctive green-on-black aesthetic with terminal fonts and glow effects
- [x] OT-P0-007 | Docker container integration | Monitor and control scenarios via Docker API

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Historical performance analytics | Store and visualize performance trends over time
- [ ] OT-P1-002 | Automated health alerts | Notify operators of degraded health or failures
- [ ] OT-P1-003 | Batch scenario operations | Execute start/stop/restart on multiple scenarios simultaneously
- [ ] OT-P1-004 | Interactive terminal interface | Built-in command interface for system operations
- [ ] OT-P1-005 | Custom dashboard layouts | Allow users to customize metric panels and views
- [ ] OT-P1-006 | Advanced log analysis | Search, filter, and export logs with pattern matching

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Multi-server monitoring | Aggregate monitoring across distributed Vrooli installations
- [ ] OT-P2-002 | Mobile-optimized interface | Responsive design with mobile-first interactions
- [ ] OT-P2-003 | Predictive health modeling | ML-based predictions of resource exhaustion or failures
- [ ] OT-P2-004 | Integration with external APM tools | Export metrics to Prometheus, Grafana, Datadog
- [ ] OT-P2-005 | Role-based access control | Limit operations based on user permissions
- [ ] OT-P2-006 | Scenario dependency visualization | Graph view of inter-scenario dependencies

## 🧱 Tech Direction Snapshot

**Frontend Stack**:
- Vanilla JavaScript for minimal bundle size and direct DOM control
- Custom CSS with Matrix cyberpunk theme (green #00ff41, dark backgrounds, monospace fonts)
- WebSocket client for real-time updates
- Canvas API for performance charts
- No framework dependencies – prioritize simplicity and performance

**Backend Stack**:
- Go API server for scenario management and Docker integration
- Node.js proxy server for UI hosting and WebSocket relay
- Express-based routing and API proxying
- Docker SDK for container monitoring

**Data Flow**:
- UI ↔ Node.js server ↔ Go API backend
- WebSocket connections for live updates
- Docker API for container stats
- Mock data fallbacks for development/testing

**Non-Goals**:
- Not replacing dedicated APM tools (Prometheus, Grafana) for production monitoring
- Not building custom container orchestration (use existing Docker/Kubernetes)
- Not supporting Windows-native deployments (Linux/macOS only)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- Docker (container monitoring and control)
- PostgreSQL (optional: for storing historical metrics)
- Redis (optional: for caching scenario states)

**Scenario Dependencies**:
- None – App Monitor is a foundational monitoring tool

**Launch Sequencing**:
1. Core monitoring API with Docker integration (P0 targets)
2. Matrix-themed UI with real-time updates
3. Deploy via Cloudflare tunnel for Tier 1 access
4. Historical analytics and alerting (P1 targets)
5. Multi-server and mobile support (P2 targets)

**Risks**:
- Docker API permissions may require elevated privileges in some environments
- WebSocket connection stability on poor networks
- Performance overhead of monitoring many scenarios simultaneously

## 🎨 UX & Branding

**Visual Identity**:
- **Matrix Cyberpunk Theme**: Green-on-black (#00ff41 on #0a0a0a) with cyan accents (#00ffff)
- **Typography**: Share Tech Mono (monospace) for terminal authenticity, Orbitron for headers
- **Visual Effects**: Animated matrix rain background, glow effects on hover, scan lines in header
- **Color Coding**: Green (running/online), Red (stopped/error), Yellow (warning), Blue (info)

**Interaction Patterns**:
- Immediate status clarity – health visible at a glance
- Hover states with border glow and background inversion
- Smooth 0.3s transitions on all interactive elements
- Real-time updates without jarring page reloads
- Touch-friendly targets for mobile interfaces

**Accessibility**:
- Maintain WCAG AA color contrast despite dark theme
- Keyboard navigation for all controls
- Screen reader support for status indicators
- Reduced motion option to disable animations

**Voice & Personality**:
- Professional technical aesthetic – no unnecessary playfulness
- Direct, concise status messaging
- Technical accuracy over marketing language
- "Hacker command center" tone that enhances focus

**Performance Expectations**:
- Initial page load < 2 seconds
- Real-time update latency < 500ms
- Responsive interactions < 100ms
- Memory usage < 50MB typical

## 📎 Appendix

**Legacy Content Reference**: The original PRD contained detailed UI specifications for components, layout structure, and responsive design patterns. These details now live in `ui/src/styles/` (CSS implementation), `ui/docs/DESIGN_SYSTEM.md` (component specifications), and `docs/PROGRESS.md` (implementation milestones).

**Browser Compatibility**: Requires modern browsers with ES6+ support, WebSocket support, Canvas API for charts, and CSS Grid and Flexbox support.
