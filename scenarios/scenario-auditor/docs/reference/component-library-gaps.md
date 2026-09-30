# Scenario Auditor component-library gaps

This product keeps its specialized frame because the library shell does not
carry the interaction model described in the review record.

```shell-ejection
{"archetype":"navigated-console","reason":"Scenario Auditor is an operations console whose sidebar routes, active-agent controls, vulnerability scanner, automated fixes, and settings share audit state and stop-agent actions. AppShell/2 cannot own those coupled remediation controls without redesigning the audit product.","files":["ui/src/App.tsx"]}
```
