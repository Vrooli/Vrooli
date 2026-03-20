# Interactive Desktop Guide

The Interactive Desktop feature lets you control a virtual desktop from your browser to manually validate generated Electron apps. Instead of passively watching a smoke test recording, you interact with the app in real time.

## Architecture

```
Browser (noVNC in React)
    → WebSocket → Go API proxy
    → websockify (VNC→WS bridge)
    → x11vnc (VNC server)
    → Xvfb virtual display
    → Electron app
```

## Prerequisites

- **x11vnc** and **websockify** installed. Install via the VNC resource:
  ```bash
  vrooli resource install vnc
  # or manually:
  sudo apt-get install x11vnc websockify
  ```
- **Xvfb** installed (already required for smoke tests)
- A built Electron app (build stage must have completed)

## Usage

### From the Pipeline UI

1. Run the pipeline through at least the build stage
2. In the **Smoke Test** section, click **Launch Interactive Desktop**
3. Configure resolution (default 1280x720) and click **Start Session**
4. The noVNC canvas connects to the virtual desktop
5. Interact with the desktop using mouse and keyboard
6. Click **Launch App** to start the Electron app on the virtual display
7. Click **Stop Session** when done

### From the Apps View

1. Navigate to the **Apps** tab
2. Click on an app to open its detail drawer
3. Click **Open Desktop** in the Interactive Desktop section

## Session Lifecycle

- **Creating**: Xvfb display is being started, VNC toolchain launching
- **Running**: Desktop is active and accepting connections
- **Stopping**: Tearing down VNC processes and display
- **Stopped**: All resources cleaned up
- **Error**: Something went wrong (check error message)

## Idle Timeout

Sessions are automatically reaped after 30 minutes of inactivity. The browser sends heartbeat pings every 30 seconds while connected to keep the session alive.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/livedesktop/sessions` | Start a new session |
| GET | `/api/v1/livedesktop/sessions` | List all sessions |
| GET | `/api/v1/livedesktop/sessions/{id}` | Get session details |
| POST | `/api/v1/livedesktop/sessions/{id}/heartbeat` | Keep session alive |
| POST | `/api/v1/livedesktop/sessions/{id}/launch` | Launch an app |
| DELETE | `/api/v1/livedesktop/sessions/{id}` | Stop a session |
| GET | `/api/v1/livedesktop/sessions/{id}/ws` | WebSocket VNC proxy |

## Troubleshooting

### "VNC backend unavailable"
- Check that x11vnc and websockify are installed: `which x11vnc websockify`
- Check if ports 5900-5999 and 6080-6180 are available

### Black screen in VNC canvas
- The window manager may not have started. Check API logs for WM errors
- Try launching the Electron app — it should appear once started

### Connection drops frequently
- Check network stability
- The heartbeat interval is 30s — if the browser tab is backgrounded, some browsers throttle timers

### Session stuck in "creating" state
- Xvfb may have failed to start. Check for `/tmp/.X*-lock` files
- Ensure sufficient system resources (memory, /tmp space)
