# vrooli-cli-go

`vrooli-cli-go` is a thin Go client for typed `vrooli ... --json` CLI output.
It runs the root `vrooli` binary, decodes the generated `vrooli.cli.v1` proto
contracts with `protojson`, and returns the full response messages so callers
keep envelope fields such as `success` and `discovery_failures`.

## Contract

- Typed methods inject `--json` and decode generated proto messages.
- `Output(ctx, args...)` is the raw escape hatch for commands that are not typed
  yet; it does not inject flags.
- Command and decode errors are returned to the caller. The client never turns an
  error into an empty response.
- A 30 second timeout is applied only when the caller's context has no deadline.
- Unknown JSON fields are discarded during decode so newer CLI fields do not
  break older consumers.

## Usage

```go
client := vroolicli.New()

resources, err := client.ListResources(ctx)
if err != nil {
    return err
}
for _, resource := range resources.GetResources() {
    _ = resource.GetName()
}
```

Tests can replace process execution with the `Runner` seam:

```go
client := vroolicli.New(vroolicli.WithRunner(fakeRunner))
```

## Local module wiring

Scenarios are separate Go modules, so replace directives are not transitive.
Consumers in this repository need both replaces:

```go
require github.com/vrooli/vrooli-cli-go v0.0.0

replace github.com/vrooli/vrooli-cli-go => ../../../packages/vrooli-cli-go
replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
```
