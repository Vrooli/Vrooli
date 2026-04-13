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
3. Define `environment_exports` for every scenario-facing variable this resource provides.
4. Run `vrooli resource validate {{RESOURCE_NAME}}` and `vrooli scenario validate-env <scenario>` before removing any compatibility paths.
5. Update `docs/OPERATIONS.md` with production runtime notes.
6. Add smoke/integration coverage for the real service behavior.
