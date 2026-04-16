# Lint Phase

**ID**: `lint`  
**Timeout**: 30 seconds  
**Optional**: No  
**Requires Runtime**: No

The lint phase discovers top-level scenario components, matches each one to a supported lint handler, runs the matched handler, and applies policy for unmatched components.

This is intentionally **component-based**, not folder-assumption-based. The phase does not assume:

- `api/` means Go
- `ui/` means Node/React
- only `api/`, `ui/`, and `cli/` matter

It looks for lint contracts such as `go.mod`, `package.json`, `pyproject.toml`, or Python source/config evidence at the component root.

## Current Handler Set

| Handler | Match Evidence | Tools |
|---------|----------------|-------|
| `go_module` | `go.mod` | `golangci-lint`, fallback `go vet` |
| `node_package` | `package.json` | `tsc` when `tsconfig.json` exists, `eslint` when config exists |
| `python_project` | `pyproject.toml`, `setup.py`, `requirements.txt`, `pytest.ini`, `mypy.ini`, or Python source | `ruff`, fallback `flake8`, optional `mypy` |

## Discovery Model

The phase evaluates:

- scenario root metadata and root-level files
- direct child directories only

Examples of discovered components:

- `api/`
- `ui/`
- `cli/`
- `worker/`
- `desktop/`
- `proxy/`

It does not recursively hunt for arbitrary nested subprojects.

## Policy

After handler matching, the phase applies policy to unmatched components.

Default policy:

- `api/` present without a supported lint contract: error
- `ui/` present without a supported lint contract: warning
- `cli/` present without a supported lint contract: warning
- other code-bearing unmatched top-level components: warning

This is separate from handler findings. A component can therefore have:

- handler issues such as lint/type findings
- policy issues such as “present without supported lint contract”

## Severity Handling

There are two independent failure paths:

1. Handler execution findings
   - type errors fail the phase
   - lint warnings fail only when strict mode is enabled for that handler/component

2. Policy findings
   - warnings are reported
   - errors fail the phase

## Configuration

Configure lint behavior in `.vrooli/testing.json`:

```json
{
  "lint": {
    "handlers": {
      "go_module": { "enabled": true, "strict": true },
      "node_package": { "enabled": true, "strict": true },
      "python_project": { "enabled": true, "strict": false }
    },
    "policy": {
      "unconfigured_common_components": {
        "api": "error",
        "ui": "warning",
        "cli": "warning"
      },
      "unmatched_code_components": "warning"
    },
    "components": {
      "worker": {
        "handler": "go_module",
        "strict": true
      }
    },
    "ignore": ["docs", "assets", "coverage", "test"]
  }
}
```

## Output

The phase reports by component, not by language.

Example:

```text
[lint] Linting api (go_module)
[lint] Go: golangci-lint found no issues
[lint] Linting ui (node_package)
[lint] Node: eslint found 2 issue(s)
[lint] cli: common component is present without a supported lint contract
[lint] Lint completed with 3 issue(s) across 2 component(s)
```

## See Also

- [Phases Overview](../README.md)
- [Dependencies Phase](../dependencies/README.md)
- [Smoke Phase](../smoke/README.md)
