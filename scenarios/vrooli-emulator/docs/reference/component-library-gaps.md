# Vrooli Emulator component-library gaps

This product keeps its specialized frame because the library shell does not
carry the interaction model described in the review record.

```shell-ejection
{"archetype":"navigated-console","reason":"Vrooli Emulator is a session list/detail and live-device workspace with bridge state, noVNC interaction, and mobile panel transitions. AppShell/2 cannot carry the live emulator surface and session controls as generic navigation.","files":["ui/src/App.tsx","ui/src/pages/SessionListPage.tsx","ui/src/pages/SessionDetailPage.tsx"]}
```
