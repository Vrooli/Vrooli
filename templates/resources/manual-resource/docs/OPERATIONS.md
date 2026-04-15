# Operations

`{{RESOURCE_NAME}}` is scaffolded as a manual resource.

## Operator Checklist

- Document every prerequisite explicitly.
- Capture the manual install process step by step.
- Keep any local state in canonical resource storage directories, not repo-local `data/`.
- Define the validation probes Vrooli can run after setup.
- State what Vrooli does not own.
