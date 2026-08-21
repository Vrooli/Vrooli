# Platform Capability Matrix

The API uses native collectors. A capability that has no native backend is
reported as `unsupported`; it is never reported as a measured zero.

| Capability | Linux amd64/arm64 | macOS amd64/arm64 | Windows amd64 | First sample and units |
|---|---|---|---|---|
| CPU utilization | native proc/stat | native host CPU backend | native Windows backend | First delta is unavailable; percent 0–100. |
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
