import { state } from "../state.js";
import { buildWebSocketUrl, textDecoder } from "../utils.js";
import { debugWorkspace } from "./constants.js";
import {
  handleTabAdded,
  handleTabUpdated,
  handleTabRemoved,
  handleActiveTabChanged,
  handleSessionAttached,
  handleSessionDetached,
} from "./tabs.js";
import { applyIdleTimeoutFromServer } from "./idle-timeout.js";

export function connectWorkspaceWebSocket() {
  if (state.workspaceSocket) {
    const { readyState } = state.workspaceSocket;
    if (readyState === WebSocket.OPEN || readyState === WebSocket.CONNECTING) {
      return;
    }
    try {
      state.workspaceSocket.close();
    } catch (_error) {
      // ignore close failures on stale sockets
    }
  }

  const url = buildWebSocketUrl("/ws/workspace/stream");
  const socket = new WebSocket(url);
  state.workspaceSocket = socket;

  socket.addEventListener("open", () => {
    if (state.workspaceSocket !== socket) {
      return;
    }
    if (debugWorkspace) {
      console.log("Workspace WebSocket connected");
    }
    if (state.workspaceReconnectTimer) {
      clearTimeout(state.workspaceReconnectTimer);
      state.workspaceReconnectTimer = null;
    }
  });

  socket.addEventListener("message", async (event) => {
    if (state.workspaceSocket !== socket) {
      return;
    }
    try {
      let rawData = event.data;
      if (rawData instanceof Blob) {
        rawData = await rawData.text();
      } else if (rawData instanceof ArrayBuffer) {
        rawData = textDecoder.decode(rawData);
      }

      const data = JSON.parse(rawData);

      if (
        data.type === "status" &&
        data.payload?.status === "upstream_connected"
      ) {
        return;
      }

      if (data.activeTabId !== undefined && data.tabs !== undefined) {
        if (debugWorkspace) {
          console.log("Received initial workspace state");
        }
        return;
      }

      handleWorkspaceEvent(data);
    } catch (error) {
      console.error("Failed to parse workspace event:", error);
    }
  });

  socket.addEventListener("close", () => {
    if (state.workspaceSocket !== socket) {
      return;
    }
    if (debugWorkspace) {
      console.log("Workspace WebSocket closed, reconnecting...");
    }
    state.workspaceSocket = null;
    if (!state.workspaceReconnectTimer) {
      state.workspaceReconnectTimer = setTimeout(() => {
        connectWorkspaceWebSocket();
      }, 3000);
    }
  });

  socket.addEventListener("error", (error) => {
    if (state.workspaceSocket !== socket) {
      return;
    }
    console.error("Workspace WebSocket error:", error);
  });
}

function handleWorkspaceEvent(event) {
  if (!event || !event.type) return;

  switch (event.type) {
    case "workspace-full-update":
      if (debugWorkspace) {
        console.log("Full workspace update:", event.payload);
      }
      break;
    case "tab-added":
      handleTabAdded(event.payload);
      break;
    case "tab-updated":
      handleTabUpdated(event.payload);
      break;
    case "tab-removed":
      handleTabRemoved(event.payload);
      break;
    case "active-tab-changed":
      handleActiveTabChanged(event.payload);
      break;
    case "session-attached":
      handleSessionAttached(event.payload);
      break;
    case "session-detached":
      handleSessionDetached(event.payload);
      break;
    case "idle-timeout-changed":
      if (
        event.payload &&
        typeof event.payload.idleTimeoutSeconds === "number"
      ) {
        applyIdleTimeoutFromServer(event.payload.idleTimeoutSeconds);
      }
      break;
    case "keyboard-toolbar-mode-changed":
      if (debugWorkspace) {
        console.log(
          "Keyboard toolbar mode changed event (deprecated):",
          event.payload,
        );
      }
      break;
    default:
      if (debugWorkspace) {
        console.log("Unknown workspace event:", event.type);
      }
  }
}
