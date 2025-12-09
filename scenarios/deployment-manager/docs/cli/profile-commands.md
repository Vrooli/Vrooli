# Profile Commands

Commands for creating, configuring, and managing deployment profiles.

A **profile** captures all configuration needed to deploy a scenario to a specific tier: target tiers, dependency swaps, environment settings, and secrets strategy.

## profiles / profile list

List all deployment profiles.

```bash
deployment-manager profiles
# or
deployment-manager profile list
```

**Output:**

```json
{
  "profiles": [
    {
      "id": "profile-1704067200",
      "name": "picker-wheel-desktop",
      "scenario": "picker-wheel",
      "tiers": [2],
      "version": 3,
      "updated_at": "2025-01-15T12:00:00Z"
    },
    {
      "id": "profile-1704153600",
      "name": "picker-wheel-saas",
      "scenario": "picker-wheel",
      "tiers": [4],
      "version": 1,
      "updated_at": "2025-01-16T12:00:00Z"
    }
  ]
}
```

---

## profile create

Create a new deployment profile.

```bash
deployment-manager profile create <name> <scenario> [--tier <tier>]
```

**Arguments:**
- `<name>` - Human-readable profile name
- `<scenario>` - Target scenario name

**Flags:**
- `--tier <tier>` - Target tier (1-5 or name). Can specify multiple: `--tier 2 --tier 4`

**Example:**

```bash
deployment-manager profile create picker-wheel-desktop picker-wheel --tier 2
```

**Output:**

```json
{
  "id": "profile-1704067200",
  "name": "picker-wheel-desktop",
  "scenario": "picker-wheel",
  "tiers": [2],
  "swaps": {},
  "secrets": {},
  "settings": {},
  "version": 1,
  "created_at": "2025-01-15T12:00:00Z"
}
```

---

## profile show

Retrieve a profile's full configuration.

```bash
deployment-manager profile show <id>
```

**Arguments:**
- `<id>` - Profile ID

**Output:**

```json
{
  "id": "profile-1704067200",
  "name": "picker-wheel-desktop",
  "scenario": "picker-wheel",
  "tiers": [2],
  "swaps": {
    "postgres": "sqlite",
    "redis": "in-process"
  },
  "secrets": {
    "jwt_secret": {"class": "per_install_generated"},
    "session_secret": {"class": "user_prompt"}
  },
  "settings": {
    "env": {
      "LOG_LEVEL": "info",
      "DEBUG": "false"
    }
  },
  "version": 5,
  "created_at": "2025-01-15T12:00:00Z",
  "updated_at": "2025-01-15T14:30:00Z",
  "created_by": "system",
  "updated_by": "system"
}
```

---

## profile update

Update a profile's target tiers.

```bash
deployment-manager profile update <id> [--tier <tier>]
```

**Arguments:**
- `<id>` - Profile ID

**Flags:**
- `--tier <tier>` - New target tier(s)

**Example:**

```bash
# Change from desktop-only to desktop + mobile
deployment-manager profile update profile-123 --tier 2 --tier 3
```

---

## profile delete

Delete a deployment profile.

```bash
deployment-manager profile delete <id>
```

**Arguments:**
- `<id>` - Profile ID

**Confirmation:** The command will prompt for confirmation unless `--json` flag is used.

---

## profile export

Export a profile to a JSON file.

```bash
deployment-manager profile export <id> [--output <path>]
```

**Arguments:**
- `<id>` - Profile ID

**Flags:**
- `--output <path>` - Output file path. Defaults to `<profile-name>.json`

**Example:**

```bash
deployment-manager profile export profile-123 --output ./backups/picker-wheel-desktop.json
```

---

## profile import

Import a profile from a JSON file.

```bash
deployment-manager profile import <path> [--name <override>]
```

**Arguments:**
- `<path>` - Path to JSON file

**Flags:**
- `--name <override>` - Override the profile name from file

**Example:**

```bash
deployment-manager profile import ./backups/picker-wheel-desktop.json --name imported-profile
```

---

## profile set

Set a specific field in a profile.

```bash
deployment-manager profile set <id> <key> [value]
```

**Arguments:**
- `<id>` - Profile ID
- `<key>` - Field to set (including dotted paths like `env`)
- `[value]` - Value to set. If omitted with `env` key, prompts for key-value pair.

**Examples:**

```bash
# Set environment variable
deployment-manager profile set profile-123 env LOG_LEVEL info
deployment-manager profile set profile-123 env DEBUG false

# Set other settings
deployment-manager profile set profile-123 settings.timeout 30000
```

**Special Keys:**
- `env` - Sets environment variables in `settings.env`

---

## profile swap

Manage dependency swaps for a profile.

```bash
deployment-manager profile swap <id> <add|remove> <from> [to]
```

**Arguments:**
- `<id>` - Profile ID
- `<add|remove>` - Operation
- `<from>` - Original dependency name
- `[to]` - Replacement dependency (required for `add`)

**Examples:**

```bash
# Add a swap
deployment-manager profile swap profile-123 add postgres sqlite

# Remove a swap
deployment-manager profile swap profile-123 remove postgres
```

**Notes:**
- Swaps are profile-only; they don't modify the actual scenario
- Use `swaps analyze` first to understand impact

---

## profile versions

Show version history for a profile.

```bash
deployment-manager profile versions <id>
```

**Arguments:**
- `<id>` - Profile ID

**Output:**

```json
{
  "profile_id": "profile-1704067200",
  "versions": [
    {"version": 1, "updated_at": "2025-01-15T12:00:00Z", "updated_by": "system"},
    {"version": 2, "updated_at": "2025-01-15T13:00:00Z", "updated_by": "system"},
    {"version": 3, "updated_at": "2025-01-15T14:00:00Z", "updated_by": "system"}
  ]
}
```

---

## profile diff

Compare two versions of a profile.

```bash
deployment-manager profile diff <id>
```

**Arguments:**
- `<id>` - Profile ID

**Output:** Shows differences between the current version and the previous version.

```json
{
  "profile_id": "profile-1704067200",
  "from_version": 2,
  "to_version": 3,
  "changes": {
    "swaps": {
      "added": {"redis": "in-process"},
      "removed": {}
    },
    "settings.env": {
      "added": {"DEBUG": "false"},
      "changed": {},
      "removed": {}
    }
  }
}
```

---

## profile rollback

Rollback a profile to a previous version.

```bash
deployment-manager profile rollback <id> [--version <n>] [--to-version <n>]
```

**Arguments:**
- `<id>` - Profile ID

**Flags:**
- `--version <n>` - Version to rollback to (alias: `--to-version`)

**Example:**

```bash
# Rollback to version 2
deployment-manager profile rollback profile-123 --version 2
```

**Notes:**
- Creates a new version with the old configuration
- Does not delete intermediate versions

---

## profile analyze

Analyze a profile's current configuration.

```bash
deployment-manager profile analyze <id>
```

**Arguments:**
- `<id>` - Profile ID

**Output:**

```json
{
  "profile_id": "profile-1704067200",
  "fitness_with_swaps": {
    "tier_2": {"overall": 70, "portability": 65, "resources": 75}
  },
  "remaining_blockers": [],
  "warnings": ["Bundle size may exceed 200MB with current assets"]
}
```

---

## profile save

Manually save the current state as a new version.

```bash
deployment-manager profile save <id>
```

**Arguments:**
- `<id>` - Profile ID

**Notes:**
- Most modifications auto-save
- Use this after complex changes to create a restore point

---

## Profile Structure

```json
{
  "id": "profile-{timestamp}",
  "name": "human-readable-name",
  "scenario": "scenario-name",
  "tiers": [2, 3],
  "swaps": {
    "original-dep": "replacement-dep"
  },
  "secrets": {
    "secret-name": {"class": "per_install_generated|user_prompt|infrastructure"}
  },
  "settings": {
    "env": {"KEY": "value"},
    "custom": "settings"
  },
  "version": 1,
  "created_at": "ISO8601",
  "updated_at": "ISO8601",
  "created_by": "system|user",
  "updated_by": "system|user"
}
```

---

## Related

- [Swap Commands](swap-commands.md) - Analyze swap impacts before applying
- [Secret Commands](secret-commands.md) - Configure secrets for the profile
- [Deployment Commands](deployment-commands.md) - Deploy using a profile
