// @ts-check

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

export const TerminalFocusMode = Object.freeze({
  AUTO: "auto",
  FORCE: "force",
  SKIP: "skip",
});

/** @typedef {typeof TerminalFocusMode[keyof typeof TerminalFocusMode]} TerminalFocusModeValue */

/**
 * Decide whether focus should return to the terminal instance.
 * @param {TerminalTab | null | undefined} tab
 * @param {TerminalFocusModeValue} focusMode
 * @returns {boolean}
 */
function shouldRefocusTerminal(tab, focusMode) {
  if (focusMode === TerminalFocusMode.FORCE) {
    return true;
  }
  if (focusMode === TerminalFocusMode.SKIP) {
    return false;
  }
  if (typeof document === "undefined") {
    return true;
  }
  const activeElement = document.activeElement;
  if (!activeElement || activeElement === document.body) {
    return true;
  }
  if (!tab?.container) {
    return true;
  }
  if (activeElement instanceof HTMLElement && tab.container.contains(activeElement)) {
    return true;
  }
  return false;
}

/**
 * Keep the terminal viewport visually in sync without stealing focus unnecessarily.
 * @param {TerminalTab} tab
 * @param {{ focusMode?: TerminalFocusModeValue; scroll?: boolean }} [options]
 */
export function refreshTerminalViewport(
  tab,
  { focusMode = TerminalFocusMode.AUTO, scroll = true, respectUserScroll = true } = {},
) {
  if (!tab?.term) {
    return;
  }

  const userScrollState = tab.userScroll || {
    active: false,
    pinnedToBottom: true,
    touchActive: false,
    momentumActive: false,
    lastInteraction: 0,
  };

  const userActivelyScrolling =
    respectUserScroll !== false &&
    (userScrollState.touchActive === true ||
      userScrollState.momentumActive === true ||
      (typeof userScrollState.lastInteraction === "number" &&
        Date.now() - userScrollState.lastInteraction < 1200) ||
      (userScrollState.active === true &&
        userScrollState.pinnedToBottom === false));

  let focusApplied = false;
  const allowFocus =
    focusMode === TerminalFocusMode.FORCE || !userActivelyScrolling;

  if (allowFocus) {
    try {
      if (shouldRefocusTerminal(tab, focusMode)) {
        tab.term.focus();
        focusApplied = true;
      }
    } catch (error) {
      console.warn("Failed to focus terminal:", error);
    }
  }

  const shouldScroll = (() => {
    if (!scroll || typeof tab.term.scrollToBottom !== "function") {
      return false;
    }
    if (respectUserScroll === false) {
      return true;
    }
    if (userActivelyScrolling) {
      return false;
    }
    return isTerminalViewportPinnedToBottom(tab.term);
  })();

  let scrolled = false;
  if (shouldScroll) {
    try {
      tab.term.scrollToBottom();
      scrolled = true;
      if (tab.userScroll) {
        tab.userScroll.pinnedToBottom = true;
        tab.userScroll.active = false;
        tab.userScroll.momentumActive = false;
        tab.userScroll.lastInteraction = 0;
        if (tab.userScroll.releaseTimer) {
          clearTimeout(tab.userScroll.releaseTimer);
          tab.userScroll.releaseTimer = null;
        }
      }
    } catch (error) {
      console.warn("Failed to scroll terminal:", error);
    }
  }

  const needsRefresh = scrolled || focusApplied || focusMode === TerminalFocusMode.FORCE;

  if (
    needsRefresh &&
    typeof tab.term.refresh === "function" &&
    typeof tab.term.rows === "number"
  ) {
    try {
      tab.term.refresh(0, Math.max(0, tab.term.rows - 1));
    } catch (error) {
      console.warn("Failed to refresh terminal viewport:", error);
    }
  }
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @returns {boolean}
 */
function clampTerminalColumns(tab) {
  if (!tab?.term || !tab.container) {
    return false;
  }

  const viewport = tab.container.querySelector(".xterm-viewport");
  if (!(viewport instanceof HTMLElement)) {
    return false;
  }

  const hostRect = tab.container.getBoundingClientRect();
  const viewportRect = viewport.getBoundingClientRect();
  const hostWidth = hostRect.width;
  const viewportWidth = viewportRect.width;

  if (!Number.isFinite(hostWidth) || !Number.isFinite(viewportWidth) || hostWidth <= 0) {
    return false;
  }

  const overshoot = viewportWidth - hostWidth;
  const tolerance = 0.75; // ignore fractional differences below a pixel
  if (overshoot <= tolerance) {
    return false;
  }

  const currentCols = typeof tab.term.cols === "number" ? tab.term.cols : 0;
  if (currentCols <= 1) {
    return false;
  }

  const approxCellWidth = viewportWidth / currentCols;
  if (!Number.isFinite(approxCellWidth) || approxCellWidth <= 0) {
    return false;
  }

  const columnsToTrim = Math.max(1, Math.ceil(overshoot / approxCellWidth));
  const nextCols = Math.max(1, currentCols - columnsToTrim);
  if (nextCols >= currentCols) {
    return false;
  }

  try {
    tab.term.resize(nextCols, tab.term.rows);
    return true;
  } catch (error) {
    console.warn("Failed to clamp terminal width:", error);
    return false;
  }
}

/**
 * Determine whether the visible viewport is already pinned to the buffer bottom.
 * @param {import("../types.d.ts").TerminalTab['term']} term
 */
function isTerminalViewportPinnedToBottom(term) {
  const buffer = term?.buffer?.active;
  if (!buffer || typeof buffer.baseY !== "number") {
    return true;
  }
  const viewportY = typeof buffer.viewportY === "number" ? buffer.viewportY : buffer.ydisp;
  if (typeof viewportY === "number") {
    if (viewportY >= buffer.baseY) {
      return true;
    }
  }

  if (typeof buffer.length === "number" && typeof term?.rows === "number") {
    const viewportBottom = (typeof viewportY === "number" ? viewportY : buffer.baseY) + term.rows;
    return viewportBottom >= buffer.length - 1;
  }

  return false;
}

/**
 * Fit the terminal to its container, clamp overflow, and refresh the viewport.
 * Returns whether the geometry changed as a result of the operation.
 * @param {TerminalTab} tab
 * @param {{ focusMode?: TerminalFocusModeValue; respectUserScroll?: boolean }} [options]
 * @returns {{ colsChanged: boolean; rowsChanged: boolean }}
 */
export function fitTerminal(
  tab,
  {
    focusMode = TerminalFocusMode.AUTO,
    respectUserScroll = true,
    forceFit = false,
  } = {},
) {
  if (!tab?.term) {
    return { colsChanged: false, rowsChanged: false };
  }

  const previousCols = typeof tab.term.cols === "number" ? tab.term.cols : 0;
  const previousRows = typeof tab.term.rows === "number" ? tab.term.rows : 0;

  let containerWidth = 0;
  let containerHeight = 0;
  if (tab.container && typeof tab.container.getBoundingClientRect === "function") {
    const rect = tab.container.getBoundingClientRect();
    containerWidth = Number.isFinite(rect.width) ? Math.round(rect.width) : 0;
    containerHeight = Number.isFinite(rect.height)
      ? Math.round(rect.height)
      : 0;
  }

  const lastWidth = tab.layoutCache?.width ?? 0;
  const lastHeight = tab.layoutCache?.height ?? 0;
  const geometryChanged =
    containerWidth !== lastWidth || containerHeight !== lastHeight;

  const shouldApplyFit =
    (forceFit || geometryChanged) && typeof tab.fitAddon?.fit === "function";

  const userScroll = tab.userScroll;
  const userRecentlyActive =
    userScroll &&
    (userScroll.touchActive === true ||
      userScroll.momentumActive === true ||
      (typeof userScroll.lastInteraction === "number" &&
        Date.now() - userScroll.lastInteraction < 600));

  if (shouldApplyFit) {
    if (userRecentlyActive && forceFit !== true) {
      if (tab.userScroll) {
        tab.userScroll.lastInteraction = Date.now();
      }
    } else {
      try {
        tab.fitAddon.fit();
      } catch (error) {
        console.warn("Failed to run fit addon:", error);
      }
    }
  }

  if (!tab.layoutCache) {
    tab.layoutCache = { width: containerWidth, height: containerHeight };
  } else {
    tab.layoutCache.width = containerWidth;
    tab.layoutCache.height = containerHeight;
  }

  const clamped = clampTerminalColumns(tab);

  refreshTerminalViewport(tab, { focusMode, respectUserScroll });

  const currentCols = typeof tab.term.cols === "number" ? tab.term.cols : previousCols;
  const currentRows = typeof tab.term.rows === "number" ? tab.term.rows : previousRows;

  return {
    colsChanged: clamped || currentCols !== previousCols,
    rowsChanged: currentRows !== previousRows,
  };
}
