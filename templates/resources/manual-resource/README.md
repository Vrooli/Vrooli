# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `manual-resource` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `manual`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

## Next Steps

1. Keep any operator-managed state outside the repo through the resource storage/runtime layer.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
3. Extend `cli/main.go` only when the resource needs commands beyond the standard native lifecycle surface.
4. Document manual prerequisites and validation probes explicitly.
