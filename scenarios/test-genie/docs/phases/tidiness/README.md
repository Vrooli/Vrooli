# Tidiness Phase

The `tidiness` phase delegates file/function quality checks to the **tidiness-manager** scenario (the same provider scenario-auditor uses for type-safety rules) and maps the returned violations into findings. It is a **finding producer** (source=`tidiness`): its output feeds the ecosystem-manager `tidiness` dimension.

## How It Runs

Test Genie resolves the `tidiness-manager` API via `api-core/discovery`, then calls:

```
POST {tidiness-manager}/api/v1/scan/type-safety
{ "scenario_name": "<scenario>", "include_patterns": true }
```

Each returned violation becomes one `tidiness`-source finding (code = rule id, severity normalized, location = file path).

## Skip Behaviour (not a failure)

`tidiness` is an optional external dependency. If tidiness-manager is not reachable, or the scan call fails, the phase **skips with a clear message** and does not fail the suite — mirroring the runnability-gate pattern. Start tidiness-manager to enable it.

## Opt-Out

```bash
test-genie execute <scenario> --skip tidiness
```
