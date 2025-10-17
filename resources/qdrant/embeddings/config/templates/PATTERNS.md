# Patterns

This document captures code patterns, integration patterns, anti-patterns, and best practices used in this project.

## Code Patterns

<!-- EMBED:CODEPATTERN:START -->
### Pattern Name
**Category:** [Creational|Structural|Behavioral|Functional]
**Purpose:** What problem does this pattern solve?
**When to Use:** Situations where this pattern applies
**When NOT to Use:** Situations to avoid this pattern

#### Implementation
```javascript
// Example implementation
class PatternExample {
  // Pattern code here
}
```

#### Usage Example
```javascript
// How to use this pattern
const instance = new PatternExample()
instance.method()
```

#### Benefits
- List of advantages
- Why this pattern works well here

#### Considerations
- Things to watch out for
- Performance implications
- Maintenance considerations
<!-- EMBED:CODEPATTERN:END -->

## Integration Patterns

<!-- EMBED:INTEGRATION:START -->
### Integration Pattern Name
**Systems:** What systems are being integrated
**Pattern Type:** [API Gateway|Message Queue|Event Bus|Webhook|Polling|etc.]
**Direction:** [Unidirectional|Bidirectional|Pub-Sub]

#### Architecture
```
System A --> Pattern --> System B
         <--         <--
```

#### Implementation
```javascript
// Integration code example
const integration = {
  source: 'SystemA',
  destination: 'SystemB',
  method: 'webhook',
  retryPolicy: {
    attempts: 3,
    backoff: 'exponential'
  }
}
```

#### Error Handling
- Retry strategy
- Fallback behavior
- Error notification

#### Monitoring
- What metrics to track
- Alert thresholds
- Success criteria
<!-- EMBED:INTEGRATION:END -->

## Anti-Patterns to Avoid

<!-- EMBED:ANTIPATTERN:START -->
### Anti-Pattern Name
**Why It's Problematic:** The issues this causes
**How to Recognize:** Signs this anti-pattern is present
**Common Causes:** Why developers fall into this trap

#### Bad Example
```javascript
// DON'T DO THIS
// Example of the anti-pattern
function badPattern() {
  // Problematic code
}
```

#### Good Alternative
```javascript
// DO THIS INSTEAD
// Proper implementation
function goodPattern() {
  // Better approach
}
```

#### Refactoring Steps
1. Identify instances of the anti-pattern
2. Plan the refactoring approach
3. Update and test incrementally
4. Verify improvements
<!-- EMBED:ANTIPATTERN:END -->

## Best Practices

<!-- EMBED:BESTPRACTICE:START -->
### Practice Name
**Area:** [Security|Performance|Maintainability|Testing|etc.]
**Principle:** The underlying principle
**Implementation:** How to apply this practice

#### Guidelines
- Specific rules to follow
- Common exceptions
- Edge cases to consider

#### Example
```javascript
// Best practice example
// Good: Clear, maintainable, performant
async function bestPracticeExample() {
  try {
    const result = await operation()
    return { success: true, data: result }
  } catch (error) {
    logger.error('Operation failed', { error })
    return { success: false, error: error.message }
  }
}
```

#### Validation
- How to verify this practice is followed
- Automated checks available
- Code review checklist items
<!-- EMBED:BESTPRACTICE:END -->

## Error Handling Patterns

<!-- EMBED:ERRORHANDLING:START -->
### Error Handling Strategy
**Scope:** Where this applies
**Error Types:** What errors are handled
**Recovery Strategy:** How the system recovers

#### Implementation
```javascript
// Error handling pattern
class ErrorHandler {
  static async withRetry(fn, options = {}) {
    const { retries = 3, delay = 1000 } = options
    
    for (let i = 0; i < retries; i++) {
      try {
        return await fn()
      } catch (error) {
        if (i === retries - 1) throw error
        await new Promise(r => setTimeout(r, delay * Math.pow(2, i)))
      }
    }
  }
}
```

#### Usage
```javascript
const result = await ErrorHandler.withRetry(
  () => fetchData(),
  { retries: 3, delay: 1000 }
)
```
<!-- EMBED:ERRORHANDLING:END -->

## Data Access Patterns

<!-- EMBED:DATAACCESS:START -->
### Data Access Pattern
**Type:** [Repository|DAO|Active Record|etc.]
**Database:** What database/storage system
**Use Case:** When to use this pattern

#### Implementation
```javascript
// Repository pattern example
class UserRepository {
  async findById(id) {
    // Data access logic
  }
  
  async save(user) {
    // Persistence logic
  }
  
  async delete(id) {
    // Deletion logic
  }
}
```

#### Transaction Handling
```javascript
// How transactions are managed
async function transactionalOperation() {
  const transaction = await db.beginTransaction()
  try {
    await operation1(transaction)
    await operation2(transaction)
    await transaction.commit()
  } catch (error) {
    await transaction.rollback()
    throw error
  }
}
```
<!-- EMBED:DATAACCESS:END -->

## State Management Patterns

<!-- EMBED:STATE:START -->
### State Management Pattern
**Type:** [Redux|MobX|Context|Zustand|etc.]
**Scope:** [Global|Component|Feature]
**Persistence:** How state is persisted

#### Store Structure
```javascript
// State structure example
const storeStructure = {
  user: {
    profile: {},
    preferences: {}
  },
  app: {
    theme: 'light',
    language: 'en'
  },
  data: {
    items: [],
    loading: false,
    error: null
  }
}
```

#### Actions/Mutations
```javascript
// State update patterns
const actions = {
  setUser: (state, user) => {
    state.user = user
  },
  updatePreferences: (state, prefs) => {
    state.user.preferences = { ...state.user.preferences, ...prefs }
  }
}
```
<!-- EMBED:STATE:END -->

## Testing Patterns

<!-- EMBED:TESTING:START -->
### Testing Pattern
**Type:** [Unit|Integration|E2E|Contract]
**Framework:** Testing framework used
**Coverage Target:** What should be tested

#### Test Structure
```javascript
// Test pattern example
describe('Component/Module', () => {
  beforeEach(() => {
    // Setup
  })
  
  afterEach(() => {
    // Cleanup
  })
  
  it('should behave correctly', () => {
    // Arrange
    const input = setupTestData()
    
    // Act
    const result = functionUnderTest(input)
    
    // Assert
    expect(result).toMatchExpectedOutput()
  })
})
```

#### Mocking Strategy
```javascript
// How to mock dependencies
const mockDependency = {
  method: jest.fn().mockResolvedValue(expectedResult)
}
```
<!-- EMBED:TESTING:END -->

## Communication Patterns

<!-- EMBED:COMMUNICATION:START -->
### Communication Pattern
**Type:** [REST|GraphQL|WebSocket|gRPC|Message Queue]
**Protocol:** Communication protocol used
**Format:** Data format (JSON/XML/Protobuf)

#### Request/Response Pattern
```javascript
// API communication pattern
class APIClient {
  async request(endpoint, options = {}) {
    const response = await fetch(endpoint, {
      ...this.defaultOptions,
      ...options
    })
    
    if (!response.ok) {
      throw new APIError(response)
    }
    
    return response.json()
  }
}
```

#### Event-Driven Pattern
```javascript
// Event communication pattern
class EventBus {
  emit(event, data) {
    // Publish event
  }
  
  on(event, handler) {
    // Subscribe to event
  }
}
```
<!-- EMBED:COMMUNICATION:END -->

## Security Patterns

<!-- EMBED:SECURITYPATTERN:START -->
### Security Pattern Name
**Threat Mitigated:** What security threat this addresses
**Implementation Level:** [Application|Network|Data]

#### Implementation
```javascript
// Security pattern example
class SecureHandler {
  // Input validation
  validateInput(input) {
    // Validation logic
  }
  
  // Output encoding
  encodeOutput(output) {
    // Encoding logic
  }
  
  // Rate limiting
  async checkRateLimit(userId) {
    // Rate limit logic
  }
}
```

#### Usage Guidelines
- When to apply this pattern
- Configuration requirements
- Testing approach
<!-- EMBED:SECURITYPATTERN:END -->

## Deployment Patterns

<!-- EMBED:DEPLOYMENT:START -->
### Deployment Pattern
**Strategy:** [Blue-Green|Canary|Rolling|Recreate]
**Environment:** [Dev|Staging|Production]
**Rollback Time:** How quickly can rollback occur

#### Process
1. Pre-deployment checks
2. Deployment steps
3. Validation steps
4. Rollback triggers

#### Configuration
```yaml
deployment:
  strategy: blue-green
  health_check:
    endpoint: /
    timeout: 30s
  rollback:
    automatic: true
    threshold: 5% error rate
```
<!-- EMBED:DEPLOYMENT:END -->

---

## Pattern Decision Matrix

| Pattern | Use When | Don't Use When | Complexity | Performance |
|---------|----------|----------------|------------|-------------|
| Repository | Need data abstraction | Simple CRUD only | Medium | Good |
| Event Bus | Loose coupling needed | Simple direct calls work | High | Good |
| Factory | Complex object creation | Simple objects | Medium | Good |
| Observer | Multiple listeners | Single listener | Low | Good |
| Singleton | Single instance needed | Testing is priority | Low | Excellent |

## Pattern Relationships

```
Repository Pattern
    ├── Uses Factory Pattern (for entity creation)
    ├── Implements Unit of Work (for transactions)
    └── Works with Specification Pattern (for queries)

Event Bus Pattern
    ├── Implements Observer Pattern
    ├── Uses Command Pattern (for events)
    └── Can use Message Queue Pattern (for async)
```

## Code Review Checklist

### Pattern Compliance
- [ ] Appropriate pattern selected for use case
- [ ] Pattern correctly implemented
- [ ] No anti-patterns introduced
- [ ] Error handling follows patterns
- [ ] Testing patterns followed

### Best Practices
- [ ] SOLID principles followed
- [ ] DRY principle applied appropriately
- [ ] Clear separation of concerns
- [ ] Consistent naming conventions
- [ ] Adequate documentation

## Resources

- [Design Patterns Reference](https://refactoring.guru/design-patterns)
- [Project Style Guide](./docs/style-guide.md)
- [Architecture Decision Records](./docs/architecture/)