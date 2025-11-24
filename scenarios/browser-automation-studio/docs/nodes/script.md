# Script Node

`[REQ:BAS-NODE-SCRIPT-EXECUTE]` runs arbitrary JavaScript in the current frame, enabling advanced logic without leaving the workflow graph.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Expression** | JavaScript function body | Yes | Use `return ...` to produce a value.
| **Timeout (ms)** | Max execution time | No | Clamped between 100 and 120 000; default 30 000.
| **Store result as** | Optional variable name | No | Saves the return value for later nodes.

## Runtime Behavior

1. The automation compiler passes the expression/timeout/storeResult fields directly into the contract instruction; validation is handled upstream (UI + validator).
2. BrowserlessEngine forwards the instruction to the CDP adapter, which evaluates the script in-page and returns the JSON-serializable result.
3. When `storeResult` is set, the executor persists the returned value so downstream Use Variable nodes can read it.

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
