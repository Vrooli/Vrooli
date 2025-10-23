// @ts-nocheck

import {
  INPUT_BATCH_MAX_CHARS,
  INPUT_CONTROL_FLUSH_PATTERN,
  LOCAL_ECHO_ENABLED,
  LOCAL_ECHO_MAX_BUFFER,
  LOCAL_ECHO_TIMEOUT_MS,
} from "./state.js";
import { appendEvent, recordTranscript, showError } from "./event-feed.js";
import { debugLog } from "./debug.js";
import { scheduleMicrotask, textEncoder } from "./utils.js";

export function ensureInputSequence(tab, meta = {}) {
  if (!tab) {
    return 0;
  }
  if (meta && typeof meta.seq === "number") {
    return meta.seq;
  }
  if (!Number.isInteger(tab.inputSeq)) {
    tab.inputSeq = 0;
  }
  tab.inputSeq += 1;
  meta.seq = tab.inputSeq;
  return meta.seq;
}

export function queueInput(tab, value, meta = {}) {
  if (!tab || typeof value !== "string" || value.length === 0) {
    return;
  }
  if (!Array.isArray(tab.pendingWrites)) {
    tab.pendingWrites = [];
  }
  ensureInputSequence(tab, meta);
  if (typeof meta.batchSize !== "number") {
    meta.batchSize = value.length;
  }
  tab.pendingWrites.push({ value, meta });
  if (tab.telemetry) {
    tab.telemetry.queued += 1;
    tab.telemetry.lastBatchSize = Array.isArray(tab.pendingWrites)
      ? tab.pendingWrites.length
      : 0;
  }
  debugLog(tab, "queued-input", {
    length: value.length,
    appendNewline: meta.appendNewline === true,
    pendingCount: tab.pendingWrites.length,
  });
}

export function flushPendingWrites(tab) {
  if (!tab || !tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    return;
  }
  if (!Array.isArray(tab.pendingWrites) || tab.pendingWrites.length === 0) {
    return;
  }
  const queue = tab.pendingWrites.splice(0);
  if (queue.length > 0 && tab.telemetry) {
    tab.telemetry.batches += 1;
    tab.telemetry.lastBatchSize = queue.length;
  }
  queue.forEach((item) => {
    if (item && item.meta && typeof item.meta.batchSize !== "number") {
      item.meta.batchSize =
        typeof item.value === "string" ? item.value.length : 0;
    }
    const success = transmitInput(tab, item.value, item.meta);
    if (!success) {
      tab.pendingWrites.unshift(item);
    }
  });
}

export function transmitInput(tab, value, meta = {}) {
  if (!tab || !tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    return false;
  }
  if (typeof value !== "string" || value.length === 0) {
    return false;
  }

  const shouldAppendNewline = meta.appendNewline === true;
  const normalized = shouldAppendNewline
    ? value.endsWith("\n")
      ? value
      : `${value}\n`
    : value;
  if (!normalized) return true;

  const seq = ensureInputSequence(tab, meta);
  const dataBytes = textEncoder.encode(normalized);
  const sourceString = "";
  const sourceBytesFull = textEncoder.encode(sourceString);
  const sourceLen = Math.min(sourceBytesFull.length, 0xffff);

  const headerLength = 1 + 8 + 2 + sourceLen;
  const frame = new Uint8Array(headerLength + dataBytes.length);
  let offset = 0;

  frame[offset] = 0x01;
  offset += 1;

  let seqValue = Number(seq);
  for (let i = 7; i >= 0; i -= 1) {
    frame[offset + i] = seqValue & 0xff;
    seqValue = Math.floor(seqValue / 256);
  }
  offset += 8;

  frame[offset] = (sourceLen >> 8) & 0xff;
  frame[offset + 1] = sourceLen & 0xff;
  offset += 2;

  if (sourceLen > 0) {
    frame.set(sourceBytesFull.subarray(0, sourceLen), offset);
    offset += sourceLen;
  }

  frame.set(dataBytes, offset);

  try {
    tab.socket.send(frame);
  } catch (error) {
    appendEvent(tab, "stdin-send-error", error);
    return false;
  }

  if (tab.telemetry) {
    tab.telemetry.sent += 1;
    tab.telemetry.lastBatchSize =
      meta && typeof meta.batchSize === "number" ? meta.batchSize : 1;
  }
  debugLog(tab, "sent-input", {
    length: normalized.length,
    seq,
    appendedNewline: shouldAppendNewline,
    batchSize: meta && typeof meta.batchSize === "number" ? meta.batchSize : 1,
  });

  recordTranscript(tab, {
    timestamp: Date.now(),
    direction: "stdin",
    encoding: "utf-8",
    data: btoa(normalized),
  });

  if (meta.eventType) {
    appendEvent(tab, meta.eventType, {
      length: normalized.length,
      source: meta.source || undefined,
      command: meta.command || undefined,
    });
  }

  if (meta.clearError !== false) {
    showError(tab, "");
  }

  return true;
}

export function sendResize(tab, cols, rows) {
  if (!tab || !tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    return;
  }
  if (
    !Number.isInteger(cols) ||
    !Number.isInteger(rows) ||
    cols <= 0 ||
    rows <= 0
  ) {
    return;
  }
  const { cols: lastCols, rows: lastRows } = tab.lastSentSize;
  if (cols === lastCols && rows === lastRows) {
    return;
  }
  const payload = {
    type: "resize",
    payload: { cols, rows },
  };
  try {
    tab.socket.send(JSON.stringify(payload));
    tab.lastSentSize = { cols, rows };
    appendEvent(tab, "terminal-resize", { cols, rows });
  } catch (error) {
    appendEvent(tab, "terminal-resize-error", error);
  }
}

export function handleTerminalData(tab, data, ensureSessionForPendingInput) {
  if (!tab || typeof data !== "string" || data.length === 0) {
    return;
  }

  if (tab.telemetry) {
    tab.telemetry.typed += data.length;
  }

  debugLog(tab, "terminal-data", { length: data.length });

  if (LOCAL_ECHO_ENABLED) {
    applyLocalEcho(tab, data);
  }
  enqueueTerminalInput(tab, data);

  if (!tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    ensureSessionForPendingInput?.(tab, "auto-start:terminal-input");
  }
}

export function enqueueTerminalInput(tab, data, ensureSessionForPendingInput) {
  if (!tab) return;

  const now =
    typeof performance !== "undefined" && typeof performance.now === "function"
      ? performance.now()
      : Date.now();

  if (!tab.inputBatch) {
    tab.inputBatch = {
      chunks: [data],
      size: data.length,
      createdAt: now,
    };
    if (INPUT_CONTROL_FLUSH_PATTERN.test(data)) {
      flushInputBatch(
        tab,
        { reason: "control-char" },
        ensureSessionForPendingInput,
      );
    } else {
      scheduleInputBatchFlush(tab, ensureSessionForPendingInput);
    }
    return;
  }

  tab.inputBatch.chunks.push(data);
  tab.inputBatch.size += data.length;

  if (INPUT_CONTROL_FLUSH_PATTERN.test(data)) {
    flushInputBatch(
      tab,
      { reason: "control-char" },
      ensureSessionForPendingInput,
    );
    return;
  }

  if (tab.inputBatch.size >= INPUT_BATCH_MAX_CHARS) {
    flushInputBatch(
      tab,
      { reason: "batch-size" },
      ensureSessionForPendingInput,
    );
    return;
  }

  scheduleInputBatchFlush(tab);
}

export function scheduleInputBatchFlush(tab, ensureSessionForPendingInput) {
  if (!tab) return;
  if (tab.inputBatchScheduled) {
    return;
  }
  tab.inputBatchScheduled = true;
  scheduleMicrotask(() => {
    tab.inputBatchScheduled = false;
    flushInputBatch(tab, { reason: "microtask" }, ensureSessionForPendingInput);
  });
}

export function flushInputBatch(
  tab,
  options = {},
  ensureSessionForPendingInput,
) {
  if (!tab || !tab.inputBatch) {
    return;
  }

  tab.inputBatchScheduled = false;

  const { chunks, size } = tab.inputBatch;
  tab.inputBatch = null;

  if (!Array.isArray(chunks) || chunks.length === 0) {
    return;
  }

  const payload = chunks.join("");
  if (payload.length === 0) {
    return;
  }

  const isMicrotaskFlush = options.reason === "microtask";
  if (
    isMicrotaskFlush &&
    payload.length === 1 &&
    payload.charCodeAt(0) === 10
  ) {
    debugLog(tab, "flush-input-batch", {
      reason: "microtask-skipped-newline",
      size,
      skipped: true,
    });
    tab.pendingWrites.unshift({
      value: payload,
      meta: { appendNewline: false },
    });
    return;
  }

  const meta = {
    appendNewline: false,
    eventType: "terminal-batch",
    source: "terminal",
    batchSize: size,
    flushReason: options.reason || "flush",
  };

  debugLog(tab, "flush-input-batch", {
    reason: meta.flushReason,
    length: payload.length,
    size,
  });

  if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
    const sent = transmitInput(tab, payload, meta);
    if (!sent) {
      queueInput(tab, payload, meta);
      ensureSessionForPendingInput?.(tab, "auto-start:transmit-failed");
    }
    return;
  }

  queueInput(tab, payload, meta);
}

export function applyLocalEcho(tab, data) {
  if (!tab || typeof data !== "string" || data.length === 0) {
    return;
  }
  if (!tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    return;
  }
  if (!Array.isArray(tab.localEchoBuffer)) {
    tab.localEchoBuffer = [];
  }

  pruneStaleLocalEcho(tab);

  const now =
    typeof performance !== "undefined" && typeof performance.now === "function"
      ? performance.now()
      : Date.now();

  let buffer = "";
  for (const char of data) {
    if (!isPrintableLocalEchoChar(char)) {
      continue;
    }
    buffer += char;
    tab.localEchoBuffer.push({ value: char, timestamp: now });
    if (tab.localEchoBuffer.length > LOCAL_ECHO_MAX_BUFFER) {
      tab.localEchoBuffer.shift();
    }
  }

  if (buffer.length > 0) {
    tab.term.write(buffer);
    debugLog(tab, "local-echo-applied", { length: buffer.length });
  }
}

function isPrintableLocalEchoChar(char) {
  if (typeof char !== "string" || char.length === 0) return false;
  const code = char.codePointAt(0);
  return typeof code === "number" && code >= 0x20 && code <= 0x7e;
}

export function pruneStaleLocalEcho(tab) {
  if (
    !tab ||
    !Array.isArray(tab.localEchoBuffer) ||
    tab.localEchoBuffer.length === 0
  ) {
    return;
  }
  const now =
    typeof performance !== "undefined" && typeof performance.now === "function"
      ? performance.now()
      : Date.now();

  while (tab.localEchoBuffer.length > 0) {
    const entry = tab.localEchoBuffer[0];
    if (!entry) break;
    if (now - entry.timestamp <= LOCAL_ECHO_TIMEOUT_MS) {
      break;
    }
    tab.localEchoBuffer.shift();
  }

  if (tab.localEchoBuffer.length > LOCAL_ECHO_MAX_BUFFER) {
    tab.localEchoBuffer.splice(
      0,
      tab.localEchoBuffer.length - LOCAL_ECHO_MAX_BUFFER,
    );
  }
}

export function filterLocalEcho(tab, text) {
  if (!tab || typeof text !== "string" || text.length === 0) {
    return text;
  }

  if (!Array.isArray(tab.localEchoBuffer) || tab.localEchoBuffer.length === 0) {
    return text;
  }

  pruneStaleLocalEcho(tab);

  if (tab.localEchoBuffer.length === 0) {
    return text;
  }

  const pending = tab.localEchoBuffer;
  let result = "";
  let mismatchDetected = false;
  for (const char of text) {
    const next = pending[0];
    if (next && next.value === char) {
      pending.shift();
      debugLog(tab, "local-echo-ack", { char, remaining: pending.length });
      continue;
    }
    mismatchDetected = true;
    debugLog(tab, "local-echo-mismatch", {
      expected: next ? next.value : null,
      actual: char,
      remaining: pending.length,
    });
    result += char;
  }

  if (mismatchDetected) {
    tab.localEchoBuffer.length = 0;
  }

  return result;
}
