# Code Signing Implementation Plan

> **Status**: Phase 5 Complete - All Phases Done
> **Created**: 2025-12-09
> **Target**: deployment-manager v1.1
> **Complexity**: High
> **Estimated Effort**: 5 phases over multiple sessions
> **Progress**: Phase 1 ✓ | Phase 2 ✓ | Phase 3 ✓ | Phase 4 ✓ | Phase 5 ✓

## Executive Summary

This plan implements end-to-end code signing integration for bundled desktop deployments. When complete, the deployment-manager will:

1. Store platform-specific signing configuration in deployment profiles
2. Validate signing prerequisites before packaging
3. Generate properly-configured electron-builder settings
4. Integrate with the secrets system for secure credential handling
5. Support macOS notarization automation

**Why this matters**: Unsigned desktop apps trigger OS security warnings that block adoption. Enterprise environments often reject unsigned executables entirely. Code signing is required for professional desktop distribution.

---

## Architecture Principles

### Screaming Architecture

The codebase structure will immediately communicate "this handles code signing":

```
api/
  codesigning/                    # Top-level package - unmistakable purpose
    ├── types.go                  # Domain types (SigningConfig, platform configs)
    ├── config.go                 # Configuration loading and defaults
    ├── handler.go                # HTTP handlers
    ├── handler_test.go
    ├── repository.go             # Storage interface
    ├── sql_repository.go         # SQL implementation
    ├── sql_repository_test.go
    ├── validation/               # Subpackage for validation concerns
    │   ├── validator.go          # Core validation interface and implementation
    │   ├── validator_test.go
    │   ├── prerequisites.go      # Tool and certificate detection
    │   ├── prerequisites_test.go
    │   └── rules.go              # Business validation rules
    ├── generation/               # Subpackage for config generation
    │   ├── generator.go          # Build config generation interface
    │   ├── generator_test.go
    │   ├── electron_builder.go   # electron-builder specific generation
    │   └── entitlements.go       # macOS entitlements generation
    └── platforms/                # Platform-specific implementations
        ├── detector.go           # Interface for platform detection
        ├── windows.go            # Windows Authenticode specifics
        ├── windows_test.go
        ├── macos.go              # macOS codesign/notarization specifics
        ├── macos_test.go
        ├── linux.go              # Linux GPG specifics
        └── linux_test.go
```

### Boundary of Responsibility Enforcement

Each component has a single, well-defined responsibility:

| Component | Responsibility | Does NOT Do |
|-----------|---------------|-------------|
| `types.go` | Define domain structures | Validation, persistence, I/O |
| `handler.go` | HTTP request/response | Business logic, storage |
| `repository.go` | Data persistence abstraction | HTTP, validation |
| `validation/validator.go` | Structural validation | File I/O, external calls |
| `validation/prerequisites.go` | External tool/cert detection | Validation rules, storage |
| `generation/generator.go` | Build config creation | Signing execution, HTTP |
| `platforms/*.go` | Platform-specific logic | Cross-platform concerns |

### Testing Seams

Every external dependency is abstracted behind an interface:

```go
// FileSystem - abstracts all file operations
type FileSystem interface {
    Exists(path string) bool
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte, perm os.FileMode) error
    Stat(path string) (os.FileInfo, error)
    MkdirAll(path string, perm os.FileMode) error
}

// CommandRunner - abstracts external command execution
type CommandRunner interface {
    Run(ctx context.Context, name string, args ...string) (stdout []byte, stderr []byte, err error)
    LookPath(name string) (string, error)
}

// CertificateParser - abstracts certificate reading
type CertificateParser interface {
    ParsePKCS12(data []byte, password string) (*CertificateInfo, error)
    ParsePEM(data []byte) (*CertificateInfo, error)
    ValidateChain(cert *CertificateInfo) error
}

// TimeProvider - abstracts time for expiration testing
type TimeProvider interface {
    Now() time.Time
}

// EnvironmentReader - abstracts environment variable access
type EnvironmentReader interface {
    GetEnv(key string) string
    LookupEnv(key string) (string, bool)
}
```

**Benefits**:
- Unit tests use mock implementations
- No file system access in tests
- No external command execution in tests
- Deterministic time-based tests
- Full coverage without integration complexity

---

## Domain Model

### Core Types

```go
// SigningConfig is the top-level code signing configuration.
// It contains platform-specific settings and controls whether signing is enabled.
type SigningConfig struct {
    // Enabled controls whether code signing is applied during packaging.
    // When false, all platform configs are ignored.
    Enabled bool `json:"enabled"`

    // Windows contains Authenticode signing configuration.
    Windows *WindowsSigningConfig `json:"windows,omitempty"`

    // MacOS contains Apple code signing and notarization configuration.
    MacOS *MacOSSigningConfig `json:"macos,omitempty"`

    // Linux contains GPG signing configuration.
    Linux *LinuxSigningConfig `json:"linux,omitempty"`
}

// WindowsSigningConfig contains Windows Authenticode signing settings.
type WindowsSigningConfig struct {
    // CertificateSource specifies how the certificate is provided.
    // Values: "file", "store", "azure_keyvault", "aws_kms"
    CertificateSource string `json:"certificate_source"`

    // CertificateFile is the path to the .pfx/.p12 certificate file.
    // Required when CertificateSource is "file".
    CertificateFile string `json:"certificate_file,omitempty"`

    // CertificatePasswordEnv is the environment variable containing the certificate password.
    // The actual password is never stored in the config.
    CertificatePasswordEnv string `json:"certificate_password_env,omitempty"`

    // CertificateThumbprint identifies the certificate in Windows Certificate Store.
    // Required when CertificateSource is "store".
    CertificateThumbprint string `json:"certificate_thumbprint,omitempty"`

    // TimestampServer is the RFC 3161 timestamp server URL.
    // Recommended: "http://timestamp.digicert.com" or "http://timestamp.sectigo.com"
    TimestampServer string `json:"timestamp_server,omitempty"`

    // SignAlgorithm specifies the digest algorithm.
    // Values: "sha256" (recommended), "sha384", "sha512"
    SignAlgorithm string `json:"sign_algorithm,omitempty"`

    // DualSign enables SHA-1 + SHA-256 dual signing for Windows 7 compatibility.
    DualSign bool `json:"dual_sign,omitempty"`
}

// MacOSSigningConfig contains Apple code signing and notarization settings.
type MacOSSigningConfig struct {
    // Identity is the signing identity (e.g., "Developer ID Application: Name (TEAMID)").
    // Can be the full name or just the Team ID.
    Identity string `json:"identity"`

    // TeamID is the Apple Developer Team ID (10-character alphanumeric).
    TeamID string `json:"team_id"`

    // HardenedRuntime enables the hardened runtime, required for notarization.
    HardenedRuntime bool `json:"hardened_runtime"`

    // Notarize enables Apple notarization for Gatekeeper approval.
    // Requires AppleIDEnv and AppleIDPasswordEnv (or AppleAPIKey*).
    Notarize bool `json:"notarize"`

    // EntitlementsFile is the path to the entitlements.plist file.
    // Required for apps that need specific entitlements (JIT, unsigned memory, etc.)
    EntitlementsFile string `json:"entitlements_file,omitempty"`

    // ProvisioningProfile is the path to the .provisionprofile file.
    // Required for Mac App Store or enterprise distribution.
    ProvisioningProfile string `json:"provisioning_profile,omitempty"`

    // --- Notarization Credentials (App-Specific Password method) ---

    // AppleIDEnv is the environment variable containing the Apple ID email.
    AppleIDEnv string `json:"apple_id_env,omitempty"`

    // AppleIDPasswordEnv is the environment variable containing the app-specific password.
    AppleIDPasswordEnv string `json:"apple_id_password_env,omitempty"`

    // --- Notarization Credentials (API Key method - preferred) ---

    // AppleAPIKeyID is the API Key ID from App Store Connect.
    AppleAPIKeyID string `json:"apple_api_key_id,omitempty"`

    // AppleAPIKeyFile is the path to the .p8 API key file.
    AppleAPIKeyFile string `json:"apple_api_key_file,omitempty"`

    // AppleAPIIssuerID is the Issuer ID from App Store Connect.
    AppleAPIIssuerID string `json:"apple_api_issuer_id,omitempty"`
}

// LinuxSigningConfig contains Linux GPG signing settings.
type LinuxSigningConfig struct {
    // GPGKeyID is the GPG key ID or fingerprint for signing.
    GPGKeyID string `json:"gpg_key_id,omitempty"`

    // GPGPassphraseEnv is the environment variable containing the GPG passphrase.
    GPGPassphraseEnv string `json:"gpg_passphrase_env,omitempty"`

    // GPGHomedir overrides the default GPG home directory.
    GPGHomedir string `json:"gpg_homedir,omitempty"`
}
```

### Validation Result Types

```go
// ValidationResult contains the outcome of signing configuration validation.
type ValidationResult struct {
    Valid       bool                         `json:"valid"`
    Platforms   map[string]PlatformValidation `json:"platforms"`
    Errors      []ValidationError            `json:"errors"`
    Warnings    []ValidationWarning          `json:"warnings"`
}

// PlatformValidation contains validation results for a specific platform.
type PlatformValidation struct {
    Configured    bool              `json:"configured"`
    ToolInstalled bool              `json:"tool_installed"`
    ToolPath      string            `json:"tool_path,omitempty"`
    ToolVersion   string            `json:"tool_version,omitempty"`
    Certificate   *CertificateInfo  `json:"certificate,omitempty"`
    Errors        []string          `json:"errors"`
    Warnings      []string          `json:"warnings"`
}

// CertificateInfo contains parsed certificate details.
type CertificateInfo struct {
    Subject      string    `json:"subject"`
    Issuer       string    `json:"issuer"`
    SerialNumber string    `json:"serial_number"`
    NotBefore    time.Time `json:"not_before"`
    NotAfter     time.Time `json:"not_after"`
    IsExpired    bool      `json:"is_expired"`
    DaysToExpiry int       `json:"days_to_expiry"`
    KeyUsage     []string  `json:"key_usage"`
    IsCodeSign   bool      `json:"is_code_sign"`
}

// ValidationError represents a blocking validation issue.
type ValidationError struct {
    Code        string `json:"code"`
    Platform    string `json:"platform"`
    Field       string `json:"field,omitempty"`
    Message     string `json:"message"`
    Remediation string `json:"remediation"`
}

// ValidationWarning represents a non-blocking validation issue.
type ValidationWarning struct {
    Code     string `json:"code"`
    Platform string `json:"platform"`
    Message  string `json:"message"`
}
```

### Generated Config Types

```go
// ElectronBuilderSigningConfig is the signing portion of electron-builder config.
type ElectronBuilderSigningConfig struct {
    Win *ElectronBuilderWinSigning `json:"win,omitempty"`
    Mac *ElectronBuilderMacSigning `json:"mac,omitempty"`
}

// ElectronBuilderWinSigning maps to electron-builder's win signing config.
type ElectronBuilderWinSigning struct {
    CertificateFile          string `json:"certificateFile,omitempty"`
    CertificatePassword      string `json:"certificatePassword,omitempty"`
    CertificateSubjectName   string `json:"certificateSubjectName,omitempty"`
    CertificateSha1          string `json:"certificateSha1,omitempty"`
    SignAndEditExecutable    bool   `json:"signAndEditExecutable"`
    SignDlls                 bool   `json:"signDlls"`
    TimeStampServer          string `json:"timeStampServer,omitempty"`
    Rfc3161TimeStampServer   string `json:"rfc3161TimeStampServer,omitempty"`
    SigningHashAlgorithms    []string `json:"signingHashAlgorithms,omitempty"`
}

// ElectronBuilderMacSigning maps to electron-builder's mac signing config.
type ElectronBuilderMacSigning struct {
    Identity                string `json:"identity,omitempty"`
    HardenedRuntime         bool   `json:"hardenedRuntime"`
    GatekeeperAssess        bool   `json:"gatekeeperAssess"`
    Entitlements            string `json:"entitlements,omitempty"`
    EntitlementsInherit     string `json:"entitlementsInherit,omitempty"`
    ProvisioningProfile     string `json:"provisioningProfile,omitempty"`
    Notarize                interface{} `json:"notarize,omitempty"` // bool or NotarizeConfig
}
```

---

## Implementation Phases

### Phase 1: Foundation (Types & Schema)

**Goal**: Establish the type system and schema without any runtime behavior.

**Deliverables**:

1. **`api/codesigning/types.go`**
   - All domain types defined above
   - JSON struct tags for serialization
   - Comprehensive godoc comments

2. **`api/codesigning/config.go`**
   - Default values and constants
   - Configuration loading helpers
   - Environment variable resolution (via interface)

3. **Update JSON Schema** (`docs/deployment/bundle-schema.desktop.v0.1.json`)
   - Add `code_signing` object to schema
   - Define all nested platform schemas
   - Add validation constraints (patterns, enums)

4. **Update bundle types** (`api/bundles/types.go`)
   - Add `CodeSigning *codesigning.SigningConfig` to Manifest

**Testing**:
- Unit tests for type serialization/deserialization
- Schema validation tests with valid/invalid examples
- Round-trip tests (Go struct → JSON → Go struct)

**Files Created/Modified**:
```
api/codesigning/types.go          # NEW
api/codesigning/types_test.go     # NEW
api/codesigning/config.go         # NEW
api/codesigning/config_test.go    # NEW
api/bundles/types.go              # MODIFIED
docs/deployment/bundle-schema.desktop.v0.1.json  # MODIFIED
```

---

### Phase 2: Validation Layer

**Goal**: Implement comprehensive validation without external dependencies.

**Deliverables**:

1. **`api/codesigning/validation/validator.go`**
   ```go
   // Validator validates signing configurations.
   type Validator interface {
       // ValidateConfig checks structural validity of a SigningConfig.
       ValidateConfig(config *SigningConfig) *ValidationResult

       // ValidateForPlatform checks config is valid for a specific target platform.
       ValidateForPlatform(config *SigningConfig, platform string) *ValidationResult
   }

   // NewValidator creates a validator with the given dependencies.
   func NewValidator(opts ...ValidatorOption) Validator
   ```

2. **`api/codesigning/validation/rules.go`**
   - Rule: Windows config requires certificate source
   - Rule: Windows file source requires certificate_file
   - Rule: macOS notarization requires credentials
   - Rule: macOS team_id must be 10 alphanumeric characters
   - Rule: TimestampServer must be valid URL
   - Rule: Environment variable names must be valid identifiers

3. **`api/codesigning/validation/prerequisites.go`**
   ```go
   // PrerequisiteChecker verifies external signing prerequisites.
   type PrerequisiteChecker interface {
       // CheckPrerequisites validates tools and certificates are available.
       CheckPrerequisites(ctx context.Context, config *SigningConfig) *ValidationResult
   }

   // NewPrerequisiteChecker creates a checker with injected dependencies.
   func NewPrerequisiteChecker(
       fs FileSystem,
       cmd CommandRunner,
       certs CertificateParser,
       env EnvironmentReader,
       time TimeProvider,
   ) PrerequisiteChecker
   ```

4. **Prerequisite checks implemented**:
   - Windows: `signtool.exe` in PATH or Windows SDK
   - Windows: Certificate file exists and is parseable
   - Windows: Certificate is valid for code signing
   - Windows: Certificate not expired (warn if <30 days)
   - macOS: `codesign` available (comes with Xcode)
   - macOS: `xcrun notarytool` available (for notarization)
   - macOS: Signing identity exists in keychain
   - Linux: `gpg` available
   - Linux: GPG key exists in keyring

**Testing**:
- Unit tests with mock FileSystem, CommandRunner, etc.
- Test each validation rule independently
- Test prerequisite detection with mocked tool responses
- Test certificate expiration with mocked TimeProvider

**Files Created**:
```
api/codesigning/validation/validator.go          # NEW
api/codesigning/validation/validator_test.go     # NEW
api/codesigning/validation/rules.go              # NEW
api/codesigning/validation/rules_test.go         # NEW
api/codesigning/validation/prerequisites.go      # NEW
api/codesigning/validation/prerequisites_test.go # NEW
api/codesigning/interfaces.go                    # NEW (shared interfaces)
api/codesigning/mocks/mocks.go                   # NEW (test mocks)
```

---

### Phase 3: Storage & API

**Goal**: Persist signing configurations and expose via REST API.

**Deliverables**:

1. **`api/codesigning/repository.go`**
   ```go
   // Repository persists signing configurations.
   type Repository interface {
       // Get retrieves the signing config for a profile.
       Get(ctx context.Context, profileID string) (*SigningConfig, error)

       // Save stores or updates the signing config for a profile.
       Save(ctx context.Context, profileID string, config *SigningConfig) error

       // Delete removes the signing config for a profile.
       Delete(ctx context.Context, profileID string) error

       // GetForPlatform retrieves config for a specific platform only.
       GetForPlatform(ctx context.Context, profileID string, platform string) (interface{}, error)

       // SaveForPlatform updates only a specific platform's config.
       SaveForPlatform(ctx context.Context, profileID string, platform string, config interface{}) error
   }
   ```

2. **`api/codesigning/sql_repository.go`**
   - Store signing config as JSONB in profiles table (new column)
   - Or separate `profile_signing_configs` table for isolation
   - Implement all Repository methods

3. **`api/codesigning/handler.go`**
   ```go
   // Handler handles code signing HTTP requests.
   type Handler struct {
       repo      Repository
       validator validation.Validator
       checker   validation.PrerequisiteChecker
       log       func(string, map[string]interface{})
   }

   // Routes:
   // GET    /api/v1/profiles/{id}/signing           - Get signing config
   // PUT    /api/v1/profiles/{id}/signing           - Update full config
   // PATCH  /api/v1/profiles/{id}/signing/{platform} - Update platform config
   // DELETE /api/v1/profiles/{id}/signing           - Remove all signing config
   // DELETE /api/v1/profiles/{id}/signing/{platform} - Remove platform config
   // POST   /api/v1/profiles/{id}/signing/validate  - Validate prerequisites
   // GET    /api/v1/signing/prerequisites           - Check available tools (no profile)
   ```

4. **Database migration**
   ```sql
   -- Option A: Add column to profiles
   ALTER TABLE profiles ADD COLUMN signing_config JSONB DEFAULT NULL;

   -- Option B: Separate table (better for audit/versioning)
   CREATE TABLE profile_signing_configs (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
       config JSONB NOT NULL,
       created_at TIMESTAMPTZ DEFAULT NOW(),
       updated_at TIMESTAMPTZ DEFAULT NOW(),
       updated_by TEXT DEFAULT 'system',
       UNIQUE(profile_id)
   );
   ```

5. **CLI Commands**
   ```bash
   # Configure signing
   deployment-manager signing set <profile> --platform windows \
       --cert ./cert.pfx \
       --password-env WIN_CERT_PASSWORD \
       --timestamp http://timestamp.digicert.com

   deployment-manager signing set <profile> --platform macos \
       --identity "Developer ID Application: Vrooli (TEAMID)" \
       --team-id TEAMID \
       --hardened-runtime \
       --notarize \
       --apple-id-env APPLE_ID \
       --apple-password-env APPLE_APP_PASSWORD

   # View config
   deployment-manager signing show <profile>
   deployment-manager signing show <profile> --platform windows

   # Validate prerequisites
   deployment-manager signing validate <profile>
   deployment-manager signing prerequisites  # No profile, just check tools

   # Remove config
   deployment-manager signing remove <profile>
   deployment-manager signing remove <profile> --platform macos
   ```

**Testing**:
- Repository tests with test database
- Handler tests with mock repository
- CLI tests with mock API client
- Integration test: full round-trip

**Files Created/Modified**:
```
api/codesigning/repository.go          # NEW
api/codesigning/sql_repository.go      # NEW
api/codesigning/sql_repository_test.go # NEW
api/codesigning/handler.go             # NEW
api/codesigning/handler_test.go        # NEW
api/server/routes.go                   # MODIFIED (add signing routes)
api/server/server.go                   # MODIFIED (wire handler)
cli/signing/commands.go                # NEW
cli/signing/commands_test.go           # NEW
migrations/NNNN_add_signing_config.sql # NEW
```

---

### Phase 4: Generation Layer

**Goal**: Generate electron-builder and platform-specific configs from signing settings.

**Deliverables**:

1. **`api/codesigning/generation/generator.go`**
   ```go
   // Generator creates build tool configurations from signing settings.
   type Generator interface {
       // GenerateElectronBuilder creates electron-builder signing config.
       GenerateElectronBuilder(config *SigningConfig) (*ElectronBuilderSigningConfig, error)

       // GenerateEntitlements creates macOS entitlements.plist content.
       GenerateEntitlements(config *MacOSSigningConfig, capabilities []string) ([]byte, error)

       // GenerateNotarizeScript creates notarization afterSign script.
       GenerateNotarizeScript(config *MacOSSigningConfig) ([]byte, error)
   }
   ```

2. **`api/codesigning/generation/electron_builder.go`**
   - Map SigningConfig → electron-builder JSON structure
   - Handle environment variable references (`${ENV_VAR}`)
   - Generate both `win` and `mac` sections

3. **`api/codesigning/generation/entitlements.go`**
   - Generate valid entitlements.plist XML
   - Include standard entitlements for Electron apps
   - Support custom entitlements (JIT, unsigned executable memory, etc.)

4. **`api/codesigning/generation/notarize_script.go`**
   - Generate `afterSign` JavaScript for electron-builder
   - Support both credential methods (app password vs API key)
   - Include proper error handling and logging

5. **Integration with bundle assembly**
   - `POST /api/v1/bundles/assemble` includes signing config
   - `POST /api/v1/bundles/export` embeds generated electron-builder config
   - New endpoint: `POST /api/v1/bundles/signing-config` returns just signing portion

**Example generated electron-builder config**:
```json
{
  "win": {
    "certificateFile": "./certs/windows.pfx",
    "certificatePassword": "${WIN_CERT_PASSWORD}",
    "signAndEditExecutable": true,
    "signDlls": true,
    "rfc3161TimeStampServer": "http://timestamp.digicert.com",
    "signingHashAlgorithms": ["sha256"]
  },
  "mac": {
    "identity": "Developer ID Application: Vrooli (ABC123XYZ)",
    "hardenedRuntime": true,
    "gatekeeperAssess": true,
    "entitlements": "build/entitlements.mac.plist",
    "entitlementsInherit": "build/entitlements.mac.plist"
  },
  "afterSign": "scripts/notarize.js"
}
```

**Example generated entitlements.plist**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.cs.allow-jit</key>
    <true/>
    <key>com.apple.security.cs.allow-unsigned-executable-memory</key>
    <true/>
    <key>com.apple.security.cs.disable-library-validation</key>
    <true/>
    <key>com.apple.security.automation.apple-events</key>
    <true/>
</dict>
</plist>
```

**Testing**:
- Unit tests for each generation function
- Snapshot tests comparing generated output to expected
- Validation that generated config is valid electron-builder schema
- Integration test: generate → write → verify parseable

**Files Created/Modified**:
```
api/codesigning/generation/generator.go          # NEW - Generator interface and DefaultGenerator
api/codesigning/generation/generator_test.go     # NEW - Comprehensive tests
api/codesigning/generation/electron_builder.go   # NEW - electron-builder config generation
api/codesigning/generation/entitlements.go       # NEW - macOS entitlements.plist generation
api/codesigning/generation/notarize_script.go    # NEW - afterSign JavaScript generation
api/bundles/handler.go                           # MODIFIED - Added GenerateSigningConfig endpoint
api/server/routes.go                             # MODIFIED - Added /api/v1/bundles/signing-config route
```

---

### Phase 5: Platform Integration

**Goal**: Implement platform-specific tool invocation and verification.

**Deliverables**:

1. **`api/codesigning/platforms/detector.go`**
   ```go
   // PlatformDetector detects signing tools and capabilities.
   type PlatformDetector interface {
       // DetectTools returns available signing tools for the current platform.
       DetectTools(ctx context.Context) (*ToolDetectionResult, error)

       // DetectCertificates finds available signing certificates.
       DetectCertificates(ctx context.Context, platform string) ([]CertificateInfo, error)
   }
   ```

2. **`api/codesigning/platforms/windows.go`**
   - Detect signtool.exe in Windows SDK paths
   - Parse PFX/P12 certificates
   - Validate certificate is code-signing capable
   - List certificates in Windows Certificate Store

3. **`api/codesigning/platforms/macos.go`**
   - Verify codesign and notarytool availability
   - Query keychain for signing identities
   - Validate identity format
   - Check notarization credentials

4. **`api/codesigning/platforms/linux.go`**
   - Verify gpg availability
   - List available GPG keys
   - Validate key can sign

5. **Pre-deployment validation integration**
   - Add `SigningValidation` check to validation pipeline
   - Include certificate expiry warnings
   - Block deploy if signing configured but prerequisites missing

**CLI Enhancement**:
```bash
# Discover available certificates
deployment-manager signing discover --platform windows
deployment-manager signing discover --platform macos

# Output:
# Windows Certificates Found:
#   1. CN=Vrooli Inc, O=Vrooli Inc (Thumbprint: ABC123...)
#      Expires: 2026-03-15 (467 days remaining)
#      Key Usage: Code Signing ✓
#
# macOS Identities Found:
#   1. Developer ID Application: Vrooli (ABC123XYZ)
#      Expires: 2025-11-20 (346 days remaining)
#   2. Apple Development: developer@vrooli.com (XYZ789)
#      Expires: 2025-06-01 (174 days remaining)
```

**Testing**:
- Unit tests with mocked command execution
- Platform-specific tests (run only on appropriate OS)
- Certificate parsing tests with test certificates
- Integration tests with real tools (CI matrix)

**Files Created**:
```
api/codesigning/platforms/detector.go          # NEW
api/codesigning/platforms/detector_test.go     # NEW
api/codesigning/platforms/windows.go           # NEW
api/codesigning/platforms/windows_test.go      # NEW
api/codesigning/platforms/macos.go             # NEW
api/codesigning/platforms/macos_test.go        # NEW
api/codesigning/platforms/linux.go             # NEW
api/codesigning/platforms/linux_test.go        # NEW
api/codesigning/platforms/testdata/            # Test certificates
```

---

## API Reference

### Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/profiles/{id}/signing` | Get signing configuration |
| PUT | `/api/v1/profiles/{id}/signing` | Replace full signing configuration |
| PATCH | `/api/v1/profiles/{id}/signing/{platform}` | Update platform-specific config |
| DELETE | `/api/v1/profiles/{id}/signing` | Remove all signing configuration |
| DELETE | `/api/v1/profiles/{id}/signing/{platform}` | Remove platform-specific config |
| POST | `/api/v1/profiles/{id}/signing/validate` | Validate signing prerequisites |
| GET | `/api/v1/signing/prerequisites` | Check available tools (no profile) |
| GET | `/api/v1/signing/discover/{platform}` | Discover available certificates |

### Request/Response Examples

**PUT /api/v1/profiles/{id}/signing**
```json
{
  "enabled": true,
  "windows": {
    "certificate_source": "file",
    "certificate_file": "./certs/windows.pfx",
    "certificate_password_env": "WIN_CERT_PASSWORD",
    "timestamp_server": "http://timestamp.digicert.com",
    "sign_algorithm": "sha256"
  },
  "macos": {
    "identity": "Developer ID Application: Vrooli (ABC123XYZ)",
    "team_id": "ABC123XYZ",
    "hardened_runtime": true,
    "notarize": true,
    "apple_api_key_id": "KEYID123",
    "apple_api_key_file": "./certs/AuthKey_KEYID123.p8",
    "apple_api_issuer_id": "issuer-uuid"
  }
}
```

**POST /api/v1/profiles/{id}/signing/validate**
```json
{
  "valid": false,
  "platforms": {
    "windows": {
      "configured": true,
      "tool_installed": true,
      "tool_path": "C:\\Program Files (x86)\\Windows Kits\\10\\bin\\x64\\signtool.exe",
      "tool_version": "10.0.22000.0",
      "certificate": {
        "subject": "CN=Vrooli Inc, O=Vrooli Inc",
        "issuer": "CN=DigiCert Code Signing CA",
        "not_after": "2026-03-15T00:00:00Z",
        "is_expired": false,
        "days_to_expiry": 467,
        "is_code_sign": true
      },
      "errors": [],
      "warnings": []
    },
    "macos": {
      "configured": true,
      "tool_installed": true,
      "tool_path": "/usr/bin/codesign",
      "errors": ["Identity 'Developer ID Application: Vrooli (ABC123XYZ)' not found in keychain"],
      "warnings": []
    }
  },
  "errors": [
    {
      "code": "MACOS_IDENTITY_NOT_FOUND",
      "platform": "macos",
      "field": "identity",
      "message": "Signing identity not found in keychain",
      "remediation": "Import the Developer ID certificate into your login keychain, or run 'security find-identity -v -p codesigning' to list available identities"
    }
  ],
  "warnings": []
}
```

---

## CLI Reference

### Commands Summary

```bash
# Configuration
deployment-manager signing set <profile> --platform <platform> [options]
deployment-manager signing show <profile> [--platform <platform>]
deployment-manager signing remove <profile> [--platform <platform>]

# Validation
deployment-manager signing validate <profile>
deployment-manager signing prerequisites

# Discovery
deployment-manager signing discover --platform <platform>

# Generation (for manual use)
deployment-manager signing generate-config <profile> --output electron-builder.json
deployment-manager signing generate-entitlements <profile> --output entitlements.plist
```

### Examples

```bash
# Set up Windows signing
deployment-manager signing set my-profile --platform windows \
    --cert "./certs/code-signing.pfx" \
    --password-env "WIN_CSC_KEY_PASSWORD" \
    --timestamp "http://timestamp.digicert.com"

# Set up macOS signing with API key (preferred)
deployment-manager signing set my-profile --platform macos \
    --identity "Developer ID Application: My Company (TEAMID)" \
    --team-id "TEAMID" \
    --hardened-runtime \
    --notarize \
    --api-key-id "KEYID" \
    --api-key-file "./certs/AuthKey.p8" \
    --api-issuer "ISSUER-UUID"

# Validate before building
deployment-manager signing validate my-profile

# Output:
# ✓ Windows: Ready
#   Certificate: CN=My Company (expires in 467 days)
#   Tool: signtool.exe v10.0.22000.0
#
# ✗ macOS: Configuration Error
#   Error: Identity not found in keychain
#   Remediation: Import Developer ID certificate or check identity name
#
# Validation: FAILED (1 error)
```

---

## Error Codes

| Code | Platform | Description |
|------|----------|-------------|
| `SIGNING_DISABLED` | all | Signing is disabled but validation requested |
| `PLATFORM_NOT_CONFIGURED` | all | No config for requested platform |
| `WIN_SIGNTOOL_NOT_FOUND` | windows | signtool.exe not in PATH or SDK |
| `WIN_CERT_FILE_NOT_FOUND` | windows | Certificate file doesn't exist |
| `WIN_CERT_INVALID` | windows | Certificate file is corrupted or wrong format |
| `WIN_CERT_NOT_CODE_SIGN` | windows | Certificate missing code signing EKU |
| `WIN_CERT_EXPIRED` | windows | Certificate has expired |
| `WIN_CERT_PASSWORD_MISSING` | windows | Password env var not set |
| `WIN_CERT_PASSWORD_WRONG` | windows | Password doesn't decrypt certificate |
| `MACOS_CODESIGN_NOT_FOUND` | macos | codesign not available |
| `MACOS_NOTARYTOOL_NOT_FOUND` | macos | notarytool not available (Xcode 13+) |
| `MACOS_IDENTITY_NOT_FOUND` | macos | Signing identity not in keychain |
| `MACOS_IDENTITY_EXPIRED` | macos | Signing identity has expired |
| `MACOS_TEAM_ID_INVALID` | macos | Team ID format invalid |
| `MACOS_NOTARIZE_CREDS_MISSING` | macos | Notarization requires credentials |
| `LINUX_GPG_NOT_FOUND` | linux | gpg not available |
| `LINUX_KEY_NOT_FOUND` | linux | GPG key not in keyring |

---

## Security Considerations

### Secrets Handling

1. **Never store passwords in config** - Only environment variable references
2. **Never log passwords** - Sanitize all log output
3. **Memory clearing** - Zero password buffers after use (where possible in Go)
4. **Audit trail** - Log signing config changes (without secrets)

### Certificate Security

1. **File permissions** - Warn if certificate files are world-readable
2. **Expiration monitoring** - Warn when certificates expire within 30 days
3. **Revocation checking** - Future: check CRL/OCSP for revoked certs

### API Security

1. **Authentication required** - All signing endpoints require valid auth token
2. **Profile ownership** - Users can only access their own profiles' signing config
3. **Rate limiting** - Validation endpoints should be rate-limited (expensive operations)

---

## Testing Strategy

### Unit Tests

Every component has comprehensive unit tests using mocked dependencies:

```go
func TestValidator_ValidateConfig_WindowsRequiresCertSource(t *testing.T) {
    v := NewValidator()
    config := &SigningConfig{
        Enabled: true,
        Windows: &WindowsSigningConfig{
            // Missing CertificateSource
            CertificateFile: "./cert.pfx",
        },
    }

    result := v.ValidateConfig(config)

    assert.False(t, result.Valid)
    assert.Contains(t, result.Errors, ValidationError{
        Code:     "WIN_CERT_SOURCE_REQUIRED",
        Platform: "windows",
        Field:    "certificate_source",
        Message:  "Windows signing requires certificate_source",
    })
}
```

### Integration Tests

Test the full flow with a real database and mocked external tools:

```go
func TestSigningFlow_ConfigureValidateGenerate(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Create profile
    profileID := createTestProfile(t, db)

    // Configure signing
    client := NewTestClient(t)
    err := client.SetSigning(profileID, testSigningConfig)
    require.NoError(t, err)

    // Validate
    result, err := client.ValidateSigning(profileID)
    require.NoError(t, err)
    assert.True(t, result.Valid)

    // Generate
    config, err := client.GenerateElectronBuilder(profileID)
    require.NoError(t, err)
    assert.NotNil(t, config.Win)
    assert.Equal(t, "./cert.pfx", config.Win.CertificateFile)
}
```

### Platform Tests

Tests that require actual platform tools run in CI matrix:

```go
//go:build darwin

func TestMacOS_DetectIdentities(t *testing.T) {
    if os.Getenv("CI") == "" {
        t.Skip("Skipping macOS identity detection outside CI")
    }

    detector := platforms.NewMacOSDetector(
        realFileSystem{},
        realCommandRunner{},
    )

    identities, err := detector.ListIdentities(context.Background())
    require.NoError(t, err)
    // CI should have at least the self-signed test identity
    assert.NotEmpty(t, identities)
}
```

### Test Fixtures

```
api/codesigning/testdata/
├── certs/
│   ├── valid-codesign.pfx       # Valid code signing cert (password: "test")
│   ├── expired-codesign.pfx     # Expired cert for testing
│   ├── no-codesign-eku.pfx      # Cert without code signing EKU
│   └── corrupted.pfx            # Invalid file
├── configs/
│   ├── valid-full.json          # Complete valid config
│   ├── valid-windows-only.json  # Windows only
│   ├── invalid-missing-field.json
│   └── invalid-bad-team-id.json
└── expected/
    ├── electron-builder-full.json
    ├── entitlements-default.plist
    └── notarize-script.js
```

---

## Migration & Rollout

### Database Migration

```sql
-- migrations/NNNN_add_signing_config.sql

-- Up
ALTER TABLE profiles ADD COLUMN signing_config JSONB DEFAULT NULL;

CREATE INDEX idx_profiles_signing_enabled
    ON profiles ((signing_config->>'enabled'))
    WHERE signing_config IS NOT NULL;

-- Down
DROP INDEX IF EXISTS idx_profiles_signing_enabled;
ALTER TABLE profiles DROP COLUMN IF EXISTS signing_config;
```

### Feature Flag

During rollout, gate new functionality:

```go
const FeatureCodeSigning = "code_signing"

func (h *Handler) GetSigning(w http.ResponseWriter, r *http.Request) {
    if !features.IsEnabled(FeatureCodeSigning) {
        http.Error(w, `{"error":"code signing feature not enabled"}`, http.StatusNotImplemented)
        return
    }
    // ... rest of handler
}
```

### Rollout Phases

1. **Alpha**: Feature flag on for internal testing only
2. **Beta**: Enable for opt-in users, gather feedback
3. **GA**: Remove feature flag, enable for all

---

## Documentation Updates

### Files to Create

```
docs/guides/code-signing.md           # Comprehensive guide
docs/api/signing.md                   # API reference
docs/cli/signing-commands.md          # CLI reference
docs/guides/certificates/             # Platform-specific cert guides
  ├── windows-authenticode.md
  ├── macos-developer-id.md
  ├── macos-notarization.md
  └── linux-gpg.md
```

### Files to Update

```
docs/guides/bundle-manifest-schema.md  # Add code_signing section
docs/workflows/desktop-deployment.md   # Add signing steps
docs/ROADMAP.md                        # Mark as complete
docs/README.md                         # Add to table of contents
```

---

## Success Criteria

### Phase 1 Complete When

- [x] All types compile and serialize correctly
- [x] JSON Schema updated and validates correctly
- [x] 100% test coverage on types
- [x] Bundle manifest accepts code_signing field

### Phase 2 Complete When

- [x] All validation rules implemented
- [x] Prerequisite checks work on all platforms
- [x] Clear error messages with remediation steps
- [x] 100% test coverage with mocks

### Phase 3 Complete When

- [x] Signing config persists to database
- [x] All API endpoints functional
- [x] CLI commands working
- [x] Integration tests pass

### Phase 4 Complete When

- [x] electron-builder config generates correctly
- [x] entitlements.plist generates correctly
- [x] Notarization script generates correctly
- [x] Bundle export includes signing config

### Phase 5 Complete When

- [x] Tool detection works on all platforms
- [x] Certificate discovery works
- [x] Pre-deployment validation includes signing checks
- [x] End-to-end: config → validate → generate → build → signed app

---

## Appendix: Reference Materials

### External Documentation

- [electron-builder Code Signing](https://www.electron.build/code-signing)
- [Apple Notarization](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)
- [Windows Authenticode](https://docs.microsoft.com/en-us/windows/win32/seccrypto/cryptography-tools)
- [GPG Signing](https://www.gnupg.org/documentation/manuals/gnupg/)

### Related PRDs

- OT-P0-026: Actionable Remediation Steps
- OT-P1-016: Tier-Specific Settings UI

### Related Files in Codebase

- `api/bundles/handler.go` - Bundle assembly
- `api/bundles/validation.go` - Manifest validation
- `api/profiles/` - Profile management
- `docs/deployment/bundle-schema.desktop.v0.1.json` - JSON Schema
