# Secrets Manager

> **Agent-first password manager with bounded delegation and deployment-aware credential operations**

Secrets Manager is a self-hosted password manager for daily human use and
controlled software access. It provides encrypted vault items, field-scoped
reveal, revision-safe edits, current-snapshot grants, approval requests, and
metadata-only activity. Its existing credential coverage, scanning, and
deployment workflows remain available to platform operators.

Secrets Manager owns credential discovery, metadata-safe operator surfaces, and
grant authoring. It does not own credential values at rest, runtime delivery
decisions, encryption, node key material, or fleet revocation; those remain
control-plane responsibilities.

## Product boundary

The API requires `SECRETS_MANAGER_VAULT_KEY` for durable secret custody. In a
local deployment, `SECRETS_MANAGER_OWNER_TOKEN` remains an explicit operator
fallback; a production relying-party deployment sets
`SECRETS_MANAGER_AUTHENTICATOR_JWKS_URL` and verifies Scenario Authenticator
tokens against the configured issuer and audience before resolving workspace
membership. The UI keeps credentials in memory only. List and detail calls
never return secret fields; reveal requires one-time assurance bound to the
exact item operation. See [PRD](PRD.md) and [security contract](docs/internal/SECURITY.md)
for support limits and claims that remain pending.

## 🎯 Business Value

- **Pre-Launch Confidence**: Verify all resource secrets before production deployment
- **Security Posture Visibility**: Real-time vulnerability scanning across all scenarios/resources
- **Deployment Readiness**: Tier-aware secret strategies (Tier 1-5) for packaging apps across platforms
- **Compliance Intelligence**: Unified health score combining credential readiness, vulnerability counts, and risk metrics
- **Operator Efficiency**: Guided workflows from detection → remediation without tribal knowledge

**Target Users**: Platform engineers, ecosystem maintainers, CI/CD pipelines, deployment automation

**Monetization Paths**:
- Internal infrastructure tool (reduces downtime from missing secrets)
- SaaS tier for enterprise Vrooli deployments
- Integration point for deployment-manager and scenario-to-* tooling

## 🏗️ Architecture

### Stack
- **API**: Go 1.21+ with Gorilla Mux, PostgreSQL for metadata/telemetry
- **UI**: React 18 + TypeScript + Vite + shadcn/ui (dark chrome theme)
- **Dependencies**: PostgreSQL, native credential authority, optional claude-code for auto-remediation
- **Lifecycle**: v2.0 service.json with standardized health checks, production bundle serving

### Key Components
- **Credential Intelligence**: Discovers resource manifest descriptors, checks native-authority status, and surfaces missing/invalid entries
- **Security Scanner**: Pattern-based detection of hardcoded secrets, SQL injection risks, insecure HTTP usage
- **Compliance Aggregator**: Blends credential readiness + vulnerability stats into unified metrics
- **Deployment Manifest Builder**: Generates tier-specific bundles for deployment-manager and scenario-to-* consumers

## 🚀 Quick Start

### Prerequisites
- Vrooli CLI installed (`vrooli setup --yes yes` from repo root)
- PostgreSQL resource running (`vrooli resource start postgres`)
- A supported native credential authority (the Vault resource is only required for a capability that explicitly declares it)

### Setup & Run
```bash
cd scenarios/secrets-manager

# One-time setup (builds API binary, UI bundle, seeds database)
make setup  # or: vrooli scenario setup secrets-manager

# Start the scenario
make start  # or: vrooli scenario start secrets-manager

# Access points (ports assigned by lifecycle system):
#   API:  http://localhost:${API_PORT}/health
#   Dashboard: http://localhost:${UI_PORT}

# View logs
make logs

# Stop
make stop
```

### Testing
```bash
# Run full phased test suite (structure → unit → integration)
make test  # or: vrooli scenario test secrets-manager

# Component tests
make test-api  # Go unit tests
make test-ui   # React component tests
make test-cli  # BATS CLI tests
```

## 📖 Documentation

- **[PRD](PRD.md)**: Operational targets (P0/P1/P2), success metrics, and integration requirements
- **[Requirements Index](requirements/index.json)**: Detailed requirement specs with validation criteria
- **[Progress Log](docs/internal/PROGRESS.md)**: Development history and % completion tracking
- **[Known Issues](docs/internal/PROBLEMS.md)**: Current blockers and follow-up tasks
- **[Research Notes](docs/RESEARCH.md)**: References and implementation learnings

## 🔌 Integration Points

### Consuming Secrets Manager

#### CLI Usage
```bash
# Get compliance status
secrets-manager security compliance

# List credential coverage
secrets-manager credentials coverage

# Scan for vulnerabilities
secrets-manager security scan

# Show effective override strategy for one scenario/tier
secrets-manager overrides effective picker-wheel --tier tier-2-desktop

# Export deployment manifest
secrets-manager deployment plan --scenario picker-wheel --tier tier-2-desktop --json
```

#### API Endpoints
```
GET  /api/v1/health                      # Health check (schema-compliant)
GET  /api/v1/credentials/status           # Credential-authority coverage summary
GET  /api/v1/security/vulnerabilities    # Filtered vulnerability list
GET  /api/v1/security/compliance         # Unified compliance metrics
POST /api/v1/credentials/provision        # Provision through the credential authority
GET  /api/v1/deployment/secrets          # Tier-aware manifest export
```

### Integrating with Other Scenarios
```typescript
// Example: deployment-manager requesting secrets for a package
const response = await fetch('http://localhost:16739/api/v1/deployment/secrets', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    scenario: 'picker-wheel',
    tier: 2,  // Desktop deployment
    include_optional: false
  })
});

const manifest = await response.json();
// manifest.secrets: { key: "value", strategy: "generate|prompt|delegate|strip" }
```

### Browser extension

The human browser extension source lives in `extension/` and is packaged by
Scenario To Extension with its caller-owned source option:

```bash
scenario-to-extension extension generate secrets-manager \
  --source-path "$PWD/extension" \
  --app-name "Vrooli Secrets Manager" \
  --api-endpoint "http://localhost:${API_PORT}" \
  --permissions nativeMessaging,storage,activeTab,scripting,tabs
```

The native host calls the Secrets Manager authority at
`SECRETS_MANAGER_NATIVE_HOST_URL` and authenticates its installed transport
with the `vrooli/secrets-manager/native-host:transport-token` credential. The
host keeps the owner session token only in memory for the active browser
session. Origin approval, safe account metadata selection, item revision, and
field selection are required before a fill. Save and update are explicit popup
actions with revision protection; the extension never submits a page form or
stores returned credentials.

## 🎨 UI Features

- **Orientation Hub**: Hero stats (configured resources, risk score, missing secrets) + journey cards
- **Credential Coverage Module**: Per-resource drilldowns with severity badges and missing credential callouts
- **Vulnerability Filter Panel**: Severity, component type, and search filtering
- **Compliance Callout**: Weighted risk score with color-coded status
- **Dark Chrome Theme**: WCAG AA contrast, lucide icons, subtle animations

## 🔐 Security Considerations

- Secrets are **never logged** or returned in API responses (only metadata and validation status)
- File content endpoint (`/files/content`) includes path traversal safeguards
- Provisioning sends values only over stdin to the control plane; status endpoints return metadata only
- The password-manager domain owns encrypted vault metadata, item envelopes, explicit trash/restore, grants, assurance, and audit metadata; ordinary resource credential coverage remains a control-plane responsibility
- Security scan patterns are versioned and validated before use
- PostgreSQL stores only secret **metadata**, not values

## 📊 Operational Status

**Current State** (as of 2025-11-18):
- ✅ Core credential-authority validation working
- ✅ Security scanning functional
- ✅ API health checks schema-compliant
- ✅ Production bundle serving via Express
- ✅ Password-manager vault shell with locked reveal and safe activity
- ✅ Deployment tier strategies and guided remediation flows remain available through the existing operator surface

See [docs/internal/PROGRESS.md](docs/internal/PROGRESS.md) for detailed completion metrics.

## 🤝 Contributing

1. Agents detect gaps via `scenario status` and `scenario-auditor`
2. Improvements tracked in `requirements/index.json` with `[REQ:ID]` tags
3. Tests validate operational targets → automatically update PRD checkboxes
4. Progress logged in `docs/internal/PROGRESS.md` for future agents

Run `vrooli scenario test secrets-manager` for the scenario's server-owned test conventions and evidence.

## 📜 License

MIT License - see [LICENSE](../../LICENSE)

---

**🚨 Critical Reminder**: Always start via `make start` or `vrooli scenario start secrets-manager`. Direct binary execution bypasses lifecycle management and breaks process monitoring.
