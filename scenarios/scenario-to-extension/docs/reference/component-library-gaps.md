# Scenario-to-Extension shell review

Scenario-to-Extension is one extension-generation workflow. Generator, Builds,
Templates, and Testing are workflow stages inside that tool, not application
destinations. The header status control and stage tabs are therefore
scenario-owned workspace chrome; `AppShell/2` would impose a routed console
model that the workflow does not have.

```shell-ejection
{"archetype":"standalone-document","reason":"Single extension-generation workflow whose Generator/Builds/Templates/Testing tabs are in-place stages rather than application destinations; AppShell/2 would impose the wrong navigation model.","files":["ui/index.html","ui/app.js"]}
```
