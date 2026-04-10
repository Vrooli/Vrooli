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
    "sample_count": 42
  }
}
```

The `metrics` field is `null` when no monitor is active (e.g., app not launched, or `monitorFactory` not configured).

---

## GET `/api/v1/livedesktop/sessions/{id}/ws`

WebSocket endpoint for VNC proxy. The browser's noVNC client connects here. Binary VNC frames are proxied bidirectionally to the websockify instance.

This is a WebSocket upgrade endpoint — not a regular HTTP endpoint.
