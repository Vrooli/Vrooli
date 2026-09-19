# Operations

`opencode` is organized as an `external-cli` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns OpenCode-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. The only
specialised Go packages today are `cli/internal/permissions` (+
`permissionscli`) for the governed `permission.bash` map and
`cli/internal/upstreamcheck` for the upstream release comparison. Install,
binary download, and config/auth shaping live in `lib/install.sh` /
`lib/common.sh`; agent-manager invokes the raw `opencode` binary directly.

## Operator Checklist

- Keep upstream binary/install/version expectations declared in `resource.json`.
- Route mutable config and auth files through canonical resource storage instead of repo-local paths.
- Separate auth/config validation from raw binary detection.
- Prefer shared lifecycle and invoke behavior before adding resource-local commands.
