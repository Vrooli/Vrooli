# CLI Reference

## Installation

```bash
cd scenarios/development-toolchain-validator/cli && ./install.sh
```

Binary installs to `~/.vrooli/bin/development-toolchain-validator`.

## Configuration

```bash
development-toolchain-validator configure api_base http://localhost:${API_PORT}/api/v1
development-toolchain-validator configure token <optional-auth-token>
```

## Commands

### References

```bash
# List all registered reference scenarios
development-toolchain-validator references list [--json]

# Register a reference scenario
development-toolchain-validator references add <name> --template <template>

# Show reference details
development-toolchain-validator references show <name> [--json]

# Remove a reference
development-toolchain-validator references remove <name>
```

### Skills

```bash
# List skill connections for a reference
development-toolchain-validator skills list --reference <name> [--json]

# Connect a skill to a reference
development-toolchain-validator skills connect <skill-id> --reference <name>

# Show connection details (version, drift, expectations)
development-toolchain-validator skills show <skill-id> --reference <name> [--json]

# Disconnect a skill from a reference
development-toolchain-validator skills disconnect <skill-id> --reference <name>

# Refresh version pin after reviewing drift
development-toolchain-validator skills refresh <skill-id> --reference <name>
```

### Expectations

```bash
# List expectations for a connection
development-toolchain-validator expectations list \
  --connection <skill-id>:<reference> [--json]

# Add a structural expectation
development-toolchain-validator expectations add structural \
  --connection <skill-id>:<reference> \
  --type <folder|file|snippet> \
  --path <path-or-glob> \
  [--required true|false] \
  [--snippet-content <content>] \
  [--snippet-location <location>] \
  --description <text>

# Add a CLI tool assertion
development-toolchain-validator expectations add cli-tool \
  --connection <skill-id>:<reference> \
  --command <command-string> \
  --path <jsonpath-expression> \
  --op <operator> \
  [--value <expected-value>] \
  [--timeout <seconds>] \
  --description <text>

# Remove an expectation
development-toolchain-validator expectations remove <expectation-id>
```

### Validation

```bash
# Run full validation for a reference
development-toolchain-validator validate <reference> [--json]

# Run validation for a specific skill connection only
development-toolchain-validator validate <reference> --skill <skill-id> [--json]

# View validation history
development-toolchain-validator validate history <reference> [--limit 10] [--json]
```

### Drift

```bash
# Check drift for all connections on a reference
development-toolchain-validator drift check --reference <reference> [--json]

# Show drift details for a specific skill
development-toolchain-validator drift show <skill-id> --reference <reference>
```

### Baselines [P1]

```bash
# Run all tooling baselines
development-toolchain-validator baselines run <reference> [--json]

# Run specific baseline
development-toolchain-validator baselines run <reference> --tool <auditor|test-genie|completeness>

# Configure baseline expectations
development-toolchain-validator baselines configure <tool> \
  --reference <reference> \
  [--expected-violations 0] \
  [--expected-all-pass true] \
  [--expected-min-score 96]

# View baseline history
development-toolchain-validator baselines history <reference> [--json]

# Check for regressions
development-toolchain-validator baselines regressions <reference> [--json]
```

### Reports [P1]

```bash
# Full report for a reference
development-toolchain-validator report <reference> [--json]

# Coverage map
development-toolchain-validator coverage <reference> [--json]

# Maturity scores
development-toolchain-validator maturity <reference> [--json]
```

### System

```bash
# Check API health
development-toolchain-validator status

# Show version
development-toolchain-validator version

# Show help
development-toolchain-validator help
```

## Output Modes

All commands support `--json` for machine-readable output. Default output is human-readable formatted text.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation failures detected |
| 2 | Configuration or connection error |
| 3 | API unreachable |
