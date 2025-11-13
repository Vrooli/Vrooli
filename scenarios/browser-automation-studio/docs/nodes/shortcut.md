# Shortcut Node

`[REQ:BAS-NODE-SHORTCUT-DISPATCH]` sends high-level keyboard shortcuts (Ctrl/Cmd combos, sequences) to the active page, making it easy to trigger in-app commands.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Shortcut(s)** | Array of shortcuts to fire sequentially | Yes | Each entry can be a string (`Ctrl+S`) or a sequence array.
| **Delay (ms)** | Wait between shortcuts | No | Spread out combos for apps that need time to respond.
| **Timeout (ms)** | Wait for the target element before focusing | No | Defaults to 30 000.
| **Focus selector** | Optional element to focus before sending shortcuts | No | Ensures shortcuts apply to the right context.

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:858-911` parses the config, normalizes arrays, and validates that at least one shortcut or key sequence is provided.
2. Browserless focuses the optional element and dispatches the requested key combinations using Chrome DevTools `Input.dispatchKeyEvent`, respecting `delayMs` where provided.
3. Telemetry logs each shortcut so Execution Viewer shows what combos were sent.

## Example

```json
{
  "type": "shortcut",
  "data": {
    "shortcuts": ["Meta+K", "Enter"],
    "delayMs": 150,
    "focusSelector": "body"
  }
}
```

## Tips

- Use shortcut nodes for generic commands (save, open command palette); use Keyboard nodes when you need low-level keydown/up control.
- Focus selectors are especially helpful for rich editors that swallow shortcuts when the wrong container is active.
- Combine with Wait nodes if the shortcut triggers navigation or heavy UI updates.

## Related Nodes

- **Keyboard** – Fine-grained key events.
- **Focus** – Ensure the correct component has focus before firing shortcuts.
