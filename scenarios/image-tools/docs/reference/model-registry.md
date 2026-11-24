# Model Registry Reference

The canonical, machine-readable registry seed is
[`api/internal/models/registry.seed.json`](../../api/internal/models/registry.seed.json)
(schema + load/checksum policy: [`api/internal/models/README.md`](../../api/internal/models/README.md)).
This page is the human catalog: what each operation currently offers, where to download it,
where to watch for updates, and which models are deliberately **blocked**.

Authored 2026-06-16 from license-verified research. Policy choices are recorded in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Design rules baked into the seed

- **Enabled does not mean installed.** Enabled models may participate in host-fit selection whether or not their weights are on disk. A job that selects an enabled, rights-permitted catalog model with a governed (non-manual) source acquires the weights on demand before it runs; see [Weight lifecycle](#weight-lifecycle-auto-acquire-and-adaptive-offload). Manual, custom and imported entries still need `image-tools models install <id>`.
- **Declared OS/architecture support gates admission.** When `hardware.os_arch` is populated, automatic selection and explicit overrides require an exact host match (for example, `linux/amd64`). An empty list means the registry has no OS/architecture constraint for that model. This declaration filters candidates; it is not proof that a backend or model has been exercised on every listed host.
- **Headless + ComfyUI-free.** Every op has ≥1 standalone backend; `requires_comfyui` is always `false`. ComfyUI is an optional plug-in only.
- **Prefer one CPU-capable default per op**, plus one higher-quality (often GPU) tier. If rights or readiness are unresolved, an operation may have no default; the catalog doctor reports it unavailable.
- **License class does not gate normal local use** (operator stance 2026-11-21: best models for normal use; licensing matters for monetization, hosting for others and redistribution). A code license still does not establish pretrained-weight rights, so record the weight grant honestly: label `commercial_use` (`no`/`conditional`) with notes and a `prohibited`/`unknown` rights record, and the shared rights owner will refuse monetized, hosted and redistributed use. Traps are listed in the gates table below.
- **`explicit_use_only` keeps uncensored checkpoints out of automatic selection.** A model carrying `capability_labels.explicit_use_only` (the NSFW SDXL checkpoints) can be installed and enabled, but the selector never picks it for an ordinary request: it runs only when named (`--model`, a pinned default). Its outputs are always reviewed in strict mode and flagged `sensitive`, so the UI shows them blurred.
- **Optional Python sidecar** unlocks PyTorch-only quality tiers (RestoreFormer++, SCUNet, Restormer, DDColor-large, RAM++, Marigold, Florence-2) without ever being required.
- **Sizes/VRAM are approximate**; **checksums are captured on first download** (never hand-written).
- **Enabled weight-backed models must be installable.** `image-tools models doctor`
  checks that enabled seed entries have direct `source.assets[]` records with
  artifact `kind` and positive `min_bytes`. Phase 1 has migrated the enabled
  seed catalog off documentation-page stubs; new enabled weight-backed entries
  must add direct assets before they ship.
- **Weightless catalog entries are explicit.** `builtin`, `computed`, and
  `library-go` entries have no model weights and are treated as always
  installed. `library-cgo` remains a provisioned backend until its host/library
  assets are made explicit by the backend doctor work.

## Weight lifecycle: auto-acquire and adaptive offload

Weights are a cache of governed catalog sources, managed by
`api/internal/modelstorage`. The installer remains the only component that
downloads, verifies or deletes weights; the storage owner decides when.

**Auto-acquire.** When the model an AI job (or an NSFW/OCR analysis job)
needs is not installed, the submit edge accepts it if the model is enabled,
passes `RequireLocalUsePermitted`, has a non-manual seed source (direct
assets or a pinned repo snapshot), and would be runnable once installed: its
artifacts satisfy the load-smoke contract (for example, a `.pth` checkpoint
on `onnxruntime` never can) and its native backend is ready
(`image-tools backends doctor`). The submit response carries a warning and
the ETA includes the download. During admission, before any RAM, CPU or GPU
lease, the job acquires the weights through the installer and shared disk
admission; the job message reads `acquiring weights for <id>: …`. Concurrent
jobs for the same model share one acquisition. If headroom would fall below
the reserve, an offload pass runs first; if headroom above the floor is still
smaller than the install peak, the job fails with the needed and available
bytes and the recovery commands. If free space suffices but shared disk
admission refuses the peak because other jobs hold active reservations, the
job returns to the queue with that reason and retries with capacity backoff.
Headroom is computed from raw free bytes; other owners' in-flight
reservations are enforced by disk admission, not yet shown in
`models storage`.

**Auto-revalidate.** An installed model whose runtime lifecycle is
`smoke_pending`, `build_not_ready` or `needs_backend` is revalidated by the
job the same way (`revalidating <id>: …`) through the `models install` path:
weights that still validate are not downloaded and reserve no disk; only the
load smoke, local build or backend check runs. `smoke_failed`,
`smoke_unsupported` and `env_not_provisioned` still refuse at submit with the
doctor guidance, because re-running cannot change them.

**Disk planning.** An acquisition reserves 1.25× the artifact at the model
root (staging is renamed into place on the same filesystem), ¼× for tool
temporaries and at most 1 GiB of Hugging Face cache (repo snapshots download
into staging via `local_dir`). Catalog-declared sizes are bounded by shared
host admission against the 120 GiB floor; the fixed 25 GB ceiling applies
only to sizes estimated by a metadata probe. Opt out per request with
`--no-auto-acquire` (`AIParams.auto_acquire=false`) or for the service with
`IMAGE_TOOLS_MODEL_AUTO_ACQUIRE=off`.

**Adaptive offload.** Headroom is free bytes on the models filesystem minus
the 120 GiB shared disk floor. At or above *ample* (96 GiB) nothing idles out.
Between the *reserve* (24 GiB) and ample, the idle threshold shrinks
geometrically from 30 days to 1 day (about 5.5 days half-way). Below the
reserve, or below reserve plus a pending acquisition, the highest-priority
eligible models are offloaded until the target is restored. Priority is
`(idle hours + 1) × bytes ÷ re-acquisition seconds`, where re-acquisition is
transfer time at `IMAGE_TOOLS_INSTALL_MB_PER_SECOND` plus a two-minute fixed
cost, so large idle models go first. A pass runs every 30 minutes (first pass
20 minutes after start) and before an acquisition that needs room.

Last use is the later of the latest successful durable job
(`image-tools measures usage`) and the install receipt time. A model with
neither is `use_unknown` and is never offloaded. Protections (any one blocks
offload): `in_use` (shared use lease held), `held_by_active_job` (a queued,
admitting or running job names it), `kept_by_operator`
(`image-tools models keep <id>`), `installed_within_grace` (installed less
than an hour ago), `not_auto_reacquirable` (manual, custom, imported or
local-path entries), `locally_built_export` (BuildProfile exports such as
`big-lama-1024-export`), and `use_unknown`. Every offload attempt writes a
durable receipt with the evidence it used.

```bash
image-tools models storage                 # headroom, threshold, ranked candidates, receipts
image-tools models offload                 # dry-run: what a pass would remove now
image-tools models offload --apply         # remove the selection, writing receipts
image-tools models offload --model dncnn   # targeted dry-run (add --apply to remove)
image-tools models keep sd-1.5 --reason "showcase"   # never offload
image-tools models unkeep sd-1.5
```

## Where models live

Every byte of model weight lives under one managed root (`<data root>`, for
example `~/.vrooli/data/vrooli/image-tools`). Nothing else may hold one.

- **`models/<id>` and `adapters/<id>`** hold installed weights. Each has an
  install receipt (checksum, size, time) in the scenario database.
- **`staging/`** holds an acquisition while it downloads. It is renamed into
  `models/` or `adapters/` only after the bytes verify, on the same filesystem.
- **`cache/`** holds library state that is not a catalog model: the Hugging Face
  hub cache, `torch.hub` checkpoints, insightface, rembg and similar. It has no
  receipts by design, and `image-tools models strays` accounts for its bytes.
- **`receipts/stray-actions.jsonl`** records every adopt and reclaim.

Only the governed install step may reach the network. It writes into `staging/`
(or, for a small hub companion, `cache/huggingface/hub`) and refuses any
destination outside the root (`fetch.ErrOutsideManagedRoot`).

Everything else runs contained. Every process image-tools starts to load a
model (the Python sidecar, iopaint, smoke and build runs, perceptual metrics)
gets each library cache and home variable pinned under `cache/`, runs with the
hubs offline (`HF_HUB_OFFLINE=1`, `TRANSFORMERS_OFFLINE=1`), and cannot open a
connection that leaves the machine: `image_tools_sidecar/_containment.py`
refuses it with a message saying to declare the file in the catalog. The
variables, each library's own default location and the packages that need them
are one table, `api/internal/modelenv/pins.json`. Adding a library that stores
weights is one row there; a contract test fails when a watchlisted package is in
`requirements.lock` without a row, and another fails when any inference command
leaves a known variable unpinned. Libraries that ignore their variable
(insightface) must be given the pinned directory as an argument (`root=` from
`INSIGHTFACE_HOME`).

A model that needs something at load time must declare it in its catalog entry,
so the governed install fetches it and receipts it:

- weights and files: `source.assets[]` or `source.repo`;
- small Hub repos the runtime reads by repo id (diffusers `from_single_file`
  resolves its pipeline config and tokenizer this way): `source.hub_companions[]`,
  each a pinned 40-character commit and a required `allow_patterns`. They are
  fetched into `cache/huggingface/hub` as part of the install, and an install
  that cannot place one does not publish.

A missing file at job time fails with an actionable error instead of
downloading, and job-time auto-acquire (or `models install`) puts it in place.

**Strays.** Bytes that escaped before containment, or that appear in the managed
root with no receipt, are found by `image-tools models strays` (it changes
nothing). It scans each pinned library's default location, the managed root's
`models/` and `adapters/`, and reports the bytes with one action per finding:

| Action | Meaning |
|---|---|
| `adopt` | A catalog entry refers to these files. `models adopt-stray <id>` moves them to the same relative place under `cache/` (or records a receipt in place for files already at `models/<id>`) and logs a receipt. |
| `reclaim` | A duplicate of a copy that is already managed, or untracked managed bytes nothing has written to for 24 hours. `models reclaim-stray <id>` deletes it after re-checking and logs a receipt. |
| `report_only` | Another application's files (Ollama, ComfyUI, another scenario's Hugging Face models) or bytes no catalog entry refers to. Never moved or deleted. |

`image-tools models containment` prints the pins, the offline flags and the disk
floor in force. `models inventory` adds a finding when stray bytes exist.

## Verdicts: what works and what is not recommended

Every model carries a **verdict for every operation it serves**, and a roll-up for
the model. The grades, best first:

| Grade | Meaning |
|---|---|
| `recommended` | Best-known choice for the operation, backed by a judged comparison or the operator's ratings |
| `works` | Produces correct output; not shown to be the best |
| `experimental` | Runs, but quality or reliability is unproven or mixed |
| `untested` | No evidence yet (the grade of every new model; never blank) |
| `not recommended` | Measured and found worse than the alternatives, or the operator rated it poorly; it still runs when chosen |
| `broken` | Does not run or repeatedly fails |

Each verdict has a one-line **reason** and **evidence pointers** (a research file, a
judged comparison, a live run, job history, the operator's ratings). They show in
`models list|get|select|explain`, `ai list` (the default column), the UI models card
and the create-panel model picker. The CLI line starts with the grade, so
`image-tools models list` reads at a glance.

**Derived from data, no second list.** `api/internal/verdicts` computes a verdict
from three inputs, in this order (`verdicts.Derive` is the executable statement of
the rule and `derive_test.go` the table):

1. **Curated claims**, seeded in
   [`api/internal/verdicts/evidence.seed.json`](../../api/internal/verdicts/evidence.seed.json):
   the newest claim for a (model, operation) sets the starting grade and reason; a
   `"*"` operation covers every op the model serves, and a specific claim overrides it.
2. **Job history**: the last 20 terminal jobs per (model, operation) from
   `job_trace`. Success earns `works` (three or more jobs at 80 percent or better)
   or `experimental` (some success); repeated failure with no success earns
   `broken`; a curated `recommended` or `works` is demoted to `experimental` after
   five jobs at under 50 percent success; a curated `broken` with a recent run of
   successes is lifted to `experimental` ("re-qualify"). Only a fault of the model or
   the runtime counts as a failure. Each failed job records a `failure_kind`
   (`api/internal/jobs/failure.go`): `fault` (the default, and what a record from before
   the kind existed reads as), `refused` (consent, spend cap or the output safety
   review), `invalid_input` (a request the model cannot serve as given, such as a
   phrase a SegFormer parser has no label for; the sidecar signals it with
   `_common.InputError` and exit status 65), `interrupted` (a restart) or `capacity` (GPU
   memory another service took mid-run). The last four say nothing about the model and
   are left out of the history. A run that ends in `capacity` (a model with no CPU path
   whose GPU memory was taken) goes back to the queue once, after 16 s, and keeps
   `capacity_retries` and `capacity_retry_reason` in its meta; a second loss fails the
   job as `capacity`, and its error names the processes that held the GPU
   (`GPU memory held when it failed: pid ...`, read from NVML at that moment).
3. **Operator ratings** (below): from three ratings, a mean of 4.0 or more lifts to
   `recommended` and a mean of 2.0 or less drops to `not recommended`; in between,
   ratings lift an untested or experimental model to `works`. A rating never lifts a
   model that job history just demoted, and never overrides `broken`.

The reason always names the evidence that decided the grade.

**Recording a finding (the one step).** After you compare, qualify or break a model,
add a claim to `evidence.seed.json`:

```json
{"model_id": "flux-2-klein-4b", "operation": "edit_instruct", "grade": "recommended",
 "reason": "Judged best identity-preserving edit of the six compared",
 "kind": "judged_comparison", "ref": "research:RESULTS.md#edit-comparison", "date": "2026-11-21"}
```

`kind` is one of `qualification`, `judged_comparison`, `live_run`, `provider_evidence`
or `operator_assessment`; `ref` is a path or `research:` pointer (relative to the
research topic folder) a person can open; `grade` is `recommended`, `works`,
`experimental`, `not_recommended` or `broken`, never `untested`. `go test
./internal/verdicts` rejects a claim for a model that is not in the seed, for an
operation the model does not serve, and a claim with a missing or multi-line reason,
a missing ref, an unknown kind or a bad date. Nothing else needs editing: the verdict
reaches the CLI, API and UI on restart. Claims are for seed models; an imported or
custom model earns its grade from job history and ratings alone.

### Ratings

The operator rates any succeeded job's output 1 to 5 with an optional note:

```bash
image-tools jobs rate <job-id> --score 4 --note "kept the face, hair colour drifted"
image-tools jobs ratings --model flux-2-klein-4b      # the evidence behind a verdict
image-tools jobs unrate <job-id>
```

The UI shows five touch-sized stars under a finished Create or Enhance result. One
tap saves the score and the note field saves separately. Ratings live in
`job_rating` (one per job, with the model and operation copied from `job_trace`), so
they survive the job's retention. A rating stores no image, so it is safe on a
blurred or private result. Only a succeeded job with a trace row can be rated.

## Enabled-state overrides

The seed's `enabled` flag is the baseline. An operator or agent can override it with
`image-tools models enable <id> [--disable] [--reason "..."] [--pin]`, and drop the
override with `image-tools models reset <id>`. An override is durable state that
outlives the binary, so it records **who** set it (the declared actor, not an
authenticated identity), **when**, **why**, and the **seed's flag at that moment**.
`models get`, the API and the UI show all of it, and mark an override that
contradicts the seed.

The rule, applied once at every boot (the only moment the seed can change):

- a **pinned** override always wins; a pin needs a reason;
- an **unpinned** override wins only while the seed still says what it said when the
  override was written. When the seed's flag has changed since, the seed change is
  newer, so the override is marked **superseded** (kept for the record, not applied);
- an override written before provenance existed keeps its effect, is attributed to
  `unknown (before provenance)`, and is **adopted**: the current seed becomes its
  baseline, so the next seed change can supersede it;
- `models doctor` reports an override with no recorded author that contradicts the
  seed as a warning with both ways out (`--pin --reason` or `models reset`), and
  every other contradicting or superseded override as information.

This is the rule that fixes the 2026-11-10 case where an agent's override left
RestoreFormer++ disabled after the seed enabled it: the override had no provenance
and nothing ever compared it with the seed again.

## Why a row is off

Every seed row with `"enabled": false` carries `"disabled": {"kind": ..., "reason": "one line"}`;
the seed refuses a disabled row without one and an enabled row that still has one
(`validateSeedInvariants`). The reason shows as the verdict reason of a disabled model
that has no other evidence (`models get|list|explain`, the Models card and the picker).
A runtime reason reads **broken**, the others **untested** with "disabled: ..."; a model
that has real evidence keeps its grade and carries the reason as one more evidence
entry. The reason is hidden once an operator override enables the model, because it
described the seed's state.

| Kind | Meaning | Re-enable when |
|---|---|---|
| `runtime` | cannot run on this build (no runner, backend, or loadable weight format) | the runner or a pinned loadable weight exists |
| `needs_install` | weights or backend need a manual download, export or install the catalog cannot do | a pinned asset or supported provisioning route exists |
| `hardware` | does not fit this host's GPU or memory | the host changes |
| `superseded` | a recommended model covers the operation and nobody has measured a gain from this one | a comparison shows a gain |
| `candidate` | registered for comparison and not yet judged | it is judged (add a claim in `evidence.seed.json`) |

**Licensing is never a kind.** A model is not switched off for an unverified or
restrictive licence: the operator's rule is the best model for normal local use, and
rights gate monetized, hosted or redistributed use through the rights record and
`commercial_use` notes. An import of an adapter that is registered but not proven also
starts disabled with a `runtime` reason (the family's own pending note).

## Add a model

Pick the row that matches what you have. Each is the shortest honest path; the
verdict starts as `untested` and moves with job history and ratings, so you never
declare quality up front.

| You have | Steps |
|---|---|
| A **Hugging Face diffusers repo** of a known family (a Flux.2 klein variant, Qwen-Image-Edit, an SD or SDXL checkpoint) | `image-tools models import <org/repo> --wait` |
| An **ONNX** (or other single-file) model for an operation that already has a script | `image-tools models add-custom <id> --operation <op> --backend onnxruntime --download-url <url> --wait`, or `--local-path <file>` |
| A model that should ship **in the seed** | One row in `registry.seed.json` plus, once judged, a claim in `evidence.seed.json`. For weights on the Hugging Face Hub, `python3 scripts/pin_hf_assets.py <org/repo> --include <glob>` prints the commit-pinned `source.assets[]` rows (URL, sha256, size) so nobody types a digest; see [Seed rows](#seed-rows) |
| A **Transformers** checkpoint (torch + `transformers`, no diffusers pipeline) for an operation that has a sidecar module | The seed row with `backend: "<name>-transformers"`, then the [Transformers backend](#add-a-transformers-backend) steps |
| A repo whose pipeline class has **no adapter** | [Add a diffusers family](#add-a-diffusers-family), then the first row |

**Diffusers import, what it does for you.** `models inspect <repo>` is the dry run
(it installs nothing). It reads the repo's `model_index.json` pipeline class and:

- finds the **adapter family** from that class (`families.go`), copying its
  operations, pipeline class and minimum diffusers version onto the entry;
- infers the **architecture** (pipeline class, then model-card tags, then lineage);
- fetches only the **component files**, skipping the duplicate root single-file
  checkpoint many repos ship beside them;
- registers the entry **enabled** and selectable as soon as the weights install, with
  `commercial_use: conditional` and the source's licence (or `unverified`);
- prints **`needs:`** lines for anything missing, in plain words, for example
  `no diffusers adapter is registered for pipeline class "FooPipeline": add a
  DiffusersFamily row in api/internal/models/families.go and an Adapter in ...`.
  An import with no adapter is refused with that line rather than registered as a
  model whose first job fails; an adapter that is not yet proven registers the entry
  disabled and says why.

Pass `--architecture` only when inference says none, and `--operation` (repeatable)
only to narrow the family's operations.

**Licensing never blocks local use.** The attestation (`--attest-commercial-rights`)
is optional. Without it the model runs locally and its conditional label keeps
hosted, monetized and redistributed use refused. With it the label becomes
`commercial_use: yes`. The same completion applies to `add-custom`: a missing licence
is stored as `unverified` and nothing is refused on licence grounds.

**ONNX models are drop-in only when the operation's script reads their tensor
layout.** The sidecar op scripts (`depth.py`, `bg_removal.py`, ...) load the first
`.onnx` in the model directory, so a model with the same input and output contract
needs no code. A model with a different contract needs that script extended, and it
is the verdict (broken after repeated failed jobs) that tells you, not a doctor
finding.

<a id="seed-rows"></a>**Seed rows** follow the checklist below for the data a shipped entry needs.

1. Edit [`api/internal/models/registry.seed.json`](../../api/internal/models/registry.seed.json)
   by hand. Do not mass-edit the seed.
2. Choose the operation and tier deliberately:
   - exactly one enabled default should serve each operation;
   - quality tiers may be enabled only when their license, backend, and host
     requirements are truthful;
   - disabled quality tiers still need a clear reason in `io.notes` or
     `capability_labels.known_risks`.
3. For weight-backed models, add direct `source.assets[]` entries (or a pinned
   `source.repo`):
   - use a direct artifact URL, not a model-card or HTML page;
   - set `filename`, `kind`, and positive `min_bytes`;
   - pin every asset's `sha256`. For a Hugging Face repo, `python3
     scripts/pin_hf_assets.py <org/repo> --include <glob>` prints the rows (LFS
     digests come from the Hub listing, small files are hashed once); for a GitHub
     release asset that publishes no digest, stream it once through `sha256sum`
     (`curl -sL <url> | sha256sum`; nothing is saved). Only when neither is possible
     leave `checksum.status` as capture-on-download so install records pin the
     verified bytes;
   - declare everything the runtime reads at load time (see
     [Where models live](#where-models-live)): extra files as assets, small Hub
     config repos as `source.hub_companions[]`. Inference runs offline, so an
     undeclared dependency fails the job instead of downloading.
4. For weightless models, use only the backends the registry treats as explicit
   no-weight providers (`builtin`, `computed`, `library-go`) unless backend
   doctor has a registered runtime provider for the family.
5. Confirm the backend row exists in [`backends.md`](backends.md). If the backend
   needs host software, route provisioning through Scenario Dependency Analyzer;
   never install with raw `pip`, `go get`, `npm`, or an OS package manager.
6. Run the fast gates:

```bash
cd scenarios/image-tools && make coverage-floors
image-tools models doctor --json
image-tools backends doctor --json
```

`models doctor` must be green before the model ships enabled. `backends doctor`
may stay red for missing host provisioning, but it must identify a registered
provider and concrete provisioning path rather than a catalog-only gap.

### Add a Transformers backend

For a torch + `transformers` model that is not a diffusers pipeline (SAM 2, SegFormer,
Florence-2). Every file the model loads is a pinned asset: Transformers resolves its
config, tokenizer and processor from the model directory with `local_files_only`, so a
file that is not declared is a failed job, not a download.

1. **The seed row**: `backend: "<name>-transformers"` and `source.assets[]` from
   `scripts/pin_hf_assets.py` (weights as `safetensors`, every small JSON/txt file as
   `config`). Load the community port that needs no `trust_remote_code`; the sidecar
   never executes code from a model repository.
2. **The sidecar module** (`image_tools_sidecar/<name>.py`) with `--model --image --out
   --device`, and a `smoke(model_dir, deep)` function.
3. **One row in each of three tables**, all data:
   `transformersSmokeKinds` in `api/internal/models/smoke.go` (backend to smoke kind),
   a `--kind` in `smoke.py`, and a provider spec in `api/internal/ai/provider_catalog.go`
   (`hostTool: "uv"`, `gpuProbe: "torch"`, `warmSidecar: true`, one technique per
   operation). The lifecycle gate, the provisioner and the GPU-device binding read the
   smoke table, so there is no fourth list.
4. **The manifest**: add the backend to each served operation's `backends` in
   `manifests.seed.json`, then `make capability-gen backends-doc`.
5. Install it, run one live job on a real image, and add the claim. `models doctor`
   must read `ready` (the install-time smoke loaded the checkpoint).

### Add a gateway-served vision model

For a general vision-language model (Qwen3-VL, Gemma, Gemini, GPT) that should answer
`caption`, `object_detection` and `tagging` next to the Florence-2 specialist. It is **one
row and no code**: image-tools asks the model in words and parses the answer, and the AI
Gateway decides which concrete model a role means, on this machine through Ollama or
hosted.

1. **Choose a gateway role** that the gateway's resource policy already maps to the
   model. `resources/ollama/model-policy.json` maps `vision.default` to a local vision
   model; `resources/openrouter/model-policy.json` maps it to a hosted one. To use a model
   no role covers, add a role there, not here.
2. **Add the row** to `registry.seed.json`, copying `qwen3-vl-4b` (local) or
   `gemini-flash-lite-vision` (hosted):
   - `backend: "gateway-vision"` for a model the gateway runs through Ollama (no consent
     needed, nothing leaves the machine), or `backend: "openrouter"` for a hosted model
     (BYOK: needs `--byok`, a spend cap and, when the image shows people, the consent
     affirmation, because the image is sent).
   - `operations`: any of `caption`, `object_detection`, `tagging`.
   - `runtime.gateway_role`: the role from step 1.
   - for `object_detection`, `runtime.box_order` and `runtime.box_scale`: how the model
     writes a box. Qwen writes `[x1,y1,x2,y2]` (`xyxy`) and Gemini `[y1,x1,y2,x2]`
     (`yxyx`), both on a 0..1000 grid (`box_scale: 1000`; `0` means source pixels). Look at
     one raw answer to tell. A wrong order shows as boxes that are transposed.
   - no `source.assets`: the weights belong to the gateway's Ollama or to the provider, so
     the row reads installed from the start and `models install` has nothing to fetch.
3. **Prove it**: `image-tools ai caption|detect|tag <img> --model <id> --wait` on a real
   photograph (a hosted row needs `--byok`, and `--consent` only when the photo shows people), compare with the Florence-2
   output, and add a claim to `evidence.seed.json` with the numbers. The default does not
   move unless the new model wins on quality, then speed.

The answer's `model` and the job's meta record the concrete model the gateway chose, so a
role remapped in the gateway policy is visible in every result. Image-tools holds no CPU,
RAM or GPU claim for such a job: it waits on the call, and the gateway's capacity broker
places the model. A cold local model on a busy GPU can take minutes to load; the call
allows ten.

### Add a diffusers family

Only when a repo's pipeline class has no adapter. Two files, one test:

1. `api/internal/models/families.go`: a `DiffusersFamily` row with the `Name`,
   `PipelineClass`, the `Operations` the adapter serves (`text_to_image` exactly when
   it generates without an input image), `MinRuntime`, and `Ready: false` with a
   `Pending` reason until an attended run proves it.
2. `api/internal/sidecar/py/image_tools_sidecar/_diffusers.py`: an `Adapter(...)` in
   `_ADAPTERS` (kwarg builder, dtype, offload, `generates`).
3. Run `go test ./internal/models`. `TestDiffusersFamilyAdaptersMirrorPython` and
   `TestDiffusersFamiliesAreCompleteEnoughToImport` name the missing half in plain
   words. Flip `Ready` in both files together after the attended run, and the
   family's imports become selectable. The sidecar is embedded in the Go binary, so
   the change needs a restart.

**Repo repairs the loader makes for you.** Community SDXL repos often arrive with a
component the declared pipeline cannot use. `repo_component_overrides` in
`_diffusers.py` fixes the two seen so far at every `from_pretrained` site, so the
registry row needs nothing extra: a tokenizer saved as `tokenizer.json` alone
(LUSTIFY; transformers 5 layout) is rebuilt as the slow `CLIPTokenizer` diffusers
expects, and an `EDMDPMSolverMultistepScheduler` declared beside an
epsilon-prediction UNet (Big Lust v1.6, which otherwise samples pure noise) becomes
`EulerDiscreteScheduler`. A new component fault is one more function there with a
unit test in `test_diffusers_adapters.py`. A checkpoint whose images come out as
structureless noise is the signature of a scheduler or VAE mismatch, not a bad seed.

## Add a capability

An operation is **declared once**, as a capability manifest row; everything else is
generated or looked up. The steps by how new the capability is
([`CAPABILITY-MODEL.md`](../concepts/CAPABILITY-MODEL.md) has the tier table):

1. **Declare it.** Add a row to
   [`api/internal/capability/manifests.seed.json`](../../api/internal/capability/manifests.seed.json):
   `id`, `family`, `mechanism`, `engine`, `io`, `requires`, `backends`,
   `consent_weight`, the `params_schema` for its knobs and, if it appears in the
   workspace, its `panel`. `go test ./internal/capability` names what is missing
   (an undeclared input or output, a backend no model uses, a consent weight that
   disagrees with the safety policy). `engine: "ai"` makes it a job the AI engine
   runs (give it a `cli` row so `image-tools ai <verb>` exists); `unwired` is only
   for an operation with no runner yet. `io.output.kind` also decides the safety
   review: `image` and `mask` outputs are reviewed as rasters, `text`, `data` and
   `mesh` outputs (a caption, detections, an embedding) are not.
2. **Generate.** `make capability-gen` regenerates the UI catalog, CLI table, typed
   parameter views and wire binding from the manifests and proto; the drift gates
   fail until you run it. No per-op proto field, command or form is hand-written.
3. **Bind a technique** only if the operation needs a pipeline no existing technique
   provides: a row in `api/internal/ai/provider_catalog.go` (technique and provider).
   A new operation of an existing mechanism and family (another ONNX extractor,
   another inpaint-style editor) needs no code.
4. **Give it a model.** A seed row (or `models import`) whose `operations` lists the
   new op. A new op with no enabled model is reported unavailable by `models doctor`.
5. **Record what you find.** Claims in `evidence.seed.json` for the new (model, op)
   pairs once you have compared them; until then the verdict reads `untested`.
6. **Run the gates.** `go test ./internal/capability ./internal/models ./internal/verdicts`,
   `make endpoints` if you added an RPC, and `vrooli scenario test image-tools --phases unit`.

## Self-serve import (bring your own model)

Beyond the seed catalog, operators can import a model they own through a guided
**inspect → confirm → install** flow over CLI + RPC + UI
(`api/internal/models/import.go`, `internal/hfmeta`, `architecture.go`). It is
the model-side twin of the adapter import in
[`adapter-registry.md`](adapter-registry.md) and reuses the same
`internal/fetch` download/checksum spine.

1. **Inspect (dry run).** `image-tools models inspect <source>` fetches
   metadata and previews the proposal without installing anything. `<source>`
   may be:
   - a **Hugging Face repo id** (the gold path, e.g.
     `runwayml/stable-diffusion-v1-5`),
   - a **direct URL** to a single-file checkpoint, or
   - a **local path**.

   The layout is auto-detected — a **single-file checkpoint** routes to
   `source.assets[]` on the `stable-diffusion.cpp` backend (with diffusers as
   an alt for derived ops); a **diffusers repo** routes to a pinned
   `source.repo` snapshot on the diffusers backend (decision D3). The
   architecture is **inferred** (`InferArchitecture`) from pipeline-class / tags
   and returned with a **confidence + evidence** string — an ambiguous source
   returns low/none confidence and is never silently guessed.

2. **Confirm.** `image-tools models import <source>` registers the entry; add
   `--architecture <arch>` only when inference returned no confidence. `--id` / `--name` / `--operation` override the
   defaults; base txt2img architectures (sd15/sdxl/flux) declare
   `text_to_image` + `image_to_image` and **derive** inpaint/outpaint via the
   architecture table, so the import immediately offers its native + proven-derived
   ops.

3. **Install.** The entry registers **add-only at local tier** with a
   `provenance: user-imported` label, then installs via the shared fetch spine
   with a checksum pinned on first download (`--wait` blocks once on the job).

**Commercial-rights gate (decision D4).** A user-imported model's license is
read when declared, else marked `unverified`. Without attestation the entry is
`commercial_use: conditional` and **public/BYOK, hosted, monetized and
redistributed use is refused** until the operator passes
`--attest-commercial-rights`, which records the attestation and flips the entry to
`commercial_use: yes`. Local use never needs it.

```bash
image-tools models inspect runwayml/stable-diffusion-v1-5
image-tools models import runwayml/stable-diffusion-v1-5 --wait
# serving an unverified-license import publicly requires attestation:
image-tools models import ./my-finetune.safetensors --architecture sdxl \
  --attest-commercial-rights --wait
```

## Backend engines (what the Go API shells out to)

| Engine | Used for | Notes |
|---|---|---|
| `stable-diffusion.cpp` | t2i / i2i / inpaint / outpaint | CPU-capable diffusion; supports SD1.x/2.x, SDXL, SD3.x, Flux. Not PixArt/Cascade/Kandinsky. |
| `IOPaint` | object removal (LaMa / MI-GAN) | Batteries-included standalone server. |
| `rembg` + `onnxruntime` | background removal | MIT lib; ONNX is the cross-platform path. |
| `realesrgan-ncnn-vulkan` (+ONNX) | upscale / denoise | Vulkan-first binary; ship ONNX for CPU. |
| `onnxruntime-go` (CGo) | colorize, NSFW, detection, tagging, embeddings, SAM, depth | Shared ONNX backbone. |
| `llama.cpp` (CGo) | captioning (moondream2 / SmolVLM) | GGUF vision-language. |
| `gocv` / OpenCV (CGo) | face detection (YuNet), blur/quality (Laplacian, BRISQUE) | Shared CV backbone. |
| pure Go | duplicate detect (goimagehash), QR/barcode (gozxing), resolution math | Zero CGo. |
| `python-sidecar` (optional) | RestoreFormer++, SCUNet, Restormer, DDColor-large, RAM++, Marigold, Florence-2 | Opt-in; never required. |

## Registry by operation

Legend: **D**=default · **Q**=quality tier · **V**=size/CPU variant · **N**=nice-to-have · 🟢 CPU-capable · 🔴 GPU-tier

### Generation & editing
| Op | Tier | Model | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| text_to_image / image_to_image | 🟢 D | SD 1.5 | OpenRAIL-M ✅ | ~4.0GB safetensors (~1GB q4) | [direct safetensors](https://huggingface.co/stable-diffusion-v1-5/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors) · [sd.cpp](https://github.com/leejet/stable-diffusion.cpp) |
| text_to_image / image_to_image | 🔴 Q | SDXL 1.0 | OpenRAIL++-M ✅ | ~6.9GB (~2GB q4) | [HF](https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0) |
| text_to_image / image_to_image | 🔴 Q+ | FLUX.1 schnell | Apache-2.0 ✅ | ~6.8GB q4 | [HF](https://huggingface.co/black-forest-labs/FLUX.1-schnell) |
| edit_instruct | 🟢 D | InstructPix2Pix | OpenRAIL-M ✅ | ~7.7GB safetensors | [direct safetensors](https://huggingface.co/timbrooks/instruct-pix2pix/resolve/main/instruct-pix2pix-00-22000.safetensors) |
| edit_instruct | 🔴 Q (on-demand install) | Qwen-Image-Edit (2509) | Apache-2.0 ✅ | ~57.7GB repo (Q4 GGUF ~12-16GB) | [HF repo snapshot](https://huggingface.co/Qwen/Qwen-Image-Edit-2509) — `QwenImageEditPlusPipeline`; multi-ref + ControlNet; current selector requires 24GB VRAM (historical attended CPU-offload proof ran on a 16GB GPU) |
| text_to_image / image_to_image / edit_instruct | 🔴 Q (enabled; GPU default via tier ranking) | FLUX.2 [klein] 4B | Apache-2.0 ✅ *(4B only; 9B/dev non-commercial)* | ~16GB installed diffusers tree (transformer 7.75GB + Qwen3 text encoder 8.05GB + VAE) | [HF repo snapshot](https://huggingface.co/black-forest-labs/FLUX.2-klein-4B) pinned `e7b7dc27`, weight digests pinned — `Flux2KleinPipeline` (released diffusers 0.37; lock pins 0.38.0). One unified model: generate, reference transform and instruction edit, 4 steps. Staged placement (text encoder, freed, then denoiser + VAE): measured peak host RSS 7.8–8.4 GiB (`min_ram_gb` 11), and ~2.5 GiB VRAM when streamed beside Ollama (`min_vram_gb` 6); 40–110 s per image on an RTX 4070 Ti SUPER, proven 2026-11-21 (showcase-20261121). CPU-only hosts keep SD 1.5 / InstructPix2Pix |
| text_to_image / image_to_image / edit_instruct | 🟡 nice-to-have (enabled, never a default) | FLUX.2 [klein] 9B | FLUX Non-Commercial v2.1 (normal local use only) | ~32GB tree | [`unsloth/FLUX.2-klein-9B`](https://huggingface.co/unsloth/FLUX.2-klein-9B) pinned `455f1273` (unmodified mirror of the gated official repo). Proven 2026-11-21: 160-242 s/image streamed, peak RSS 11.5-17.1 GiB (`min_ram_gb` 22), CUDA 2.4-2.6 GiB (`min_vram_gb` 4); not better than 4B on identity-preserving edits, see DECISIONS.md |
| edit_instruct | 🟡 nice-to-have (enabled, never a default; no live run yet) | FLUX.1 Kontext [dev] | FLUX.1 [dev] Non-Commercial (normal local use only) | ~33GB tree | [`unsloth/FLUX.1-Kontext-dev`](https://huggingface.co/unsloth/FLUX.1-Kontext-dev) pinned `ed585f92`; `flux-1-kontext` family. Installed and smoked; 28 GiB host-RAM floor was never admittable on the shared dev host |
| edit_instruct | ⚪ Q (disabled, pending adapter) | LongCat-Image-Edit-Turbo | Apache-2.0 ✅ | ~34GB (DF11 quant smaller) | [HF](https://huggingface.co/meituan-longcat/LongCat-Image-Edit-Turbo) — `LongCatImageEditPipeline` needs recent diffusers; enable after an attended run |
| inpaint / outpaint / garment_replace | 🟢 D | SD 1.5 inpainting | OpenRAIL-M ✅ | ~4.2GB ckpt | [direct checkpoint](https://huggingface.co/stable-diffusion-v1-5/stable-diffusion-inpainting/resolve/main/sd-v1-5-inpainting.ckpt) — garment_replace is a masked inpaint driven by a prompt |
| face_swap | ⚪ N (disabled, manual provision) | IP-Adapter FaceID stack (SD 1.5) | Apache-2.0 + InsightFace terms ⚠️ | ~5.6GB stack | [IP-Adapter-FaceID](https://huggingface.co/h94/IP-Adapter-FaceID) — **conditional**: the FaceID variant depends on InsightFace weights whose published terms are non-commercial; verify or replace that component before commercial use. Declared experimental; not blocked by policy |
| inpaint / outpaint | 🔴 Q | SDXL inpainting 0.1 | OpenRAIL++-M ✅ | ~6.9GB | [HF](https://huggingface.co/diffusers/stable-diffusion-xl-1.0-inpainting-0.1) |
| text_to_image / image_to_image / inpaint | 🟡 N, `explicit_use_only` (NSFW-capable photoreal) | Lustify SDXL v9 | Community checkpoint, licence not reviewed (local use unrestricted; hosting/selling needs review) ⚠️ | ~6.9GB fp16 | [`kkkkk26/lustify_v9`](https://huggingface.co/kkkkk26/lustify_v9) pinned revision, weight digests pinned. `inpaint` is the derived SDXL inpaint pipeline over the same weights (no separate inpaint checkpoint to download). |
| text_to_image / image_to_image / inpaint | 🟡 N, `explicit_use_only` (NSFW-capable photoreal) | Big Lust v1.6 (SDXL merge) | Community merge; the conversion repo declares CreativeML OpenRAIL-M ⚠️ | ~6.9GB fp16 | [`John6666/big-lust-v16-sdxl`](https://huggingface.co/John6666/big-lust-v16-sdxl) pinned revision, digests pinned |
| text_to_image / image_to_image / inpaint | 🟡 N, `explicit_use_only` (strongest SFW realism, mildly permissive) | Juggernaut XL v9 | CreativeML OpenRAIL-M; commercial terms on the RunDiffusion page ⚠️ | ~6.9GB fp16 variant | [`RunDiffusion/Juggernaut-XL-v9`](https://huggingface.co/RunDiffusion/Juggernaut-XL-v9) pinned, `runtime.variant: fp16` so only fp16 files download and load |
| text_to_image / image_to_image / inpaint | 🟡 N, `explicit_use_only` (anime/illustration option) | Illustrious XL v0.1 | Illustrious TERM_OF_USE restricts some commercial use ⚠️ | ~6.9GB | [`OnomaAIResearch/Illustrious-xl-early-release-v0`](https://huggingface.co/OnomaAIResearch/Illustrious-xl-early-release-v0) pinned; Danbooru-tag prompts |
| object_removal | 🟢 D | MI-GAN | MIT ✅ | ~27MB | [IOPaint traced weight](https://github.com/Sanster/models/releases/download/migan/migan_traced.pt) · [repo](https://github.com/Picsart-AI-Research/MI-GAN) |
| object_removal | 🟢 Q | Big-LaMa (disabled) | Conditional — pretrained-weight grant unresolved | ~198MB | [upstream weight questions #366](https://github.com/advimman/lama/issues/366) / [#368](https://github.com/advimman/lama/issues/368) · [repo](https://github.com/advimman/lama) |
| background_replace | 🔴 Q | IC-Light V1 (relight) | Apache-2.0 ✅ | ~2GB | [repo](https://github.com/lllyasviel/IC-Light) — *V2 is non-commercial* |

> background_replace **default** is non-AI: BiRefNet/u2net cutout → composite (see below).

### Enhancement & restoration
| Op | Tier | Model | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| upscale | 🟢 D | Real-ESRGAN (disabled; no default) | Conditional — pretrained-weight grant unresolved | ~10MB default pair | [open upstream weight question #1016](https://github.com/xinntao/Real-ESRGAN/issues/1016) · [official model zoo](https://github.com/xinntao/Real-ESRGAN/blob/master/docs/model_zoo.md) |
| upscale (anime) | 🟢 V | Real-CUGAN (disabled; no pinned archive) | Conditional — pretrained-weight grant unverified | ~45MB | [official model docs](https://github.com/bilibili/ailab/blob/main/Real-CUGAN/README_EN.md) · [ncnn releases](https://github.com/nihui/realcugan-ncnn-vulkan/releases) |
| upscale | 🔴 Q | SwinIR (disabled; no default) | Conditional — repository code is Apache-2.0; pretrained-weight grant and training-data rights unverified | ~142MB | [official repo](https://github.com/JingyunLiang/SwinIR) · [DIV2K terms](https://huggingface.co/datasets/eugenesiow/Div2k) |
| denoise | ⚪ off | DnCNN | BSD-3 / MIT ✅ | <5MB | [KAIR direct weight](https://github.com/cszn/KAIR/releases/download/v1.0/dncnn_color_blind.pth) |
| denoise | 🟢 D | SCUNet (ONNX re-export, `Heliosoph/scunet-onnx`; default, tiled onnxruntime sidecar) | Apache-2.0 ✅ | ~70MB | [repo](https://github.com/cszn/SCUNet) |
| deblur | 🟢 D | NAFNet | MIT ✅ | ~92MB ONNX | [OpenCV ONNX export](https://huggingface.co/opencv/deblurring_nafnet/resolve/main/deblurring_nafnet_2025may.onnx) · [repo](https://github.com/megvii-research/NAFNet) |
| deblur | 🔴 Q | Restormer | MIT ✅ | ~135MB | [repo](https://github.com/swz30/Restormer) |
| colorize | 🟢 V | DDColor tiny (fp16 ONNX, ipsilondev conversion) | Apache-2.0 ✅ | ~121MB | [ONNX](https://huggingface.co/ipsilondev/ddcolor-onnx) · [repo](https://github.com/piddnad/DDColor). Oversaturates next to large. |
| colorize | 🟢 D | DDColor large (fp32 ONNX, mindhash export) | Apache-2.0 per cards; third-party export, lineage unverified | ~912MB | [ONNX](https://huggingface.co/mindhash/ddcolor) pinned, digest pinned · [repo](https://github.com/piddnad/DDColor). Default since 2026-11-23: beat tiny on CIEDE2000 against real colour originals. |
| face_restore | 🟢 D | RestoreFormer++ (ONNX + YuNet alignment) | code Apache-2.0; checkpoint rights unresolved (FFHQ) — normal use | ~294MB | [ONNX export](https://github.com/facefusion/facefusion-assets/releases/download/models-3.0.0/restoreformer_plus_plus.onnx) · [repo](https://github.com/wzhouxiff/RestoreFormerPlusPlus) |
| face_restore | 🟢 V | CodeFormer (ONNX + YuNet alignment) | S-Lab 1.0 (non-commercial) — normal use | ~377MB | [ONNX export](https://github.com/facefusion/facefusion-assets/releases/download/models-3.0.0/codeformer.onnx) · [repo](https://github.com/sczhou/CodeFormer) |
| old_photo_restore | — | pipeline: Real-ESRGAN + DDColor + face restore | — | — | not promoted |
| old_photo_restore | 🔴 Q | MS Bringing-Old-Photos | MIT ✅ | ~1–2GB | [repo](https://github.com/microsoft/Bringing-Old-Photos-Back-to-Life) |

### Segmentation & depth
| Op | Tier | Model | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| background_removal | 🟢 D | IS-Net general-use | Apache-2.0 ✅ | ~44MB | [rembg](https://github.com/danielgatis/rembg) |
| background_removal | 🟢 V | U²-Net (u2netp/silueta) | Apache-2.0 ✅ | ~4.7–43MB | [rembg](https://github.com/danielgatis/rembg) |
| background_removal / replace | 🔴 Q | BiRefNet (tiny ONNX) | MIT ✅ | ~224MB | [direct ONNX](https://github.com/ZhengPeng7/BiRefNet/releases/download/v1/BiRefNet-general-bb_swin_v1_tiny-epoch_232.onnx) |
| segment | 🟢 D | MobileSAM | Apache-2.0 ✅ | ~45MB ONNX pair | [encoder](https://huggingface.co/PulpCut/mobilesam-onnx/resolve/main/mobilesam.encoder.onnx) · [decoder](https://huggingface.co/PulpCut/mobilesam-onnx/resolve/main/mobilesam.decoder.onnx) · [repo](https://github.com/ChaoningZhang/MobileSAM) |
| segment | 🔴 Q | SAM 2.1 large (tiny ~128MB) | Apache-2.0 ✅ | ~900MB | [repo](https://github.com/facebookresearch/sam2) |
| segment | 🟢 N | HQ-SAM (sharpest edges) | Apache-2.0 ✅ | ~40MB (Light) | [repo](https://github.com/SysCV/sam-hq) |
| depth_map | 🟢 D | Depth-Anything-V2 Large | CC-BY-NC-4.0 — normal use | ~1.34GB | [direct ONNX](https://github.com/fabio-sim/Depth-Anything-ONNX/releases/download/v2.0.0/depth_anything_v2_vitl.onnx) · [HF](https://huggingface.co/depth-anything/Depth-Anything-V2-Large) |
| depth_map | 🟢 V | Depth-Anything-V2 Small | Apache-2.0 ✅ (monetized use) | ~99MB | [direct ONNX](https://github.com/fabio-sim/Depth-Anything-ONNX/releases/download/v2.0.0/depth_anything_v2_vits.onnx) · [HF](https://huggingface.co/depth-anything/Depth-Anything-V2-Small) |
| depth_map | 🟢 V | Depth-Anything **V1** Large | Apache-2.0 ✅ (monetized large tier) | ~1.3GB | [fabio-sim ONNX v1.0.0](https://github.com/fabio-sim/Depth-Anything-ONNX/releases/download/v1.0.0/depth_anything_vitl14.onnx) digest pinned · [repo](https://github.com/LiheYoung/Depth-Anything). Behind V2 Large (blurrier edges). |
| normal_map | 🟢 D | normals-from-depth (computed) | n/a ✅ | 0 | (math) |
| normal_map | 🔴 Q | Marigold-Normals v1-1 | Apache-2.0 ⚠️ | ~2GB | [HF](https://huggingface.co/prs-eth/marigold-normals-v1-1) — *verify weight card at bundle* |

### Analysis & extraction
| Op | Tier | Model/lib | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| ocr | 🟢 D | Tesseract (gosseract) | Apache/MIT ✅ | ~4MB English data | [eng.traineddata](https://raw.githubusercontent.com/tesseract-ocr/tessdata_fast/main/eng.traineddata) · [gosseract](https://github.com/otiai10/gosseract) |
| ocr | 🟢 D | PaddleOCR PP-OCRv5 (ONNX detector + English and Latin recognisers) | Apache-2.0 ✅ | ~104MB | [`monkt/paddleocr-onnx`](https://huggingface.co/monkt/paddleocr-onnx) pinned, digests pinned · [repo](https://github.com/PaddlePaddle/PaddleOCR). Default; reads display type and scene text. Detection runs at a short scale pyramid. |
| nsfw_classify | 🟢 D | AdamCodd ViT-base NSFW | Apache-2.0 ✅ | ~345MB f32 | [pinned full-precision ONNX](https://huggingface.co/AdamCodd/vit-base-nsfw-detector/resolve/8587de998f441aac03fdd57a85d2e4cb808c7d64/onnx/model.onnx) |

The NSFW sidecar follows the pinned ViT processor profile: 384×384 bilinear resize, pixel rescale by `1/255`, and channel mean/std `0.5`. The ONNX graph exposes dynamic spatial axes while its positional embeddings require the 384×384 patch grid; its sidecar uses that model-specific fallback.
| nsfw_classify | 🟢 Q | Falconsai NSFW (pinned ONNX) | Apache-2.0 ✅ | ~328MB | [`onnx-community/nsfw_image_detection-ONNX`](https://huggingface.co/onnx-community/nsfw_image_detection-ONNX) at `1ceb3c7`, sha256 `a4316a4f…`. The reviewer's precise classifier (0.00–0.13 on the 2026-11-23 diverse SFW set); paired with `adamcodd-vit-nsfw` (high recall, 384 px) because either can miss what the other catches. |
| age_estimate | 🟢 D | FairFace ViT age (ONNX) | Apache-2.0 ✅ (dima806 fine-tune of ViT-base on FairFace) | ~0.35GB | [`onnx-community/fairface_age_image_detection-ONNX`](https://huggingface.co/onnx-community/fairface_age_image_detection-ONNX) pinned. Age **bins** from a face crop; not an identity model. One of the four models of the bundled minor-safety reviewer (with `falconsai-nsfw`, `adamcodd-vit-nsfw` and `yunet`); the 10-19 bin cannot separate 17 from 19. |
| caption | 🟢 V | moondream2 (not installed; no longer default) | Apache-2.0 ✅ | ~3.7GB GGUF pair | [text](https://huggingface.co/ggml-org/moondream2-20250414-GGUF/resolve/main/moondream2-text-model-f16_ct-vicuna.gguf) · [mmproj](https://huggingface.co/ggml-org/moondream2-20250414-GGUF/resolve/main/moondream2-mmproj-f16-20250414.gguf) |
| caption | 🟢 V | SmolVLM-256M | Apache-2.0 ✅ | ~279MB GGUF pair | [model](https://huggingface.co/ggml-org/SmolVLM-256M-Instruct-GGUF/resolve/main/SmolVLM-256M-Instruct-Q8_0.gguf) · [mmproj](https://huggingface.co/ggml-org/SmolVLM-256M-Instruct-GGUF/resolve/main/mmproj-SmolVLM-256M-Instruct-Q8_0.gguf) |
| caption / object_detection | 🟢 D | Florence-2 large (Transformers port) | MIT ✅ | ~1.55GB | [`florence-community/Florence-2-large`](https://huggingface.co/florence-community/Florence-2-large) pinned, digests pinned. Default for both since 2026-11-23 (beat SmolVLM-256M and YOLOX-Tiny on real photos). No tagging task: tagging stays with wd14. |
| object_detection | 🔴 broken | YOLOX-Tiny (verdict `broken`: zero detections, the sidecar does not decode the raw head) | Apache-2.0 ✅ | ~20MB | [direct ONNX](https://github.com/Megvii-BaseDetection/YOLOX/releases/download/0.1.1rc0/yolox_tiny.onnx) |
| object_detection | 🟢 V | NanoDet | Apache-2.0 ✅ | ~1–5MB | [repo](https://github.com/RangiLyu/nanodet) |
| object_detection | 🟢 Q | RT-DETRv2 (official) | Apache-2.0 ✅ | ~76MB | [repo](https://github.com/lyuwenyu/RT-DETR) |
| tagging | 🟢 D | WD14 wd-vit-v3 | Apache-2.0 ✅ | ~379MB | [direct ONNX](https://huggingface.co/SmilingWolf/wd-vit-tagger-v3/resolve/main/model.onnx) |
| tagging | 🟢 Q | RAM++ | Apache-2.0 ✅ | ~300MB | [repo](https://github.com/xinyu1205/recognize-anything) |
| face_detection | 🟢 D | YuNet | MIT ✅ | ~230KB | [direct ONNX](https://media.githubusercontent.com/media/opencv/opencv_zoo/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx) |
| quality_assessment | 🟢 D | Laplacian + resolution (computed) | own ✅ | 0 | in-process pure Go |
| quality_assessment | 🟢 Q | BRISQUE | OpenCV Apache + attribution ✅ | ~25KB | [opencv_contrib](https://github.com/opencv/opencv_contrib) |
| quality_assessment | 🟢 N | LAION aesthetic (+CLIP) | MIT ✅ | ~350MB | [repo](https://github.com/christophschuhmann/improved-aesthetic-predictor) |
| duplicate_detect | 🟢 D | goimagehash (pure Go) | BSD-2 ✅ | ~0 | [repo](https://github.com/corona10/goimagehash) |
| embedding | 🟢 V | nomic-embed-vision-v1.5 (class-token embedding) | Apache-2.0 ✅ | ~97MB int8 | [direct ONNX](https://huggingface.co/nomic-ai/nomic-embed-vision-v1.5/resolve/main/onnx/model_int8.onnx) |
| embedding | 🟢 D | SigLIP2 base (vision tower, fp32 ONNX) | Apache-2.0 ✅ | ~372MB | [`onnx-community/siglip2-base-patch16-224-ONNX`](https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX) pinned, digests pinned · [upstream](https://huggingface.co/google/siglip2-base-patch16-224). Default since 2026-11-23 (0.395 vs 0.191 same-minus-cross cosine margin). Image embeddings only (text tower not installed). |
| qr_barcode_read | 🟢 D | gozxing (pure Go) | MIT ✅ | ~0 | [repo](https://github.com/makiuchi-d/gozxing) |

## Gates — license gates and exclusions

The registry's `blocklist[]` carries two kinds of entry (`gate`):

- **`license`** — the license refuses only the uses in `gated_uses`
  (`monetized_use`, `hosted_inference`, `redistribute_weights`, …). Normal local
  use is never license-gated (operator stance 2026-11-21). A seeded row may carry
  a license-gated model (for example `codeformer`); its rights record must then
  keep the gated uses refused (`commercial_use` not `yes`, rights classification
  not `unrestricted`), which `Load` and `models doctor` enforce. The shared rights
  evaluator (`packages/ai-provider-go/policy`) applies the gate from that record.
- **`exclusion`** — never seeded, for a non-license reason (`exclusion_kind`:
  `safety`, `technical` or `hardware_fit`).

`image-tools models blocklist` lists both with what each restricts. The seed's
`reason` text is the source of truth; the table summarises it.

| Model | Op | License | Gate |
|---|---|---|---|
| **CodeFormer** | face_restore | S-Lab 1.0 (NC) | license — seeded as `codeformer` for normal use |
| **GFPGAN** | face_restore | "Apache" but embeds StyleGAN2 (NVIDIA-NC) + DFDNet (CC-BY-NC-SA); ncnn port GPL | license |
| **bria RMBG-1.4/2.0** | bg_removal | CC-BY-NC-4.0 | license (bundled in the rembg zoo) |
| **FastSAM** | segment | AGPL-3.0 (via Ultralytics) | license — hosted/distributed use only |
| **Ultralytics YOLOv8/v11** | detection | AGPL-3.0 | license — hosted/distributed use; ONNX export does not remove AGPL |
| **YOLO-NAS** | detection | weights Deci-NC | license |
| **InsightFace SCRFD/RetinaFace** | face_detect | weights NC | license |
| **Surya OCR** | ocr | weights OpenRAIL revenue-capped (<$5M) | license — monetized/hosted use |
| **4x-UltraSharp & community ESRGAN** | upscale | CC-BY-NC-SA | license (often mislabeled "mit") |
| **SUPIR** | upscale | custom NC | license (also GPU-heavy, 12–30GB) |
| **APISR** | upscale | GPL + academic weights | license |
| **MAT** | object_removal | CC-BY-NC-4.0 | license |
| **FLUX.1 dev / Fill** | t2i / inpaint | FLUX [dev] NC | **exclusion (hardware_fit)** — 12B + T5-XXL do not fit the 16GB shared GPU |
| **Stable Cascade** | t2i | non-commercial | license (no sd.cpp support; would need a diffusers row) |
| **IC-Light V2** | bg_replace | non-commercial | license |
| **Depth-Anything-V2 Base/Large/Giant** | depth | CC-BY-NC-4.0 | license — Large seeded as the depth_map default |
| **DSINE** | normal_map | custom NC | license |
| **SD 3.5 / Turbo** | t2i | Stability $1M gate | license — monetized use above the cap |
| **Qwen2.5-VL-3B** | caption | Qwen Research NC | license |
| **LLaVA** | caption | Llama/Vicuna NC weights | license |
| **pyiqa MUSIQ / CLIP-IQA** | quality | PolyForm/NTU NC | license |
| **libpHash** | dup_detect | GPL-3.0 | license — distributed binaries only |
| **tuotoo/qrcode** | qr_barcode | GPL-3.0 | license — distributed binaries only |

## Keeping this current

- **Per-model updates:** follow each entry's `update_source` (HF model card or GitHub releases).
- **Re-verify licenses periodically** — several here changed over time (NAFNet/Restormer dropped "non-commercial"; bria/Depth-Anything-V2 added NC tiers; prs-eth migrated Marigold cards). Re-fetch the canonical LICENSE before bundling a new weight.
- **Always check the *weights* license separately from the *code* license**, and remember ONNX/GGUF export never strips AGPL or a non-commercial weight license.
