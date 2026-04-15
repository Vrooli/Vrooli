# Operations

`{{RESOURCE_NAME}}` is scaffolded as a cloud API resource.

## Operator Checklist

- Replace the placeholder endpoint and health URL.
- Wire credentials to the real secret source.
- Keep any local cache/config state in canonical resource storage directories, not repo-local `data/`.
- Document auth rotation and failure modes.
- Clarify which API actions are safe for smoke checks.
