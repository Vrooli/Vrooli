// @ts-check

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
  sanitizeWorkspaceTabsFromServer,
  reportWorkspaceAnomaly,
  applyWorkspaceSnapshot,
} from "./tabs.js";
import { applyIdleTimeoutFromServer } from "./idle-timeout.js";

/** @typedef {import("../types.d.ts").WorkspaceEventEnvelope} WorkspaceEventEnvelope */
/** @typedef {import("../types.d.ts").WorkspaceTabSnapshot} WorkspaceTabSnapshot */

/**
 * Establish a workspace WebSocket connection and register lifecycle handlers.
 * @returns {void}
 */
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

  socket.addEventListener(
    "message",
    /** @param {MessageEvent} event */
    async (event) => {
      if (state.workspaceSocket !== socket) {
        return;
      }
      try {
        let rawData = event.data;
      if (rawData instanceof Blob) {
        rawData = await rawData.text();
      } else if (rawData instanceof ArrayBuffer) {
        rawData = textDecoder.decode(rawData);
      } else if (typeof rawData !== "string") {
        rawData = String(rawData);
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
    },
  );

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

/**
 * @param {WorkspaceEventEnvelope | { type?: string; [key: string]: unknown }} event
 */
function handleWorkspaceEvent(event) {
  if (!event || !event.type) return;

  switch (event.type) {
    case "workspace-full-update": {
      if (debugWorkspace) {
        console.log("Full workspace update:", event.payload);
      }
      const payload =
        event && typeof event.payload === "object"
          ? /** @type {Record<string, unknown>} */ (event.payload)
          : null;
      if (!payload) {
        return;
      }

      const rawTabs = Array.isArray(payload.tabs)
        ? /** @type {unknown[]} */ (payload.tabs)
        : [];
      const { tabs: sanitizedTabs, duplicateIds, invalidIds } =
        sanitizeWorkspaceTabsFromServer(rawTabs);
      if (duplicateIds.length || invalidIds.length) {
        reportWorkspaceAnomaly({ duplicateIds, invalidIds });
      }

      const requestedActiveId =
        typeof payload.activeTabId === "string"
          ? payload.activeTabId.trim()
          : null;
      applyWorkspaceSnapshot(sanitizedTabs, { activeTabId: requestedActiveId });

      break;
    }
    case "tab-added":
      if (isWorkspaceTabSnapshot(event.payload)) {
        handleTabAdded(event.payload);
      }
      break;
    case "tab-updated":
      if (isWorkspaceTabSnapshot(event.payload)) {
        handleTabUpdated(event.payload);
      }
      break;
    case "tab-removed":
      if (isTabIdentifierPayload(event.payload)) {
        handleTabRemoved(event.payload);
      }
      break;
    case "active-tab-changed":
      if (isTabIdentifierPayload(event.payload)) {
        handleActiveTabChanged(event.payload);
      }
      break;
    case "session-attached":
      if (isSessionAttachmentPayload(event.payload)) {
        handleSessionAttached(event.payload);
      }
      break;
    case "session-detached":
      if (isSessionDetachmentPayload(event.payload)) {
        handleSessionDetached(event.payload);
      }
      break;
    case "idle-timeout-changed":
      if (isIdleTimeoutPayload(event.payload)) {
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

/**
 * @param {unknown} value
 * @returns {value is WorkspaceTabSnapshot}
 */
function isWorkspaceTabSnapshot(value) {
  if (!value || typeof value !== "object") {
    return false;
  }
  const candidate = /** @type {Record<string, unknown>} */ (value);
  return (
    typeof candidate.id === "string" &&
    typeof candidate.label === "string" &&
    typeof candidate.colorId === "string"
  );
}

/**
 * @param {unknown} value
 * @returns {value is { id: string }}
 */
function isTabIdentifierPayload(value) {
  if (!value || typeof value !== "object") {
    return false;
  }
  const candidate = /** @type {Record<string, unknown>} */ (value);
  return typeof candidate.id === "string";
}

/**
 * @param {unknown} value
 * @returns {value is { tabId: string; sessionId: string }}
 */
function isSessionAttachmentPayload(value) {
  if (!value || typeof value !== "object") {
    return false;
  }
  const candidate = /** @type {Record<string, unknown>} */ (value);
  return (
    typeof candidate.tabId === "string" &&
    typeof candidate.sessionId === "string"
  );
}

/**
 * @param {unknown} value
 * @returns {value is { tabId: string }}
 */
function isSessionDetachmentPayload(value) {
  if (!value || typeof value !== "object") {
    return false;
  }
  const candidate = /** @type {Record<string, unknown>} */ (value);
  return typeof candidate.tabId === "string";
}

/**
 * @param {unknown} value
 * @returns {value is { idleTimeoutSeconds: number }}
 */
function isIdleTimeoutPayload(value) {
  if (!value || typeof value !== "object") {
    return false;
  }
  const candidate = /** @type {Record<string, unknown>} */ (value);
  return typeof candidate.idleTimeoutSeconds === "number";
}
