# Notes component-library gaps

This product keeps its current workspace frame because the library shell does not
carry the product-specific interaction model described in the review record.

```shell-ejection
{"archetype":"standalone-document","reason":"Dual SmartNotes and ZenNotes editor workspace. Its note list, folders, tags, editor header, and Zen mode are simultaneous content interactions; AppShell/2 cannot carry that dual-pane/editor contract without redesigning the product surface.","files":["ui/src/index.html","ui/src/zen-index.html","ui/src/script.js","ui/src/zen-script.js"]}
```
