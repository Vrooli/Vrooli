# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `compose-service` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `compose-service`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

Use this template when the resource needs a coordinated runtime graph instead of a single container.

## Next Steps

1. Keep runtime state outside the repo; use `${RESOURCE_*_DIR}` or platform-native equivalents when bind mounts are needed.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
3. Extend `cli/main.go` only when the resource needs commands beyond the standard native lifecycle surface.
4. Replace placeholder images and ports in `compose.yaml`.
5. Document real dependency and readiness semantics in `docs/OPERATIONS.md`.
