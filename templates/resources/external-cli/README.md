# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `external-cli` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `external-cli`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

Use this template for CLIs like `codex`, `claude-code`, `terraform`, or `ffmpeg`.

## Next Steps

1. Keep mutable runtime state outside the repo; if the CLI needs files, resolve them through the resource storage/runtime layer.
2. Extend `cli/main.go` only when the resource needs commands beyond the standard native lifecycle surface.
3. Replace placeholder install/version checks with the real binary contract.
