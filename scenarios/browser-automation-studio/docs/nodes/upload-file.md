# Upload File Node

`[REQ:BAS-NODE-UPLOAD-FILE]` attaches local files to `<input type="file">` elements and handles multi-file uploads without brittle manual scripts.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Target selector** | CSS selector for the file input | Yes | Must point to a genuine file input; custom widgets still need scripts.
| **File paths** | Absolute paths (one per line) | Yes | Paths are resolved on the machine running Browserless.
| **Timeout (ms)** | Wait for the input | No | Defaults to 30 000; min 500.
| **Wait after (ms)** | Delay after attaching files | No | Helpful for apps that validate immediately.

## Runtime Behavior

1. `api/browserless/runtime/instructions.go:1505-1567` trims selectors, validates file lists, and normalizes timeouts/waits.
2. Browserless resolves the selector, verifies it’s a file input, and calls `chromedp.SetUploadFiles`, pointing to the provided paths.
3. Execution artifacts store the selector and file count (paths themselves are not logged for security).

## Example

```json
{
  "type": "uploadFile",
  "data": {
    "selector": "input[type=file]",
    "filePaths": [
      "/home/user/Documents/spec.pdf",
      "/home/user/Documents/screenshot.png"
    ],
    "waitForMs": 500
  }
}
```

## Tips

- Ensure the files exist on the execution host; missing files cause immediate failures.
- Use Wait or Assert nodes afterwards to confirm the UI registers the attachments.
- For drag-and-drop uploaders, combine this node with Script/Drag nodes that simulate dropping the file element.

## Related Nodes

- **Click** – Open file dialogs before uploading if the input is hidden.
- **Set/Use Variable** – Store the file path list if you need to reuse it later.
