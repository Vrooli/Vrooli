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
/** @type {((event: Event) => void) | null} */
let visualViewportHandler = null;
/** @type {number | null} */
let pendingWorkHandle = null;
/** @type {boolean} */
let fitRequested = false;
/** @type {boolean} */
let layoutRequested = false;
/** @type {{ width: number; height: number; scale: number } | null} */
let lastVisualViewportMetrics = null;
/** @type {boolean} */
let forceNextFit = false;

/**
 * Cancel a previously scheduled animation or timeout frame.
 * @param {number | null} handle
 * @returns {void}
 */
function cancelFrame(handle) {
  if (handle === null) {
    return;
  }
  if (typeof cancelAnimationFrame === "function") {
    cancelAnimationFrame(handle);
  } else if (typeof window !== "undefined" && typeof window.clearTimeout === "function") {
    window.clearTimeout(handle);
  }
}

/**
 * Schedule a callback on the next animation frame (or a timeout fallback).
 * @param {FrameRequestCallback} callback
 * @returns {number | null}
 */
function scheduleFrame(callback) {
  if (typeof requestAnimationFrame === "function") {
    return requestAnimationFrame(callback);
  }
  if (typeof window !== "undefined" && typeof window.setTimeout === "function") {
    return window.setTimeout(callback, 16);
  }
  return null;
}

function detachResizeObserver() {
  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
}

function detachWindowResizeListener() {
  if (windowResizeHandler && typeof window !== "undefined") {
    window.removeEventListener("resize", windowResizeHandler);
    windowResizeHandler = null;
  }
}

function detachVisualViewportListeners() {
  if (
    visualViewportHandler &&
    typeof window !== "undefined" &&
    window.visualViewport
  ) {
    window.visualViewport.removeEventListener("resize", visualViewportHandler);
    window.visualViewport.removeEventListener("scroll", visualViewportHandler);
  }
  visualViewportHandler = null;
  lastVisualViewportMetrics = null;
}

function handleViewportMutation() {
  queueDeferredWork({ fit: true, layout: true });
}

/**
 * @param {VisualViewport} viewport
 */
function captureViewportMetrics(viewport) {
  const width =
    typeof viewport.width === "number" ? Math.round(viewport.width) : 0;
  const height =
    typeof viewport.height === "number" ? Math.round(viewport.height) : 0;
  const scale =
    typeof viewport.scale === "number"
      ? Number(viewport.scale.toFixed(4))
      : 1;
  return { width, height, scale };
}

/**
 * @param {VisualViewport} viewport
 */
function shouldProcessViewportMutation(viewport) {
  const nextMetrics = captureViewportMetrics(viewport);
  const previousMetrics = lastVisualViewportMetrics;
  lastVisualViewportMetrics = nextMetrics;
  if (!previousMetrics) {
    return true;
  }
  return (
    previousMetrics.width !== nextMetrics.width ||
    previousMetrics.height !== nextMetrics.height ||
    previousMetrics.scale !== nextMetrics.scale
  );
}

function attachVisualViewportListeners() {
  if (typeof window === "undefined" || !window.visualViewport) {
    return;
  }
  detachVisualViewportListeners();
  lastVisualViewportMetrics = captureViewportMetrics(window.visualViewport);
  visualViewportHandler = () => {
    if (!window.visualViewport) {
      return;
    }
    if (shouldProcessViewportMutation(window.visualViewport)) {
      handleViewportMutation();
    }
  };
  window.visualViewport.addEventListener("resize", visualViewportHandler);
  window.visualViewport.addEventListener("scroll", visualViewportHandler);
}

function runFitForActiveTab() {
  const tab = watcherConfig?.getActiveTab?.();
  if (!tab || !tab.term) {
    forceNextFit = false;
    return;
  }

  const userScroll = tab.userScroll;
  const userStillInteracting =
    !!userScroll &&
    (userScroll.touchActive === true ||
      userScroll.momentumActive === true ||
      (typeof userScroll.lastInteraction === "number" &&
        Date.now() - userScroll.lastInteraction < 650));

  if (userStillInteracting && forceNextFit !== true) {
    if (typeof window !== "undefined") {
      window.setTimeout(() => {
        queueDeferredWork({ fit: true });
      }, 160);
    }
    return;
  }

  const previousCols = typeof tab.term.cols === "number" ? tab.term.cols : 0;
  const previousRows = typeof tab.term.rows === "number" ? tab.term.rows : 0;

  /** @type {{ colsChanged: boolean; rowsChanged: boolean }} */
  let fitResult = { colsChanged: false, rowsChanged: false };
  try {
    fitResult = fitTerminal(tab, { forceFit: forceNextFit });
  } catch (error) {
    console.warn("Failed to refresh terminal during layout adjustment:", error);
  } finally {
    forceNextFit = false;
  }

  if (!watcherConfig?.notifyTabSized) {
    return;
  }

  const currentCols = typeof tab.term.cols === "number" ? tab.term.cols : previousCols;
  const currentRows = typeof tab.term.rows === "number" ? tab.term.rows : previousRows;

  if (
    fitResult.colsChanged ||
    fitResult.rowsChanged ||
    currentCols !== previousCols ||
    currentRows !== previousRows
  ) {
    try {
      watcherConfig.notifyTabSized(tab);
    } catch (error) {
      console.warn("Failed to notify tab sized:", error);
    }
  }
}

function runDeferredWork() {
  pendingWorkHandle = null;

  if (!watcherConfig) {
    fitRequested = false;
    layoutRequested = false;
    forceNextFit = false;
    return;
  }

  const shouldFit = fitRequested;
  const shouldLayout = layoutRequested && Boolean(watcherConfig.onLayoutChanged);

  fitRequested = false;
  layoutRequested = false;

  if (shouldFit) {
    runFitForActiveTab();
  }

  if (shouldLayout) {
    try {
      watcherConfig.onLayoutChanged?.();
    } catch (error) {
      console.warn("Failed during layout change callback:", error);
    }
  }

  if ((fitRequested || layoutRequested) && pendingWorkHandle === null) {
    pendingWorkHandle = scheduleFrame(runDeferredWork);
  }
}

function queueDeferredWork({ fit = false, layout = false, forceFitRequest = false } = {}) {
  if (!watcherConfig) {
    if (forceFitRequest) {
      forceNextFit = false;
    }
    return;
  }

  if (fit) {
    fitRequested = true;
  }
  if (layout) {
    layoutRequested = true;
  }
  if (forceFitRequest) {
    forceNextFit = true;
  }

  if (pendingWorkHandle !== null) {
    return;
  }

  pendingWorkHandle = scheduleFrame(runDeferredWork);
}

function scheduleLayoutCallback() {
  if (!watcherConfig?.onLayoutChanged) {
    return;
  }
  queueDeferredWork({ layout: true });
}

function applyFitIfNeeded() {
  queueDeferredWork({ fit: true });
}

function observeTargets() {
  if (!watcherConfig) {
    return;
  }

  detachResizeObserver();
  detachWindowResizeListener();
  detachVisualViewportListeners();

  const targets = [
    elements.terminalHost,
    elements.layout,
    elements.detailsDrawer,
  ].filter((target) => target instanceof HTMLElement);

  if (typeof ResizeObserver === "function" && targets.length > 0) {
    resizeObserver = new ResizeObserver(() => {
      queueDeferredWork({ fit: true, layout: true });
    });
    targets.forEach((target) => resizeObserver?.observe(target));
  } else if (typeof window !== "undefined") {
    windowResizeHandler = () => {
      queueDeferredWork({ fit: true, layout: true });
    };
    window.addEventListener("resize", windowResizeHandler);
  }

  attachVisualViewportListeners();

  queueDeferredWork({ fit: true });
}

/**
 * Initialize observers that keep the active terminal sized to its container.
 * @param {LayoutWatcherConfig} config
 */
export function initializeTerminalLayoutWatcher(config) {
  teardownTerminalLayoutWatcher();
  watcherConfig = config;
  pendingWorkHandle = null;
  fitRequested = false;
  layoutRequested = false;
  forceNextFit = false;
  observeTargets();
  return teardownTerminalLayoutWatcher;
}

/**
 * Request an explicit sizing pass for the active terminal.
 * @param {{ force?: boolean }} [options]
 */
export function requestActiveTerminalFit(options = {}) {
  const forceFitRequest = options && options.force === true;
  queueDeferredWork({ fit: true, forceFitRequest });
}

/**
 * Disconnect observers and listeners created by the terminal layout watcher.
 */
export function teardownTerminalLayoutWatcher() {
  detachResizeObserver();
  detachWindowResizeListener();
  detachVisualViewportListeners();
  cancelFrame(pendingWorkHandle);
  pendingWorkHandle = null;
  fitRequested = false;
  layoutRequested = false;
  watcherConfig = null;
  forceNextFit = false;
}
