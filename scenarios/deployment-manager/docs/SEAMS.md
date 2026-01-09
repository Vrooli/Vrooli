# Seams Documentation

This document describes the seams (deliberate boundaries where behavior can be substituted) in the deployment-manager scenario. Seams make the code easier to test, safer to change, and more modular.

## Active Seams

### 1. Domain Error Types (`repository.go`)

**Purpose:** Abstracts database-specific errors into domain-specific errors, preventing database implementation details from leaking into handlers.

**Domain Errors:**
```go
var ErrProfileNotFound = errors.New("profile not found")
```

**Usage:**
- Repository methods return `ErrProfileNotFound` instead of `sql.ErrNoRows`
- Handlers check errors using `errors.Is(err, ErrProfileNotFound)`

**Files using this seam:**
- `repository.go:286` - GetScenarioAndTier returns ErrProfileNotFound
- `handlers_secrets.go:19,75` - handleIdentifySecrets, handleGenerateSecretTemplate
- `handlers_profiles.go:256,328` - handleValidateProfile, handleCostEstimate

**Testing benefit:** Tests don't need to import `database/sql` or know about SQL-specific error types.

---

### 2. Time Provider Seam (`config.go`)

**Purpose:** Allows substitution of time-dependent behavior for testing.

**Interface:**
```go
type TimeProvider interface {
    Now() time.Time
}
```

**Usage:**
- Get current time: `GetTimeProvider().Now()`
- Override in tests by setting `DefaultTimeProvider`

**Files using this seam:**
- `handlers_secrets.go:55,103,137,151,165` - Timestamp responses
- `handlers_profiles.go:63,97,304,350` - Profile ID generation, response timestamps
- `handlers_deployments.go:67` - Deployment status timestamp
- `handlers_telemetry.go:87` - Telemetry event timestamps

**Testing benefit:** Tests can inject a fixed time provider to verify time-dependent behavior deterministically.

---

### 3. HTTP Client Seam (`http_client.go`)

**Purpose:** Allows substitution of HTTP client behavior for testing external service calls.

**Interface:**
```go
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

**Usage:**
- Inject via context: `WithHTTPClient(ctx, mockClient)`
- Retrieve in handlers/services: `GetHTTPClient(ctx).Do(req)`

**Files using this seam:**
- `services_analyzer.go:32` - fetchSkeletonBundle
- `services_secrets_client.go:34` - fetchBundleSecrets

**Testing benefit:** Tests can inject mock HTTP clients to simulate external service responses without network calls.

---

### 4. Profile Repository Seam (`repository.go`)

**Purpose:** Abstracts database operations for profile management, enabling in-memory or mock implementations for testing.

**Interface:**
```go
type ProfileRepository interface {
    List(ctx context.Context) ([]Profile, error)
    Get(ctx context.Context, idOrName string) (*Profile, error)
    Create(ctx context.Context, profile *Profile) (string, error)
    Update(ctx context.Context, idOrName string, updates map[string]interface{}) (*Profile, error)
    Delete(ctx context.Context, idOrName string) (bool, error)
    GetVersions(ctx context.Context, idOrName string) ([]ProfileVersion, error)
    GetScenarioAndTier(ctx context.Context, idOrName string) (string, int, error)
}
```

**Usage:**
- Server struct holds `profiles ProfileRepository`
- Handlers call `s.profiles.Get(...)` instead of direct SQL

**Files using this seam:**
- `handlers_profiles.go` - All profile CRUD handlers
- `handlers_secrets.go` - handleIdentifySecrets, handleGenerateSecretTemplate

**Testing benefit:** Tests can inject mock repositories to verify handler behavior without database setup.

---

### 5. Configuration Resolution Seam (`config.go`)

**Purpose:** Centralizes environment variable resolution for external service URLs and paths.

**Interface:**
```go
type ConfigResolver interface {
    ResolveAnalyzerURL() (string, error)
    ResolveSecretsManagerURL() (string, error)
    ResolveTelemetryDir() string
}
```

**Usage:**
- `GetConfigResolver().ResolveAnalyzerURL()`
- Override `DefaultConfigResolver` for tests

**Files using this seam:**
- `services_analyzer.go:17` - fetchSkeletonBundle
- `services_secrets_client.go:15` - fetchBundleSecrets
- `handlers_dependencies.go:25` - handleAnalyzeDependencies
- `handlers_telemetry.go:34` - telemetryDir

**Testing benefit:** Tests can inject custom config resolvers to control service URLs and paths without environment variables.

---

### 6. Domain Logic Seams (Existing)

These seams separate pure business logic from HTTP handling:

| File | Functions | Purpose |
|------|-----------|---------|
| `domain_deployment.go` | `ValidateDeploymentProfile()`, `GenerateDeploymentID()` | Deployment validation logic |
| `domain_dependencies.go` | `DetectCircularDependencies()`, `CalculateAggregateRequirements()` | Dependency analysis algorithms |
| `domain_templates.go` | `GenerateSecretTemplate()`, `IsEnvFormat()` | Secret template generation |
| `services_fitness.go` | `calculateFitnessScore()`, `getTierName()` | Fitness scoring logic |
| `bundle_validation.go` | `validateDesktopBundleManifestBytes()`, validation helpers | Bundle manifest validation |
| `bundle_mapper.go` | `applyBundleSecrets()` | Secret mapping to manifests |

**Testing benefit:** Domain functions can be unit tested in isolation without HTTP or database setup.

---

## Improvement Opportunities

### Completed in Second Pass

#### 1. Error Type Abstraction ✓
**Status:** Implemented in second seam pass.

- Added `ErrProfileNotFound` domain error in `repository.go`
- Updated `GetScenarioAndTier()` to return domain error instead of `sql.ErrNoRows`
- Updated all handlers to use `errors.Is(err, ErrProfileNotFound)` instead of comparing against `sql.ErrNoRows`

#### 2. Time Seam ✓
**Status:** Implemented in second seam pass.

- Added `TimeProvider` interface and `RealTimeProvider` in `config.go`
- Added `GetTimeProvider()` accessor function
- Updated all handlers to use `GetTimeProvider().Now()` instead of `time.Now()`
- Tests can now inject a mock time provider for deterministic testing

### Medium Priority

#### 3. File System Seam
**Current state:** Direct file operations in telemetry handlers using `os.MkdirAll`, `os.OpenFile`, `os.ReadDir`.

**Recommendation:** Abstract file operations:
```go
type FileSystem interface {
    MkdirAll(path string, perm os.FileMode) error
    OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
    ReadDir(name string) ([]os.DirEntry, error)
}
```

**Files affected:** `handlers_telemetry.go`

#### 4. Command Execution Seam
**Current state:** `exec.Command` called directly in config resolver for `vrooli scenario port` fallback.

**Recommendation:** Abstract command execution:
```go
type CommandRunner interface {
    Output(name string, args ...string) ([]byte, error)
}
```

**Files affected:** `config.go`

### Low Priority

#### 5. JSON Schema Loading Seam
**Current state:** Schema loaded from filesystem using `runtime.Caller` for path resolution.

**Recommendation:** Allow schema injection for tests:
```go
type SchemaLoader interface {
    LoadDesktopBundleSchema() (*jsonschema.Schema, error)
}
```

**Files affected:** `bundle_validation.go`

---

## Testing Patterns

### Using the Domain Error Seam

```go
func TestHandleIdentifySecrets_NotFound(t *testing.T) {
    mockRepo := &MockProfileRepository{
        GetScenarioAndTierFunc: func(ctx context.Context, idOrName string) (string, int, error) {
            return "", 0, ErrProfileNotFound
        },
    }
    server := &Server{profiles: mockRepo}
    // ... test code expects 404 response
}
```

### Using the Time Provider Seam

```go
type MockTimeProvider struct {
    FixedTime time.Time
}

func (m MockTimeProvider) Now() time.Time {
    return m.FixedTime
}

func TestWithFixedTime(t *testing.T) {
    original := DefaultTimeProvider
    defer func() { DefaultTimeProvider = original }()

    fixedTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
    DefaultTimeProvider = MockTimeProvider{FixedTime: fixedTime}

    // ... test code using deterministic time
    // All GetTimeProvider().Now() calls will return fixedTime
}
```

### Using the HTTP Client Seam

```go
func TestFetchSkeletonBundle(t *testing.T) {
    mockClient := &MockHTTPClient{
        DoFunc: func(req *http.Request) (*http.Response, error) {
            return &http.Response{
                StatusCode: 200,
                Body:       io.NopCloser(strings.NewReader(`{"manifest":{}}`)),
            }, nil
        },
    }
    ctx := WithHTTPClient(context.Background(), mockClient)
    // ... test code using ctx
}
```

### Using the Repository Seam

```go
func TestHandleListProfiles(t *testing.T) {
    mockRepo := &MockProfileRepository{
        ListFunc: func(ctx context.Context) ([]Profile, error) {
            return []Profile{{ID: "test-1", Name: "Test"}}, nil
        },
    }
    server := &Server{profiles: mockRepo}
    // ... test code
}
```

### Using the Config Resolver Seam

```go
func TestWithCustomConfig(t *testing.T) {
    original := DefaultConfigResolver
    defer func() { DefaultConfigResolver = original }()

    DefaultConfigResolver = &MockConfigResolver{
        AnalyzerURL: "http://test:8080",
    }
    // ... test code
}
```

---

## Seam Enforcement Guidelines

1. **New external service integrations** must go through the HTTP client seam
2. **New database operations** should use or extend the repository interface
3. **New environment-based configuration** must be added to the ConfigResolver interface
4. **Pure business logic** should be extracted to `domain_*.go` files
5. **Tests should exercise seams** rather than mocking internal implementation details
