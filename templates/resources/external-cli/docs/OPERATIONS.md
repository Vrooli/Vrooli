# Operations

`{{RESOURCE_NAME}}` is scaffolded as an external CLI resource.

## Operator Checklist

- Document install guidance per OS.
- Pin a minimum supported version.
- Route mutable files through canonical resource storage directories instead of repo-local `data/`.
- Separate auth/config probing from binary detection.
- Describe any interactive steps that cannot yet be automated.
