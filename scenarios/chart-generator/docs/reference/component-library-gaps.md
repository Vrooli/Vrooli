# Chart Generator shell review

Chart Generator is one chart-building workspace. Its chart-library, styles,
and data tabs switch editor modes within the same preview surface, while the
header owns export actions and product context. These are not application
destinations, so `AppShell/2` would introduce the wrong navigation model.

```shell-ejection
{"archetype":"standalone-document","reason":"Single chart-building workspace whose chart/style/data tabs are in-place editor modes and whose header owns export actions; no application destinations exist to migrate.","files":["ui/src/App.tsx","ui/src/components/chart/chart-preview.tsx","ui/src/components/chart/chart-type-selector.tsx","ui/src/components/chart/data-panel.tsx","ui/src/components/chart/style-selector.tsx"]}
```
