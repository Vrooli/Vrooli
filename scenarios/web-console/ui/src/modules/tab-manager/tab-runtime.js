// @ts-check

import { initialSuppressedState } from "../state.js";

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

/**
 * Reset or initialize the per-session runtime state tracked on a terminal tab.
 * @param {TerminalTab | null | undefined} tab
 * @param {{
 *   preserveTranscript?: boolean;
 *   preserveEvents?: boolean;
 *   resetSuppressed?: boolean;
 *   resetTerminal?: boolean;
 *   markDetached?: boolean;
 * }} [options]
 */
export function resetTabRuntimeState(tab, options = {}) {
  if (!tab) return;

  const {
    preserveTranscript = false,
    preserveEvents = false,
    resetSuppressed = true,
    resetTerminal = true,
    markDetached = false,
  } = options;

  if (!preserveTranscript) {
    tab.transcript = [];
    tab.transcriptByteSize = 0;
  } else {
    tab.transcript = Array.isArray(tab.transcript) ? tab.transcript : [];
    tab.transcriptByteSize = Number(tab.transcriptByteSize) || 0;
  }
  tab.transcriptHydrated = false;
  tab.transcriptHydrating = false;

  if (!preserveEvents) {
    tab.events = [];
  } else {
    tab.events = Array.isArray(tab.events) ? tab.events : [];
  }

  if (resetSuppressed) {
    tab.suppressed = initialSuppressedState();
  } else {
    const suppressed = tab.suppressed || initialSuppressedState();
    Object.keys(suppressed).forEach((key) => {
      suppressed[key] = 0;
    });
    tab.suppressed = suppressed;
  }

  tab.pendingWrites = [];
  tab.localEchoBuffer = [];
  tab.inputBatch = null;
  tab.inputBatchScheduled = false;
  tab.inputSeq = 0;
  tab.telemetry = {
    typed: 0,
    queued: 0,
    sent: 0,
    batches: 0,
    lastBatchSize: 0,
  };
  tab.layoutCache = {
    width: 0,
    height: 0,
  };
  tab.lastSentSize = { cols: 0, rows: 0 };
  tab.errorMessage = "";

  tab.replayPending = false;
  tab.replayComplete = true;
  tab.lastReplayCount = 0;
  tab.lastReplayTruncated = false;
  tab.hasReceivedLiveOutput = false;
  tab.hasEverConnected = false;
  tab.reconnecting = false;
  tab.wasDetached = Boolean(markDetached);
  if (tab.userScroll) {
    tab.userScroll.active = false;
    tab.userScroll.pinnedToBottom = true;
    tab.userScroll.touchActive = false;
    tab.userScroll.momentumActive = false;
    tab.userScroll.lastInteraction = 0;
    if (tab.userScroll.releaseTimer) {
      clearTimeout(tab.userScroll.releaseTimer);
      tab.userScroll.releaseTimer = null;
    }
  }

  if (tab.heartbeatInterval) {
    clearInterval(tab.heartbeatInterval);
  }
  tab.heartbeatInterval = null;

  if (tab.term && resetTerminal && typeof tab.term.reset === "function") {
    try {
      tab.term.reset();
    } catch (_error) {
      // ignore reset failures for environments without a real terminal
    }
  }
}
