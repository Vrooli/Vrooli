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

This writer skill's declared producer registration is `friction-inbox/*`,
but the prose below references `bug-inbox/<scope>/<slug>` — a prefix
outside `writes_to[]`. The prose-scan rule should fire on this line.
