# Product Requirements Document (PRD)

> **Template Version**: 2.0.0
> **Scenario**: typing-test
> **Last Updated**: 2025-11-19

## 🎯 Overview

**Purpose**: Interactive web application for testing and improving typing speed and accuracy. Helps users develop keyboard proficiency essential for productivity in digital work environments.

**Primary Users**: Students, professionals, educators, and anyone seeking to improve typing skills for personal or career development.

**Deployment Surfaces**:
- Web UI for interactive typing tests
- Go API for result processing and persistence
- CLI for quick terminal-based typing tests

**Value Proposition**: Provides real-time feedback on typing speed (WPM) and accuracy with progress tracking, difficulty levels, and AI-powered coaching suggestions. Targets educational institutions and corporate training markets.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Core typing test interface | Real-time typing interface with text prompt and input field
- [x] OT-P0-002 | WPM calculation | Accurate words-per-minute calculation using standard formula
- [x] OT-P0-003 | Accuracy tracking | Real-time error detection and accuracy percentage display
- [x] OT-P0-004 | Health check endpoint | API responds to /health endpoint with proper status
- [x] OT-P0-005 | Lifecycle management | setup/develop/test/stop commands work properly

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Progress tracking | Store and display typing history with PostgreSQL
- [x] OT-P1-002 | Difficulty levels | Multiple text difficulty options (easy/medium/hard)
- [x] OT-P1-003 | Leaderboard | Global and personal best scores display
- [ ] OT-P1-004 | AI coaching | Personalized tips using Ollama integration

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Custom texts | Allow users to practice with their own text
- [ ] OT-P2-002 | Export results | Download typing history as CSV/PDF
- [ ] OT-P2-003 | Keyboard heatmap | Visual representation of typing patterns

## 🧱 Tech Direction Snapshot

- **Frontend Stack**: Vanilla JavaScript with real-time input capture for minimal latency; Node.js for dev server
- **Backend Stack**: Go with Gin framework for API endpoints and session management
- **Data Storage**: PostgreSQL for user sessions, typing history, and leaderboard persistence
- **AI Integration**: Ollama for personalized coaching suggestions based on typing patterns
- **CLI Tool**: Standalone Go binary for terminal-based typing tests

**Non-Goals**:
- Mobile app development (web-responsive only)
- Multi-language keyboard layouts in v1
- Gaming/competitive features beyond basic leaderboards

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- PostgreSQL (user sessions, history storage)
- Node.js runtime (UI development server)
- Go 1.21+ (API and CLI compilation)

**Optional Resources**:
- Ollama (AI coaching features, graceful degradation if unavailable)

**Launch Sequencing**:
1. Core typing test with WPM/accuracy (P0)
2. Progress tracking and difficulty levels (P1)
3. Leaderboard integration
4. AI coaching rollout (dependent on Ollama availability)

**Known Risks**:
- Input latency on slower devices may affect user experience
- AI coaching quality depends on Ollama model performance
- Competitive market requires differentiation through AI features

## 🎨 UX & Branding

**Visual Palette**: Clean, minimalist design with focus on readability. High-contrast text for reduced eye strain during extended typing sessions. Monospace font for typing area to maintain consistent character width.

**Accessibility Commitments**:
- WCAG 2.1 AA compliance
- Keyboard-only navigation
- Screen reader support for progress announcements
- Clear error indicators without color-only reliance

**Voice & Personality**: Encouraging and supportive. Celebrates improvements without being patronizing. Error feedback is constructive rather than critical. AI coaching uses friendly, mentor-like tone.

**Interaction Design**: Immediate visual feedback on keypresses, smooth transitions between difficulty levels, non-intrusive error highlighting that doesn't break typing flow.

## 📎 Appendix

**Revenue Model**: Educational market ($50/month per school × 20 schools = $1K/month) + corporate training ($200/month per company × 5 companies = $1K/month) + individual premium ($5/month × 100 users = $500/month). Total: ~$30K annual revenue potential.

**Market Positioning**: Differentiate through AI-powered coaching and seamless Vrooli ecosystem integration. Position as professional development tool rather than gaming platform.

**Progress History**:
- 2025-09-28: 20% → 85% (Fixed dependencies, verified all P0 requirements working, P1 mostly complete)
