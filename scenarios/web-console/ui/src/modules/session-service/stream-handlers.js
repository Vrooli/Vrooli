import { state } from "../state.js";
import { appendEvent, recordTranscript, showError } from "../event-feed.js";
import { filterLocalEcho } from "../input-buffer.js";
import { debugLog } from "../debug.js";
import { textDecoder } from "../utils.js";
import { notifyActiveTabUpdate } from "./callbacks.js";
import { setTabPhase, setTabSocketState } from "./tab-state.js";

export function handleStreamEnvelope(tab, envelope) {
  if (!tab || !envelope || typeof envelope.type !== "string") return;

  switch (envelope.type) {
    case "output":
      handleOutputPayload(tab, envelope.payload);
      break;
    case "output_replay":
      handleReplayPayload(tab, envelope.payload);
      break;
    case "status":
      handleStatusPayload(tab, envelope.payload);
      break;
    case "heartbeat":
      appendEvent(tab, "session-heartbeat", envelope.payload);
      break;
    default:
      appendEvent(tab, "session-envelope", envelope);
  }
}

function handleOutputPayload(tab, payload, options = {}) {
  if (!tab || !payload || typeof payload.data !== "string") return;

  let text = payload.data;
  if (payload.encoding === "base64") {
    try {
      const bytes = Uint8Array.from(atob(payload.data), (c) => c.charCodeAt(0));
      text = textDecoder.decode(bytes);
    } catch (error) {
      appendEvent(tab, "decode-error", error);
      return;
    }
  }

  const filteredText = filterLocalEcho(tab, text);

  debugLog(tab, "output-received", {
    length: typeof text === "string" ? text.length : 0,
    renderedLength: typeof filteredText === "string" ? filteredText.length : 0,
    replay: options.replay === true,
    timestamp: payload.timestamp || null,
  });

  if (filteredText.length > 0) {
    tab.term.write(filteredText);
  }

  if (options.replay !== true && !tab.hasReceivedLiveOutput) {
    tab.hasReceivedLiveOutput = true;
    const forceRender = () => {
      try {
        if (tab.id === state.activeTabId && document.hasFocus()) {
          tab.term.focus();
        }
        tab.fitAddon?.fit();
        tab.term.scrollToBottom();
        tab.term.write("");
        tab.term.refresh(0, tab.term.rows - 1);
      } catch (error) {
        console.warn("Failed to refresh terminal on first output:", error);
      }
    };
    forceRender();
    requestAnimationFrame(forceRender);
    setTimeout(forceRender, 50);
  }

  if (options.record !== false) {
    const entry = {
      timestamp: payload.timestamp ? Date.parse(payload.timestamp) : Date.now(),
      direction: payload.direction || "stdout",
      encoding: payload.encoding,
      data: payload.data,
    };
    if (options.replay === true) {
      entry.replay = true;
    }
    recordTranscript(tab, entry);
  }

  if (tab.id === state.activeTabId) {
    notifyActiveTabUpdate();
  }
}

function handleReplayPayload(tab, payload) {
  if (!tab || !payload || typeof payload !== "object") return;

  const chunks = Array.isArray(payload.chunks) ? payload.chunks : [];
  const wasPending = tab.replayPending === true;

  if (wasPending) {
    tab.replayPending = false;
    tab.replayComplete = false;
    tab.lastReplayCount = 0;
    tab.lastReplayTruncated = false;

    if (!tab.hasEverConnected && chunks.length > 0) {
      try {
        tab.term.reset();
      } catch (_error) {
        // ignore
      }
      tab.transcript = [];
    }
  }

  chunks.forEach((chunk) => {
    if (!chunk || typeof chunk !== "object") return;
    handleOutputPayload(
      tab,
      { ...chunk, direction: chunk.direction || "stdout" },
      { replay: true, record: chunk.record !== false },
    );
    tab.lastReplayCount += 1;
  });

  if (payload.truncated !== undefined) {
    tab.lastReplayTruncated = Boolean(payload.truncated);
  }

  if (payload.complete) {
    tab.replayComplete = true;
    tab.hasEverConnected = true;
    const eventPayload = {
      count: tab.lastReplayCount,
      truncated: tab.lastReplayTruncated,
    };
    if (payload.generated) {
      const generatedTs = Date.parse(payload.generated);
      if (!Number.isNaN(generatedTs)) {
        eventPayload.generatedAt = generatedTs;
      }
    }
    appendEvent(tab, "output-replay-complete", eventPayload);

    if (tab.id === state.activeTabId) {
      const forceRender = () => {
        try {
          tab.term.focus();
          tab.fitAddon?.fit();
          tab.term.scrollToBottom();
          tab.term.write("");
          tab.term.refresh(0, tab.term.rows - 1);
        } catch (error) {
          console.warn("Failed to refresh terminal after replay:", error);
        }
      };
      requestAnimationFrame(forceRender);
      setTimeout(forceRender, 50);
    }
  } else {
    tab.replayPending = true;
  }

  if (tab.id === state.activeTabId) {
    notifyActiveTabUpdate();
  }
}

function handleStatusPayload(tab, payload) {
  if (!tab) return;
  appendEvent(tab, "session-status", payload);

  if (payload && typeof payload === "object") {
    recordTranscript(tab, {
      timestamp: payload.timestamp ? Date.parse(payload.timestamp) : Date.now(),
      direction: "status",
      message: JSON.stringify(payload),
    });

    if (payload.status === "started") {
      setTabPhase(tab, "running");
      setTabSocketState(tab, "open");
      showError(tab, "");
      return;
    }

    if (payload.status === "command_exit_error") {
      setTabPhase(tab, "closed");
      setTabSocketState(tab, "disconnected");
      showError(tab, `Command exited: ${payload.reason || "unknown error"}`);
      notifyPanic(tab, payload);
      return;
    }

    if (payload.status === "closed") {
      setTabPhase(tab, "closed");
      setTabSocketState(tab, "disconnected");
      if (payload.reason && !payload.reason.includes("client_requested")) {
        showError(tab, `Session closed: ${payload.reason}`);
      }
      return;
    }

    return;
  }

  recordTranscript(tab, {
    timestamp: Date.now(),
    direction: "status",
    message: JSON.stringify(payload),
  });
}

function notifyPanic(tab, payload) {
  if (!tab) return;
  const reason =
    payload && typeof payload === "object" && payload.reason
      ? payload.reason
      : "unknown error";
  tab.term.write(`\r\n\u001b[31mCommand exited\u001b[0m: ${reason}\r\n`);
  tab.term.write("Review the event feed or transcripts for details.\r\n");
}
