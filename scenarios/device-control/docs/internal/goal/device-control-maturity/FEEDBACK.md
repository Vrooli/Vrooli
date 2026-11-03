# User Feedback and Constraints

## Durable requirements

- Keep working through the full agreed scope; do not stop after a few minutes or after one successful item.
- Make TV, Android phone, and other supported devices straightforward to control through common language and shared capture operations where possible.
- Distinguish screen wake, unlock/authenticated readiness, and navigation to Home/launcher. Reusable sequences should compose into workflows.
- Determine wireless-debugging support per device/OS. Where supported, provide an enable/disable flow that is reusable. Do not infer sleep caused an isolated Wi-Fi outage; repeat controlled observations.
- Screenshots should show the real full frame by default. The reported black top bar was not on the physical TV. User found masking/reduction unhelpful and wanted it removed or at least disabled by default; do not redact without an explicit supported request.
- Recordings must capture actual frames at useful rates, report dropped frames and duration honestly, and avoid unacceptable playback lag. Sound should be available as an optional recording mode when supported.
- A lack of physical prompt response is missing evidence, not an implementation failure. User is currently away from the phone and cannot accept prompts.
- Improve AI-powered control speed and quality, expose reusable screenshot/semantic/evidence tools, evaluate decision models through AI Gateway, retain outcome quality and fallback, and promote only validated reusable flows.
- Portal's custom TV launcher may be falling back to Google launcher; diagnose rather than assume a crash. Improve crash/lifecycle diagnostics and fix evidenced root causes. Long-press menus should use “app” terminology and offer start/restart/stop for owned apps; a non-running scenario app should offer start.
- Actively reduce relevant debt, duplication, unclear contracts, reliability gaps, and performance cost. Validate professionally and report proof and gaps.

## Authority details

- TV and phone use was authorized for this work, including device restarts.
- PIN 5555 was provided for the test phone and should only be used through the existing credential system.
- User is away from the phone. Do not trigger MediaProjection consent until they can accept prompts.
- Device identity must be freshly confirmed before changing a physical device.
