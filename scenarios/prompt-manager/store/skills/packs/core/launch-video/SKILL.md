---
name: "launch-video"
description: "Discover offer-eligible customer value, then direct, capture, compose, critique, and deliver a source-grounded launch-video draft with inexpensive teaser and deliberate flagship profiles, using real product evidence and local HyperFrames rendering."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["marketing", "video", "launch"]
  icon: "video"
  status: "active"
  revision: 50
  createdAt: "2026-09-17T00:00:00Z"
  updatedAt: "2026-11-22T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "landing-page-business-suite", "brand-manager", "device-studio", "program-runtime"]
    commands: ["prompt-manager skill read", "brand-manager", "vrooli scenario port", "program-runtime library run", "device-studio renders export"]
  origin:
    kind: "authored"
---
## Tools focus: Launch Video

Produce a launch-video draft for **one sellable scenario**: discover useful customer workflows, select benefits the intended offer reliably provides, and show genuine product proof. Preserve approved claim meaning and record every source. The render is a draft for operator judgement, never a published or approved asset; a blocked launch stays incomplete (a concept study is no substitute).

Direct understanding before polish: problem, promise, decisive genuine product proof, a usable next action. Duration follows the workflow and readable proof unless a channel or operator limit applies; a changed value proposition means a rebrief.

| Profile (skill input, not a CLI flag) | Path | Cost boundary |
|---|---|---|
| `teaser` (default) | One story, one direction, shot plan, draft, exported-file review | About 15-30 s (24 s default); reuse licensed audio or compare 2 governed takes |
| `flagship` (major launch, hero film, investor-quality) | Three directions, four styleframes, animatic, final composition, critique, revision | Requested duration; 3 governed takes; record render, retry and generation costs |

Narration is opt-in. Method: `references/` (relative): `production.md` (checkpoints, discovery and offer eligibility, visual continuity, shots, styleframes; read before choosing scenes), `capture.md`, `safety.md` (before any lifecycle action or capture), `review.md`, `audio-review.md`, `plan-format.md` (version 3; `--historical` for retained 1 and 2), `examples.json` and `learning.md` (`scripts/production.py examples <intent>`; never copy a reference film), plus `broll.md`, `screen-presentation.md`, `channel-variants.md`, `upstream.md`. `scripts/production.py --help` covers plan checks, shot retrieval and media inspection; `scripts/workflow.py` freezes render inputs and joins verification. Load only relevant upstream HyperFrames guidance (recorded commit, session scratch); run no upstream self-update or feedback command. Vrooli owns authority, truth, provenance and delivery; HyperFrames owns composition and rendering.

### 1. Boundaries

In scope: one launch video for one scenario (or the landing-page home route's bundle video), with its sources, capture, composition, render, poster, share copy, recorded conflicts and gaps, and the demo world on a presentation instance. Out of scope: non-launch videos; uploading, posting, scheduling; approving a Content Desk draft; editing brand-manager, landing-page or `docs/marketing/` canon; seeding live or writing to the operator's workspace; surfaces the instance cannot own (control plane, node fleet), reported as gaps.

### 2. Resolve the product

Write at the top of `sources.md`: scenario key; product name (`scenarios/<scenario>/.vrooli/service.json` `service.displayName`); brand slug (`vrooli scenario info <scenario> --json` `.scenario.branding.brand`; null means the scenario key, assumption recorded); brand record (`brand-manager assignments status <scenario>`); landing route (`/apps/<brand slug>`, `/` for a bundle), confirmed against the live presentation (landing-page data, not manifest data).

### 3. Story sources, in priority order

Resolve audience, offer, deliverable, variant and deployment targets first (Offer Desk graph and release path, package and readiness evidence); commercial intent does not prove reliability. Read every row that exists; a higher row wins; record each conflict with both values and the winner in `sources.md`.

| # | Source | How to read | Use for |
|---|---|---|---|
| 1 | Live landing page | `curl -s -X POST -H 'Content-Type: application/json' -d '{"route":"<route>"}' http://127.0.0.1:$(vrooli scenario port landing-page-business-suite API_PORT)/landing_page_business_suite.v1.LandingConfigService/GetLandingConfig` | Approved meaning, availability qualifiers, exact-copy limits, theme, assets |
| 2 | Verified Content Desk claims | `content-desk claims list` (stopped: record "not available") | Claims the video may state |
| 3 | Campaign, business and voice docs | `docs/marketing/strategy/CAMPAIGNS.md`, `PRODUCT-LINE.md`, `STRATEGY.md`, `CHANNELS.md`, `docs/marketing/catalogs/post-types/video/`, `scenarios/<scenario>/docs/business/GO-TO-MARKET.md`, `MONETIZATION.md`, PRD branding | Angle, channels, date, audience, pricing, voice, length, captions, disclosure |

Limits: read the landing page through the API, never seed files (empty presentation: not listed; continue from row 2 and say so). Keep availability qualifications; narrow or omit a claim the film would mislead on without an in-film qualifier (the operator rejected persistent demo, mock or date labels; provenance belongs in the report and share context). Write no claim a row does not support; withhold a disputed one. Claim fidelity, not page order: reorder, omit, combine or paraphrase without widening scope, certainty, availability, performance or pricing, keeping wording, source and qualification in `product-truth.md`. Locked or legally fixed wording stays verbatim; a meaning-changing rewrite is recorded and omitted until approved. Discovery, `product-brief.json` and offer eligibility follow `references/production.md`.

### 4. Visual sources and authority

Brand Manager owns identity assets and explicit restrictions; the landing page is a visual reference, not a layout mandate. Choose a product-specific treatment, record its relationship to the sources, preserve shipped marks and report conflicts. Read `brand-manager brands get <brand id>`, `brands tokens <brand id>`, `assets list --brand-id <brand id>`, `assets download <asset id> --out <file>`, `styles get --id <style id>` and `design generate --brand-id <brand id>` (save its output as `composition/DESIGN.md`), the rendered route (`references/production.md`, "Landing-page visual continuity"), and `scenarios/<scenario>/ui/src/design-tokens.css`. An empty brand `colors` or `typography` is not unbranded: read the container style and product line first; a product-line style beats `docs/marketing/strategy/IMAGE_STYLE.md`. Use shipped icons and `og-image.png` (from `brand-manager apply run`) as is. Copy fonts and logos into `composition/assets/`; load nothing from the network at render time. Compare video frames with the saved references before expensive rendering.

Device treatment goes through `device-studio.launch-video-shot` in Program Runtime (`save`, `submit` with a stable idempotency key, `observe`, `cancel`, then `device-studio renders export` to a new path); see `references/screen-presentation.md`. Add no polling ledger or render loop.

### 5. Footage: the real product

Show the actual product; never rebuild its interface in composition HTML. Prefer a **presentation instance** (`docs/architecture/presentation-instances.md`). Complete `safety-report.md` from `references/safety.md` (what database, storage, dependencies and effects follow the instance), then start only what it establishes as safe:

```bash
vrooli scenario start <scenario> --instance presentation --variant-dependencies <verified-duplicable-dependencies>
vrooli scenario status <scenario> --instance presentation
```

The follow list changes dependency discovery and does not isolate startup; a listed missing variant fails closed (never remove it). A shared-build refusal is a stop, never a reason to restart live or rebuild common packages. Stop only instances you started. If safe capture is unavailable, reuse current verified evidence or report the blocker; live read-only capture needs the task's authority and a privacy assessment, never an automatic fallback. Authorized mocked data uses the bounded frontend fixture path (`capture_mode: ui-fixture`, `proof_scope: ui`, claims limited to what the UI shows).

**An instance starts empty and empty is not the deliverable**: an undressed instance is worse than live. Dress the demo world before any capture through the product's own interfaces (never live, never a database write, invented identifiers only; method and fallback in `references/capture.md`) and record it in `demo-world.md` and `sources.md`. A surface that cannot be populated is populated another way, cut, or shipped empty as a recorded defect.

Reuse footage of the current build (`~/.vrooli/evidence/`, the effort folder, `scenarios/<scenario>/bas/flows/`) only if it shows the current UI and story scenes, else at the instance's `UI_PORT`. Prove each benefit with a connected sequence of real actions and results (never the first open or camera motion alone) at source pixels that meet output pixels on both axes; read every capture back. **Keep every reject in `alternates/footage/` with a line on why.** Never repaint the product; highlights and labels are HTML overlays; land cuts on the music cues (`<brag assets>/music/cues/<track>.music-cues.json`) unless readability suffers.

### 6. Delivery

Output goes to `~/.vrooli/plan-artifacts/efforts/<campaign effort>/launch-video-<date>/` if a launch effort exists, else `~/.vrooli/evidence/<brand slug>-launch-video-<date>/`. Render a draft without approval and never send `hyperframes feedback` (it posts publicly); brag's step 1 is replaced by §2-§4. HyperFrames checks are a technical prerequisite, not creative acceptance; re-review the file after any mastering or poster change. Deliver decisions the operator can reverse cheaply: every asset class you chose from ships its rejects.

```
README.md  DECISIONS.md  sources.md  demo-world.md  brag-plan.md  production.json  feedback.json
qualification.json  safety-report.md  product-truth.md  quality-review.json  quality-review.md
audio-selection.md  composition-brief.md  visual-reference/  composition/ (index.html, DESIGN.md, assets/)
alternates/{music (every take + PROVENANCE.md), footage, posters, copy}/
brag.mp4  brag.jpg  share-copy.txt
```

`sources.md` records per source the path or command, read time and use (or "empty", "not available"); `quality-review.json` binds technical, viewer-comprehension and craft findings to the final SHA-256. `DECISIONS.md` is read first: one row per choice with a plausible alternative (track and take, clip framing, scene order, any cut, caption wording, poster, tone), with Chose, Alternatives, Why and **Cost to switch** in time and rework ("re-render only", "recapture one clip", "rebuild the demo world"); a ruled-out option with no alternative still gets a row. An empty `alternates/` subfolder is a finding.

### 7. Tools and guardrails

- For repeatable beat, type, transition and camera timing, use the versioned
  `scripts/motion_library.py` compiler described in `references/production.md`
  and start from `references/motion-plan.example.json`. Keep the reviewed cue
  sheet, request and compiled plan with frozen composition inputs; apply the
  resolved plan in the composition code and retain the editor-reviewed cut map.
- HyperFrames is a managed host tool: `vrooli host hyperframes` with explicit Node, browser, FFmpeg and FFprobe identities, provisioned through SDA's `host-tool` surface, never raw package-manager installs, implicit `npx` downloads or scratch launchers; render locally with `HYPERFRAMES_NO_TELEMETRY=1`, never on HyperFrames cloud, Lambda or Cloud Run; see `references/safety.md`.
- Audio licence status is a blocker: verify the terms of brag's bundled SFX and fallback music before publication, record them in `sources.md`, and name the source when putting an unresolved licence to the operator.
- Launch music comes only from the judged Music Library bank (`music-library catalog reserve --purpose launch-video --project <project-id> --revision <revision-id> --consumer <stable-run-id>`; commit, release, rollback and accept lifecycle, generation fallback and the rule that unjudged takes stay candidates are in `references/audio-review.md`). Keep every candidate in `alternates/music/`.
- For a multi-cut, evidence-complete delivery, use the bounded coordinator in `references/pipeline.md`. It composes existing validators, freezes, managed HyperFrames, LV3 and cleanup receipts; it does not capture surfaces, judge taste, stage a landing-page embed or publish.
- Do not commit or publish output unless the operator requests it.

### 8. Output expectations and gates

You may create the output folder, start and stop only instances the safety assessment establishes as safe, populate verified instance-owned surfaces through supported interfaces, and capture live read-only only when authorized. You must not publish, upload or approve the video, force a build another running instance serves, edit brand, landing-page or marketing sources to remove a conflict, or state a claim §3 does not support. A required live restart or unsafe shared build is a capture blocker: record it, continue independent work.

Gates before done:
- **Delivery:** folder checked against §6; each absent file named with why.
- **Fullness (before render):** an empty, single-item or placeholder-named main surface is fixed, cut or reported as a defect.
- **Scene purpose:** each scene adds evidence, context, emotion or closure to the one promise; decisive proof is checked against observed product evidence.
- **Craft:** version-3 benefit, offer and scene checks before capture and polish, then the gates in `references/production.md` (flagship: styleframes and animatic first).
- **Technical:** `hyperframes check` zero errors; hashes re-checked after rendering; audio verified (listen, or record the gap); CTA, public URL, destination and QR verified from compressed final frames (`references/review.md`).
- **Review:** technical, viewer-comprehension and craft reviews of the exact exported MP4, scene by scene, bound to its hash. Missing product proof is a delivery failure even if a safe study renders.
- Budgets, owned jobs, cleanup and receipts recorded; the report names video and poster paths, driving sources, conflicts, gaps and unpopulated surfaces, plus a work record (`vrooli-memory journal note --kind work-record`).

### Troubleshooting & Edge Cases

| Symptom | Cause and fix |
|---|---|
| "Loading..." frames, empty or slideshow screencast, spinner clip, wrong start view | Mechanics in `references/capture.md` (settle, screencast frames, `scrollTop` animation, warm-up, reset) |
| A panel is empty or 503s on the instance only | A follow-list dependency is not running at that variant; start `<dependency>@presentation` after its safety assessment, else cut and record the gap |
| The product looks unused, or a seeded panel stays empty | Undressed instance (likeliest failure; it passes every other gate): dress the demo world; no `create` verb means feed the ingestion surface; a single-instance service means populate another way, cut, or report |
| A generation run OOMs on a shared GPU | Capacity broker is advisory: `vrooli capacity policy get`, shrink your own footprint and wait; never stop another tenant |
