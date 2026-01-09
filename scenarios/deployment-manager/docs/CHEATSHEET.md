# Deployment Manager Cheat Sheet

> Quick reference for common deployment-manager commands. Copy-paste ready.

## Essential Workflow

```bash
# 1. Check compatibility
deployment-manager fitness <scenario> --tier 2

# 2. Create profile
deployment-manager profile create my-profile <scenario> --tier 2

# 3. Apply swaps (if needed)
deployment-manager swaps apply <profile-id> postgres sqlite

# 4. Check secrets
deployment-manager secrets identify <profile-id>

# 5. Validate
deployment-manager validate <profile-id> --verbose

# 6. Package
deployment-manager package <profile-id> --packager scenario-to-desktop
```

---

## Health & Analysis

```bash
# Check API health
deployment-manager status

# Analyze scenario dependencies
deployment-manager analyze <scenario>

# Check fitness for specific tier
deployment-manager fitness <scenario> --tier 2        # Desktop
deployment-manager fitness <scenario> --tier desktop  # Same thing
deployment-manager fitness <scenario>                 # All tiers
```

## Profile Management

```bash
# List all profiles
deployment-manager profiles

# Create profile for desktop deployment
deployment-manager profile create <name> <scenario> --tier 2

# View profile details
deployment-manager profile show <profile-id>

# Update profile tier
deployment-manager profile update <profile-id> --tier 3

# Set environment variable
deployment-manager profile set <profile-id> env LOG_LEVEL info

# Export profile to JSON
deployment-manager profile export <profile-id> --output ./profile.json

# Import profile from JSON
deployment-manager profile import ./profile.json --name imported-profile

# View version history
deployment-manager profile versions <profile-id>

# Rollback to previous version
deployment-manager profile rollback <profile-id> --version 2

# Delete profile
deployment-manager profile delete <profile-id>
```

## Dependency Swaps

```bash
# List available swaps for scenario
deployment-manager swaps list <scenario>

# Analyze swap impact
deployment-manager swaps analyze postgres sqlite

# Check cascading effects
deployment-manager swaps cascade postgres sqlite

# Apply swap to profile
deployment-manager swaps apply <profile-id> postgres sqlite

# Apply and show new fitness
deployment-manager swaps apply <profile-id> postgres sqlite --show-fitness
```

## Secrets

```bash
# Identify required secrets
deployment-manager secrets identify <profile-id>

# Generate .env template
deployment-manager secrets template <profile-id> --format env

# Generate JSON template
deployment-manager secrets template <profile-id> --format json

# Validate secrets configuration
deployment-manager secrets validate <profile-id>
```

## Deployment & Packaging

```bash
# Pre-deployment validation
deployment-manager validate <profile-id>
deployment-manager validate <profile-id> --verbose

# Estimate SaaS/cloud costs
deployment-manager estimate-cost <profile-id>
deployment-manager estimate-cost <profile-id> --verbose

# Deploy (with dry-run option)
deployment-manager deploy <profile-id>
deployment-manager deploy <profile-id> --dry-run

# Package for desktop
deployment-manager package <profile-id> --packager scenario-to-desktop

# Check deployment status
deployment-manager deployment status <deployment-id>
```

## Logs & Telemetry

```bash
# View logs
deployment-manager logs <profile-id>

# Filter by level
deployment-manager logs <profile-id> --level error

# Search logs
deployment-manager logs <profile-id> --search "migration"

# Format as table
deployment-manager logs <profile-id> --format table
```

## Configuration

```bash
# Set API base URL
deployment-manager configure api_base http://localhost:8080

# Set authentication token
deployment-manager configure token <your-token>
```

## Output Formatting

```bash
# JSON output (prefix any command)
deployment-manager --json profiles

# Table output
deployment-manager --format table profiles
```

---

## Tier Mapping

| Input | Tier |
|-------|------|
| `local`, `1` | Tier 1 - Local Dev |
| `desktop`, `2` | Tier 2 - Desktop |
| `mobile`, `ios`, `android`, `3` | Tier 3 - Mobile |
| `saas`, `cloud`, `web`, `4` | Tier 4 - SaaS |
| `enterprise`, `on-prem`, `5` | Tier 5 - Enterprise |

## Common Swaps

| From | To | Best For |
|------|----|----------|
| `postgres` | `sqlite` | Desktop, Mobile |
| `redis` | `in-process` | Desktop, Mobile |
| `ollama` | packaged models | Desktop (offline) |
| `browserless` | `playwright-driver` | Desktop |

## Bundle Manifest (via REST API)

```bash
# Get API port
API_PORT=$(vrooli scenario port deployment-manager API_PORT)

# Assemble bundle manifest
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/assemble" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "<scenario>", "tier": "tier-2-desktop"}'

# Export with checksum
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/export" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "<scenario>", "tier": "tier-2-desktop"}' > bundle.json

# Validate manifest
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/validate" \
  -H "Content-Type: application/json" \
  -d @bundle.json
```

## Build Desktop Installers

```bash
cd scenarios/<scenario>/platforms/electron

# Install dependencies
pnpm install

# Build for specific platform
pnpm run dist:win    # Windows MSI
pnpm run dist:mac    # macOS PKG
pnpm run dist:linux  # Linux AppImage + DEB

# Build for all platforms
pnpm run dist:all
```

---

## Quick Troubleshooting

| Problem | Command |
|---------|---------|
| API not responding | `vrooli scenario start deployment-manager` |
| Unknown API port | `vrooli scenario port deployment-manager API_PORT` |
| Fitness score is 0 | `deployment-manager fitness <scenario> --tier 2` (check blockers) |
| Secrets validation failed | Apply swaps to remove infrastructure dependencies |

---

**Full Documentation**: [README.md](README.md) | **Step-by-Step Guide**: [DEPLOYMENT-GUIDE.md](DEPLOYMENT-GUIDE.md)
