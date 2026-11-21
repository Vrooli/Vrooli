# Accelerator platform fixtures

Captured tool output used to assert the accelerator fact projection on
platforms this repository has no live host for.

| File | Platform | Provenance |
|---|---|---|
| `system_profiler_apple_silicon.txt` | macOS, Apple Silicon | The `system_profiler SPDisplaysDataType` shape already exercised by `TestParseSystemProfilerGPUs` and `TestCollectDarwinGPUs` in `collector_test.go`, extended with the fields the real command emits around the `Chipset Model` line the parser reads. |
| `wmic_video_controller_nvidia.txt` | Windows, NVIDIA discrete | The `wmic path win32_VideoController get name` shape already exercised by `TestCollectWindowsGPUs` in `collector_test.go`, including the CRLF line endings and trailing blank line `wmic` emits. |
| `rocm_smi_showid.txt` | Linux, AMD ROCm | **Declared gap.** No AMD host was available. This file records the documented `rocm-smi --showid` output shape so the parser has a target; the accelerator fact projection does not read it, because `hasROCmDevice` decides from the kernel compute interface plus the device tree, both of which the fixture tests drive directly. |

Live placement verification on macOS and Windows is recorded as `unknown`, never
as a pass. See the plan's cross-platform verification rule.
