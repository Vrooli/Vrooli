# Frame Switch Node

`[REQ:BAS-NODE-FRAME-SWITCH]` keeps workflows aligned with iframe-heavy applications by explicitly controlling which browsing context future nodes use.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Switch strategy** | `selector`, `index`, `name`, `url`, `parent`, or `main`. | No | Defaults to `selector`. |
| **IFrame selector** | CSS selector resolved inside the current context. | When `switchBy = selector` | Trimmed; must match at least one iframe. |
| **Frame index** | Zero-based index of the iframe to enter. | When `switchBy = index` | Must be \>= 0. |
| **Frame name** | Matches the iframe's `name` attribute. | When `switchBy = name` | Case-sensitive match. |
| **URL pattern** | Substring or `/regex/` applied to the iframe URL. | When `switchBy = url` | Same matcher rules as Tab Switch. |
| **Timeout (ms)** | How long to wait when searching for selector/index/name/url targets. | No | Default 30,000 ms; enforced minimum 500 ms. |

`parent` walks one level up the frame stack and `main` clears the stack completely so subsequent selectors run in the top-level document.

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:1366-1406` validates the strategy-specific inputs and emits a Browserless instruction with the normalized mode + value.
2. `api/browserless/cdp/frame_actions.go:12-113` resolves the requested frame via selector/index/name/url and pushes the resulting scope onto a managed stack; `parent` pops once while `main` clears the stack entirely.
3. Subsequent steps automatically reuse the active frame scope because the session calls `s.evalWithFrame` for DOM work, so clicks, extracts, and scripts target the correct iframe until another Frame Switch resets it.

## Example

```json
{
  "type": "frameSwitch",
  "data": {
    "switchBy": "selector",
    "selector": "iframe[data-testid='payment-frame']",
    "timeoutMs": 20000
  }
}
```

## Tips

- Use `main` when you finish interacting with an iframe; otherwise selectors may keep looking inside a stale child context.
- For dynamic embeds (e.g., Stripe, Zendesk), prefer `url` or `name` since DOM positions can shift during feature rollouts.
- Pair with `Wait` nodes to block until the iframe content is ready before drilling in.
- Remember that Frame Switch only changes context; you still need Click/Type nodes to interact with elements inside the frame.

## Related Nodes

- **Tab Switch** - Get to the correct window first, then scope into its frames.
- **Script** - Run scoped JavaScript once the frame is active.
- **Wait / Assert** - Verify that the iframe surfaced the correct app before proceeding.
