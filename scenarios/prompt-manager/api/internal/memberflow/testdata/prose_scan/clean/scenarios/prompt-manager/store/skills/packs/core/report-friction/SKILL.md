---
name: report-friction
description: Write one friction entry per invocation.
license: CC-BY-4.0
metadata:
  requires:
    scenarios: []
    commands: []
  origin:
    kind: authored
---

# report-friction

Writes one friction entry per invocation:

`prompt-manager team knowledge-add meta-optimization --topic="friction-inbox/<scope>/<slug>" --content="..."`

`friction-inbox/*` is declared in this skill's `skill.json::writes_to[]`,
so the prose scanner stays clean for this writer skill.
