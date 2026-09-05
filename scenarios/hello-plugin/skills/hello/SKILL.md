---
name: hello
description: Run the standalone hello-plugin fixture.
metadata:
  requires:
    scenarios: []
    commands:
      - hello-plugin hello
      - hello-plugin status
---

# Hello Plugin

Use the installed fixture to prove an agent can invoke a capability without the Vrooli control plane.

```bash
hello-plugin hello --name agent
hello-plugin status
```
