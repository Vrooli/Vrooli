---
name: "launch-video"
description: "Produce a short launch-video draft for one sellable Vrooli scenario from its official advertising sources and real captures of the running product, using the third-party brag method and the HyperFrames renderer."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["marketing", "video", "launch"]
  icon: "video"
  status: "active"
  revision: 9
  createdAt: "2026-09-17T00:00:00Z"
  updatedAt: "2026-09-18T20:05:00Z"
  requires:
    scenarios: ["prompt-manager", "landing-page-business-suite", "brand-manager"]
    commands: ["prompt-manager skill read", "brand-manager", "vrooli scenario port"]
  origin:
    kind: "authored"
---
## Tools focus: Launch Video

Produce a launch-video draft for **one sellable scenario** that tells the story the product is officially advertised with and shows the real running product. Sources are read in a fixed priority order, every source used is recorded, and the render is a draft for operator judgement — never a published or approved asset.

Required reading:
- `prompt-manager skill read brag` — the third-party method (plan → composition brief → HyperFrames → render, poster, share copy). This skill changes its inputs and some of its rules; the table in §6 lists every change.
- `path:docs/marketing/catalogs/post-types/video/demo-recording.md` — the video post-type canon (claim flags, failure modes).

The HyperFrames domain skills (`hyperframes-core`, `hyperframes-creative`, `hyperframes-animation`, `hyperframes-keyframes`, `hyperframes-cli`) are not provided by Vrooli. Download them from `https://github.com/heygen-com/hyperframes` (`skills/`) into the session scratchpad and read them before you write the composition.

---

### 1. Boundaries

| In scope | Out of scope |
|---|---|
| One launch video for one scenario, or one bundle video for the landing-page home route | Non-launch videos (tutorials, ads, dev logs) |
| Reading official sources, capturing the live product, composing, rendering, poster, share copy | Uploading, posting, or scheduling the video |
| Recording which sources were used and which conflicts were found | Approving a Content Desk draft or activating a video post type |
| Reporting gaps in branding, landing-page, and marketing sources | Editing brand-manager, landing-page content, or `docs/marketing/` canon |
| Capturing a presentation instance, or live data labelled as live | Seeding a live instance, or any write to the operator's workspace |
| **Authoring the demo world inside the presentation instance** (§5b) — sessions, groups, transcripts, artifacts, and any other product data the story shows | Fixture data for surfaces the presentation instance cannot own (the control plane, the node fleet) where no seeding mechanism exists — report the gap (§5b) |

---

### 2. Resolve the product

Resolve these values before you read any story source. Write them at the top of `sources.md`.

| Value | Read from | Aquila example |
|---|---|---|
| Scenario key | the request | `web-console` |
| Product name | `scenarios/<scenario>/.vrooli/service.json` → `service.displayName` | `Aquila` |
| Brand slug | `vrooli scenario info <scenario> --json` → `.scenario.branding.brand` | `aquila` |
| Landing-page route | `/apps/<brand slug>` for one product; `/` for a bundle video | `/apps/aquila` |
| Brand record | `brand-manager assignments status <scenario>` | brand `2e797e31…` v5 |

If `branding` is null the scenario declares no brand: use the scenario key as the slug and record the assumption.

The landing route is **not** derivable from the scenario manifest — it is landing-page data. Confirm it against the live presentation (§3 row 1) rather than assuming `/apps/<slug>`.

---

### 3. Story sources, in priority order

Read every row that exists. A higher row wins a conflict. Record each conflict in `sources.md` with both values and the row that won.

| # | Source | How to read it | Use it for |
|---|---|---|---|
| 1 | Live landing-page presentation | `curl -s -X POST -H 'Content-Type: application/json' -d '{"route":"<route>"}' http://127.0.0.1:$(vrooli scenario port landing-page-business-suite API_PORT)/landing_page_business_suite.v1.LandingConfigService/GetLandingConfig` | Hero and section copy (use verbatim), section order (scene order), capability status, page theme, published assets |
| 2 | Verified Content Desk claims | `content-desk claims list` (needs the content-desk scenario running) | Which product claims the video may state |
| 3 | Campaign and product-line entries | `path:docs/marketing/strategy/CAMPAIGNS.md` and `path:docs/marketing/strategy/PRODUCT-LINE.md` (search for the product name) | Campaign angle, channels, launch date, open video slots |
| 4 | Scenario business docs | `scenarios/<scenario>/docs/business/GO-TO-MARKET.md`, `MONETIZATION.md`, the PRD branding section | Audience, positioning, pricing framing |
| 5 | Global voice and channel rules | `path:docs/marketing/strategy/STRATEGY.md`, `path:docs/marketing/strategy/CHANNELS.md`, `path:docs/marketing/catalogs/post-types/video/` | Voice, length, caption, and disclosure rules |

Rules:
- Read the landing page through the API. Do not read the presentation seed files; the published database content is the truth.
- If row 1 returns an empty presentation, the product is not listed. Continue from row 2 and state "not on the landing page" in `sources.md`.
- Show a capability whose status is `preview` or `coming soon` only with that label, or leave it out.
- Do not write a claim that no row supports.

---

### 4. Visual sources, in priority order

| # | Source | How to read it | Use it for |
|---|---|---|---|
| 1 | brand-manager brand and product-line style | `brand-manager brands get <brand id>`, `brand-manager brands tokens <brand id>`, `brand-manager assets list --brand-id <brand id>`, `brand-manager assets download <asset id> --out <file>`, `brand-manager styles get --id <style id>`, `brand-manager design generate --brand-id <brand id>` | Logo, tile style, glow, corner ratio, colors and fonts when defined |
| 2 | Landing-page theme and assets | row 1 of §3 (`presentation.page.theme`, `presentation.assets`); fonts from the `@font-face` rules in the live page CSS | Background, accent, display and body fonts |
| 3 | Scenario UI tokens and icons | `scenarios/<scenario>/ui/src/design-tokens.css`, `scenarios/<scenario>/ui/public/public/` | App base color, fallback colors, app icons |

Rules:
- Save the `design generate` output as `composition/DESIGN.md`. HyperFrames reads it as the design spec.
- A brand with empty `colors`/`typography` is not unbranded: its palette comes from the container style and product line named in `brand-manager brands get`. Read those before reporting a gap.
- For a product in a product line, the line style wins over `path:docs/marketing/strategy/IMAGE_STYLE.md`. That precedence is stated in `IMAGE_STYLE.md` §"Scope and precedence".
- A branded scenario's shipped icons and `og-image.png` are rendered by `brand-manager apply run`, not generated. Use them as-is.
- Copy font files and logo files into `composition/assets/`. Do not load them from the network at render time.

---

### 5. Footage: the real product

Show the running product. Do not rebuild the product UI in HTML. HTML is for titles, labels, highlight rings, and brand scenes only.

#### 5a. Choose the workspace you capture

Capture a **presentation instance** — a named non-live instance of the scenario that owns its own database, storage, ports, and process environment. It runs the same code as live and differs only in data. `path:docs/architecture/presentation-instances.md` is the contract.

A presentation instance starts **empty, and empty is not the deliverable.** It is a clean stage you are responsible for dressing (§5b). Choosing it removes the operator's data; it does not supply anything in its place. A launch video shot on an undressed presentation instance is worse than one shot on live, because it shows a product nobody uses.

```bash
vrooli scenario restart <scenario>          # only if the working tree changed since live started
vrooli scenario start <scenario> --instance presentation \
  --variant-dependencies <duplicable dependencies>
vrooli scenario status <scenario> --instance presentation   # read its own UI_PORT
```

- `--variant-dependencies` lists the dependencies that must resolve at this instance's own variant. A listed dependency **fails closed**: if `<dependency>@presentation` is not running, the call errors instead of answering from live. Start each listed dependency at the same variant, or leave it off the list and accept live data from it.
- Leave a dependency off the list when it cannot be duplicated (the control plane, the node fleet) or when its data is not on screen.
- The variant refuses to build while another instance of the scenario is running, because they share build outputs. Restart live first. Editing any shared package is enough to make the scenario stale and trigger this.
- Stopping the presentation instance does not affect live. Stop it when the captures are done.

Three classes of data still arrive live and must be checked in every frame before use: control-plane answers, the node fleet, and host facts (hostname, real paths, device names). Never show the operator's shell.

If a presentation instance cannot be started, capture live, show the data as it is, and put a visible "LIVE WORKSPACE · <capture date>" label in the video. Record in `sources.md` why the presentation instance was unavailable.

#### 5b. Dress the demo world

**This is product content design, not fixture generation, and it is the single largest task in this skill.** Budget for it accordingly. A launch video has to look like software people already use: a viewer decides in the first second whether this thing has users, and they decide it from how full the screen is, not from the captions.

Write the demo world **before** any capture, as a deliberate artifact:

1. **List every surface the storyboard puts on screen.** For each one, name what a heavy, competent user's copy of it would contain after a few weeks.
2. **Find the product's richest rendering surfaces and seed content that exercises them.** Read the UI source for what it can render that plain data never shows: markdown, code blocks with syntax highlighting, diagrams, tables, charts, images, math, embedded media, threaded replies. A launch video's job is to show what the product *can do*, and a surface seeded with one-line strings shows none of it. Seed the content that makes each renderer visibly work — a message containing a table and a diagram is worth more than fifty messages containing a sentence. Then put those surfaces on screen: showing the product's default view three times covers one capability, not three.
3. **Populate each surface through the product's own interfaces** — its CLI, its API, its UI — against the presentation instance, never live. Records written straight into a database bypass the validation the product applies and produce states the product cannot reach.
4. **When a surface has no CLI or RPC write path, find the ingestion surface the product itself uses.** Derived content — conversations, feeds, timelines, notifications, indexes — often has no create RPC because the product never creates it directly: it ingests it. Look for hook or webhook endpoints, file tailers and watchers, and importers, then feed those. Absence of a `create` verb is not absence of a write path, and concluding otherwise produces an empty scene. Anything ingested this way is still authored content, held to step 2.
   Ingestion surfaces are usually authenticated per instance — a token under the instance's own state directory, not the caller's. Read the presentation instance's own secret; live's will not work, and reaching for it is a sign you are pointed at the wrong instance.
5. **Author the content.** Names, titles, timestamps, and text are written, not generated filler. `Session 1 / Session 2 / Session 3` and lorem ipsum read as a demo; a viewer discounts everything around them. Vary ages, states, and lengths — uniform data looks synthetic. Invent an organisation and stay inside it: one consistent fictional team, project, and repo across every surface, so the frames agree with each other.
6. **Invent every identifier.** No real customer, employee, host, fleet node, path, or repository — including the operator's. Fictional throughout, and internally consistent.
7. **Honour the capability status from §3.** A fuller demo makes availability claims more persuasive, so the labelling rule tightens rather than relaxes: a `preview` or `coming-soon` capability stays labelled no matter how real its data looks.

Record the demo world in `sources.md`: each surface, how it was populated, roughly how much, and the command or interface used. Anyone must be able to rebuild it.

**Empty surfaces.** If a surface on the storyboard cannot be populated — no seeding mechanism exists, or it reads a single-instance service the presentation instance cannot own (the node fleet is the standard case) — you have three options, in order: populate it another way, cut the scene and tell a story the data supports, or ship the scene empty. Shipping it empty is the last resort, requires the gap in `sources.md`, and must be **reported to the operator as a defect in the video**, not filed as a footnote in a plan file. Never describe an empty surface as "honest" and move on; honesty is about claims, and an empty panel is a production failure, not a virtue.

#### 5c. Capture rules

Order of footage sources:
1. Existing captures of the current build: `~/.vrooli/evidence/`, the product's effort folder under `~/.vrooli/plan-artifacts/efforts/`, and `scenarios/<scenario>/bas/flows/`. Use them only if they show the current UI and the story scenes.
2. New captures of the running instance at its own `UI_PORT`.

- **On live, interact read-only**: select, switch views, open viewers, scroll, and type into search fields. Do not send input to sessions, create or delete items, or take over a device.
- **On a presentation instance, writing is the job** (§5b) — create, send input, and populate freely. The read-only rule protects the operator's workspace; it is not a capture technique, and applying it to a presentation instance is what produces an empty video. Confirm which instance you are pointed at before the first write: check the port against `vrooli scenario status <scenario> --instance presentation`, not the browser tab.
- Capture one clip per story scene. Name the clip after the scene.
- **Every product scene is a moving clip, not a still.** Record the product being *used*: the pointer travelling to a control, the click, the panel opening, the list scrolling, the view changing. A screenshot with a slow scale on it is not motion — it is a photograph of software, and it reads as a mockup. Stills are for holds at the end of a clip and for brand scenes only. A scene whose only movement is a camera push fails the motion gate (§9).
- Desktop captures use a 1920×1080 viewport. Phone captures use 390×844 at device scale 3.
- Drive the browser with puppeteer-core against `/usr/bin/google-chrome`, wait for `networkidle2`, then wait several more seconds before the first frame. `google-chrome --headless --screenshot` captures the loading fallback no matter what `--timeout` says.
- Record motion with the Chrome DevTools Protocol `Page.startScreencast` frame stream. Write each frame with its timestamp. Join the frames with ffmpeg concat (per-frame durations) into 30 fps H.264.
- **The screencast emits a frame only when the page changes, so the driving technique decides whether you get motion or a slideshow.** A wheel event is one discrete jump and yields one frame; a long travel driven by wheel events produces a handful of frames. Drive long travels by animating the scroll container's `scrollTop` with `requestAnimationFrame` and an ease, which yields hundreds. Sudden state changes (a panel opening) legitimately produce few frames — judge by watching the clip, not by frame count alone.
- **Run slow one-off work in a warmup phase before the screencast starts.** Lazy-loaded renderers — diagrams, charts, editors, maps — can take many seconds on their first draw headless, and filming that captures a spinner. Open the surface, wait for it to finish drawing, return it to its starting position, and only then start recording.
- **Reset the workspace before every clip.** Capturing is not read-only on a presentation instance: selections, view modes, scroll positions and panel order persist, so clip *n* starts from wherever clip *n−1* left off. Reapply a known layout before each recording, or later clips will be shot in a state the storyboard never described.
- **Measure the clips, then build the timeline to them.** Record first, read each clip's real duration, and set scene starts and durations from those numbers — snapping cuts to the nearest strong cue (§5d) rather than forcing a clip to fit a guessed slot.
- Take a still from the last frame of each clip for holds and camera pushes.
- Read every capture back (contact sheet or frames) before you use it. Read the browser console too: a presentation instance surfaces a missing follow-list dependency as a failed request, not as a visible error.
- Verify motion by sampling frames from *different* moments of the same clip and comparing them. Two identical frames mean the clip is a still with extra steps.

#### 5d. Direct the eye

A viewer cannot find the control a caption is talking about in a 1920×1080 screen in two seconds. Show them.

- **Ring the thing you name.** When a caption names a control, region, or list, put a highlight on it: an accent-coloured rounded rectangle with a soft glow, animated in over ~0.3 s, positioned in the same coordinate space as the framed capture. When the next highlight appears, drop the previous one to a low opacity rather than removing it, so the viewer keeps the context.
- **Measure ring positions; do not eyeball them.** Read the real bounding boxes out of the running page (`getBoundingClientRect` over the controls you intend to ring), then scale by the frame's display ratio — a 1920-wide capture shown 1500 wide scales by 0.78125. Measure in the same layout the clip was shot in, because the boxes move when the layout does.
- **Only ring what holds still.** A ring is absolutely positioned and does not track content that scrolls underneath it. Ring fixed chrome — toolbars, sidebars, controls — and let a camera move carry the attention on a scrolling surface instead.
- **Move the camera with intent.** Push in on the region under discussion and pan to the next one, instead of holding a full-screen view for the whole scene. Scale and translate the frame wrapper, never the page.
- **Land highlights and cuts on the music.** The bundled tracks ship beat and cue metadata in `<brag assets>/music/cues/<track>.music-cues.json`; read the preset for the chosen track and place highlight reveals on its strong cues. Readability wins over the grid: drop a cue that would rush a caption.
- Highlights, labels, and rings are HTML overlays. Never repaint the product's own UI (§5).

---

### 6. Changes to the brag method

| brag default | launch-video rule |
|---|---|
| Step 1 reads the repo (`index.html`, CSS, README) | Replace step 1 with §2–§4. Write the 9-question rubric answers from those sources. |
| Recreate UI in HTML | Use real captures (§5). |
| 15–25 s | Use the requested length. Without a request, use 30 s for landscape. |
| Tone inferred | Use `polished` unless the request names a tone. |
| Output in `brag-output/` in the project | Output in `~/.vrooli/plan-artifacts/efforts/<campaign effort>/launch-video-<date>/` if a launch effort exists; else `~/.vrooli/evidence/<brand slug>-launch-video-<date>/`. |
| Preview, then render after approval | Render a draft without approval. The draft is the review artifact. |
| Send `hyperframes feedback` after render | Do not send feedback. It posts to a public channel. |

All other brag rules apply: hook in the first 2–3 s, reading-time floors, beat locks, `hyperframes check` as the gate, poster baked as frame 0, `share-copy.txt`.

---

### 7. Output

The output folder contains:

```
sources.md             resolved product values, every source read (with revision/date), the captured instance and follow list, the demo world (§5b) surface by surface, conflicts and winners, gaps
demo-world.md          what was populated, the fictional organisation, and the exact commands to rebuild it
brag-plan.md           brag plan, storyboard mapped to landing-page sections
composition-brief.md   brag brief
composition/           HyperFrames project (index.html, DESIGN.md, assets/)
music-candidates/      every generated take plus PROVENANCE.md (§8); ship the rejects, not just the chosen track
brag.mp4               rendered draft
brag.jpg               poster
share-copy.txt         one caption, landing-page wording
```

`sources.md` records, per source: path or command, read time, what was used, and "empty" or "not available" when that is the result.

---

### 8. Tools and guardrails

- Node.js 22+, the HyperFrames CLI, ffmpeg, and headless Chrome are external tools. Install HyperFrames in the session scratchpad (`npm i hyperframes@<version>` under Node 22). Do not add it to any scenario manifest or lockfile.
- Set `HYPERFRAMES_NO_TELEMETRY=1` for every HyperFrames command.
- Use brag's bundled SFX, and its music only as a fallback when generation is unavailable. Their licence status is a **blocker to resolve, not a label to apply**: brag's own `assets/music/README.md` says the exact terms must be verified before publication, and the skill's MIT licence covers its code, not the bundled audio. Check the track's source terms, record what you find in `sources.md`, and if they are unresolved say so to the operator as an open decision with the source named — do not write "not verified" and treat the matter as closed.
- **Generate the music; do not reach for a shelf.** Read `prompt-manager skill read music-generation` and follow it. A bundled or library track whose terms are unresolved is a blocker to publication; a track generated locally from permissively licensed weights has no terms to resolve and can be directed at *this* video instead of chosen from a catalogue. Generate a batch of at least ten takes, select one for the cut, and deliver **all** of them in `music-candidates/` with `PROVENANCE.md` — the operator's most likely next instruction is "use a different one", and that is free to satisfy only if the alternatives are already there.
- **When a generator disappoints, find out what it actually received before blaming it.** Music and image models are commonly fronted by a planner or prompt-rewriter that restates the brief before the generating model sees it, and a small planner restates everything toward the bland middle: a brief asking for "144 BPM, cold, detuned, menacing" came back as 65 BPM "shimmering, melancholic, ethereal". Read the log for the prompt that reached the model, and turn the rewrite off when the brief is authored rather than sketched. Pass structured values — tempo, key, duration — in their own fields; a rewriter discards free text first.
- **Describe the sound, not the category.** "Modern tech product launch, polished, uplifting, confident" is the definition of stock music and will generate exactly that. Name the rhythm, the bass character, and the instruments: pattern and timbre, not mood and market. If the operator supplies reference tracks, mine them for tags rather than adjectives — a stock library's own genre/mood/movement labels are a better prompt vocabulary than anything invented from scratch.
- Do not render with HyperFrames cloud, Lambda, or Cloud Run. Render locally.
- Do not use git.

---

### 9. Output expectations

You may:
- Create the output folder and everything in §7.
- Start and stop a presentation instance of the product and its listed dependencies.
- Create, write, and populate freely **inside a presentation instance** (§5b).
- Capture live read-only.

You must:
- Record every source and conflict in `sources.md`.
- Record in `sources.md` which instance was captured, the follow list used, and which of the three live-data classes (§5a) appear in any frame.
- **Dress the demo world before capturing (§5b), and record it in `demo-world.md`.**
- **Apply the fullness gate before the render.** Read every scene's frames and ask of each: would a viewer believe real people use this product? A scene whose main surface is empty, single-item, or placeholder-named fails. Fix it, cut it, or report it as a defect in the draft — a failing scene may not pass silently into the render.
- **Apply the motion gate before the render.** Sample frames from the start, middle and end of every product scene. If the product itself did not move — only the camera did — the scene fails (§5c).
- **Apply the capability gate before the render.** Every scene must show a distinct surface, and the set of scenes must cover the product's strongest rendering capabilities (§5b step 2). Three scenes of the same view is one scene shown three times.
- Report to the operator, alongside the video, every surface that could not be populated and what the video shows instead.
- Restart live before starting the presentation instance when the working tree has changed.
- Pass `hyperframes check` with zero errors before the render.
- Report the video path, the poster path, the sources that drove the story, the conflicts, and the gaps to the operator.
- Record a work record with `vrooli-memory journal note --kind work-record`.

You must not:
- Publish, upload, or approve the video.
- Force a build that another running instance serves.
- Edit brand, landing-page, or marketing sources to remove a conflict. Report the conflict.
- State a claim or a capability that §3 does not support.

---

### Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Variant start refused: "needs a rebuild, but it shares build outputs with running instance(s) …" | The working tree changed since the running instances built | `vrooli scenario status <scenario>` | Restart live, then start the presentation instance. Do not force the build. |
| Every capture shows "Loading…" | `google-chrome --headless --screenshot` fires before the app mounts | `ffprobe`/byte-size of two captures taken with different `--timeout` values — identical size means neither waited | Use puppeteer-core with `networkidle2` plus a settle delay (§5c) |
| A panel is empty or a request 503s on the presentation instance only | A follow-list dependency is not running at that variant, and discovery fails closed by design | Compare the browser console against the same capture of live | Start `<dependency>@presentation`, or remove it from `--variant-dependencies` and accept its live data |
| `vrooli scenario logs --instance <variant>` is rejected both after and before the command | Known CLI defect; there is no working spelling | — | Read the instance's process output directly; report the defect |
| Terminal panes render as a small device frame in headless captures | The headless browser joins the workspace as a secondary screen | Screenshot the default view | A presentation instance has no other screen, so its terminals render full size. On live, capture the messages or viewer surfaces instead and never press Take over. |
| Screencast file is empty or not a valid video | puppeteer `page.screencast` failed in headless mode | `ffprobe` the file | Use `Page.startScreencast` frames and ffmpeg concat (§5) |
| Landing-page API returns an empty presentation | Wrong slug, or the product is not listed | Compare `branding.brand` with the home route's app list | Use `/` to list apps; if still absent, continue from §3 row 2 |
| `brand-manager design generate` shows "Not yet defined" | The brand inherits its palette instead of declaring one | `brand-manager brands get <brand id>` — it names the container style and product line | Read the palette from the container style (`brand-manager styles get --id <id>`); only call it a gap when no style is referenced either |
| `content-desk claims list` fails | content-desk is stopped | `vrooli scenario status content-desk` | Record "not available" in `sources.md`; use only landing-page claims |
| `hyperframes check` reports `0 sample(s)` / `0/0 text checks` | A lint error disabled the layout and contrast audits | Read the Lint block | Fix lint errors first, then re-run `check` |
| Rendered video is silent | An `<audio>` element has no `id` | `hyperframes lint` (`media_missing_id`) | Add an `id` to every `<audio>` |
| A clip shows wrong frames or disappears mid-scene | `<video data-start>` inside an element that also has `data-start` | `hyperframes lint` (`video_nested_in_timed_element`) | Put the video in a non-timed wrapper; animate the wrapper |
| Captures show an old UI | Reused evidence predates the current build | Compare with a fresh screenshot | Capture again (§5 source 2) |
| The product looks unused: one item per list, empty panels, placeholder names | The presentation instance was captured undressed — its emptiness was mistaken for the goal | Count the items visible in each scene's main surface | Dress the demo world (§5b) and recapture. This is the most likely failure of this skill; it passes every other gate. |
| A clip has only a handful of frames and plays as a slideshow | The travel was driven by wheel events, which emit one frame each | Count the captured frames against the seconds driven | Animate `scrollTop` with `requestAnimationFrame` (§5c) |
| A clip films a loading spinner where a diagram or chart should be | A lazy-loaded renderer's first draw is slower than the wait before capture | Screenshot the same surface after a much longer wait | Move the first draw into a warmup phase before the screencast starts (§5c) |
| A clip starts in the wrong view, session, or scroll position | An earlier recording left that state behind; capture mutates the presentation instance | Compare the clip's first frame with the storyboard | Reapply a known layout before every recording (§5c) |
| A surface shows only a summary or the first part of its content | The list view truncates and the product has a separate full-content view (reader, expand, detail) | Look for a "open in reader" / "expand" affordance in the row | Capture the full view as its own scene; it is usually the better scene |
| A seeded surface still reads empty and there is no `create` verb | The content is ingested, not created — the write path is a hook, watcher, tailer or importer | Search the API for hook/webhook routes and file watchers | Feed the ingestion surface with the instance's own token (§5b step 4) |
| A CLI fix has no effect and the command still rejects valid input | The command's `cli/manifest.json` entry still constrains it; the parser refuses before the handler runs | Compare the manifest entry against the handler's expectations | Fix both layers; a handler change alone is not a fix |
| A panel stays empty after seeding | It reads a single-instance service the presentation instance does not own (typically the node fleet) | Check whether the surface's data has any per-instance store at all | Populate another way, cut the scene, or ship it empty **and report it as a defect** (§5b) — never as an acceptable outcome |
| Generated music ignores the brief and sounds generic | A planner or prompt-rewriter restated the caption before the generating model saw it | Read the generation log for the prompt that actually reached the model | Turn the rewrite off and pass structured values (tempo, key) in their own fields, not in free text (§8) |
| Generated music sounds like stock library music | The prompt named the category and the mood instead of the rhythm and the instruments | Re-read the prompt: count adjectives versus named sounds | Describe pattern and timbre; mine the operator's reference tracks for genre/mood tags (§8) |
| A generation run OOMs intermittently on a shared GPU | The capacity broker is advisory on this host, so residency is first-come, first-served | `vrooli capacity policy get` — check `enforce` and `preempt_enabled` | Shrink your own footprint (CPU offload) and wait for headroom; do not stop another tenant's service without asking |

Promotion note: §2–§4 source resolution is deterministic. If this skill runs for several products, move it into a governed program that writes `sources.md`, and keep only story and scene judgement in this skill.
