# Vrooli Assistant component-library gaps

This product keeps its specialized frame because the library shell does not
carry the interaction model described in the review record.

```shell-ejection
{"archetype":"standalone-document","reason":"Vrooli Assistant is an assistant conversation surface with desktop overlay, history, settings, and Electron-specific windows. AppShell/2 is not an overlay/desktop-window contract, so the assistant frame remains product-owned.","files":["ui/src/App.tsx","ui/electron/overlay.html","ui/electron/history.html","ui/electron/settings.html"]}
```
