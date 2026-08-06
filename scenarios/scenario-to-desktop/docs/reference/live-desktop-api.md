# Live Desktop API Reference

## POST `/api/v1/livedesktop/sessions`

Start a new interactive desktop session.

**Request Body:**
```json
{
  "scenario_name": "my-scenario",
  "width": 1280,
  "height": 720,
  "app_path": "/path/to/electron/app"
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| scenario_name | string | yes | — | Scenario this session is for |
| width | int | no | 1280 | Display width in pixels |
| height | int | no | 720 | Display height in pixels |
| app_path | string | no | — | Optional app to auto-launch |

**Response (201):**
```json
{
  "id": "uuid",
  "scenario_name": "my-scenario",
  "state": "running",
  "vnc_port": 5900,
  "ws_port": 6080,
  "width": 1280,
  "height": 720,
  "created_at": "2026-03-20T12:00:00Z",
  "last_heartbeat": "2026-03-20T12:00:00Z"
}
```

---

## GET `/api/v1/livedesktop/sessions`

List all sessions.

**Response (200):** Array of session objects.

---

## GET `/api/v1/livedesktop/sessions/{id}`

Get session details.

**Response (200):** Session object. **404** if not found.

---

## POST `/api/v1/livedesktop/sessions/{id}/heartbeat`

Keep a session alive. Called automatically by the UI every 30 seconds.

**Response (200):** `{"status": "ok"}`

---

## POST `/api/v1/livedesktop/sessions/{id}/launch`

Launch an application on the session's virtual display.

**Request Body:**
```json
{
  "app_path": "/path/to/electron/app"
}
```

**Response (200):** `{"status": "launched"}`

---

## DELETE `/api/v1/livedesktop/sessions/{id}`

Stop and clean up a session.

**Response (200):** `{"status": "stopped"}`

---

## GET `/api/v1/livedesktop/sessions/{id}/metrics` {#process-metrics}

Returns the full process metrics report for a session, including all resource samples and startup timing. This is the detailed endpoint for charting and analysis — the `GET /sessions/{id}` response includes a lightweight `metrics` summary instead.

**Response (200) — when monitor is active:**
```json
{
  "startup": {
    "launch_at": "2026-03-20T12:00:00Z",
    "splash_visible_at": "2026-03-20T12:00:00.5Z",
    "splash_duration_ms": 500,
    "ready_at": "2026-03-20T12:00:01.2Z",
    "ready_ms": 1200
  },
  "samples": [
    {
      "timestamp": "2026-03-20T12:00:01Z",
      "cpu_percent": 25.5,
      "rss_bytes": 157286400,
      "peak_bytes": 209715200,
      "threads": 8
    }
  ],
  "summary": {
    "peak_rss_bytes": 209715200,
    "avg_rss_bytes": 157286400,
    "peak_cpu_percent": 45.2,
    "avg_cpu_percent": 18.7,
    "max_threads": 12,
    "sample_count": 60,
    "duration_ms": 60000
  }
}
```

**Response (200) — when no monitor is running:**
```json
{"status": "no_monitor"}
```

**Session View `metrics` field** (included in `GET /sessions/{id}` and `POST /sessions` responses):
```json
{
  "metrics": {
    "splash_duration_ms": 500,
    "splash_detected": true,
    "ready_duration_ms": 1200,
    "ready_detected": true,
    "current_cpu_percent": 12.3,
    "current_rss_mb": 150.0,
    "peak_rss_mb": 200.0,
    "sample_count": 42,
    "process_roles": [
      {
        "role": "electron_main",
        "available": true,
        "process_count": 1,
        "rss_bytes": 157286400,
        "peak_rss_bytes": 209715200,
        "sample_count": 42
      },
      {
        "role": "electron_gpu",
        "available": false,
        "unsupported": true
      }
    ]
  }
}
```

The `metrics` field is `null` when no monitor is active (e.g., app not launched, or `monitorFactory` not configured).

Process attribution is scoped to the Linux `/proc` process tree rooted at the
launched application. `process_count` counts unique PIDs observed during the
run; RSS and thread values are the current aggregate at the last sample, with
peak values retained separately. `unsupported: true` means the platform
adapter cannot provide that role, while `available: false` means the role was
supported but not observed. Consumers must preserve those states instead of
rendering them as zero.

## Launch-performance evidence

Smoke-test runs persist two separate, redacted launch traces beside the
recording: a protocol trace and a demo trace. Each trace uses the
`launch-trace.v1` schema and includes a run identity, monotonic timestamps,
wall-clock timestamps, component/role attribution, and lifecycle events for:

- recorder/protocol or demo start and completion;
- Electron readiness and splash creation, load, show, ready-to-show, and first
  usable paint;
- bundled-runtime spawn, token, health, readyz, and port discovery;
- scenario-server readiness; and
- main-window creation, load, show, and application readiness.

The producer manifest references the checksummed trace/profile artifacts and
projects named phase durations. Missing or malformed traces make performance
evidence `degraded` or `unavailable`; they cannot become a passing capability
gate. Optional Chromium, main-process CPU, and heap profiling is enabled only
with `S2D_PROFILE_MODE=chromium|cpu|heap|all`; default `disabled` creates no
profile artifacts.

For repeated cold or warm runs, aggregate only comparable traces (same host
fingerprint, artifact digest, display, deployment mode, and profiler mode).
Use nearest-rank p50 and p95, retain min/max/spread, and keep an explicit
non-comparable result when host or artifact identity changes. The reference
Linux Xvfb/openbox review budget is advisory until a stable baseline exists:
process-to-splash-first-paint p95 ≤ 1 second.

---

## GET `/api/v1/livedesktop/sessions/{id}/ws`

WebSocket endpoint for VNC proxy. The browser's noVNC client connects here. Binary VNC frames are proxied bidirectionally to the websockify instance.

This is a WebSocket upgrade endpoint — not a regular HTTP endpoint.
