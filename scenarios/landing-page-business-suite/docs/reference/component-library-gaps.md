# Landing Page Business Suite component-library gaps

This product keeps its specialized frame because the library shell does not
carry the interaction model described in the review record.

```shell-ejection
{"archetype":"navigated-console","reason":"Landing Page Business Suite combines public landing, user-auth, and admin surfaces with separate navigation and billing/content workflows. A single AppShell/2 frame would cross product boundaries and discard auth/marketing shell behavior.","files":["ui/src/App.tsx","ui/src/shared/ui/AuthPageLayout.tsx","ui/src/app/routes/adminRoutes.tsx","ui/src/app/routes/publicRoutes.tsx"]}
```
