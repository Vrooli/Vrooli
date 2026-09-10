# scenario-to-cloud CLI commands

<!-- Generated from cli/manifest.json by `STC_WRITE_CLI_DOC=1 go test ./cli/ -run TestCLIDocMatchesManifest`. Do not edit by hand. -->

Operate cloud deployments over the canonical domain: identities and selectors, executable plans reviewed before apply, durable operations with server-side waits, recovery points, credentials, edge and publication. Typed surfaces use the generated Connect clients; exit codes are 0 ok, 1 failed, 2 refused, 3 pending, 124 observer timeout.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Succeeded, or nothing to change (`no_op`) |
| 1 | Failed: the operation ended `failed`, `failed_recovery` or `cancelled`, or the server answered an untyped error |
| 2 | Refused before any effect: authority, ambiguous selector, stale or mismatched plan digest, request-key conflict, invalid flag grammar |
| 3 | Pending: an operation is admitted but not terminal, or input is required (the durable handoff reference is printed) |
| 124 | Observer timeout: the server-side wait bound elapsed; the operation is unchanged and the reattach command is printed |

## Deployment selectors

Every command that targets a deployment accepts exactly one selector: `--deployment <id>` (or the id as the positional argument), `--scenario <id> --environment <env>`, `--scenario <id> --domain <domain>`, or `--scenario <id> --host <host>`. The selector is resolved through `DeploymentsService.ResolveDeployment`; more than one match is refused with `deployment_selector_ambiguous` (exit 2) and the candidate ids are listed. Identities are printed the same way everywhere: `deployment: <id>  scenario: <id>  environment: <env>  target: <machine:id|host:host> (<transport>)`.

Machine output (`--json`) is the server's typed message as proto JSON (snake_case field names, lossless); human output never has to be parsed.

## Command groups

### manifest

Cloud manifest authoring and validation

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `manifest validate` | read | true | local | Validate a cloud manifest |
| `manifest schema` | read | true | local | Print the manifest JSON schema |
| `manifest init` | write | true | local | Write a starter manifest for a scenario |
| `manifest template` | read | true | local | Print a manifest template |
| `manifest doctor` | read | true | local | Diagnose a manifest against the repository |
| `manifest fix` | write | true | local | Apply the manifest fixes the doctor proposes |

Examples:

```bash
scenario-to-cloud manifest validate <manifest> [--json]
scenario-to-cloud manifest schema [--json]
scenario-to-cloud manifest init [--scenario <value>] [--host <value>] [--domain <value>] [--out <value>] [--json]
scenario-to-cloud manifest template [--json]
scenario-to-cloud manifest doctor <manifest> [--json]
scenario-to-cloud manifest fix <manifest> [--json]
```

### bundle

Release bundles (local store and target)

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `bundle build` | write | true | local | Build the mini-Vrooli bundle for a manifest |
| `bundle list` | read | true | local | List local bundles |
| `bundle stats` | read | true | local | Local bundle store statistics |
| `bundle delete` | destructive | false | local | Delete one local bundle |
| `bundle cleanup` | destructive | false | local | Prune local bundles per scenario |
| `bundle vps-list` | read | true | local | List bundles retained on the deployment's target (GET /deployments/{id}/bundles/vps) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `bundle vps-gc` | destructive | false | local | Garbage-collect target bundles keeping the newest (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |

Examples:

```bash
scenario-to-cloud bundle build <manifest> [--json]
scenario-to-cloud bundle list [--json]
scenario-to-cloud bundle stats [--json]
scenario-to-cloud bundle delete [--sha <value>] [--filename <value>] [--json]
scenario-to-cloud bundle cleanup [--scenario <value>] [--keep <value>] [--json]
scenario-to-cloud bundle vps-list [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud bundle vps-gc [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--keep <value>] [--dry-run] [--json]
```

### deployment

Deployment lifecycle over identities, executable plans and durable operations

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `deployment create` | write | true | local | Create or update a deployment record from a manifest |
| `deployment resolve` | read | true | DeploymentsService.ResolveDeployment | Resolve a selector to its stable identity (the facets fill the nested selector message); ambiguity is refused (exit 2) with candidates (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment list` | read | true | DeploymentsService.ListDeployments | List deployments (proto JSON with --json) |
| `deployment get` | read | true | DeploymentsService.GetDeployment | Read a deployment record (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment delete` | destructive | false | local | Delete a deployment record (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment plan` | read | true | PlansService.CompilePlan | Preview the executable plan: digest, changes, data effects, downtime, recovery strategy, handoff (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment apply` | destructive | true | PlansService.ApplyPlan | Admit a reviewed plan digest as a durable operation and wait once (run-eligible under a scenario-to-cloud:destructive session grant; confirmation required) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment execute` | destructive | false | local | Plan, print the review (unless --yes), apply the exact digest shown and wait once (CompilePlan then ApplyPlan) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment start` | destructive | false | local | Plan and apply the start scope (CompilePlan then ApplyPlan with scope start) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment stop` | destructive | false | local | Stop the workload on the target (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment history` | read | true | local | Deployment history events (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment health` | read | true | HealthService.GetHealthObservation | Typed health observation: status, freshness, checks, next actions (unknown is never healthy) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment rollback` | destructive | true | local | Governed rollback to the published predecessor: --dry-run issues a preview reference, --confirm executes it (run-eligible under a scenario-to-cloud:destructive session grant; confirmation required) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |

Examples:

```bash
scenario-to-cloud deployment create <manifest> [--name <value>] [--json]
scenario-to-cloud deployment resolve [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud deployment list [--scenario <value>] [--environment <value>] [--status <value>] [--page-size <value>] [--page-token <value>]
scenario-to-cloud deployment get [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud deployment delete [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--stop] [--cleanup] [--json]
scenario-to-cloud deployment plan [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--scope <value>] [--force-bundle] [--show-commands] [--json]
scenario-to-cloud deployment apply [deployment_id] --plan-digest <plan-digest> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--scope <value>] [--request-key <value>] [--preflight] [--no-wait] [--timeout <value>] [--json]
scenario-to-cloud deployment execute [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--yes] [--scope <value>] [--force-bundle] [--show-commands] [--request-key <value>] [--preflight] [--no-wait] [--timeout <value>] [--json]
scenario-to-cloud deployment start [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--yes] [--force-bundle] [--show-commands] [--request-key <value>] [--preflight] [--no-wait] [--timeout <value>] [--json]
scenario-to-cloud deployment stop [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud deployment history [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud deployment health [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud deployment rollback [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--dry-run] [--confirm] [--preview-ref <value>] [--review-ref <value>] [--release-digest <value>] [--repair-bundle-sha256 <value>] [--expected-bundle-sha256 <value>] [--data-compatibility <value>] [--request-key <value>] [--json]
```

### deployment recovery-points

Consistent recovery points and restore

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `deployment recovery-points list` | read | true | local | Recovery points of the deployment (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment recovery-points capture` | write | false | local | Capture a consistent recovery point (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment recovery-points verify` | read | true | local | Verify a recovery point (--open also proves the recovery key resolves) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `deployment recovery-points restore` | destructive | true | local | Restore a recovery point onto a replacement target; --dry-run verifies and calls no restore (run-eligible under a scenario-to-cloud:destructive session grant; confirmation required) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |

Examples:

```bash
scenario-to-cloud deployment recovery-points list [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud deployment recovery-points capture [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--retention <value>] [--recovery-point <value>] [--release-digest <value>] [--json]
scenario-to-cloud deployment recovery-points verify [deployment_id] --recovery-point <recovery-point> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--open] [--json]
scenario-to-cloud deployment recovery-points restore [deployment_id] --recovery-point <recovery-point> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--target-ref <value>] [--into <value>] [--dry-run] [--json]
```

### publication

Governed publication bound to an approved review

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `publication request` | write | false | EvidenceService.RequestPublication | Open a publication and prepare its review (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `publication apply` | destructive | false | EvidenceService.ApplyPublication | Activate the approved publication by its reviewed digest (approval re-checked server-side) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `publication status` | read | true | EvidenceService.GetPublication | Publication standing (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |

Examples:

```bash
scenario-to-cloud publication request [deployment_id] --request-key <request-key> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--profile <value>] [--candidate-commit <value>] [--artifact-digest <value>] [--channel <value>] [--json]
scenario-to-cloud publication apply [deployment_id] --request-key <request-key> --review-ref <review-ref> --plan-digest <plan-digest> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud publication status [deployment_id] --request-key <request-key> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
```

### redeploy (top level)

Manifest-first convenience wrapper

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `redeploy` | destructive | false | local | Create or update the record from a manifest, then plan, review, apply and wait once (same path as deployment execute) |

Examples:

```bash
scenario-to-cloud redeploy <manifest> [--name <value>] [--yes] [--no-wait] [--request-key <value>] [--timeout <value>] [--force-bundle] [--preflight] [--show-commands] [--json]
```

### preflight

Target readiness checks and host fixes

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `preflight run` | read | true | local | Run preflight checks for a manifest |
| `preflight requirements` | read | true | local | Print the canonical VPS requirement policy |
| `preflight fix-firewall` | destructive | false | local | Open the required inbound ports on the target (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `preflight fix-processes` | destructive | false | local | Stop stale scenario processes on the target (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `preflight disk-usage` | read | true | local | Disk usage on the target (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `preflight disk-cleanup` | destructive | false | local | Run declared cleanup actions on the target (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |

Examples:

```bash
scenario-to-cloud preflight run <manifest> [--json]
scenario-to-cloud preflight requirements [--json]
scenario-to-cloud preflight fix-firewall [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--user <value>] [--key-path <value>] [--ssh-port <value>] [--json]
scenario-to-cloud preflight fix-processes [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--user <value>] [--key-path <value>] [--ssh-port <value>] [--workdir <value>] [--json]
scenario-to-cloud preflight disk-usage [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--user <value>] [--key-path <value>] [--ssh-port <value>] [--json]
scenario-to-cloud preflight disk-cleanup [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--action <value>] [--user <value>] [--key-path <value>] [--ssh-port <value>] [--json]
```

### vps

Manifest-scoped VPS setup/deploy plans and disposable QEMU instances

### vps setup

Host setup plan

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `vps setup plan` | read | true | local | Compile the setup plan |
| `vps setup apply` | destructive | false | local | Apply the setup plan by its reviewed digest (compiles first when no digest is given) |

Examples:

```bash
scenario-to-cloud vps setup plan <manifest> <bundle>
scenario-to-cloud vps setup apply <manifest> <bundle> [--plan-digest <value>]
```

### vps deploy

Runtime deploy plan

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `vps deploy plan` | read | true | local | Compile the deploy plan |
| `vps deploy apply` | destructive | false | local | Apply the deploy plan by its reviewed digest (compiles first when no digest is given) |

Examples:

```bash
scenario-to-cloud vps deploy plan <manifest>
scenario-to-cloud vps deploy apply <manifest> [--plan-digest <value>]
```

### vps instance

Disposable local QEMU instances

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `vps instance plan` | read | true | local | Validate the local QEMU lane |
| `vps instance create` | write | false | local | Register a disposable VM |
| `vps instance start` | write | false | local | Start a registered VM |
| `vps instance stop` | write | false | local | Stop a registered VM |
| `vps instance wait-for-ssh` | read | true | local | Wait until the VM accepts SSH |
| `vps instance snapshot` | write | false | local | Snapshot a VM |
| `vps instance reset` | destructive | false | local | Reset a VM to a snapshot |
| `vps instance destroy` | destructive | false | local | Destroy a VM |

Examples:

```bash
scenario-to-cloud vps instance plan [--name <value>] [--image <value>] [--workdir <value>]
scenario-to-cloud vps instance create [--name <value>] [--image <value>] [--workdir <value>] [--memory <value>] [--cpus <value>] [--profile <value>] [--user <value>] [--authorized-key <value>] [--ssh-port <value>]
scenario-to-cloud vps instance start <instance_id>
scenario-to-cloud vps instance stop <instance_id>
scenario-to-cloud vps instance wait-for-ssh <instance_id>
scenario-to-cloud vps instance snapshot <instance_id> [snapshot]
scenario-to-cloud vps instance reset <instance_id> [snapshot]
scenario-to-cloud vps instance destroy <instance_id>
```

### inspect

Remote state, drift, metrics, bounded logs and files

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `inspect plan` | read | true | local | Preview the inspection commands for a manifest |
| `inspect status` | read | true | local | Fetch remote status and logs for a manifest |
| `inspect live` | read | true | local | Live state of a deployment |
| `inspect drift` | read | true | local | Drift between desired and observed state |
| `inspect metrics` | read | true | local | Deployment metrics |
| `inspect logs` | read | true | local | Bounded, API-redacted log retrieval (--tail max 2000, --max-bytes bounds the output) |
| `inspect files` | read | true | local | List or read files under the deployment workdir |

Examples:

```bash
scenario-to-cloud inspect plan <manifest> [--json]
scenario-to-cloud inspect status <manifest> [--json]
scenario-to-cloud inspect live <deployment_id> [--json]
scenario-to-cloud inspect drift <deployment_id> [--json]
scenario-to-cloud inspect metrics <deployment_id> [--json]
scenario-to-cloud inspect logs <deployment_id> [--source <value>] [--level <value>] [--search <value>] [--tail <value>] [--since <value>] [--max-bytes <value>] [--json]
scenario-to-cloud inspect files <deployment_id> [--content] [--json]
```

### process

Remote process control

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `process kill` | destructive | false | local | Kill a remote process |
| `process restart` | destructive | false | local | Restart a remote scenario process |
| `process control` | destructive | false | local | Scenario control action on the target |
| `process vps-action` | destructive | false | local | Host-level action on the target |

Examples:

```bash
scenario-to-cloud process kill <deployment_id> <pid> [--signal <value>] [--json]
scenario-to-cloud process restart <deployment_id> [--json]
scenario-to-cloud process control <deployment_id> <action> [--json]
scenario-to-cloud process vps-action <deployment_id> <action> [--json]
```

### edge

Edge routing, DNS and TLS

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `edge status` | read | true | EdgeService.GetEdgeObservation | Typed edge observation: routes, private listeners, DNS, TLS, readiness (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `edge dns-check` | read | true | local | Check DNS configuration |
| `edge dns-records` | read | true | local | List DNS records |
| `edge caddy` | destructive | false | local | Caddy control (reload, restart, status, validate) |
| `edge tls` | read | true | local | TLS certificate information |
| `edge tls-renew` | destructive | false | local | Renew TLS certificates |

Examples:

```bash
scenario-to-cloud edge status [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud edge dns-check <deployment_id> [--json]
scenario-to-cloud edge dns-records <deployment_id> [--json]
scenario-to-cloud edge caddy <deployment_id> <action> [--json]
scenario-to-cloud edge tls <deployment_id> [--json]
scenario-to-cloud edge tls-renew <deployment_id> [--domain <value>] [--force] [--json]
```

### secrets

Workspace, scenario and deployment secrets (values never in arguments)

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `secrets set` | destructive | false | local | Set a secret on the selected targets |
| `secrets get` | destructive | false | local | Read secret metadata (or the value with --reveal) |
| `secrets verify` | read | true | local | Verify a secret matches across targets |
| `secrets delete` | destructive | false | local | Delete a secret from the selected targets |
| `secrets plan-get` | read | true | local | Read the scenario secrets plan |

Examples:

```bash
scenario-to-cloud secrets set <key> [--value <value>] [--generate <value>] [--targets <value>] [--restart] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--all-deployments] [--json]
scenario-to-cloud secrets get <key> [--targets <value>] [--reveal] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--all-deployments] [--json]
scenario-to-cloud secrets verify <key> [--targets <value>] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--all-deployments] [--json]
scenario-to-cloud secrets delete <key> [--targets <value>] [--restart] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--all-deployments] [--json]
scenario-to-cloud secrets plan-get <scenario_id> [--json]
```

### scenario

Scenario inventory used by manifests

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `scenario list` | read | true | local | List deployable scenarios |
| `scenario ports` | read | true | local | Ports a scenario declares |
| `scenario deps` | read | true | local | Dependency closure of a scenario |

Examples:

```bash
scenario-to-cloud scenario list [--json]
scenario-to-cloud scenario ports <scenario_id> [--json]
scenario-to-cloud scenario deps <scenario_id> [--impact] [--verbose] [--json]
```

### task

Investigation tasks

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `task create` | write | true | local | Create an investigation task |
| `task list` | read | true | local | List tasks |
| `task get` | read | true | local | Read a task |
| `task stop` | write | true | local | Stop a task |
| `task agent-status` | read | true | local | Investigation agent status |

Examples:

```bash
scenario-to-cloud task create <deployment_id> [--type <value>] [--json]
scenario-to-cloud task list [--json]
scenario-to-cloud task get <task_id> [--json]
scenario-to-cloud task stop <task_id> [--json]
scenario-to-cloud task agent-status [--json]
```

### operation

Durable cloud operations: attach by id across disconnects and owner restarts

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `operation get` | read | true | OperationsService.GetOperation | Typed standing of one operation (exit 3 while non-terminal; proto JSON with --json) |
| `operation wait` | read | true | OperationsService.WaitOperation | Block once server-side until terminal; exit 124 when the observer bound elapses (operation unchanged) |
| `operation resume` | read | true | local | Attach to an operation started elsewhere (API, UI) after a disconnect; same contract as wait (WaitOperation) |
| `operation cancel` | write | false | OperationsService.CancelOperation | Record a cancellation intent, honoured at the next declared cancel point |
| `operation list` | read | true | OperationsService.ListDeploymentOperations | Operations admitted against a deployment (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |

Examples:

```bash
scenario-to-cloud operation get <operation_id>
scenario-to-cloud operation wait <operation_id> [--timeout <value>]
scenario-to-cloud operation resume <operation_id> [--timeout <value>] [--json]
scenario-to-cloud operation cancel <operation_id>
scenario-to-cloud operation list [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
```

### credential

Deployment credential bindings (values only on standard input)

| Command | Effect | Agent-runnable | Binding | Description |
|---|---|---|---|---|
| `credential list` | read | true | CredentialsService.ListBindings | Bindings, versions, consumers and acknowledgements (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `credential rotate` | destructive | false | CredentialsService.RotateCredential | Rotate one binding (versioned; consumers verified before the predecessor is retired) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `credential revoke` | destructive | false | CredentialsService.RevokeCredential | Revoke one binding everywhere it was distributed (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `credential recover` | destructive | false | CredentialsService.RecoverCredentials | Recover the credential store onto a replacement host (passphrase on stdin) (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `credential rotation-get` | read | true | CredentialsService.GetRotation | Standing of one credential operation (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |
| `credential rotation-resume` | destructive | false | CredentialsService.ResumeRotation | Resume an operation parked on operator input (--deployment <id> \| <id> \| --scenario <id> --environment <env> \| --scenario <id> --domain <domain> \| --scenario <id> --host <host>) |

Examples:

```bash
scenario-to-cloud credential list [deployment_id] [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud credential rotate [deployment_id] --binding <binding> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--value-stdin] [--request-key <value>] [--json]
scenario-to-cloud credential revoke [deployment_id] --binding <binding> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--request-key <value>] [--json]
scenario-to-cloud credential recover [deployment_id] --bundle-ref <bundle-ref> --passphrase-stdin [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--request-key <value>] [--json]
scenario-to-cloud credential rotation-get [deployment_id] --rotation <rotation> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--json]
scenario-to-cloud credential rotation-resume [deployment_id] --rotation <rotation> [--deployment <value>] [--scenario <value>] [--environment <value>] [--domain <value>] [--host <value>] [--operator-confirmed] [--json]
```

## RPCs without a direct command

| Service | Method | Reason |
|---|---|---|
| OperationsService | ReconcileOperations | Owner-only reconciliation pass; run by the API on startup and periodically, not an operator command. |
| CredentialsService | BreakGlass | Emergency access requires the confirmation string on the API surface with an audited operator identity; no headless alias. |
| EvidenceService | GetReleaseEvidence | Evidence cells are read through Deployment Manager's readiness surfaces; the CLI exposes the publication standing only. |
| DataService | CaptureRecoveryPoint | DataService is not mounted by the API; recovery points use the REST routes under deployment recovery-points. |
| DataService | ListRecoveryPoints | See CaptureRecoveryPoint: REST route used. |
| DataService | RestoreRecoveryPoint | See CaptureRecoveryPoint: REST route used. |
| DataService | VerifyRecoveryPoint | See CaptureRecoveryPoint: REST route used. |
| DataService | EvaluateRollback | Rollback admission is evaluated inside deployment rollback --dry-run through the recovery route. |
| DataService | PruneRecoveryPoints | Retention pruning is owned by the API's retention policy, not an operator command. |
| ReleasesService | BuildRelease | Release identity is built by the API during execute/plan; bundle build is the operator's input. |
| ReleasesService | GetRelease | Release facts are printed inside deployment plan/get output. |
| ReleasesService | VerifyRelease | Verification runs as the release.verify plan action; no standalone operator verb. |

