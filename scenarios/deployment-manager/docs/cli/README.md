# CLI Reference

The deployment-manager CLI provides command-line access to all deployment orchestration features.

## Installation

The CLI is automatically installed when you set up the deployment-manager scenario:

```bash
vrooli scenario setup deployment-manager
```

Or install manually:

```bash
./packages/cli-core/install.sh scenarios/deployment-manager/cli --name deployment-manager
```

## Global Options

These options apply to all commands:

| Option | Description | Example |
|--------|-------------|---------|
| `--json` | Output in JSON format | `deployment-manager status --json` |
| `--format <fmt>` | Output format: `json` or `table` | `deployment-manager profiles --format table` |

> **Note**: Global options are consumed once and apply to all subsequent commands in the session.

## Configuration

```bash
# Set API base URL (auto-detected from lifecycle by default)
deployment-manager configure api_base http://localhost:8080

# Set authentication token
deployment-manager configure token <your-token>
```

Configuration is stored in `~/.deployment-manager/config.json`.

## Command Groups

| Group | Purpose | Documentation |
|-------|---------|---------------|
| **Overview** | Health checks, analysis, fitness scoring | [overview-commands.md](overview-commands.md) |
| **Profiles** | Create, configure, version deployment profiles | [profile-commands.md](profile-commands.md) |
| **Swaps** | Dependency replacement analysis and application | [swap-commands.md](swap-commands.md) |
| **Secrets** | Secret identification, templates, validation | [secret-commands.md](secret-commands.md) |
| **Deployments** | Package, deploy, validate, monitor | [deployment-commands.md](deployment-commands.md) |

## Quick Reference

### Overview Commands

```bash
deployment-manager status                           # Check API health
deployment-manager analyze <scenario>               # Dependency DAG
deployment-manager fitness <scenario> --tier 2     # Fitness score for tier
```

### Profile Commands

```bash
deployment-manager profiles                         # List all profiles
deployment-manager profile create <name> <scenario> --tier 2
deployment-manager profile show <id>
deployment-manager profile update <id> --tier 3
deployment-manager profile delete <id>
deployment-manager profile export <id> --output ./profile.json
deployment-manager profile import ./profile.json --name override-name
deployment-manager profile set <id> env KEY value
deployment-manager profile swap <id> add postgres sqlite
deployment-manager profile versions <id>
deployment-manager profile rollback <id> --version 2
```

### Swap Commands

```bash
deployment-manager swaps list <scenario>            # Available swaps
deployment-manager swaps analyze postgres sqlite    # Impact analysis
deployment-manager swaps cascade postgres sqlite    # Cascading effects
deployment-manager swaps info <swap_id>             # Swap details
deployment-manager swaps apply <profile> postgres sqlite --show-fitness
```

### Secret Commands

```bash
deployment-manager secrets identify <profile-id>   # Required secrets
deployment-manager secrets template <profile-id> --format env
deployment-manager secrets template <profile-id> --format json
deployment-manager secrets validate <profile-id>   # Validate configuration
```

### Deployment Commands

```bash
deployment-manager validate <profile-id> --verbose # Pre-flight checks
deployment-manager deploy <profile-id> --dry-run   # Simulate deployment
deployment-manager deploy <profile-id> --async     # Background deployment
deployment-manager deployment status <deployment-id>
deployment-manager package <profile-id> --packager scenario-to-desktop
deployment-manager packagers list                   # Available packagers
deployment-manager packagers discover               # Discover new packagers
deployment-manager logs <profile-id> --level error --format table
deployment-manager estimate-cost <profile-id> --verbose
```

## Tier Mapping

The `--tier` flag accepts names or numbers:

| Input | Tier |
|-------|------|
| `local`, `1` | Tier 1 (Local/Dev Stack) |
| `desktop`, `2` | Tier 2 (Desktop) |
| `mobile`, `ios`, `android`, `3` | Tier 3 (Mobile) |
| `saas`, `cloud`, `web`, `4` | Tier 4 (SaaS/Cloud) |
| `enterprise`, `on-prem`, `5` | Tier 5 (Enterprise) |

## Error Handling

The CLI returns appropriate exit codes:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Invalid arguments |
| `3` | API connection failed |
| `4` | Resource not found |
| `5` | Validation failed |

## Auto-Detection

By default, the CLI auto-discovers the API base URL:

```bash
# Uses vrooli scenario port internally
vrooli scenario port deployment-manager API_PORT
```

If the scenario isn't running through the lifecycle system, the CLI will warn and require manual configuration.

## Examples

### Complete Desktop Deployment Flow

```bash
# 1. Check scenario fitness
deployment-manager fitness picker-wheel --tier desktop

# 2. Create a profile
deployment-manager profile create pw-desktop picker-wheel --tier 2

# 3. Apply required swaps
deployment-manager swaps apply profile-123 postgres sqlite --show-fitness

# 4. Configure secrets
deployment-manager secrets identify profile-123
deployment-manager secrets template profile-123 --format env > .env.template

# 5. Validate before deployment
deployment-manager validate profile-123 --verbose

# 6. Package for desktop
deployment-manager package profile-123 --packager scenario-to-desktop
```

### Export and Import Profiles

```bash
# Export a profile for backup or sharing
deployment-manager profile export profile-123 --output ./my-profile.json

# Import on another system
deployment-manager profile import ./my-profile.json --name imported-profile
```

### Monitor Deployments

```bash
# View recent logs
deployment-manager logs profile-123 --format table

# Filter by severity
deployment-manager logs profile-123 --level error

# Search for specific issues
deployment-manager logs profile-123 --search "migration failed"
```

## Source Code

The CLI is implemented in Go using `packages/cli-core`:

- Entry point: `scenarios/deployment-manager/cli/app.go`
- Command groups: `scenarios/deployment-manager/cli/{overview,profiles,swaps,deployments}/commands.go`
- Utilities: `scenarios/deployment-manager/cli/cmdutil/util.go`

## Related

- [Deployment Guide](../DEPLOYMENT-GUIDE.md) - Complete deployment walkthrough
- [API Reference](../api/README.md) - REST API documentation
- [Desktop Workflow](../workflows/desktop-deployment.md) - Full desktop deployment guide
- [Roadmap](../ROADMAP.md) - Implementation status and planned work
