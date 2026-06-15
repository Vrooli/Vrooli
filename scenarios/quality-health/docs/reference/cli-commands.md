# CLI Commands — Quality Health

The CLI must stay thin over the API. It owns argument parsing, output formatting, exit codes, and API invocation; it must not duplicate contract evaluation logic.

## audit

```bash
quality-health audit <scenario> [--surface <id>] [--rule <id>] [--commands] [--autofix-preview] [--json]
```

Runs a Quality Health audit. Human output groups findings by remediation path. JSON output follows the API response shape.

## contracts list

```bash
quality-health contracts list [--language <language>] [--framework <framework>] [--json]
```

Lists contract registry entries and their applicability.

## explain

```bash
quality-health explain <finding-id> [--scenario <scenario>] [--json]
```

Explains a stable finding ID, why it matters, and the exact repair path.

## fix-config

```bash
quality-health fix-config <scenario> [--rule <id>] --dry-run [--json]
quality-health fix-config <scenario> [--rule <id>] --apply [--json]
```

Previews or applies deterministic config fixes. Dry-run is the default safety posture. `--apply` must be explicit.

## Existing Commands

Only the cli-core standard commands are active after Phase 1. Phase 2 should add the Quality Health commands above.
