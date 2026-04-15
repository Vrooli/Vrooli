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
2. Extend `cli/main.go` only when the resource needs commands beyond the standard native lifecycle surface.
3. Document manual prerequisites and validation probes explicitly.
