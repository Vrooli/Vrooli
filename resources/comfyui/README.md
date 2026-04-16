# ComfyUI Resource

Managed ComfyUI workflow runtime for local image-generation and visual AI workflows.

## Intent

- Resource ID: `comfyui`
- Category: `ai`
- Driver: `docker-service`
- Portability tier: `partial`

## Use Cases

- Run local image-generation workflows with node-based visual composition.
- Execute reusable ComfyUI workflows for scenario automation and content pipelines.
- Provide a local visual AI runtime without depending on hosted image-generation services.

## Architecture

This resource is partially aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, and health metadata.
- `cli.enabled` is currently `false`, so this resource does not yet expose the standard Go resource CLI surface.
- `cli.sh` and `lib/` still contain the active operator and workflow behavior.
- When this resource gets a native CLI later, the intended home for resource-specific Go logic is `cli/internal/...`, not `cli/main.go`.

The intended long-term escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane where possible
3. add a thin native CLI only when the resource is ready to adopt the standard contract
4. move ComfyUI-specific logic into `cli/internal/...` rather than growing shell sprawl further

## Usage

```bash
# Check status through the current shell-driven interface
./cli.sh status

# Default API and web UI
curl http://localhost:8188/system_stats
```

Default endpoint:

- Web/API: `http://localhost:8188`

## Notes

- This resource remains shell-driven for now. I did not invent a Go CLI surface that `resource.json` does not declare.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- When native migration begins, treat `cli.sh`/`lib/` as transitional and move behavior into `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/comfyui/docs/OPERATIONS.md) as the architecture boundary for future migration work.
