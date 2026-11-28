---
name: "ux-capture"
description: "Capture what a page really looks like, edit that capture into a proposal, and hand both to a decision page with honest fidelity labels."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  revision: 1
  modes: ["tools"]
  tags: ["ux", "capture", "redesign", "decisions", "fidelity"]
  requires:
    scenarios: ["react-component-library", "browser-automation-studio"]
    commands: ["react-component-library page capture", "react-component-library page render"]
  origin:
    kind: "authored"
---

## Show me the real thing, then show me the change

Every redesign starts from what the page looks like today, captured, not
remembered. Then edit that capture into the proposal, so before and after
differ only by the change. One command does the capture; BAS is the browser.

### 1. Capture

```bash
react-component-library page capture --scenario image-tools                 # a served scenario, route /
react-component-library page capture --scenario X --route /jobs --step click:Open\ navigation
react-component-library page capture --url https://example.com              # public page
react-component-library page capture --file mockup.html                     # a local HTML file
```

Default views are phone (390 px) and desktop, light and dark; add `--views
small-phone` for 320 px. The command fails when any view has no usable pixels:
a blank or flat image is never reported as a capture. Read the printed bundle
directory. It holds `capture.json` (capture ID, source URL, route, build, time),
`pixels/<view>-<theme>.png`, `page/<view>-<theme>.html` (frozen, self-contained,
editable), `outline.md`, and a `README.md` with the fidelity table.

If the README says the dark view has "same pixels as light", the app ignores
the system theme. Say so; do not present it as a dark design.

### 2. Read the outline before the pixels

`outline.md` lists headings, landmarks and every action top to bottom, each
with an id like `n93` and its box (`@x,y w×h`). Use it to find the region to
change and to measure the problem ("eight tabs, 49 px each"). Every element in
the frozen page carries the same id: `data-ux-node~="n93"`.

### 3. Edit the frozen page into the proposal

Copy `page/<view>-<theme>.html` per theme and edit the copy. Swap in real
component markup and CSS (capture the RCL story harness with `page capture
--scenario react-component-library --route '/preview/react-component-library:<Component>/harness.html?story=<id>'`
and lift from its frozen page). A frozen page keeps only the CSS its captured
state uses, so a state the page was not in (a drawer, a menu, another nav mode)
needs that state captured from the story first. Keep the edit in a small script beside the
capture when it is more than a hand edit, so it can be re-run on a fresh capture.
Then render each copy in its own theme:

```bash
react-component-library page render proposal-light.html --views phone,small-phone --themes light
```

Look at every PNG yourself before you show it to anyone.

### 4. Label fidelity honestly

| Label | Means |
|---|---|
| `captured` | Pixels from the running product. |
| `rendered` | The working product or component with the change, rendered. |
| `composited` | A captured page with real component parts swapped in, re-rendered. Not the finished app. |
| `sketch` | Anything drawn by hand or by a model. |

Never upgrade a label because it looks good. A mixed picture takes the lowest label.

### 5. Hand off to a decision

Put today beside the proposal on one HTML page and send it as a Notification Hub
explainer: read `prompt-manager skill read notification-hub` for the rules. For
UX decisions in the experience-engineering topic, its `present-ux-decisions`
skill has the layout and a starter page. Show capture IDs and source URLs at the
bottom of the page, never at the top.
