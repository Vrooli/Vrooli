import { state } from "../state.js";
import { appendEvent, recordTranscript, showError } from "../event-feed.js";
import { filterLocalEcho } from "../input-buffer.js";
import { debugLog } from "../debug.js";
import { textDecoder } from "../utils.js";
import { showToast } from "../notifications.js";
import { fitTerminal } from "../app/terminal-utils.js";
import { notifyActiveTabUpdate } from "./callbacks.js";
import { setTabPhase, setTabSocketState } from "./tab-state.js";
import { fetchSessionTranscriptRequest } from "./transport.js";

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

  const shouldFilter = options.replay !== true && options.skipLocalEcho !== true;
  const renderedText = shouldFilter ? filterLocalEcho(tab, text) : text;

  debugLog(tab, "output-received", {
    length: typeof text === "string" ? text.length : 0,
    renderedLength: typeof renderedText === "string" ? renderedText.length : 0,
    replay: options.replay === true,
    timestamp: payload.timestamp || null,
  });

  if (typeof renderedText === "string" && renderedText.length > 0) {
    tab.term.write(renderedText);
  }

  if (options.replay !== true && !tab.hasReceivedLiveOutput) {
    tab.hasReceivedLiveOutput = true;
    const forceRender = () => {
      try {
        fitTerminal(tab);
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
    const truncated = Boolean(payload.truncated);
    if (truncated && !tab.lastReplayTruncated) {
      notifyReplayTruncated(tab, payload);
    }
    tab.lastReplayTruncated = truncated;
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
          fitTerminal(tab);
        } catch (error) {
          console.warn("Failed to refresh terminal after replay:", error);
        }
      };
      requestAnimationFrame(forceRender);
      setTimeout(forceRender, 50);
    }

    void hydrateTranscriptForTab(tab, {
      generatedAt: payload.generated,
      reason: wasPending ? "replay-pending" : "replay",
    });
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

function notifyReplayTruncated(tab, payload) {
  const eventPayload = {
    count: typeof payload.count === "number" ? payload.count : null,
    truncated: true,
    generatedAt: payload.generated || null,
  };
  appendEvent(tab, "output-replay-truncated", eventPayload);
  if (tab.id === state.activeTabId) {
    const promise = showToast(
      "Session history was trimmed; loading archived transcript…",
      "warning",
      4200,
    );
    if (promise && typeof promise.catch === "function") {
      promise.catch(() => {});
    }
  }
  notifyActiveTabUpdate();
}

export async function hydrateTranscriptForTab(tab, options = {}) {
  if (!tab || !tab.session || !tab.session.id) return;
  if (tab.transcriptHydrated || tab.transcriptHydrating) return;

  tab.transcriptHydrating = true;
  appendEvent(tab, "transcript-hydration-start", {
    reason: options.reason || null,
    sessionId: tab.session.id,
  });

  try {
    const response = await fetchSessionTranscriptRequest(tab.session.id);
    if (!response || !response.ok) {
      const status = response ? response.status : null;
      appendEvent(tab, "transcript-hydration-failed", {
        status,
        message: response ? await safeReadText(response) : "no response",
      });
      return;
    }

    const raw = await response.text();
    if (!raw) {
      tab.transcriptHydrated = true;
      appendEvent(tab, "transcript-hydration-complete", {
        sessionId: tab.session.id,
        entryCount: 0,
      });
      return;
    }

    const entries = parseTranscriptNdjson(tab, raw);
    renderHydratedTranscript(tab, entries);
    tab.transcriptHydrated = true;
    appendEvent(tab, "transcript-hydration-complete", {
      sessionId: tab.session.id,
      entryCount: entries.length,
      generatedAt: options.generatedAt || null,
    });
  } catch (error) {
    appendEvent(tab, "transcript-hydration-error", {
      message: error instanceof Error ? error.message : String(error),
      sessionId: tab.session?.id || null,
    });
  } finally {
    tab.transcriptHydrating = false;
    if (tab.id === state.activeTabId) {
      notifyActiveTabUpdate();
    }
  }
}

async function safeReadText(response) {
  try {
    return await response.text();
  } catch (_error) {
    return "";
  }
}

function parseTranscriptNdjson(tab, raw) {
  const entries = [];
  if (typeof raw !== "string" || raw.trim() === "") {
    return entries;
  }

  raw.split(/\r?\n/).forEach((line, index) => {
    const trimmed = line.trim();
    if (!trimmed) return;
    try {
      const parsed = JSON.parse(trimmed);
      entries.push(parsed);
    } catch (error) {
      appendEvent(tab, "transcript-parse-error", {
        index,
        message: error instanceof Error ? error.message : String(error),
      });
    }
  });

  return entries;
}

function renderHydratedTranscript(tab, entries) {
  if (!tab || !tab.term || typeof tab.term.write !== "function") {
    return entries;
  }

  try {
    if (typeof tab.term.reset === "function") {
      tab.term.reset();
    }
  } catch (_error) {
    // ignore reset errors
  }

  tab.transcript = [];
  tab.transcriptByteSize = 0;

  let outputChunks = 0;

  entries.forEach((entry) => {
    const normalized = normalizeTranscriptEntry(entry);
    if (!normalized) return;

    recordTranscript(tab, normalized);

    if (
      normalized.direction === "stdout" ||
      normalized.direction === "stderr"
    ) {
      const text = decodeTranscriptChunk(normalized);
      if (text) {
        tab.term.write(text);
        outputChunks += 1;
      }
    }
  });

  if (!tab.hasReceivedLiveOutput && outputChunks > 0) {
    tab.hasReceivedLiveOutput = true;
  }

  forceTerminalRender(tab);

  return entries;
}

function normalizeTranscriptEntry(entry) {
  if (!entry || typeof entry !== "object") {
    return null;
  }

  const direction =
    typeof entry.direction === "string" ? entry.direction : "stdout";

  let timestamp = Date.now();
  if (entry.timestamp !== undefined) {
    if (typeof entry.timestamp === "number" && Number.isFinite(entry.timestamp)) {
      timestamp = entry.timestamp;
    } else {
      const parsed = Date.parse(String(entry.timestamp));
      if (!Number.isNaN(parsed)) {
        timestamp = parsed;
      }
    }
  }

  return {
    timestamp,
    direction,
    encoding:
      typeof entry.encoding === "string" && entry.encoding
        ? entry.encoding
        : undefined,
    data: typeof entry.data === "string" ? entry.data : undefined,
    message: typeof entry.message === "string" ? entry.message : undefined,
  };
}

function decodeTranscriptChunk(entry) {
  if (!entry || typeof entry.data !== "string") {
    return "";
  }

  if (entry.encoding === "base64") {
    try {
      const binary = atob(entry.data);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i += 1) {
        bytes[i] = binary.charCodeAt(i);
      }
      return textDecoder.decode(bytes);
    } catch (error) {
      console.warn("Failed to decode transcript chunk:", error);
      return "";
    }
  }

  return entry.data;
}

function forceTerminalRender(tab) {
  if (!tab || !tab.term || typeof tab.term.refresh !== "function") {
    return;
  }

  const render = () => {
    try {
      fitTerminal(tab);
    } catch (error) {
      console.warn("Failed to refresh terminal after hydration:", error);
    }
  };

  render();

  if (typeof requestAnimationFrame === "function") {
    requestAnimationFrame(render);
  }
  setTimeout(render, 50);
}
