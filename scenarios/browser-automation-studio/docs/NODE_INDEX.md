# Browser Automation Studio Node Index

> Source of truth for all drag-and-drop nodes exposed in the Browser Automation Studio palette. The list mirrors `WORKFLOW_NODE_DEFINITIONS` in `ui/src/constants/nodeCategories.ts` and the compiler's `supportedStepTypes`, making it easy to confirm parity between UI, runtime, and documentation.

| Category | Node | Description | Reference |
| --- | --- | --- | --- |
| Navigation & Context | Navigate | Open a URL or jump to another scenario runbook | [docs/nodes/navigate.md](./nodes/navigate.md) |
|  | Scroll | Scroll the viewport or a container until a target is visible | [docs/nodes/scroll.md](./nodes/scroll.md) |
|  | Tab Switch | Activate an existing tab or wait for a popup target | [docs/nodes/tab-switch.md](./nodes/tab-switch.md) |
|  | Frame Switch | Enter/exit iframes using selector, index, or URL heuristics | [docs/nodes/frame-switch.md](./nodes/frame-switch.md) |
|  | Rotate | Swap viewport orientation/angle mid-run for mobile flows | [docs/nodes/rotate.md](./nodes/rotate.md) |
|  | Gesture | Dispatch swipe, pinch, tap, and long-press gestures | [docs/nodes/gesture.md](./nodes/gesture.md) |
| Pointer & Gestures | Click | Click DOM elements with timing + retry controls | [docs/nodes/click.md](./nodes/click.md) |
|  | Hover | Move the cursor and hold to trigger hover-only UI | [docs/nodes/hover.md](./nodes/hover.md) |
|  | Drag & Drop | Simulate HTML5 drag/drop or pointer drag sequences | [docs/nodes/drag-drop.md](./nodes/drag-drop.md) |
|  | Focus | Focus an element before typing to trigger UI hooks | [docs/nodes/focus.md](./nodes/focus.md) |
|  | Blur | Defocus to fire validation/onBlur handlers | [docs/nodes/blur.md](./nodes/blur.md) |
| Forms & Input | Type | Type text with delay/randomization controls | [docs/nodes/type.md](./nodes/type.md) |
|  | Select | Choose dropdown options by value/text/index (single or multi) | [docs/nodes/select.md](./nodes/select.md) |
|  | Upload File | Attach local files to `<input type="file">` elements | [docs/nodes/upload-file.md](./nodes/upload-file.md) |
|  | Shortcut | Dispatch keyboard shortcuts (Ctrl/Cmd + key combos) | [docs/nodes/shortcut.md](./nodes/shortcut.md) |
|  | Keyboard | Low-level keydown/keyup/keypress simulation | [docs/nodes/keyboard.md](./nodes/keyboard.md) |
| Data & Variables | Extract | Grab text/attributes/json and optionally store variables | [docs/nodes/extract.md](./nodes/extract.md) |
|  | Set Variable | Create/update a workflow variable from static/extracted data | [docs/nodes/set-variable.md](./nodes/set-variable.md) |
|  | Use Variable | Validate or transform variables for downstream nodes | [docs/nodes/use-variable.md](./nodes/use-variable.md) |
|  | Script | Run arbitrary JavaScript within the current frame | [docs/nodes/script.md](./nodes/script.md) |
| Assertions & Observability | Wait | Block until selectors, requests, or custom probes settle | [docs/nodes/wait.md](./nodes/wait.md) |
|  | Assert | Validate expressions, selectors, or variable comparisons | [docs/nodes/assert.md](./nodes/assert.md) |
|  | Screenshot | Capture viewport or element screenshots for artifacts | [docs/nodes/screenshot.md](./nodes/screenshot.md) |
| Workflow Logic | Conditional | Branch workflows using expression/element/variable checks | [docs/nodes/conditional.md](./nodes/conditional.md) |
|  | Loop | Execute a sub-graph repeatedly via for-each, repeat, or while logic | [docs/nodes/loop.md](./nodes/loop.md) |
|  | Call Workflow | Run another saved workflow inline inside the same session | [docs/nodes/workflow-call.md](./nodes/workflow-call.md) |
| Storage & Network | Set Cookie | Create/update cookies (name/value/domain/path/samesite) | [docs/nodes/cookie-storage.md](./nodes/cookie-storage.md) |
|  | Get Cookie | Read cookies into variables for assertions or reuse | [docs/nodes/cookie-storage.md](./nodes/cookie-storage.md) |
|  | Clear Cookie | Delete specific cookies or flush the jar | [docs/nodes/cookie-storage.md](./nodes/cookie-storage.md) |
|  | Set Storage | Write to localStorage/sessionStorage with JSON validation | [docs/nodes/cookie-storage.md](./nodes/cookie-storage.md) |
|  | Get Storage | Read storage values and store them as variables | [docs/nodes/cookie-storage.md](./nodes/cookie-storage.md) |
|  | Clear Storage | Remove storage keys or wipe the namespace | [docs/nodes/cookie-storage.md](./nodes/cookie-storage.md) |
|  | Network Mock | Stub, modify, or block HTTP traffic via Fetch interception | [docs/nodes/network-mock.md](./nodes/network-mock.md) |

## Using This Index

- **Parity checks** - when a new node ships, update this table right after editing `WORKFLOW_NODE_DEFINITIONS` so docs stay in sync.
- **Documentation** - the Reference column should always link to a README; treat any missing entry as a doc gap and add one immediately.
- **QA sweeps** - testers can use the table as a checklist to ensure every node has coverage in unit, Vitest, and workflow-level suites.

_Last updated: 2025-11-13 14:15:28 UTC_
