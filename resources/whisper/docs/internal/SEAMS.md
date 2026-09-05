# Whisper Resource — Internal Seams

This file enumerates the Go interfaces in the whisper resource CLI that
are intentional testability boundaries. Each entry pairs a seam with
its production wiring and its canonical test fake. New seams should
land here when added; renames should update the row.

## hwprobe.Probe (host-capability detection)

| | |
|---|---|
| **Seam** | The "what hardware is on this host?" question, used by the model recommender. |
| **Interface** | `cli/internal/hwprobe/hwprobe.go::Probe` (`Detect(ctx) (HostCapabilities, error)`) |
| **Production wiring** | `recommend.Default()` constructs a `&hwprobe.SystemProbe{}`. The `SystemProbe` reads CPU via `runtime.NumCPU()`, RAM via stdlib per-OS branches (`/proc/meminfo` on Linux, `sysctl hw.memsize` on Darwin, `wmic ComputerSystem` on Windows), and GPUs via `nvidia-smi --query-gpu` parsed out-of-process. `SystemProbe` also accepts injected readers (`RAMReader`, `GPUReader`, `CPUCount`) so individual probe paths can be tested without OS dependence. |
| **Test fake** | `cli/internal/hwprobe/mocks::FakeProbe` (programmable `Caps`, `Err`). |
| **Why it exists** | The recommender's whole job is to ask "what does this host look like?" and pick a model. Without the seam, every recommender test would need real `/proc/meminfo` and a real nvidia-smi — flaky in CI and impossible across platforms. The `Probe` boundary keeps `recommend.Pick` a pure function with table-driven coverage, and confines the ambient OS reads to one file per platform. |
