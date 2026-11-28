# Browser & devices in the existing Aquila workspace

**Design proposal for implementation.** [Walkthrough](index.html) · [Interactive prototype](prototype.html) · [Served baseline and provider inventory](CURRENT.md)

## Entry points and return behavior

| Launch point | Placement in Aquila | Destination |
| --- | --- | --- |
| **Browser & devices** | A labelled row immediately above Archive in the existing Sessions footer. The same row appears inside the existing left drawer on phones. | Open the last authorized target across the workspace. On first use, start the default managed browser on demand. If the last target is offline, show its recovery action and leave the target picker available. |
| **Globe** | Keep the existing location beside Terminal in Messages and the corresponding Terminal and grid controls. | Open the target shared with this conversation. When none is attached, offer the same picker. Selecting a target to view and granting an agent control are two distinct actions. |
| **Browser & devices** with zero sessions | Place the same labelled action on the existing empty-workspace welcome surface. Make it reachable in tabs and grid display modes from their existing workspace chrome. | Same global entry. A person can open a browser before starting an agent. |
| **Messages / Terminal** in the viewer | Keep the pane-view controls. Give the browser/device screen and its activity strip the space used by conversation input in Messages. | Return to the same session and view with its draft. Persistent targets and active agent leases continue independently of the displayed view. |

Implementation anchors are the shared [Sessions rail and drawer](../../../../ui/src/components/SessionSidebar.tsx#L783), [phone header](../../../../ui/src/components/Workspace.tsx#L2330), [existing globe handler](../../../../ui/src/components/Workspace.tsx#L1696), [empty workspace](../../../../ui/src/components/Workspace.tsx#L2040), and [viewer mount](../../../../ui/src/components/WorkspacePaneShell.tsx#L280).

## Switch, start, and connect

1. Select **the target name and chevron** at the upper left of the viewer. The same list contains browsers, physical devices, and emulators. Each row has a name, type, availability, and control status. The current target is checked. Desktop uses a dialog. Phone uses a bottom sheet and keeps Sessions in the existing left drawer.
2. Select a row. If the target is available, open its current screen. If the target is offline or sleeping, show the provider's recovery action in the viewer. Start at the selected target's Live edge and keep each target's history separate.
3. Use **Add or connect** from the same list. **New browser** creates an independent managed browser with a fresh profile. **Start an emulator** shows supported configurations and opens the new virtual device when it is ready. **Connect a device** lists known devices and opens Device Control onboarding for a new one. Reuse the fleet pairing/Bridge paths where the target requires them. Show connecting, failure, and retry on the owning route and return the authorized target to the viewer after success.

## Control at a glance

| Viewer state | Header | Primary action and input |
| --- | --- | --- |
| Live with no driver | **Ready** | **Use browser** or **Control device**. Send input after exclusive control is confirmed. |
| Live while you drive | **You have control** | **Share** to give a named agent a turn. If you interrupted an active agent, show **Return to [agent]**. Input goes to the selected target. |
| Live while an agent drives | **[Agent] is driving** | **Take control**, when available. View the stream with local input disabled. **Capturing screenshots** appears while passive capture is active. |
| Reviewing a past screenshot | Keep the live driver in the header. Label the screen **Reviewing screenshot**, show actor, time and action, and mark it **Read-only**. | **Return to Live** or drag to the rightmost stop. The review cursor changes independently of the live lease. Evaluate input permissions again when returning to Live. |

The current driver comes from the owner's confirmed lease. Opening an eligible idle target can confirm human control on entry. If the target remains idle, offer the explicit action above. During a pending handoff, show progress and gate input until confirmed. If a run finishes, keep the target open, release its driver, and keep eligible screenshots. An authorized next agent can attach and continue. If the streaming connection drops, report the actual live state and retain access to available history.

## Agent activity beneath the screen

- Present a horizontal strip in chronological order. Each checkpoint has a screenshot or unavailable state, timestamp, named actor, and action. Show the first and last eligible frame of an agent's turn and capture material steps such as completed navigation or a settled visible change. Use a bounded capture cadence. Group checkpoints by turn when the strip grows. Give the selected item its context and a route to the source conversation where available.
- A timeline cursor selects saved screenshots. The far-right position is **Live** and the Live button stays visible even when older thumbnails scroll offscreen. Keyboard End reaches the same destination. On phones, keep the strip directly below the screen and reserve thumb reach for the current control action. Adjust thumbnails and metadata for the narrow viewport.
- Append new screenshots while the agent works. Preserve the selected checkpoint while the user reviews history. Show later screenshots without changing the live driver or moving the selection. For a stopped or disconnected target, the rightmost slot shows that status while the retained checkpoints remain available.
- Scope history to the stable target ID. Browser Automation Studio and Device Control own their screenshot artifacts and lease-bound capture. Aquila renders an authorized, ordered event projection with actor/run linkage and opaque image references. Treat browser navigation history as a separate provider operation. Enforce the provider's capture and history permissions, retention, and redaction. Represent a withheld, expired, or failed capture with metadata and an honest availability label.

## Handoff and verification

The [prototype](prototype.html) is a local interactive reconstruction based on the served desktop and phone captures. It uses sample messages, a simulated provider, and recorded images of the sample storefront. Opening **Share → Codex** adds sample checkpoints while Codex drives. While reviewing, move the scrubber all the way to the right to return to the current screen. The same **Add or connect** menu demonstrates all three target types.

Before shipping, verify the actual **Sessions** and **Messages** entry points on desktop and phone, including zero sessions, grid, and tabs modes. Exercise real browser creation and continuity across two agents, physical pairing and reconnect, and emulator open/start under their supported transports. Prove that a second checkpoint can arrive while reviewing an older image and that Live restores the current screen and confirmed driver. Confirm authorized history remains readable after the agent run ends and that unavailable screenshots have a labelled state. The provider seams and current evidence are listed in [CURRENT.md](CURRENT.md).

Chromium verified the prototype at 1440 × 900, 390 × 844, and 320 × 720 in light and dark. The checks exercised both existing-shell entry paths, all three chooser options, takeover and handback, view changes across conversations, a new checkpoint while reviewing, and returning to Live with the keyboard at the right edge. The served baseline uses intercepted API fixtures. Actual provider launch, streaming, control, and retention require integration verification.
