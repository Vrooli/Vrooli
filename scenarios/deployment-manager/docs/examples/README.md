# Examples

Real-world deployment case studies and reference materials.

## Case Studies

| Example | Tier | Status | Description |
|---------|------|--------|-------------|
| [Picker Wheel Desktop](picker-wheel-desktop.md) | 2 | Partial | Desktop deployment with SQLite swap |
| [Picker Wheel Cloud](picker-wheel-cloud.md) | 4 | Planned | VPS deployment |
| [System Monitor Desktop](system-monitor-desktop.md) | 2 | Planned | System monitoring desktop app |

## Bundle Manifests

Reference `bundle.json` files for desktop deployment:

| Manifest | Description |
|----------|-------------|
| [desktop-happy.json](manifests/desktop-happy.json) | Happy path: UI + API + SQLite |
| [desktop-playwright.json](manifests/desktop-playwright.json) | With bundled Playwright + Chromium |

## Using Examples

### As Learning Material

1. Read the case study to understand the approach
2. Review the manifest to see the configuration
3. Apply similar patterns to your scenario

### As Templates

```bash
# Copy a manifest as starting point
cp docs/examples/manifests/desktop-happy.json my-bundle.json

# Edit for your scenario
# - Update app.name, app.version
# - Adjust services for your architecture
# - Configure secrets for your needs
```

## Manifest Schema

All manifests follow schema v0.1. See [Bundle API Documentation](../api/bundles.md) for the full schema reference.

### Key Sections

```json
{
  "schema_version": "v0.1",
  "target": "desktop",
  "app": { "name": "...", "version": "..." },
  "ipc": { "mode": "loopback-http", ... },
  "swaps": [ { "original": "postgres", "replacement": "sqlite" } ],
  "secrets": [ { "id": "...", "class": "per_install_generated|user_prompt" } ],
  "services": [ { "id": "api", "type": "api-binary", ... } ]
}
```

## Contributing Examples

When adding examples:

1. Create case study markdown documenting the real experience
2. Include both successes and failures/learnings
3. Add bundle manifest if applicable
4. Link from this README

## Related

- [Deployment Guide](../DEPLOYMENT-GUIDE.md) - Single entry point for deploying scenarios
- [Desktop Workflow](../workflows/desktop-deployment.md) - Full deployment guide
- [Bundle API](../api/bundles.md) - Manifest schema reference
- [Dependency Swapping](../guides/dependency-swapping.md) - Swap strategies
- [Roadmap](../ROADMAP.md) - Implementation status and planned work
