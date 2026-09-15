# CLI Commands — Quality Health

The CLI must stay thin over the API. It owns argument parsing, output formatting, exit codes, and API invocation; it must not duplicate contract evaluation logic.

## audit

```bash
quality-health audit run <scenario> [--surface <id>] [--rule <id>] [--commands] [--autofix-preview] [--json]
```

Runs a Quality Health audit. Human output groups findings by remediation path. JSON output follows the API response shape.

## contracts list

```bash
quality-health contracts list [--language <language>] [--framework <framework>] [--json]
```

Lists contract registry entries and their applicability.

## explain

```bash
quality-health explain finding <finding-id> [--scenario <scenario>] [--rule <rule-id>] [--json]
```

Explains a stable finding ID, why it matters, and the exact repair path.

## fix-config

```bash
quality-health fix-config run <scenario> [--rule <id>] [--dry-run] [--apply] [--json]
quality-health fix-config apply <scenario> [--rule <id>] [--json]
```

Previews or applies deterministic config fixes. Dry-run is the default safety posture. Applying changes requires either `fix-config run --apply` or the explicit `fix-config apply` command.

## Existing Commands

The cli-core standard `status` and `configure` commands remain available.
