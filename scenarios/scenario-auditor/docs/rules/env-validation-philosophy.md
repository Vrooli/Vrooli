# Environment Variable Validation Rule - Design Philosophy

## Overview

The `env_validation` rule enforces explicit, fail-fast validation of environment variables across Go, Bash, and JavaScript/TypeScript. This document explains the intentional design decisions that make this rule strict and opinionated.

## Core Principles

### 1. Explicit Over Implicit

**Principle**: Validation must be explicit and visible in the code.

**Rationale**:
- Makes configuration requirements immediately obvious to developers
- Prevents silent failures with incorrect defaults
- Facilitates code review and security audits
- Documents runtime dependencies

**Example of Disallowed Pattern**:
```go
// ❌ DISALLOWED: Implicit default
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
```

**Approved Pattern**:
```go
// ✅ APPROVED: Explicit validation with conscious default
port := os.Getenv("PORT")
if port == "" {
    log.Fatal("PORT environment variable is required")
}
// OR with explicit non-production default:
port := os.Getenv("PORT")
if port == "" {
    if os.Getenv("ENV") == "development" {
        port = "8080" // Development-only default
    } else {
        log.Fatal("PORT is required in production")
    }
}
```

### 2. Fail-Fast Philosophy

**Principle**: Applications should fail immediately at startup if required configuration is missing.

**Rationale**:
- Prevents runtime errors deep in application logic
- Makes deployment issues immediately visible
- Reduces debugging time
- Prevents partial service degradation

**Why We Disallow Fallback Patterns**:

```bash
# ❌ DISALLOWED: Silent fallback
PORT="${PORT:-8080}"

# ❌ DISALLOWED: Nullish coalescing
const port = process.env.PORT ?? '8080';

# ❌ DISALLOWED: Destructuring with defaults
const { API_URL = 'http://localhost' } = process.env;
```

**Problems with these patterns**:
1. **Production Risk**: Wrong defaults can cause production issues
2. **Configuration Drift**: Different environments may use different defaults
3. **Security**: Default ports/URLs may be insecure
4. **Documentation**: Hidden defaults make configuration requirements unclear

### 3. Locality of Validation

**Principle**: Validation must occur within 8 lines of variable assignment.

**Rationale**:
- Enforces code structure where validation is near usage
- Prevents using env vars before validation
- Makes code easier to review and understand
- Reduces cognitive load

**Example of Violation**:
```go
// ❌ REJECTED: Validation too far from assignment
dbURL := os.Getenv("DATABASE_URL")  // Line 10

// ... 10 lines of other code ...

if dbURL == "" {  // Line 20 - TOO FAR!
    log.Fatal("DATABASE_URL required")
}
```

**Correct Approach**:
```go
// ✅ APPROVED: Validation immediately follows
dbURL := os.Getenv("DATABASE_URL")
if dbURL == "" {
    log.Fatal("DATABASE_URL required")
}

// OR: Refactor into function
func getDBURL() string {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL required")
    }
    return dbURL
}
```

### 4. Comment Scanning is Intentional

**Principle**: The rule intentionally scans commented code and reports violations.

**Rationale**:
- Catches dangerous "TODO: uncomment before deploy" scenarios
- Prevents temporary disabling of validation from being committed
- Enforces clean commits without commented-out code
- Security-critical code should never be commented out

**Example of Detected Issue**:
```go
// ❌ CAUGHT: This is a security risk!
apiKey := os.Getenv("API_KEY")
// if apiKey == "" { log.Fatal("API_KEY required") }
// TODO: uncomment before production
useAPIKey(apiKey)
```

**If you need to document validation patterns**:
- Use documentation files (.md) not code comments
- Documentation files are excluded from scanning
- Keep example code in dedicated example directories

### 5. No Implicit Default Patterns

**Languages Affected**: Go, Bash, JavaScript/TypeScript

**Disallowed Patterns**:

#### Bash Parameter Expansion
```bash
# ❌ DISALLOWED
PORT="${PORT:-8080}"
DB_URL="${DATABASE_URL:?}"  # Even with error message
```

#### JavaScript/TypeScript Modern Patterns
```javascript
// ❌ DISALLOWED: Nullish coalescing
const port = process.env.PORT ?? '8080';

// ❌ DISALLOWED: Destructuring with defaults
const { API_URL = 'http://localhost:3000' } = process.env;

// ❌ DISALLOWED: Logical OR
const config = process.env.CONFIG || 'default';
```

**Why?**
- These patterns hide validation logic
- Defaults are not documented or obvious
- Different team members may not understand the patterns
- Explicit validation is clearer and more maintainable

## Approved Validation Patterns

### Go
```go
// Simple validation
value := os.Getenv("VAR")
if value == "" {
    log.Fatal("VAR is required")
}

// With panic
value := os.Getenv("VAR")
if value == "" {
    panic("VAR is required")
}

// Length check
value := os.Getenv("VAR")
if len(value) == 0 {
    log.Fatalf("VAR is required")
}
```

### Bash
```bash
# Explicit check
if [[ -z "${VAR}" ]]; then
  echo "VAR is required"
  exit 1
fi

# Or with validation function
require_env VAR
```

### JavaScript/TypeScript
```javascript
// Simple check
const value = process.env.VAR;
if (!value) {
  throw new Error('VAR is required');
}

// Undefined check
if (value === undefined) {
  throw new Error('VAR is required');
}

// With length
if (!value?.length) {
  throw new Error('VAR is required');
}
```

## Sensitive Variable Handling

**Additional Rule**: Sensitive variables must NEVER be logged.

**Sensitive Keywords**:
- PASSWORD
- SECRET
- KEY
- TOKEN
- API_KEY
- PRIVATE
- CREDENTIAL
- AUTH
- CERTIFICATE
- CERT

**Example Violations**:
```go
// ❌ SECURITY VIOLATION
apiKey := os.Getenv("API_KEY")
log.Printf("Using API key: %s", apiKey)

// ❌ SECURITY VIOLATION
secret := os.Getenv("SECRET_TOKEN")
fmt.Println("Token:", secret)
```

**Approved**:
```go
// ✅ SAFE: Don't log the value
apiKey := os.Getenv("API_KEY")
if apiKey == "" {
    log.Fatal("API_KEY is required")
}
log.Println("API key loaded successfully")  // Don't print the value!
```

## Multi-Variable Declarations

**Special Handling**: When multiple env vars are assigned in one statement, each must be validated independently.

```go
// Rule tracks BOTH variables
host, port := os.Getenv("HOST"), os.Getenv("PORT")

// ❌ VIOLATION: PORT never validated
if host == "" {
    log.Fatal("HOST required")
}

// ✅ APPROVED: Both validated
if host == "" {
    log.Fatal("HOST required")
}
if port == "" {
    log.Fatal("PORT required")
}
```

## Rationale Summary

This rule is intentionally strict because:

1. **Security**: Unvalidated env vars are a common security vulnerability
2. **Reliability**: Fail-fast prevents runtime errors
3. **Maintainability**: Explicit validation makes code self-documenting
4. **Best Practices**: Encourages proper configuration management
5. **Production Safety**: Prevents silent failures with incorrect defaults

## When to Disable This Rule

This rule should **rarely** be disabled. However, acceptable exceptions include:

1. **Optional configuration** with genuinely safe defaults (document why)
2. **Development-only code** clearly marked as such
3. **Backward compatibility** requirements (with security review)

**How to disable**:
- Add to rule exclusions in scenario configuration
- Document the business justification
- Ensure security review approval

## Related Rules

- `security_hardcoded_secrets`: Prevents hardcoded credentials
- `cors_wildcard`: Prevents overly permissive CORS
- `sql_injection`: Prevents SQL injection
