# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `docker-service` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `docker-service`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

## Next Steps

1. Replace the placeholder container image in `resource.json`.
2. Define real health checks and port mappings.
3. Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths; do not replace them with repo-local `data/` paths.
4. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
5. Extend `cli/main.go` only when the resource needs commands beyond the standard native lifecycle surface.
6. Define `environment_exports` for every scenario-facing variable this resource provides.
7. Run `vrooli resource validate {{RESOURCE_NAME}}` and `vrooli scenario validate-env <scenario>` before removing any compatibility paths.
8. Update `docs/OPERATIONS.md` with production runtime notes.
9. Add smoke/integration coverage for the real service behavior.
