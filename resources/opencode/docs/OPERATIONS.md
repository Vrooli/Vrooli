# Operations

`opencode` is organized as an `external-cli` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns OpenCode-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. The only
specialised Go packages today are `cli/internal/permissions` (+
`permissionscli`) for the governed `permission.bash` map and installed-major
plugin projection, and `cli/internal/config` / `ensurecli` for config and
provider shaping. Install, binary download, and version prefixes live in
`packages/harness-lifecycle` and the Scenario Dependency Analyzer;
agent-manager invokes the raw `opencode` binary directly.

Release lines are declared as acquisition channels in `resource.json`: the
empty/default channel is OpenCode 1 (`opencode-ai`) and `v2` is OpenCode 2
(`@opencode/cli`). `vrooli resource update opencode --channel v2` is the
governed one-command upgrade; `--major 2` resolves the same channel. A plain
update follows the installed major so it never silently downgrades a V2 host.

## Operator Checklist

- Keep upstream binary/install/version expectations declared in `resource.json`.
- Route mutable config and auth files through canonical resource storage instead of repo-local paths.
- Separate auth/config validation from raw binary detection.
- Prefer shared lifecycle and invoke behavior before adding resource-local commands.
