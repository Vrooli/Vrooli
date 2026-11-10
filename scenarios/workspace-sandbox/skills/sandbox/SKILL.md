---
name: sandbox
description: "Give an agent or tool a safe, isolated copy of a project folder to work in, review what changed, and then accept or discard the changes, without the Vrooli control plane. Copy-on-write workspaces that are fast and storage-efficient."
metadata:
  requires:
    scenarios: []
    commands:
      - workspace-sandbox sandbox create
      - workspace-sandbox change diff
      - workspace-sandbox change promote
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
