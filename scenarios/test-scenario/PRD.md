# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Canonical Specification
> **Source of Truth**: PRD Control Tower (API/UI/CLI)

## 🎯 Overview

**Purpose**: Test scenario for validating Vrooli infrastructure components, particularly the security scanner's ability to detect hardcoded secrets and lifecycle management compliance.

**Primary users**: Vrooli developers and CI/CD systems

**Deployment surfaces**: API, CLI

This scenario intentionally contains security vulnerabilities (hardcoded secrets, credentials) to validate that the security scanning infrastructure correctly identifies and reports them. It serves as a permanent test fixture for the ecosystem's security tooling.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Lifecycle protection enforcement | Binary refuses to run outside Vrooli lifecycle system
- [x] OT-P0-002 | Hardcoded secret detection | Contains intentional hardcoded secrets for scanner validation
- [x] OT-P0-003 | Basic API server | Simple HTTP server for infrastructure testing

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Health endpoint | Standard /health endpoint for lifecycle validation
- [x] OT-P1-002 | CLI installation | Basic CLI for testing CLI installation workflows

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Additional vulnerability types | Expand test coverage to more vulnerability patterns
- [ ] OT-P2-002 | Performance testing fixtures | Add endpoints for load testing infrastructure

## 🧱 Tech Direction Snapshot

**Tech Stack**:
- Go API server with minimal dependencies
- Standard Go testing infrastructure
- PostgreSQL integration for database testing

**Architectural Intent**:
- Keep implementation minimal and focused on testing needs
- Intentionally insecure code for scanner validation
- Lifecycle-aware binary with proper environment checks

**Non-goals**:
- Production deployment (test fixture only)
- Security hardening (intentionally vulnerable)
- Feature richness (simplicity is the feature)

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- PostgreSQL (for database integration testing)

**Launch Sequencing**:
1. Used immediately by development and CI/CD systems
2. Referenced by security scanner validation tests
3. Updated as new infrastructure patterns emerge

**Risks**:
- Must never be deployed to production environments
- Requires clear documentation of intentional vulnerabilities

## 🎨 UX & Branding

**Visual Style**: Command-line focused, minimal output

**Tone**: Technical, diagnostic

**Accessibility**: Not applicable (internal testing tool)

**User Experience Promise**: "Clear validation of infrastructure capabilities through intentional test cases"

## 📎 Appendix

### Intentional Security Vulnerabilities

This scenario contains the following intentional vulnerabilities for scanner validation:
- Hardcoded database passwords
- Hardcoded API keys
- Hardcoded JWT secrets
- Hardcoded database URLs with credentials

These are **intentional test fixtures** and should be detected by security scanning tools.

### Testing Infrastructure

The scenario validates:
- Lifecycle protection enforcement
- Security scanner detection capabilities
- CLI installation workflows
- Health endpoint compliance
- Basic API server patterns
