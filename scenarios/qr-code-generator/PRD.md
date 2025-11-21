# Product Requirements Document (PRD)

> **Scenario**: qr-code-generator
> **Template Version**: 2.0.0
> **Status**: Active
> **Last Updated**: 2025-11-19

## 🎯 Overview

**Purpose**: Provides instant, local QR code generation and batch processing with customization options. This scenario adds the fundamental capability to convert any text, URL, or data into scannable QR codes without relying on external services or third-party APIs.

**Primary Users**:
- Small businesses needing QR codes for marketing campaigns
- Event organizers creating tickets and check-in codes
- Developers requiring quick QR generation for applications
- Marketing teams running batch campaigns

**Deployment Surfaces**:
- **API**: REST endpoints for programmatic QR generation
- **UI**: Web interface with retro gaming aesthetic
- **CLI**: Command-line tool for scripting and automation

**Intelligence Amplification**: Agents can now autonomously create QR codes for sharing data, creating event materials, generating product tags, or embedding information in physical spaces. This compounds with document generation, marketing automation, and event management scenarios to enable complete end-to-end workflows.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Health check endpoint | API responds with service status and feature availability
- [x] OT-P0-002 | Basic QR generation | Generate QR code from text input via POST /generate
- [x] OT-P0-003 | Web interface | Retro-themed UI accessible for manual QR generation
- [x] OT-P0-004 | CLI tool | Command-line interface for QR generation with automatic API discovery
- [x] OT-P0-005 | Batch processing | Generate multiple QR codes efficiently via POST /batch
- [x] OT-P0-006 | Customization options | Support size, color, background, and error correction parameters
- [x] OT-P0-007 | Lifecycle integration | Proper management via Vrooli system (make start/stop/test)

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | QR art generation | Artistic QR codes with custom designs via n8n workflows
- [ ] OT-P1-002 | QR puzzle generator | Interactive QR puzzles for engagement
- [ ] OT-P1-003 | Redis caching | Cache generated QR codes to avoid regeneration
- [ ] OT-P1-004 | Export formats | Multiple output formats (PNG, SVG, PDF, JPEG)

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Analytics tracking | Track QR code generation statistics and usage
- [ ] OT-P2-002 | URL shortening | Integrate URL shortening before QR encoding
- [ ] OT-P2-003 | Logo embedding | Add custom logos to QR code center
- [ ] OT-P2-004 | Dynamic QR codes | QR codes with editable destinations

## 🧱 Tech Direction Snapshot

**API Stack**: Go 1.21+ with HTTP server, dynamic port allocation, structured logging
**UI Stack**: Node.js/Express server with retro gaming theme (VT323 font, neon aesthetics)
**CLI Stack**: Bash wrapper with automatic API port discovery via lsof
**QR Library**: github.com/skip2/go-qrcode for proven, mature QR code generation

**Data Storage**: No persistent storage required for P0; Redis optional for caching (P1)
**Integration Strategy**: REST API for programmatic access, CLI for scripting, UI for manual use

**Non-Goals**:
- External API dependencies for QR generation
- User authentication or access control (public service)
- Cloud-based generation (all processing local)
- Real-time analytics (deferred to P2)

## 🤝 Dependencies & Launch Plan

**Required Resources**: None - standalone scenario with optional resources

**Optional Resources**:
- **n8n**: Advanced workflow automation for art generation and puzzle creation (P1/P2)
- **Redis**: Caching layer to avoid regenerating identical QR codes (P1)

**Scenario Dependencies**: None - foundational capability that other scenarios can consume

**Launch Risks**:
- QR library limitations: Mitigated by using proven go-qrcode library
- Performance at scale: Mitigated by parallel batch processing
- Resource conflicts: Mitigated by optional resource pattern

**Launch Sequencing**: Ready for immediate deployment; no upstream dependencies

## 🎨 UX & Branding

**Visual Style**:
- **Color Scheme**: Dark background with neon green accents (80s arcade aesthetic)
- **Typography**: Retro monospace (VT323 font family)
- **Layout**: Single-page arcade cabinet style with animated elements
- **Animations**: Subtle arcade-style effects, scanline overlays

**Personality & Tone**:
- Fun and nostalgic, energetic retro gaming vibe
- Target feeling: Nostalgic joy mixed with utility
- Error messages maintain playful tone while being informative

**Accessibility**:
- High contrast for visibility (neon green on black)
- Desktop-first, mobile-compatible responsive design
- Clear visual feedback for all interactions
- WCAG AA color contrast compliance for critical elements

**User Experience Promise**: Quick, privacy-focused QR generation that feels fun and engaging while delivering professional results
