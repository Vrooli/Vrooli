# Recording Injection Architecture

Browser recording is installed through one context-level mechanism: `context.addInitScript()`.

```text
RecordingContextInitializer
  ├─ InitScriptInjectionStrategy
  │    └─ context.addInitScript(generated recording script)
  └─ EventRouteManager
       └─ page-level recording event route
```

The init script runs in the main page context before application code. This lets it capture DOM events and wrap History API navigation while remaining dormant until a recording session activates it.

`InjectionStrategyName` is the literal `init-script`. The narrow injector contract supports script registration, verification, lifecycle cleanup, and diagnostic counters. No runtime provider selection or configuration key chooses an injection implementation.

`INJECTION_DIAGNOSTICS=true` enables verbose logging. It does not change injection behavior.
