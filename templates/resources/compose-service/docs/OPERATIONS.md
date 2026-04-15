# Operations

`{{RESOURCE_NAME}}` is scaffolded as a compose-managed multi-service resource.

## Operator Checklist

- Replace placeholder images in `compose.yaml`.
- Keep bind mounts rooted in canonical resource storage directories rather than repo-local `data/`.
- Define explicit dependency and readiness rules.
- Document data directories, backup, and restore behavior.
- Add teardown guidance for local development and production support.
