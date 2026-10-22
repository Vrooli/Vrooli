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
- Transparent outputs are tested over both light and dark film backgrounds;
  inspect alpha edges and contact shadows instead of assuming export support.

Record the treatment and reason beside the existing shot plan: source/capture
area, framing owner, exact theme or saved shot revision, fit/crop, output profile,
alpha/background policy, source-to-final time mapping, representative review and
known limits. No new plan schema or automatic taste score is needed.

## Capability checkpoint, 2026-10-21

These observations guide discovery; recheck the selected running build/profile.
BAS has a shared continuous-video presentation/export path in source; its
qualification notes distinguish source repair from managed-build proof. Device
Studio has accepted fixed photographic phone/table and laptop/browser appearance,
bounded fixed-video and screen-entry evidence. Physical recording/scene work and
laptop articulation remain active owner qualification; a supported render-mode
name is not visual acceptance. Transparent device-only output was not established
by the inspected public contracts; the inspected physical worker sets
`film_transparent = False`. Do not advertise alpha or hinge support from these
observations. Use current owner evidence if it supersedes this checkpoint.

Sources: `path:scenarios/device-studio/docs/concepts/INTEGRATIONS.md`,
`path:scenarios/device-studio/docs/domains/rendering.md`,
`path:scenarios/device-studio/docs/reference/cli-commands.md`,
`path:scenarios/device-studio/docs/internal/PROGRESS.md`, and
`path:scenarios/browser-automation-studio/docs/PROBLEMS.md` (BAS-RF-151).
