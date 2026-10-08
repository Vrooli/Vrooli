# App Monitor component-library gaps

App Monitor is an ambient resource monitor whose primary interaction is a draggable floating controller. The controller opens tab, action, and workspace overlays and changes behavior by the active preview route; it is not application destination navigation.

```shell-ejection
{"archetype":"navigated-console","reason":"App Monitor's floating controller is a product-specific overlay router for tabs, actions, and preview workspace state. AppShell/2 navigation would replace the draggable controller and break the route/overlay interaction contract, so the controller remains scenario-owned until a library archetype supports this interaction model.","files":["ui/src/components/Shell.tsx"]}
```
