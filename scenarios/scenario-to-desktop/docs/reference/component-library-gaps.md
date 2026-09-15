# Scenario To Desktop component-library gaps

This product keeps its specialized workspace frame because the library shell does not
carry the simultaneous workflow regions described in the review record.

```shell-ejection
{"archetype":"navigated-console","reason":"Scenario-to-Desktop is a pipeline generator workspace with inventory, generator, validation, signing, live desktop, captures, and pipeline sidebar state. Its navigation is coupled to persistent pipeline selection and execution progress, which AppShell/2 does not model.","files":["ui/src/components/layout/GeneratorLayout.tsx","ui/src/components/layout/PipelineSidebar.tsx","ui/src/components/layout/SidebarNavigation.tsx"]}
```
