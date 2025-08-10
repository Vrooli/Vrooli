# Vrooli CLI

Command-line interface for managing Vrooli instances.

## Installation

```bash
# From the monorepo root
pnpm install
pnpm run build

# Link globally for development
cd packages/cli
npm link
```

### Command History Storage

The CLI uses a flexible storage system for command history:

- **SQLite Storage** (default): High-performance database storage for unlimited history
- **JSON Storage** (fallback): Automatic fallback when SQLite bindings are unavailable, limited to 10,000 entries

If you see a message about SQLite bindings not found, you can either:
1. Continue using JSON storage (works perfectly for most users)
2. Build SQLite bindings: `pnpm rebuild better-sqlite3`

Note: The CLI will always work regardless of which storage backend is used.

## Usage

### Authentication

```bash
# Login interactively
vrooli auth login

# Login with credentials
vrooli auth login --email user@example.com --password yourpassword

# Check authentication status
vrooli auth status

# Get current user info
vrooli auth whoami

# Logout
vrooli auth logout
```

### Profile Management

```bash
# List profiles
vrooli profile list

# Create new profile
vrooli profile create staging --url https://staging.vrooli.com

# Switch profiles
vrooli profile use staging
```

### Routine Management

```bash
# Import a single routine
vrooli routine import ./my-routine.json

# Import all routines from a directory
vrooli routine import-dir ./routines/

# Export a routine
vrooli routine export <routine-id> -o backup.json

# List routines
vrooli routine list --limit 20 --mine

# Validate routine JSON
vrooli routine validate ./my-routine.json

# Execute a routine
vrooli routine run <routine-id> --watch
```

### Global Options

- `-p, --profile <profile>` - Use a specific profile
- `-d, --debug` - Enable debug output
- `--json` - Output in JSON format
- `-h, --help` - Show help
- `-V, --version` - Show version

## Development

```bash
# Build the CLI
pnpm run build

# Run in development mode
pnpm run dev

# Type checking
pnpm run type-check

# Linting
pnpm run lint
```

## Configuration

The CLI stores configuration in `~/.vrooli/config.json`:

```json
{
  "currentProfile": "default",
  "profiles": {
    "default": {
      "url": "http://localhost:5329",
      "authToken": "...",
      "refreshToken": "..."
    },
    "production": {
      "url": "https://api.vrooli.com",
      "authToken": "...",
      "refreshToken": "..."
    }
  }
}
```

## Examples

### Basic Workflow

```bash
# Set up profile for local development
vrooli profile create local --url http://localhost:5329

# Authenticate
vrooli auth login

# Import some routines
vrooli routine import-dir ./examples/routines/

# List what was imported
vrooli routine list --mine

# Test a routine
vrooli routine run <routine-id> --input '{"message": "Hello World"}'
```

### Production Deployment

```bash
# Set up production profile
vrooli profile create production --url https://api.vrooli.com

# Switch to production
vrooli profile use production

# Authenticate with production credentials
vrooli auth login

# Import validated routines
vrooli routine import-dir ./dist/routines/ --fail-fast
```

### Command History

The CLI automatically tracks command history for easy reuse:

```bash
# View command history
vrooli history list

# Search history
vrooli history search "routine import"

# View history statistics
vrooli history stats

# Export history
vrooli history export --format json > history-backup.json
```

History is stored in `~/.vrooli/history.json` (JSON storage) or `~/.vrooli/history.db` (SQLite storage).

## Troubleshooting

### "Could not locate the bindings file" Error

This error indicates that native SQLite bindings aren't built. The CLI will automatically fall back to JSON storage, which works perfectly for most use cases. If you want to use SQLite storage:

```bash
# Rebuild native modules
pnpm rebuild better-sqlite3
```

### Build Issues

If the CLI isn't reflecting your changes:

```bash
# Clean and rebuild
cd packages/cli
rimraf dist
pnpm run build
```

### Import Assertion Warnings

You may see warnings about deprecated import assertions. These are harmless and will be addressed in a future update. They don't affect CLI functionality.