# Code Signing Guide

Code signing ensures your desktop installers are trusted by operating systems and users. Without signing:
- **Windows**: SmartScreen warnings block installation
- **macOS**: Gatekeeper prevents app from running
- **Linux**: Package managers may reject unsigned packages

## Signing is Managed by scenario-to-desktop

Code signing configuration and execution is handled by **scenario-to-desktop**, not deployment-manager. This ensures:
- Single source of truth for signing credentials
- Direct access to platform-specific signing tools
- No credential duplication across scenarios

## Complete Signing Documentation

For full code signing instructions, including:
- Platform-specific requirements (Windows Authenticode, macOS Developer ID, Linux GPG)
- Certificate acquisition and costs
- Tool installation and verification
- Configuration via UI or CLI
- Troubleshooting common issues

**See: [scenario-to-desktop Code Signing Guide](../../../scenario-to-desktop/docs/SIGNING.md)**

## Quick Reference

| Platform | Requirement | Cost | Where to Sign |
|----------|-------------|------|---------------|
| Windows | Authenticode certificate | $200-700/year | Windows only |
| macOS | Apple Developer ID + notarization | $99/year | macOS only |
| Linux | GPG key | Free | Any platform |

## Integration with deploy-desktop

The `deploy-desktop` command automatically:
1. **Applies signing config** if provided via `--signing-config` (Step 2.5a)
2. **Checks signing readiness** before building (Step 2.5b)
3. **Warns** if signing is not configured (non-blocking)
4. **Uses** scenario-to-desktop's signing configuration during installer builds

### Option 1: Inline with deploy-desktop (Recommended)

Create a signing config JSON file and pass it directly:

```bash
deployment-manager deploy-desktop --profile my-profile --signing-config ./signing.json
```

**Example `signing.json`:**
```json
{
  "enabled": true,
  "windows": {
    "certificate_source": "file",
    "certificate_file": "./certs/my-cert.pfx",
    "certificate_password_env": "WIN_CERT_PASSWORD",
    "timestamp_server": "http://timestamp.digicert.com",
    "sign_algorithm": "sha256"
  },
  "macos": {
    "identity": "Developer ID Application: Your Company (TEAMID)",
    "team_id": "TEAMID",
    "hardened_runtime": true,
    "notarize": true,
    "apple_api_key_id": "KEYID",
    "apple_api_key_file": "./certs/AuthKey.p8",
    "apple_api_issuer_id": "ISSUER-UUID"
  },
  "linux": {
    "gpg_key_id": "YOUR_GPG_KEY_ID",
    "gpg_passphrase_env": "GPG_PASSPHRASE"
  }
}
```

### Option 2: Configure separately via scenario-to-desktop

```bash
# Use scenario-to-desktop UI
# Navigate to Signing tab in scenario-to-desktop web UI

# Or use scenario-to-desktop CLI
scenario-to-desktop signing set <scenario> --platform windows --certificate-file ./cert.pfx
scenario-to-desktop signing set <scenario> --platform macos --identity "Developer ID Application: Your Name"
scenario-to-desktop signing set <scenario> --platform linux --gpg-key-id YOUR_KEY_ID
```

## Deployment-Manager Signing Endpoints (Deprecated)

The signing endpoints in deployment-manager (`/api/v1/profiles/{id}/signing`) are **deprecated** and will be removed in a future version. They currently proxy to scenario-to-desktop.

To migrate:
1. Use scenario-to-desktop's signing CLI or API directly
2. Or set `SIGNING_PROXY_ENABLED=true` in deployment-manager to auto-proxy

## Related Documentation

- [Desktop Deployment Guide](../DESKTOP-DEPLOYMENT-GUIDE.md) - Full deployment workflow
- [Auto-Updates Guide](auto-updates.md) - Configure update channels
- [scenario-to-desktop SIGNING.md](../../../scenario-to-desktop/docs/SIGNING.md) - Complete signing reference
