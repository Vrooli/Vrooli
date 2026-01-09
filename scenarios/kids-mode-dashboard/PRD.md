# Product Requirements Document (PRD)

> **Template Version**: 2.0.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

**Purpose**: A safe, engaging web dashboard designed specifically for children that provides a controlled, educational browsing environment with age-appropriate content and parental controls.

**Primary users / verticals**:
- Parents and guardians managing children's digital screen time
- Educational institutions providing safe web access for students
- Libraries and public spaces offering kid-safe internet stations

**Deployment surfaces**:
- Web UI (kid-friendly dashboard interface)
- API (configuration and content management)
- CLI (administration and setup)

Kids Mode Dashboard creates a permanent capability for Vrooli to offer child-safe digital environments, enabling future scenarios focused on education, parental controls, and age-appropriate content delivery.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Safe web dashboard | Provide a kid-friendly web interface with large, colorful navigation and safe content
- [ ] OT-P0-002 | Health monitoring | API health endpoints for service status and uptime tracking
- [ ] OT-P0-003 | Basic content filtering | Block access to inappropriate content and adult websites
- [ ] OT-P0-004 | Simple authentication | Parent PIN or password to access settings and exit kid mode
- [ ] OT-P0-005 | Educational content links | Curated collection of age-appropriate educational websites and resources

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Time limits and scheduling | Configure daily screen time limits and allowed usage windows
- [ ] OT-P1-002 | Activity reporting | Generate reports showing what content was accessed and when
- [ ] OT-P1-003 | Age-based profiles | Support multiple child profiles with age-appropriate content filtering
- [ ] OT-P1-004 | Custom content allowlists | Parents can add approved websites to accessible content
- [ ] OT-P1-005 | Search integration | Kid-safe search functionality with strict filtering
- [ ] OT-P1-006 | Rewards and gamification | Achievement badges for learning milestones and educational progress

### 🟢 P2 – Future / expansion ideas
- [ ] OT-P2-001 | Mobile companion app | Parent monitoring and control via mobile application
- [ ] OT-P2-002 | Multi-device sync | Sync profiles, settings, and time limits across multiple devices
- [ ] OT-P2-003 | AI content analysis | Machine learning to automatically detect and filter inappropriate content
- [ ] OT-P2-004 | Educational analytics | Insights into learning patterns and subject preferences
- [ ] OT-P2-005 | Social features | Safe, moderated messaging between approved friends for collaborative learning

## 🧱 Tech Direction Snapshot

- **Preferred stacks**: Go API (lightweight, fast HTTP server), embedded HTML/CSS/JavaScript UI (simple deployment), responsive web design
- **Data storage**: PostgreSQL for user profiles, activity logs, and content allowlists; Redis for session management and temporary locks
- **Integration strategy**: Standalone service that can integrate with browser-automation-studio for automated content safety testing; optional integration with notification-hub for parent alerts
- **Non-goals**: Not a full web browser replacement, not a complete parental control suite for all devices, not a content creation platform

## 🤝 Dependencies & Launch Plan

**Required resources**:
- postgres (user profiles, activity logs, content lists)
- redis (session state, authentication tokens)

**Scenario dependencies**:
- session-authenticator (parent authentication and PIN management)
- notification-hub (optional - parent alerts for time limits or flagged content)

**Operational risks**:
- Content filtering effectiveness depends on maintaining up-to-date blocklists
- Browser compatibility issues with embedded UI approach
- Performance impact if activity logging becomes too verbose

**Launch sequencing**:
1. Phase 1 (Week 1-2): Core dashboard and basic content filtering deployed to localhost
2. Phase 2 (Week 3-4): Add authentication, time limits, and activity reporting
3. Phase 3 (Week 5+): Enhanced features like profiles, custom allowlists, and rewards

## 🎨 UX & Branding

**Visual style**: Bright, playful, child-friendly design with large buttons and icons. Colorful palette with primary colors (red, blue, yellow, green). Large readable fonts (16px+ for body text). Animated transitions and friendly illustrations to create an engaging, non-threatening interface.

**Accessibility commitments**:
- WCAG 2.1 AA compliance minimum
- High contrast mode for visually impaired children
- Keyboard navigation for all controls
- Screen reader support for blind and low-vision users
- Simple language at 3rd-grade reading level or below

**Voice & personality**: Friendly, encouraging, patient, and supportive. Uses simple language, positive reinforcement, and celebrates learning. Avoids technical jargon and complex instructions. Think "helpful learning companion" rather than "strict authority figure."

**Brand positioning**: The safe, fun gateway to educational digital content for children. Empowers parents with peace of mind while giving kids freedom to explore and learn in a protected environment.
