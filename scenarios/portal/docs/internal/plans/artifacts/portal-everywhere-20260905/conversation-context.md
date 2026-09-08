# Preserved conversation context

This is a structured preservation of the discussion available to the author. It is not a verbatim transcript and does not claim access to omitted messages. No screenshots or generated image attachments were supplied in this discussion. Diagrams, wireframes, contracts, source snapshots, and research notes are preserved locally.

## Origin and motivation

The operator described scenarios becoming permanent capabilities through skills, governed programs, and authoritative APIs. Deployment Manager provided the motivating example: immutable release review identity, typed evidence producers, deduplicated Swarm work, independent human checks, and evidence-backed release gates. The lesson is that a program completing is not equivalent to the desired real-world outcome being proven.

The operator wanted the same maturity pattern for capabilities that are difficult, slow, or unreliable. Scenario-to-Desktop was discussed as a deployment ramp whose self-improvement work should mature all documented desktop journeys, using emulators and Bridge hosts where available.

For Device Control, the immediate problem was agent overhead when controlling a TV: orientation, finding the device, discovering capabilities, constructing a workflow, interpreting screenshots, and using AI control. BAS was the browser analogy. Earlier implementation in this conversation reported versioned flow reuse, verified promotion, repair preservation, effort telemetry, and governed skills/programs. Physical-TV timing and complete broad-suite health were explicitly not proven. Do not inherit those implementation reports as a current baseline.

## Evolution of the proposed architecture

1. The operator recalled an old vrooli-prefixed desktop assistant scenario.
2. Inspection identified vrooli-assistant: a hotkey issue-capture and agent-spawning overlay.
3. The discussion considered rebuilding it as full desktop automation across Windows, macOS, and Linux.
4. The operator clarified that Bridge installs Vrooli on connected nodes; remote capability should run there and cooperate through Bridge.
5. Portal was introduced as the unified chat, search, embedded-scenario, and app-opening experience.
6. Web Console was identified as a related machine/device interface retaining terminal-control ownership.
7. Initial advice favored a separate desktop-control capability with a native Portal companion.
8. Further source inspection found Device Control's existing host-desktop adapter and shared targetmodel infrastructure.
9. The final recommendation became: extend Device Control, keep Bridge as remote owner, and put the native companion source/configuration in Portal.
10. Scenario-to-Desktop packages the companion through a versioned extension mechanism rather than a forked vanilla template.

## Explicit user expectations

- The native companion behaves like the ordinary Portal UI.
- It opens with a global shortcut and has tray/menu integration.
- It can appear as a minimal floating pill and expand into a palette or full UI.
- Its expanded mode uses the full normal UI, with shared state and API contracts.
- The current machine is a target like connected machines and devices.
- Native context includes pre-focus window/pointer capture, selected text where available, region selection, and annotation.
- Portal can show and control connected surfaces as naturally as scenario widgets.
- Manual input and AI task delegation coexist coherently.
- Programs can combine BAS, Device Control, terminal, and scenario operations.
- Optional capabilities can be absent from deployment bundles without breaking Portal.
- Smart fallback must preserve target, account, authority, and verified task meaning.
- Repeated successful work becomes faster through verified reusable procedures.
- Windows and macOS matter alongside Linux; cross-compilation alone is insufficient.
- The implementation must be maintainable, portable, mature, performant, and professionally validated.
- The plan must be executable by an agent working until objective deliverables are achieved.

## Clarifications that must survive handoff

The companion's local host can differ from the API server host. Native application permission does not replace task authorization. A connected node may be headless or locked. A screenless TV transport may still support useful controls. A saved workflow is not a copied screenshot/click trace. A failed network request after an action is not proof that the action failed. An optional application dependency may become a required task dependency.

Voice, notifications, screenshots, and remote control are not categorically native-only; browser versions can support subsets. The native shell adds system-wide integration and privileged affordances behind typed boundaries.

## Alternatives considered and rejected

- A separate replacement desktop workflow engine in vrooli-assistant: duplicates Device Control's existing ownership.
- All desktop execution inside Portal: makes the UI/API a monolithic capability owner.
- Bridge implementing application controls: mixes transport with domain semantics.
- Copying the Electron template into Portal: creates permanent regeneration and security drift.
- Unrestricted generated Python as the default workflow: does not establish portability, bounded effects, or verified replay.
- Treating every input event as a program: adds orchestration overhead to interactive sessions.
- Treating every target as a screen: excludes useful property/media/sensor devices.

## Planning authority

The operator requested an extremely detailed implementation plan, not immediate implementation or deployment. This artifact authorizes planning records and local evidence preservation. Future implementation follows the plan and active session authority. Public launch, paid infrastructure, user permission grants, and irreversible deletion need the applicable explicit authority. These limits must not become excuses to leave reversible engineering work incomplete.
