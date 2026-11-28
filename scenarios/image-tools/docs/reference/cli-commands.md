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

### `image-tools jobs rate <id> --score <1-5> [--note <text>]`

Rate a succeeded job's output from 1 (poor) to 5 (excellent) with an optional note
(`JobsService/RateJob`). One rating per job; rating again replaces it. Ratings feed
the model verdicts. Uses the **mutation contract**. `jobs unrate <id>` removes it
(`ClearJobRating`) and `jobs ratings [--model <id>] [--operation <op>] [--limit <n>]`
lists them, newest first (`ListJobRatings`).

```bash
image-tools jobs rate 7f3c --score 4 --note "kept the face, hair colour drifted"
image-tools jobs ratings --model flux-2-klein-4b
```

### `image-tools jobs watch <id>`

Stream a job's progress until it reaches a terminal state. Consumes the
server-streaming `JobsService/WatchJob` RPC, so it is appended outside
the manifest path in `register.go` and recorded in the manifest's
`omitted[]` array. `--json` emits one `ProgressEvent` per line.

```bash
image-tools jobs watch <job-id>
```

### `image-tools jobs download <id> [--index <n>] --out <path>`

Fetch one result of a finished job by job id through the managed blob
endpoint (`GET /api/v1/blobs/{key}`), instead of copying files out of the blob
directory. `--index` selects a variation when the job produced several
(`result_refs`); it defaults to `0`, and `result_ref` is the first entry.

```bash
image-tools jobs download <job-id> --out result.png
image-tools jobs download <job-id> --index 2 --out variation-3.png
```

## Scenario commands — `ai` and `analyze` (model lifecycle execution)

AI commands submit durable jobs through the REST multipart edge and can block
once with `--wait`. Analysis commands run synchronous image-to-data operations
and record terminal jobs for observability.

The AI submit command table is **generated** from
`api/internal/capability/manifests.seed.json` (`make capability-gen` →
`cli/domains/ai/commands.generated.go`). Every AI-catalog operation has a verb,
including the restoration ops `deblur`, `face-restore`, and `old-photo-restore`;
those three refuse with provisioning guidance until their backends are installed.

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
`colorize` keep the same submit/wait shape. When the selected model is an
enabled, rights-permitted catalog entry with a governed (non-manual) source and
its weights are missing, the job acquires them before it runs: the submit
response warns, the ETA includes the download, and `jobs get` shows
`acquiring weights for <id>: …` while the job is admitting. Pass
`--no-auto-acquire` to refuse instead. An installed model stuck at
`smoke_pending` (or `build_not_ready` / `needs_backend`) is revalidated by the
job the same way, without re-downloading valid weights. If other jobs' disk reservations block
the download, the job returns to the queue and retries. Manual, custom and
imported models, models whose artifacts cannot pass the load smoke, and models
whose backend is not provisioned still refuse at submit with actionable
install/provisioning guidance.

Two identity/appearance verbs follow the same shape:

```bash
# Masked clothing/hair/makeup restyle. Served by the enabled SD1.5-inpainting
# default, so it behaves like inpaint once the diffusers venv is provisioned.
image-tools ai garment-replace photo.jpg --mask outfit.png --prompt "leather jacket" --wait --out restyled.png

# Face swap: the input is the picture to edit, --reference is the photo that
# supplies the face. Runs locally (no consent needed). Several faces: the largest
# is replaced by default; --target-face N picks another, --all-faces swaps every
# face, --source-face N picks a face in the reference photo.
image-tools ai face-swap target.jpg --reference face.jpg --wait --out swapped.png
image-tools ai face-swap group.jpg --reference face.jpg --target-face 1 --color-match 0.7

# Where it would run, and whether it needs --consent (a cloud route does):
image-tools ai face-swap target.jpg --reference face.jpg --byok --explain
image-tools ai face-swap target.jpg --reference face.jpg --byok --consent --max-cost 0.25 --wait
```

Operation-specific parameters travel as JSON through `--op-params` and are
validated against the operation's capability schema before submission:

```bash
image-tools ai object-removal photo.jpg --mask object.png \
  --op-params '{"grow_px":12,"halo_threshold":40}'
```

An operation with no declared schema rejects `--op-params` rather than ignoring it.

Neither verb is policy-gated locally. A deployment that turns the consent control
on (`IMAGE_TOOLS_REQUIRE_CONSENT=1`) asks for `--consent` on these high-weight ops,
and the provenance/content-scan controls may likewise be enabled independently.
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

### Cloud routing: wait budget and spend caps

Image Tools is local first. For `ai edit`, `ai img2img` and `ai generate` the
cloud (AI Gateway / OpenRouter) is used in three cases, always inside a spend
cap:

1. The host has no usable GPU, so the best local answer is a weak CPU model: the
   job goes to the cloud instead.
2. The job waited for local admission (queue, host capacity, or a model being
   prepared) longer than its wait budget and has not started running: its local
   admission is cancelled and it is re-routed once. A job that is already
   running is never moved, so it never runs or bills twice.
3. You asked for it (`--byok`, `--role`, `--model openrouter-image`).

| Flag | Meaning |
|---|---|
| `--max-wait <dur>` | Longest local wait before the re-route, for example `90s` or `10m`. `0` waits locally without limit. Default: the deployment's `IMAGE_TOOLS_CLOUD_WAIT_BUDGET` (10m). |
| `--max-cost <usd>` | Per-job cloud spend cap, for example `0.25`. `0` allows no cloud spend for this job. Default: `IMAGE_TOOLS_CLOUD_MAX_JOB_USD` ($0.25). |
| `--fallback-policy local_only` | Never use the cloud for this job. `--require-gpu`, `--force-cpu`, an explicit local `--model` and conditioning adapters also keep it local. |
| `--edit-style identity\|stylized` | Steers the cloud role for edits (below). Default: detected from the prompt. |
| `--no-preserve-identity` | Send an `edit` instruction exactly as written (below). |

The rolling daily cap ($3, `IMAGE_TOOLS_CLOUD_MAX_DAILY_USD`) belongs to the
deployment and cannot be set per request. Both caps are read from recorded
`cost_usd` in the job history; a refusal at submit is HTTP 402 and queues
nothing, and a cap that has filled by the time a re-route or the cloud call
would happen leaves the job waiting locally.

**Which role.** A role you pass wins. Generation uses `image.generate.quality`
under `--quality-policy quality`, otherwise `image.generate.default`. Edits use
`image.edit.identity` (Gemini, keeps the subject recognisable) unless the prompt
restyles the image (anime, manga, cartoon, illustration, cel-shaded, watercolor,
painting, sketch, line art, pixel art, vector, 3D render, "in the style of"),
which uses `image.edit.default` (Seedream). `--edit-style` overrides the
detection; with no signal the identity role is used.

**When the cloud refuses.** A job the policy re-routed to the cloud goes back to
exactly its original local route, instead of failing, if the cloud refuses it
before any billing (out of credits, bad credentials, provider unreachable, rate
limited) or the caps fill before the call. Its meta then reads
`route_choice=local`, `route_reason=cloud_unavailable_fell_back_local`,
`route_cloud_refusal` (the provider's error) and `route_cloud_refused_at`.
The refusal also pauses automatic cloud routing for 30 minutes
(`IMAGE_TOOLS_CLOUD_COOLDOWN`), so other waiting jobs stay local rather than
meeting the same refusal. A job you sent to the cloud yourself (`--byok`, `--role`)
is not held back and gets the honest error. A failure that may have been billed
is never retried locally.

**Where a job ran.** `image-tools jobs get <id>` shows these in the result meta:
`route_choice` (`local` or `cloud`), `route_reason` (`no_usable_gpu`,
`no_local_model`, `wait_budget_exceeded`, `caller_requested_cloud`),
`route_rerouted`, `route_waited_seconds`, `route_local_model` and
`route_local_tier` (what it would have run on), `route_role`, `route_max_cost_usd`,
`route_estimated_cost_usd`, `route_daily_cap_usd`, and the backend's `cost_usd`.
A local job with the fallback available reports `route_fallback=armed`. A failed
job keeps the `route_*` keys.

**Identity preservation.** `ai edit` appends one sentence asking the model to keep
the subject's exact pose, face and markings (and, unless the instruction restyles
the image, its colors), unless your prompt already says so. The prompt the
backend received is always in the meta as `effective_prompt` (with
`prompt_original` and `identity_preservation` = `applied`, `already_present` or
`off`). `--no-preserve-identity` sends the instruction verbatim.

```bash
# Re-route to the cloud if local capacity is not free within 90 seconds, spending at most $0.20
image-tools ai edit in.png --prompt "put a small knitted hat on it" --max-wait 90s --max-cost 0.20 --wait --out edited.png
# Stay local no matter what
image-tools ai edit in.png --prompt "put a small knitted hat on it" --fallback-policy local_only --wait --out edited.png
```

### OpenRouter role selection (`--role`)

The `ai generate`, `img2img` and `edit` verbs accept `--role <role>` to route the
request through the gateway with a specific role from
`resources/openrouter/model-policy.json`. The role selects the upstream model and
its policy: `image.generate.default` is an illustration model (Seedream),
`image.generate.quality` a premium one, `image.edit.default` a reference-capable
edit model, and `image.vector.default` / `image.generate.logo` emit flat SVG.

A named role expresses gateway intent, so when the cloud tier is permitted
(`--byok` or `--fallback-policy cloud_allowed`) the gateway model ranks first
under every quality policy. Without that permission the role is ignored and a
warning says so. `--explain` forwards `--role`, `--quality-policy` and
`--fallback-policy`, so the dry run matches the real submit.

Vector roles return SVG. When `--out` names another format, the CLI writes the
file with the correct extension (for example `mark.png` becomes `mark.svg`) and
reports the path it wrote.

```bash
image-tools ai generate --role image.generate.default --byok --prompt "a finished app icon: a constellation lyre on a dark navy tile" --wait --out concept.png
image-tools ai generate --role image.vector.default --byok --prompt "eagle line art" --explain
image-tools ai edit in.png --role image.edit.identity --byok --instruction "remove the shadow" --wait --out edited.png
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

### `image-tools models enable <id> [--disable] [--reason <why>] [--pin]`

Enable (default) or disable (`--disable`) a model as an **override** over the seed.
Calls `ModelsService/SetModelEnabled`. The override records who set it (from
`IMAGE_TOOLS_ACTOR`, `AI_AGENT` or `USER`), when, and `--reason`. Without `--pin`
a later change of the seed's flag supersedes it at the next boot; `--pin` keeps it
and requires a reason. Uses the **mutation contract**. See
[Enabled-state overrides](model-registry.md#enabled-state-overrides).

```bash
image-tools models enable real-esrgan
image-tools models enable real-esrgan --disable
image-tools models enable restoreformer-plus-plus --disable --pin --reason "judge flagged skin artifacts"
```

### `image-tools models reset <id>`

Drop a model's override so it follows the seed again (`ModelsService/ClearModelOverride`;
`not_found` when it has none). Uses the **mutation contract**.

### `image-tools models blocklist`

List restricted models: license gates (normal local use allowed; only the
listed monetized, hosted or redistributed uses refused) and exclusions (kept
out of the catalog for a safety, technical or hardware-fit reason). Calls
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

### `image-tools models storage [--receipts N]`

Show model-weight storage. Calls `ModelsService/GetModelStorage`: free bytes,
headroom above the shared disk floor, mode, the adaptive idle threshold, every
installed weight-backed model in offload order (bytes, idle time and its basis,
re-acquisition estimate, protections, selection) and recent offload receipts.

```bash
image-tools models storage
image-tools models storage --json
```

### `image-tools models offload [--model <id>]... [--need-bytes N] [--apply]`

Dry-run the adaptive offload (default) or apply it. Calls
`ModelsService/OffloadModels`. `--model` (repeatable or comma-separated)
targets named models, bypassing the idle threshold but never a protection.
`--apply` removes the selection and prints one receipt per attempt. Offloaded
catalog weights are re-acquired automatically by the next job that needs them.

```bash
image-tools models offload                       # what would go now, and why
image-tools models offload --model dncnn --apply # offload one idle model
```

### `image-tools models keep <id> [--reason <text>] [--unkeep]` / `image-tools models unkeep <id>`

Pin a model so offload never removes its weights, or remove the pin. Calls
`ModelsService/SetModelKeep`.

```bash
image-tools models keep sd-1.5 --reason "showcase renders"
image-tools models unkeep sd-1.5
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
`ModelsService/ImportModel`. A diffusers repo finds its adapter from its pipeline
class and offers that family's operations; `--architecture` is needed only when
inference returned no confidence (never guessed silently). The entry is enabled
and graded `untested`, carries a `user-imported` provenance label, and a repo
whose pipeline class has no adapter is refused with the missing piece named.
`--attest-commercial-rights` is optional: without it the model runs locally and
hosted, monetized and redistributed use stays refused. Uses the
**mutation contract**. `models inspect <source>` is the dry run and prints the
adapter found and any `needs:` lines.

```bash
image-tools models import runwayml/stable-diffusion-v1-5
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

## Scenario commands — `ops` deterministic run surface

Every deterministic operation is a subcommand, generated from the `OpParams`
wire contract (`make capability-gen`). One flag exists per scalar parameter, named
by kebab-casing the proto field, so a parameter added to `ops.proto` is reachable
from the CLI without a handler edit, and a flag can never name a field the server
does not read. Flag descriptions live in `cli/domains/ops/params.go`.

```bash
image-tools ops list                          # names, categories, summaries
image-tools ops resize photo.jpg --width 1600 --out hero.webp
image-tools ops screenshot_cleanup shot.png --padding-px 8 --out clean.png
image-tools ops canny photo.jpg --amount 1.4 --out edges.png
```

`--out` chooses the output encoding by extension (`--out result.webp` re-encodes
to WebP for any operation). Repeated parameters take the plural field name, e.g.
`--keep-colors` and `--sizes`.

### `image-tools ops screenshot_cleanup <input>`

Trim the uniform bands a screenshot carries around its subject: window chrome,
the frame a picture was embedded in, and letterbox bars. Parameters:

| Flag | Meaning |
|---|---|
| `--padding-px` | margin kept (or restored) around the detected content |
| `--tolerance` | per-channel match tolerance as a fraction of full scale (default 0.04) |
| `--max-trim-percent` | per-side ceiling (default 0.25, max 0.45); reaching it aborts the trim rather than eating content |

Anything printed **on** the picture (a floating button, a burned-in toolbar) is
not a band; chain the masked `inpaint` operation for that. The seeded
`screenshot-cleanup` Look runs the trim and then upscales the result.

## Scenario commands — `ops` vector bridge (`rasterize`, `vectorize`, `icon_container`)

Three deterministic operations turn a flat mark into every icon target. All three
are pure Go, always available, and reachable from the CLI and the REST edge
(`POST /api/v1/ops/{operation}?output=bytes`).

### `image-tools ops vectorize <input>`

Trace a flat raster mark into a clean SVG: k-means palette (or pinned
`--keep-colors` entries), per-colour masks, boundary tracing, Ramer–Douglas–Peucker
simplification, and one nonzero-winding path per layer. The SVG renders identically
under the pure-Go rasterizer (`fill-rule=evenodd` is avoided because oksvg ignores
it).

```bash
image-tools ops vectorize logo.png --keep-colors '#ffffff' --keep-colors '#22d3ee' \
  --clip-to-largest-rounded-region --inset-px 30 --tolerance-px 0.8 --out logo.svg
```

### `image-tools ops rasterize <input.svg>`

Render an SVG at an exact pixel size. Uses pure-Go oksvg, escalates to headless
Chrome only when the SVG uses a feature oksvg silently drops, and fails by name
when that is unavailable. `--background` flattens the result onto an opaque hex
colour.

```bash
image-tools ops rasterize logo.svg --width 512 --height 512 --out logo-512.png
image-tools ops rasterize logo.svg --width 180 --height 180 --background '#0f172a' --out apple-touch-icon.png
```

### `image-tools ops icon_container <input>`

Pack PNG renders of one SVG or raster into an ICO or ICNS container. Repeatable
`--sizes` sets the exact pixel sizes; ICNS emits both the standard and @2x OSTypes
for 256 and 512.

```bash
image-tools ops icon_container logo.svg --format ico --sizes 16 --sizes 24 --sizes 32 \
  --sizes 48 --sizes 64 --sizes 128 --sizes 256 --out icon.ico
image-tools ops icon_container logo.svg --format icns --sizes 32 --sizes 64 --sizes 128 \
  --sizes 256 --sizes 512 --sizes 1024 --out icon.icns
```

`image-tools ops list` reports the three operations and lists `ico` and `icns` as
encodable formats.

## Scenario commands — `safety` (policy + reviewer feedback)

`safety policy` reports the resolved Responsible-Use policy. The feedback
commands label a reviewer refusal or flag for calibration; they never restore a
deleted output, unblur a sensitive result, or change a stored verdict.

```bash
image-tools safety policy

# The common correction: the reviewer refused something legitimate.
image-tools safety mark-legitimate <job-id> --note "adult subject"
# The alias and the explicit form are equivalent:
image-tools safety feedback <job-id> --label legitimate
image-tools safety feedback <job-id> --label confirmed
image-tools safety feedback <job-id> --label uncertain
# Label one variation of a multi-image job.
image-tools safety feedback <job-id> --ref out/a.png --label legitimate
image-tools safety unmark <job-id>

image-tools safety feedback-list [--label legitimate|confirmed|uncertain] \
    [--reason sexual_unassessable] [--model <id>] [--operation <op>] [--job <id>]
image-tools safety feedback-stats [--since 2026-11-01T00:00:00Z]
image-tools safety corpus [--out corpus.jsonl] [--since <rfc3339>] [--label <label>]
image-tools safety quarantine [--limit N] [--offset N]
```

`safety quarantine` lists refused outputs retained for bounded admin review. The
bytes are never served by the ordinary blob route; use
`GET /api/v1/safety/quarantine/<id>?download=1` (append
`X-Image-Tools-Admin-Token` on a public deployment). The Activity page shows the
same queue with an explicit reveal. Label an entry with
`safety feedback <job> --ref <original_ref> --label legitimate`.

`feedback-stats` reports `false_positives / (false_positives + confirmed)` overall,
by reason code and by reviewer profile. The corpus export is JSON Lines; replay a
candidate calibration with `scripts/safety-corpus-replay.py` (see
[`SECURITY.md`](../internal/SECURITY.md#false-positive-corpus-and-feedback-loop)).



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


### Prepared-layer parallax

`ops composite` and `ops parallax` accept `--params-file` (typed OpParams JSON)
and repeated `--plate` paths matching the declared layer order. Parallax also
requires a stable `--operation-id`. It submits once to the existing job owner,
prints the job identity, waits once and verifies/downloads a lossless PNG-sequence
ZIP to `--out`. Retry the same operation id after an uncertain outcome; different
inputs are rejected. `jobs get`, `jobs wait`, `jobs cancel` and `jobs download`
operate on that same durable job. See [Layered photography](layered-photography.md)
for preparation, camera limits, missing-coverage checks and scope.
