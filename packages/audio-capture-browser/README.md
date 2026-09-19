# Browser audio capture client

`@vrooli/audio-capture-browser` owns the browser-side audio protocol client,
signal processing, and capture lifecycle state machines used by scenario UIs.
`audio-tools` is the steward: its STT WebSocket handler defines the ATV2 wire
contract that this package implements, and the package's conformance tests run
against that contract.

Presentation belongs to `react-component-library` and reaches consumers through
copy-adoption. Transport adapters and scenario-specific API/proto mapping remain
inside each consuming scenario. The browser client cannot live under
`scenarios/audio-tools/clients/` because TypeScript scenario UIs are intentionally
isolated from one another; the governed `file:` package boundary is the supported
cross-scenario adoption mechanism.

Run the package's tests with `pnpm test` or through the control plane:

```bash
vrooli package test audio-capture-browser
```

The package is a standalone pnpm workspace. Its lockfile and test dependencies
are package-owned; install dependency changes through
`scenario-dependency-analyzer deps install` and keep scenario consumers on the
governed `file:` adoption boundary.
