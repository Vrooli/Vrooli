# User Feedback and Constraints

## Operator amendment OF-2026-11-03-01 — acknowledged, open

> Can you find me the recordings?

Resolved by locating and verifying the three matched Moto recording-on clips and one degraded unmatched clip; the copies are linked from the conversation and retain their source hashes. This request does not close the maturity effort.

> Thanks. Is there anything we should improve with the goal's data or the goal message to ensure that it works more effectively and does more? It seems to be working rather slowly. I suspect it's not being ambitious enough. Also I think we may have missed out on some capabilities that I'd like to be super easy to do with device control, such as installing (temporarily and permanently) scenarios to devices (using scenario to Android or whatever is applicable for the device) for testing purposes,and fixing any reliability issues with running bas workflows on them

**Required response:** Add scenario-built app deployment and on-device BAS reliability as explicit acceptance outcomes. Distinguish temporary test install from durable install/update, including safe pre-state/cleanup/restore verification. Exercise existing scenario-to-android → immutable artifact → Device Control verified install → BAS workflow owners as one discoverable journey. Include repeatable cold/warm runs, terminal app assertions, evidence, failure diagnosis and truthful target support. Review current owner capabilities before adding new abstractions. Improve delivery pace with substantial parallel owner-bounded work, one coordinator and capacity-aware WIP; continue independent work through lifecycle, physical-device or consent gates.

**State:** Acknowledged in STATUS; implementation, E7 proof and closure remain open.

## Operator correction — TV actions over local LAN (2026-11-03 UTC)

The operator corrects the assumption that TV actions require trusted HTTPS: they have been performed many times without HTTPS. Device Control's local LAN route is already usable for TV reads and supported actions; do not make a hostname, certificate, or DNS choice a prerequisite for ordinary TV control. Keep the actual owner boundary clear: the current Portal TV app's signed-in lifecycle client separately rejects non-HTTPS origins, and the Portal API requires a verified human operator. Do not treat that client check as a universal Device Control restriction or ask the operator for TLS material just to perform Device Control actions.

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
- Make scenario-built apps easy to install temporarily for device testing and durably for operator use, and make Browser Automation Studio workflows reliable on supported devices.
- Make scenario-built apps easy to install temporarily for testing and durably for operator use; make Browser Automation Studio workflows reliable on supported devices.
- Actively reduce relevant debt, duplication, unclear contracts, reliability gaps, and performance cost. Validate professionally and report proof and gaps.

## Authority details

- TV and phone use was authorized for this work, including device restarts.
- PIN 5555 was provided for the test phone and should only be used through the existing credential system.
- User is away from the phone. Do not trigger MediaProjection consent until they can accept prompts.
- Device identity must be freshly confirmed before changing a physical device.
