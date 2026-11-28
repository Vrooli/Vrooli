# Research & Implementation References

This document captures references, patterns, and learnings discovered during secrets-manager development.

---

## 🔐 Credential Authority

**Reference**: resource `resource.json` descriptors and `vrooli credentials`.

**Key Learnings**:
- Manifests declare the logical identity, field, environment name, and required status.
- `vrooli credentials status --format json` reports metadata only; it never returns a value, and it carries the provider state so `configured: false` cannot be misread while the store is down.
- `vrooli credentials provision --identity <logical-id> --field <field>` accepts a value on stdin only; secrets-manager owns inventory, keyring, and backup operations.
- `vrooli credentials doctor` diagnoses the host backend; `secrets-manager backup status` reports recovery coverage without printing a value.
- Availability is probed lazily and read-shaped, on first use rather than at construction, and cached for the process lifetime. Read paths degrade: a missing or unreachable credential never blocks a scenario start, and the declaring resource reports unhealthy instead. Write paths (recovery export/restore, Vault bootstrap) still fail closed.
- Three conditions stay distinct end to end and each has its own operator action: value unconfigured, provider unreachable, provider absent on this host.
- Vault may be a capability-specific service or explicit mirror, never an ordinary credential fallback.

---

## 🔍 Security Scanning Patterns

### Pattern Detection Rules
**Reference**: `api/security_scanner.go`, `api/scanner.go`

**Categories Implemented**:
1. **Hardcoded Secrets**: Regex patterns for API keys, tokens, passwords in code
2. **SQL Injection**: Unsafe query construction (string concatenation)
3. **HTTP Insecurity**: Unencrypted connections, CORS wildcards
4. **Path Traversal**: Unsafe file path handling

**Pattern Examples**:
```go
// Hardcoded API key pattern
regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?token)\s*[:=]\s*["']([a-zA-Z0-9_-]{20,})["']`)

// SQL injection risk
regexp.MustCompile(`(?i)(execute|query|prepare)\s*\(\s*["'].*\s*\+\s*`)

// CORS wildcard
regexp.MustCompile(`AllowedOrigins.*\*|Access-Control-Allow-Origin.*\*`)
```

**Severity Scoring**:
- Critical: Exposed credentials, SQL injection with user input
- High: CORS wildcards, insecure HTTP in production
- Medium: Weak crypto, missing input validation
- Low: Deprecated functions, info disclosure

**Context Enrichment**:
- File path + line number
- Code snippet (5 lines context)
- Component name (extracted from file path)
- Remediation suggestion (mapped from pattern ID)

---

## 🏗️ React + Vite UI Architecture

### Component Organization
**Reference**: `ui/src/` structure

**Pattern**: Feature-based organization
```
src/
├── components/
│   ├── layout/        # Header, Sidebar, Footer
│   ├── vault/         # Vault-specific views
│   ├── security/      # Vulnerability tables/filters
│   └── ui/            # shadcn/ui primitives
├── hooks/             # React Query hooks
├── lib/               # Utilities (API client, formatters)
└── App.tsx            # Router + layout shell
```

**Key Libraries**:
- **@tanstack/react-query**: API state management, caching, background refetch
- **@vrooli/iframe-bridge**: Embed in app-monitor or other dashboards
- **shadcn/ui**: Accessible component primitives (built on Radix UI)
- **lucide-react**: Icon system
- **tailwind-merge + clsx**: Dynamic class composition

### API Integration Pattern
```typescript
// hooks/useVaultStatus.ts
import { useQuery } from '@tanstack/react-query';

export function useVaultStatus(resourceFilter?: string) {
  return useQuery({
    queryKey: ['vault-status', resourceFilter],
    queryFn: async () => {
      const params = resourceFilter ? `?resource=${resourceFilter}` : '';
      const res = await fetch(`${API_BASE_URL}/vault/secrets/status${params}`);
      return res.json();
    },
    refetchInterval: 30000, // Auto-refresh every 30s
    staleTime: 10000,
  });
}
```

**Health Check Integration**:
```typescript
// UI health endpoint must return api_connectivity field
{
  "status": "healthy",
  "api_connectivity": "healthy" | "degraded" | "unavailable",
  "bundle_exists": true,
  ...
}
```

---

## 🎨 Dark Chrome Theme Guidelines

### Design System
**Reference**: `ui/src/index.css`, shadcn/ui config

**Color Palette**:
```css
:root {
  --background: 224 71% 4%;     /* Near black */
  --foreground: 213 31% 91%;    /* Light gray text */
  --primary: 210 40% 98%;       /* White/cyan highlights */
  --destructive: 0 63% 31%;     /* Red for errors */
  --success: 142 76% 36%;       /* Green for success */
  --warning: 38 92% 50%;        /* Amber for warnings */
}
```

**Accessibility Requirements**:
- All text must meet WCAG AA contrast (4.5:1 for normal, 3:1 for large)
- Interactive elements require 3:1 contrast with background
- Focus indicators visible on all keyboard-navigable elements
- Semantic color + text fallbacks (don't rely on color alone)

**Animation Guidelines**:
- Use `prefers-reduced-motion` media query for motion-sensitive users
- Keep transitions subtle (150-300ms duration)
- No auto-playing animations or flashing content

---

## 📊 PostgreSQL Schema Design

### Metadata Storage Strategy
**Reference**: `api/internal/<domain>/storage/postgres/schema.sql`

**Core Tables**:
1. **resource_secrets**: Secret definitions (key, type, required, validation pattern)
2. **secret_validations**: Historical validation results (status, timestamp, error details)
3. **secret_scans**: Scan execution metadata (duration, resources scanned, findings count)

**Design Decisions**:
- Store secret **metadata** only - never store actual values in database
- Use JSONB for extensibility (validation_details, scan_metadata)
- Timestamp all entries for trend analysis
- Foreign key from secret_validations → resource_secrets for relational queries

**Indexing**:
```sql
CREATE INDEX idx_resource_secrets_name ON resource_secrets(resource_name);
CREATE INDEX idx_secret_validations_timestamp ON secret_validations(validation_timestamp DESC);
CREATE INDEX idx_secret_scans_timestamp ON secret_scans(scan_timestamp DESC);
```

---

## 🧪 Testing Architecture

### Phased Testing Approach
**Reference**: `/docs/testing/architecture/PHASED_TESTING.md`

**Phases**:
1. **Structure**: Validate service.json, Makefile, PRD structure
2. **Unit**: Go API tests, React component tests, CLI BATS tests
3. **Integration**: API contract tests, health check validation
4. **Performance**: Response time budgets, Lighthouse scores (UI)
5. **Acceptance**: End-to-end user flows (UI automation via browser-automation-studio)

**Requirement Linking**:
```go
// In test files, add [REQ:ID] tags for automatic tracking
func TestVaultStatusEndpoint(t *testing.T) {
    // [REQ:SEC-VLT-004] Vault health endpoint
    resp, err := http.Get("http://localhost:16739/api/v1/vault/secrets/status")
    // ...
}
```

**Coverage Reporting**:
- Go: `go test -coverprofile=coverage.out ./...`
- UI: `vitest --coverage`
- Requirements: Auto-generated from `[REQ:ID]` tags when tests run

---

## 🚀 Deployment Tier Model

### Tier Definitions
**Reference**: `/docs/deployment/README.md`

| Tier | Platform | Secret Strategy | Example |
|------|----------|-----------------|---------|
| 1 | Local Dev | Read from Vault | Full Vrooli installation with vault |
| 2 | Desktop | Generate/Prompt | Electron app prompts for API keys |
| 3 | Mobile | Generate/Delegate | Mobile app uses OAuth delegation |
| 4 | SaaS | Backend-managed | Secrets stored in cloud KMS |
| 5 | Enterprise | Customer Vault | Enterprise install provisions own vault |

**Strategy Types**:
- **strip**: Remove secret from bundle (not needed for this tier)
- **generate**: Auto-generate placeholder (e.g., random DB password for local SQLite)
- **prompt**: Ask user to provide during first-run setup
- **delegate**: Use OAuth/OIDC delegation instead of direct secret
- **backend-managed**: Secret provisioned by backend service (SaaS tier)

**Manifest Format** (to be implemented):
```json
{
  "scenario": "picker-wheel",
  "tier": 2,
  "secrets": [
    {
      "key": "DB_PASSWORD",
      "strategy": "generate",
      "generator_hint": "Use SQLCipher with random 32-char passphrase"
    },
    {
      "key": "OPENAI_API_KEY",
      "strategy": "prompt",
      "prompt_message": "Enter your OpenAI API key (get one at platform.openai.com)"
    },
    {
      "key": "INTERNAL_SERVICE_TOKEN",
      "strategy": "strip",
      "reason": "Desktop tier doesn't use internal microservices"
    }
  ]
}
```

---

## 🔗 Cross-Scenario Integration Patterns

### Consuming Secrets-Manager APIs

**Pattern 1: Pre-Flight Deployment Check**
```typescript
// deployment-manager checking readiness before package build
const compliance = await fetch('http://localhost:16739/api/v1/security/compliance')
  .then(r => r.json());

if (compliance.risk_score > 7.0) {
  throw new Error(`Security risk too high: ${compliance.risk_score}/10. Fix vulnerabilities first.`);
}

if (compliance.vault_coverage_percent < 80) {
  throw new Error(`Only ${compliance.vault_coverage_percent}% of secrets configured. Run: secrets-manager provision`);
}
```

**Pattern 2: Dynamic Port Resolution**
```bash
# Other scenarios discovering secrets-manager API port
API_PORT=$(vrooli scenario port secrets-manager API_PORT)
curl "http://localhost:${API_PORT}/api/v1/vault/secrets/status"
```

**Pattern 3: CLI Integration**
```bash
# CI pipeline checking compliance
if ! secrets-manager status --min-score 8.0; then
  echo "Security compliance failed. See: secrets-manager security scan"
  exit 1
fi
```

---

## 📚 Standards & Schemas

### Lifecycle v2.0 Requirements
**Reference**: `/cli/commands/scenario/schemas/service.schema.json`

**Critical Fields**:
- `lifecycle.health.endpoints`: Must define `{api: "/health", ui: "/health"}`
- `lifecycle.setup.condition`: Binaries, CLI, and ui-bundle checks for staleness detection
- `lifecycle.develop.steps[].condition.file_exists`: Track binary/bundle dependencies
- Production bundle serving: Use `node server.cjs` not `pnpm run dev`

**Health Response Schemas**:
- API: Must include `status`, `service`, `timestamp`, `readiness`, `dependencies`
- UI: Must include `status`, `api_connectivity`, `bundle_exists`

### PRD Structure Requirements
**Reference**: `templates/scenarios/react-vite/PRD.md`

**Required Sections**:
- 🎯 Capability Definition
- 📊 Success Metrics
- 🏗️ Technical Architecture
- 🖥️ CLI Interface Contract
- 🔄 Integration Requirements
- 🎨 Style and Branding Requirements
- 💰 Value Proposition
- 🧬 Evolution Path
- 🔄 Scenario Lifecycle Integration
- 🚨 Risk Mitigation

**Operational Targets**: Must use `OT-{Priority}-{Number}` format and link to requirements

---

## 🧠 Lessons Learned

### What Worked Well
1. **Fallback Strategy**: Vault CLI + local file fallback makes development resilient
2. **React Query**: Automatic caching and refetching simplified UI state management
3. **Phased Testing**: Structure → Unit → Integration flow catches issues early
4. **Requirements Linking**: `[REQ:ID]` tags enable automatic PRD sync

### What Needs Improvement
1. **Monolithic main.go**: 2204 lines make refactoring difficult
2. **Build-Time Config**: Baking API_PORT into UI bundle prevents runtime flexibility
3. **Error Visibility**: Scanner errors logged but not surfaced in UI
4. **Documentation Lag**: README/PROGRESS created late in development cycle

### Future Research Areas
1. **Agent-Driven Remediation**: Can claude-code agents auto-fix vulnerabilities?
2. **Secret Rotation Automation**: Should secrets-manager handle rotation workflows?
3. **Cross-Repo Scanning**: Extend scanner to detect secrets in git history?
4. **Policy Engine**: Threshold-based blocking of scenario starts?

---

**Last Updated**: 2025-11-18
**Contributors**: scenario-improver agents, Vrooli team
