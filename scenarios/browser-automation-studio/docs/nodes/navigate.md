# Navigate Node

`[REQ:BAS-NODE-NAVIGATE]` covers the first step in most workflows: loading an arbitrary URL or reusing one of your locally managed scenarios/apps. The node also owns preview capture so builders can validate destinations before running a workflow end-to-end.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Destination type** | `url` or `scenario` | No (auto-detected) | Selecting “App” switches the form to scenario selection and enables scenario port resolution.
| **URL** | Absolute URL to open | Yes (when destination = url) | Accepts https/http/file/etc. Normalized via `InstructionParam.URL`.
| **Scenario name** | Scenario slug as registered with the Vrooli lifecycle | Yes (when destination = scenario) | Populated from `useScenarioStore`. The runtime resolves the app URL/port before navigating.
| **Scenario path** | Optional path appended to the resolved base URL | No | Supports absolute or relative paths; normalization mirrors the runtime’s `ScenarioPath` handling.
| **Wait until** | `load`, `domcontentloaded`, `networkidle` | No | Maps to the Chrome DevTools Protocol `Page.navigate` `waitUntil` option via Browserless.
| **Timeout (ms)** | Navigation timeout | No | Defaults to 30 000 ms; enforced in `runtime/instructions.go`.
| **Post-wait (ms)** | Extra delay after load | No | Useful for heavy SPAs before continuing to the next node.

### Preview + Console
- **Preview panel** fetches a screenshot using the active workflow viewport (with sane clamps of 200-10 000 px).
- **Console tab** streams recent logs from the preview request, helping debug CSP/auth issues without running the full workflow.
- Preview requests respect the `scenarioPath`, so broken deep links are caught immediately.

## Runtime Behavior

1. The compiler emits a `navigate` step with the raw node data.
2. `runtime/instructions.go:675-716` normalizes the config, resolves scenario URLs when needed, and sets `InstructionParam.URL`, `Scenario`, `ScenarioPath`, and wait/timeout fields.
3. The Browserless session performs the actual navigation, waiting for the requested lifecycle event before releasing the workflow to subsequent nodes.
4. Execution artifacts capture the resolved URL, response timing, and console/network telemetry for replay/debugging.

## Examples

### Go to a SaaS dashboard
```json
{
  "type": "navigate",
  "data": {
    "destinationType": "url",
    "url": "https://app.example.com/login",
    "waitUntil": "networkidle",
    "timeoutMs": 45000,
    "waitForMs": 1500
  }
}
```

### Jump into a local scenario after launching it via lifecycle
```json
{
  "type": "navigate",
  "data": {
    "destinationType": "scenario",
    "scenario": "system-monitor",
    "scenarioPath": "/investigations/new"
  }
}
```

The runtime resolves `system-monitor` to its assigned localhost port and appends `/investigations/new` before navigating.

## Tips & Limitations

- Launch scenarios with `make start` / `vrooli scenario start <name>` so the Navigate node can resolve ports. Offline apps return an error before execution begins.
- For authenticated SaaS targets, consider adding a `Wait` node after navigation to ensure login forms finish animating before `Type` nodes fire.
- The preview screenshot shares throttling with the main workflow; avoid spamming the refresh button while executions are running to keep resource usage predictable.
- Scenario navigation inherits the parent workflow’s viewport; coordinate with Rotate/Gesture nodes when testing responsive breakpoints.

## Related Nodes

- **Wait**: Delay until specific elements appear after navigation.
- **Set Variable**: Capture resolved URLs for logging or branches.
- **Frame/Tab Switch**: Use immediately after navigation when flows spawn popups or embed content.
