# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Canonical Specification
> **Template**: PRD Control Tower v2.0

## 🎯 Overview

Automated accessibility compliance testing, remediation, and monitoring that ensures every Vrooli scenario meets WCAG 2.1 AA standards. This scenario acts as a guardian that makes all generated apps accessible to users with disabilities, providing both automated fixes and intelligent guidance for complex accessibility issues.

**Purpose**: Adds permanent capability for automated WCAG compliance scanning, auto-remediation, and compliance reporting across all Vrooli UI scenarios.

**Primary Users**: Developers, compliance officers, QA teams building or maintaining Vrooli scenarios with web UIs.

**Deployment Surfaces**: CLI (audit/fix/report commands), API (audit endpoints), UI (compliance dashboard), N8n workflows (scheduled audits).

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | WCAG 2.1 AA compliance scanning | Automated scanning engine for all scenario UIs
- [ ] OT-P0-002 | API endpoints for on-demand audits | REST API accepting scenario ID and WCAG level
- [ ] OT-P0-003 | Auto-remediation for common issues | Automatic fixes for contrast, alt text, ARIA labels
- [ ] OT-P0-004 | Browserless visual regression testing | Integration with Browserless for DOM analysis
- [ ] OT-P0-005 | Compliance dashboard | Dashboard showing all scenarios' accessibility scores
- [ ] OT-P0-006 | CLI audit commands | CLI commands for audit, fix, and report generation

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Machine learning for pattern recognition | ML model training on successful fixes
- [ ] OT-P1-002 | Accessible component library | Reusable library of accessible UI patterns
- [ ] OT-P1-003 | Scheduled audit workflows | N8n workflows for automated periodic audits
- [ ] OT-P1-004 | VPAT documentation generator | Automated VPAT/compliance report generation
- [ ] OT-P1-005 | Git hook integration | Pre-deployment validation via git hooks
- [ ] OT-P1-006 | Real-time monitoring | Continuous accessibility monitoring for live scenarios

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Screen reader testing simulation | Automated screen reader compatibility testing
- [ ] OT-P2-002 | Keyboard navigation flow analysis | Automated keyboard nav path validation
- [ ] OT-P2-003 | Custom Vrooli accessibility rules | Custom rules specific to Vrooli UI patterns
- [ ] OT-P2-004 | Multi-language accessibility support | Accessibility checks for internationalized content
- [ ] OT-P2-005 | Voice navigation testing | Voice control compatibility validation

## 🧱 Tech Direction Snapshot

**Preferred Stack**:
- Go API server with axe-core integration for scanning
- React UI with accessible-by-default components
- PostgreSQL for audit history and learned patterns
- Browserless for headless browser automation

**Data Storage**:
- PostgreSQL for audit reports, violations, remediation patterns
- Optional Qdrant for pattern vector storage
- Optional Redis for caching audit results

**Integration Strategy**:
- Shared automation processes orchestrated by the API/CLI
- Resource CLI commands for Browserless, Ollama, PostgreSQL
- Direct axe-core library integration for scanning

**Non-goals**:
- Not a full testing framework (focused on accessibility only)
- Not replacing manual accessibility review (complements it)
- Not enforcing design decisions (flags issues, doesn't dictate solutions)

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- postgres: Store audit history and compliance reports
- browserless: Visual regression testing and DOM analysis
- ollama: Intelligent fix suggestions for complex issues

Audit sequencing and remediation orchestration are handled inside the API, so no separate workflow resource is needed.

**Optional Resources**:
- redis: Cache audit results for performance
- qdrant: Vector storage for pattern matching

**Risks**:
- False positives in automated audits may require manual review option
- Performance impact on scenarios during audits requires async processing
- Breaking UI during auto-fix requires snapshot/rollback capability

**Launch Sequencing**:
1. Core scanning engine with manual fix suggestions
2. Auto-remediation for common patterns
3. Dashboard and reporting
4. ML-based pattern learning

## 🎨 UX & Branding

**Visual Palette**: Professional dashboard inspired by GitHub Actions and Lighthouse reports, using high-contrast design (dogfooding accessibility).

**Typography**: Modern, highly readable system fonts with WCAG AAA contrast.

**Accessibility Commitments**: WCAG AAA compliant internally (leading by example), keyboard-first navigation, screen reader optimized, respects user preferences (motion, color, contrast).

**Voice/Personality**: Helpful and educational tone. Confident and supportive mood. Target feeling: "I'm protected and guided."

**Motion Language**: Subtle animations respecting prefers-reduced-motion, progress indicators for scans.

## 📎 Appendix

**References**:
- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [Axe-core Documentation](https://github.com/dequelabs/axe-core)
- [Section 508 Standards](https://www.section508.gov/)
