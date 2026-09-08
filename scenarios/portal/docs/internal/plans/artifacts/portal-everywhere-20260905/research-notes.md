# Research notes and provenance

Primary sources were inspected during the discussion and refreshed selectively on 2026-09-05. These are bounded paraphrases, not copied articles. No third-party performance claim is an acceptance result for Vrooli. Dependency/license decisions require a pinned implementation review.

## Electron security

Source: [Electron security](https://www.electronjs.org/docs/latest/tutorial/security)

Use isolated sandboxed renderers, disabled Node integration for remote content, narrow IPC, and sender validation. This supports a trusted shell around untrusted scenario embeds.

## XDG RemoteDesktop

Source: [XDG RemoteDesktop](https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.RemoteDesktop.html)

Linux remote-desktop portal exposes session-based device selection and input facilities. Backend availability must be tested on named environments.

## Windows interactive services

Source: [Windows interactive services](https://learn.microsoft.com/en-us/windows/win32/services/interactive-services)

Windows service execution and user desktop interaction require a deliberate session boundary. Use a user-session helper and authenticated IPC.

## WebRTC TURN

Source: [WebRTC TURN](https://webrtc.org/getting-started/turn-server)

Direct peer connectivity is not always available; relay infrastructure is part of a reliable WebRTC design. Account for deployment and bandwidth costs.

## Terminator

Source: [Terminator](https://github.com/mediar-ai/terminator)

A Windows desktop automation implementation candidate. Evaluate native accessibility coverage and packaging against the fixture before adoption.

## Agent S

Source: [Agent S](https://github.com/simular-ai/Agent-S)

An open computer-use framework useful for studying agent observation/action loops and evaluation. Its stated platform reach is not evidence that Vrooli adapters pass.

## Microsoft UFO

Source: [Microsoft UFO](https://github.com/microsoft/UFO)

A research and implementation reference for hybrid GUI/API automation and distributed agents. Reuse concepts selectively instead of replacing Vrooli owners.

## Raycast screen awareness

Source: [Raycast screen awareness](https://manual.raycast.com/ai/screen-awareness)

Reference for presenting screen context in a fast assistant interaction. It does not establish a Vrooli capability or a license to copy product assets.

## Codex computer use

Source: [Codex computer use](https://openai.com/index/codex-for-almost-everything/)

Product reference for combining agent work and computer interaction. The public terminal-agent repository does not establish that the desktop application source is open.

## Apple Accessibility

Source: [Apple Accessibility](https://developer.apple.com/documentation/applicationservices/axuielement)

Native semantic-access API candidate identified in prior discussion. Reverify exact methods and supported OS versions during adapter design.

## Apple ScreenCaptureKit

Source: [Apple ScreenCaptureKit](https://developer.apple.com/documentation/screencapturekit)

Native capture API candidate with permission requirements. Validate packaging identity and actual capture on the supported host.

## Linux libei

Source: [Linux libei](https://libinput.pages.freedesktop.org/libei/)

Emulated-input infrastructure intended for compositor-controlled access. Evaluate it alongside desktop portals for Wayland support.

## Power Automate recorder

Source: [Power Automate recorder](https://learn.microsoft.com/en-us/power-automate/desktop-flows/recording-flow)

Reference for demonstration capture and editable workflows. Recording still needs semantic conversion and independent acceptance in Vrooli.

## Raycast pricing

Source: [Raycast pricing](https://www.raycast.com/pricing)

Commercial packaging reference from the discussion. Do not copy current prices into the plan as a durable market fact.

## RustDesk

Source: [RustDesk](https://github.com/rustdesk/rustdesk)

Remote-session UX and transport reference. Review its license at a pinned revision before incorporating code; study does not imply adoption.

## Commercial interpretation

The proposed differentiator is verified reusable work across owned machines, browsers, and devices. Audience, pricing, and willingness to pay remain hypotheses. Validate three concrete journeys and collect measured cost/support data before publishing claims.
