---
name: "launch-video"
description: "Direct, capture, compose, critique, and deliver a source-grounded launch-video draft with inexpensive teaser and deliberate flagship profiles, using real product evidence and local HyperFrames rendering."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["marketing", "video", "launch"]
  icon: "video"
  status: "active"
  revision: 15
  createdAt: "2026-09-17T00:00:00Z"
  updatedAt: "2026-09-22T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "landing-page-business-suite", "brand-manager"]
    commands: ["prompt-manager skill read", "brand-manager", "vrooli scenario port"]
  origin:
    kind: "authored"
---
## Tools focus: Launch Video

Produce a launch-video draft for **one sellable scenario** that tells the story the product is officially advertised with and shows the real running product. Sources are read in a fixed priority order, every source used is recorded, and the render is a draft for operator judgement — never a published or approved asset.

Choose `profile: teaser` unless the request calls for a major launch, hero film,
flagship, or investor-quality demonstration. Choose `flagship` for those requests.
These are skill inputs, not flags on a Vrooli CLI. Narration is opt-in.

**An upgrade is additive:** keep populated real product captures and the requested
music character, then add direction, custom brand animation, and editorial craft.
Do not substitute an all-typography/schematic film or a hand-synthesized score for
those requirements. The operator rejected exactly that substitution in revision 12.
Before polishing, inspect usable clips and verify that captures occupy at least
half the film, with a sustained interaction/result as the decisive proof. Plan
custom animation around those clips. A blocked launch request stays blocked; an
unsolicited concept study is not its replacement, baseline, or upgraded output.
A deliberately product-light brand film is a different brief, not an exception
silently invented by the agent.

| Profile | Production path | Cost boundary |
|---|---|---|
| teaser | One core story, one direction, shot plan, draft, exported-file review | Approximately 15–30 s; reuse licensed audio or compare 2 governed takes |
| flagship | Three distinct directions, four styleframes, animatic, final composition, critique and revision | Requested duration; compare 3 governed takes initially; record render/retry and generation costs |

Read `references/production.md` for the checkpoint sequence and visual classes.
Use its accepted Vega example to calibrate the capture/animation balance. Derive
each new product's appearance from that product's sources; do not copy Vega's theme.
Read `references/safety.md` before any lifecycle action or capture.
Read `references/audio-review.md` when choosing audio or reviewing an export.
Use `scripts/production.py --help` for file-only plan checks, targeted shot retrieval,
and delivered-media inspection. Read `references/plan-format.md` when writing the plan.
Resolve these paths relative to this skill's canonical directory.

Required source doctrine: `path:docs/marketing/catalogs/post-types/video/demo-recording.md`.
Upstream methods and licenses: `references/upstream.md`. Load only relevant upstream
HyperFrames domain guidance from a recorded commit into session scratch; do not
install another end-to-end skill or run upstream self-update/public-feedback commands.
Vrooli owns authority, truth, media provenance, creative decisions, and delivery.
HyperFrames owns its composition and rendering contract.

---

### 1. Boundaries

| In scope | Out of scope |
|---|---|
| One launch video for one scenario, or one bundle video for the landing-page home route | Non-launch videos (tutorials, ads, dev logs) |
| Reading official sources, capturing the running product, composing, rendering, poster, share copy | Uploading, posting, or scheduling the video |
| Recording which sources were used and which conflicts were found | Approving a Content Desk draft or activating a video post type |
| Reporting gaps in branding, landing-page, and marketing sources | Editing brand-manager, landing-page content, or `docs/marketing/` canon |
| Capturing a presentation instance, or live data labelled as live | Seeding a live instance, or any write to the operator's workspace |
| **Authoring the demo world inside the presentation instance** (§5b) — sessions, groups, transcripts, artifacts, and any other product data the story shows | Fixture data for surfaces the presentation instance cannot own (the control plane, the node fleet) where no seeding mechanism exists — report the gap (§5b) |

---

### 2. Resolve the product

Resolve these values before you read any story source. Write them at the top of `sources.md`.

| Value | Read from | Example |
|---|---|---|
| Scenario key | the request | `<scenario>` |
| Product name | `scenarios/<scenario>/.vrooli/service.json` → `service.displayName` | `<product name>` |
| Brand slug | `vrooli scenario info <scenario> --json` → `.scenario.branding.brand` | `<brand slug>` |
| Landing-page route | `/apps/<brand slug>` for one product; `/` for a bundle video | `/apps/<brand slug>` |
| Brand record | `brand-manager assignments status <scenario>` | resolved brand ID and revision |

If `branding` is null the scenario declares no brand: use the scenario key as the slug and record the assumption.

The landing route is **not** derivable from the scenario manifest — it is landing-page data. Confirm it against the live presentation (§3 row 1) rather than assuming `/apps/<slug>`.

---

### 3. Story sources, in priority order

Read every row that exists. A higher row wins a conflict. Record each conflict in `sources.md` with both values and the row that won.

| # | Source | How to read it | Use it for |
|---|---|---|---|
| 1 | Live landing-page presentation | `curl -s -X POST -H 'Content-Type: application/json' -d '{"route":"<route>"}' http://127.0.0.1:$(vrooli scenario port landing-page-business-suite API_PORT)/landing_page_business_suite.v1.LandingConfigService/GetLandingConfig` | Approved meaning, availability qualifiers, exact-copy restrictions when stated, page theme, published assets |
| 2 | Verified Content Desk claims | `content-desk claims list` (needs the content-desk scenario running) | Which product claims the video may state |
| 3 | Campaign and product-line entries | `path:docs/marketing/strategy/CAMPAIGNS.md` and `path:docs/marketing/strategy/PRODUCT-LINE.md` (search for the product name) | Campaign angle, channels, launch date, open video slots |
| 4 | Scenario business docs | `scenarios/<scenario>/docs/business/GO-TO-MARKET.md`, `MONETIZATION.md`, the PRD branding section | Audience, positioning, pricing framing |
| 5 | Global voice and channel rules | `path:docs/marketing/strategy/STRATEGY.md`, `path:docs/marketing/strategy/CHANNELS.md`, `path:docs/marketing/catalogs/post-types/video/` | Voice, length, caption, and disclosure rules |

Rules:
- Read the landing page through the API. Do not read the presentation seed files; the published database content is the truth.
- If row 1 returns an empty presentation, the product is not listed. Continue from row 2 and state "not on the landing page" in `sources.md`.
- Show a capability whose status is `preview` or `coming soon` only with that label, or leave it out.
- Do not write a claim that no row supports. A marketing source is not proof of shipped behavior: reconcile it with current product evidence. Withhold a disputed claim; source priority never authorizes a known falsehood.
- Preserve claim fidelity, not page order. Reorder, omit, combine, or paraphrase supported claims without widening scope, certainty, availability, performance, or pricing. Retain the original, proposed wording, source, and qualification in `product-truth.md`.
- Keep legally fixed wording and explicitly locked claims verbatim. If a rewrite would change meaning or a policy requires new approval, record the requested rewrite and omit it until approved. Do not approve it yourself.

---

### 4. Visual sources and authority

Brand Manager owns identity assets and explicit brand restrictions. The current
official landing page supplies the video's presentation treatment by default.
Use the rows below by role, not as a global palette override. A different page
accent does not authorize recoloring a shipped logo, nor does a generic brand
color erase the page's established treatment. Record any explicit-rule conflict.

| # | Source | How to read it | Use it for |
|---|---|---|---|
| 1 | brand-manager brand and product-line style | `brand-manager brands get <brand id>`, `brand-manager brands tokens <brand id>`, `brand-manager assets list --brand-id <brand id>`, `brand-manager assets download <asset id> --out <file>`, `brand-manager styles get --id <style id>`, `brand-manager design generate --brand-id <brand id>` | Logo, tile style, glow, corner ratio, colors and fonts when defined |
| 2 | Rendered landing-page design, theme and assets | row 1 of §3 (`presentation.page.theme`, `presentation.assets`), plus screenshots and computed styles of the published product route | Palette roles, type hierarchy, surfaces, lighting, motifs, framing, motion and assets |
| 3 | Scenario UI tokens and icons | `scenarios/<scenario>/ui/src/design-tokens.css`, `scenarios/<scenario>/ui/public/public/` | App base color, fallback colors, app icons |

Rules:
- Save the `design generate` output as `composition/DESIGN.md`. HyperFrames reads it as the design spec.
- A brand with empty `colors`/`typography` is not unbranded: its palette comes from the container style and product line named in `brand-manager brands get`. Read those before reporting a gap.
- For a product in a product line, the line style wins over `path:docs/marketing/strategy/IMAGE_STYLE.md`. That precedence is stated in `IMAGE_STYLE.md` §"Scope and precedence".
- A branded scenario's shipped icons and `og-image.png` are rendered by `brand-manager apply run`, not generated. Use them as-is.
- Copy font files and logo files into `composition/assets/`. Do not load them from the network at render time.
- **Carry the landing page's visual identity into the video.** Inspect the rendered product route, not only its JSON theme. Save opening/product/ending references. Extend `composition/DESIGN.md` with a source-to-video mapping using `references/production.md` §"Landing-page visual continuity". A matching logo and two colors are insufficient.
- Preserve shipped marks and explicit brand restrictions. Use the landing page's established presentation treatment without requiring a separate styling request. Record differing tokens by role and the selected authority. If the page conflicts with an explicit mandatory restriction, retain that restriction and report the mismatch. Do not modify brand or landing-page canon to remove it.
- Compare representative video frames beside the saved page references before rendering. Record what carries over and why any cue changes. Keep the film's own story order, readable timing, and real product captures. Do not replace those captures with illustrative landing-page mockups or recolor the app to match the composition.

---

### 5. Footage: the real product

Show the actual product. Do not rebuild its interface in composition HTML.
Custom typography, brand geometry, transitions, and explanatory animation are
welcome around real captures; they should amplify the product rather than replace
it. Product-derived crops/layers retain their captured content and provenance.

#### 5a. Choose the workspace you capture

Prefer a **presentation instance** — a named non-live instance with separate ports and process environment. Verify which database, storage, dependencies, and background effects actually follow that instance; host metrics and shared services may remain live. Code and build artifacts can be shared. `path:docs/architecture/presentation-instances.md` is the contract.

A presentation instance starts **empty, and empty is not the deliverable.** It is a clean stage you are responsible for dressing (§5b). Verified instance-owned stores separate authored content from the operator’s data; shared surfaces need their own assessment. A launch video shot on an undressed presentation instance is worse than one shot on live, because it shows a product nobody uses.

Complete `safety-report.md` using `references/safety.md` first. Record current
runtime state, source/build locations, dependency startup behavior, discovery routing,
resource ownership, background effects, and the operator's protected surfaces.
Start only the presentation instance and dependencies the assessment establishes as safe.

```bash
vrooli scenario start <scenario> --instance presentation \
  --variant-dependencies <verified-duplicable-dependencies>
vrooli scenario status <scenario> --instance presentation
```

The follow list changes dependency discovery; it does not isolate dependency startup.
Unlisted dependencies and explicit endpoint overrides can still reach live services.
A listed missing variant fails closed. Do not remove it to make a failed shot appear
healthy. Shared build refusal is a stop condition, never a direction to restart live.
Do not rebuild common packages or critical surfaces to obtain footage.

Check host facts, control-plane responses, node data, and background jobs separately.
Stop only instances this production started, through lifecycle commands, after capture.
If safe capture is unavailable, reuse only current verified evidence or report the
capture blocker. Live read-only capture requires the task's authority and an explicit
privacy assessment; there is no automatic live fallback. A labeled explanatory study may be planned internally, but do not polish and
present it as the answer to a launch request. Only an explicit study request may
pass the study polish gate.

When the operator authorizes mocked presentation data, inspect a bounded frontend
fixture path before giving up on capture: a copied, unmodified frontend, authored
schema-validated responses, and a browser that blocks all unhandled network calls.
Capture its actual controls and rendering; label demo data visibly. Record source
and bundle hashes, fixture responses, blocked requests and browser errors. This is
`capture_mode: ui-fixture`, `proof_scope: ui`; it is not a running presentation
scenario or proof of backend collection, persistence, external execution, or speed.
Restrict its claims accordingly. Never fake an AI investigation or invent controls.
Normal instance capture remains preferred. See the safety reference for choosing
between a data variant, Baseline Modes, and an isolated frontend capture.

#### 5b. Dress the demo world

**This is product content design.** Budget for the surfaces the selected story needs. Populate coherent, useful content so each captured state communicates a credible workflow; density alone is not quality.

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
- **On a presentation instance, writing is the job** (§5b) — create and populate only verified instance-owned surfaces within the safety assessment. The read-only rule protects the operator's workspace; it is not a capture technique, and applying it to a presentation instance is what produces an empty video. Confirm which instance you are pointed at before the first write: check the port against `vrooli scenario status <scenario> --instance presentation`, not the browser tab.
- Capture one clip per story scene. Name the clip after the scene.
- **Every product scene is a moving clip, not a still.** Record the product being *used*: the pointer travelling to a control, the click, the panel opening, the list scrolling, the view changing. A screenshot with a slow scale on it is not motion — it is a photograph of software, and it reads as a mockup. Stills are for holds at the end of a clip and for brand scenes only. A scene whose only movement is a camera push fails the motion gate (§9).
- Choose capture dimensions from the shot crop and final display size. A 1920×1080 viewport is a starting point, not a zoom budget. Record actual encoded dimensions, crop in source pixels, and maximum output footprint; require at least one source pixel per output pixel on both axes. Reframe or recapture a deficient shot. Phone capture may use 390×844 at device scale 3; verify the encoded clip retains that resolution.
- Drive the browser with puppeteer-core against `/usr/bin/google-chrome`, wait for `networkidle2`, then wait several more seconds before the first frame. `google-chrome --headless --screenshot` captures the loading fallback no matter what `--timeout` says.
- Record motion with the Chrome DevTools Protocol `Page.startScreencast` frame stream. Write each frame with its timestamp. Join the frames with ffmpeg concat (per-frame durations) into 30 fps H.264.
- **The screencast emits a frame only when the page changes, so the driving technique decides whether you get motion or a slideshow.** A wheel event is one discrete jump and yields one frame; a long travel driven by wheel events produces a handful of frames. Drive long travels by animating the scroll container's `scrollTop` with `requestAnimationFrame` and an ease, which yields hundreds. Sudden state changes (a panel opening) legitimately produce few frames — judge by watching the clip, not by frame count alone.
- **Run slow one-off work in a warmup phase before the screencast starts.** Lazy-loaded renderers — diagrams, charts, editors, maps — can take many seconds on their first draw headless, and filming that captures a spinner. Open the surface, wait for it to finish drawing, return it to its starting position, and only then start recording.
- **Reset the workspace before every clip.** Capturing is not read-only on a presentation instance: selections, view modes, scroll positions and panel order persist, so clip *n* starts from wherever clip *n−1* left off. Reapply a known layout before each recording, or later clips will be shot in a state the storyboard never described.
- **Measure the clips, then build the timeline to them.** Record first, read each clip's real duration, and set scene starts and durations from those numbers — snapping cuts to the nearest strong cue (§5d) rather than forcing a clip to fit a guessed slot.
- Take a still from the last frame of each clip for holds and camera pushes.
- Read every capture back (contact sheet or frames) before you use it. Read the browser console too: a presentation instance surfaces a missing follow-list dependency as a failed request, not as a visible error.
- Verify motion by sampling frames from *different* moments of the same clip and comparing them. Two identical frames mean the clip is a still with extra steps.
- **Keep every capture you take, including the ones you reject.** A clip you shot and did not use goes to `alternates/footage/` with one line on why (§7). Deleting it is the expensive mistake: a rejected clip costs a few hundred KB, and re-shooting it later costs a whole capture session with the presentation instance restarted and the demo world rebuilt. This applies to framings, takes, scene orders and poster frames alike — if you made a choice, the thing you chose against ships next to the thing you chose.

#### 5d. Direct the eye

A viewer cannot find the control a caption is talking about in a 1920×1080 screen in two seconds. Show them.

- **Choose one focal cue.** Use crop, framing, contrast, a cursor, or one measured highlight to identify the subject. Add a ring only when the composition needs it. Remove a previous cue when it competes with the new focal point; do not accumulate glowing rectangles.
- **Measure ring positions; do not eyeball them.** Read the real bounding boxes out of the running page (`getBoundingClientRect` over the controls you intend to ring), then scale by the frame's display ratio — a 1920-wide capture shown 1500 wide scales by 0.78125. Measure in the same layout the clip was shot in, because the boxes move when the layout does.
- **Only ring what holds still.** A ring is absolutely positioned and does not track content that scrolls underneath it. Ring fixed chrome — toolbars, sidebars, controls — and let a camera move carry the attention on a scrolling surface instead.
- **Move the camera with intent.** Name what the viewer should notice because of each major move. Plan anticipation, action, settle, and a readable hold. A locked camera is valid. Scale and translate the capture wrapper, never repaint the product. Avoid constant drift and arbitrary perspective.
- **Land highlights and cuts on the music.** The bundled tracks ship beat and cue metadata in `<brag assets>/music/cues/<track>.music-cues.json`; read the preset for the chosen track and place highlight reveals on its strong cues. Readability wins over the grid: drop a cue that would rush a caption.
- Highlights, labels, and rings are HTML overlays. Never repaint the product's own UI (§5).

---

### 6. Changes to the brag method

| brag default | launch-video rule |
|---|---|
| Step 1 reads the repo (`index.html`, CSS, README) | Replace step 1 with §2–§4. Write the 9-question rubric answers from those sources. |
| Recreate UI in HTML | Use real captures (§5). |
| 15–25 s | Use the requested length; teaser defaults to 24 s. Flagship length follows the selected story. |
| Tone inferred | Derive story, camera, density, typography, and audio from this product; flagship compares three directions. |
| Output in `brag-output/` in the project | Output in `~/.vrooli/plan-artifacts/efforts/<campaign effort>/launch-video-<date>/` if a launch effort exists; else `~/.vrooli/evidence/<brand slug>-launch-video-<date>/`. |
| Preview, then render after approval | Render a draft without approval. The draft is the review artifact. |
| Send `hyperframes feedback` after render | Do not send feedback. It posts to a public channel. |

Retain brag’s early hook, reading-time floors, optional cue synchronization, poster selection, and share copy. HyperFrames checks are a technical prerequisite, not creative acceptance. Review the delivered file after all mastering or poster changes. Prefer a strong natural opening frame; do not insert a one-frame poster flash without reviewing playback.

---

### 7. Delivery

**You are not delivering a video. You are delivering a set of decisions the operator can reverse cheaply.** A draft is for judgement, and judgement needs the alternatives in the room: the operator cannot ask "use that other one" about a take you deleted, and cannot ask "why not the other order" about a choice they cannot see you made. Shipping only the winner turns every piece of feedback into a re-run of the whole skill.

The delivery folder contains:

```
README.md              what this is, what to look at first, what to say if you want it changed
DECISIONS.md           every choice that had a plausible alternative, with the switch cost (below)
sources.md             resolved product values, every source read (with revision/date), the captured instance and follow list, the demo world (§5b) surface by surface, conflicts and winners, gaps
demo-world.md          what was populated, the fictional organisation, and the exact commands to rebuild it
brag-plan.md           core story and source-linked storyboard, ordered for film
production.json        profile, claims, shots, timing, provenance, and review references
safety-report.md      verified capture/startup boundaries and actual actions
product-truth.md      supported claims, qualifications, restrictions, and brand tokens
visual-reference/     rendered landing-page references; mapping and comparison in composition/DESIGN.md
quality-review.json   timestamped critique bound to delivered file SHA-256
quality-review.md     readable critique, revision outcomes, and remaining limitations
audio-selection.md    candidates actually auditioned, measurements, selection reason
composition-brief.md   brag brief
composition/           HyperFrames project (index.html, DESIGN.md, assets/)
alternates/
  music/               every generated take plus PROVENANCE.md (§8)
  footage/             every clip captured and not used, plus why (§5c)
  posters/             several poster frames, not the one you silently picked
  copy/                the captions, kickers and titles you wrote and did not use
brag.mp4               rendered draft
brag.jpg               poster
share-copy.txt         one caption preserving approved meaning
```

`sources.md` records, per source: path or command, read time, what was used, and "empty" or "not available" when that is the result.

**`DECISIONS.md` is the one the operator reads first.** One row per decision, and the last column is what makes it useful:

| Decision | Chose | Alternatives | Why | Cost to switch |
|---|---|---|---|---|
| Track | `take-02` | other retained takes in `alternates/music/` | only take on the briefed tempo with the strongest hi-hats | ~5 min — re-snap the grid to the new tempo, re-render |
| Machines scene | **cut** | — | no seeding path for the node registry | expensive — needs a seeding mechanism first |

The switch cost tells the operator which pushbacks are free and which are a day of work, so they know what is worth asking for. State it in time and in what has to be redone — "re-render only", "recapture one clip", "rebuild the demo world". A decision with no alternative still gets a row: record what was ruled out and why, especially a scene you cut.

Record a row for every choice you actually made, at minimum: the track and take, each clip's framing, the scene order, any scene cut, the caption and kicker wording, the poster frame, and the tone. If you found yourself weighing something, it is a row.

Empty `alternates/` subfolders are a finding, not a tidy result: either you generated no options, which means you did not explore, or you deleted them, which §5c forbids. Say which.

---

### 8. Tools and guardrails

- Node.js, the pinned HyperFrames CLI, ffmpeg, and headless Chrome are external tools. Reuse a verified installed toolchain first. Route any install through `scenario-dependency-analyzer` under `path:docs/package-governance.md`, including scratch installations. Never invoke a raw package-manager install or implicit `npx` download. Record versions and tool paths. Do not rebuild the platform to obtain a renderer.
- Set `HYPERFRAMES_NO_TELEMETRY=1` for every HyperFrames command.
- Use brag's bundled SFX, and its music only as a fallback when generation is unavailable. Their licence status is a **blocker to resolve, not a label to apply**: brag's own `assets/music/README.md` says the exact terms must be verified before publication, and the skill's MIT licence covers its code, not the bundled audio. Check the track's source terms, record what you find in `sources.md`, and if they are unresolved say so to the operator as an open decision with the source named — do not write "not verified" and treat the matter as closed.
- **Route model-generated music through the governed capability.** Read `music-tools styles list` and the current compose contract. Reuse suitable provenance-complete takes when available. Otherwise begin with the profile’s bounded candidate count, structured duration/tempo/key, and capacity admission; wait through `music-tools jobs wait <job-id>` and retrieve with `music-tools takes list --job <job-id>`. Compare the actual audio before selecting; see `references/audio-review.md`. The retired `music-generation` skill is only a pointer; never build a virtualenv or download weights from a skill. A bundled or library track whose terms are unresolved is a blocker to publication; the governed ACE-Step resource records model and licence provenance. Search prior delivery folders and the governed take pool before declaring audio unavailable. Preserve the operator's previous style family and variety, not just the most recent agent's selection. If generation cannot safely run, use licensed reuse or a silent timing rough; a rough is not the soundtrack deliverable. Do not replace requested music with oscillator/noise synthesis unless the operator explicitly requested an original score. See `references/audio-review.md`; report unreviewed audio. Select one take for the cut and deliver **all** candidates in `alternates/music/` with `PROVENANCE.md` (§7).
- **When a generator disappoints, find out what it actually received before blaming it.** Music and image models are commonly fronted by a planner or prompt-rewriter that restates the brief before the generating model sees it, and a small planner restates everything toward the bland middle: a brief asking for "144 BPM, cold, detuned, menacing" came back as 65 BPM "shimmering, melancholic, ethereal". Read the log for the prompt that reached the model, and turn the rewrite off when the brief is authored rather than sketched. Pass structured values — tempo, key, duration — in their own fields; a rewriter discards free text first.
- **Describe the sound, not the category.** "Modern tech product launch, polished, uplifting, confident" is the definition of stock music and will generate exactly that. Name the rhythm, the bass character, and the instruments: pattern and timbre, not mood and market. If the operator supplies reference tracks, mine them for tags rather than adjectives — a stock library's own genre/mood/movement labels are a better prompt vocabulary than anything invented from scratch.
- Do not render with HyperFrames cloud, Lambda, or Cloud Run. Render locally.
- Do not commit or publish production output unless the operator requests it. Read-only provenance inspection is allowed.

---

### 9. Output expectations

You may:
- Create the output folder and everything in §7.
- Start and stop only the product/dependency instances the safety assessment establishes as safe.
- Populate verified instance-owned surfaces through supported interfaces (§5b).
- Capture live read-only only when the task authorizes it and the safety/privacy assessment passes.

You must:
- **Deliver the alternatives, not just the winner (§7).** Every asset class you chose from ships its rejects in `alternates/`, and every choice that had one ships a `DECISIONS.md` row with its switch cost.
- **Check the delivery folder against §7 before reporting.** A missing file is invisible to the operator, who has no reason to know what should have been there — list §7 and confirm each line, or say which one is absent and why.
- Record every source and conflict in `sources.md`.
- Record in `sources.md` which instance was captured, the follow list used, and which presentation-owned, real-host, or shared-control-plane data appears in any frame.
- **Dress the demo world before capturing (§5b), and record it in `demo-world.md`.**
- **Apply the fullness gate before the render.** Read every scene's frames and ask of each: would a viewer believe real people use this product? A scene whose main surface is empty, single-item, or placeholder-named fails. Fix it, cut it, or report it as a defect in the draft — a failing scene may not pass silently into the render.
- **Apply the motion gate before the render.** Sample frames from the start, middle and end of every product scene. If the product itself did not move — only the camera did — the scene fails (§5c).
- **Apply the scene-purpose gate.** Every scene adds evidence, context, emotion, or closure to the one core promise. Revisit a surface when its state change proves the story. Cut redundant scenes; do not maximize feature coverage.
- Pass the story, claim, focus, motion-purpose, source-resolution, and final-file gates in `references/production.md`. In flagship mode, review styleframes and the animatic before full polish.
- Pass the visual-continuity review against the product's rendered landing page. Reuse a current reviewed treatment for routine teasers; record the reference and check for source changes.
- Inspect the actual exported MP4, record timestamped defects, preserve prior renders, fix major defects, and review the revised export. Missing product proof remains a delivery failure even if a safe composition study renders.
- Report to the operator, alongside the video, every surface that could not be populated and what the video shows instead.
- Treat any required live restart or unsafe shared build as a capture blocker. Record it and continue safe independent work.
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
| Variant start refused: "needs a rebuild, but it shares build outputs with running instance(s) …" | The working tree changed since the running instances built | `vrooli scenario status <scenario>` | Stop capture startup. Reuse verified current footage or report the blocker. Do not restart live or force the build. |
| Every capture shows "Loading…" | `google-chrome --headless --screenshot` fires before the app mounts | `ffprobe`/byte-size of two captures taken with different `--timeout` values — identical size means neither waited | Use puppeteer-core with `networkidle2` plus a settle delay (§5c) |
| A panel is empty or a request 503s on the presentation instance only | A follow-list dependency is not running at that variant, and discovery fails closed by design | Compare the browser console against the same capture of live | Start `<dependency>@presentation` only after its safety assessment passes; otherwise cut the dependent shot and record the gap |
| Instance log syntax is rejected | CLI version or flag placement differs | Read current `vrooli scenario logs --help` | Use the supported instance flag; if unavailable, read only the intended instance’s recorded log path and report the gap |
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
| The operator asks "can you use a different one" and it costs a whole re-run | The alternatives were never delivered, so the request has to regenerate them before it can answer | Look in `alternates/` — is the class they are asking about there at all? | Ship rejects from the start (§7). This is cheap in advance and expensive afterwards |
| A rejected clip has to be re-shot to be reconsidered | It was deleted after the cut instead of moved to `alternates/footage/` | Compare the clips captured in the session against the clips in the folder | Keep every capture (§5c); a presentation-instance recapture costs a whole session |
| An `alternates/` subfolder is empty | Either no options were generated, or they were generated and discarded | Check the session's own working directory for discarded files | Say which of the two it was in `DECISIONS.md` — an empty folder is a finding, not a clean result (§7) |
| A generation run OOMs intermittently on a shared GPU | The capacity broker is advisory on this host, so residency is first-come, first-served | `vrooli capacity policy get` — check `enforce` and `preempt_enabled` | Shrink your own footprint (CPU offload) and wait for headroom; do not stop another tenant's service without asking |

Promotion note: §2–§4 source resolution is deterministic. If this skill runs for several products, move it into a governed program that writes `sources.md`, and keep only story and scene judgement in this skill.
