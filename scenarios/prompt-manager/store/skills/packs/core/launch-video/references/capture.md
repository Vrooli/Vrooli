# Capture for the edit

Choose the viewport, framing, pointer and focal regions in the shot plan. Select
the presentation treatment using `screen-presentation.md`: browser/app window,
qualified device scene or deliberate direct detail, with an explicit source-to-
detail handoff. Keep clean source footage so framing can change without recapture.
Use the
actual responsive frontend for mobile footage; shrinking a desktop capture is
not a mobile demonstration. Prefer a clean browser shell for full windows and a
thin device rim for phone content. Check that shells leave controls, charts,
tooltips and captions visible. Avoid decorative camera islands over app content.
Compare a full-size representative frame before recording the sequence.

Build a coherent fictional team/project through verified presentation-owned
interfaces. Write credible histories, decisions, names and varied messages. Show
supported rich renderers with useful tables, diagrams, code and checklists.
Exercise workflows that matter to the promise: grouping/collapse, customization,
sort/filter, search and export are options, not a compulsory feature checklist.
Record the initial state, actual action, observed result and claim scope. Inspect
visible item previews as well as titles: distinct titles over repeated generic
histories still look staged. Give each visible work item a useful, distinct next
step through safe presentation-owned data; do not paint replacement product text.

Warm fonts and lazy renderers before the take. Return to the known start state.
Wait for the meaningful content to settle. Capture generous handles before and
after actions; retain raw and rejected takes with reasons. Use an eased scroll
path with anticipation, action, settle and reading holds. Capture more useful
material than the cut needs, then compress waiting in the edit. Check the first
and last frames after trimming; representative middle frames miss loading flashes.

## Direct a sequence, then edit the recorded proof

Give each selected benefit a start, several useful actions and a readable payoff.
Continue beyond navigation when the next action reveals the capability. Opening
an HTML preview and expanding it is a setup beat; scrolling to a useful section
can show why the expanded view matters. Record enough source material to choose
that progression in the edit. A successful click is not the stopping condition
for a craft pass.

Use one sequence for related proof. For a file viewer, qualify the supported
formats on the target build, then plan a concentrated inspection montage. Music,
video,3D and slide decks are candidates, not an availability checklist. Show a
useful action in each selected renderer: play/seek audio or video, rotate a model,
change a slide, scroll a document, inspect cells or zoom an image. Use only the
controls the actual build exposes. Cut unsupported formats. Do not imply editing
or execution from a preview. Group the proven variety under one customer benefit
so the saved screen time can show another benefit.

Establish the output's origin before compressing repeated navigation. For example,
show the file link in an agent message, its native sidebar preview and expansion
into the inspection view. Preserve enough framing and time to understand that
connection, then cut faster between related formats. Starting every shot inside
an isolated viewer can hide the integration that makes the feature useful. The
first reveal and later repetitive travel have different editorial value.

For a reading sequence, decide whether the viewer should read specific words or
recognize the richness and breadth of a conversation. Capture enough messages,
sections or native reply navigation for that purpose. Compress travel between
distinct answers, tables, diagrams or decisions, then settle on the useful detail.
A slow scroll over a few lines does not demonstrate breadth; accelerating an
insufficient take does not reveal content that was never captured.

For session management, test a visible transformation: move related sessions into
a group, change a label/color/icon, collapse the group and reveal the resulting
organization or attention state. Choose the actions the product and story support.
An unread/sidebar screenshot may establish the start or result; inspect whether
it is carrying a workflow that the film has not demonstrated.

Record the action/result coverage and editorial decision beside the existing
shot plan. Use this compact table; several rows may feed one final scene:

| Benefit/sequence | Native actions and visible payoff | Sources | Edit and final reading hold | Craft disposition |
|---|---|---|---|---|
| Inspect outputs | Open HTML → expand → scroll to the decision; cut to verified media/model/deck interactions | Take IDs, hashes and raw intervals for each renderer | One montage; accelerate navigation, settle on each distinct payoff | Captured, recapture, or intentional hold with reason |
| Organize active work | Group → customize → collapse/reopen → show the useful attention state | Take IDs, hashes and raw intervals | Match-cut pointer/focal positions; hold the organized result | Captured, recapture, or intentional hold with reason |

Edit the real footage through the existing HyperFrames composition. Select useful
raw ranges first. Remove dead navigation time. Use piecewise retiming or speed
ramps for repetitive travel, then return to a readable pace at the result. Combine
short inserts with match cuts, directed crops and eased camera moves. Tie cuts and
accents to the score when that serves the sequence. Preserve natural direction,
spatial continuity and one focal point per beat. Do not stack constant zoom,
scroll, cursor movement and new captions at the same instant.

Retain source IDs/hashes, raw intervals, speed changes, omitted waits and the
final timeline mapping. Do not let a compressed edit imply unverified response
speed or simultaneous actions. Put a timing qualification on screen when the edit
would otherwise make that claim. Keep the exact recorded result legible at the
smallest intended player size. Inspect seek/cut boundaries in the rendered movie;
a smooth composition preview does not prove extracted-video timing.

A still or hold can supply a useful comparison or reading beat. In each craft
pass, list static product intervals and sequences that stop at opening/expansion.
For each, record the stronger native action attempted, or the specific viewer
reason for retaining the hold. A pan over a still does not close an omitted
workflow finding. Recapture before adding effects when source coverage is weak.

Choose a pointer deliberately for this product's direction. Retain observed
positions/timestamps and indicate whether it is baked in or added during replay.
Do not add a second pointer or fabricate an observed gesture. Retain a clean
source where supported so another pointer can be chosen later. Inspect style
scale and contrast on both light and dark UI; a default white arrow is not an
intentional campaign choice. Cursor-free shots can be valid when no action needs
pointing. Choose a locked camera when a pan would compete with the real pointer.

Record these facts per source:

| Fact | Evidence |
|---|---|
| Provenance | Product, build/bundle hash, date, instance/fixture mode and proof scope |
| Pixel geometry | CSS viewport, requested DPR, actual encoded raster, element bounds |
| Edit | Raw duration, trim interval, retime segments, result holds, crop and peak output footprint |
| Style | Browser/device frame, pointer preset/scale, baked versus replayed pointer |
| Safety | Privacy review, fixture schema check, deny-unhandled-egress report where applicable |
| Outcome | Actual interactions/results, browser errors, rejected-take reason |

Measure focus bounds in the same layout that was recorded. Require source pixels
for the maximum displayed crop on both axes. A CDP screencast can ignore requested
DPR; probe the encoded file. Do not infer raster resolution from browser settings.
Use closer capture when pixels or text legibility are insufficient.


Aquila's maintained seed is `scenarios/web-console/bas/seeds/seed.sh`; its usage
and cleanup contract are in `scenarios/web-console/docs/reference/configuration.md`.
Prefer a native isolated API/UI with explicit target-owned authentication. The
seed publishes fictional messages through the actual hooks and keeps a receipt;
it does not run an agent. Avoid copying the prior film's private response server.

BAS `capture --interaction-flow-file take.json` sends a bounded action graph to
its existing CaptureService. Use native wait/assert actions to prove start,
action and result. Retain separate execution IDs for separate takes; the native
capture owner already writes execution-specific subdirectories. Request video,
screenshot, console and network together when useful, and retain their returned
references. Device scale may increase screenshot pixels while continuous video
remains at CSS resolution. Video metadata distinguishes requested geometry from
`measured_width`, `measured_height`, `measured_duration_ms` and
`measurement_status`. An unavailable measurement requires a local probe, never
an inferred width × DPR. The device-scale live stream is SDK polling and has a
different cadence from continuous Playwright video; select by the needed evidence.

Prefer BAS's current capture/export contracts. Verify the output rather than
assuming preview parity, a saved style or a completed status proves it. Inspect
actual raster, cursor trajectory, first/action/result frames and timeline end.
A recorded continuous video and a screenshot replay are different sources. State
which was used. Use a composition wrapper only for an observed owner limitation;
retain the raw capture and transformation, and retire the workaround after the
native path is verified. Do not widen a timeout or truncate the timeline to make
an export look successful.

For seek-heavy captured video, inspect the actual keyframe spacing before a full
render. When a short-GOP derivative is useful, retain its original input hash,
exact encoder arguments, decoded duration and measured keyframe gaps; re-encoding
is a documented edit, not a substitute for repairing owner resource failures.

For a routed test target, keep its paired SQL/file lease alive and use BAS
`--routed-test-mode` so browser target calls retain the test header. Producers
must receive the same lease ID. Verify exact fixture rows in primary and leased
databases in addition to lease counters; a discarded request context can evade
counter-based detection. Delete owned fixtures through the same routed target
before clearing the lease. A successful native capture is evidence for that
build and fixture scope, not a desktop package or real-agent execution claim.
