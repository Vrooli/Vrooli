# Lifestyle Dashboard component-library gaps

`AppShell/2` owns the application frame, route navigation, health status, and
refresh utility. Settings still contains two feature-owned viewport panels for
editing lifestyle preferences; those panels are page behavior inside the main
pane, not a second application shell. Keep them under a scoped ejection until
the library has a settings-workspace primitive.

```shell-ejection
{"archetype":"navigated-console","reason":"SettingsPage owns two preference editor viewport panels inside the routed main pane; they are feature behavior rather than application chrome and remain scenario-owned under the governed AppShell/2 frame.","files":["ui/src/pages/SettingsPage.tsx"]}
```
