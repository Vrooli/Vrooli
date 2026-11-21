# Component library gaps

The routed paired experience uses the library-owned `AppShell/2`. The first-run pairing gate is a separate full-viewport experience shown before a device session exists, so it remains scenario-owned until a library archetype can represent that onboarding flow without implying a paired navigation shell.

```shell-ejection
{"archetype":"navigated-console","reason":"The first-run pairing gate is a pre-session full-viewport onboarding experience. Mounting the navigated console shell would expose paired navigation before a device session exists; the scenario-owned flow is retained with its own setup and join controls.","files":["ui/src/features/onboarding/OnboardingScreen.tsx"]}
```
