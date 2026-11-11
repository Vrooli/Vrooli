---
name: "backdrop-studio"
description: "Generate hero background and ambient imagery for a landing page or promotional surface that matches the design system: pick a style and surface size, compose a plan, render candidates, select one, check text legibility and release it."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["backdrop-studio", "hero-image", "background", "backdrop", "landing-page", "ambient", "imagery", "render", "design-system"]
  requires:
    scenarios: ["backdrop-studio"]
    commands: ["backdrop-studio surfaces list"]
  origin:
    kind: "authored"
---

## backdrop-studio

Seeded scaffolds are deterministic; model renders return candidates you select.

| You want | Command |
|---|---|
| Surfaces and styles | `backdrop-studio surfaces list`, `backdrop-studio catalog list` |
| Seeded presets | `backdrop-studio scaffold list`, `backdrop-studio scaffold render` |
| Plan, then render | `backdrop-studio compose plan`, `backdrop-studio render submit`, `backdrop-studio render candidates` |
| Choose and release | `backdrop-studio render select`, `backdrop-studio release create` |
| Put a screenshot in a device frame | `backdrop-studio compose device-frame` |
| Is text readable on it | `backdrop-studio legibility measure` |

Run `backdrop-studio <group> <command> --help` for flags.
