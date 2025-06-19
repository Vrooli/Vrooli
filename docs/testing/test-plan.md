# Test Plan

This document details the Test Plan for the Vrooli platform. It specifies the scope, objectives, resources, and criteria for testing activities, aligning with the overall [Test Strategy](./test-strategy.md).

## Quick Reference

### Common Test Commands
```bash
# Run all tests (⚠️ 5+ min timeout needed)
pnpm test

# Package-specific tests
cd packages/server && pnpm test          # Server tests (5+ min)
cd packages/ui && pnpm test              # UI tests (3+ min)
cd packages/jobs && pnpm test            # Jobs tests (3+ min)

# Watch mode for development
cd packages/server && pnpm test-watch

# Coverage reports
cd packages/server && pnpm test-coverage # (5+ min)

# Type checking (use specific files when possible)
cd packages/server && tsc --noEmit src/file.ts  # Single file
cd packages/server && pnpm run type-check        # Full package (4+ min)
```

### Critical Testing Rules
1. **Always use `.js` extensions** in TypeScript imports
2. **Never mock Redis/PostgreSQL** - use testcontainers
3. **Use `__test` directories** (not `__tests`)
4. **Wrap database tests** with `withDbTransaction()`
5. **Set extended timeouts** for long-running operations

## 1. Introduction

The purpose of this Test Plan is to provide a clear and comprehensive guide for all testing activities related to the Vrooli project. It ensures that testing is conducted in a systematic and effective manner, contributing to the delivery of a high-quality product.

This plan integrates with the [AI Maintenance System](../ai-maintenance/README.md) for continuous test quality improvement and coverage tracking.

## 2. Scope of Testing

### 2.1. Features to be Tested

Testing will cover all major functional and non-functional aspects of the Vrooli platform, including but not limited to:

-   **Core Platform Functionality:** (As defined in `docs/roadmap.md` Phase 1)
    -   User Authentication and Authorization
    -   AI Chat Functionality (including message storage, context handling)
    -   Routines Implementation (creation, execution, triggers, visualization)
    -   Commenting, Reporting, and Moderation Features
-   **Three-Tier AI System:**
    -   Tier 1: Coordination Intelligence
    -   Tier 2: Process Intelligence
    -   Tier 3: Execution Intelligence
    -   Event Bus: Redis-based communication
-   **UI/UX:**
    -   Navigation and User Flows
    -   Responsive Design and Mobile Experience
    -   Component Library (Storybook integration and component behavior)
    -   MSW (Mock Service Worker) integration for isolated testing
-   **API Endpoints:** All publicly exposed and internal APIs
-   **Integrations:** Interactions with any third-party services or internal microservices
-   **Data Integrity:** Verification of data storage, retrieval, and consistency

### 2.2. Features Not to be Tested (Initially)

-   Advanced AI agent swarm behaviors beyond defined routine execution (emergent capabilities)
-   Specific third-party integrations not yet implemented
-   Exhaustive performance and load testing under extreme conditions (phased approach)

### 2.3. Current Test Coverage Baseline

As of the latest assessment:
- **Server Package**: ~151 test files
- **UI Package**: ~66 test files  
- **Jobs Package**: ~17 test files
- **Shared Package**: Multiple utility tests

Target: **80%+ coverage by Phase 2 (T+12-24 weeks)**

## 3. Testing Objectives

-   **Identify Defects:** Discover and document software defects as early as possible
-   **Verify Requirements:** Ensure that the application meets the specified functional and non-functional requirements
-   **Validate User Experience:** Confirm that the application is intuitive, usable, and meets user expectations
-   **Assess Quality:** Provide an overall assessment of the application's quality and readiness for release
-   **Ensure Stability:** Verify the stability and reliability of the application under normal operating conditions
-   **Maintain Coverage:** Achieve and maintain 80%+ test coverage across all packages
-   **Support AI Development:** Ensure emergent capabilities have proper test infrastructure

## 4. Test Resources

### 4.1. Testing Framework and Tools

-   **Primary Framework:** Vitest (NOT Mocha/Chai)
-   **Test Containers:** For Redis and PostgreSQL (never mock these)
-   **UI Testing:** Vitest + React Testing Library + MSW
-   **E2E Testing:** Cypress (installed, implementation pending)
-   **Coverage:** Vitest coverage with C8
-   **Component Testing:** Storybook for visual testing

### 4.2. Test Environment

-   **Local Development:** Docker-based test containers
-   **CI Environment:** GitHub Actions with containerized services
-   **Test Data:** Fixtures using real application shape functions (see [fixtures-and-data-flow.md](./fixtures-and-data-flow.md))

### 4.3. Test Helpers and Infrastructure

-   **Database Helpers:** `/packages/server/src/__test/helpers/`
-   **Transaction Wrapper:** `withDbTransaction()` for test isolation
-   **Mock Services:** `/packages/jobs/src/__test/mocks/services.ts`
-   **UI Test Setup:** `/packages/ui/src/__test/setup.vitest.ts`

## 5. Test Writing Guidelines

### 5.1. Import Requirements

```typescript
// ✅ CORRECT - Always use .js extension
import { UserModel } from "../models/User.js";
import { uuid } from "@vrooli/shared";  // Package imports don't need extension

// ❌ WRONG - Missing .js extension
import { UserModel } from "../models/User";
import { uuid } from "@vrooli/shared/id/uuid.js";  // Never use deep imports
```

### 5.2. Database Test Pattern

```typescript
import { withDbTransaction } from "../__test/helpers/db.js";

describe("UserModel", () => {
    it("should create user", async () => {
        await withDbTransaction(async (tx) => {
            // Your test code here - transaction auto-rollbacks
            const user = await UserModel.create(tx, userData);
            expect(user).toBeDefined();
        });
    });
});
```

### 5.3. Architectural Patterns

-   **Use Real Functions:** Always use actual shape functions from the codebase
-   **Avoid Mocks:** Prefer integration over unit tests where practical
-   **Round-Trip Testing:** Test full data flow (UI → API → DB → API → UI)
-   **Fixture Reuse:** Leverage existing fixtures from test helpers

See [fixtures-and-data-flow.md](./fixtures-and-data-flow.md) for comprehensive patterns.

## 6. Package-Specific Testing Guidance

### 6.1. Server Package (`/packages/server`)

-   **Endpoint Tests:** Located in `/src/endpoints/logic/*.test.ts`
-   **Model Tests:** Test database operations with transactions
-   **Service Tests:** Test business logic with real dependencies
-   **Execution Tiers:** Test each tier independently with event bus

Example structure:
```typescript
// endpoint test
describe("UserEndpoints", () => {
    const app = testingServer();
    
    it("should create user", async () => {
        const response = await app.request
            .post("/api/user")
            .send(userData);
        expect(response.status).toBe(201);
    });
});
```

### 6.2. UI Package (`/packages/ui`)

-   **Component Tests:** Use React Testing Library
-   **Hook Tests:** Test custom hooks in isolation
-   **Storybook Tests:** Visual regression testing
-   **MSW Integration:** Mock API responses for component testing

Example:
```typescript
import { render, screen } from "@testing-library/react";
import { TestWrapper } from "../__test/TestWrapper.js";

test("renders user profile", () => {
    render(
        <TestWrapper>
            <UserProfile userId="123" />
        </TestWrapper>
    );
    expect(screen.getByRole("heading")).toBeInTheDocument();
});
```

### 6.3. Jobs Package (`/packages/jobs`)

-   **Schedule Tests:** Test cron job execution
-   **Queue Tests:** Test job processing with Bull
-   **Service Mocks:** Use provided mock services

### 6.4. Shared Package (`/packages/shared`)

-   **Utility Tests:** Pure function testing
-   **Type Tests:** TypeScript compilation tests
-   **Validation Tests:** Schema validation testing

## 7. AI Maintenance Integration

### 7.1. Test Coverage Tracking

Tests are tracked using the AI maintenance system:

```typescript
// AI_CHECK: TEST_COVERAGE=15 | LAST: 2024-11-20
// Tests added for user authentication flow
```

### 7.2. Test Quality Reviews

Regular AI-driven reviews check for:
- Missing test cases
- Poor test descriptions
- Lack of edge case coverage
- Flaky test patterns

### 7.3. Maintenance Tasks

- **TEST_COVERAGE**: Improve test coverage in specific modules
- **TEST_QUALITY**: Enhance test reliability and clarity
- **TEST_PERFORMANCE**: Optimize slow-running tests

## 8. Schedule & Timeline

Testing activities are integrated into the development sprints:

-   **Unit/Integration Testing:** Continuous during development
-   **Component Testing:** During UI development sprints  
-   **E2E Testing:** Before major releases (Phase 2+)
-   **Performance Testing:** Phase 3 (T+24 weeks)
-   **Security Testing:** Via emergent AI agents (ongoing)

### 8.1. Current Phase (Phase 1: T+0-12 weeks)
- Focus on unit and integration test coverage
- Establish testing patterns and infrastructure
- Document testing guidelines

### 8.2. Phase 2 (T+12-24 weeks)
- Achieve 80%+ test coverage
- Implement E2E test suite
- Begin performance benchmarking

## 9. Entry and Exit Criteria

### 9.1. Entry Criteria (Start of a Test Cycle/Phase)

-   Development phase complete with passing linter
-   Test environment stable with containers running
-   Test data fixtures available
-   Unit/integration tests passing locally
-   Build successfully deployed to test environment

### 9.2. Exit Criteria (Conclusion of a Test Cycle/Phase)

-   All planned tests executed
-   Coverage targets met (80%+ for Phase 2)
-   Critical/high-severity defects fixed and retested
-   Outstanding defects within acceptable limits
-   Test summary report prepared and reviewed
-   AI maintenance checks passed

## 10. E2E Testing Roadmap

### 10.1. Current State
- Cypress installed in UI package
- No E2E tests implemented yet
- MSW provides foundation for mocking

### 10.2. Implementation Plan (Phase 2)
1. **User Journey Tests**: Authentication, profile management
2. **Routine Execution**: End-to-end routine creation and execution
3. **AI Chat Integration**: Full conversation flows
4. **Cross-Browser Testing**: Chrome, Firefox, Safari

### 10.3. E2E Test Structure
```typescript
// cypress/e2e/user-journey.cy.ts
describe("User Registration Journey", () => {
    it("completes full registration flow", () => {
        cy.visit("/register");
        cy.fillRegistrationForm(userData);
        cy.submitForm();
        cy.verifyEmailSent();
        cy.confirmEmail();
        cy.verifyDashboardAccess();
    });
});
```

## 11. Performance and Security Testing

### 11.1. Performance Testing (Phase 3)

**Approach**: Integrated with swarm execution testing

-   **Baseline Metrics**: Response times, throughput, resource usage
-   **Load Scenarios**: Concurrent users, routine executions, chat sessions
-   **Tools**: k6 for load testing, Vitest bench for microbenchmarks
-   **Integration Points**: Database queries, Redis operations, AI model calls

### 11.2. Security Testing

**Approach**: Emergent through AI agent capabilities

-   **Traditional Testing**: OWASP Top 10 coverage
-   **AI-Driven Testing**: Security agents identify vulnerabilities
-   **Continuous Monitoring**: Real-time threat detection
-   **Compliance**: GDPR, data privacy requirements

## 12. CI/CD Integration

### 12.1. Test Execution Pipeline

```yaml
# GitHub Actions workflow
test:
  strategy:
    matrix:
      package: [server, ui, jobs, shared]
  steps:
    - name: Run tests
      run: cd packages/${{ matrix.package }} && pnpm test
      timeout-minutes: 10
```

### 12.2. Parallelization Strategy
- Split tests by package
- Use Vitest's built-in parallel execution
- Separate unit and integration test jobs
- Cache dependencies between runs

### 12.3. Failure Handling
- Automatic retry for flaky tests (max 2 attempts)
- Detailed failure reports with logs
- Slack notifications for critical failures
- Block deployment on test failures

## 13. Common Pitfalls and Solutions

### 13.1. Import Extension Errors

**Problem**: `Cannot find module './User'`

**Solution**: Always add `.js` extension
```typescript
import { User } from "./User.js";  // ✅ Correct
```

### 13.2. Database Test Pollution

**Problem**: Tests fail when run together but pass individually

**Solution**: Use transaction wrapper
```typescript
await withDbTransaction(async (tx) => {
    // Test code here
});
```

### 13.3. Timeout Errors

**Problem**: `Test timed out after 2000ms`

**Solution**: Set extended timeouts
```typescript
test("long running test", async () => {
    // test code
}, 30000);  // 30 second timeout
```

### 13.4. Mock Service Worker Issues

**Problem**: API calls not intercepted in tests

**Solution**: Ensure MSW is started in test setup
```typescript
beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

## 14. Risk Analysis (Testing Specific)

| Risk ID | Risk Description                                      | Likelihood | Impact | Mitigation Strategy                                                                 |
| :------ | :---------------------------------------------------- | :--------- | :----- | :---------------------------------------------------------------------------------- |
| TR-001  | Insufficient test coverage leading to missed defects. | Medium     | High   | AI-driven coverage analysis; automated coverage reporting; mandatory PR checks     |
| TR-002  | Import extension errors breaking tests               | High       | Medium | Linting rules; documentation; code review checklist                               |
| TR-003  | Test environment instability                         | Low        | Medium | Containerized dependencies; health checks; automatic recovery                      |
| TR-004  | Flaky tests providing unreliable feedback            | Medium     | High   | Transaction isolation; deterministic test data; retry mechanisms                   |
| TR-005  | Slow test execution blocking development             | Medium     | Medium | Parallel execution; focused test runs; performance monitoring                      |
| TR-006  | Missing AI tier integration tests                    | Medium     | High   | Comprehensive tier testing strategy; event bus testing                             |

## 15. Test Deliverables

-   **This Test Plan**: Living document updated with project evolution
-   **Test Cases**: Documented as code with clear descriptions
-   **Coverage Reports**: Generated via `pnpm test-coverage`
-   **Test Fixtures**: Maintained in `__test` directories
-   **Defect Reports**: GitHub Issues with `bug` label
-   **Test Logs**: CI/CD artifacts and local test output
-   **AI Maintenance Reports**: Regular quality assessments

## 16. Monitoring and Metrics

### 16.1. Key Metrics
- **Test Coverage**: Target 80%+ by Phase 2
- **Test Execution Time**: Monitor for performance regression
- **Flaky Test Rate**: Target <2% flaky tests
- **Defect Discovery Rate**: Track bugs found in testing vs production

### 16.2. Reporting
- Weekly test coverage updates
- Sprint retrospective test metrics
- Monthly AI maintenance assessments

---

This Test Plan is a living document and will be updated as the project evolves. For implementation details and patterns, refer to [fixtures-and-data-flow.md](./fixtures-and-data-flow.md) and the [AI Maintenance Guide](../ai-maintenance/README.md).