# The Declared Scenario

Status: shipped on 2026-08-19.

Vrooli scenarios declare build, process, port, storage, readiness, dependency, and peer-wiring facts once in `.vrooli/service.json`. The control plane executes that contract locally (Tier 1), while Scenario Dependency Analyzer projects the same contract into desktop bundles (Tier 2). Neither tier reconstructs scenario behavior from lifecycle shell strings.

## Contract boundary

The canonical schema is `.vrooli/schemas/service.schema.json`, and `internal/scenario` is its Go representation. Every scenario manifest in the fleet validates against it; the count is a census output, not a contract term.

Each long-running process is one `components` entry:

```json
{
  "ports": {
    "api": {"env_var": "API_PORT", "range": "15000-19999"}
  },
  "components": {
    "api": {
      "role": "api",
      "build": {
        "kind": "go_module",
        "dir": "api",
        "output": "api/example-api"
      },
      "run": {
        "argv": ["{{bin.api}}"],
        "cwd": "api",
        "port": "api",
        "data_dirs": ["data"],
        "log_dir": "logs",
        "readiness": {"type": "http", "path": "/health", "timeout_ms": 30000}
      }
    }
  }
}
```

`build.kind` selects a registered native builder. The shipped registry covers the fleet with `go_module`, `pnpm_vite`, `node_bundle`, and `reuse`. Reserved future kinds fail explicitly instead of falling through to shell. Component artifact freshness is derived from `build`; lifecycle setup conditions are not a second freshness authority.

`run.argv` is an argument vector. It is never interpreted by a shell. `cwd`, `env`, `port`, `data_dirs`, `log_dir`, `readiness`, `depends_on`, and `supervised_by` carry the facts previously embedded in commands and naming conventions. Port keys publish their declared `env_var`; there is no step-name port inference.

Roles determine stable launch order when graph edges do not: APIs first, workers and sidecars next, UIs last. `depends_on` supports `wait: started|ready`; `ready` requires a component readiness probe. `supervised_by` is a process-ownership edge and participates in cycle validation.

## Lifecycle execution

The setup and develop phases are derived:

1. Setup provisions governed shared packages and executes the registered builder for every non-reused component.
2. Develop topologically orders components and launches exactly one tracked process per component.
3. The platform seam configures process groups and termination for the host OS.

Lifecycle steps remain only for finite provisioning operations that are not component builds. Their sole execution field is `exec: string[]`; the removed `run` string has no schema field, Go field, reader, compatibility alias, or fallback. The surviving steps are all argv-native; `structure-health fleet census` reports how many there are.

The fleet omits phase objects that contain no steps. Both canonical templates
therefore generate a component manifest with lifecycle health metadata but no
empty setup, develop, or test phase.

The manifest `test` phase was deleted because `vrooli scenario test` delegates directly to test-genie. Setup freshness conditions were deleted because component builders own freshness. Structure-health no longer carries rules for either retired shape.

`internal/lifecycle` contains a build-failing portability assertion that rejects reintroduced `BashCommand`, direct `bash`/`sh -c`, `pkill`, or `ps` executor paths outside tests. `SCENARIO_SHELL_FORBIDDEN` separately rejects shell interpreters and shell files in declared lifecycle/component argv.

Template Manager uses the same process boundary. Generation and relocation
hooks declare `argv`, optional environment overrides, and a working directory;
they execute without a command shell. Hooks cannot install or update
dependencies, tidy modules, or format generated files. Component builders and
Scenario Dependency Analyzer own dependency preparation, while template source
must already satisfy formatting and lint contracts. Generated CLIs are
installed only by the control-plane `cliinstall` boundary.

## Ports and runtime storage

Tier 1 allocates every declared port through the control plane and expands only the closed `${NAME}`/`$NAME` environment language. Missing variables, unsupported shell defaults, and dotted expression syntax are errors.

Component `data_dirs` and `log_dir` are created before launch through the owned-directory seam. Scenarios observe host state; host remediation remains a control-plane responsibility.

Resources also receive typed declarations rather than lifecycle scripts. For
example, the landing-page template declares
`dependencies.resources.postgres.database`; Tier 1 passes that declaration to
Postgres's native, idempotent `ensure` capability before component launch. The
resource creates the per-scenario database through its managed service
connection, and its native content-removal path owns cleanup.

## Peer scenarios

Scenario dependencies use typed bindings:

```json
{
  "dependencies": {
    "scenarios": {
      "landing-page-business-suite": {
        "required": false,
        "startup_policy": "try_start",
        "degraded_behavior": "Local work remains available.",
        "bundle_policy": "discover",
        "bindings": [
          {
            "env_var": "BAS_API_URL",
            "form": "http_base_url",
            "port": "api",
            "when_unavailable": "omit"
          }
        ]
      }
    }
  }
}
```

Tier 1 publishes an atomic peer record under `<runtime-home>/peers/<scenario>.json` after the durable process and allocated ports exist. Records contain scenario identity, service ports, and the control API endpoint; shutdown removes the record. `internal/scenarioenv` resolves bindings from those records and emits named errors for unavailable required peers.

Tier 2 uses the same record codec. A desktop bundle either embeds a peer or discovers it. Embedded peer services are projected into the bundle with a `<peer>--` service prefix. Discovered peers are resolved from the desktop peer directory and injected into every service environment according to the same binding rules.

The final live Tier 1 proof started Landing Page Business Suite and Browser
Automation Studio together through the control plane. LPBS published API port
17691; the environment of the freshly launched argv-native BAS API process
contained `BAS_ENTITLEMENT_SERVICE_URL=http://127.0.0.1:17691`. BAS ran all of its
declared components. Stopping both scenarios removed both peer records.

## Desktop projection

Scenario Dependency Analyzer reads `components` as the only deployment service authority. Bundle file inventory is also derived from component build directories and outputs. The former filesystem scanner and all folder-to-service inference code were deleted.

Projection preserves:

- service identity, role, argv, cwd, environment, and dependency edges;
- build artifact location for Linux, macOS, and Windows;
- port ranges and `${service.port}` references;
- data/log directories and readiness probes;
- UI `dist_root`/assets;
- peer bundle policy and bindings.

Desktop control IPC declares port `0`. The runtime allocates it from the same collision-safe manager used for service ports, so concurrently installed bundles start without a fixed-port collision. The runtime publishes the allocated IPC port in its peer record.

The repository currently contains two committed desktop bundle manifests (`hello-desktop` and `browser-automation-studio`), both with `ipc.port: 0`. Fidelity tests also build manifests directly from the real Tier 1 scenarios. Browser Automation Studio contributes its API, UI, sidecar, and peer edge; Scenario to MCP contributes its reused registry process.

## Generated-template acceptance

Both canonical templates were generated from clean disposable destinations
with native hooks enabled and then started only through `vrooli scenario
start`. The React template reached health with exactly its declared API and UI
processes on ports 16841 and 23953. The landing-page template reached health
with exactly its declared API and UI processes on ports 19213 and 22392 after
its typed Postgres database was ensured. Both instances stopped through the
control plane; Template Manager removed both generated trees, and the landing
database was removed through Postgres's native content command. No disposable
scenario, relocated proto tree, or database remains.

## Security and host requirements

Resource exports remain the authority for resource-derived environment values. Structure-health enforces that component environment values do not duplicate resource exports, contain secret-bearing literals, or hardcode peer loopback ports.

Host tools can declare `min_version`; setup validates those requirements before scenario-owned work. UI components must serve built production assets. Readiness is explicit metadata, not a sleep or command-chain convention.

## Enforced invariants

The final structural gates include:

- `SCENARIO_MANIFEST_INVALID` — enforced canonical-schema validation;
- `SCENARIO_COMPONENT_INVALID` — build/process/port/graph integrity;
- `SCENARIO_SHELL_FORBIDDEN` — no shell-owned declared work;
- `SCENARIO_PORT_ENV_CONVENTION` — one port-key/env-var convention;
- `SCENARIO_PEER_BINDING_INVALID` and `SCENARIO_HARDCODED_PEER_ADDRESS`;
- `SCENARIO_REDECLARES_RESOURCE_ENV` and `SCENARIO_SECRET_LITERAL`;
- `SCENARIO_UI_SERVES_BUILD`.

The deterministic final census of 2026-08-19 is recorded in [census-final.json](declared-scenario-evidence/census-final.json); its manifest, adopter, component and step totals are population figures that move with the fleet, so read them there rather than here. What this contract fixes are the zeros: no live shell syntax, no lifecycle-invoked shell reference, and no canonical-schema violation. Re-run `structure-health fleet census` to confirm they are still zero.

## Shell artifacts that remain

Shell files can still be test fixtures, examples, investigations, external hook payloads, CLI bootstrap entrypoints, or explicit operator utilities. They are not lifecycle authority. Every remaining scenario shell file has an individual disposition in [scenario-shell-dispositions.md](declared-scenario-evidence/scenario-shell-dispositions.md).

## Implementation decisions that differed from the plan

- The plan's literal `grep -l '"run":'` acceptance command was invalid after migration because `components.*.run` is the canonical object. The structural lifecycle query is the valid proof and returns zero lifecycle `run` fields.
- The plan expected num[sot]:8 committed desktop bundles; the repository contains num[sot]:2. Real-scenario manifest construction tests cover the wider projection contract without inventing artifacts.
- Template Manager contained num[sot]:2 `make test` orientation checks, not num[sot]:4. Both now execute `['make', 'test']` through the platform seam.
- An enforced rule over every `.sh` file would contradict the plan's required “kept with reason” dispositions. The shipped rule therefore owns declared execution surfaces, while the census owns file inventory.
- The full fleet structure-health HTTP rollup exceeded its client deadline. The local canonical-schema/shell tests and deterministic census provide the exact manifest gates without weakening them.

## Evidence

- [Final census](declared-scenario-evidence/census-final.json)
- [Cleanup ledger](declared-scenario-evidence/cleanup-ledger.md)
- [Shell-file dispositions](declared-scenario-evidence/scenario-shell-dispositions.md)
- [Tier fidelity: hello-desktop](declared-scenario-evidence/tier-fidelity-hello-desktop.md)
- [Tier fidelity: browser-automation-studio](declared-scenario-evidence/tier-fidelity-browser-automation-studio.md)
- [Phase 6 equivalence table](declared-scenario-evidence/equivalence-table.md)

This is one contract projected into both tiers. Tier 1 and Tier 2 may differ in packaging and process supervision, but they do not differ in what the scenario declares.
