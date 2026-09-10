# Component-library gaps

Flow Verifier now uses `AppShell/2` for application navigation. The scenario
detail page retains its local tab navigation because it switches between the
feature's overview, graph, trace, and history surfaces inside one route; it is
feature navigation rather than a second application shell.

```shell-ejection
{"archetype":"navigated-console","reason":"ScenarioDetailPage's nav is an in-page feature tab set for one flow detail route, not application navigation. AppShell/2 owns the application destinations; keeping these tabs with the detail feature preserves the flow analysis interaction.","files":["ui/src/pages/ScenarioDetailPage.tsx"]}
```
