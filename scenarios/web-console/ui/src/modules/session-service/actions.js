import { appendEvent, renderEventMeta, showError } from "../event-feed.js";
import {
  queueInput,
  flushPendingWrites,
  transmitInput,
  sendResize,
  handleTerminalData as bufferHandleTerminalData,
} from "../input-buffer.js";
import { proxyToApi, formatCommandLabel, parseApiError } from "../utils.js";
import { showToast } from "../notifications.js";
import { refreshTabButton } from "../tab-manager.js";
import { connectWebSocket } from "./socket.js";
import {
  queueSessionOverviewRefresh,
  notifySessionActionsChanged,
} from "./callbacks.js";
import { setTabPhase, setTabSocketState } from "./tab-state.js";

export async function startSession(tab, options = {}) {
  if (!tab || tab.phase === "creating" || tab.phase === "running") return;
  showError(tab, "");
  setTabPhase(tab, "creating");
  setTabSocketState(tab, "connecting");

  try {
    const payload = {};
    if (options.reason) payload.reason = options.reason;
    if (options.command) payload.command = options.command;
    if (Array.isArray(options.args) && options.args.length > 0)
      payload.args = options.args;
    if (options.metadata) payload.metadata = options.metadata;
    if (tab.id) payload.tabId = tab.id;

    const response = await proxyToApi("/api/v1/sessions", {
      method: "POST",
      json: payload,
    });
    if (!response.ok) {
      let text = "";
      try {
        text = await response.text();
      } catch (_error) {
        text = "";
      }
      const { message, raw } = parseApiError(
        text,
        `API error (${response.status})`,
      );
      const error = new Error(message);
      error.status = response.status;
      if (raw) {
        error.payload = raw;
      }
      throw error;
    }
    const data = await response.json();
    const sessionArgs = Array.isArray(data.args)
      ? [...data.args]
      : Array.isArray(options.args)
        ? [...options.args]
        : [];
    const sessionCommand = data.command || options.command || "";
    tab.session = {
      ...data,
      command: sessionCommand,
      args: sessionArgs,
      commandLine: formatCommandLabel(sessionCommand, sessionArgs),
    };
    tab.wasDetached = false;
    refreshTabButton(tab);
    tab.transcript = [];
    tab.transcriptByteSize = 0;
    tab.events = [];
    tab.suppressed = tab.suppressed || {};
    Object.keys(tab.suppressed).forEach((key) => {
      tab.suppressed[key] = 0;
    });
    tab.pendingWrites = Array.isArray(tab.pendingWrites)
      ? tab.pendingWrites
      : [];
    tab.errorMessage = "";
    tab.lastSentSize = { cols: 0, rows: 0 };
    tab.inputSeq = 0;
    tab.replayPending = false;
    tab.replayComplete = true;
    tab.lastReplayCount = 0;
    tab.lastReplayTruncated = false;
    tab.hasReceivedLiveOutput = false;
    tab.term.reset();
    renderEventMeta(tab);
    appendEvent(tab, "session-created", {
      ...data,
      reason: options.reason || null,
      command: tab.session.command,
      args: tab.session.args,
    });
    connectWebSocket(tab, data.id, {
      onReconnect: () => reconnectSession(tab, data.id),
    });
    setTabPhase(tab, "running");
    queueSessionOverviewRefresh(250);
    notifySessionActionsChanged();
  } catch (error) {
    setTabPhase(tab, "idle");
    setTabSocketState(tab, "disconnected");
    const message =
      error instanceof Error ? error.message : "Unknown error starting session";
    showError(tab, message);
    appendEvent(tab, "session-error", error);
    tab.wasDetached = true;
    refreshTabButton(tab);
    notifySessionActionsChanged();
    if (
      typeof message === "string" &&
      message.toLowerCase().includes("capacity reached")
    ) {
      queueSessionOverviewRefresh(100);
      await showToast(
        "Terminal pool is full. Sign out stale sessions or wait a few seconds.",
        "error",
        4000,
      );
    }
  }
}

export async function stopSession(tab) {
  if (!tab || !tab.session) return;
  if (tab.phase === "closing") return;
  setTabPhase(tab, "closing");
  setTabSocketState(tab, "closing");
  appendEvent(tab, "session-stop-requested", { id: tab.session.id });
  try {
    await proxyToApi(`/api/v1/sessions/${tab.session.id}`, {
      method: "DELETE",
    });
  } catch (error) {
    appendEvent(tab, "session-stop-error", error);
  } finally {
    try {
      tab.socket?.close();
    } catch (_error) {
      // ignore
    }
    tab.socket = null;
    setTabPhase(tab, "closed");
    setTabSocketState(tab, "disconnected");
    tab.wasDetached = true;
    refreshTabButton(tab);
    queueSessionOverviewRefresh(250);
    notifySessionActionsChanged();
  }
}

export async function reconnectSession(tab, sessionId) {
  if (!tab || !sessionId) {
    return;
  }
  if (tab.reconnecting) {
    return;
  }

  tab.reconnecting = true;
  try {
    setTabSocketState(tab, "connecting");
    const response = await proxyToApi(`/api/v1/sessions/${sessionId}`);
    if (!response.ok) {
      const status = response.status;
      appendEvent(tab, "session-reconnect-miss", { sessionId, status });
      if (status === 404) {
        tab.session = null;
        setTabPhase(tab, "idle");
        setTabSocketState(tab, "disconnected");
        showError(tab, "Session expired. Type to launch a new shell.");
        tab.wasDetached = true;
        refreshTabButton(tab);
        queueSessionOverviewRefresh(250);
      } else {
        setTabSocketState(tab, "error");
        showError(tab, `Unable to reconnect (status ${status})`);
        tab.wasDetached = true;
        refreshTabButton(tab);
      }
      notifySessionActionsChanged();
      return;
    }
    const data = await response.json();
    tab.session = {
      ...data,
      commandLine: formatCommandLabel(data.command, data.args),
    };
    tab.wasDetached = false;
    refreshTabButton(tab);
    connectWebSocket(tab, sessionId, {
      onReconnect: () => reconnectSession(tab, sessionId),
    });
    setTabPhase(tab, "running");
    showError(tab, "");
    notifySessionActionsChanged();
  } catch (error) {
    appendEvent(tab, "session-reconnect-error", {
      sessionId,
      message: error instanceof Error ? error.message : String(error),
    });
    setTabSocketState(tab, "error");
    showError(tab, "Unable to reconnect to session");
    tab.wasDetached = true;
    refreshTabButton(tab);
    notifySessionActionsChanged();
  } finally {
    tab.reconnecting = false;
  }
}

export function handleTerminalData(tab, data) {
  bufferHandleTerminalData(tab, data, ensureSessionForPendingInput);
}

export function ensureSessionForPendingInput(tab, reason) {
  if (!tab) return;

  if (tab.session && tab.session.id && !tab.socket) {
    if (tab.reconnecting) {
      showError(tab, "Reconnecting to session…");
      return;
    }
    showError(tab, "Reconnecting to session…");
    reconnectSession(tab, tab.session.id).catch((error) => {
      appendEvent(tab, "reconnect-on-input-failed", error);
      if (tab.phase === "idle" || tab.phase === "closed") {
        showError(tab, "Starting new session…");
        startSession(tab, {
          reason: reason || "auto-start:input-after-reconnect-fail",
        }).catch((startError) => {
          appendEvent(tab, "session-error", startError);
          showError(
            tab,
            startError instanceof Error
              ? startError.message
              : "Unable to start terminal session",
          );
        });
      }
    });
    return;
  }

  if (tab.phase === "idle" || tab.phase === "closed") {
    showError(tab, "Starting new session…");
    startSession(tab, { reason: reason || "auto-start:input" }).catch(
      (error) => {
        appendEvent(tab, "session-error", error);
        showError(
          tab,
          error instanceof Error
            ? error.message
            : "Unable to start terminal session",
        );
      },
    );
    return;
  }

  if (!tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    showError(tab, "Connecting…");
  }
}

export function queueInputForTab(tab, value, meta) {
  queueInput(tab, value, meta);
}

export function flushPendingWritesForTab(tab) {
  flushPendingWrites(tab);
}

export function transmitInputForTab(tab, value, meta) {
  return transmitInput(tab, value, meta);
}

export function sendResizeForTab(tab, cols, rows) {
  sendResize(tab, cols, rows);
}
