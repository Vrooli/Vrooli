---
name: "asset-studio"
description: "Produce marketing images and videos with a consistent product identity: define product identities, import brand canon, set a campaign media spend limit, regenerate a render from its recorded intent, and check render provenance and conformance."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["asset-studio", "marketing", "images", "video", "render", "identity", "campaign", "provenance", "creative"]
  requires:
    scenarios: ["asset-studio"]
    commands: ["asset-studio identities list"]
  origin:
    kind: "authored"
---

## asset-studio

Structured-data-driven production with render provenance.

| You want | Command |
|---|---|
| Identities | `asset-studio identities list`, `asset-studio identities create-product`, `asset-studio identities import` |
| A render's status and receipt | `asset-studio renders show` |
| Redo a render | `asset-studio renders regenerate` |
| Spend limit | `asset-studio renders set-campaign-budget` |
| Quality signal | `asset-studio renders analyze-conformance` |

Run `asset-studio <group> <command> --help` for flags.
