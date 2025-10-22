import { appendEvent, showError } from "../event-feed.js";
import { flushPendingWrites, sendResize } from "../input-buffer.js";
import { normalizeSocketData } from "../utils.js";
import { notifySessionActionsChanged } from "./callbacks.js";
import { setTabPhase, setTabSocketState } from "./tab-state.js";
import { handleStreamEnvelope } from "./stream-handlers.js";

export function connectWebSocket(tab, sessionId, { onReconnect } = {}) {
  if (!tab || !sessionId) {
    return;
  }

  const url = buildStreamUrl(sessionId);
  const previousSocket = tab.socket;
  const socket = new WebSocket(url);
  tab.inputSeq = 0;
  tab.socket = socket;
  tab.replayPending = true;
  tab.replayComplete = false;
  tab.lastReplayCount = 0;
  tab.lastReplayTruncated = false;

  if (previousSocket && previousSocket !== socket) {
    try {
      previousSocket.close();
    } catch (error) {
      appendEvent(tab, "ws-close-error", {
        message: error instanceof Error ? error.message : String(error),
      });
    }
  }

  socket.addEventListener("open", () => {
    if (tab.socket !== socket) {
      return;
    }
    setTabSocketState(tab, "open");
    appendEvent(tab, "ws-open", { sessionId });
    if (
      typeof tab.term.cols === "number" &&
      typeof tab.term.rows === "number"
    ) {
      sendResize(tab, tab.term.cols, tab.term.rows);
    }
    flushPendingWrites(tab);

    if (tab.heartbeatInterval) {
      clearInterval(tab.heartbeatInterval);
    }
    tab.heartbeatInterval = setInterval(() => {
      if (tab.socket === socket && socket.readyState === WebSocket.OPEN) {
        try {
          socket.send(JSON.stringify({ type: "heartbeat", payload: {} }));
        } catch (error) {
          console.warn("Heartbeat send failed:", error);
        }
      }
    }, 30000);
  });

  socket.addEventListener("message", async (event) => {
    if (tab.socket !== socket) {
      return;
    }
    const raw = await normalizeSocketData(event.data, (error) => {
      appendEvent(tab, "ws-decode-error", error);
    });
    if (!raw || raw.trim().length === 0) {
      appendEvent(tab, "ws-empty-frame", null);
      return;
    }
    try {
      const envelope = JSON.parse(raw);
      handleStreamEnvelope(tab, envelope);
    } catch (error) {
      appendEvent(tab, "ws-message-error", {
        message: error instanceof Error ? error.message : String(error),
        raw,
      });
      showError(tab, "Failed to process stream message");
    }
  });

  socket.addEventListener("error", (error) => {
    if (tab.socket !== socket) {
      return;
    }
    setTabSocketState(tab, "error");
    showError(tab, "WebSocket error occurred");
    appendEvent(tab, "ws-error", error);
  });

  socket.addEventListener("close", (event) => {
    if (tab.socket !== socket) {
      return;
    }

    if (tab.heartbeatInterval) {
      clearInterval(tab.heartbeatInterval);
      tab.heartbeatInterval = null;
    }

    setTabSocketState(tab, "disconnected");
    appendEvent(tab, "ws-close", {
      code: event.code,
      reason: event.reason,
      sessionId,
    });

    tab.socket = null;

    const isExplicitClose = tab.phase === "closing";

    if (isExplicitClose) {
      setTabPhase(tab, "closed");
    } else if (
      tab.session &&
      tab.phase === "running" &&
      typeof onReconnect === "function"
    ) {
      appendEvent(tab, "ws-reconnecting", {
        sessionId,
        attempt: 1,
        code: event.code,
      });
      setTimeout(() => {
        if (tab.session && tab.session.id === sessionId && !tab.socket) {
          Promise.resolve(onReconnect()).catch((error) => {
            appendEvent(tab, "ws-reconnect-failed", error);
            setTabPhase(tab, "closed");
            showError(
              tab,
              "Session disconnected. Type to start a new session.",
            );
          });
        }
      }, 1000);
    } else if (tab.phase !== "closed" && tab.phase !== "idle") {
      setTabPhase(tab, "closed");
    }

    notifySessionActionsChanged();
  });
}

function buildStreamUrl(sessionId) {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const host = window.location.host;
  return `${protocol}//${host}/ws/sessions/${sessionId}/stream`;
}
