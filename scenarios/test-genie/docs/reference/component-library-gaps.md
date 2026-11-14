# Test Genie component-library gaps

This product keeps its specialized frame because the library shell does not
carry the interaction model described in the review record.

```shell-ejection
{"archetype":"navigated-console","reason":"Test Genie is a test-run operations console whose scenario/run/requirement subtabs, breadcrumbs, and execution controls share run context. AppShell/2 cannot own that nested validation navigation without changing the test product contract.","files":["ui/src/App.tsx","ui/src/components/layout/TabNav.tsx","ui/src/components/layout/ScenarioDetailTabNav.tsx","ui/src/components/layout/SubtabNav.tsx"]}
```
