# Picker Wheel shell review

Picker Wheel is one interactive decision workspace. The wheel canvas is the
primary surface; its preset, custom, saved, and history tabs switch modes
inside that workspace, and the settings panel controls the same experience.
They are not application destinations, so `AppShell/2` would add the wrong
navigation model. Keep the workspace frame scenario-owned as a standalone
document until a compatible tool-workspace archetype exists.

```shell-ejection
{"archetype":"standalone-document","reason":"Single wheel workspace whose preset/custom/saved/history tabs and settings panel are in-place tool modes, not application destinations; AppShell/2 navigation would replace the product interaction model.","files":["ui/index.html","ui/script.js"]}
```
