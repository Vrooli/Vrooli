---
name: "device-studio"
description: "Make a polished device mockup or presentation from an app screenshot or screen recording: create a project, import and inspect source media, build shots, and render a phone or 3D device scene from authentic app captures."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["device-studio", "device-mockup", "mockup", "screenshot", "3d", "presentation", "app-capture", "marketing-visual"]
  requires:
    scenarios: ["device-studio"]
    commands: ["device-studio projects create"]
  origin:
    kind: "authored"
---

## device-studio

Projects hold source media, inspected frames and shots; revisions are immutable.

| You want | Command |
|---|---|
| Start a project | `device-studio projects create`, `device-studio projects list` |
| Bring in a capture and inspect it | `device-studio media inspect`, `device-studio media list`, `device-studio media get` |
| Compose shots | `device-studio shots create`, `device-studio shots save`, `device-studio shots list` |

Run `device-studio <group> <command> --help` for flags.

For capturing the screenshot from a real TV or phone first, read the `device-control` skill.
