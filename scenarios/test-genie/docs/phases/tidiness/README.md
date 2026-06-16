# Tidiness Phase

The `tidiness` phase delegates maintainability checks to the **tidiness-manager** scenario and maps the returned findings into Test Genie findings. It is a **finding producer** (source=`tidiness`): its output feeds the ecosystem-manager `tidiness` dimension.

Static-quality contracts, lint policy, type policy, and config strictness are owned by the `quality` phase through **quality-health**.

## How It Runs

Test Genie resolves the `tidiness-manager` API via `api-core/discovery`, then calls:

```
POST {tidiness-manager}/api/v1/scan/tidiness
{ "scenario_name": "<scenario>" }
```

Each returned finding becomes one `tidiness`-source finding (code = rule id or category, severity normalized, location = file path/line when available).

## Skip Behaviour (not a failure)

`tidiness` is an optional external dependency. If tidiness-manager is not reachable, or the scan call fails, the phase **skips with a clear message** and does not fail the suite — mirroring the runnability-gate pattern. Start tidiness-manager to enable it.

## Opt-Out

```bash
test-genie execute <scenario> --skip tidiness
```
