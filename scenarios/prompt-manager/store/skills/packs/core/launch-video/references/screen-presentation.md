# Stage the screen as part of the film

Choose a coherent presentation treatment for product recordings. For desktop
launch films, start with a deliberate browser/app frame and composed background,
then consider a device scene where it adds context or a strong reveal. A thin
rounded border around a floating rectangle is not by itself a finished treatment.
Judge chrome, edge detail, light/shadow, spacing, typography and screen scale
together. Preserve the accepted story and native interactions when improving it.

## Choose from the shot's purpose

| Shot needs | Starting treatment | Owner and qualification |
|---|---|---|
| Sustained desktop workflow, message/file relationship or readable controls | Restrained browser/app window on the film's stage | BAS replay/export styling when that is the media path; otherwise the existing HyperFrames composition can supply simple presentation chrome around authentic footage |
| Establish physical context, a hero reveal or a stationary device in a room | Fixed photographic device scene | Device Studio, with the selected plate, screen projection and recording profile qualified; changing screen content does not require moving hardware |
| Move from a device scene into detailed product proof | Device scene followed by a source-continuous screen entry, or a motivated match cut | Device Studio's qualified camera profile for its supported motion; preserve source time, aspect and app surface at the HyperFrames handoff |
| Laptop lid opening, device orbit or changing physical occlusion/parallax | Actual articulated/physical scene | Device Studio's selected physical device/motion/backend must have a representative look frame and continuous sample; a photographic crop push does not qualify these actions |
| Place a device on the film's own background | A verified device-only output with alpha or another explicitly supported compositing contract | Check the actual owner output profile, edges, shadow/reflection policy and encoded alpha. RGBA input or a PNG extension does not prove transparent output; an ordinary H.264 MP4 does not carry the needed alpha |
| Read a dense detail after the workspace/device is established | Deliberate near-full-screen product detail, followed by a useful result hold | HyperFrames crop/reframe of the authentic source; record why removing chrome helps. Return to wider context when the relationship matters |

For Aquila's desktop offer, favor browser/app framing through grouping, messages
and file opening; use a laptop/desktop scene for a short establishing or transition
beat if qualified. Do not put desktop footage in a phone to decorate it: mobile
availability and setup remain separate offer questions. A large readable screen
can be more useful than a distant device throughout a long workflow.

## Keep one presentation owner per layer

Inspect the source first: app-only pixels, a browser window, or an OS desktop.
Avoid accidental double browser bars or framing the entire slide as a monitor.
A physical device can contain an intentional browser window, but every added
layer costs screen area. Keep authored window titles neutral; do not invent a
working address, OS integration or installed-app state.

BAS owns capture and its shared replay presentation/export styling. Its current
source includes browser themes, device frames and continuous-recording styling.
A raw capture does not acquire those settings merely because the replay preview
looks framed. Select recorded-video versus replay-frame intent explicitly,
preserve baked pointer pixels without adding another cursor, and inspect the
actual exported file on the running build. Use scoped export settings; do not
change the operator's global replay configuration for one film. See
`path:scenarios/browser-automation-studio/docs/replay-style.md` and
`path:scenarios/browser-automation-studio/docs/replay-layout.md`.

HyperFrames owns the assembled edit, background, labels, transitions and audio.
When it already owns cropped/retimed footage, simple window chrome can live in
that composition without re-exporting every clip through BAS. It is presentation
around captured pixels, not a reconstruction of the app. Reuse a suitable existing
treatment rather than building another frame catalog or rendering service.

Device Studio owns calibrated/device/physical shots. Use the existing
`device-studio.launch-video-shot` operation and verified CLI export described in
SKILL.md. It consumes existing project/media IDs; importing/inspecting media and
choosing a usable setup are explicit prerequisites. Do not replace an unavailable
owner profile with a private Blender worker in the launch skill. Current scope
and repair belong to the device owner, not another film-local job loop.

## Design the join before the device shot

Choose the next shot's actual screen rectangle, source asset and source time
before rendering a screen entry. Device Studio's photographic camera supports
an optional `camera_motion.target_rect` using integer normalized coordinates
(x/y/width/height, basis10000); omitted means full-output contain. The full active
viewport is centered and contained inside the destination. Use the computed
contained bounds, not merely the outer rectangle, in the next composition. Set
the matte to the film background. The device/environment fade belongs to the
owner; the following surrounding title/chrome can arrive after that fade.

Continue the same recording across the join, with the next source frame after
the device trim and compatible clock/rate/aspect. No opacity effect repairs a
source-time jump or different take. Compare last/first decoded frames, screen
corners, content, pointer and motion. Keep the complete viewport visible through
the join. Default endpoint behavior remains available for a full-frame handoff. Prefer a
side destination when the following shot needs large text alongside the screen.
That choice does not constrain later scenes.
This contract is opaque video plus matte, not a claim of alpha device output.

For a physical camera move, distinguish the device panel's outer bounds from
the contained recording's active bounds. Target the latter when matching a
16:9 workspace inside a taller laptop panel. Let the camera reach its endpoint
before fading the environment so both layers share a settled rectangle. An
intentional source hold can establish the device before native actions begin;
record that hold and the following action interval in the source map. Inspect
the original media before mapping it: an already composited device shot is not
the raw app recording. Keep this precision local to the join; later scenes can
change layout, scale and motion for their own story purpose.
For a mesh-backed screen, camera fitting must use the renderer's actual UV
mapping, not a nearest-vertex bounding rectangle: bevels or inset charts can
make a small analytic residual hide a visible double image. Review a decoded
endpoint over the next native frame at matching source time, including corners,
text and cursor. An image overlay or feature alignment can diagnose a mismatch;
the rendered fade still needs visual review at delivery scale.

## Select photographic motion without bending the room

| Intended motion | Route |
|---|---|
| Lift the screen into the next composed window while a room stays still | Device Studio `screen-entry-stationary-scene-v1`; set `scene_rect` and `target_rect`, keep source time continuous, review the room fade |
| Preserve a photographed room while a known device moves independently | Device Studio typed `SOURCE_CAMERA_PLATE_V1` photographic mode; retain the reviewed clean plate, estimated support plane and separate device/contact-shadow pass; keep the camera fixed and qualify a short actual device-motion sample |
| A nearby camera translation/rotation with depth separation | Image Tools prepared-layer `parallax` sequence; use filled layers at explicit distances, bounded camera values and `require_opaque`; inspect actual edge/occlusion frames |
| Physical device rotation, laptop hinge, or large orbit | Qualified Device Studio physical scene and motion; do not substitute a flat plate warp or a layered card |

A homography that straightens a photographed screen can shear or rotate the
whole room. The legacy projective-contain profile is a crop/rectification effect;
use it only when that visible change is intended. A depth map alone does not
qualify parallax. Image Tools owns depth estimation, segmentation, inpainting and
prepared-layer rendering; its lossless PNG-sequence ZIP includes frame clocks and
source hashes. Preparation remains explicit and model readiness applies. Film
assembly/encoding remains with its existing owner. See
`path:scenarios/image-tools/docs/reference/layered-photography.md`.

Choose preparation from the starting asset: known-camera geometry first for a
new controlled scene; fit/render a known device into a device-free plate; or
segment, fit, remove and replace a device already photographed. Keep device,
support plane and background separate. Use existing device geometry for laptops;
image-to-object generation is an optional prop route. A model catalog entry or
successful mask does not qualify camera recovery, pose/hinge fit, background
repair, contact shadows or a physical screen handoff. Follow the owner document's
readiness gates and inspect a short motion sample before recutting an accepted
film. Successful individual stages do not establish the full hybrid pipeline.

In a photographic source-camera composition, a fixed room and rotating device
are distinct from camera parallax. Estimated depth meshes can leave visible
holes at table edges, foliage and other discontinuities even when source UVs
are correct. Inspect the rendered background before animating; use the explicit
source-camera plate mode when a stationary photographic composition serves the
shot. This mode rejects camera drift. A different source/output aspect ratio
also needs a matching plate fit and physical-camera projection; a successful
import alone does not prove this alignment. The current source-camera mode
rejects mismatched aspects. An explicit photographic framing crop can prepare
a compatible source: apply the identical native crop to source, clean plate and
masks, retain the parents/transform, and regenerate geometry for the actual new
input before review. This does not authorize cropping a product screen capture.
Keep experimental geometry
reprojection separate from an accepted photographic plate treatment.

An authored clean plate is a supported repair input when its full prompt, exact
source/output/mask hashes and tool evidence are retained. Distinguish full-frame
generation from outside-mask exact repair and from local model inference;
do not invent an undisclosed checkpoint to satisfy provenance fields. Image
Tools' normal lossless encoding can retain an owned PNG artifact; verify decoded
pixel equality and keep the original generated output and encoding lineage.

Use `image-tools.scene-preparation-stage` for bounded native stage submission,
observation, cancellation, immutable preparation/review and optional Device
Studio import. Retain its exact operation identity after an uncertain response.
The program does not choose acceptable masks, repair quality or scene motion.
Preparation acceptance qualifies its stated source/geometry scope; it does not
qualify a rendered camera move. Generated object meshes need an actual intake
preview and material/UV inspection before becoming film assets. For a known
laptop, prefer the existing device mesh over an approximate generated object.

## Qualify the treatment before the whole film

Use actual product footage for a representative start, action and result. Review
a full-size frame and short encoded sample at the intended player size. Check:

- The message/link → sidebar preview → expansion relationship survives both
  framing and cuts. A device reveal must not replace that first meaningful setup.
- The screen has enough source pixels; UI text, pointer and controls remain clear
  after perspective, containment, compression and any final crop. Preserve aspect
  rather than stretching the app to fit a device.
- Bezels/chrome do not obscure actions. Screen corners, occlusion, reflections,
  shadows and moving joints are coherent, with no unwanted desktop/browser layer
  left on an app-only handoff.
- A camera move settles for useful reading/actions. Lid or orbit motion earns its
  time and does not compete with simultaneous native scrolling and captions.
- Surface materials remain plausible throughout the move. Check texture
  semantics and reflected light shapes when a scene looks glossy or synthetic;
  increasing samples cannot repair an unsuitable roughness map.
- Transparent outputs are tested over both light and dark film backgrounds;
  inspect alpha edges and contact shadows instead of assuming export support.

Record the treatment and reason beside the existing shot plan: source/capture
area, framing owner, exact theme or saved shot revision, fit/crop, output profile,
alpha/background policy, source-to-final time mapping, representative review and
known limits. No new plan schema or automatic taste score is needed.

## Capability checkpoint, 2026-10-22

These observations guide discovery; recheck the selected running build/profile.
BAS has a shared continuous-video presentation/export path in source; its
qualification notes distinguish source repair from managed-build proof. Device
Studio has accepted fixed photographic phone/table and laptop/browser appearance,
bounded fixed-video and screen-entry evidence. The physical owner has rendered
a four-second Aquila laptop camera move with separate scene/device geometry,
actual recorded screen pixels, a measured off-center endpoint and a settled
handoff. That qualifies this selected geometry-first device/scene/motion
combination, not arbitrary photo reconstruction or laptop articulation. SAM2
and MoGe stages have native execution evidence; photographic background repair
and generated props remain subject to their specific visual gates. A supported
render-mode name is not visual acceptance. A native source-camera photographic
static sample now preserves the desk plate and separately renders a known
device/contact shadow; a 48-frame, 128-sample native motion proof also passes for
the selected fixed-camera desk/phone combination, with its claim released.
An explicitly authored phone clean plate also passed preparation review and
native scene import. A separately retained 16:9 derivative with fresh geometry
also passes a known-phone static render using explicitly authored placement;
this does not qualify original-device pose fitting or camera parallax.
Internal transparent compositing does not establish a public
device-only alpha export. Do not advertise alpha or hinge support from these
observations. Use current owner evidence if it supersedes this checkpoint.

Sources: `path:scenarios/device-studio/docs/concepts/INTEGRATIONS.md`,
`path:scenarios/device-studio/docs/domains/rendering.md`,
`path:scenarios/device-studio/docs/reference/cli-commands.md`,
`path:scenarios/device-studio/docs/internal/PROGRESS.md`, and
`path:scenarios/browser-automation-studio/docs/PROBLEMS.md` (BAS-RF-151).
