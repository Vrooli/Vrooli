---
name: "music-generation"
description: "Pointer to the governed music-tools composition capability."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["marketing", "audio", "music", "video"]
  status: "retired"
  revision: 3
  updatedAt: "2026-09-19T00:00:00Z"
---

# Retired: use the governed composition capability

This skill no longer owns model setup, dependency installation, weight
acquisition, or generation. Those operations belong to the checksum-verified
`ace-step` managed resource and the `music-tools` scenario.

Use the scenario surface instead:

```bash
music-tools styles list
music-tools compose run --style launch-trap --takes 10 --duration 45
music-tools jobs wait <job-id>
music-tools takes list --job <job-id>
```

The scenario records model, licence lane, seed, authored/sent captions, and
capacity rung for every take. Do not create a virtualenv or download weights
from this pointer.
