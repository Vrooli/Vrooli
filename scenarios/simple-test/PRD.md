# Product Requirements Document (PRD)

> **Version**: 1.0.0
> **Last Updated**: 2025-11-18
> **Status**: Canonical Specification
> **Source of Truth**: Lifecycle Validation Framework

## 🎯 Overview

**Simple Test** is a minimal Node.js application designed to validate Vrooli's lifecycle management system. It serves as a reference implementation and testing harness for the core scenario lifecycle infrastructure.

**Purpose**: Provide a lightweight, predictable application that exercises all lifecycle phases (setup, develop, test, stop) to validate the framework works correctly before deploying complex scenarios.

**Primary users**: Vrooli framework developers, scenario authors, and CI/CD systems validating lifecycle infrastructure.

**Deployment surfaces**: CLI, API health endpoint, automated testing framework.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Lifecycle compliance | Execute all lifecycle phases without errors (setup, develop, test, stop)
- [x] OT-P0-002 | Health endpoint | Respond to /health with proper status code and JSON
- [x] OT-P0-003 | Process management | Start/stop cleanly via lifecycle system
- [x] OT-P0-004 | Test coverage | Achieve ≥80% code coverage with comprehensive test suite
- [x] OT-P0-005 | Database integration | Connect to PostgreSQL and validate schema

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Phased testing | Implement dependency, structure, unit, integration, and performance test phases
- [x] OT-P1-002 | Makefile integration | Support make start/test/logs/stop commands
- [ ] OT-P1-003 | CLI commands | Provide simple-test CLI with status and help commands

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Performance benchmarks | Establish baseline performance metrics for lifecycle operations
- [ ] OT-P2-002 | Extended validation | Add validation for edge cases and error recovery scenarios
- [ ] OT-P2-003 | Documentation | Comprehensive guide for using simple-test as a template

## 🧱 Tech Direction Snapshot

- **API Stack**: Go-based API server with Gin framework
- **UI Server**: Node.js/Express serving static files
- **Data Storage**: PostgreSQL for test data persistence
- **Testing**: Jest for unit tests, Bash scripts for phased integration testing
- **Lifecycle**: v2.0 service.json configuration with health checks
- **Non-goals**: Production deployment, complex features, real user functionality

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- PostgreSQL: Test data storage and lifecycle validation
- Qdrant: Auto-detected for embeddings (optional)

**Scenario Dependencies**: None (intentionally minimal)

**Launch Plan**:
1. Validate lifecycle system works correctly
2. Serve as template for new scenario scaffolding
3. Use as reference in CI/CD pipeline health checks
4. Maintain as gold standard for minimal scenario structure

**Risks**:
- Over-engineering: Must remain minimal to serve its purpose
- Drift from lifecycle changes: Requires updates when framework changes
- Test brittleness: Keep tests focused on essential behavior only

## 🎨 UX & Branding

**Visual palette**: Minimal, functional, developer-focused
**Accessibility**: Basic keyboard navigation, clear error messages
**Voice/personality**: Straightforward, technical, documentation-oriented
**Target feeling**: "Confident that the lifecycle system works correctly"

**Design principles**:
- Simplicity over features
- Clarity over elegance
- Predictability over flexibility
- Documentation through code

## 📎 Appendix

### Related Documentation
- TEST_IMPLEMENTATION_SUMMARY.md: Comprehensive test coverage report
- .vrooli/service.json: Lifecycle configuration
- test/run-tests.sh: Phased test orchestration

### Reference Status
This scenario achieved 93.75% code coverage with 42 passing tests across 4 test suites, serving as the reference implementation for Vrooli's testing infrastructure.
