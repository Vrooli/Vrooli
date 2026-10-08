# Recording Injection Troubleshooting

The driver installs the recording script with `BrowserContext.addInitScript()`.

## Script did not load

- Confirm recording context initialization completed before navigation.
- Confirm the page belongs to that browser context.
- Check browser console errors and content security policy.
- Verify the generated script contains the expected recording binding name.

## Script loaded but is not ready

Check browser console errors and the `INJECTION_DIAGNOSTICS=true` logs. The init script must finish registering its event handlers before recording can capture events.

## Script is not in the main context

Confirm the browser context supports `addInitScript()` and that the page was navigated after the initializer registered the script. Main-context execution is required for History API event capture.

## Verify an external page

The recording diagnostics route can navigate an initialized page to an external URL and check script load, readiness, and main-context execution. The check requires network access to that page.
