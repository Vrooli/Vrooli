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

## Journal acceptance before transport

AUTH option A source contract (2026-10-05): each original PCM interval is eligible for wire transmission only after its selected journal mode accepts it. Persistent IndexedDB acceptance means transaction completion, not provisional request success, physical fsync or guaranteed browser retention. Explicit bounded memory/reduced mode also accepts before transport and reports reduced durability. If persistent storage fails, the existing reduced-mode compatibility path seeds only committed chunks and accepts the failed interval once in bounded memory before sending. Failed memory acceptance ends capture visibly; no successful frame/done follows it.

Committed cursors advance after acceptance. Appends, processed acknowledgements and discard serialize under the journal owner. Fresh sockets, reconnect replay, stop/done and bounded HTTP recovery obey the same acceptance frontier. Disposal/restart generations fence stale transport and acknowledgement callbacks. Prepared audio plus retained journal data remains within the existing16MiB capture budget; a burst exceeding it stops visibly rather than silently dropping or sending early. The separately bounded whole-turn retry copy retains its existing ceiling.

Default100ms batching and optional20ms first frame preserve exact sample quality/cursors. Disposable delay/abort/pressure fixtures and the accelerated60-minute corpus establish ordering and bounds, not browser p95 latency, physical devices or live-provider readiness. Runtime installation and measured latency/caller qualification remain separate gates. No requirement status is changed solely by this source repair.
