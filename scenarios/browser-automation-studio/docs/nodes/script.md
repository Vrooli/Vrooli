# Script Node

`[REQ:BAS-NODE-SCRIPT-EXECUTE]` runs arbitrary JavaScript in the current frame, enabling advanced logic without leaving the workflow graph.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Expression** | JavaScript function body | Yes | Use `return ...` to produce a value.
| **Timeout (ms)** | Max execution time | No | Clamped between 100 and 120 000; default 30 000.
| **Store result as** | Optional variable name | No | Saves the return value for later nodes.

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:1750-1785` trims the expression, enforces presence, clamps timeout, and records `storeResult` if provided.
2. Browserless injects the expression into the page context, runs it via `chromedp.Evaluate`, and returns the result (JSON-serializable) to the workflow engine.
3. When `storeResult` is set, the value is saved into the execution context, making it available to Use Variable nodes.

## Example

```json
{
  "type": "evaluate",
  "data": {
    "expression": "const price = document.querySelector('#total').textContent; return price.replace('$', '');",
    "timeoutMs": 20000,
    "storeResult": "cart.total"
  }
}
```

## Tips

- Keep scripts small; complex automation logic belongs in dedicated nodes or backend services.
- Handle errors inside the expression to provide clearer failure messages.
- Access variables via `window.__basContext` (if exposed) or rely on Set/Use Variable nodes for data passing.

## Related Nodes

- **Set/Use Variable** – Store script results for downstream steps.
- **Assert** – Validate script outcomes.
- **Conditional** – Branch based on script output stored in variables.
