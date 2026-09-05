# Auto-Update System for Desktop Applications

This guide explains how to configure automatic updates for desktop applications built with scenario-to-desktop.

## Implementation Map

- [CODE: api/generation/types.go#UpdateConfig] - canonical update config schema (`provider`, `channel`, `generic.url`, `github`)
- [CODE: api/pipeline/stage_generate.go] - generation-time validation/warnings for update configuration
- [CODE: templates/build-tools/template-generator.ts#getPublishConfig] - electron-builder publish config generation
- [CODE: templates/build-tools/template-generator.ts#getEffectiveUpdateProvider] - runtime-safe provider fallback (`generic` -> `none` when URL missing)
- [CODE: api/pipeline/stage_deploy.go] - deploy-stage update URL derivation for LPBS
- [DOC: docs/guides/DEPLOYMENT.md#upload-flow] - LPBS deployment/upload flow

## Overview

The auto-update system uses [electron-updater](https://www.electron.build/docs/api/) to check for and apply updates. Scenario-to-desktop supports multiple update providers:

| Provider | Description | Use Case |
|----------|-------------|----------|
| **generic** (default) | Self-hosted update server | Full control over update distribution |
| **github** | GitHub Releases | Open source projects with public releases |
| **none** | Disabled | Development or manual-update-only builds |

## Provider: Generic (Self-Hosted)

The generic provider is the default and recommended choice for most scenarios. It works with any HTTP server that can serve static files.

### Configuration

```json
{
  "update_config": {
    "provider": "generic",
    "channel": "stable",
    "auto_check": true,
    "generic": {
      "url": "https://updates.example.com/my-app"
    }
  }
}
```

### URL Structure

The update URL follows this pattern:

```
{base_url}/{channel}/latest.yml
{base_url}/{channel}/latest-mac.yml
{base_url}/{channel}/latest-linux.yml
{base_url}/{channel}/{artifact-file}
```

For example, with `url: "https://updates.example.com/my-app"` and `channel: "stable"`:

- `https://updates.example.com/my-app/stable/latest.yml` (Windows)
- `https://updates.example.com/my-app/stable/latest-mac.yml` (macOS)
- `https://updates.example.com/my-app/stable/latest-linux.yml` (Linux)
- `https://updates.example.com/my-app/stable/my-app-1.2.3.exe` (artifact)

### Custom Channel Paths

Use `channel_path` to customize the URL structure:

```json
{
  "update_config": {
    "provider": "generic",
    "generic": {
      "url": "https://updates.example.com/my-app",
      "channel_path": "/releases/{channel}"
    }
  }
}
```

This produces: `https://updates.example.com/my-app/releases/stable/latest.yml`

### Manifest Format

The manifest files (`latest.yml`, `latest-mac.yml`, `latest-linux.yml`) use the electron-updater YAML format:

```yaml
version: 1.2.3
path: my-app-1.2.3.exe
sha512: <base64-encoded-sha512-hash>
releaseDate: 2026-02-05T12:00:00Z
files:
  - url: https://updates.example.com/my-app/stable/my-app-1.2.3.exe
    sha512: <base64-encoded-sha512-hash>
    size: 85234567
```

### Manifest Generation

Update metadata and publish behavior are configured during template generation/build. In current code:

1. `update_config.provider` defaults to `"generic"` when omitted.
2. If provider is `generic` and `update_config.generic.url` is missing, generation logs a warning and the effective provider becomes `none` (auto-updates disabled).
3. If a valid provider config exists, electron-builder publish config is emitted and updater metadata is produced as part of the build/publish path.

When using LPBS, the deploy stage can derive an update URL (`.../api/v1/updates/{app_key}`) from the remote profile and expose it in deploy results.

## Provider: GitHub

For open source projects, GitHub Releases integration is simpler to set up:

```json
{
  "update_config": {
    "provider": "github",
    "channel": "stable",
    "auto_check": true,
    "github": {
      "owner": "your-org",
      "repo": "your-app"
    }
  }
}
```

**Note:** GitHub provider relies on electron-builder's built-in GitHub publishing, which handles manifest generation automatically. No manual manifest generation is performed.

### Private Repositories

For private repos, set `private: true` and ensure the `GH_TOKEN` environment variable is set at build time:

```json
{
  "update_config": {
    "provider": "github",
    "github": {
      "owner": "your-org",
      "repo": "your-private-app",
      "private": true
    }
  }
}
```

## Provider: None

To explicitly disable auto-updates:

```json
{
  "update_config": {
    "provider": "none"
  }
}
```

## LPBS Integration

When using the Landing Page Business Suite (LPBS) as your update server, configure the URL to point to the LPBS update endpoint:

```json
{
  "update_config": {
    "provider": "generic",
    "generic": {
      "url": "https://your-lpbs-domain.com/api/v1/updates/{app_key}"
    }
  }
}
```

Replace `{app_key}` with your actual application key registered in LPBS.

**Expected LPBS Endpoints:**
```
GET /api/v1/updates/{app_key}/{channel}/latest.yml
GET /api/v1/updates/{app_key}/{channel}/latest-mac.yml
GET /api/v1/updates/{app_key}/{channel}/latest-linux.yml
```

These endpoints should:
1. Look up the latest artifact for the given app_key and platform
2. Return manifest YAML with `Content-Type: application/x-yaml`
3. Be publicly accessible (electron-updater doesn't support authentication)

## Configuration Options

### update_config

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `provider` | string | `"generic"` | Provider type: `generic`, `github`, or `none` |
| `channel` | string | `"stable"` | Update channel: `stable`, `beta`, or `dev` |
| `auto_check` | boolean | `false` | Check for updates on app startup |

### update_config.generic

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Base URL for update server |
| `channel_path` | string | No | Custom channel path template |

### update_config.github

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | string | No | GitHub organization or user |
| `repo` | string | No | GitHub repository name |
| `private` | boolean | No | Set to `true` for private repos |

## Troubleshooting

### "Generic update provider without URL"

This warning appears when:
1. No `update_config` is provided (default provider is generic)
2. `provider: "generic"` is set without a URL

**Solution:** Either:
- Add `update_config.generic.url` with your update server URL
- Set `provider: "none"` to explicitly disable updates

### Manifest not generated

Check that:
1. Build completed successfully with artifacts
2. `update_config.generic.url` is set
3. Publish/update provider configuration is valid for the selected provider

### Update check fails at runtime

Verify:
1. Manifest files are accessible at the expected URLs
2. Server returns correct `Content-Type` header
3. SHA-512 hashes in manifest match actual files
4. Certificate is valid (HTTPS required for production)

### Hash mismatch errors

The SHA-512 hash in the manifest must match the actual file. If you manually upload artifacts:
1. Generate the hash: `shasum -a 512 myapp.exe | xxd -r -p | base64`
2. Update the manifest with the correct hash
3. Ensure no file corruption during upload

## Architecture

```
Pipeline + Templates
──────────────────────────────────────────────────────────────────────────
update_config ──► Generate Stage ──► template-generator (publish config)
                                     │
                                     ├─ provider=generic + URL    -> generic publish enabled
                                     ├─ provider=github            -> github publish enabled
                                     └─ provider missing/invalid   -> updates disabled safely

Build/Publish Output
──────────────────────────────────────────────────────────────────────────
electron-builder artifacts + updater metadata (latest*.yml)
                                     │
                                     └─► Deploy Stage (LPBS flow) uploads artifacts
                                          and derives update URL when available
```

## See Also

- [electron-updater documentation](https://www.electron.build/docs/api/)
- [LPBS Deployment](./DEPLOYMENT.md)
- [Pipeline Stages](../reference/smoke-test-pipeline.md)
