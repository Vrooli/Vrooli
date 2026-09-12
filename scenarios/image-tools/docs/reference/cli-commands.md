# CLI Commands — Image Tools

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/image-tools`, and rebuilt automatically when its
sources change (cli-core's stale-detection rebuilds before any command
that touches the API).

## Source of truth: `cli/manifest.json`

The CLI's command surface (groups, commands, positionals, flags,
RPC bindings, governance metadata) is declared in
[`cli/manifest.json`](../../cli/manifest.json) and validated against
[`.vrooli/schemas/cli-manifest.schema.json`](../../../../.vrooli/schemas/cli-manifest.schema.json)
(schema id `cli-manifest/v1`). The manifest is loaded at startup by
`cliapp.LoadFromManifest`, which:

- builds each domain's `SubcommandGroup` from its manifest group
- wires each command's `binding.method` (e.g. `ModelsService.ListModels`)
  to a handler registered in the domain's `register.go` bindings map
- fails loudly on missing handlers, dead handlers, or unknown groups

Per-domain tests use `cliapp.RequireProtoServiceCoverage` to assert
that every RPC on the bound proto service either has a manifest command
binding or appears in the manifest's `omitted[]` list with a reason —
adding a new RPC without exposing it as a CLI command (or explicitly
omitting it) fails the test.

The manifest's `governance` block (`effect`, `run_eligible`,
`permissions`, `requires_confirmation`) is consumed by prompt-manager
to derive action certainty automatically; scenarios that adopt the
manifest don't need hand-classified action-safety lists.

`binding.kind` is currently `connect-rpc` only, modelling unary RPCs.
Commands that can't be expressed that way — the canonical example is
`jobs watch`, which consumes a server-streaming RPC — are appended to the
loaded group outside the manifest path in the domain's `register.go` and
documented in the manifest's `omitted[]` array.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start image-tools` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `image-tools status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
image-tools status
image-tools status --json
```

### `image-tools configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
image-tools configure api_base http://localhost:15001/api/v1
image-tools configure token <token>
```

Read values back without an argument:

```bash
image-tools configure api_base
```

## Scenario commands — `jobs` (durable async lifecycle)

Inspect and control server-owned durable jobs. `jobs wait` is the
canonical block-once verb — it blocks server-side until the job is
terminal and returns it. Never poll `jobs get` in a loop.

### `image-tools jobs get <id>`

Get the current record for one job. Calls `JobsService/GetJob`. Uses the
**data-retrieval contract**.

```bash
image-tools jobs get <job-id>
```

### `image-tools jobs wait <id>`

Block once until the job is terminal, then print it. Calls
`JobsService/WaitJob`. A client disconnect does not affect the job.

```bash
image-tools jobs wait <job-id>
```

### `image-tools jobs list [--limit <n>]`

List recent jobs, newest first. Calls `JobsService/ListJobs`.

```bash
image-tools jobs list --limit 20
```

### `image-tools jobs cancel <id>`

Cancel a running or queued job. Calls `JobsService/CancelJob`. Uses the
**mutation contract**.

```bash
image-tools jobs cancel <job-id>
```

### `image-tools jobs watch <id>`

Stream a job's progress until it reaches a terminal state. Consumes the
server-streaming `JobsService/WatchJob` RPC, so it is appended outside
the manifest path in `register.go` and recorded in the manifest's
`omitted[]` array. `--json` emits one `ProgressEvent` per line.

```bash
image-tools jobs watch <job-id>
```

## Scenario commands — `ai` and `analyze` (model lifecycle execution)

AI commands submit durable jobs through the REST multipart edge and can block
once with `--wait`. Analysis commands run synchronous image-to-data operations
and record terminal jobs for observability.

No-download model-lifecycle paths are intentionally exposed through the same
CLI surface as provisioned model paths:

```bash
image-tools analyze probe in.png --json
image-tools analyze quality in.png --json
image-tools analyze duplicate in.png --json
image-tools ai naturalize in.png --wait --out naturalized.png
image-tools ai normal-map in.png --wait --out normal.png
```

`naturalize` uses the built-in provider and `normal-map` uses the computed
provider, so neither command requires model downloads or host AI packages.
Model-backed commands such as `generate`, `upscale`, `bg-removal`, `depth`, and
`colorize` keep the same submit/wait shape but correctly refuse with actionable
install/provisioning guidance until their weights and backend runtime are ready.
For an attended clean-host install proof, run
`IMAGE_TOOLS_ALLOW_MODEL_DOWNLOADS=1 make model-install-e2e` from the scenario
directory; it loops enabled weight-backed models through `models install --wait`
and resumes safely on already-installed models.

### Conditioning flags (`--lora` / `--controlnet` / `--ip-adapter`)

The generative `ai` submit verbs (`generate`, `img2img`) accept a
**repeatable** conditioning flag per adapter kind (the same flags feed the
read-only `--explain` dry run on those verbs), building a typed, ordered
adapter stack (see
[`adapter-registry.md`](adapter-registry.md)). The first colon-delimited field
is always the adapter id; a trailing numeric field is the scale; a trailing
non-numeric field is the conditioning/reference image blob key:

| Flag | Spec form |
|---|---|
| `--lora` | `id[:scale]` |
| `--controlnet` | `id[:scale[:conditioning_image_key]]` |
| `--ip-adapter` | `id[:scale]:reference_image_key` |

The resolver validates each adapter against the chosen model's architecture
(compatible + enabled + installed + Ready), orders the stack
LoRA → ControlNet → IP-Adapter, clamps each scale to the adapter's range, and
elevates the op's consent weight to `max(op, adapters...)`. It fails closed on
an incompatible / disabled / not-installed / not-Ready adapter or a missing
required reference image. Add `--explain` to print which model/technique would
run and the resolved adapter stack without submitting.

```bash
image-tools ai generate --model sd-1.5 --prompt "a serene mountain lake" \
  --lora lcm-lora-sdv1-5:1.0 --wait --out lake.png
image-tools ai generate --model sd-1.5 --prompt "a stone castle" \
  --controlnet controlnet-canny-sd15:1.0:<control-image-key> --explain
```

## Scenario commands — `models` (registry read + enable/disable)

Inspect the declarative model registry and toggle which models are
enabled. The seed catalog is read-only; enable/disable writes a SQLite
overlay.

### `image-tools models list [--operation <op>]`

List model registry entries, optionally filtered to one operation. Calls
`ModelsService/ListModels`.

```bash
image-tools models list
image-tools models list --operation upscale
```

### `image-tools models get <id>`

Get one model entry by id. Calls `ModelsService/GetModel`.

```bash
image-tools models get sd-1.5
```

### `image-tools models operations`

List the registry operation vocabulary. Calls
`ModelsService/ListOperations`.

```bash
image-tools models operations
```

### `image-tools models select <operation> [--override <id>]`

Preview which enabled model would run for an operation on this host
(honoring the per-op default and any override) without executing. Calls
`ModelsService/SelectModel` and surfaces the hardware-fit reason +
warnings.

```bash
image-tools models select upscale
image-tools models select upscale --override real-esrgan
```

### `image-tools models enable <id> [--disable]`

Enable (default) or disable (`--disable`) a model. Calls
`ModelsService/SetModelEnabled`, persisting the overlay. Uses the
**mutation contract**.

```bash
image-tools models enable real-esrgan
image-tools models enable real-esrgan --disable
```

### `image-tools models blocklist`

List license-encumbered models excluded from the catalog. Calls
`ModelsService/ListBlocklist`.

```bash
image-tools models blocklist
```

### `image-tools models doctor`

Diagnose catalog installability and policy integrity. Calls
`ModelsService/DoctorCatalog` and exits non-zero when enabled seed models lack
direct installable assets, an operation has no installable enabled model, or
commercial/checksum policy is incoherent.

```bash
image-tools models doctor
image-tools models doctor --json
```

### `image-tools models inspect <source>`

Dry-run preview of a bring-your-own import source without installing. Calls
`ModelsService/InspectModelSource`. `<source>` is a Hugging Face repo id (gold
path), a direct URL, or a local path. Reports the detected layout (single-file
checkpoint vs diffusers repo), the **inferred architecture** with confidence +
evidence, license, NSFW flag, size, the ops the import would offer, and a
proposed id. Installs nothing. See [`model-registry.md`](model-registry.md#self-serve-import-bring-your-own-model).

```bash
image-tools models inspect runwayml/stable-diffusion-v1-5
image-tools models inspect ./my-finetune.safetensors --json
```

### `image-tools models import <source> [--architecture <arch>] [--id <id>] [--name <name>] [--operation <op>] [--attest-commercial-rights]`

Register and install an imported model add-only at local tier. Calls
`ModelsService/ImportModel`. `--architecture` is required when inference
returned no confidence (never guessed silently). The entry carries a
`user-imported` provenance label; public/BYOK serving of an unverified-license
import is blocked until `--attest-commercial-rights` is passed. Uses the
**mutation contract**.

```bash
image-tools models import runwayml/stable-diffusion-v1-5 --architecture sd15
image-tools models import ./my-finetune.safetensors --architecture sdxl --attest-commercial-rights
```

## Scenario commands — `adapters` (conditioning-adapter catalog)

Inspect and manage the conditioning-adapter catalog (LoRA / ControlNet /
IP-Adapter) — the sibling of `models`. The seed catalog is read-only;
enable/disable and install write a SQLite overlay. Every adapter ships
`Ready=false` until an attended GPU run proves it (see
[`adapter-registry.md`](adapter-registry.md) and
[`../internal/TESTING.md`](../internal/TESTING.md)).

### `image-tools adapters list [--kind <k>] [--architecture <arch>]`

List adapter catalog entries, optionally filtered by kind or architecture.
Calls `AdaptersService/ListAdapters`. Each row shows kind, architecture,
weight, and effective enabled / Ready / install state.

```bash
image-tools adapters list
image-tools adapters list --kind controlnet --architecture sd15
```

### `image-tools adapters get <id>`

Get one adapter in detail (scale range, license, pending reason, install
record). Calls `AdaptersService/GetAdapter`.

```bash
image-tools adapters get lcm-lora-sdv1-5
```

### `image-tools adapters compatible [--model <id>] [--architecture <arch>]`

List adapters compatible with a base model (by architecture). Calls
`AdaptersService/ListCompatibleAdapters`.

```bash
image-tools adapters compatible --model sd-1.5
image-tools adapters compatible --architecture sdxl
```

### `image-tools adapters enable <id> [--disable]`

Enable (default) or disable (`--disable`) an adapter, persisting the overlay.
Calls `AdaptersService/SetAdapterEnabled`. Uses the **mutation contract**.

```bash
image-tools adapters enable controlnet-canny-sd15
image-tools adapters enable controlnet-canny-sd15 --disable
```

### `image-tools adapters install <id> [--wait]`

Submit a governed download job for the adapter weights. Calls
`AdaptersService/InstallAdapter`; `--wait` blocks once until the download
finishes. Re-running an installed adapter returns `already_installed`.

```bash
image-tools adapters install lcm-lora-sdv1-5 --wait
```

### `image-tools adapters remove <id>`

Remove installed adapter weights. Calls `AdaptersService/RemoveAdapter`. Uses
the **mutation contract**.

```bash
image-tools adapters remove lcm-lora-sdv1-5
```

### `image-tools adapters inspect <source>`

Dry-run preview of an adapter import source (HF repo id / URL / local path):
inferred kind + architecture (with confidence + evidence), license, size, and a
proposed id. Calls `AdaptersService/InspectAdapterSource`. Installs nothing.

```bash
image-tools adapters inspect h94/IP-Adapter
```

### `image-tools adapters import <source> [--kind <k>] [--architecture <arch>] [--preprocessor <p>] [--id <id>] [--name <name>] [--attest-commercial-rights]`

Register and install a custom adapter add-only at local tier. Calls
`AdaptersService/ImportAdapter`. Confirm `--kind` and `--architecture` when
inspect could not infer them; `--preprocessor` applies to a ControlNet. Carries
a `user-imported` provenance label with the same `--attest-commercial-rights`
gate as model import.

```bash
image-tools adapters import h94/IP-Adapter --kind ip-adapter --architecture sd15
```

### `image-tools adapters doctor`

Catalog integrity check — enabled adapters must declare a concrete fetch
strategy (assets / repo / local_path). Calls `AdaptersService/DoctorCatalog`
and exits non-zero on findings. Uses the **operational contract**.

```bash
image-tools adapters doctor
image-tools adapters doctor --json
```

## Scenario commands — `backends` (AI backend software readiness)

### `image-tools backends doctor`

Diagnose host software availability for registered inference backends and flag
enabled catalog backend families that do not yet have a runtime provider. Calls
`ModelsService/DoctorBackends` and exits non-zero when a local backend is
missing or a catalog-declared backend cannot be probed/executed yet. Hardware
fit remains reported by `models select`.

```bash
image-tools backends doctor
image-tools backends doctor --json
```

## Output contracts

Every scenario command should render through one of three human
contracts. Proto-backed commands should use `cliapp.RenderProtoList`
or `cliapp.RenderProtoMutation`: human consumers see the report, while
`--json` consumers receive the proto JSON response shape.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status`, `health`, `audit`, `validate`, `doctor` | Status → Triage → Next Steps |
| **Data Retrieval** | `list`, `get`, `view`, `search` | Summary → Results → Retrieval Hints |
| **Mutation** | `create`, `update`, `delete`, `start`, `stop` | Result → What Changed → Next Command |

For commands that aggregate multiple API calls or produce a
non-proto report, use the `RunContext` render helpers directly
(`ctx.RenderList`, `ctx.RenderMutation`, or the operational report
helpers).

## Adding a new command

For a new domain, mirror the `jobs` / `models` command groups: a
`Register(core, manifest)` returning a `SubcommandGroup` built via
`cliapp.LoadFromManifest`, plus one handler per RPC in `handlers.go`.

For a command inside an existing domain:

1. If the command needs a new API endpoint (RPC), add it first per
   [`api-endpoints.md`](api-endpoints.md#adding-a-new-endpoint). The
   manifest's coverage test will fail otherwise on the next CLI build.
2. Add a command entry to the matching group in
   [`cli/manifest.json`](../../cli/manifest.json): `name`, optional
   `description`, `positionals` / `flags`, the `binding` (service +
   method), and the `governance` block (`effect`, `run_eligible`,
   `permissions`, optional `requires_confirmation`). The schema in
   `.vrooli/schemas/cli-manifest.schema.json` is authoritative.
3. Implement the handler in `cli/domains/<domain>/handlers.go` (or a
   focused sibling file) with signature
   `func(ctx cliapp.RunContext) error`. Read values with
   `ctx.Flag(...)`, `ctx.BoolFlag(...)`, `ctx.Positional(...)`, and
   `ctx.JSON()`.
4. Add the handler to the bindings map in
   `cli/domains/<domain>/register.go` keyed by `"<Service>.<Method>"`
   so `cliapp.LoadFromManifest` can wire it. Missing handler or
   dead handler both fail at startup.
5. Handler implementation should:
   - Construct generated Connect clients with
     `cliapp.NewConnectHTTPClient(core)` for proto-typed operations.
   - Use `cliapp.UploadFile` only for documented multipart REST
     exceptions (append those outside the manifest path in
     `register.go` and document them in the manifest's `omitted[]`).
   - Render proto-backed responses with `cliapp.RenderProtoList` or
     `cliapp.RenderProtoMutation`.
6. Add endpoint metadata in the API handler module and add a matching
   row to `api/cmd/gen-endpoints/cli_commands_seed.json`. Then run
   `make endpoints`; do not edit [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
   by hand.
7. Add a row to this document.
8. Add a handler test in
   `cli/domains/<domain>/handlers_test.go` using `clitest.NewTestApp`
   + `clitest.NewAPIServer` + `clitest.CaptureStdout` (see
   [`../internal/TESTING.md`](../internal/TESTING.md)). Driving the
   handler via `cliapp.NewTestRunContextFromArgs` against the manifest's
   schema gives the closest parity with the dispatched path.

## Command structure principles

- **Subcommand groups** (`jobs list`, `models get`) over flat
  verbs (`list-jobs`, `get-model`). Discoverability via `--help`
  is the goal.
- **Positional for required, flags for optional.** `models get <id>`
  not `models get --id <id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start image-tools`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
