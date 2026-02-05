## Steer focus: Unit Testing Architecture

Prioritize **establishing robust unit test infrastructure** in `scenarios/{{TARGET}}/` across all service directories (ui/, api/, sidecars). This skill steers toward a testing foundation that makes it easy for agents and developers to add, maintain, and trust tests.

Your goal is to ensure the target scenario has **professional test architecture**—proper file organization, isolated test execution, testable production code, organized mocks, and systematic coverage patterns—so that test quality skills can build on a solid foundation.

Do **not** focus on test coverage percentages or fixing failing tests. This skill is about the **backbone** of unit testing, not the content of individual tests.

Required reading:
- `prompt-manager skills read visited-tracker-tools`

---

### 0. Why This Skill Exists

Unit test architecture problems are invisible until they compound:

- **Scattered test files**: Developers can't find tests, duplicates emerge, coverage gaps hide
- **Shared test state**: Tests pass in isolation but fail when run together, or vice versa
- **Untestable production code**: Hard-coded dependencies make mocking impossible
- **Disorganized mocks**: Each test reinvents mocks, leading to drift and maintenance burden
- **Happy-path-only structure**: Test suites lack systematic edge case coverage
- **Brittle tests**: Tests break when implementation changes, not when behavior changes

**The distinction from test.md skill:**

| test.md (Test Suite Strengthening) | This Skill (Unit Testing Architecture) |
|-----------------------------------|----------------------------------------|
| Improves test quality and coverage | Establishes test infrastructure |
| Adds edge case tests | Creates patterns for systematic edge case testing |
| Fixes flaky tests | Prevents flakiness through proper isolation setup |
| Reviews individual test assertions | Sets up mocking infrastructure |
| Works within existing architecture | Builds the architecture other skills depend on |

**When architecture is right:**
- Adding a new test is obvious (clear file location, existing helpers, mock patterns)
- Tests are trustworthy (isolated, deterministic, fast)
- Production code is testable (dependency injection, clear boundaries)
- Edge cases are systematically covered (not ad-hoc)

---

### 1. Scope Boundaries

**In scope**
- Test file organization and naming conventions by language/platform
- Test isolation patterns (setup/teardown, fixture management)
- Mock organization (centralized mock packages, factory patterns)
- Production code testability (dependency injection, interface boundaries)
- Testcontainers setup for database and external service testing
- Test helper/utility organization
- Systematic edge case coverage patterns (equivalence partitioning, boundary analysis)
- Test configuration files (vitest.config.ts, *_test.go patterns)

**Out of scope**
- Individual test quality or assertions → see `test` skill
- Test coverage metrics or increasing coverage → see `test` skill
- E2E/integration test workflows → see `e2e-testing` skill
- Fixing specific failing tests → see `test` skill
- Performance testing setup → different concern

---

### 2. Test File Organization

### 2.1 The Co-location Principle

**Unit tests live alongside the code they test.** This is non-negotiable for unit tests.

```
# CORRECT: Co-located unit tests
api/
  handlers/
    user.go
    user_test.go          # Right next to user.go
  services/
    auth.go
    auth_test.go          # Right next to auth.go

ui/src/
  components/
    Button/
      Button.tsx
      Button.test.tsx     # Right next to Button.tsx
```

```
# WRONG: Separate test directories for unit tests
api/
  handlers/
    user.go
  tests/                  # Far from source
    handlers/
      user_test.go        # Hard to find, easy to forget
```

**Why co-location matters:**
- When you break code, the test is right there
- Import paths are simple (no `../../..` chains)
- Missing tests are immediately visible
- File renames don't orphan tests

### 2.2 Language-Specific Naming Conventions

| Language | Test File Pattern | Test Function Pattern | Example |
|----------|-------------------|----------------------|---------|
| **Go** | `*_test.go` | `TestFunctionName` or `TestType_Method` | `handler_test.go`, `TestCreateUser` |
| **TypeScript** | `*.test.ts` / `*.test.tsx` | `describe`/`it` blocks | `Button.test.tsx` |
| **Python** | `test_*.py` | `test_function_name` | `test_user_service.py` |

**Go specifics:**
- Test files excluded from production builds automatically
- Use subtests: `t.Run("scenario", func(t *testing.T) {...})`
- Table-driven tests for systematic coverage

**TypeScript specifics:**
- Use Vitest (not Jest) for Vite projects—10-20x faster
- Configure in `vitest.config.ts` or `vite.config.ts`
- Use `@testing-library/react` for component testing

**Python specifics:**
- Use `pytest` (not unittest)
- `conftest.py` for shared fixtures at each directory level
- Fixtures with `yield` for automatic cleanup

### 2.3 Directory Structure Decision Tree

```
Where should this test file go?
│
├─ Is it a unit test for a specific source file?
│   └─ YES → Same directory, same name + test suffix
│            Go: handler.go → handler_test.go
│            TS: Button.tsx → Button.test.tsx
│
├─ Is it testing integration between multiple components?
│   └─ YES → tests/integration/ at service root
│            scenarios/{{TARGET}}/api/tests/integration/
│
├─ Is it a test helper/utility?
│   └─ YES → Dedicated testutil package
│            Go: internal/testutil/
│            TS: src/test-utils/
│
└─ Is it a shared fixture/factory?
    └─ YES → Test utilities directory
             Go: internal/testutil/fixtures.go
             TS: src/test-utils/factories/
```

---

### 3. Test Isolation Patterns

### 3.1 The Isolation Principle

**Every test must be independent.** Tests must:
- Pass when run alone
- Pass when run with other tests
- Pass in any order
- Pass on any developer's machine
- Pass in CI with no special setup

### 3.2 The Arrange-Act-Assert Pattern

Structure ALL tests with this pattern:

```go
func TestCreateUser_ValidInput(t *testing.T) {
    // ARRANGE: Set up test data and dependencies
    repo := NewMockUserRepository()
    service := NewUserService(repo)
    input := CreateUserInput{Name: "Alice", Email: "alice@test.com"}

    // ACT: Execute the code under test
    user, err := service.CreateUser(input)

    // ASSERT: Verify the results
    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
    assert.NotEmpty(t, user.ID)
}
```

```typescript
it('creates user with valid input', async () => {
    // ARRANGE
    const repo = createMockUserRepository();
    const service = new UserService(repo);
    const input = { name: 'Alice', email: 'alice@test.com' };

    // ACT
    const user = await service.createUser(input);

    // ASSERT
    expect(user.name).toBe('Alice');
    expect(user.id).toBeDefined();
});
```

### 3.3 Isolation Checklist

| Isolation Concern | Pattern | Anti-pattern |
|-------------------|---------|--------------|
| **Shared state** | Fresh fixtures per test | Global variables modified by tests |
| **File system** | `t.TempDir()` / temp directories | Writing to fixed paths |
| **Time** | Inject time provider | `time.Now()` in production code |
| **Randomness** | Seed or inject RNG | `rand.Int()` without seed |
| **Environment** | Mock env reader | Direct `os.Getenv()` |
| **Database** | Transaction rollback or testcontainers | Shared test database |
| **External services** | Mocks or testcontainers | Real external calls |

### 3.4 Go Isolation Pattern

```go
func TestHandler(t *testing.T) {
    // Use t.TempDir() for automatic cleanup
    tempDir := t.TempDir()

    // Use t.Cleanup() for custom cleanup
    cleanup := setupTestLogger()
    t.Cleanup(cleanup)

    // Subtests for organization
    t.Run("ValidInput", func(t *testing.T) {
        // Each subtest is isolated
    })

    t.Run("InvalidInput", func(t *testing.T) {
        // Fresh state for each subtest
    })
}
```

### 3.5 TypeScript Isolation Pattern

```typescript
describe('UserService', () => {
    let service: UserService;
    let mockRepo: MockUserRepository;

    beforeEach(() => {
        // Fresh instances for each test
        mockRepo = createMockUserRepository();
        service = new UserService(mockRepo);
    });

    afterEach(() => {
        // Clear any shared state
        vi.clearAllMocks();
    });

    it('creates user', () => {
        // Test with fresh service instance
    });
});
```

---

### 4. Mock Organization

### 4.1 Centralized Mock Packages

**Mocks belong in dedicated packages/directories, not scattered through test files.**

```
# Go: Centralized mock package
api/
  services/
    user/
      service.go
      service_test.go
      mocks/                    # Dedicated mock package
        repository.go           # MockUserRepository
        email_sender.go         # MockEmailSender
        mocks.go               # Shared mock utilities
```

```
# TypeScript: Centralized mock directory
ui/src/
  services/
    user/
      userService.ts
      userService.test.ts
  __mocks__/                    # Jest/Vitest convention
    userRepository.ts
  test-utils/
    mocks/                      # Alternative: explicit directory
      createMockUser.ts
      createMockRepository.ts
```

### 4.2 Mock Builder Pattern (Go)

Create chainable mock builders for complex dependencies:

```go
// mocks/repository.go
type MockRepository struct {
    users     map[string]*User
    findError error
    saveError error
}

func NewMockRepository() *MockRepository {
    return &MockRepository{
        users: make(map[string]*User),
    }
}

func (m *MockRepository) WithUser(u *User) *MockRepository {
    m.users[u.ID] = u
    return m
}

func (m *MockRepository) WithFindError(err error) *MockRepository {
    m.findError = err
    return m
}

// Usage in tests:
repo := NewMockRepository().
    WithUser(&User{ID: "1", Name: "Alice"}).
    WithFindError(ErrNotFound)
```

### 4.3 Mock Factory Pattern (TypeScript)

Create factory functions for consistent mock creation:

```typescript
// test-utils/mocks/userMocks.ts
export function createMockUser(overrides: Partial<User> = {}): User {
    return {
        id: 'test-user-id',
        name: 'Test User',
        email: 'test@example.com',
        createdAt: new Date('2024-01-01'),
        ...overrides,
    };
}

export function createMockUserRepository(): MockUserRepository {
    return {
        findById: vi.fn(),
        save: vi.fn(),
        delete: vi.fn(),
    };
}

// Usage in tests:
const user = createMockUser({ name: 'Alice' });
const repo = createMockUserRepository();
repo.findById.mockResolvedValue(user);
```

### 4.4 When to Mock vs. Use Real Implementations

```
Should I mock this dependency?
│
├─ Is it an external service (API, database, file system)?
│   └─ YES → Mock it OR use testcontainers
│
├─ Is it slow (network, disk, computation)?
│   └─ YES → Mock it for unit tests
│
├─ Is it non-deterministic (time, random, external state)?
│   └─ YES → Mock it
│
├─ Is it a simple value object or pure function?
│   └─ NO need to mock → Use real implementation
│
└─ Is it owned by another team/service?
    └─ YES → Mock it (don't depend on their implementation)
```

### 4.5 Testcontainers for Database Testing

When tests need real database behavior, use testcontainers instead of mocks:

**Go:**
```go
import (
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestWithDatabase(t *testing.T) {
    ctx := context.Background()

    // Start real Postgres in Docker
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15"),
        postgres.WithDatabase("testdb"),
    )
    require.NoError(t, err)
    t.Cleanup(func() { container.Terminate(ctx) })

    // Get connection string
    connStr, err := container.ConnectionString(ctx)
    require.NoError(t, err)

    // Run migrations and tests against real database
    db := connectToDatabase(connStr)
    // ... test with real database
}
```

**TypeScript:**
```typescript
import { PostgreSqlContainer } from '@testcontainers/postgresql';

describe('UserRepository', () => {
    let container: StartedPostgreSqlContainer;
    let db: Database;

    beforeAll(async () => {
        container = await new PostgreSqlContainer().start();
        db = await connectToDatabase(container.getConnectionUri());
        await runMigrations(db);
    });

    afterAll(async () => {
        await db.close();
        await container.stop();
    });

    it('saves user to database', async () => {
        // Test with real database
    });
});
```

---

### 5. Writing Testable Production Code

### 5.1 The Dependency Injection Principle

**Production code must accept dependencies, not create them.**

```go
// UNTESTABLE: Hard-coded dependency
func ProcessOrder(orderID string) error {
    db := database.Connect()  // Can't substitute for testing
    return db.Save(orderID)
}

// TESTABLE: Injected dependency
func ProcessOrder(db Database, orderID string) error {
    return db.Save(orderID)
}

// In production:
ProcessOrder(realDB, orderID)

// In tests:
ProcessOrder(mockDB, orderID)
```

### 5.2 Interface-Based Design

Define interfaces for dependencies, implement them for production and testing:

```go
// Define interface at point of use
type OrderRepository interface {
    Save(order Order) error
    FindByID(id string) (*Order, error)
}

// Production implementation
type PostgresOrderRepo struct {
    db *sql.DB
}

// Test implementation
type MockOrderRepo struct {
    orders map[string]*Order
}

// Service accepts interface
type OrderService struct {
    repo OrderRepository  // Can be either implementation
}
```

### 5.3 Functional Core, Imperative Shell

Separate pure logic from side effects:

```go
// FUNCTIONAL CORE: Pure, deterministic, easy to test
func CalculateDiscount(price float64, tier string) float64 {
    switch tier {
    case "gold":
        return price * 0.20
    case "silver":
        return price * 0.10
    default:
        return 0
    }
}

// IMPERATIVE SHELL: Side effects, uses DI
type OrderService struct {
    repo   OrderRepository
    mailer EmailSender
}

func (s *OrderService) PlaceOrder(order Order) error {
    // Use functional core
    discount := CalculateDiscount(order.Total, order.CustomerTier)
    order.Discount = discount

    // Side effects through injected dependencies
    if err := s.repo.Save(order); err != nil {
        return err
    }
    return s.mailer.SendConfirmation(order)
}
```

### 5.4 Testability Checklist

| Concern | Testable Pattern | Untestable Pattern |
|---------|------------------|-------------------|
| **Dependencies** | Constructor injection | `new` inside methods |
| **Time** | `TimeProvider` interface | `time.Now()` direct calls |
| **Environment** | `EnvReader` interface | `os.Getenv()` direct calls |
| **File system** | `FileSystem` interface | Direct `os` package calls |
| **HTTP clients** | Injected client interface | Global `http.DefaultClient` |
| **Logging** | Injected logger | Global logger instance |
| **Configuration** | Passed as parameter | Global config singleton |

---

### 6. Systematic Edge Case Coverage

### 6.1 Equivalence Partitioning

Instead of testing every possible input, group inputs into classes that behave the same way:

```go
func TestAgeValidation(t *testing.T) {
    tests := []struct {
        name     string
        age      int
        valid    bool
        partition string  // Document which partition
    }{
        // Valid partition (0-150)
        {"valid_young", 1, true, "valid"},
        {"valid_middle", 50, true, "valid"},
        {"valid_old", 100, true, "valid"},

        // Invalid: negative partition
        {"invalid_negative", -1, false, "negative"},
        {"invalid_very_negative", -100, false, "negative"},

        // Invalid: over maximum partition
        {"invalid_over_max", 151, false, "over_max"},
        {"invalid_way_over", 1000, false, "over_max"},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            result := ValidateAge(tc.age)
            assert.Equal(t, tc.valid, result)
        })
    }
}
```

### 6.2 Boundary Value Analysis

Test at the edges where bugs commonly occur:

```go
func TestFieldLengthValidation(t *testing.T) {
    // Field accepts 1-100 characters
    tests := []struct {
        name   string
        length int
        valid  bool
        why    string
    }{
        // Boundary values
        {"below_min", 0, false, "just below minimum"},
        {"at_min", 1, true, "exactly at minimum"},
        {"above_min", 2, true, "just above minimum"},
        {"below_max", 99, true, "just below maximum"},
        {"at_max", 100, true, "exactly at maximum"},
        {"above_max", 101, false, "just above maximum"},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            input := strings.Repeat("a", tc.length)
            result := ValidateFieldLength(input)
            assert.Equal(t, tc.valid, result, tc.why)
        })
    }
}
```

### 6.3 Standard Edge Case Checklist

Every function handling external input should have tests for:

| Category | Test Cases |
|----------|-----------|
| **Null/Empty** | `nil`, `null`, `undefined`, `""`, `[]`, `{}` |
| **Boundaries** | min, max, min-1, max+1 |
| **Type edges** | 0, -1, MAX_INT, MIN_INT, NaN, Infinity |
| **Strings** | Unicode, emoji, control characters, very long, whitespace-only |
| **Collections** | Empty, single item, many items, duplicates |
| **Dates** | Leap years, DST transitions, timezone boundaries |
| **Concurrency** | Simultaneous calls, race conditions (if applicable) |

### 6.4 Table-Driven Test Template

Use this template to ensure systematic coverage:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
        errMsg   string
        category string  // "happy_path", "boundary", "error", "edge_case"
    }{
        // Happy path
        {"valid_basic", validInput, expectedOutput, false, "", "happy_path"},

        // Boundaries
        {"at_min_boundary", minInput, minOutput, false, "", "boundary"},
        {"at_max_boundary", maxInput, maxOutput, false, "", "boundary"},

        // Error cases
        {"invalid_empty", emptyInput, zero, true, "input required", "error"},
        {"invalid_over_max", overMaxInput, zero, true, "exceeds maximum", "error"},

        // Edge cases
        {"unicode_input", unicodeInput, unicodeOutput, false, "", "edge_case"},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            result, err := Function(tc.input)

            if tc.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tc.errMsg)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

---

### 7. Test Helper Organization

### 7.1 Go Test Utilities Package

```
api/internal/testutil/
├── helpers.go          # Core utilities
├── fixtures.go         # Test data factories
├── assertions.go       # Custom assertions
└── mocks.go           # Shared mock utilities
```

```go
// helpers.go
package testutil

import (
    "testing"
    "net/http/httptest"
)

// AssertStatus checks HTTP response status
func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
    t.Helper()
    if rec.Code != expected {
        t.Fatalf("expected status %d, got %d: %s", expected, rec.Code, rec.Body.String())
    }
}

// MustDecodeJSON decodes JSON response or fails test
func MustDecodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
    t.Helper()
    var result T
    if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
        t.Fatalf("failed to decode JSON: %v", err)
    }
    return result
}

// SetupTestEnvironment creates isolated test environment
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
    t.Helper()
    tempDir := t.TempDir()

    env := &TestEnvironment{
        TempDir: tempDir,
    }

    t.Cleanup(func() {
        env.Cleanup()
    })

    return env
}
```

### 7.2 TypeScript Test Utilities

```
ui/src/test-utils/
├── index.ts            # Re-exports
├── setup.ts            # Test setup (imported by vitest.config.ts)
├── renderWithProviders.tsx  # Custom render with context providers
├── factories/
│   ├── userFactory.ts
│   └── orderFactory.ts
└── mocks/
    ├── handlers.ts     # MSW handlers
    └── server.ts       # MSW server setup
```

```typescript
// test-utils/renderWithProviders.tsx
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

interface CustomRenderOptions extends RenderOptions {
    queryClient?: QueryClient;
}

export function renderWithProviders(
    ui: React.ReactElement,
    options: CustomRenderOptions = {}
) {
    const {
        queryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
            },
        }),
        ...renderOptions
    } = options;

    function Wrapper({ children }: { children: React.ReactNode }) {
        return (
            <QueryClientProvider client={queryClient}>
                {children}
            </QueryClientProvider>
        );
    }

    return render(ui, { wrapper: Wrapper, ...renderOptions });
}
```

```typescript
// test-utils/setup.ts
import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

// Cleanup after each test
afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

// Mock ResizeObserver (not available in jsdom)
global.ResizeObserver = vi.fn().mockImplementation(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
}));
```

---

### 8. Unit Testing Architecture Audit

Before making changes, assess `{{TARGET}}`'s current testing architecture.

### 8.1 Audit Commands

```bash
# Find test file organization pattern
echo "=== Test File Locations ==="
# Go tests
find scenarios/{{TARGET}}/api -name "*_test.go" | head -20

# TypeScript tests
find scenarios/{{TARGET}}/ui -name "*.test.ts" -o -name "*.test.tsx" | head -20

# Check for __tests__ directories (inconsistent pattern)
find scenarios/{{TARGET}} -type d -name "__tests__"

# Find mock organization
echo "=== Mock Organization ==="
find scenarios/{{TARGET}} -type d -name "mocks" -o -name "__mocks__"
find scenarios/{{TARGET}} -name "*mock*" -type f | head -20

# Find test utilities
echo "=== Test Utilities ==="
find scenarios/{{TARGET}} -type d -name "testutil" -o -name "test-utils"
find scenarios/{{TARGET}} -name "conftest.py"

# Check testcontainers usage
echo "=== Testcontainers ==="
rg "testcontainers" scenarios/{{TARGET}}/ --type go --type ts -l

# Check for dependency injection patterns
echo "=== Dependency Injection ==="
rg "interface\s+\w+Repository" scenarios/{{TARGET}}/api --type go
rg "interface.*Service" scenarios/{{TARGET}}/ui --type ts

# Find inline mocks (should be centralized)
echo "=== Inline Mocks (potential issue) ==="
rg "type mock\w+ struct" scenarios/{{TARGET}}/api --type go
rg "const mock\w+ = " scenarios/{{TARGET}}/ui --type ts
```

### 8.2 Red Flags Checklist

- [ ] Test files in separate `tests/` directory instead of co-located
- [ ] Inconsistent naming (`*.test.ts` mixed with `*.spec.ts`)
- [ ] No dedicated mock package (mocks defined inline in each test file)
- [ ] No test utilities package (`testutil/` or `test-utils/`)
- [ ] Hard-coded dependencies in production code (no DI)
- [ ] Global state used in tests (shared variables between tests)
- [ ] Database tests that skip when database unavailable (no testcontainers)
- [ ] Missing `t.Helper()` calls in Go test helpers
- [ ] Missing `beforeEach`/`afterEach` cleanup in TypeScript tests
- [ ] No table-driven tests for validation logic

### 8.3 Document Findings

Record audit results in `scenarios/{{TARGET}}/docs/internal/UNIT_TEST_ARCHITECTURE.md`:

```markdown
# {{TARGET}} Unit Testing Architecture

## Last Updated
[Date]

## Test Organization Status
- [ ] Go tests co-located with source files
- [ ] TypeScript tests co-located with source files
- [ ] Consistent naming conventions
- [ ] Test utilities package exists

## Mock Organization Status
- [ ] Centralized mock packages
- [ ] Mock factory/builder patterns
- [ ] No inline mock definitions in test files

## Testability Status
- [ ] Dependency injection used throughout
- [ ] Interfaces defined for external dependencies
- [ ] Time/environment/filesystem abstracted

## Infrastructure Status
- [ ] Testcontainers configured for database tests
- [ ] Test setup files configured
- [ ] CI runs tests successfully

## Issues Found
1. [File:line] - Issue description
2. ...

## Priority Improvements
1. [Highest impact] - Why
2. ...
```

---

### 9. Memory Management with Visited Tracker

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `unit-test-architecture`.

---

### 10. Output Expectations

You may update in `scenarios/{{TARGET}}/`:
- Add mock packages (`api/*/mocks/`, `ui/src/test-utils/mocks/`)
- Add test utility packages (`api/internal/testutil/`, `ui/src/test-utils/`)
- Add test configuration files (`vitest.config.ts`, test setup files)
- Refactor production code for testability (add interfaces, DI patterns)
- Move test files to proper co-located positions
- Add testcontainers setup for database-dependent tests
- Create mock factory/builder patterns
- Standardize test file naming

You must:
- Keep `{{TARGET}}` fully functional and non-regressed
- Ensure all existing tests still pass
- Follow language-specific conventions (Go, TypeScript, Python)
- Create reusable patterns, not one-off solutions
- Document architecture decisions in `docs/internal/UNIT_TEST_ARCHITECTURE.md`

You must NOT:
- Add new tests (focus on infrastructure)
- Change test assertions or coverage
- Remove existing tests
- Add heavyweight DI frameworks (prefer simple constructor injection)
- Over-abstract (interfaces only where needed for testing)

**Avoid superficial changes that rename files or move code without genuinely improving test architecture.**
