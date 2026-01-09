# Requirements Structure

This directory contains the requirements for the visited-tracker scenario, organized into modules that align with operational targets from the PRD.

## Module Organization

### 01-campaign-tracking (P0)
**Operational Target**: OT-P0-001 - Campaign-Based File Tracking

Core functionality for creating campaigns, tracking file visits, calculating staleness scores, and providing CLI/API interfaces.

**Requirements**:
- VT-REQ-001: Campaign-Based File Tracking
- VT-REQ-002: Visit Count Tracking
- VT-REQ-003: Staleness Scoring Algorithm
- VT-REQ-004: CLI Interface
- VT-REQ-005: JSON File Persistence

**Validation**: Unit tests, API tests, CLI BATS tests

### 02-web-interface (P1)
**Operational Target**: OT-P1-001 - Web Interface & API

HTTP API and React-based web interface for manual campaign management, file synchronization, and prioritization.

**Requirements**:
- VT-REQ-006: HTTP API
- VT-REQ-007: Web Interface
- VT-REQ-008: File Synchronization
- VT-REQ-009: File Prioritization
- VT-REQ-010: Campaign Export/Import

**Validation**: API integration tests, UI workflow tests

### 03-advanced-features (P2)
**Operational Target**: OT-P2-001 - Advanced Features

Future enhancements including analytics, git history integration, multi-project management, and automated discovery.

**Requirements**:
- VT-REQ-011: Advanced Analytics
- VT-REQ-012: Git History Integration
- VT-REQ-013: Multi-Project Management
- VT-REQ-014: Automated File Discovery

**Validation**: TBD (planned for future iterations)

## Testing Strategy

Each module includes validation methods at multiple layers:
- **Unit**: Core logic and algorithms
- **API**: HTTP endpoints and data flow
- **UI**: User workflows and visual components
- **E2E**: Complete user journeys across the system

See `/bas/` for automated workflow tests organized by capability.
