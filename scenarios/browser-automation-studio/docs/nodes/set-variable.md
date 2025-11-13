# Set Variable Node

`[REQ:BAS-NODE-VARIABLE-SET]` stores values in the workflow execution context so later nodes can reuse, transform, or assert on them.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Variable name** | Context key to store | Yes | Supports dot notation (e.g., `auth.token`).
| **Store as** | Optional alias/destination | No | Defaults to the variable name.
| **Source type** | `static`, `expression`, `extract` | Yes | Determines which inputs appear.
| **Value type** | `text`, `number`, `boolean`, `json` | For `static` source | Controls how the runtime coerces the value.
| **Expression** | JavaScript returning a value | For `expression` source | Runs in the page context.
| **Selector / extract type** | DOM selector + type | For `extract` source | Mirrors Extract node behavior.
| **Attribute** | Attribute name | Required when extract type = `attribute`.
| **Capture all matches** | Store array instead of first match | Optional | For extract source.
| **Timeout (ms)** | Wait for selector (extract) | Optional | Defaults to 0.

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:1347-1412` normalizes configuration, enforces required fields per source, and clamps timeouts.
2. For static sources, values are coerced based on `valueType`. Expression sources execute in the browser context; extract sources behave like Extract nodes.
3. The value is stored in the execution context map (`ExecutionContext`), and snapshots are written to artifacts for debugging.

## Example

```json
{
  "type": "setVariable",
  "data": {
    "name": "creds.user",
    "sourceType": "expression",
    "expression": "return window.sessionStorage.getItem('user');"
  }
}
```

## Tips

- Use dot-separated names (`user.session.token`) to mimic JSON structures.
- When storing JSON, set `valueType = "json"` so the runtime parses it automatically.
- Combine with Use Variable for templating (`Hello {{name}}`).
- Keep expression logic defensive; errors bubble up and fail the node.

## Related Nodes

- **Use Variable** – Reads/Transforms stored values.
- **Extract** – Use as the source for Set Variable when scraping DOM data.
- **Assert** – Validate variables before using them downstream.
