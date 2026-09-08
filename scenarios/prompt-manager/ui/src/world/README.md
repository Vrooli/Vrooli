# world

The 3D world module. Six layers with one-way dependencies, enforced by ESLint
(`import/no-restricted-paths`, see `eslint.config.js`) and proven by the
fixtures in `sim/__lint__` through `__lint__/layerRule.test.ts`.

| Layer | May import | Owns |
|---|---|---|
| `config` | nothing (zod only) | tuning, scene records, biome sets, weather presets |
| `sim` | `config` | seeded terrain, water, biomes, sites, navigation, actors, weather and views |
| `engine` | `config` | canvas, lighting rig, post chain, camera rig, quality governor, assets, diagnostics |
| `scene` | `engine`, `sim`, `config` | R3F views: terrain tiles, water, places, biome props, actors, weather and labels |
| `hud` | `sim`, `config`, `data` | summary strip, agent card, team panel, ticker, settings, 2D mode, editor chrome |
| `data` | `sim`, `config`, generated proto clients | WorldService client, feed stream, fallback poll, actions, runtime |

`index.tsx` is the route component and the only module that composes `scene`
and `hud`. Nothing outside `src/world` imports `scene` or `hud`.

Scenes compose a base landscape with an optional floorplate-derived centre.
The park and the office exterior share water-aware terrain and vegetation stands;
the office centre supplies a level, dry floor. Prop slots own emission. One global
vegetation culler owns visibility and buffers; resting cameras reuse its result.
One framing solver serves home and focus. A focused resting actor can face the
viewer visually while its simulation heading stays authoritative.

Terrain, water and room surfaces declare `userData.cameraSurface` on their mesh
or parent group. The camera's shared wheel/touch dolly path queries visible
declared surfaces and limits inward travel using the near-plane sphere and hit
normal. Undeclared overlays and clouds do not anchor zoom; a sky miss retains
the controller's focal-plane behavior. `cameraZoomGuard` diagnostics report the
last queried travel limit and query cost. Room slabs also declare
`geometry.userData.cameraObstacle = 'box'`, preserving Drei's instance metadata.
The camera sweeps its near-plane sphere
against those rendered box transforms for all translation, including orbit,
pan and mixed input. Local expanded boxes conservatively protect rotated and
scaled instances; rounded corners may stop slightly early. Contact constrains
translation while preserving rotation, and an existing overlap may escape
through its nearest face. When a new layout encloses the previous eye, recovery
lifts it to the first clear vertical gap across all overlapping boxes, after
terrain clearance. Eye and target translate together; clear views stay fixed.
`cameraObstacleGuard` reports box count, accepted motion fraction, query cost
and any recovery lift. Terrain retains its separate height-field sweep.
Collision queries update only eligible meshes and their ancestors. A world-space
query box excludes distant obstacles before inverse transforms and local hull
checks, keeping repeated walking and third-person boom sweeps within the measured
frame budget even in the 400-agent office stress cases.
Target bounds can remove part of cursor zoom's sideways translation and bend
the eye's path toward an angled surface. The guard checks each linear segment
between target-axis clamp points, rather than assuming the eye follows the
cursor ray throughout. Controller regressions cover a target already at its
boundary and a continuous gesture reaching multiple bounds during motion.

Focus, Home and intro commands plan straight eye segments through the same guards. A clear
direct path wins; otherwise the planner compares an overhead route with side
routes around the obstructing bounds. One cancelable route writer runs before
the controller clearance update, and follow translation waits until it finishes.
Completion requires the actual guarded pose to reach the destination. Changed
geometry can trigger a bounded replan; an unavailable route reports `blocked`
in `cameraPoseRoute` instead of claiming successful framing. The diagnostics
include the command owner and planning cost. The configured transition duration
applies to the whole route, including detours. Reduced motion traverses
validated segments without displaying intermediate frames.

The camera browser replay accepts `--obstruction-fixtures` to insert temporary
instanced boxes into the rendered park and office scenes. It verifies Home
detours, frame and inter-frame near-plane clearance, and stationary overlap
recovery with unchanged view direction. Fixtures borrow existing geometry via a
browser-injected React inspection hook; no application test API or capture mode
is involved. This proves rendered obstruction handling, not the complete world
regeneration path or physical-device input behavior.

Hardware smoke captures record live costs and invariants before producing a
fixed-time world-canvas golden. Full-page screenshots remain separate evidence.
Calibration proposes budgets from a fresh, complete hardware batch; applying an
increase requires an explicit cost explanation.

World Settings exposes the world seed in normal builds. Apply seed (or Enter)
accepts an unsigned 32-bit integer and updates the page URL before preparation.
Typing and blur do not regenerate the world. The URL retains the seed across
reloads; enter 1 to restore the shipped seed. This is independent of saved
global preferences and the session-only tuning overrides below.

Weather choices also live in the page URL. Clear, Cloudy, Rain and Snow override
the simulated presentation; Automatic removes both weather and diagnostic
pressure overrides and resumes activity-driven weather. These choices update
the environment without regenerating the world. The normal settings browser
replay is `node scripts/world-smoke/settings.mjs <fresh-evidence-directory>`.

The built UI exposes session tuning through `?workbench=1` → World Settings →
World workbench. Development builds expose it by default. Numeric edits commit
on Enter or blur and retain keyboard focus; invalid drafts show an accessible
error. Reset restores defaults and clears drafts. Reload clears all workbench
overrides; these controls do not save global world preferences.
While expanded, the workbench refreshes preparation stage outcomes, generation
source/reason, and cache residency twice per second. Generated worlds, terrain
meshes and prepared geometry show separate byte budgets and hit/miss/eviction
counts. Closing the workbench stops this refresh. Shared buffers mean these
estimates cannot be added to obtain total process or GPU memory.
The navigation overlay renders every committed grid cell as a terrain-aligned
marker: green is walkable and red is blocked. Markers show through structures
and do not participate in picking or camera clearance. Preparation uses the
cooperative scheduler and cancels on replacement or disable; buffers and GPU
geometry are released when no longer displayed. The workbench reports cell
counts and buffer bytes. Recipe export/import preserves the overlay choice.
The biome overlay uses the same cancellable marker renderer but samples the
committed terrain lattice and stored biome IDs. Its legend shows each biome's
diagnostic color and the status lists assignment counts. It does not rerun
classification. The habitat overlay reads the generated habitat field on the
same lattice, shows excluded samples in gray, and reports each class's count.
Navigation, biome and habitat overlays are mutually exclusive; recipes preserve
the choice and older recipes default newer overlay flags to off.
The camera collision overlay outlines visible structural boxes using the same
transforms and near-plane expansion as the camera sweep. It refreshes after
commits and lens/aspect changes, shows the box count and clearance radius, and
releases its line geometry when disabled. Its recipe flag defaults to off for
older exports. These outlines cover structural camera guards; terrain clearance
and measured furniture interaction anchors require separate diagnostics.

Generated worlds include a byte-per-terrain-sample habitat field. Stable IDs
describe none, meadow, woodland, rocky slope, wetland and open water. Habitat
comes from committed biome assignments; samples outside the terrain disc, on
painted paths or inside built footprints are excluded. This environmental mask
does not replace navigation eligibility. Generation owns and transfers the field
with its other immutable buffers. Four-neighbor connected regions provide
deterministic root-sample IDs, packed member indices, and world-space bounds.
Diagonal contact does not join regions. Consumers must select actual members;
bounds alone do not establish habitat eligibility. Generation rejects inputs
exceeding 32,768 regions rather than truncating or merging components. Region
labels and membership buffers participate in worker transfer and cache accounting.
The generator version is now `living-world-v4`; recipes from an older generator
are rejected explicitly. Rabbits and squirrels use the bounded habitat pools;
their routes, canopy clearance and trunk attachment have browser geometry checks.

The workbench can download a ready synthetic world's recipe. The allowlisted
format includes its synthetic roster descriptor, effective tuning, layout edits,
tree-clearance points, resolved view, canvas dimensions and renderer name. It
does not serialize live rosters, messages, event feeds or request payloads.
`data/diagnosticRecipe.ts` validates imported JSON and restores deterministic
generation inputs with a fixed clock. Generator version and environment
definition fingerprints guard compatibility; they are not a build-source hash.
Built recipes additionally include SHA-256 fingerprints for UI runtime source
and Vite configuration, package/lockfile inputs, and every bundled world asset.
Each asset record carries its relative path, byte count and content hash. The
manifest states its source and dependency scopes; it does not fingerprint
external workspace package source or claim to hash the final JavaScript chunks.
Non-Vite test consumers report build provenance as unavailable (`null`).
Import synthetic recipe validates a JSON file (at most 1 MiB) before opening a
new synthetic presentation. The recipe is stored in this tab's session storage
and survives reloads; its URL marker alone cannot transfer it to another tab.
Import restores the roster grouping, tuning, layout, clearance points and
resolved view. The camera approaches the saved destination through normal
clearance guards and respects reduced motion. Saved live-world preferences are
not read or written for the imported synthetic presentation. Live-world
anonymization remains pending.

`sim/ambient/schedule.ts` supplies stateless absolute-time sky event identities.
Meteor opportunities use independent seeded buckets, weighted colors and bounded
durations. Great fireballs observe a conservative 24-hour world-wide cooldown:
a preceding rare candidate suppresses the next even if that candidate was itself
suppressed. Comets retain one identity throughout a three-to-five-day window in
a 30-day opportunity. Eligibility gates presentation without consuming identities;
reduced motion retains static comets and suppresses meteors. Direct time seeks
return only currently active events, with no missed-event replay. Fixed-sample
tests cover rates, variants, cooldowns, multi-day identity and 15–144 Hz sampling.
`scene/ambient/skyPresentation.ts` connects schedules to fixed meteor/comet slot
pools and one scene-owned motion lease. Slots retain admitted identities, expire
without backlog, and release on disable or disposal. The generic pool rejects
oversized/malformed input before mutation and reports active admission rejections
without accumulating a frame-rate-dependent count. `FrameDriver` supports
independent animation leases and requests a cleanup frame on final release.
`SkyEvents` now renders two reusable meteor quads and one static comet quad in
the world. Procedural materials provide colored heads, tapered tails, a fading
wake and fireball fragments. Scheduled directions are seeded above the horizon;
the workbench's forced previews use the same materials in front of the camera
without terrain occlusion. Preview expiry restores ordinary scheduling. Ambient
life disables all sky slots; reduced motion suppresses meteors and retains static
comets. Ambient life persists through WorldService for live worlds; its optional
wire field preserves the enabled default for older configurations and explicit
false values. Synthetic recipes carry the same choice, with older recipes
defaulting to enabled, and do not write live preferences. One scene timer wakes at
the next event start, end or comet opportunity; it
pauses while hidden and seeks current state on resume. Long waits are split at
the browser timer limit without rendering early. Moving events use FrameDriver
leases. Remaining world preferences, celestial comet
orientation, wildlife, and final visual/performance qualification remain pending.

`config/clock.ts` owns presentation UTC, civil minutes, mode and time scale.
Lighting and sky events share this clock; agent/feed time remains independent.
Live mode reads current wall time, fixed mode freezes an instant, and controlled
progression supports deterministic tests. Civil minutes use the current browser
timezone unless the recipe pins an explicit civil timezone. Snapshots include
the resolved timezone; live sampling and focus refresh the browser-zone identity
without constructing an Intl formatter per rendered frame. Tests cover leap day
and a DST jump.
The workbench can freeze, seek a UTC instant, advance one minute and resume live
time. Frozen sky effects retain their current pose without animation leases or
boundary timers. Recipe export pins the current UTC instant and civil timezone;
import and reload restore both. A cross-timezone browser test verifies identical
rendered lighting in UTC and America/New_York. Unknown zones are rejected before
import. Older recipes without a timezone use the importing browser's zone;
older recipes without a clock retain live behavior. Astronomical
mode and full capture-tool
adoption of the shared clock remain pending.

Live lighting uses cyclic smooth interpolation between the centres of the four
civil-time bands. Each anchor applies the scene's overrides before blending;
numeric properties interpolate directly and colors interpolate in linear light.
The shared clock is sampled once per second by default. Named period controls
remain exact presets. Recipes distinguish clock-based lighting from preset
lighting, retaining the same rendered values after a pinned-time import/reload.
Tests cover every anchor, former band boundaries and midnight; browser checks
inspect the renderer's exposure and directional key intensity/color directly.

`CelestialSky` draws a bounded sun disc and halo independently from the readable
directional key light. Its civil-time orbit is explicitly stylized, with preset
elevations for named controls. Weather attenuates the disc; it disappears below
the horizon. The camera-centred quad writes background clip depth, so finite far
planes cannot remove it while opaque world geometry still occludes it. One
geometry/material is shared across frames and disposed on unmount. Low/High GPU
fixtures verify visibility, short-far-plane behavior and foreground occlusion;
final sky composition and celestial art qualification remain pending.

`lunarPhase` in `config/celestial.ts` computes UTC-based geocentric phase,
illumination and waxing/waning from orbital elements with lunar perturbations.
The mathematical reference is [Paul Schlyter's position calculations](https://stjarnhimlen.se/comp/ppcomp.html).
The checked-in reference data comes from the [USNO 2026 phase service](https://aa.usno.navy.mil/api/moon/phases/year?year=2026).
All 50 phase instants pass a predeclared 0.2-degree longitude tolerance. This
evidence qualifies those dates, not observer-position accuracy, parallax,
libration or eclipses.

The moon presenter uses the validated phase and latitude to place a body relative
to the stylized sun. Its sphere-normal shader derives the terminator from the
sun direction transformed into the moon's local frame; a bounded procedural
surface supplies crater-like relief. It shares the sun's quad geometry and
background-depth convention. Lunar geometry is cached per live minute, with
exact recomputation for frozen instants. GPU checks on Low and High cover new,
crescent, first quarter, gibbous, full and last quarter illuminated fractions.
Screenshots demonstrate the phases; final lunar surface art and combined
celestial qualification remain pending.

`starField.ts` creates one fictional seeded field of 2,048 directions and
brightness values in 32 KiB of attribute buffers. Low/Medium/High/Ultra draw
stable prefixes of 384/768/1,536/2,048 points without reallocating geometry.
The camera-centred field renders at background depth and fades at the horizon,
in daylight, under clouds and with above-horizon moon illumination. It has no
twinkle timer or motion lease; reduced motion retains it. Browser checks verify
profile reuse, visibility controls and unchanged world generation. This is a
stylized field, not a star catalogue or observer-specific constellation model.

Firefly sites use the committed connected wetland index. Only orthogonal
interfaces with another eligible habitat qualify; excluded paths/buildings and
diagonal neighbors do not create edges. Cooperative selection retains at most
48 sites ranked by a separate seeded hash. Low/Medium/High/Ultra present stable
prefixes of 12/24/48/48 without changing world generation. A single procedural
point draw uses 768 bytes of attributes. Absolute-clock vertical motion keeps
each light on its habitat sample; reduced motion retains static lights. Night,
clear weather and the ambient-life toggle control visibility. Moving lights own
one demand-render lease, released on freeze, suppression or unmount. Selection
cancels on habitat replacement and never writes agent navigation or occupancy.
Close-up browser evidence verifies visible lights and profile reuse. The motion
gallery below includes natural-density clips; combined maximum-load
qualification remains open.

Butterflies anchor to ground plants or shrubs whose surrounding 3×3 habitat
samples are all meadow. The cooperative selector deduplicates terrain samples,
uses a separate seeded ranking, and retains at most 16 sites. Quality presents
stable 4/8/16/16 prefixes. Small figure-eight loops stay within 0.3 terrain cells
of the anchor and sample actual ground height; full-loop tests cover generated
Park and Office building exclusion. A single instanced draw uses original
procedural wings, body and antennae with seeded orange/blue patterns and flap
phases. Geometry and animation were authored here without external assets.
The shared clock freezes both travel and flapping exactly. Reduced motion,
nighttime, non-clear weather and disabled ambient life suppress butterflies and
release their animation lease. The default seed's browser evidence contains one
eligible site; dense selection limits are tested separately. Final wildlife art
and natural-density clips remain unqualified.

Bird routes use a cooperative summed habitat mask to reject any orbit bounding
square containing excluded, wetland or open-water samples. A bounded list of
64 ranked candidates yields up to four separated routes, with quality prefixes
of 1/2/4/4. Circular paths cruise 18 metres above the highest terrain sample,
with small altitude variation and independent seeded flap/glide phases. This
height policy is terrain-relative; arbitrary asset-canopy clearance has not
been qualified. Birds and butterflies share the instanced presenter lifecycle,
while retaining separate procedural silhouettes, routes and animation leases.
Bird geometry and motion are original code, with no external asset source.
Birds appear in clear daylight/twilight and are suppressed by night, reduced
motion or ambient-off. Browser checks cover four-to-one quality reduction,
buffer reuse, freeze and visibility; generated Park/Office route tests cover
habitat and projected building exclusions. Combined natural-density clips and
final wildlife art remain pending.

Rabbits use short axis-aligned routes whose complete swept footprint passes
both navigation and meadow/woodland habitat checks. Selection scans navigation
cooperatively, retains 128 ranked candidates and publishes at most two separated
routes; quality presents 1/1/2/2 animals. Each animal alternates 18 seconds idle
with four seconds of smooth travel and small hops, replayed from absolute time.
Idle rabbits have no animation lease. The shared boundary-wake helper starts
the next travel window, including after tab resume; frozen time retains pose.
Ambient-off, non-clear weather and reduced motion suppress rabbits. One instanced
draw pools 13 ellipsoid parts per animal using original geometry composition
and colors. The 0.38-metre route footprint encloses the model. Tests sample full
Park/Office routes and footprint corners against navigation, habitat and built
regions; browser evidence verifies movement, idle lease release and scheduled
restart. This decorative behavior does not reserve seats or modify agent paths.
Natural-density clips, slope-contact polish and final animal art remain open.

Fish opportunities use actual connected water-mesh area, with a 12 m² minimum;
separate small ponds are never combined. Cooperative selection retains at most
64 eligible ponds and eight sites per pond. Each site's projected in-circle is
inside a rendered water triangle and passes the open-water habitat mask, keeping
jumps, splash particles and expanding ripples away from protected ground and
clipped banks. One seeded opportunity per 90 seconds selects a pond/site without
consuming agent RNG; seeking returns only currently alive events. Two reusable
presentation slots show a short jump followed by splash and ripple, with quality
admission limits 1/1/2/2. Geometry, colors and motion are original procedural
work. Effects draw after the transparent water pass while retaining depth tests.
Shared-clock freeze, ambient-off and reduced-motion behavior match other life;
event boundaries own wake timers and live effects own one animation lease.
The conservative triangle footprint limits ripple size. Final fish art,
cross-region water-height polish and natural-density clips remain pending.

Moisture basins now lower the generated height field itself, with the terrain
edge falloff, and water classification samples that same physical height.
Previously a separate classification-only depression could put pond geometry
below visible ground and hide ripples. Generator v4 invalidates older generated
recipes explicitly. Targeted terrain/navigation/wildlife tests and an inspected
visible ripple capture cover the repair.

Workbench wildlife previews cover fireflies, butterflies, birds, rabbit idle and
travel, and fish jump/splash/ripple. They select committed eligible habitat and
seek the ordinary deterministic schedule; no preview-only animals are injected.
The UI freezes the shared clock, selects clear weather and an appropriate named
lighting preset, then focuses through the normal collision-aware camera rig.
Ambient-off and reduced motion explain suppression. Missing habitat is reported;
new requests and context changes cancel pending selection. The current preview
can be exported using the existing synthetic recipe: browser import/reload
checks preserve the exact instant, ripple phase/site and focused camera. Eight
normal-UI captures show all preview modes without fixture camera movement.

`Play from here` advances a frozen preview at real speed without jumping to the
wall clock. Freeze holds the current playback instant; Resume live time returns
to wall time. The workbench distinguishes all three states. The
[motion gallery](../../evidence/living-world/wildlife-gallery-20260905.html)
contains ordinary Park and Office day (45 s each) and night (20 s each)
recordings, Park close-up playback of all five wildlife families, and Office
fish playback. Decoded frames were inspected. Its
manifest records clip hashes, encoded timestamps and served JavaScript hashes.
These are canvas recordings at unchanged density, not physical-device timing
qualification. All eight Office preview modes also have normal-camera browser
checks and inspected captures. Combined worst-case measurements and final art
remain pending; ordinary-density evidence now covers both scene types.

`window.__worldDiagnostics.ambientCosts()` reports CPU callback timing and
active/capacity/profile-limit counts for sky, fireflies, butterflies, birds,
rabbits and fish. Each family retains 256 timings in a fixed buffer (12 KiB
total); recording does not allocate history, and inspection computes recent
mean/p95/max. `resetAmbientCosts()` starts a fresh observation window.
Unmounted families clear current counts and timing while retaining history.
These timings exclude generation, draw submission and GPU work. The summed
latest callbacks describe the latest observation, not a combined p95 or total
frame cost. Normal seed-1 day/night observations in both scenes exercise the
accounting; they do not qualify forced maximum populations or physical devices.

Synthetic capacity observations now cover High day/night in Park and Office
with 25 synthetic actors. Isolated browser fixtures fill existing event pools
and duplicate eligible butterfly anchors to reach 16 instances with varied
phases. Night uses 48 fireflies, two rabbits, two fish effects and three sky
events; day uses 16 butterflies, four birds, two rabbits and two fish effects.
The normal application does not install these fixture overrides. Nine overview
checks and seven Park pond-camera checks pass; both fish effects submit their
20 combined draws in the pond view. Recorded total scene draws range from
90 to 134 for these poses, with coarse sampled callback-total p95 of 0.2–0.3 ms.
These are fixed-capacity observations on integrated ANGLE, not natural-density
clips or comprehensive worst-case qualification. Storm overlap, moving-camera
coverage and physical-device measurements remain open. Reproduce with
`node scripts/world-smoke/ambient-capacity.mjs` (add `--pond` for the pond pose).

Normal UI weather checks now cover Cloudy/Rain/Snow suppression and Clear
restoration of the exact frozen fish event in both scenes. Suppression empties
all ambient pools and releases their leases. An explicit frozen fireball preview
remains available in Rain while normal pointer input moves the camera. Twenty-one
browser checks pass; captured frame/GPU observations are included as raw evidence.
The camera starts at a fish close-up, so these views do not establish whole-scene
worst-case cost. Reproduce with `node scripts/world-smoke/ambient-weather.mjs`.

The asset-source manifest now records original procedural geometry and animation
provenance for all six ambient families. A separate source fingerprint report
identifies the implementation used for this review. Normal UI regeneration
checks start with an active fish ripple, apply seeds 2/3/1 in both scenes, and
verify that all five wildlife presenters select the committed habitat and stay
within pool limits. This checks settled regeneration; it does not prove every
possible cancellation interleaving.

Integer setting descriptors now own seed and synthetic actor-count bounds,
defaults, descriptions and declared impact. URL parsing and the seed control use
the same decimal-digit-only parser. Malformed, unsafe or out-of-range values
fall back with an explicit error; applying a valid seed clears it. The generated
configuration reference consumes these descriptors. Scene, quality, time and weather choice descriptors also drive public controls,
strict URL vocabulary and generated documentation. Their IDs come from the
existing runtime schemas; scene/quality/period guards use those same canonical
lists. Invalid choices report the declared fallback. An explicit Clock URL
takes precedence over saved named lighting preferences. Remaining numeric
controls and development tuning still need metadata and impact-dispatch
unification.

Diagnostic DPR, multisampling, lamp-light count and pressure overrides now use
strict decimal validation with visible fallback errors. Rendering bounds reuse
the quality schema; omitted or invalid values retain the active profile, while
pressure returns to Automatic. Fractions are accepted only for DPR and pressure.
The generated reference includes their bounds, units and default sources. Built
browser checks inspect effective renderer profiles, including preserved defaults
and valid fractional pressure. Impact dispatch remains separate unfinished work.

Layout surface fields now have an exhaustive material/geometry impact map. The
generation key includes only generation-owned surface values; renderer-only
edits retain terrain, navigation and room placement. Workbench rows and the
generated reference show these declared impacts. Browser checks change actual
floor roughness and commons segments while retaining generated buffers, then
change room width and observe exactly one regeneration. Other setting families
still need complete impact classification and dispatch coverage.

Slab construction now depends only on the geometry fields it reads. Material
edits reuse slab descriptions and memoized instance elements; commons segment
changes also leave room/corridor slabs intact. A floor-thickness edit rebuilds
slab inputs without regenerating the world. The surface-impact browser script
checks these reference identities and still verifies structural regeneration.
This does not claim that Drei stops its normal per-frame instance processing.

Live tuning replacement preserves simulation state and the cached HUD view when
its equipment-tier inputs are unchanged. The view selector accepts only those
tuning fields it reads. Changed tiers publish one revision without replacing
actors or navigation; equivalent tier arrays do not republish. Targeted tests
also preserve pending run signals and accumulated time across a tick-duration
edit. The full tuning remains available to subsequent simulation ticks.

Workbench controls now include schema-declared string enums and exclude fields
marked read-only by schema metadata. AO uses its authoritative `aoQuality`
selector; the normalized compatibility `ao` flag is no longer an ineffective
checkbox. Browser checks inspect actual AO pass mount/unmount through all three
choices and verify no world regeneration. The transparency policy now sets the
installed N8AO pass’s top-level `autoDetectTransparency` property and explicitly
disables `configuration.transparencyAware`; the former configuration-only write
did not disable scanning. The checks verify both properties after mounting.

Validated tuning updates structurally share unchanged branches with the previous
applied configuration. The workbench commits override and resolved tuning
together through a functional state update; resets still resolve against shipped
values. Repeated equivalent commits reuse the resolved root. Runtime tests verify
that presentation edits keep the live feed attached, while changed data tuning
replaces its handle once and unmount cleans up. Browser checks retain data-tuning
identity through material, geometry and structural edits plus Reset.

Slime body colors now respond to live roster presentation changes. Each actor
retains its last uploaded color; changed values update the existing instanced
attribute and mark it dirty once. Geometry and generated world buffers remain
intact, and unchanged frames do not request another color upload. Browser checks
change and restore a synthetic actor’s color at the store presentation boundary
and inspect attribute values, upload versions and resource identity.

Slime presenters retain one physical material and update surface properties and
wobble uniforms in place. Unrelated actor tuning no longer replaces material or
mesh resources. Uniform handles and animation time remain intact. Three.js
physical-material setters retain shader-feature invalidation when clearcoat or
sheen crosses zero; ordinary scalar edits do not force recompilation. Browser
checks exercise roughness, wobble speed, breathing and profile toggles while
retaining the actual material, geometry, mesh and uniform objects.

Seeded slime animation offsets update their existing instance attribute. Only
sphere resolution changes rebuild body geometry; new geometry receives the
current offsets, including zero. Unchanged frames do not re-upload offsets.

Diagnostic GPU collectors keep their lifetime across diagnostic tuning edits.
Sample-window changes trim recent history in place; pending queries remain valid
and lower in-flight limits pause admission until they drain. Publication and
overlay cadence edits do not discard GPU timing history or reinstall collectors.

Shadow-resolution edits release obsolete shadow targets and request a fresh
shadow render. The directional light stays mounted; Three allocates the new
target at the requested size on the next render. Unchanged dimensions preserve
the existing target. Browser checks verify target dimensions and single disposal
of each replaced target.

Every label tuning leaf declares a shared impact: surface edits update materials,
size/pool edits update geometry, and scheduling/placement/selection edits update
the existing pipeline. Generated recipe tests cover each label setting and prove
that these edits retain terrain, layout and dressing identities.

Label budgets are strict, including zero: either a zero label budget or a zero
profile budget removes the text-mesh pool. Focused and hovered labels receive
priority within the budget. Restoring a positive budget recreates a bounded pool
without regenerating the world.

Editable camera settings have exhaustive live-impact declarations shared with
the workbench and generated docs. Initial position and intro duration are
initialization-only values. The unused minimum-clearance control was removed;
the smoke acceptance check independently retains its two-metre threshold.
Generated tests cover recipe stability for every editable camera setting.

The camera rig applies field-of-view and near/far clip changes to the existing
lens and refreshes its projection matrix only when those values change. Lens
edits retain camera controls and generated world state without issuing a new
pose command. Canvas options establish the initial camera only.

The operator zoom target and the workbench dolly-to-cursor checkbox share the
active camera tuning value. Either control updates both displays and the camera
controls. Reset restores the shipped zoom default; saved operator preferences
and imported recipe zoom targets enter through the same tuning value.

Simulation impact declarations cover every schema leaf. Committed cache-capacity
edits resize the existing LRU and retain its most recently used paths. Event
history shrinks immediately to the newest entries and publishes one view update,
including when equipment tiers change in the same edit. Pending signals, tick
carry, actor state and generated world buffers remain intact.

Layout impact declarations cover every schema leaf and determine generation
inputs. Wall height and lamp placement changes rebuild renderer instances while
retaining terrain, navigation and room placement. Structural layout edits still
regenerate. Office browser checks verify wall height and corridor lamp count,
positions and scale, alongside buffer retention and reset.

Actor generation inputs are projected from the shared impact declarations.
Generated tests vary every actor schema leaf: only body clearance changes the
layout recipe, and terrain identity remains stable for all actor edits.

Actor setting impacts are exhaustive against the actor tuning type and shared
by workbench labels and generated documentation. Shadow placement and size edits
retain their mesh, material and falloff texture. Colour and opacity update the
existing material; falloff resolution or gradient changes replace and dispose
only the procedural texture. Other setting groups and the central committed-edit
dispatcher still require completion.

Architecture: `../../../docs/concepts/WORLD-ARCHITECTURE.md`. Levers:
`../../../docs/reference/configuration.md` (world tuning section). Sim rules:
`../../../docs/concepts/WORLD-SIM.md`. HUD: `../../../docs/concepts/WORLD-HUD.md`.
Assets: `../../../docs/guides/WORLD-ASSETS.md`.
Terrain: `../../../docs/concepts/WORLD-TERRAIN.md`. Weather:
`../../../docs/concepts/WORLD-WEATHER.md`.
