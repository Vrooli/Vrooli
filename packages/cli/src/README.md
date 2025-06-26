# Vrooli CLI

Command-line interface for managing Vrooli instances.

## Installation

The CLI is built as part of the server package:

```bash
cd packages/server
pnpm install
pnpm run build
```

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
vrooli routine import ./routine.json

# Import directory of routines
vrooli routine import-dir ./staged-routines/

# List routines
vrooli routine list --limit 20

# Get routine details
vrooli routine get <routine-id>

# Export routine
vrooli routine export <routine-id> -o exported-routine.json

# Validate routine file
vrooli routine validate ./routine.json

# Run routine
vrooli routine run <routine-id> --watch
```

### Global Options

- `--profile <name>` - Use specific profile
- `--debug` - Enable debug output
- `--json` - Output in JSON format

## Configuration

Configuration is stored in `~/.vrooli/config.json`

```json
{
  "currentProfile": "default",
  "profiles": {
    "default": {
      "url": "http://localhost:5329",
      "authToken": "...",
      "refreshToken": "..."
    }
  }
}
```

## Development

To run the CLI in development mode:

```bash
cd packages/server
pnpm run cli:dev <command>
```