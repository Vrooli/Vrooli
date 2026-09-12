# Testing — Image Tools

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## API testing

### CRUD reference — `notes` end-to-end

The `notes` domain is the canonical CRUD reference. New scenarios add
their first non-trivial mutation by copying its layering one file at a
time. The pattern from wire to render:

| Layer | File | What it owns |
|---|---|---|
| Wire contract | `packages/proto/schemas/image-tools/v1/notes/notes.proto` | `Note`, `service NotesService`, `ListNotesResponse`, `CreateNoteRequest`, `CreateNoteResponse`, `GetNoteRequest`, `GetNoteResponse` |
| REST metadata contract | `packages/proto/schemas/image-tools/v1/notes/attachments.proto` | `Attachment` and `UploadAttachmentResponse` for the multipart upload exception |
| Connect error mapping | `internal/notes/service_error_mapping.go` | Typed sentinels become Connect codes (`invalid_argument`, `not_found`, `internal`) |
| REST error envelope | `packages/proto/schemas/image-tools/v1/shared/errors.proto` + `internal/httpx/errors.go::WriteError` | Typed body for REST exceptions, with canonical codes (`invalid_request`, `not_found`, `internal`) |
| Domain types | `internal/notes/types.go::{Note, Attachment, CreateInput, ErrInvalidNote, ErrNoteNotFound}` | Domain-pure (no proto imports); typed sentinels translate into Connect errors at the handler edge |
| Repository interface | `internal/notes/repository.go::Repository` | Persistence seam — `Create` / `Get` / `List` |
| Repository impl | `internal/notes/sqlite.go::NewSQLiteRepository` | sqlite-backed `Repository`; production wires it once in `main.go` |
| Schema | `internal/notes/schema.{sql,go}::Schema()` | Domain-owned table DDL embedded via `go:embed`; collected by `internal/modules/registry.go::AllSchemas()` and applied at boot via `apidb.EnsureSchemas` |
| Repository test | `internal/notes/sqlite_test.go` | Real handle via `db.NewSQLite(t)` + `apidb.EnsureSchemas(ctx, d, ...providers...)` over system + notes (the canonical compose pattern) |
| Service | `internal/notes/service.go::Service` (+ `NewService`) | Application layer: validation (`title` required after whitespace trim), default substitution (`defaultListLimit = 100` when caller passes 0). Handler depends on this, not the repository. |
| Service test | `internal/notes/service_test.go` | Substitutes `mocks.FakeRepository` (from co-located `internal/notes/mocks/`); pins the validation, default-substitution, and error-propagation contracts |
| Connect handler test | `handlers/notes/connect_handler_test.go` | Substitutes `mocks.FakeService` and exercises the generated Connect client/handler path |
| Multipart handler test | `handlers/notes/attachments_handler_test.go` | Uses `blobstore.MemoryBlobStore` plus test metadata repositories to exercise file-upload success and error paths |
| Mocks | `internal/notes/mocks/{repository,service}.go::{FakeRepository,FakeService}` | Co-located with the domain (Pass-3 pattern) — `FakeRepository` carries state for service tests; `FakeService` records inputs for handler tests. Both use atomic call counters + per-method error knobs. Deleting `internal/notes/` takes them along. |
| UI client | `ui/src/api/notes.ts` | `notesClient = createClient(NotesService, transport)` plus `uploadAttachment` for multipart metadata |
| UI tests | `ui/src/api/notes.test.ts` + component tests | Mock generated client methods and `uploadAttachment`; REST helper tests stub `global.fetch` |
| CLI client | `cli/domains/notes/{register,handlers,attach_handler}.go` | `Register(core)` returns a `cliapp.SubcommandGroup`; handlers use generated Connect clients or `cliapp.UploadFile` and render via cli-core reports |
| CLI test | `cli/domains/notes/handlers_test.go` | Spins a real `httptest.Server` via `testutil.NewAPIServer`, captures stdout via `testutil.CaptureStdout` |

## Coverage thresholds

| Module | Floor | Where it's enforced |
|---|---|---|
| UI (`ui/`) | 85% lines / branches / functions / statements | `ui/vite.config.ts` `test.coverage`; CI runs `pnpm test:coverage` |
| API (`api/`) | 75% total | `.github/workflows/test.yml` `api` job |
| CLI (`cli/`) | 75% total | `.github/workflows/test.yml` `cli` job |
| Model lifecycle packages | 85% each for `internal/models`, `internal/ai`, `internal/backends` | `.github/workflows/test.yml` `api` job; local `make coverage-model-lifecycle` |

### Go (API + CLI)

Both Go modules gate on a 75% total floor in CI. The threshold is
intentionally lower than the UI's 85% because Go has more
declaration-only surface (interfaces, generated proto types, struct
types) that doesn't carry executable lines — 75% in Go is roughly
equivalent to 85% in TypeScript by lines-per-meaningful-coverage.

`internal/testutil/...` is excluded from the denominator. Those
packages exist to support tests; including them would create the wrong
incentive (writing tests *of* test helpers to inflate the gate). Each
test-utils package has its own self-test (see
`internal/testutil/db/sqlite_test.go`,
`internal/testutil/httpx/server_test.go`, etc.) so the substrate
itself is still verified — it's just not what the production-coverage
number tracks.

Tighten the threshold rather than loosening it when a new file lands
without tests; that's the signal that drives the test-first habit.

The model lifecycle package floors are package-local gates for the
catalog/install/selection/backend substrate. They intentionally sit next
to the total API gate: a broad handler test can keep total coverage over
75% while a substrate package loses its own regression net. Current
floors are the earned Phase 5 bars and should ratchet upward with real
fixture-backed E2E coverage, not with tests that only exercise helpers.

## Headless-completeness

`image-tools`' hard tenet (DECISIONS.md): **every operation runs from the CLI
with no UI, no ComfyUI, and no GPU.** For the deterministic ops (req
IMG-P0-001) this is the *whole* implementation — they are pure-Go/WASM
transforms with zero model downloads, so the acceptance is simply: on a fresh
machine with the scenario started, every op succeeds from the CLI offline.

**How it is verified:**

1. **Unit (no server):** `api/internal/ops/*_test.go` runs every op over the
   codec layer (golden/round-trip assertions); `cli/domains/ops/ops_test.go`
   asserts the CLI exposes one run command per op.
2. **Handler (httptest):** `api/handlers/ops/rest_test.go` drives the multipart
   run edge + blob serve + format override + error paths without a DB.
3. **Live smoke (the real acceptance):** with `vrooli scenario start
   image-tools`, run each op and confirm the output:

   ```bash
   image-tools ops resize    in.png  --out r.png   --width 256
   image-tools ops convert   in.png  --out o.avif  --format avif
   image-tools ops filter    in.png  --out g.webp  --filter grayscale
   image-tools ops compress  in.png  --out c.jpg   --target-bytes 20000
   image-tools ops overlay   in.png  --out w.png   --text "© Vrooli"
   image-tools ops metadata  in.png            # prints EXIF/dimensions JSON
   image-tools jobs list                        # each op recorded as a job
   ```

   No GPU, no ComfyUI, no model pull is involved — the AVIF/WebP/HEIC codecs
   are embedded WASM (pure Go). The BAS smoke flow automates the UI path of the
   same slice.

### AI ops (req IMG-P0-002/003/004) — fakes in CI, attended live gate

The model-backed ops (generation, enhancement, OCR, NSFW) cannot run in CI: the
standalone backend binaries (`sd`, `realesrgan-ncnn-vulkan`, `rembg`, `iopaint`,
`tesseract`) and the CPU-default model weights are not installed, and download
is the Phase-4 model-management concern. The headless tenet still holds — it is
verified in two layers:

1. **No-download live gate:** on a running local scenario, `make headless-ai-e2e`
   exercises the public CLI surfaces that require no model download or host AI
   package: `analyze probe`, `analyze quality`, `analyze duplicate`, `ai
   naturalize --wait`, and `ai normal-map --wait`. It generates its own PNG
   fixture, asserts both durable AI outputs are non-empty, and exits non-zero on
   any refusal/regression.

2. **Attended model-install gate:** on a host where large downloads are
   acceptable, run `IMAGE_TOOLS_ALLOW_MODEL_DOWNLOADS=1 make model-install-e2e`.
   The target first requires `models doctor` to be green, then loops every
   enabled weight-backed model through the public `models install <id> --wait`
   command. Re-running is safe: installed models return `already_installed` and
   do not enqueue duplicate download jobs. Without the environment opt-in, the
   target prints a single skip line and exits cleanly.

3. **Automated (CI):** the full vertical (host probe → hardware-fit select →
   backend select → materialize → execute → persist → auto-scan) runs with
   **fake providers** — `api/internal/ai/{generation,enhancement,providers}_test.go`,
   `api/internal/analysis/*_test.go`, and the handler/CLI tests. Backend
   arg-builders are unit-tested for assembly. The pure-Go `analyze probe` is the
   one model-free op verified live end-to-end.

4. **Attended acceptance gate (the provisioned headless proof):** on a host where the
   CPU default models are installed (Phase-4 `image-tools models install <id>`),
   run each AI op from the CLI with no GPU/ComfyUI and confirm completion:

   ```bash
   image-tools analyze probe  in.png                      # always works (pure-Go)
   image-tools analyze quality in.png                      # pure-Go
   image-tools analyze duplicate in.png                    # pure-Go
   image-tools ai naturalize in.png --out n.png --wait     # built-in, no weights
   image-tools ai normal-map  in.png --out normal.png --wait # computed, no weights
   image-tools ai generate    --prompt "a red bicycle" --out g.png --wait
   image-tools ai upscale     in.png --scale 4 --out up.png --wait
   image-tools ai bg-removal  in.png --out cut.png --wait
   image-tools analyze ocr    scan.png
   image-tools analyze nsfw   in.png
   ```

   Until then, an AI op on a download-free host correctly **refuses** with an
   actionable hint — e.g. `ai generate` → HTTP 409 `run image-tools models
   install sd-1.5` (live-proven). This refusal IS correct headless behavior:
   no silent failure, no crash. Flip the IMG-P0-002/003/004 acceptance to "done"
   once an attended run exercises each op with a model installed (tracked in
   PROBLEMS.md).

### GPU E2E gate (Phase 6)

GPU execution is an attended proof, not part of default CI. Use:

```bash
cd scenarios/image-tools
make gpu-e2e
```

The target composes the shipped operator surfaces instead of bypassing them:
`models select text_to_image --json` verifies live free-VRAM fit,
`backends doctor --json` verifies the GPU-capable `stable-diffusion.cpp`
runtime, `models install <id> --wait` installs the selected model if needed, and
`ai generate --wait` runs a small deterministic text-to-image job. If the host is
busy or the runtime has not been provisioned through SDA yet, the target prints a
single `SKIP gpu-e2e: ...` line and exits cleanly; when ready, it must produce a
non-empty PNG and report the output path.

### Model install gate

Catalog truthfulness is checked by `models doctor`; install truthfulness is
checked by the attended install loop:

```bash
cd scenarios/image-tools
IMAGE_TOOLS_ALLOW_MODEL_DOWNLOADS=1 make model-install-e2e
```

The target intentionally uses the same CLI path an operator uses (`models
install --wait`) so it exercises durable job submission, disk-space preflight,
artifact validation, checksum pinning, and idempotent already-installed
short-circuit behavior. Keep it out of default CI because it may download large
model artifacts and depends on external model hosts.

## Attended GPU end-to-end (conditioning + import)

The conditioning-adapter layer (LoRA / ControlNet / IP-Adapter) and the
bring-your-own model import each carry an **attended GPU** proof. Like
`make gpu-e2e`, these targets are **not part of `make test` / the default
`vrooli scenario test` run** and skip gracefully (exit 0 with a precise reason)
when a prerequisite is missing. The catalog reference is
[`../reference/adapter-registry.md`](../reference/adapter-registry.md).

| Target | Proves |
|---|---|
| `make gpu-e2e-import` | Import an HF base by repo id (`IMAGE_TOOLS_E2E_IMPORT_REPO=<repo>`) → confirm inferred architecture → install → `ai generate` at 512px → non-empty output, and the import offers its derived `image_to_image`. |
| `make gpu-e2e-lora` | SD1.5 + LCM-LoRA at 512px → non-empty output. |
| `make gpu-e2e-controlnet` | SD1.5 + Canny ControlNet at 512px (`IMAGE_TOOLS_E2E_CONTROL_IMAGE_KEY=<blob>`, a pre-made canny map or the output of the canny op). |
| `make gpu-e2e-ipadapter` | SD1.5 + IP-Adapter at 512px (`IMAGE_TOOLS_E2E_REF_IMAGE_KEY=<blob>`, the reference image). |
| `make gpu-e2e-conditioning` | Umbrella: lora + controlnet + ipadapter. |

**SD1.5 @ 512 sequential-offload default.** Each adapter target runs on
`GPU_E2E_BASE_MODEL` (default `stable-diffusion-1-5`) at
`GPU_E2E_W`×`GPU_E2E_H` (default 512) with the sidecar's sequential CPU offload,
which fits ~7–8 GB VRAM. The shared `adapter_e2e_preamble` skips when the base
model is not GPU-viable, the adapter is not in the catalog, or it is not
installed.

**SDXL higher-VRAM gating.** SDXL ControlNets carry a higher-VRAM caveat
(SDXL + ControlNet may exceed 8 GB), so their e2e is gated behind a free-VRAM
check rather than run by default on a modest GPU.

**Conditioned requests route to the diffusers backend.** Conditioning adapters
(LoRA / ControlNet / IP-Adapter) are applied by the diffusers sidecar (which
loads adapter weights through the PEFT backend — `peft` is a governed dependency
in `internal/pydeps`). `stable-diffusion.cpp` cannot apply them. Backend
selection is adapter-aware: when a request carries adapters, the selector skips
the model's native sd.cpp backend and picks the diffusers backend (emitting a
"conditioning adapters require the diffusers backend" notice), so an SD1.5 base
whose default backend is sd.cpp still runs its conditioned generate on diffusers.

**Bootstrap proving procedure (the Ready flip).** An unproven adapter ships
`Ready=false` (with a `pending` reason); the resolver refuses a conditioned
submit until `Ready=true`. A proven adapter ships `Ready=true` with an empty
`pending` (the seed invariant enforces this Ready⇔Pending pairing — no
vaporware). The operator bootstraps the durable regression guard once:

1. **Prove the sidecar path** — run the matching target on the GPU host; it
   confirms model/backend viability and (once Ready) executes the conditioned
   generate. (To prove the path *before* the flip, invoke the diffusers sidecar
   directly: `python -m image_tools_sidecar.text_to_image --model <dir>
   --architecture sd15 --lora|--controlnet|--ip-adapter <spec> …`.)
2. **Flip Ready** in `api/internal/adapters/adapters.seed.json` for that adapter
   (clear its `pending`; pin the repo revision for a ControlNet). A
   conditional-commercial ControlNet stays `enabled:false` in the seed — enable
   it per-host after confirming licence terms with `image-tools adapters enable
   <id>` (the runtime overlay), since the seed gate forbids enabling a
   conditional adapter by default.
3. **Stamp `last_validated_at`** on the matching `type: manual` validation in the
   requirement module.
4. **Re-run the target** — it now executes for real as the durable regression
   guard.

Agents never commit; the Ready flip is an operator (or operator-delegated) edit
to the working tree, reviewed before commit. Steps 2–4 are operator-owned.

**Manual-freshness protocol.** The conditioning + import e2e validations are
`type: manual` entries in
[`../../requirements/35-conditioning-adapters/module.json`](../../requirements/35-conditioning-adapters/module.json)
and
[`../../requirements/36-byo-import/module.json`](../../requirements/36-byo-import/module.json)
(`IMG-CN-E2E-{LORA,CONTROLNET,IPADAPTER}` and `IMG-CN-E2E-IMPORT`). Each carries
two optional fields the test-genie requirements schema allows on a manual
validation:

- `valid_for_days` — the freshness window (90 here).
- `last_validated_at` — when the attended run last proved it (empty = never
  proven; the adapter stays Ready=false).

`make conditioning-freshness` checks these **locally**: it reports each entry as
`FRESH`, `PENDING` (never proven), or `STALE` (last validation older than the
window) and **fails** if any manual evidence is stale — the signal to re-run the
attended GPU e2e and re-stamp.
