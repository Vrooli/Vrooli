# Audio Tools component-library gaps

Audio Tools uses the library `AppShell/2` for application navigation and responsive chrome. The settings drawer remains a scenario-owned utility because it exposes audio-specific preferences and locale controls.

```shell-ejection
{"archetype":"navigated-console","reason":"SettingsDrawer is an application-specific preferences utility opened from the library shell header. It owns audio provider preferences, font scale, reduced motion, and locale controls; AppShell/2 owns the surrounding navigation and responsive chrome.","files":["ui/src/components/shell/SettingsDrawer.tsx"]}
```
