# Apple hardware activation boundary

The iOS ramp is useful on Linux while remaining honest about Apple-only work.
These files are the complete hardware boundary:

- `api/internal/targets/prober.go` reports Linux as unsupported for the native
  simulator and names the missing macOS bridge capability.
- `api/internal/builds/builder.go` generates the project on Linux, then stops
  at `xcodebuild` with an unavailable disposition until macOS is available.
- `api/internal/journeys/plan.go` declares the twelve chapters and returns
  unavailable results until `simctl`/WebDriverAgent is available.
- `api/internal/distribution/distributor.go` keeps signing and channels
  independently unavailable until their Apple prerequisites exist.

No matrix, CLI, UI, fixture, or evidence layer may bypass these adapters or
turn an unavailable Apple operation into a pass. Register a trusted macOS
bridge with Xcode and simulator tooling, then provide Apple signing
references through the governed secrets path. Key bytes must never enter this
repository, generated project, or evidence.
