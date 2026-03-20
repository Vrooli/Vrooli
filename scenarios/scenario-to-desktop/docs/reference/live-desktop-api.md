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

## GET `/api/v1/livedesktop/sessions/{id}/ws`

WebSocket endpoint for VNC proxy. The browser's noVNC client connects here. Binary VNC frames are proxied bidirectionally to the websockify instance.

This is a WebSocket upgrade endpoint — not a regular HTTP endpoint.
