# Bundle Manifest Examples

This directory contains example `bundle.json` manifest files for different deployment configurations.

## Available Examples

| File | Description |
|------|-------------|
| `desktop-happy.json` | Minimal desktop bundle manifest (happy path) |
| `desktop-playwright.json` | Desktop bundle with Playwright testing configuration |
| `desktop-with-build.json` | Desktop bundle with full build pipeline |

## Usage

These manifests can be used as templates when creating deployment profiles. Copy and modify them for your scenario:

```bash
cp docs/examples/manifests/desktop-happy.json my-bundle.json
# Edit my-bundle.json for your scenario
deployment-manager bundles validate --manifest my-bundle.json
```

## Schema Reference

See the [Bundle Manifest Schema](../../guides/bundle-manifest-schema.md) for the full specification of all supported fields.
