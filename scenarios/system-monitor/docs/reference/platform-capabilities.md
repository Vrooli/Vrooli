# Platform Capability Matrix

The API uses native collectors. A capability that has no native backend is
reported as `unsupported`; it is never reported as a measured zero.

| Capability | Linux amd64/arm64 | macOS amd64/arm64 | Windows amd64 | First sample and units |
|---|---|---|---|---|
| CPU utilization | native proc/stat | native host CPU backend | native Windows backend | First delta is unavailable; percent 0–100. |
| CPU mode breakdown | proc/stat: user, nice, system, idle, iowait, irq, softirq, steal | kern.cp_time user/nice/system/idle; explicit refusals for unaccounted modes | GetSystemTimes user/system/idle; explicit refusals for unexposed modes | Percent by mode; unavailable modes carry reasons. |
| CPU pressure stall | /proc/pressure/cpu | explicit unsupported: PSI is Linux-specific | explicit unsupported: PSI is Linux-specific | avg10/avg60/avg300 where measured. |
| CPU saturation | proc/loadavg run queue and normalized load | vm.loadavg normalized load; run queue explicitly unsupported | explicit unsupported: no load-average source | Load is load/core; run queue is process count. |
| Context-switch and interrupt rates | proc/stat counters converted to per-second flows | explicit unsupported in this build | explicit unsupported when PDH is unavailable | First sample and counter reset are not_yet_sampled. |
| Per-core imbalance | proc/stat cpuN plus derived busiest-minus-least index | host_processor_info plus derived index | PDH processor instances plus derived index | Percentage points; no aggregate-only inference. |
| CPU throttling and derate | cgroup/sysfs backends where present | explicit unsupported where backend is absent | explicit unsupported where backend is absent | Refusals name the missing mechanism. |
| Memory utilization | native proc/meminfo | native host memory backend | native Windows backend | Instantaneous percent 0–100. |
| Network counters | native interface/proc data | native interface backend | native interface backend | Counts or interval rates; first rate is unavailable. |
| Disk usage | native statfs | native statfs | native volume API | Instantaneous percent 0–100. |
| Process sampling | `/proc` sampler | capability-dependent | capability-dependent | First process rate is unavailable; RSS is bytes. |
| GPU utilization | vendor/native backend when available | vendor/native backend when available | vendor/native backend when available | Unsupported if no backend; percent and bytes. |

Cross-builds verify that supported target packages compile. Native smoke tests
require a host for the target OS and architecture. An unavailable host is
recorded as unavailable evidence, not promoted to a support claim.

Permission failures are `failed` with a reason. Missing native support is
`unsupported`. Counter resets and first samples do not produce a rate of zero.
