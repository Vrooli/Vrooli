# Browser & devices design handoff

The operator is choosing where Aquila’s **Browser & devices** entry opens on phone and desktop. The Browser pane today looks for a browser attached to an active agent run. In the finished-run state, there is no way to reconnect or open a browser from the pane. See [the current pane](../../../ui/src/features/desktop/BrowserWorkspacePane.tsx) and [the comparison source](decision.src.html).

## Operator stance

> Gotcha, thanks. I haven't been able to use this feature, and not because I don't want to. When I open it it says the agent run has finished and that the browser session cannot be connected. There is no way to open a new browser session. And also I think we're planning on supporting device control in here (including emulations), which doesn't seem available. I think that would make the most long term sense is if there was always a browser available, and agents could simply connect to it if they choose. And I think it'd make sense that while maybe we only want one agent connected to a browser at a time, maybe we make the browsers independent from sessions so that one agent can work in one, then another can pick up where they left off, and maybe the human operator can take control at some point, and so on. And I think I'd make sense to treat real and emulated devices the same way. Thoughts on all of this? Could you use the experience engineering research task's ux redesign skill it's setting up to show me options for how this could look? I want it to be clean and intuitive and have functionality that's genuinely useful and aligned with my intent

## Decision

| Reply | Direction for the design spec |
| --- | --- |
| **A · Open the last target** (`last-target`) | Open the most recently used target directly. On first visit, open the default browser. Keep a target switcher in the viewer header. Recommended for quick return to ongoing work. |
| **B · Work beside the agent** (`beside-agent`) | Pair the viewer with the active conversation on desktop and a conversation sheet on phone. Keep the global picker available independently of the conversation. |
| **C · Browse all targets** (`browse-targets`) | Open the target list first. Show physical, emulated and browser targets with availability and control status before entering the viewer. |
| **Keep it as it is** (`keep-current`) | Close this design decision with the existing run-linked pane. |
| **No answer** | Leave the direction open. |

An A, B or C answer starts the design spec. The spec will return for review with the provider dependencies and a proposed implementation scope. The interactive prototype uses fictional browser and device state. [Open the prototype](prototype.html)

## Design the chosen layout around these flows

1. **Open with no agent.** A global entry gives the operator access to the default browser on demand. Each browser has a stable identity and exists independently of any conversation or run. The spec should show the first open, resume, opening a separate browser, startup failure and retry. Define which browser state persists after a run, a target restart and an explicit reset.
2. **Let agents attach explicitly.** An authorized agent can request a particular target and resume at its existing state. Record the target it attached to independently of the activity in the conversation. Show what happens when the target is busy and when a run ends.
3. **Hand off one driver at a time.** Show who has input control in the viewer. Specify take over, hand back, release when the agent has finished, and attach from a different agent. Define how disconnected drivers lose control and how concurrent claims resolve so the UI reports the confirmed driver.
4. **Use one flow for browsers, physical devices and emulators.** Show each in the picker with recognition, availability and driver state. Keep opening, switching and handing off consistent across them. Show only controls each target supports, such as address entry for a browser or Home and Back on a device. Include the offline physical-device and stopped-emulator paths.
5. **Verify the workflows.** Use phone and desktop layouts, both themes and 320 px for the narrow phone check. Walk through a human opening the browser on a quiet day, an agent attaching, the human taking over, a second agent resuming, and switching from the browser to a physical device and an emulator. Treat the preference as design direction and use task completion to test the interaction.

## Engineering questions for the spec

- Identify the supported provider APIs that can open and resume browser targets and connect to physical and emulated devices. Check which providers can preserve state and expose availability, capabilities and control ownership.
- Establish the authority for target inventory and input leases, the source of provider health, how authorization follows a target across runs, and what a lost connection means for viewing and input.
- Record the existing run-linked features the new surface would need to support. Verify continuity with real providers in a later implementation gate.
