// @ts-check

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

export const TerminalFocusMode = Object.freeze({
  AUTO: "auto",
  FORCE: "force",
  SKIP: "skip",
});

/** @typedef {typeof TerminalFocusMode[keyof typeof TerminalFocusMode]} TerminalFocusModeValue */

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
export function refreshTerminalViewport(tab, { focusMode = TerminalFocusMode.AUTO, scroll = true } = {}) {
  if (!tab?.term) {
    return;
  }

  try {
    if (shouldRefocusTerminal(tab, focusMode)) {
      tab.term.focus();
    }
  } catch (error) {
    console.warn("Failed to focus terminal:", error);
  }

  if (scroll && typeof tab.term.scrollToBottom === "function") {
    try {
      tab.term.scrollToBottom();
    } catch (error) {
      console.warn("Failed to scroll terminal:", error);
    }
  }

  try {
    if (typeof tab.term.write === "function") {
      tab.term.write("");
    }
    if (typeof tab.term.refresh === "function" && typeof tab.term.rows === "number") {
      tab.term.refresh(0, Math.max(0, tab.term.rows - 1));
    }
  } catch (error) {
    console.warn("Failed to refresh terminal viewport:", error);
  }
}

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
 * Fit the terminal to its container, clamp overflow, and refresh the viewport.
 * Returns whether the geometry changed as a result of the operation.
 * @param {TerminalTab} tab
 * @param {{ focusMode?: TerminalFocusModeValue }} [options]
 * @returns {{ colsChanged: boolean; rowsChanged: boolean }}
 */
export function fitTerminal(tab, { focusMode = TerminalFocusMode.AUTO } = {}) {
  if (!tab?.term) {
    return { colsChanged: false, rowsChanged: false };
  }

  const previousCols = typeof tab.term.cols === "number" ? tab.term.cols : 0;
  const previousRows = typeof tab.term.rows === "number" ? tab.term.rows : 0;

  if (typeof tab.fitAddon?.fit === "function") {
    tab.fitAddon.fit();
  }

  const clamped = clampTerminalColumns(tab);

  refreshTerminalViewport(tab, { focusMode });

  const currentCols = typeof tab.term.cols === "number" ? tab.term.cols : previousCols;
  const currentRows = typeof tab.term.rows === "number" ? tab.term.rows : previousRows;

  return {
    colsChanged: clamped || currentCols !== previousCols,
    rowsChanged: currentRows !== previousRows,
  };
}
