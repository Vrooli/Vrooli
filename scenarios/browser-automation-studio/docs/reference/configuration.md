# Configuration

## Environment variables

`ENGINE=playwright` selects the supported driver. `PLAYWRIGHT_DRIVER_URL` points to an existing driver; otherwise lifecycle supervision starts the local sidecar. `PLAYWRIGHT_CHROMIUM_PATH` can select an existing Chromium binary.

## Service manifest (`.vrooli/service.json`)

The scenario service manifest is the lifecycle and resource authority. Change it with scenario lifecycle conventions, not ad-hoc process commands.

## Schema bootstrap

The API initializes registered schema domains through routed connections. Use migrations/registry ownership for schema changes.

## CLI config file

Use the scenario CLI through the Vrooli control plane; keep credentials out of tracked configuration.

## API-base resolution precedence

The driver URL explicit environment variable takes precedence over the supervised local sidecar.

## Test/CI configuration

Run `vrooli scenario test browser-automation-studio`; Test Genie owns the run and its baseline comparison.

## Cross-references

- [Service manifest](../../.vrooli/service.json)
- [Operations](../operations/RUNBOOK.md)


## Product capture inputs and measured media

`browser-automation-studio capture --url <target> --interaction-flow-file take.json
--capture video,screenshot,console-logs,network --out <take-root>` uses the existing
CaptureService action graph after navigation. The CLI reads at most 2 MiB of JSON;
the server owns graph validation. Use wait/assert nodes around the real action and
result. The service exports into an execution-specific directory under the chosen
root, so separate takes retain their own logs, screenshots and video. Keep returned
execution/artifact references; a reused filename is not an artifact identity.

Video artifact metadata separates `requested_css_width`, `requested_css_height`
and optional `requested_device_scale_factor` from `measured_width`,
`measured_height`, `measured_duration_ms`, and `measurement_status`. Capture and
export use the same bounded media probe. An unavailable probe leaves raw media
available with unknown measurements. It never invents dimensions from DPR.
Continuous browser video can remain at CSS resolution while screenshots use
physical pixels. The device-scale live-stream path uses SDK screenshot polling;
its cadence is different from continuous video. Choose and qualify the source
needed by the edit. Artifact measurement does not certify visual quality.

`--routed-test-mode` attaches `X-Vrooli-Test-Mode: 1` to browser target requests
through the browser profile. Install and retain the target's paired SQL/file
routing lease first. This flag does not create a lease or route BAS's own capture
records. Verify primary-versus-leased effects and clean up target fixtures before
clearing the lease. WebM video artifacts are published as `video/webm`, independent
of the host MIME database.
