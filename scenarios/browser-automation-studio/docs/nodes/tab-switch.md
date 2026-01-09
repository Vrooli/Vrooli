# Tab Switch Node

`[REQ:BAS-NODE-TAB-SWITCH]` lets Vrooli Ascension workflows juggle OAuth popups, report windows, or any multi-tab interaction without brittle scripting.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Switch strategy** | How the next tab is chosen: `newest`, `oldest`, `index`, `title`, or `url`. | No | Defaults to `newest` if left blank. |
| **Tab index** | Zero-based index of the target tab. | When `switchBy = index` | Must be \>= 0; validated by the workflow validator. |
| **Title pattern** | Substring or `/regex/` that must match the tab title. | When `switchBy = title` | Trimmed before execution. |
| **URL pattern** | Substring or `/regex/` that must match the tab URL. | When `switchBy = url` | Helpful for OAuth callback URLs. |
| **Wait for new tab** | Blocks until a brand-new tab appears before switching. | No | Detects popups spawned by the previous node. |
| **Close previous tab** | Closes the tab that was active before the switch. | No | Keeps long scenarios from leaking tabs. |
| **Timeout (ms)** | How long to wait for targets and popups. | No | Defaults to 30,000 ms; UI enforces a 1,000 ms minimum. |

## Runtime Behavior

1. The automation compiler/executor forwards the mode/index/title/url/wait/close flags as-authored; validation is handled by the workflow validator/UI.
2. `api/browserless/cdp/evaluation_actions.go:92-155` keeps a live inventory of Chrome targets, optionally waits for a brand-new tab, picks one based on the configured strategy, and switches focus. When `closeOld` is set it also disposes the previous target to free memory.
3. Execution artifacts capture the active tab's title/URL and the `targetId`, making it easy to debug which window the workflow selected.

## Example

```json
{
  "type": "tabSwitch",
  "data": {
    "switchBy": "title",
    "titleMatch": "Authorize",
    "waitForNew": true,
    "timeoutMs": 45000,
    "closeOld": true
  }
}
```

## Tips

- Use `waitForNew` when the previous step triggers a popup (OAuth, report preview, payment confirmation), otherwise the workflow may switch too early.
- Regular expressions must be wrapped in slashes (e.g., `/callback\\/success/`). Anything else is treated as a case-insensitive substring match.
- Closing the previous tab keeps Browserless from accumulating zombie targets during long regression runs.
- Combine with `Wait` or `Assert` nodes right after switching to confirm the new tab loaded the correct route.

## Related Nodes

- **Navigate** - Open the page that spawns the extra tab before switching.
- **Frame Switch** - Once inside the right tab, scope into iframes cleanly.
- **Wait** - Guard against slow-loading popups.
- **Workflow Call** - Move the popup interaction into a reusable sub-workflow and invoke it after the tab switch.
