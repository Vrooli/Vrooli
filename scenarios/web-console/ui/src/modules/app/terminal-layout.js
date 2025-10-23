// @ts-check

import { elements } from "../state.js";
import { fitTerminal } from "./terminal-utils.js";

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

/**
 * @typedef {Object} LayoutWatcherConfig
 * @property {() => TerminalTab | null} getActiveTab
 * @property {(tab: TerminalTab) => void} [notifyTabSized]
 * @property {() => void} [onLayoutChanged]
 */

/** @type {LayoutWatcherConfig | null} */
let watcherConfig = null;
/** @type {ResizeObserver | null} */
let resizeObserver = null;
/** @type {((event: UIEvent) => void) | null} */
let windowResizeHandler = null;
/** @type {ReturnType<typeof requestAnimationFrame> | null} */
let pendingFitHandle = null;
/** @type {boolean} */
let pendingLayoutCallback = false;

function scheduleLayoutCallback() {
  if (!watcherConfig?.onLayoutChanged || pendingLayoutCallback) {
    return;
  }
  pendingLayoutCallback = true;
  requestAnimationFrame(() => {
    pendingLayoutCallback = false;
    watcherConfig?.onLayoutChanged?.();
  });
}

function applyFitIfNeeded() {
  if (!watcherConfig || pendingFitHandle !== null) {
    return;
  }
  pendingFitHandle = requestAnimationFrame(() => {
    pendingFitHandle = null;
    const tab = watcherConfig?.getActiveTab();
    if (!tab || !tab.term) {
      return;
    }

    const previousCols = typeof tab.term.cols === "number" ? tab.term.cols : 0;
    const previousRows = typeof tab.term.rows === "number" ? tab.term.rows : 0;

    /** @type {{ colsChanged: boolean; rowsChanged: boolean }} */
    let fitResult = { colsChanged: false, rowsChanged: false };
    try {
      fitResult = fitTerminal(tab);
    } catch (error) {
      console.warn("Failed to refresh terminal during layout adjustment:", error);
    }

    if (
      watcherConfig?.notifyTabSized &&
      (fitResult.colsChanged ||
        fitResult.rowsChanged ||
        tab.term.cols !== previousCols ||
        tab.term.rows !== previousRows)
    ) {
      watcherConfig.notifyTabSized(tab);
    }
  });
}

function observeTargets() {
  if (!watcherConfig) {
    return;
  }

  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
  if (windowResizeHandler && typeof window !== "undefined") {
    window.removeEventListener("resize", windowResizeHandler);
    windowResizeHandler = null;
  }

  const targets = [
    elements.terminalHost,
    elements.layout,
    elements.detailsDrawer,
  ].filter((target) => target instanceof HTMLElement);

  if (typeof ResizeObserver === "function" && targets.length > 0) {
    resizeObserver = new ResizeObserver(() => {
      applyFitIfNeeded();
      scheduleLayoutCallback();
    });
    targets.forEach((target) => resizeObserver?.observe(target));
  } else if (typeof window !== "undefined") {
    windowResizeHandler = () => {
      applyFitIfNeeded();
      scheduleLayoutCallback();
    };
    window.addEventListener("resize", windowResizeHandler);
  }

  applyFitIfNeeded();
}

/**
 * Initialize observers that keep the active terminal sized to its container.
 * @param {LayoutWatcherConfig} config
 */
export function initializeTerminalLayoutWatcher(config) {
  watcherConfig = config;
  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
  pendingFitHandle = null;
  pendingLayoutCallback = false;
  observeTargets();
}

/**
 * Request an explicit sizing pass for the active terminal.
 */
export function requestActiveTerminalFit() {
  applyFitIfNeeded();
}
