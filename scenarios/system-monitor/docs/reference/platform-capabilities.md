# Platform Capability Matrix

The API uses native collectors. A capability that has no native backend is
reported as `unsupported`; it is never reported as a measured zero.

| Capability | Linux amd64/arm64 | macOS amd64/arm64 | Windows amd64 | First sample and units |
|---|---|---|---|---|
| CPU utilization | native proc/stat | Mach host_processor_info (CGO required) | native Windows backend | First delta is unavailable; percent 0–100. |
| CPU mode breakdown | proc/stat: user, nice, system, idle, iowait, irq, softirq, steal | Mach host_processor_info (CGO required) user/system/idle/nice; explicit refusals for unaccounted modes | GetSystemTimes user/system/idle; explicit refusals for unexposed modes | Percent by mode; unavailable modes carry reasons. |
| CPU pressure stall | /proc/pressure/cpu | explicit unsupported: PSI is Linux-specific | explicit unsupported: PSI is Linux-specific | avg10/avg60/avg300 where measured. |
| CPU saturation | proc/loadavg run queue and normalized load | vm.loadavg normalized load; run queue explicitly unsupported | explicit unsupported: no load-average source | Load is load/core; run queue is process count. |
| Context-switch and interrupt rates | proc/stat counters converted to per-second flows | explicit unsupported in this build | explicit unsupported when PDH is unavailable | First sample and counter reset are not_yet_sampled. |
| Per-core imbalance | proc/stat cpuN plus derived busiest-minus-least index | host_processor_info (CGO required) plus derived index | PDH processor instances plus derived index | Percentage points; no aggregate-only inference. |
| CPU throttling and derate | cgroup/sysfs backends where present | explicit unsupported where backend is absent | explicit unsupported where backend is absent | Refusals name the missing mechanism. |
| Memory utilization | native proc/meminfo | native host memory backend | native Windows backend | Instantaneous percent 0–100. |
| Network counters | native interface/proc data | native interface backend | native interface backend | Counts or interval rates; first rate is unavailable. |
| PCI/USB and platform device identity and binding | shared hostinventory sysfs adapter; Linux platform-bus coverage remains open | shared hostinventory IORegistry adapter, including topology-based SoC identity | shared hostinventory SetupAPI/Configuration Manager (recognized ACPI/SD nodes; partial integrated-device coverage) | Windows amd64/arm64 cross-builds pass; native runtime qualification remains open. Bus quality is separate from device identity. ACPI/SoC nodes remain platform devices. Empty is absence only after a successful probe. Binding does not imply initialized firmware, an OS interface, or verified function. |
| Disk usage | native statfs | native statfs | native volume API | Instantaneous percent 0–100. |
| Process sampling | `/proc` sampler | capability-dependent | capability-dependent | First process rate is unavailable; RSS is bytes. |
| On-demand process read activity | Two bounded `/proc/<pid>/io` observations | explicit unsupported | explicit unsupported | Rows use process start identity and actual per-process elapsed time. Permission failures, exits and the process cap make coverage partial. |
| GPU utilization | vendor/native backend when available | vendor/native backend when available | vendor/native backend when available | Unsupported if no backend; percent and bytes. |

For devices projected from the shared native host inventory, the extensible
`attributes` map retains readiness facts under `readiness_*` keys, with
interface names and per-interface status encoded as JSON. The same projection
retains composite function identity and binding in `native_functions_json`.
Infrastructure Manager transports these attributes unchanged. Unknown,
absent, disabled, disconnected, and unverified states stay distinct; a bound
device driver does not imply firmware, interface, or function readiness.

Cross-builds verify that supported target packages compile. They do not prove
that Windows SetupAPI/Configuration Manager or macOS IORegistry behaves as
observed on a native host. Native smoke tests require a host for the target OS
and architecture. An unavailable host is recorded as unavailable evidence,
not promoted to a support claim.

Permission failures are `failed` with a reason. Missing native support is
`unsupported`. Counter resets and first samples do not produce a rate of zero.

Darwin CPU ticks use the public Mach `host_processor_info` API with
`PROCESSOR_CPU_LOAD_INFO` and four states (user, system, idle, nice). The API
requires a native Darwin SDK/toolchain with CGO enabled; builds with
`CGO_ENABLED=0` explicitly report CPU usage, mode breakdown and per-core metrics
as unsupported. Load averages remain independently available from XNU
`vm.loadavg` (three fixed-point uint32 values and an aligned 64-bit scale).
Darwin has no runnable-process field in that record; run-queue depth is
unsupported, never inferred from load or emitted as a measured zero.

Linux fixtures exercise the XNU ABI and Mach delta calculations. Darwin amd64
and arm64 no-CGO cross-builds qualify compilation of the fallback only. Native
SDK compilation and operated Intel/Apple Silicon qualification remain open.
On each native host, run `VROOLI_DARWIN_TELEMETRY_LIVE=1 go test -v
./internal/collectors -run TestDarwinNativeCPUCounters -count=1` from the API
module with CGO enabled, followed by an operated two-cycle telemetry check.
