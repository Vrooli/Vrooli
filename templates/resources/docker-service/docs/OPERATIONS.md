# Operations

`{{RESOURCE_NAME}}` is scaffolded as a single-container Docker-backed resource.

## Operator Checklist

- Pin a production-safe image tag.
- Keep runtime storage rooted in `${RESOURCE_CONFIG_DIR}`, `${RESOURCE_DATA_DIR}`, `${RESOURCE_CACHE_DIR}`, `${RESOURCE_LOGS_DIR}`, and `${RESOURCE_STATE_DIR}` rather than repo-local `data/`.
- Document volume backup/restore expectations.
- Replace placeholder health checks with service-specific readiness probes.
- Keep `environment_exports` as the canonical scenario-facing contract; do not reintroduce shell-based export wrappers.
- Validate both the resource manifest and at least one consuming scenario before treating the scaffold as complete.
- Define upgrade and rollback procedures.
