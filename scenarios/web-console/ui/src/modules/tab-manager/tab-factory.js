// @ts-check

import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";

import {
  elements,
  state,
  TAB_COLOR_DEFAULT,
  terminalDefaults,
  nextTabSequence,
  initialSuppressedState,
} from "../state.js";
import { tabCallbacks } from "./callbacks.js";
import { getTerminalHandlers } from "./terminal-handlers.js";
import {
  registerTabCustomizationHandlers,
  closeTabCustomization,
} from "./customization.js";
import { generateDefaultTabLabel } from "./tab-helpers.js";
import { resetTabRuntimeState } from "./tab-runtime.js";
import { renderTabs, setActiveTab } from "./tab-controller.js";
import { attachViewportInteractionHandlers } from "./viewport-interactions.js";

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

/**
 * @param {{
 *  focus?: boolean;
 *  id?: string | null;
 *  label?: string | null;
 *  colorId?: string | null;
 * }} [options]
 * @returns {TerminalTab | null}
 */
export function createTerminalTab({
  focus = false,
  id = null,
  label = null,
  colorId = TAB_COLOR_DEFAULT,
} = {}) {
  if (!elements.terminalHost) return null;
  if (!id) {
    id = `tab-${Date.now()}-${nextTabSequence()}`;
  }
  if (!label) {
    const existingLabels = new Set(state.tabs.map((entry) => entry.label));
    label = generateDefaultTabLabel(existingLabels);
  }

  const container = document.createElement("div");
  container.className = "terminal-screen";
  container.dataset.tabId = id;

  elements.terminalHost.appendChild(container);

  const term = new Terminal({ ...terminalDefaults });
  const terminalWithOptions = /** @type {Terminal & { setOption?: (key: string, value: unknown) => void }} */ (
    term
  );
  if (typeof terminalWithOptions.setOption === "function") {
    try {
      terminalWithOptions.setOption(
        "rendererType",
        terminalDefaults.rendererType || "canvas",
      );
      if (Number.isFinite(terminalDefaults.scrollback)) {
        terminalWithOptions.setOption("scrollback", terminalDefaults.scrollback);
      }
      if (typeof terminalDefaults.cancelEvents === "boolean") {
        terminalWithOptions.setOption("cancelEvents", terminalDefaults.cancelEvents);
      }
    } catch (error) {
      console.warn(
        "Unable to apply terminal renderer/scrollback options:",
        error,
      );
    }
  }
  const fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.open(container);
  term.write("\r");
  fitAddon.fit();

  const tab = /** @type {TerminalTab} */ ({
    id,
    label,
    defaultLabel: label,
    colorId: colorId || TAB_COLOR_DEFAULT,
    term,
    fitAddon,
    container,
    phase: "idle",
    socketState: "disconnected",
    session: null,
    socket: null,
    reconnecting: false,
    hasEverConnected: false,
    hasReceivedLiveOutput: false,
    heartbeatInterval: null,
    inputSeq: 0,
    inputBatch: null,
    inputBatchScheduled: false,
    replayPending: false,
    replayComplete: true,
    lastReplayCount: 0,
    lastReplayTruncated: false,
    transcript: [],
    transcriptByteSize: 0,
    transcriptHydrated: false,
    transcriptHydrating: false,
    events: [],
    suppressed: initialSuppressedState(),
    pendingWrites: [],
    localEchoBuffer: [],
    errorMessage: "",
    lastSentSize: { cols: 0, rows: 0 },
    telemetry: {
      typed: 0,
      queued: 0,
      sent: 0,
      batches: 0,
      lastBatchSize: 0,
    },
    layoutCache: {
      width: 0,
      height: 0,
    },
    domItem: null,
    domButton: null,
    domClose: null,
    domLabel: null,
    domStatus: null,
    wasDetached: false,
    userScroll: {
      active: false,
      pinnedToBottom: true,
      touchActive: false,
      momentumActive: false,
      releaseTimer: null,
      lastInteraction: 0,
      cleanup: null,
    },
  });

  resetTabRuntimeState(tab, { resetTerminal: false });

  const detachViewportHandlers = attachViewportInteractionHandlers(tab);
  if (typeof detachViewportHandlers === "function") {
    tab.userScroll.cleanup = detachViewportHandlers;
  }

  const handlers = getTerminalHandlers();
  term.onResize(({ cols, rows }) => {
    handlers.onResize?.(tab, cols, rows);
  });

  term.onData((data) => {
    handlers.onData?.(tab, data);
  });

  container.addEventListener("mousedown", () => {
    setActiveTab(tab.id);
  });

  state.tabs.push(tab);
  renderTabs();

  if (focus || state.activeTabId === null) {
    setActiveTab(tab.id);
  } else if (tab.container) {
    tab.container.classList.remove("active");
  }

  const { onTabCreated } = tabCallbacks;
  if (typeof onTabCreated === "function") {
    onTabCreated(tab);
  }

  return tab;
}

/**
 * Destroy a terminal tab and release associated resources.
 * @param {TerminalTab | null | undefined} tab
 */
export function destroyTerminalTab(tab) {
  if (!tab) return;
  if (state.tabMenu.open && state.tabMenu.tabId === tab.id) {
    closeTabCustomization();
  }
  if (tab.heartbeatInterval) {
    clearInterval(tab.heartbeatInterval);
    tab.heartbeatInterval = null;
  }
  tab.inputBatchScheduled = false;
  tab.inputBatch = null;
  tab.localEchoBuffer = [];
  try {
    tab.term?.dispose();
  } catch (_error) {}
  try {
    tab.socket?.close();
  } catch (_error) {}
  if (tab.container?.parentElement) {
    tab.container.parentElement.removeChild(tab.container);
  }
  if (tab.domItem?.parentElement) {
    tab.domItem.parentElement.removeChild(tab.domItem);
  }
  if (tab.userScroll?.cleanup) {
    try {
      tab.userScroll.cleanup();
    } catch (_error) {
      // ignore cleanup issues during teardown
    }
  }
  tab.domItem = null;
  tab.domButton = null;
  tab.domClose = null;
  tab.domLabel = null;
  tab.domStatus = null;

  // Active tab management is handled by higher-level controllers.
}
