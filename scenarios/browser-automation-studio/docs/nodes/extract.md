# Extract Node

`[REQ:BAS-NODE-EXTRACT-DATA]` pulls text, attributes, input values, or HTML from the DOM and (optionally) stores the result in a workflow variable.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Page URL override** | Optional absolute URL for element picker | No | Defaults to upstream Navigate node.
| **Selector** | CSS selector to extract from | Yes | Required unless using the expression source via Set Variable.
| **Extract type** | `text`, `attribute`, `value`, `html` | Yes | Defaults to `text`.
| **Attribute name** | Attribute to read | For `attribute` type | e.g., `data-testid`.
| **All matches** | Capture array instead of first match | No | Sets `allMatches` flag.
| **Store in** | Variable name for the result | Optional | Provided via Set Variable or Use Variable nodes.
| **Timeout (ms)** | Wait for selector | No | Defaults to 5 000.

## Runtime Behavior

1. The compiler treats Extract as part of the Set Variable pipeline (`setVariable` with `sourceType=extract`).
2. `api/browserless/runtime/instructions.go` (extract portion) reads the selector, wait, extract type, and attribute, enforcing validation rules.
3. Browserless queries the DOM, normalizes whitespace for text, and returns either a string or array based on `allMatches`.
4. Execution artifacts include the extracted payload (truncated) for debugging.

## Example

```json
{
  "type": "extract",
  "data": {
    "selector": "#headline",
    "extractType": "text",
    "storeResult": "pageTitle"
  }
}
```

## Tips

- When scraping lists, enable `allMatches` and pair with Loop nodes to iterate over the array.
- Attribute extraction is case-sensitive; double-check attribute names in DevTools.
- For complex parsing, store the result via Set Variable and transform it with Use Variable or Script nodes.

## Related Nodes

- **Set Variable** – Use its extract source for advanced storage options.
- **Use Variable** – Reference extracted values in later steps.
- **Assert** – Validate extracted data to ensure scrapes remain accurate.
