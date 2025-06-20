# Vrooli Testing Documentation

Welcome to the central hub for all documentation related to testing the Vrooli platform. Effective testing is crucial to ensure the quality, reliability, and performance of our application. This documentation provides guidelines, strategies, and procedures for various testing activities.

## Overview

Our testing strategy emphasizes a comprehensive approach, incorporating various types of testing at different stages of the development lifecycle. We leverage a combination of automated and manual testing techniques to achieve broad coverage and identify issues early.

**⚠️ Important**: Tests can take 15+ minutes to run in worst-case scenarios. Always use extended timeouts for test commands.

## Unified Fixture Architecture

Vrooli uses a **Unified Fixture Architecture** that ensures type-safe, consistent testing across all layers of the application. The architecture follows a factory chain pattern where each factory connects exactly two layers:

```
Form → Shape → API → Validation → Endpoint → Database → Response → UI
  ↑      ↑      ↑        ↑          ↑          ↑          ↑        ↑
Form   Shape   API    Validation  Endpoint  Database  Response   UI
Factory Factory Factory  Factory    Factory   Factory   Factory  Factory
```

**Key Principles**:
- 🔒 **Type Safety**: No `any` types - full TypeScript support throughout
- 🔗 **Layer Connection**: Each factory connects exactly 2 adjacent layers
- 🎯 **Real Functions**: Use actual application functions, not mocks
- 🔄 **Round-Trip Testing**: Validate data integrity from UI to database and back
- 📦 **Composable**: Factories can be combined to test any subset of the flow

For detailed information, see [Fixtures Overview](fixtures-overview.md).

## Quick Start

New to testing in Vrooli? Start here:
1. **[Fixtures Overview](fixtures-overview.md)** - Quick reference for using test fixtures
2. **[Writing Tests](writing-tests.md)** - Guidelines for writing effective tests  
3. **[Test Execution](test-execution.md)** - How to run tests locally

## Key Documentation Sections

### Core Testing Documentation
- **[Test Strategy](./test-strategy.md):** Outlines our high-level approach to testing, including the types of tests performed, tools and technologies used, and overall testing philosophy.
- **[Test Plan](./test-plan.md):** Describes the scope of testing, specific objectives, resources, schedule, and entry/exit criteria for testing cycles.
- **[Test Execution](./test-execution.md):** Provides practical instructions on how to run tests, set up testing environments (including mobile testing), and utilize linting for code quality.
- **[Writing Tests](./writing-tests.md):** Offers guidelines and best practices for writing effective unit, integration, and other types of tests using our standard testing stack (Vitest).
- **[Defect Reporting](./defect-reporting.md):** Explains the process for reporting, tracking, and managing defects discovered during testing.

### Fixture System Documentation
- **[Fixtures Overview](fixtures-overview.md):** Comprehensive guide to the unified fixture architecture - **START HERE!**
- **[Fixture Patterns](fixture-patterns.md):** Factory patterns and best practices for type-safe fixtures
- **[Round-Trip Testing](round-trip-testing.md):** True end-to-end testing from UI to database and back
- **[Fixture Implementation Guide](fixture-implementation-guide.md):** Step-by-step guide for implementing factory patterns
- **[Fixture Reference](fixture-reference.md):** Complete API reference for all fixture types and factories

### Fixture Layer Documentation
Each fixture layer has its own detailed README with architecture, patterns, and migration guides:
- **[API Fixtures](/packages/shared/src/__test/fixtures/api/README.md)** - Validation and transformation layer
- **[Database Fixtures](/packages/server/src/__test/fixtures/db/README.md)** - Persistence and seeding layer
- **[UI Fixtures](/packages/ui/src/__test/fixtures/README.md)** - Frontend integration and round-trip orchestration
- **[Config Fixtures](/packages/shared/src/__test/fixtures/config/README.md)** - Configuration foundation layer
- **[Error Fixtures](/packages/shared/src/__test/fixtures/errors/README.md)** - Error scenario testing
- **[Event Fixtures](/packages/shared/src/__test/fixtures/events/README.md)** - Real-time event simulation
- **[Permission Fixtures](/packages/server/src/__test/fixtures/permissions/README.md)** - Auth and authorization testing
- **[Execution Fixtures](/packages/server/src/__test/fixtures/execution/README.md)** - AI system testing

> **Note**: Each layer follows the unified factory pattern for consistency and type safety.

## Current Test Coverage

- **Shared Package**: 92.73% coverage ✅
- **Target**: 80%+ coverage across all packages
- **Focus Areas**: Permission testing, AI tier testing, round-trip validation

## Quick Commands

```bash
# Run all tests (needs 15+ min timeout)
pnpm test

# Run tests for specific package
cd packages/server && pnpm test

# Run tests with coverage
cd packages/server && pnpm test-coverage

# Run tests in watch mode
cd packages/server && pnpm test-watch
```

## Key Testing Stack

- **Test Framework**: Vitest
- **Infrastructure**: Testcontainers (PostgreSQL, Redis)
- **API Mocking**: Mock Service Worker (MSW)
- **Coverage**: Vitest coverage with c8
- **E2E**: Cypress (pending implementation)

## Contribution

All team members are encouraged to contribute to and maintain this documentation. If you identify gaps, outdated information, or areas for improvement, please update the relevant documents or propose changes.

### Documentation Maintenance

All testing documentation uses the AI maintenance tracking system:
- Before updating: Check for `AI_CHECK` tags
- After updating: Add/update `// AI_CHECK: TASK_ID=count | LAST: YYYY-MM-DD`
- See [AI Maintenance Guide](/docs/ai-maintenance/README.md) for details 