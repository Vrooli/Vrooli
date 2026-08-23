---
name: sandbox
description: Create and review isolated workspace changes without the Vrooli control plane.
---

# Workspace Sandbox

The standalone `workspace-sandbox` command starts its bundled local API on
loopback and stores state under the user-scoped data and state directories.
It does not require a Vrooli checkout or control-plane binary.

Create a clean-room sandbox:

```bash
workspace-sandbox sandbox create --name plugin-demo --scope-path /tmp --project-root /tmp --owner plugin-demo --idempotency-key plugin-demo-1
```

The standalone change commands accept the deterministic `latest` handle for the
newest active sandbox:

```bash
workspace-sandbox change diff latest
```

Promote only after reviewing the diff:

```bash
workspace-sandbox change promote latest --actor plugin-demo --confirm
```
