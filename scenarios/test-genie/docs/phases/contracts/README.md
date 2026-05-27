# Contracts Phase

The `contracts` phase validates `cli/manifest.json` bindings against the proto descriptors via **cli-health**, ensuring every advertised CLI command resolves to a real API contract and no Connect-RPC method has drifted out of sync with its CLI counterpart.

## How It Runs

Test Genie invokes the cli-health binding validation against the scenario's `cli/manifest.json`, comparing declared command bindings to the generated proto descriptors.

Equivalent operator flow:

```bash
cli-health search <scenario>
```

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip contracts
```

Disable per scenario via `.vrooli/testing.json`:

```json
{
  "phases": {
    "contracts": { "enabled": false }
  }
}
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json` (default: 60s):

```json
{
  "phases": {
    "contracts": { "timeout": "120s" }
  }
}
```
