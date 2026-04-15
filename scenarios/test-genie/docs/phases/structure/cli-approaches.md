# CLI Manifest Contract

The structure phase no longer infers scenario CLI behavior from repository layout.

`scenarios/<name>/.vrooli/service.json` is the only platform contract for scenario CLIs. Files such as `cli/go.mod`, `cli/install.sh`, or `cli/<scenario-name>` are implementation assets owned by the adapter declared in the manifest.

## What Test Genie Validates

When `cli.enabled=true`, structure validation checks that `.vrooli/service.json` declares:

- `cli.command`
- `cli.adapter.kind`
- `cli.install`
- `cli.invoke`

It then validates the adapter-owned files referenced by that manifest.

## Supported Adapters

### `go_module`

Use this when the CLI is built from Go source.

Required manifest shape:

```json
{
  "cli": {
    "enabled": true,
    "command": "my-scenario",
    "adapter": {
      "kind": "go_module",
      "module_dir": "cli"
    },
    "install": [
      {
        "os": ["linux", "darwin"],
        "kind": "command",
        "run": "bash ./cli/install.sh"
      }
    ],
    "invoke": {
      "kind": "installed_command",
      "command": "my-scenario"
    }
  }
}
```

Expected adapter assets:

- `cli/go.mod`
- Go source files under `cli/`

### `shell_script`

Use this when the CLI entrypoint is a tracked shell script.

Required manifest shape:

```json
{
  "cli": {
    "enabled": true,
    "command": "my-scenario",
    "adapter": {
      "kind": "shell_script",
      "script_path": "cli/my-scenario",
      "install_script": "cli/install.sh"
    },
    "install": [
      {
        "os": ["linux", "darwin"],
        "kind": "command",
        "run": "bash ./cli/install.sh"
      }
    ],
    "invoke": {
      "kind": "installed_command",
      "command": "my-scenario"
    }
  }
}
```

Expected adapter assets:

- `cli/my-scenario`
- `cli/install.sh`

## Important Constraint

Layout alone does not declare a CLI anymore.

These patterns are not sufficient by themselves:

- `cli/go.mod`
- `cli/main.go`
- `cli/install.sh`
- `cli/<scenario-name>`

Without a valid top-level `cli` block in `service.json`, the platform treats the scenario as not declaring a scenario CLI.

## Migration Guidance

If a scenario still relies on layout-only assumptions:

1. Add a top-level `cli` block to `.vrooli/service.json`
2. Choose `go_module` or `shell_script`
3. Point the adapter at the real implementation assets
4. Keep lifecycle CLI checks aligned with `cli.command`

See [Scenario CLI Manifest Decision](/docs/strategy/scenario-cli-manifest-decision.md) for the platform-level policy.
