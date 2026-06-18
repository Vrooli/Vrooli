# Problems — Image Tools

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-06-16 — Documentation foundation complete; implementation not started — RESOLVED 2026-06-17

**Symptom (original):** Orientation reported 5/8 gates: `scaffold-health`,
`dependency-decisions`, and `example-domain-removed` were red because only the
planning foundation existed.

**Resolution (2026-06-17):** Phase 0 + Phase 1 landed. The `notes` worked
example is **removed** end-to-end (Gate 7); the `models`/`jobs` spine domains
replace it across api/cli/proto/ui; `.vrooli/service.json` now declares the
optional ComfyUI resource + optional ffmpeg hostTool (`dependency-decisions`).
Host hardware facts come from the root `vrooli host inventory --json` CLI behind
the `capabilities` seam (not system-monitor). API/CLI/proto green + e2e boot
test green; UI tests green (162/162). See [`PROGRESS.md`](PROGRESS.md).

**Remaining for `scaffold-health` / full `make test`:** run the complete
`vrooli scenario test image-tools` suite (standards/unit/coverage/measures/
smoke) — measures phase has nothing to grade until ops declare measures
(Phase 2+).

**Refs:** `PRD.md`, `requirements/`, [`DECISIONS.md`](DECISIONS.md), [`PROGRESS.md`](PROGRESS.md).

### 2026-06-17 — react-vite template lint + standards debt (the lone `vrooli scenario test` failure)

**Symptom:** Two surfaces, both confined to **pristine template files** (verified
present at HEAD, unchanged by feature work):
1. `phase-standards` fails — OWASP "Security Headers" HIGH ×4 on
   `api/internal/httpx/errors.go` (template error writer) + three template
   test files (`internal/middleware/logging_test.go`,
   `internal/module/module_test.go`, `internal/server/server_test.go`). As of
   Phase 4 the suite is 15/18 green; the other two reds are tracked separately
   (playbooks `@scenario/self` infra regression; proto commit-gated) and are also
   not feature-introduced.
2. `pnpm lint` (`eslint .`) reports a handful of errors (plus react-refresh
   warnings): `theme.choice.{light,dark,system}` "no callsite" (dynamic-access
   false positive — they ARE used via `strings.theme.choice[c]`),
   `AppShell.tsx` aria-label string-literal, and `ThemeProvider.tsx`
   "unnecessary conditional (always falsy)" SSR guards.

**Root cause:** All are present on the **pristine react-vite scaffold at HEAD** —
not introduced by the jobs/models migration. (Standards) the OWASP
"Security Headers" rule flags any HTTP-response write lacking security headers,
including template error-rendering (`httpx/errors.go`) and HTTP-constructing
test files — a broad heuristic that over-fires on tests; this is the documented
fleet-wide standards campaign. (Lint-1) `theme.choice.*` ARE used, via dynamic
`t(strings.theme.choice[c])` in `TopBar`/`SettingsPage`; the
`strings/no-unused-keys` rule only follows static member access, so dynamic keys
read as unused (false positive). (Lint-2) The `AppShell` `aria-label="Main
content"` literal and the `ThemeProvider` `typeof window === "undefined"` SSR
guards live in files unchanged by this work.

**Workaround:** None needed for correctness — the app and API behave correctly,
and every other feature-introduced finding the suite raised (security G109, docs
manifest, go-mod-tidy, pnpm release-age, structure/proto `proto_payloads`) was
fixed. This matches the codebase's existing treatment of react-vite
template/scaffolding standards+lint debt as a tracked **upstream-template**
campaign (cf. the ecosystem-manager UI and the fleet standards campaign), not a
per-scenario fix — fixing it per scenario diverges from the template and breaks
on the next template sync.

**Real fix:** Upstream in `templates/scenarios/react-vite` + api-core: add a
security-headers middleware to the api-core server chain (so `httpx/errors.go`
responses inherit them) and exempt `*_test.go` from the OWASP Security-Headers
rule; teach `strings/no-unused-keys` to recognize dynamic `strings.x.y[expr]`
namespace access; route the `AppShell` main-content label through i18n; and drop
the dead SSR guards in `ThemeProvider` (the SPA is client-only). Then every
generated scenario inherits the fix.

**Owner:** unassigned (react-vite template + api-core maintainers).

**Refs:** `api/internal/httpx/errors.go`, `api/internal/{middleware,module,server}/*_test.go`,
`ui/src/consts/strings.generated.ts`, `ui/src/layout/AppShell.tsx`,
`ui/src/theme/ThemeProvider.tsx`, `ui/src/layout/TopBar.tsx`,
`ui/src/pages/SettingsPage.tsx`.

### 2026-06-16 — prd-control-tower generation gotchas (tooling, not scenario)

**Symptom:** `prd-control-tower prd generate --publish` failed with
`ORPHANED_CRITICAL_TARGETS`; `prd-control-tower requirements generate`
exceeded the client HTTP timeout for a large (28-target) registry.

**Root cause:** `--publish` enforces that every P0 target already has a
linked requirement (stricter than the START-HERE flow implies), and the
requirements generator builds all modules in one synchronous LLM call
that can outrun the CLI's HTTP client timeout.

**Workaround:** Generate the PRD draft without `--publish`, read the
markdown from `.generation.generated_text` in the `--json` output, and
write `PRD.md` directly; author the requirements modules directly from the
PRD's OT ids (`prd_ref: "OT-Pn-NNN"`). Then validate with
`prd-control-tower prd validate` and `... requirements validate` (both
report healthy).

**Real fix:** N/A for this scenario — upstream prd-control-tower
ergonomics. Captured so the next scenario author doesn't re-discover it.

**Owner:** unassigned.

**Refs:** prd-control-tower CLI; `requirements/index.json`.

### 2026-06-17 — metadata `strip-gps` removes ALL metadata in v1 (privacy-safe superset)

**Symptom:** `image-tools ops metadata --strip-gps` and `--strip-all` behave
identically: both produce an output with no EXIF/IPTC/XMP at all, not just GPS
removed.

**Root cause:** Selective GPS-IFD removal that preserves other EXIF requires
rebuilding the EXIF block (dsoprea/go-exif IFD builder). v1 implements strip by
re-encoding through Go's encoders, which write no metadata — so GPS is
definitely gone, but so is everything else.

**Workaround:** None needed for the privacy goal (location IS removed). Callers
who must preserve non-GPS tags should not strip in v1.

**Real fix:** Surgically rebuild EXIF minus the GPS IFD via the go-exif builder
and re-embed it; flip this entry to resolved. Tracked for a later enhancement.

**Owner:** unassigned.

**Refs:** `api/internal/ops/metadata.go` (`stripMetadata`); DECISIONS.md.

### 2026-06-17 — Runtime measures: recorder built; per-model enrichment + query surface deferred — PARTIALLY RESOLVED

**Resolution (Phase 4):** `internal/measures` now records op latency (p50/p95),
queue-wait, throughput, and terminal-state mix into SQLite `op_measure`, fed by a
new generic `jobs.Manager.OnComplete` hook — so EVERY finalized job (deterministic
+ AI) accrues op-level metrics. `Recorder.Stats` computes the aggregates on read
(`measures_test.go` covers it). The schema + `Sample` API carry `model_id`/`tier`/
`fallback_used` columns.

**Still deferred:** (1) per-model latency + fallback-tier ENRICHMENT — the AI
runner knows the resolved model + provider tier but does not yet call
`Recorder.Record` with a model-backed sample (kept the AI engine decoupled from
measures); op-level facts are captured, model attribution is not. (2) No
read/query SURFACE (proto/handler/CLI) exposes the aggregates yet — they are
recorded but not served. The test-genie `measures` phase is inference-based
(stateful-domain coverage) and is GREEN without a manifest measures block.

**Real fix:** Have the AI runner record a model-backed `Sample` on completion
(model id + tier + fallback flag), and add a `MeasuresService.Stats` RPC + CLI
`measures` verb when an operator/consumer needs the aggregates.

**Owner:** unassigned.

**Refs:** `api/internal/measures/`, `api/internal/jobs/manager.go` (`OnComplete`),
`api/main.go` (recorder wiring), `api/internal/ai/engine.go` (enrichment seam).

### 2026-06-17 — proto red: `shared.ErrorEnvelope` REST-error payload is commit-gated

**Symptom:** `phase-proto` fails with 4 ERROR `proto.rest_payload_unknown_message`
findings — endpoints `ai_submit`/`analysis_run`/`ops_run`/`ops_blob_get` declare
their REST-exception error body as `vrooli.image_tools.v1.shared.ErrorEnvelope`,
which proto-health reports as an unknown message.

**Root cause:** Commit-gated, not a working-tree defect. The working tree is
internally consistent — `packages/proto/schemas/image-tools/v1/shared/errors.proto`
declares `package vrooli.image_tools.v1.shared` with `ErrorEnvelope`, the
generated Go/TS/Python carry `shared`, and `.vrooli/endpoints.json` references
`shared.ErrorEnvelope`. proto-health validates against the COMMITTED descriptor
(HEAD), where the `errors`→`shared` move is still uncommitted (HEAD has it in
package `errors`), so the working-tree endpoints don't match the committed
descriptor. Same commit-gated pattern as Phases 2-3.

**Workaround:** None — this branch commits via an external `image-tools pN`
process; the working tree is correct and verified.

**Real fix:** The external committer commits the working tree (including the
regenerated descriptor with `shared`); proto goes green immediately. No code
change required.

**Owner:** the branch's phase-committer.

**Refs:** `packages/proto/schemas/image-tools/v1/shared/errors.proto`,
`scenarios/image-tools/.vrooli/endpoints.json`, proto-health
`rest_payload_unknown_message`.

### 2026-06-17 — AI backend live-execution unverified on CI hosts (attended acceptance gate)

**Symptom:** `image-tools ai generate|img2img|inpaint|object-removal|upscale|bg-removal|denoise`
and `analyze ocr|nsfw` return HTTP 409/503 with an actionable install hint on a
host without the backend binaries/models. Only `analyze probe` runs end-to-end
out of the box.

**Root cause:** The standalone backends (`sd` / stable-diffusion.cpp, `iopaint`,
`realesrgan-ncnn-vulkan`, `rembg`, the onnxruntime + diffusers python sidecars,
`tesseract`) and the CPU-default model weights are not installed in CI or on this
dev host, and model download-on-first-use is owned by the Phase-4 model
management layer (IMG-P0-007). The selection → plan → refuse path is correct and
live-proven (HTTP 409 `run image-tools models install sd-1.5`).

**Workaround:** The full vertical (select → execute → persist → auto-scan) is
covered by unit/integration tests with fake providers; backend arg-builders are
unit-tested for assembly. `probe` is the live headless proof.

**Real fix:** The IMG-P0-002/003/004 **headless-completeness acceptance** is an
**attended** run on a host where the CPU default models are installed (Phase 4
`models install`). Capture it as a checklist item; flip this entry to resolved
once an attended run exercises each AI op from the CLI with no GPU/ComfyUI.

**Update (2026-06-18):** The attended gate was performed. It revealed a second,
more fundamental blocker beyond absent backends — the `models install` path is a
stub that fetches landing pages, not weights (see the 2026-06-18 entry below).
Deterministic ops + `analyze probe` run end-to-end for real; no AI op can.

**Owner:** unassigned (Phase 4 wires model download; then run the attended gate).

**Refs:** `api/internal/ai/`, `api/internal/analysis/`, `docs/internal/TESTING.md`
(headless-completeness acceptance), `api/internal/models/registry.seed.json`.

### 2026-06-17 — govulncheck advisories in golang.org/x/image — RESOLVED for image-tools

**Resolution (2026-06-17):** Bumped `golang.org/x/image` v0.25.0 → **v0.42.0**
(api/go.mod; pulled x/text → v0.38.0) with user approval; `phase-security` now
passes (0 ERROR findings) and `phase-dependencies` stays green. Governance note
recorded in `.vrooli/dependencies/approved-dependencies.json` (security floor
>=0.42.0). Codec golden/round-trip tests pass on v0.42.0 (API-compatible).
**Fleet caveat:** `chart-generator` still pins x/image v0.18.0 (indirect) and is
also affected — its bump is tracked in the governance `security_notes` but not
done here (out of Phase-3 scope). Kept `version_range: "*"` to avoid red-flagging
chart-generator before its own migration.

**Symptom (original):** `phase-security` reported 4 ERROR-severity govulncheck
findings — GO-2026-4815 (tiff IFD-offset OOM), GO-2026-4962 (sfnt excessive
allocation), GO-2026-5031 (bmp out-of-bounds palette panic), GO-2026-5032 (tiff
PackBits resource consumption) — all in `golang.org/x/image`.

**Root cause:** PRE-EXISTING. `golang.org/x/image v0.25.0` is the version at HEAD
(added by Phase 2's codec layer), and these decoders are reachable from
`internal/ops/codec.go::Decode` (committed in Phase 2). The advisories are
2026-dated and were published into the vuln DB after Phase 2's green security
run, so they surface now regardless of Phase 3. Phase 3's `analysis.Probe` reuses
the same `Decode`, but does not introduce the reachability.

**Workaround:** None at the code level — the ingest guard
(`internal/storage/guard.go`) already bounds decode dimensions/bytes, mitigating
the OOM/resource-consumption class at the boundary.

**Real fix:** Bump `golang.org/x/image` to a patched release (cached versions up
to v0.42.0 are available) across `scenarios/image-tools/api` (+ `cli` if it
imports it) and update `.vrooli/dependencies/approved-dependencies.json`. This is
a dependency change (Critical Rule #5 — requires explicit permission) and is
fleet-wide (any scenario decoding images via x/image is affected), so it should
land as a coordinated bump, not a silent per-scenario edit.

**Owner:** unassigned (needs maintainer approval for the dep bump).

**Refs:** `scenarios/image-tools/api/go.mod` (x/image v0.25.0),
`internal/ops/codec.go`, `internal/storage/guard.go`.

### 2026-06-17 — playbooks red: `@scenario/self` port resolution regression (fleet infra)

**Symptom:** `phase-playbooks` fails before any step runs (0 steps, 0 asserts):
`failed to resolve port for scenario @scenario/self: ... instance key: missing
scenario name in "@scenario/self"`.

**Root cause:** PRE-EXISTING / fleet-wide. The BAS case files use the canonical
self-reference `"scenario": "@scenario/self"` (so does every other scenario —
code-facts, search-hub, device-sync-hub, measures-health, …). The root CLI's
instance-key parser now rejects it: `vrooli scenario port image-tools API_PORT`
returns the port, but `vrooli scenario port "@scenario/self" API_PORT` errors
`missing scenario name in "@scenario/self"`. A concurrent change to the
`vrooli scenario port` / api-core instance-key parsing broke the `@scenario/self`
sentinel after Phase 2's green playbooks run. Not introduced by Phase 3 (the
editor page + ops discovery are never reached).

**Workaround:** None at the scenario level — the sentinel is resolved by the BAS
workflow runner, not editable per-scenario without diverging the case template.

**Real fix:** Restore `@scenario/self` resolution in the root `vrooli scenario
port` instance-key parser (treat `@scenario/self` as "the current scenario"
rather than parsing it as `name@variant`). Filed to scenario-qa.

**Owner:** unassigned (root vrooli CLI / api-core discovery maintainers).

**Refs:** `bas/cases/deterministic-ops/ui/editor-lists-operations.json`,
`bas/cases/routed-database/proves-test-pool-routing.json`, root `vrooli scenario
port`.

### 2026-06-18 — `models install` fetches landing pages, not weights — install path is a stub for all 49 models

**Symptom:** `image-tools models install <id> --wait` reports ✅ success in ~1s
for any model, pins a real sha256, and marks it `installed`. The downloaded
"weights" are not a model. Reproduced live: `models install u2netp` produced a
**441 KB GitHub HTML page** (`<!DOCTYPE html>…`) saved as
`~/.vrooli/data/vrooli/image-tools/models/u2netp/rembg`. The selection gate then
treats the model as installed, yet no op can run.

**Root cause:** The installer downloads whatever `source.download_url` returns
and pins *its* checksum, with **no validation that the bytes are a real model
artifact** (no content-type / magic-number / expected-extension / size-band
check, no HF-resolve or release-asset resolution). The seed's `download_url`s
are HuggingFace/GitHub **landing & release pages**, not direct weight assets —
**0 of 49 seeded models** point to a `.onnx/.safetensors/.pth/.gguf/.zip` file.
So "checksum pinned on first download" gives genuine integrity of the **wrong
bytes** — false verification. This is independent of, and compounds, the absent
backend programs (2026-06-17 entry above): an AI op is blocked by *both*.

**Workaround:** None — AI ops cannot run end-to-end. Deterministic editing +
`analyze probe` are the only genuinely-working paths today (both verified live
2026-06-18).

**Real fix:** (1) `models install` must resolve and fetch the actual weight
asset(s) per model + backend file layout (HF resolve URLs / release assets),
**validate** the downloaded artifact (expected type/magic/size, not just
self-checksum), and pin an upstream-verified checksum. (2) Add a backend-program
provisioning story — install/detect `rembg`/`sd`/`realesrgan`/`iopaint`/
`tesseract`, or pivot CPU defaults toward provisionable pure-Go/ONNX-runtime
paths — backends are currently unmanaged host prerequisites with no install
path. (3) Make the registry/selector honest about *runnable* vs *declared*. This
is the foundational phase of the image-tools advanced-editing plan.

**Owner:** unassigned (image-tools model-management).

**Refs:** `api/internal/models/` (installer / `Downloader`),
`api/internal/models/registry.seed.json` (`source.download_url` ×49), the
2026-06-17 attended-acceptance entry above.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
