# Product Requirements Document (PRD)

## 🎯 Overview
**Purpose**: Create a frictionless thought-capture system that enables deep thinkers to record and organize their ideas without disrupting their cognitive flow.

**Target users**: Deep thinkers, strategists, researchers, and decision-makers who think in graph-structured patterns.

**Deployment surfaces**: Mobile-first PWA, desktop web interface, REST API, CLI integration

**Value proposition**: Eliminates the tradeoff between deep thinking and accurate capture by combining zero-friction input with LLM-powered organization.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Zero-Friction Capture | Users can begin recording thoughts within 1 second of app launch via voice, text, or media
- [x] OT-P0-002 | Dual-View System | Support both freeform spatial canvas and structured thought graph views with seamless switching

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Intelligent Suggestions | LLM generates contextual thought connections as dismissible ghost nodes
- [x] OT-P1-002 | Cross-Scenario Export | One-tap export of structured thought graphs to other Vrooli scenarios

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Provider Resilience | Automatic fallback to OpenRouter when local Ollama is unavailable
- [x] OT-P2-002 | Cross-Scheme Navigation | Enable seamless navigation between related thought schemes with visual distinction

## 🧱 Tech Direction Snapshot
**Preferred stacks**: Go (API/CLI), React (UI), WebSocket (voice streaming)
**Preferred storage**: PostgreSQL (primary), Redis (cache), IndexedDB (offline)
**Integration strategy**: Agent-manager for chat, Whisper for voice, Ollama for LLM
**Non-goals**: Real-time collaboration, external sharing, complex permissions

## 🤝 Dependencies & Launch Plan
**Required resources**: postgres, redis, ollama, whisper-stt
**Scenario dependencies**: agent-manager (required), web-console (optional), app-monitor (optional)
**Operational risks**: LLM availability, voice transcription quality, mobile performance with large schemes
**Launch sequencing**: Storage → Basic UI → Canvas → Voice → Graph → Agent → Export → Suggestions → Offline

## 🎨 UX & Branding
**User experience**: Mobile-optimized touch interactions, zero-friction input, visual feedback for sync state
**Visual design**: Clean canvas space, subtle graph visualization, clear information type distinction
**Accessibility**: WCAG 2.1 AA compliance, voice input primary, high-contrast graph visualization

## 📎 Appendix
- Canvas performance benchmarks
- LLM prompt templates
- Storage schema diagrams
- API documentation