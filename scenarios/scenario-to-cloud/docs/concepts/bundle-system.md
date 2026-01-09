# Bundle System

How mini-Vrooli bundles work.

## What is a Bundle?

A bundle is a self-contained tarball containing everything needed to run a Vrooli scenario on a fresh VPS. It's a "mini-Vrooli" - the smallest possible installation that can run your scenario.

## Bundle Contents

```
vrooli-bundle-{scenario}-{timestamp}.tar.gz
├── scripts/
│   ├── manage.sh           # Main CLI entry point
│   ├── main/
│   │   └── setup.sh        # Setup orchestration
│   └── lib/
│       └── lifecycle/      # Process management
├── scenarios/
│   └── {scenario}/         # Your scenario files
│       ├── api/
│       ├── ui/
│       └── service.json
├── resources/
│   └── {resources}/        # Required resources
├── packages/
│   └── (optional npm/go)   # Pre-built dependencies
└── .vrooli/
    └── config.json         # Bundle configuration
```

## Bundle Creation Process

### 1. Manifest Analysis

The system analyzes your manifest to determine:
- Which scenario to include
- Required resources
- Port mappings
- Dependencies

### 2. File Collection

Files are collected from:
- Core Vrooli scripts (always included)
- Target scenario directory
- Required resource directories
- Dependency scenarios (if any)

### 3. Optimization

Bundles are optimized by:
- Excluding development files (tests, docs, etc.)
- Excluding build artifacts (node_modules, vendor)
- Including only production configurations

### 4. Package Inclusion (Optional)

When `include_packages: true`:
- npm packages are bundled
- Go modules are included
- Reduces VPS build time

### 5. Compression

The final bundle is:
- Archived with tar
- Compressed with gzip
- Checksummed with SHA256

## Bundle Transfer

Bundles are transferred to VPS via SCP:

```bash
scp bundle.tar.gz user@host:~/
```

On the VPS:
```bash
mkdir -p ~/Vrooli
tar -xzf bundle.tar.gz -C ~/Vrooli
```

## Bundle Caching

Bundles are cached based on:
- Manifest hash
- Scenario file timestamps
- Resource versions

Cached bundles are reused when:
- Manifest hasn't changed
- Source files haven't changed
- `force_rebuild` is false

## Bundle vs Full Vrooli

| Feature | Bundle | Full Vrooli |
|---------|--------|-------------|
| Size | ~50-200 MB | ~2+ GB |
| Scenarios | Single | All |
| Resources | Required only | All |
| Development tools | No | Yes |
| Tests | No | Yes |
| Documentation | No | Yes |

## Customizing Bundles

### Include Additional Files

Add to your scenario's `service.json`:

```json
{
  "bundle": {
    "include": [
      "custom-scripts/",
      "data/*.json"
    ]
  }
}
```

### Exclude Files

```json
{
  "bundle": {
    "exclude": [
      "*.test.ts",
      "coverage/"
    ]
  }
}
```

### Pre-build Commands

```json
{
  "bundle": {
    "prebuild": [
      "npm run build",
      "go build ./api"
    ]
  }
}
```

## Troubleshooting

### Bundle Too Large

- Check for accidentally included `node_modules`
- Verify exclude patterns are working
- Consider disabling `include_packages`

### Missing Files

- Check file paths in manifest
- Verify files exist in source
- Check exclude patterns aren't too aggressive

### Checksum Mismatch

- Bundle may be corrupted during transfer
- Rebuild with `force_rebuild: true`
- Check disk space on local and remote
