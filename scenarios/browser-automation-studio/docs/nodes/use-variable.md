# Use Variable Node

`[REQ:BAS-NODE-VARIABLE-USE]` reads values from the execution context, optionally transforms them, and exposes the result (and alias) to subsequent nodes.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Variable name** | Name to read | Yes | Autocomplete lists known variables in the workflow.
| **Store as** | Optional alias for the transformed value | No | Defaults to the same name.
| **Transform template** | Text with `{{value}}` placeholder | No | Allows lightweight templating without Script nodes.
| **Required** | Fail if variable missing | No | When unchecked, node passes through without error.

## Runtime Behavior

1. The automation executor handles variable resolution and templating (`{{value}}`), failing fast when `required` is set but the variable is missing.
2. The result is written back into the context map under the alias so later nodes can reuse it.
3. Execution artifacts note whether the variable existed and what alias was written.

## Example

```json
{
  "type": "useVariable",
  "data": {
    "name": "creds.user",
    "storeAs": "welcomeMessage",
    "transform": "Welcome, {{value}}!",
    "required": true
  }
}
```

## Tips

- Use Use Variable nodes for templating log messages, selectors, or request payloads without adding Script nodes everywhere.
- When chaining transformations, store under new aliases to preserve the original value for debugging.
- Combine with Conditional nodes to branch when variables are missing or have unexpected values.

## Related Nodes

- **Set Variable** – Produces the values you consume here.
- **Conditional** – Branch based on variable contents.
- **Script** – For advanced transformations not covered by simple templates.
