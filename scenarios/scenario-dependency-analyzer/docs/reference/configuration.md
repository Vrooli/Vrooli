# Configuration

## Environment variables

Key variables include `API_PORT`, `UI_PORT`, `VROOLI_SCENARIOS_DIR`, and API base variables consumed by the UI.

During a release qualification run, set
`SCENARIO_DEPENDENCY_ANALYZER_DEPLOYMENT_MANAGER_READINESS_TOKEN` to enable the
authenticated owner callback for `dependencies-governed`. The callback takes
the exact candidate identity from the matching
`SCENARIO_DEPENDENCY_ANALYZER_DEPLOYMENT_MANAGER_READINESS_*` variables:
`SCENARIO`, `PROFILE_ID`, `CANDIDATE_COMMIT`, `ARTIFACT_DIGEST`, `TARGETS`,
`CHANNEL`, and `POLICY_VERSION`. Release-bound identities must also provide
`CANDIDATE_ID`, `DESTINATION_REVISION_ID`, and `AUTHORIZATION_EPOCH` together.
Set `..._URL` when discovery is unavailable. If the token is present but
identity configuration is incomplete, dependency validation fails closed
instead of emitting an unbound observation.

## Service manifest (`.vrooli/service.json`)

The manifest declares SQLite storage, scenario dependencies on `proto-health` and `code-facts`, resource dependencies, setup steps, and lifecycle commands. Scenario edges retain their canonical `bundle_policy` and optional `versionRange`; `include_ui` opts a peer's UI into the desktop bundle while its API and worker services remain eligible for embedding. Peer ports are resolved at call time and are not written as static environment bindings.

```json
{
  "dependencies": {
    "scenarios": {
      "helper-scenario": {
        "bundle_policy": "either",
        "versionRange": ">=1.2.0 <2.0.0",
        "include_ui": true
      }
    }
  }
}
```

`embed` requires the peer's local service manifest and declared build inputs.
`either` includes the compatible local copy when it is available and otherwise
uses discovery. `discover` never materializes local peer services. Required
resources from each embedded peer participate in the desktop resource plan;
optional and disabled resources do not become required plan entries.

## Schema bootstrap

SQLite schema ownership lives in API domain packages and is applied during API startup.

## CLI config file

The CLI uses shared Vrooli CLI configuration and API-base resolution for local scenario calls.

## API-base resolution precedence

The UI and CLI prefer explicit environment/config values, then lifecycle-discovered ports, then documented local defaults where the shared helper allows them.

## Test/CI configuration

`.vrooli/testing.json` enables strict Go and Node lint handlers and the scenario test suite owns phase-level validation.

## Cross-references

- `../../.vrooli/service.json`
- `../../.vrooli/testing.json`
