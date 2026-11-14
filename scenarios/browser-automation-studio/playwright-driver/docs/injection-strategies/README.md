# Recording Injection

The recording script has one supported injection path: `BrowserContext.addInitScript()`.
The recording context initializer registers the generated script once, before page navigation. It then remains dormant until recording activation arrives over the exposed binding.

## Runtime path

1. `RecordingContextInitializer.initialize()` creates `InitScriptInjectionStrategy`.
2. The injector calls `context.addInitScript()` with the generated recording script.
3. New documents run the script in the page main context before application scripts.
4. The script reports events through the recording binding and page event route.
5. Injection counters and `INJECTION_DIAGNOSTICS` support runtime diagnostics; they do not select an injection path.

## Diagnostics

`RecordingContextInitializer` exposes the init-script name, injection counters, route counters, and an optional first-page sanity check. `runExternalUrlInjectionTest` navigates an initialized page and verifies script load, readiness, and main-context execution.

Run the focused driver checks from `playwright-driver`:

```sh
pnpm exec jest --runTestsByPath tests/recording/injection/factory.test.ts tests/recording/injection/strategies.test.ts tests/unit/recording/context-initializer.test.ts --coverage=false
pnpm typecheck
```
